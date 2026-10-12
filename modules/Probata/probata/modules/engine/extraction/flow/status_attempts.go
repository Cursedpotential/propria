package flow

import (
	"context"
	"errors"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
	"strings"
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

// decorateExtractionStatus merges real parent and up to sixteen immediate child pending attempts into status.
// Inputs: actual Describe snapshot and queried result. Output: workflow/run IDs with attempt/failure fields.
// Effects: bounded child Describe reads; choose for composed monitoring without inferring coordinates from display labels.
func (s *Starter) decorateExtractionStatus(ctx context.Context, progress Progress, described *workflowservice.DescribeWorkflowExecutionResponse) Progress {
	progress = decoratePendingExtractionActivities(progress, described.GetPendingActivities())
	for n, child := range described.GetPendingChildren() {
		if n >= 16 {
			break
		}
		if !strings.HasPrefix(child.GetWorkflowId(), "entity-extraction:") && !strings.HasPrefix(child.GetWorkflowId(), "external-extraction:") {
			continue
		}
		snapshot, err := s.Client.DescribeWorkflowExecution(ctx, child.GetWorkflowId(), child.GetRunId())
		if err != nil {
			continue
		}
		active := decoratePendingExtractionActivities(Progress{}, snapshot.GetPendingActivities())
		for _, attempt := range active.Steps {
			if attempt.Attempt == 0 {
				continue
			}
			found := false
			for i := range progress.Steps {
				line := &progress.Steps[i]
				if line.WorkflowID == child.GetWorkflowId() {
					line.Attempt, line.ActivityID, line.ActivityType, line.LastFailure = attempt.Attempt, attempt.ActivityID, attempt.ActivityType, attempt.LastFailure
					if attempt.Status == "retrying" {
						line.Status, line.Detail = "retrying", attempt.LastFailure
					}
					found = true
					break
				}
			}
			if !found {
				attempt.WorkflowID, attempt.RunID = child.GetWorkflowId(), child.GetRunId()
				progress.Steps = append(progress.Steps, attempt)
			}
		}
	}
	return progress
}
