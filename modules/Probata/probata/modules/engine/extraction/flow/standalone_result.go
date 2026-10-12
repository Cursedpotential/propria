package flow

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// standaloneExtractionResult preserves successful work while making fatal standalone attempts fail in Temporal.
// Inputs: final Progress and actual workflow error. Outputs: original progress and typed fatal error with details.
// Effects: replay-version marker only; pick for Entity and External workflows sharing the same atomic Activities.
func standaloneExtractionResult(ctx workflow.Context, progress Progress, err error) (Progress, error) {
	if workflow.GetVersion(ctx, "standalone-extraction-fatal-result-v1", workflow.DefaultVersion, 1) == workflow.DefaultVersion {
		return progress, err
	}
	info := workflow.GetInfo(ctx)
	progress.WorkflowID, progress.RunID = info.WorkflowExecution.ID, info.WorkflowExecution.RunID
	if err == nil && progress.Outcome != OutcomeFailed {
		return progress, nil
	}
	progress.Outcome = OutcomeFailed
	reason := "extraction failed"
	if err != nil {
		reason = err.Error()
	}
	for _, step := range progress.Steps {
		if step.Status == StepFailed && step.Detail != "" {
			reason = step.Detail
			break
		}
	}
	if len(progress.Steps) == 0 {
		progress.Steps = []StepResult{{Step: "binding", Status: StepFailed, Detail: reason}}
	}
	return progress, temporal.NewNonRetryableApplicationError(reason, "StandaloneExtractionFailed", err, progress)
}
