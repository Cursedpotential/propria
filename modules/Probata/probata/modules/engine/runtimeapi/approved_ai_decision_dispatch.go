// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package runtimeapi

import (
	"context"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/caseidentity"
)

// ApprovedAIProjectionStarted identifies the accepted Temporal execution, not a completed projection.
// Inputs: dispatcher result. Outputs: stable workflow and actual run IDs. Effects: none.
// Choose for a committed AI decision response; graph readback remains a separate operation.
type ApprovedAIProjectionStarted struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// ApprovedAIProjectionDispatcher starts or joins one exact committed approval projection.
// Inputs: persisted source/candidate/decision identifiers and request digest. Outputs: execution IDs.
// Effects: Temporal enqueue only. Choose after DecideAICandidate succeeds, never for staging or rejection.
type ApprovedAIProjectionDispatcher interface {
	StartApprovedAIProjection(context.Context, approvedgraphai.Scope) (ApprovedAIProjectionStarted, error)
}

// AIDecisionProjection reports enqueue state independently from the durable owner decision.
// Inputs: an already committed decision. Outputs: enqueued, pending or not_requested metadata.
// Effects: none. Choose for HTTP200 decision receipts; enqueued does not mean graph projection completed.
type AIDecisionProjection struct {
	Status     string `json:"status"`
	WorkflowID string `json:"workflow_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	Retryable  bool   `json:"retryable"`
	Error      string `json:"error,omitempty"`
}

// dispatchApprovedAIDecisionProjection enqueues only after a genuine approved decision has committed.
// Inputs: exact successful store receipt and verified source version; canonical case comes from caseidentity.
// Outputs: bounded launch metadata; errors retain the committed receipt and indicate same-request retry.
// Effects: at most one five-second Temporal start/join attempt. Choose at the decision handler's success branch.
func (h *EntityExtractionHTTPHandler) dispatchApprovedAIDecisionProjection(ctx context.Context, decision, sourceVersionID, candidateID, decisionID, digest string) AIDecisionProjection {
	if decision != "approved" {
		return AIDecisionProjection{Status: "not_requested"}
	}
	scope := approvedgraphai.Scope{
		MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID,
		SourceVersionID: sourceVersionID, CandidateID: candidateID, DecisionID: decisionID, RequestDigest: digest,
	}
	workflowID, err := approvedgraphai.WorkflowID(scope)
	result := AIDecisionProjection{Status: "pending", WorkflowID: workflowID, Retryable: true}
	if err != nil {
		result.Error = "projection_scope_invalid"
		return result
	}
	var dispatcher ApprovedAIProjectionDispatcher
	if h != nil {
		dispatcher, _ = h.workflows.(ApprovedAIProjectionDispatcher)
	}
	if dispatcher == nil {
		result.Error = "projection_dispatch_unavailable"
		return result
	}
	launchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	started, err := dispatcher.StartApprovedAIProjection(launchCtx, scope)
	if err != nil {
		result.Error = "projection_enqueue_failed"
		return result
	}
	if started.WorkflowID != workflowID || started.RunID == "" {
		result.Error = "projection_dispatch_mismatch"
		return result
	}
	return AIDecisionProjection{Status: "enqueued", WorkflowID: workflowID, RunID: started.RunID}
}
