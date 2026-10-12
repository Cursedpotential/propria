package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/proffer"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	sdktemporal "go.temporal.io/sdk/temporal"
	"strings"
)

// StartPreviewAdmissionRepair starts or joins the exact attributed repair as its own Temporal workflow.
// Inputs: explicit source-bound owner action. Output: durable workflow/run IDs. Effects: Temporal start only.
// Pick for historical binding repair; extraction and source processing remain separate workflows.
func (s *temporalStarter) StartPreviewAdmissionRepair(ctx context.Context, in proffer.PreviewAdmissionRepairInput) (proffer.ContextStarted, error) {
	if err := proffer.ValidatePreviewAdmissionRepair(in); err != nil {
		return proffer.ContextStarted{}, err
	}
	raw, _ := json.Marshal(in)
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	id := "preview-admission-repair:" + in.PreviewHandle + ":" + digest[:24]
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: id, TaskQueue: s.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE, WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		Memo: map[string]interface{}{"actor_subject_uid": in.ActorSubjectUID, "preview_handle": in.PreviewHandle, "source_version_ref": in.SourceVersionRef, "repair_digest": digest}}, proffer.PreviewAdmissionRepairWorkflowName, in)
	if err != nil {
		return proffer.ContextStarted{}, err
	}
	return proffer.ContextStarted{WorkflowID: run.GetID(), RunID: run.GetRunID()}, nil
}

// PreviewAdmissionRepairStatus reads real repair progress and terminal business failure from Temporal.
// Inputs: returned workflow ID and actor. Output: actual attempt/status/receipt. Effects: describe/query/result reads.
// Pick for existing in-app polling/notifications; Temporal completion alone is never treated as repair success.
func (s *temporalStarter) PreviewAdmissionRepairStatus(ctx context.Context, id, actor string) (proffer.PreviewAdmissionRepairResult, error) {
	if !strings.HasPrefix(id, "preview-admission-repair:") || len(id) > 220 {
		return proffer.PreviewAdmissionRepairResult{}, errors.New("unknown repair workflow")
	}
	d, err := s.client.DescribeWorkflowExecution(ctx, id, "")
	if err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	var owner string
	payload := d.GetWorkflowExecutionInfo().GetMemo().GetFields()["actor_subject_uid"]
	if payload == nil {
		return proffer.PreviewAdmissionRepairResult{}, errors.New("unknown repair workflow")
	}
	err = converter.GetDefaultDataConverter().FromPayload(payload, &owner)
	if err != nil || owner != actor {
		return proffer.PreviewAdmissionRepairResult{}, errors.New("unknown repair workflow")
	}
	var result proffer.PreviewAdmissionRepairResult
	info := d.GetWorkflowExecutionInfo()
	switch info.GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		value, err := s.client.QueryWorkflow(ctx, id, "", "status")
		if err != nil {
			return result, err
		}
		err = value.Get(&result)
		return result, err
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		err = s.client.GetWorkflow(ctx, id, info.GetExecution().GetRunId()).Get(ctx, &result)
		return result, err
	default:
		failure := s.client.GetWorkflow(ctx, id, info.GetExecution().GetRunId()).Get(ctx, nil)
		var application *sdktemporal.ApplicationError
		if errors.As(failure, &application) && application.HasDetails() && application.Details(&result) == nil {
			return result, nil
		}
		result.Status, result.Reason, result.WorkflowID, result.RunID = "failed", info.GetStatus().String(), id, info.GetExecution().GetRunId()
		if failure != nil {
			result.Reason = failure.Error()
		}
		return result, nil
	}
}
