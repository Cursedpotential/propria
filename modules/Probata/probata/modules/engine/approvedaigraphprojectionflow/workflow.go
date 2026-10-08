// Package approvedaigraphprojectionflow schedules one already approved native AI graph projection.
package approvedaigraphprojectionflow

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
)

const (
	WorkflowName = "approved_ai_context_graph_projection_workflow"
	ActivityName = "project_approved_ai_context_graph_activity"
	StatusQuery  = "status"
)

// Progress exposes the exact decision selector and reference-only projection result.
// Inputs: workflow execution. Outputs: status query value. Effects: none.
// Pick for status reads; approval remains in the existing PostgreSQL ledger.
type Progress struct {
	Outcome string                               `json:"outcome"`
	Scope   approvedgraphai.Scope                `json:"scope"`
	Result  *activities.ApprovedProjectionResult `json:"result,omitempty"`
	Error   string                               `json:"error,omitempty"`
}

// Workflow schedules verification and projection for an existing approved native AI decision.
// Inputs: canonical decision scope. Outputs: bounded graph control receipt.
// Effects: one PostgreSQL read and idempotent Surreal graph write in the Activity.
// Pick after owner approval commits; a projection failure does not change the review decision.
func Workflow(ctx workflow.Context, scope approvedgraphai.Scope) (Progress, error) {
	progress := Progress{Outcome: "running", Scope: scope}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	if err := approvedgraphai.ValidateScope(scope); err != nil {
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 30 * time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2}})
	var result activities.ApprovedProjectionResult
	if err := workflow.ExecuteActivity(activityCtx, ActivityName, scope).Get(ctx, &result); err != nil {
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	if result.Count != 1 || result.ApprovedRevisionID != scope.DecisionID || result.ApprovalDigest != scope.RequestDigest || result.ControlGenerationID == "" || result.ClaimsHash == "" {
		err := errors.New("approved AI graph: incomplete or mismatched projection result")
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	progress.Outcome, progress.Result = "completed", &result
	return progress, nil
}
