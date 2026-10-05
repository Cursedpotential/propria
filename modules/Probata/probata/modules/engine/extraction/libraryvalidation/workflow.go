// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	WorkflowName        = "ToolkitLibraryValidationWorkflow"
	TaskQueue           = "proffer-v1"
	PrepareActivity     = "toolkit_library_validation_prepare_activity"
	SnapshotActivity    = "toolkit_library_validation_snapshot_activity"
	VerifyActivity      = "toolkit_library_validation_claim_activity"
	ReceiptActivity     = "toolkit_library_validation_receipt_activity"
	ProgressQuery       = "library_validation_progress"
	MaxConcurrentClaims = 4
)

// LibraryValidationInput carries only a saved shared proposal identity through the starter and Temporal history.
// Inputs: proposal_id. Outputs: tracked validation result. Effects: schedules Activities; never publishes.
type LibraryValidationInput struct {
	ProposalID string `json:"proposal_id"`
}

// Progress exposes bounded tracked workflow status without any personal proposal/source bodies.
// Inputs: workflow phase/counts; outputs: query projection. Effects: none; choose for queued/pending/error UI.
type Progress struct {
	ProposalID string `json:"proposal_id"`
	Phase      string `json:"phase"`
	Total      int    `json:"total"`
	Completed  int    `json:"completed"`
	Failed     int    `json:"failed"`
	Status     string `json:"status"`
}

// WorkflowID is stable across dispatcher retries; inputs: valid proposal ID; outputs: deterministic ID or error; effects: none.
func WorkflowID(id string) (string, error) {
	match := proposalPattern.FindStringSubmatch(id)
	if match == nil {
		return "", errors.New("invalid library proposal ID")
	}
	return "library-validation-" + match[1], nil
}

// ToolkitLibraryValidationWorkflow sequences independently tracked source fetches, claim checks and trusted receipt commit.
// Inputs: saved proposal identity. Outputs: reference-only Result. Effects: Activities only, bounded fan-out of four claims.
// Choose after durable libraryPropose; failures and uncleared currency remain visible and never trigger publication.
func ToolkitLibraryValidationWorkflow(ctx workflow.Context, input LibraryValidationInput) (Result, error) {
	if _, err := WorkflowID(input.ProposalID); err != nil {
		return Result{}, temporal.NewNonRetryableApplicationError(err.Error(), "InvalidLibraryProposal", nil)
	}
	progress := Progress{ProposalID: input.ProposalID, Phase: "preparing", Status: "pending_validation"}
	if err := workflow.SetQueryHandler(ctx, ProgressQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return Result{}, err
	}
	base := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 6 * time.Minute, ScheduleToCloseTimeout: 20 * time.Minute, HeartbeatTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, MaximumInterval: 15 * time.Second, MaximumAttempts: 3}})
	var plan Plan
	if err := workflow.ExecuteActivity(base, PrepareActivity, input).Get(ctx, &plan); err != nil {
		progress.Status = Blocked
		progress.Phase = "prepare_failed"
		return Result{}, err
	}
	if plan.ClaimCount < 1 || plan.ClaimCount > MaxClaims {
		return Result{}, temporal.NewNonRetryableApplicationError("invalid prepared claim count", "InvalidLibraryPlan", nil)
	}
	progress.Total = plan.ClaimCount
	plan.AttemptID = workflow.GetInfo(ctx).WorkflowExecution.RunID
	checks := make([]StepResult, 0, plan.ClaimCount)
	for first := 0; first < plan.ClaimCount; first += MaxConcurrentClaims {
		count := MaxConcurrentClaims
		if plan.ClaimCount-first < count {
			count = plan.ClaimCount - first
		}
		progress.Phase = "fetching_sources"
		fetches := make([]workflow.Future, count)
		for offset := 0; offset < count; offset++ {
			fetches[offset] = workflow.ExecuteActivity(base, SnapshotActivity, ClaimInput{Plan: plan, Index: first + offset})
		}
		verifies := make([]workflow.Future, count)
		progress.Phase = "checking_claims"
		for offset, future := range fetches {
			var snapshot StepResult
			if err := future.Get(ctx, &snapshot); err != nil {
				progress.Failed++
				continue
			}
			verifies[offset] = workflow.ExecuteActivity(base, VerifyActivity, ClaimInput{Plan: plan, Index: first + offset, SnapshotRef: snapshot.Ref})
		}
		for _, future := range verifies {
			if future == nil {
				continue
			}
			var check StepResult
			if err := future.Get(ctx, &check); err != nil {
				progress.Failed++
				continue
			}
			checks = append(checks, check)
			progress.Completed++
			if check.Status != Verified {
				progress.Failed++
			}
		}
	}
	progress.Phase = "committing_receipt"
	var result Result
	if err := workflow.ExecuteActivity(base, ReceiptActivity, FinishInput{Plan: plan, Checks: checks}).Get(ctx, &result); err != nil {
		progress.Status = Blocked
		progress.Phase = "receipt_failed"
		return Result{}, err
	}
	progress.Phase = "complete"
	progress.Status = result.Status
	if result.Status != Verified || result.CurrencyStatus != Cleared || !result.DatabaseCommitted {
		progress.Phase = "validation_blocked"
		return result, temporal.NewNonRetryableApplicationError("library validation did not clear every claim and currency check", "LibraryValidationBlocked", nil, result)
	}
	return result, nil
}

// WorkflowRegistrar is the worker's narrow existing registration contract; inputs: function/options; effects: registration only.
type WorkflowRegistrar interface {
	RegisterWorkflowWithOptions(any, workflow.RegisterOptions)
}

// RegisterWorkflow installs the exact starter-visible name; inputs: worker; outputs: none; effects: worker registration only.
func RegisterWorkflow(registrar WorkflowRegistrar) {
	registrar.RegisterWorkflowWithOptions(ToolkitLibraryValidationWorkflow, workflow.RegisterOptions{Name: WorkflowName})
}
