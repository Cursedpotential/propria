package flow

import (
	"context"
	"errors"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
)

// boundedExtractionFailure keeps real failure text within the existing compact status surface.
// Inputs: Temporal error message. Output: bounded reason. Effects: none; pick for status metadata only.
func boundedExtractionFailure(message string) string {
	if len(message) > 1200 {
		return message[:1200]
	}
	return message
}

// extractionFailureProgress recovers the actual business result from a typed failed Temporal execution.
// Inputs: WorkflowExecutionError chain. Output: original Progress and availability. Effects: none.
// Pick instead of discarding all-child failure detail as generic workflow_failed.
func extractionFailureProgress(err error) (Progress, bool) {
	var app *temporal.ApplicationError
	var progress Progress
	if errors.As(err, &app) && app.HasDetails() && app.Details(&progress) == nil && progress.Outcome != "" {
		return progress, true
	}
	return progress, false
}

// decoratePendingExtractionActivities attaches actual server Activity attempt and last failure metadata.
// Inputs: queried progress and bounded Describe pending Activities. Output: annotated steps.
// Effects: none; choose for standalone extractor retry display, never inferred retry counts.
func decoratePendingExtractionActivities(progress Progress, pending []*workflowpb.PendingActivityInfo) Progress {
	for n, activity := range pending {
		if n >= 16 {
			break
		}
		name := activity.GetActivityType().GetName()
		step := ""
		switch name {
		case ProposeRulesActivity:
			step = "rules"
		case ExtractModelActivity:
			step = "model"
		case ReconcileActivity:
			step = "reconcile"
		case SemanticaExtractActivity, LangExtractExtractActivity, BeginExternalRunActivity, StageExternalPageActivity, FinishExternalRunActivity:
			step = "extract"
		case ResolveConversationsActivity:
			step = "resolve"
		}
		index := -1
		for i := range progress.Steps {
			if progress.Steps[i].Step == step {
				index = i
				break
			}
		}
		if index < 0 {
			progress.Steps = append(progress.Steps, StepResult{Step: "activity:" + activity.GetActivityId(), Status: StepRunning})
			index = len(progress.Steps) - 1
		}
		line := &progress.Steps[index]
		line.ActivityID, line.ActivityType, line.Attempt = activity.GetActivityId(), name, activity.GetAttempt()
		line.LastFailure = boundedExtractionFailure(activity.GetLastFailure().GetMessage())
		if line.Attempt > 1 {
			line.Status = "retrying"
			if line.LastFailure != "" {
				line.Detail = line.LastFailure
			}
		}
	}
	return progress
}

// decorateExtractionStatus overlays bounded recorded child results and actual pending attempts.
// Inputs: actual parent Describe/query result. Output: source-correlated progress without synthetic duplicate rows.
// Effects: bounded Temporal reads only; choose for composed monitoring while sequential parent futures remain blocked.
func (s *Starter) decorateExtractionStatus(ctx context.Context, progress Progress, described *workflowservice.DescribeWorkflowExecutionResponse) Progress {
	progress = decoratePendingExtractionActivities(progress, described.GetPendingActivities())
	if described.GetWorkflowExecutionInfo().GetType().GetName() == RequestWorkflowName {
		return s.overlayRecordedExtractionChildren(ctx, progress, described)
	}
	return progress
}
