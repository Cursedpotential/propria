package proffer

import (
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"strings"
	"time"
)

const PreviewAdmissionRepairWorkflowName = "PreviewAdmissionRepairWorkflow"
const PreviewAdmissionRepairActivityName = "repair_preview_admission_activity"

// PreviewAdmissionRepairInput explicitly readmits an existing source binding without rewriting its original receipt.
// Inputs: actual handle/source version, fixed case, explicit owner mode, attribution and idempotency key.
// Output: reference-only workflow input. Effects: none until the registered repair Activity executes.
type PreviewAdmissionRepairInput struct {
	PreviewHandle    string `json:"preview_handle"`
	SourceVersionRef string `json:"source_version_ref"`
	OperatingMode    string `json:"operating_mode"`
	MatterID         string `json:"matter_id"`
	CourtCaseID      string `json:"court_case_id"`
	ActorSubjectUID  string `json:"actor_subject_uid"`
	ActorUsername    string `json:"actor_username"`
	IdempotencyKey   string `json:"idempotency_key"`
	Reason           string `json:"reason"`
}

// ValidatePreviewAdmissionRepair refuses inferred modes, unapproved case scope and incomplete owner attribution.
// Inputs: explicit repair request. Output: admission error or nil. Effects: none; pick before any repair write.
func ValidatePreviewAdmissionRepair(in PreviewAdmissionRepairInput) error {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(in.OperatingMode)); err != nil {
		return err
	}
	if !caseidentity.AdmittedIdentity(in.MatterID, in.CourtCaseID) {
		return errors.New("repair requires the approved case pair")
	}
	if _, err := uuid.Parse(in.SourceVersionRef); err != nil {
		return errors.New("repair requires the actual source version UUID")
	}
	for _, v := range []string{in.PreviewHandle, in.ActorSubjectUID, in.ActorUsername, in.IdempotencyKey, in.Reason} {
		if strings.TrimSpace(v) == "" || len(v) > 512 || strings.ContainsAny(v, "\x00\r\n") {
			return errors.New("bounded explicit repair attribution required")
		}
	}
	return nil
}

// PreviewAdmissionRepairResult identifies the appended receipt and the durable independent Temporal attempt.
// Inputs: verified store write and workflow execution. Output: status/receipt/source/attempt references only.
// Effects: none; choose for existing in-app status and failure notification presentation.
type PreviewAdmissionRepairResult struct {
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
	PreviewHandle    string `json:"preview_handle"`
	SourceVersionRef string `json:"source_version_ref"`
	ReceiptRef       string `json:"receipt_ref,omitempty"`
	EventID          int64  `json:"event_id"`
	WorkflowID       string `json:"workflow_id"`
	RunID            string `json:"run_id"`
}

// PreviewAdmissionRepairWorkflow invokes one shared repair Activity as an independent durable Temporal operation.
// Inputs: explicit owner repair and actual source binding. Output: actual failure or appended receipt with execution IDs.
// Effects: bounded Activity retries only; invalid authority is never retried indefinitely or silently defaulted.
func PreviewAdmissionRepairWorkflow(ctx workflow.Context, in PreviewAdmissionRepairInput) (PreviewAdmissionRepairResult, error) {
	info := workflow.GetInfo(ctx)
	result := PreviewAdmissionRepairResult{Status: "running", PreviewHandle: in.PreviewHandle, SourceVersionRef: in.SourceVersionRef, WorkflowID: info.WorkflowExecution.ID, RunID: info.WorkflowExecution.RunID}
	if err := workflow.SetQueryHandler(ctx, "status", func() (PreviewAdmissionRepairResult, error) { return result, nil }); err != nil {
		return result, err
	}
	if err := ValidatePreviewAdmissionRepair(in); err != nil {
		result.Status, result.Reason = "failed", err.Error()
		return result, temporal.NewNonRetryableApplicationError(result.Reason, "PreviewAdmissionRepairFailed", err, result)
	}
	options := workflow.ActivityOptions{StartToCloseTimeout: 30 * time.Second, ScheduleToCloseTimeout: 2 * time.Minute, RetryPolicy: retryPolicy(time.Second, 3)}
	var receipt PreviewAdmissionRepairResult
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), PreviewAdmissionRepairActivityName, in).Get(ctx, &receipt); err != nil {
		result.Status, result.Reason = "failed", err.Error()
		return result, temporal.NewNonRetryableApplicationError(result.Reason, "PreviewAdmissionRepairFailed", err, result)
	}
	result.Status, result.ReceiptRef, result.EventID = "completed", receipt.ReceiptRef, receipt.EventID
	return result, nil
}
