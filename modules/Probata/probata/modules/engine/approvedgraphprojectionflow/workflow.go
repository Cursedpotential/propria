// Package approvedgraphprojectionflow schedules one already reviewed graph projection.
package approvedgraphprojectionflow

import (
	"errors"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
)

const (
	WorkflowName = "approved_context_graph_projection_workflow"
	ActivityName = "project_approved_context_graph_activity"
	StatusQuery  = "status"
)

// Progress carries only the scoped projection receipt and workflow outcome.
type Progress struct {
	Outcome string                               `json:"outcome"`
	Scope   approvedgraph.Scope                  `json:"scope"`
	Result  *activities.ApprovedProjectionResult `json:"result,omitempty"`
	Error   string                               `json:"error,omitempty"`
}

// ValidateScope admits one complete canonical LIVE source and owner commit pin.
// Inputs: receipt, preview, generation and source scope. Outputs: error or nil. Effects: none.
// Pick before scheduling; the Activity still verifies the persisted completed receipt.
func ValidateScope(scope approvedgraph.Scope) error {
	if !caseidentity.AdmittedIdentity(scope.MatterID, scope.CourtCaseID) || scope.MatterMode != string(caseidentity.ModeLive) {
		return errors.New("approved graph projection requires the canonical LIVE case")
	}
	for _, id := range []string{scope.ReceiptID, scope.PreviewHandle, scope.GenerationID, scope.SourceVersionID} {
		if id == "" || len(id) > 240 || strings.TrimSpace(id) != id || strings.ContainsAny(id, "\x00\r\n") {
			return errors.New("approved graph projection requires complete bounded receipt and source pins")
		}
	}
	return nil
}

// Workflow schedules the existing atomic projection Activity for one approved revision.
// Inputs: exact canonical scope. Outputs: reference-only projection control receipt.
// Effects: one PostgreSQL resolution and Surreal projection in the Activity.
// Pick after a completed human-reviewed commit; never as a context ingestion gate.
func Workflow(ctx workflow.Context, scope approvedgraph.Scope) (Progress, error) {
	progress := Progress{Outcome: "running", Scope: scope}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	if err := ValidateScope(scope); err != nil {
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	var result activities.ApprovedProjectionResult
	if err := workflow.ExecuteActivity(activityCtx, ActivityName, scope).Get(ctx, &result); err != nil {
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	if result.ApprovedRevisionID != scope.ReceiptID || result.Count < 1 || result.ApprovalDigest == "" || result.ControlGenerationID == "" || result.ClaimsHash == "" {
		err := errors.New("approved graph projection returned a different or incomplete control receipt")
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	progress.Outcome, progress.Result = "completed", &result
	return progress, nil
}
