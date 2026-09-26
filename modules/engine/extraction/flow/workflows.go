// Byline: Claude Code · Opus 5.5 · 2026-09-25

package flow

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

// WorkflowRegistrar is the narrow worker seam (worker.Worker satisfies it).
type WorkflowRegistrar interface {
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
}

func workflowOptions(name string) workflow.RegisterOptions {
	return workflow.RegisterOptions{Name: name}
}

// RegisterWorkflows installs both extraction workflows under their exact
// names so Go function names never become a second naming scheme.
func RegisterWorkflows(registrar WorkflowRegistrar) {
	registrar.RegisterWorkflowWithOptions(EntityExtractionWorkflow, workflowOptions(ExtractionWorkflowName))
	registrar.RegisterWorkflowWithOptions(ExtractionCommitWorkflow, workflowOptions(CommitWorkflowName))
}

func shortOptions(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    3 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: 5 * time.Second, BackoffCoefficient: 2, MaximumAttempts: 3},
	})
}

func modelOptions(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Hour,
		HeartbeatTimeout:    10 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: 30 * time.Second, BackoffCoefficient: 2, MaximumAttempts: 3},
	})
}

// EntityExtractionWorkflow proposes entities and events for one Review run.
// It writes proposals only; nothing reaches the registry or the timeline
// until the owner runs ExtractionCommitWorkflow.
func EntityExtractionWorkflow(ctx workflow.Context, request ExtractionRequest) (Progress, error) {
	progress := Progress{Outcome: OutcomeRunning, Steps: []StepResult{
		{Step: "rules", Status: StepPending}, {Step: "model", Status: StepPending}, {Step: "reconcile", Status: StepPending},
	}}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	flagged := false

	progress.set("rules", StepRunning, "", nil)
	var rules ProposeResult
	if err := workflow.ExecuteActivity(shortOptions(ctx), ProposeRulesActivity, request).Get(ctx, &rules); err != nil {
		progress.set("rules", StepFailed, err.Error(), nil)
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	progress.set("rules", StepCompleted, fmt.Sprintf("%d people from %d messages' participants", rules.Proposals, rules.Messages),
		map[string]int{"proposals": rules.Proposals, "messages": rules.Messages})
	runIDs := []string{rules.ExtractionRunID}

	if !request.UseModel {
		progress.set("model", StepSkipped, "model extraction was not requested", nil)
	} else {
		progress.set("model", StepRunning, "", nil)
		var modelResult ProposeResult
		err := workflow.ExecuteActivity(modelOptions(ctx), ExtractModelActivity, request).Get(ctx, &modelResult)
		switch {
		case err != nil:
			flagged = true
			progress.set("model", StepFailed, err.Error(), nil)
		case modelResult.Skipped:
			flagged = true
			progress.set("model", StepSkipped, modelResult.Reason, nil)
		default:
			runIDs = append(runIDs, modelResult.ExtractionRunID)
			detail := fmt.Sprintf("%d proposals and %d events from %d batches", modelResult.Proposals, modelResult.Events, modelResult.Batches)
			progress.set("model", StepCompleted, detail, map[string]int{
				"proposals": modelResult.Proposals, "events": modelResult.Events, "batches": modelResult.Batches,
				"invalid_batches": len(modelResult.InvalidBatches), "ungrounded": modelResult.Ungrounded,
			})
			if len(modelResult.InvalidBatches) > 0 {
				flagged = true
				for i := range progress.Steps {
					if progress.Steps[i].Step == "model" {
						for _, invalid := range modelResult.InvalidBatches {
							progress.Steps[i].Flags = append(progress.Steps[i].Flags, entities.Flag{
								Code:   "model_batch_invalid",
								Detail: fmt.Sprintf("messages #%d-#%d: model output failed validation twice and was not used (%s)", invalid.FirstOrdinal, invalid.LastOrdinal, invalid.Reason),
							})
						}
					}
				}
			}
		}
	}

	progress.set("reconcile", StepRunning, "", nil)
	var reconciled ReconcileResult
	if err := workflow.ExecuteActivity(shortOptions(ctx), ReconcileActivity, ReconcileRequest{Extraction: request, RunIDs: runIDs}).Get(ctx, &reconciled); err != nil {
		progress.set("reconcile", StepFailed, err.Error(), nil)
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	progress.set("reconcile", StepCompleted, fmt.Sprintf("%d proposed entities (%d match committed entities)", reconciled.Proposals, reconciled.Matched),
		map[string]int{"proposals": reconciled.Proposals, "superseded": reconciled.Superseded, "matched": reconciled.Matched, "dropped": reconciled.Dropped})
	progress.Outcome = OutcomeCompleted
	if flagged {
		progress.Outcome = OutcomeCompletedFlagged
	}
	return progress, nil
}

// commitSteps are the write Activities in order. Each is idempotent: every
// row it writes has a deterministic id, so a retry inserts nothing twice.
var commitSteps = []struct{ step, activity string }{
	{"entities", CommitEntitiesActivity},
	{"aliases", CommitAliasesActivity},
	{"mentions", CommitMentionsActivity},
	{"events", CommitEventsActivity},
	{"timeline_members", CommitMembersActivity},
}

// ExtractionCommitWorkflow commits a validated proposal set: one Activity
// per step, a receipt for every attempt, and the Timesketch projection last.
// A projection that cannot run does not undo the commit; it is reported.
func ExtractionCommitWorkflow(ctx workflow.Context, request CommitRequest) (Progress, error) {
	progress := Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "validate", Status: StepPending}}}
	for _, step := range commitSteps {
		progress.Steps = append(progress.Steps, StepResult{Step: step.step, Status: StepPending})
	}
	progress.Steps = append(progress.Steps, StepResult{Step: "finalize", Status: StepPending}, StepResult{Step: "projection", Status: StepPending})
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	options := shortOptions(ctx)
	counts := map[string]int{}
	finalize := func(outcome, errText, failedAt string, promote bool) error {
		progress.set("finalize", StepRunning, "", nil)
		var result CommitStepResult
		err := workflow.ExecuteActivity(options, FinalizeCommitActivity, FinalizeRequest{
			Commit: request, Outcome: outcome, Error: errText, Counts: counts, Promote: promote, FailedAt: failedAt,
		}).Get(ctx, &result)
		if err != nil {
			progress.set("finalize", StepFailed, err.Error(), nil)
			return err
		}
		progress.set("finalize", StepCompleted, result.Detail, result.Counts)
		return nil
	}

	progress.set("validate", StepRunning, "", nil)
	var validation CommitStepResult
	if err := workflow.ExecuteActivity(options, ValidateCommitActivity, request).Get(ctx, &validation); err != nil {
		progress.set("validate", StepFailed, err.Error(), nil)
		_ = finalize(OutcomeFailed, err.Error(), "validate", false)
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	if reason := validationProblem(validation.Report, request.Digest); reason != "" {
		progress.set("validate", StepFailed, reason, nil)
		_ = finalize(OutcomeValidationFailed, reason, "validate", false)
		skipRemaining(&progress, "validate")
		progress.Outcome = OutcomeValidationFailed
		return progress, nil
	}
	progress.set("validate", StepCompleted, "every rule passed", map[string]int{
		"entities": validation.Report.Counts.Entities, "events": validation.Report.Counts.Events,
	})

	for _, step := range commitSteps {
		progress.set(step.step, StepRunning, "", nil)
		var result CommitStepResult
		if err := workflow.ExecuteActivity(options, step.activity, request).Get(ctx, &result); err != nil {
			progress.set(step.step, StepFailed, err.Error(), nil)
			_ = finalize(OutcomeFailed, err.Error(), step.step, false)
			progress.Outcome = OutcomeFailed
			return progress, nil
		}
		counts[step.step] = result.Written
		progress.set(step.step, StepCompleted, result.Detail, map[string]int{"written": result.Written, "already_present": result.Skipped})
	}
	if err := finalize(OutcomeCommitted, "", "", true); err != nil {
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	progress.Outcome = OutcomeCommitted

	if counts["timeline_members"] == 0 {
		progress.set("projection", StepSkipped, "no new timeline members to project", nil)
		return progress, nil
	}
	queue := request.ProjectionTaskQueue
	if queue == "" {
		queue = DefaultProjectionTaskQueue
	}
	projectionCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:              queue,
		ScheduleToStartTimeout: 2 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		RetryPolicy:            &temporal.RetryPolicy{InitialInterval: 10 * time.Second, MaximumAttempts: 2},
	})
	progress.set("projection", StepRunning, "", nil)
	var projection ProjectionResult
	err := workflow.ExecuteActivity(projectionCtx, BuildProjectionActivity, ProjectionRequest{
		CollectionSlug: request.CollectionSlug, CreatedBy: "extraction_commit:" + request.Actor.Username, CommitID: request.CommitID,
	}).Get(ctx, &projection)
	if err != nil {
		progress.set("projection", StepFailed, "the commit stands; the Timesketch projection did not run: "+err.Error(), nil)
		return progress, nil
	}
	progress.set("projection", StepCompleted, fmt.Sprintf("generation %d (%d members)", projection.Sequence, projection.MemberCount),
		map[string]int{"members": projection.MemberCount})
	return progress, nil
}

func validationProblem(report *commitcheck.Report, digest string) string {
	if report == nil {
		return "validation returned no report"
	}
	if !report.OK {
		var failed []string
		for _, check := range report.Checks {
			if check.Status != commitcheck.Pass {
				failed = append(failed, check.Rule+": "+check.Reason)
			}
		}
		return "validation failed — " + strings.Join(failed, "; ")
	}
	if report.Digest != digest {
		return "the proposals changed after they were validated; validate again before running"
	}
	return ""
}

func skipRemaining(progress *Progress, after string) {
	seen := false
	for i := range progress.Steps {
		if seen && progress.Steps[i].Status == StepPending && progress.Steps[i].Step != "finalize" {
			progress.Steps[i].Status = StepSkipped
		}
		if progress.Steps[i].Step == after {
			seen = true
		}
	}
}
