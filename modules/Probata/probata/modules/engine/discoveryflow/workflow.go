// Byline: Codex · GPT-6.1-sol · 2026-10-11
// Package discoveryflow durably orchestrates optional finite discovery independently of ingestion.
package discoveryflow

import (
	"github.com/Cursedpotential/probata/engine/discovery"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

const (
	WorkflowName     = "adaptive_discovery"
	AssembleActivity = "assemble_discovery_batch_activity"
	ScoreActivity    = "score_discovery_batch_activity"
	PersistActivity  = "persist_discovery_result_activity"
	FailActivity     = "fail_discovery_run_activity"
	PythonQueue      = "evidence-pipeline"
)

// Request carries only scope and durable revision IDs for one finite initial/rescore execution.
type Request = discovery.WorkRequest

// Workflow assembles, scores and publishes one immutable discovery revision without an intake/review gate.
// Inputs: durable run/scope/feedback revision. Outputs: PG revision/count receipt. Effects: schedules atomic Go/Python Activities only.
// Choose for explicitly requested discovery or durable correction rescores, never Proffer ingestion.
func Workflow(ctx workflow.Context, in Request) (discovery.Completion, error) {
	if !discovery.ID(in.RunID) || in.Scope.Validate() != nil || in.FeedbackRevision < 0 {
		return discovery.Completion{}, temporal.NewNonRetryableApplicationError("invalid discovery work reference", "DiscoveryContract", nil)
	}
	retry := &temporal.RetryPolicy{InitialInterval: time.Second, MaximumInterval: 30 * time.Second, MaximumAttempts: 3}
	goctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 3 * time.Minute, ScheduleToCloseTimeout: 12 * time.Minute, RetryPolicy: retry, WaitForCancellation: true})
	fail := func(stage string, err error) (discovery.Completion, error) {
		failureCtx, _ := workflow.NewDisconnectedContext(goctx)
		_ = workflow.ExecuteActivity(failureCtx, FailActivity, discovery.FailureRequest{WorkRequest: in, Reason: stage}).Get(failureCtx, nil)
		return discovery.Completion{}, err
	}
	var batch discovery.ScoreRequest
	if e := workflow.ExecuteActivity(goctx, AssembleActivity, in).Get(ctx, &batch); e != nil {
		return fail("assembly", e)
	}
	pyctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{TaskQueue: PythonQueue, StartToCloseTimeout: 5 * time.Minute, ScheduleToCloseTimeout: 20 * time.Minute, RetryPolicy: retry, WaitForCancellation: true})
	var result discovery.ScoreReceipt
	if e := workflow.ExecuteActivity(pyctx, ScoreActivity, batch).Get(ctx, &result); e != nil {
		return fail("scoring", e)
	}
	var done discovery.Completion
	e := workflow.ExecuteActivity(goctx, PersistActivity, discovery.PersistRequest{WorkRequest: in, BatchRef: batch.BatchRef, Receipt: result}).Get(ctx, &done)
	if e != nil {
		return fail("publication", e)
	}
	return done, e
}

// Registrar is the existing Go worker's exact workflow/Activity registration surface.
type Registrar interface {
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
	RegisterActivityWithOptions(interface{}, activity.RegisterOptions)
}

// Register installs the isolated discovery workflow and three atomic Go operations on the existing queue.
// Inputs: configured registrar and Activities. Outputs: none. Effects: registration only; Python owns its scoring registration.
func Register(r Registrar, a discovery.Activities) {
	r.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: WorkflowName})
	r.RegisterActivityWithOptions(a.Assemble, activity.RegisterOptions{Name: AssembleActivity})
	r.RegisterActivityWithOptions(a.Persist, activity.RegisterOptions{Name: PersistActivity})
	r.RegisterActivityWithOptions(a.Fail, activity.RegisterOptions{Name: FailActivity})
}
