// Byline: Codex / 2026-10-06.
package proffer

import (
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

const AIContentResumeWorkflowName = "AIContentResumeWorkflow"

// AIContentResumeInput identifies already admitted and verified source data without reimporting it.
// Inputs: the original import request, source, generation, verification and LIVE case pins.
// Outputs: six retained content-stage references. Effects: none until the workflow executes.
// Choose after a failed content stage; never use this contract to admit a new source.
type AIContentResumeInput struct {
	RequestID              string `json:"request_id"`
	SourceVersionID        string `json:"source_version_id"`
	NormalizedGenerationID string `json:"normalized_generation_id"`
	VerificationID         string `json:"verification_id"`
	OperatingMode          string `json:"operating_mode"`
	MatterID               string `json:"matter_id"`
	CourtCaseID            string `json:"court_case_id"`
}

// AIContentResumeWorkflow executes the same six content Activities against existing verified source pins.
// Inputs: original durable admission and exact generation/verification IDs. Outputs: verified content refs/counts.
// Effects: scoped derived bundles, configured model calls and additive conversation-chunk search; no acquisition,
// reparse, source approval, evidence promotion, source deletion or per-message publication. Every Activity
// rechecks the original LIVE admission and provenance; choose over restarting the entire import after a content failure.
func AIContentResumeWorkflow(ctx workflow.Context, in AIContentResumeInput) (*AIContentSummary, error) {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(in.OperatingMode)); err != nil {
		return nil, err
	}
	if !caseidentity.AdmittedIdentity(in.MatterID, in.CourtCaseID) || strings.TrimSpace(in.RequestID) == "" || len(in.RequestID) > 256 {
		return nil, fmt.Errorf("AI content resume requires an admitted case and original import request")
	}
	for _, pin := range []string{in.SourceVersionID, in.NormalizedGenerationID, in.VerificationID} {
		id, err := uuid.Parse(pin)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("AI content resume requires non-nil exact source, generation and verification UUIDs")
		}
	}
	r := &run{ctx: ctx, operatingMode: in.OperatingMode, requestID: in.RequestID,
		matterID: in.MatterID, courtCaseID: in.CourtCaseID, sourceVersionRef: Ref(in.SourceVersionID),
		operation: OperationState{Lifecycle: OperationRunning}}
	if err := workflow.SetQueryHandler(ctx, OperationQueryName, func() (OperationState, error) {
		return r.operationSnapshot(), nil
	}); err != nil {
		return nil, err
	}
	result, err := r.execAIContent(ctx, Ref(in.NormalizedGenerationID), Ref(in.VerificationID))
	r.operation.ActiveStages = []ActivityName{}
	r.operation.CurrentStage, r.operation.Wait = "", ""
	r.operation.Terminal = true
	if err != nil {
		r.operation.Lifecycle, r.operation.Reason = OperationFailed, err.Error()
		return nil, err
	}
	r.operation.Lifecycle = OperationCompleted
	return result, nil
}
