// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// Package dedupe removes the same-device duplicate messages that were committed
// before the match-up rule existed (owner 2026-10-02 15:40 EDT: "If it's a real
// duplicate from the exact same type of file from the exact same device, then
// we don't need it"; removal approved 16:00; 20:02: "Make sure all these things
// get run as Temporal activities and are traceable").
//
// A copy is a working row whose match key (device, platform, parties, sender,
// second and body hash; working.message_match_key) equals an earlier row's from
// a different source version of the same device. The earliest row is kept. The
// copy ends exactly as a message the match-up stage catches at commit time
// ends: its normalized record stays (it is part of a sealed generation and of
// its source's lineage), its working.message_occurrence row names the kept row
// as primary, and it has no working.message / third_party_message, participants,
// projection route or thread membership of its own.
//
// MessageDedupeWorkflow runs that as one Activity per step, each writing a
// receipt to working.message_dedupe_receipt:
//
//	plan                         freeze the copy set and the thread versions it touches
//	repoint_occurrences          every occurrence of a copy names the kept row
//	remove_thread_memberships    the copies leave their thread versions
//	recompute_thread_versions    bounds, horizon and membership digest from the rows left
//	remove_first_party_messages  participants, message and route, one transaction per batch
//	remove_third_party_messages  the same for acquired third-party rows
//	verify                       read-only assertions over the whole plan
//
// With dry_run every step replays the steps before it and itself in one
// transaction, reports its counts and rolls back, so the dry run executes the
// exact statements of the live run. The plan is frozen under dedupe_id: a live
// run with the same id removes exactly the copies its dry run counted, and
// expected_copies stops the run when the plan differs from what the owner saw.
package dedupe

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowName is the registered workflow type.
const WorkflowName = "message_dedupe_workflow"

// Step is one stage of the removal.
type Step string

const (
	StepPlan                     Step = "plan"
	StepRepointOccurrences       Step = "repoint_occurrences"
	StepRemoveThreadMemberships  Step = "remove_thread_memberships"
	StepRecomputeThreadVersions  Step = "recompute_thread_versions"
	StepRemoveFirstPartyMessages Step = "remove_first_party_messages"
	StepRemoveThirdPartyMessages Step = "remove_third_party_messages"
	StepVerify                   Step = "verify"
)

// Steps are the stages after the plan, in the order they run.
var Steps = []Step{
	StepRepointOccurrences, StepRemoveThreadMemberships, StepRecomputeThreadVersions,
	StepRemoveFirstPartyMessages, StepRemoveThirdPartyMessages, StepVerify,
}

// ActivityName is the registered Activity of one step.
func ActivityName(step Step) string { return "message_dedupe_" + string(step) + "_activity" }

// Rule names the copy rule a plan was frozen under.
const Rule = "same-device-match-key-v1"

// RefusalType is the non-retryable failure type of a refused step.
const RefusalType = "message_dedupe_refused"

// DefaultBatchSize and MaxBatchSize bound one live transaction.
const (
	DefaultBatchSize = 500
	MaxBatchSize     = 5000
)

var dedupeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,96}$`)

// Input starts one run.
type Input struct {
	DedupeID       string `json:"dedupe_id"`
	DryRun         bool   `json:"dry_run"`
	ExpectedCopies int64  `json:"expected_copies,omitempty"`
	BatchSize      int    `json:"batch_size,omitempty"`
}

// StepRequest is every step Activity's input.
type StepRequest struct {
	DedupeID       string `json:"dedupe_id"`
	WorkflowID     string `json:"workflow_id"`
	RunID          string `json:"run_id"`
	Step           Step   `json:"step"`
	DryRun         bool   `json:"dry_run"`
	BatchSize      int    `json:"batch_size"`
	ExpectedCopies int64  `json:"expected_copies,omitempty"`
}

// Receipt is one step's recorded outcome.
type Receipt struct {
	ReceiptID string           `json:"receipt_id"`
	Step      Step             `json:"step"`
	DryRun    bool             `json:"dry_run"`
	Counts    map[string]int64 `json:"counts"`
	Reused    bool             `json:"reused,omitempty"`
}

// Result is the run's outcome: every step's receipt, in order.
type Result struct {
	DedupeID string    `json:"dedupe_id"`
	DryRun   bool      `json:"dry_run"`
	Copies   int64     `json:"copies"`
	Receipts []Receipt `json:"receipts"`
}

// Refusal is a step that must not be retried: the plan or the data says stop.
type Refusal struct{ Reason string }

func (r Refusal) Error() string { return "message dedupe refused: " + r.Reason }

// Validate checks the input's shape.
func (in Input) Validate() error {
	if !dedupeIDPattern.MatchString(in.DedupeID) {
		return errors.New("dedupe_id must be 8-96 URL-safe characters")
	}
	if in.BatchSize < 0 || in.BatchSize > MaxBatchSize {
		return fmt.Errorf("batch_size must be 0 (default %d) to %d", DefaultBatchSize, MaxBatchSize)
	}
	if in.ExpectedCopies < 0 {
		return errors.New("expected_copies must not be negative")
	}
	return nil
}

func stepOptions(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Hour,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: 10 * time.Second, BackoffCoefficient: 2, MaximumInterval: 2 * time.Minute,
			MaximumAttempts: 3, NonRetryableErrorTypes: []string{RefusalType},
		},
	})
}

// MessageDedupeWorkflow is WorkflowName.
func MessageDedupeWorkflow(ctx workflow.Context, in Input) (Result, error) {
	if err := in.Validate(); err != nil {
		return Result{}, temporal.NewNonRetryableApplicationError(err.Error(), RefusalType, err)
	}
	batch := in.BatchSize
	if batch == 0 {
		batch = DefaultBatchSize
	}
	info := workflow.GetInfo(ctx)
	request := StepRequest{
		DedupeID: in.DedupeID, WorkflowID: info.WorkflowExecution.ID, RunID: info.WorkflowExecution.RunID,
		DryRun: in.DryRun, BatchSize: batch, ExpectedCopies: in.ExpectedCopies,
	}
	result := Result{DedupeID: in.DedupeID, DryRun: in.DryRun, Receipts: []Receipt{}}
	stepCtx := stepOptions(ctx)
	for _, step := range append([]Step{StepPlan}, Steps...) {
		request.Step = step
		var receipt Receipt
		if err := workflow.ExecuteActivity(stepCtx, ActivityName(step), request).Get(ctx, &receipt); err != nil {
			return result, fmt.Errorf("message dedupe step %s: %w", step, err)
		}
		result.Receipts = append(result.Receipts, receipt)
		if step == StepPlan {
			result.Copies = receipt.Counts["copies"]
		}
	}
	return result, nil
}
