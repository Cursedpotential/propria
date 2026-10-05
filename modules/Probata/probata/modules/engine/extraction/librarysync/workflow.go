// Byline: Codex · GPT-6.1 · 2026-10-05.
package librarysync

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	CycleWorkflowName   = "ToolkitLibrarySyncCycleWorkflow"
	WriteWorkflowName   = "ToolkitLibrarySyncWriteWorkflow"
	TaskQueue           = "proffer-v1"
	ProgressQuery       = "library_sync_progress"
	ListActivity        = "toolkit_library_sync_list_activity"
	SeenActivity        = "toolkit_library_sync_seen_activity"
	HashSourceActivity  = "toolkit_library_sync_hash_source_activity"
	RetainActivity      = "toolkit_library_sync_retain_activity"
	ExtractActivity     = "toolkit_library_sync_extract_activity"
	ObserveActivity     = "toolkit_library_sync_observe_activity"
	ClaimActivity       = "toolkit_library_sync_claim_activity"
	PrepareActivity     = "toolkit_library_sync_prepare_payload_activity"
	WriteActivity       = "toolkit_library_sync_write_activity"
	RefreshActivity     = "toolkit_library_sync_refresh_activity"
	HistoryActivity     = "toolkit_library_sync_history_activity"
	HashVersionActivity = "toolkit_library_sync_hash_version_activity"
	CurrentActivity     = "toolkit_library_sync_current_activity"
	AckActivity         = "toolkit_library_sync_ack_activity"
	FailureActivity     = "toolkit_library_sync_failure_activity"
	MaxVisitedCursors   = 1000
)

// Progress exposes only counts, phase and saved identities; inputs: workflow state; outputs: safe query projection.
// Effects: none. Parent renders explicit partial/blocked/error states without using legal warning banners as validation.
type Progress struct {
	OperationID string `json:"operation_id,omitempty"`
	Child       string `json:"child,omitempty"`
	Phase       string `json:"phase"`
	Status      string `json:"status"`
	Observed    int    `json:"observed"`
	Skipped     int    `json:"skipped"`
	Blocked     int    `json:"blocked"`
	Failed      int    `json:"failed"`
}

// CycleInput checkpoints only scope child, cursor and counts across bounded Temporal histories.
// Inputs: empty input starts case-law, then benchbooks and reference-data; outputs: CycleResult.
// Effects: Activities/continue-as-new only. Visited includes every completed page cursor for the current child.
type CycleInput struct {
	Child    string   `json:"child,omitempty"`
	Cursor   Cursor   `json:"cursor"`
	Visited  []Cursor `json:"visited,omitempty"`
	Progress Progress `json:"progress"`
}

// CycleResult reports completed scope observation or visible failures without personal bodies.
type CycleResult struct {
	Progress
	Complete bool `json:"complete"`
}

// WriteInput carries a durable app outbox UUID, never payload bytes, into Temporal.
type WriteInput struct {
	OperationID string `json:"operation_id"`
}

// WriteRequest identifies one prepared payload and workflow attempt; inputs/outputs: references only; effects: none in this type.
type WriteRequest struct {
	Prepared  Handle `json:"prepared"`
	AttemptID string `json:"attempt_id"`
}

// FailureInput names a leased operation and safe reason for persistent error reporting.
type FailureInput struct {
	Claim  Handle `json:"claim"`
	Status string `json:"status"`
	Code   string `json:"error_code"`
}

// WorkflowID returns the idempotent operation workflow ID; inputs: UUID; outputs: stable ID or error; effects: none.
func WorkflowID(id string) (string, error) {
	if !uuidID.MatchString(id) {
		return "", errors.New("invalid library sync operation UUID")
	}
	return "library-sync-write-" + id, nil
}

func activityContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 4 * time.Minute, ScheduleToCloseTimeout: 15 * time.Minute, HeartbeatTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, MaximumInterval: 10 * time.Second, MaximumAttempts: 3}})
}

// ToolkitLibrarySyncCycleWorkflow observes current B2 legal versions through independently tracked units.
// Inputs: scope checkpoint only; outputs: counts/status. Effects: bounded listing, hashing, retention, extraction and guarded staging Activities.
// Choose a separate parent-owned fifteen-minute schedule with overlap SKIP; never replace catalog-only Super Index discovery.
func ToolkitLibrarySyncCycleWorkflow(ctx workflow.Context, in CycleInput) (CycleResult, error) {
	if in.Child == "" {
		in.Child = "case-law"
	}
	if in.Child != "case-law" && in.Child != "benchbooks" && in.Child != "reference-data" {
		return CycleResult{}, temporal.NewNonRetryableApplicationError("invalid legal child", "InvalidSyncScope", nil)
	}
	progress := in.Progress
	progress.Child = in.Child
	progress.Status = Pending
	if err := workflow.SetQueryHandler(ctx, ProgressQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return CycleResult{}, err
	}
	base := activityContext(ctx)
	visited := map[Cursor]bool{}
	for _, cursor := range in.Visited {
		if visited[cursor] {
			return CycleResult{}, temporal.NewNonRetryableApplicationError("duplicate checkpoint cursor", "InvalidSyncCursor", nil)
		}
		visited[cursor] = true
	}
	for pageNumber := 0; pageNumber < MaxCyclePages; pageNumber++ {
		if visited[in.Cursor] || len(in.Visited) >= MaxVisitedCursors {
			progress.Status = Blocked
			return CycleResult{Progress: progress}, temporal.NewNonRetryableApplicationError("version cursor cycle or coverage budget", "SyncListingIncomplete", nil)
		}
		progress.Phase = "listing"
		var page Page
		if err := workflow.ExecuteActivity(base, ListActivity, ListInput{Child: in.Child, Cursor: in.Cursor}).Get(ctx, &page); err != nil {
			progress.Status = Blocked
			return CycleResult{Progress: progress}, err
		}
		if len(page.Objects) > MaxPage || (!page.Complete && (page.Next.Key == "" || page.Next == in.Cursor || visited[page.Next])) {
			progress.Status = Blocked
			return CycleResult{Progress: progress}, temporal.NewNonRetryableApplicationError("invalid version page/continuation", "SyncListingIncomplete", nil)
		}
		visited[in.Cursor] = true
		in.Visited = append(in.Visited, in.Cursor)
		for _, obj := range page.Objects {
			progress.Phase = "checking_observation"
			var seen bool
			if err := workflow.ExecuteActivity(base, SeenActivity, obj).Get(ctx, &seen); err != nil {
				progress.Failed++
				continue
			}
			if seen {
				progress.Skipped++
				continue
			}
			var h Handle
			progress.Phase = "hashing_original"
			if err := workflow.ExecuteActivity(base, HashSourceActivity, obj).Get(ctx, &h); err != nil {
				progress.Failed++
				continue
			}
			progress.Phase = "retaining_original"
			var retained Handle
			if err := workflow.ExecuteActivity(base, RetainActivity, h).Get(ctx, &retained); err != nil {
				progress.Failed++
				continue
			}
			progress.Phase = "extracting"
			var extracted Handle
			if err := workflow.ExecuteActivity(base, ExtractActivity, retained).Get(ctx, &extracted); err != nil {
				progress.Failed++
				continue
			}
			progress.Phase = "staging_observation"
			var out Outcome
			if err := workflow.ExecuteActivity(base, ObserveActivity, extracted).Get(ctx, &out); err != nil {
				progress.Failed++
				continue
			}
			progress.Observed++
			if out.Status == Blocked || out.Status == Conflicted {
				progress.Blocked++
			}
		}
		if page.Complete {
			switch in.Child {
			case "case-law":
				in.Child = "benchbooks"
			case "benchbooks":
				in.Child = "reference-data"
			default:
				progress.Phase = "complete"
				progress.Status = "observed"
				if progress.Failed > 0 || progress.Blocked > 0 {
					progress.Status = "partial"
				}
				return CycleResult{Progress: progress, Complete: true}, nil
			}
			in.Cursor = Cursor{}
			in.Visited = nil
			visited = map[Cursor]bool{}
			progress.Child = in.Child
		} else {
			in.Cursor = page.Next
		}
	}
	in.Progress = progress
	return CycleResult{Progress: progress}, workflow.NewContinueAsNewError(ctx, CycleWorkflowName, in)
}

// ToolkitLibrarySyncWriteWorkflow exports one saved revision, then reconciles retained versions before backend CAS.
// Inputs: outbox UUID only; outputs: durable safe outcome. Effects: separately tracked Activities, never legal publication.
// Choose after atomic app outbox capture; intent recovery precedes reconciliation and every non-synced outcome remains visible.
func ToolkitLibrarySyncWriteWorkflow(ctx workflow.Context, in WriteInput) (Outcome, error) {
	if _, err := WorkflowID(in.OperationID); err != nil {
		return Outcome{}, temporal.NewNonRetryableApplicationError(err.Error(), "InvalidSyncOperation", nil)
	}
	progress := Progress{OperationID: in.OperationID, Phase: "claiming", Status: Pending}
	if err := workflow.SetQueryHandler(ctx, ProgressQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return Outcome{}, err
	}
	base := activityContext(ctx)
	attempt := workflow.GetInfo(ctx).WorkflowExecution.RunID
	var claim Handle
	if err := workflow.ExecuteActivity(base, ClaimActivity, OperationInput{OperationID: in.OperationID, AttemptID: attempt}).Get(ctx, &claim); err != nil {
		return Outcome{}, err
	}
	if claim.Status == Synced {
		return Outcome{OperationID: in.OperationID, Status: Synced}, nil
	}
	fail := func(code string, cause error) (Outcome, error) {
		progress.Status = WriteUnknown
		progress.Phase = "failed"
		var out Outcome
		disconnected, _ := workflow.NewDisconnectedContext(ctx)
		if err := workflow.ExecuteActivity(activityContext(disconnected), FailureActivity, FailureInput{Claim: claim, Status: WriteUnknown, Code: code}).Get(disconnected, &out); err != nil {
			return Outcome{OperationID: in.OperationID, Status: WriteUnknown, Code: code}, temporal.NewNonRetryableApplicationError("sync failure persistence failed", "SyncFailureUnrecorded", cause)
		}
		return out, temporal.NewNonRetryableApplicationError("sync operation incomplete", "SyncOperationIncomplete", cause, out)
	}
	progress.Phase = "preparing_payload"
	var prepared Handle
	if err := workflow.ExecuteActivity(base, PrepareActivity, claim).Get(ctx, &prepared); err != nil {
		return fail("PAYLOAD_PREPARATION_FAILED", err)
	}
	progress.Phase = "writing"
	var written Handle
	// Mutations get one Activity attempt. The backend one-shot intent also protects manual/activity replay.
	writeCtx := workflow.WithRetryPolicy(base, temporal.RetryPolicy{MaximumAttempts: 1})
	_ = workflow.ExecuteActivity(writeCtx, WriteActivity, WriteRequest{Prepared: prepared, AttemptID: attempt}).Get(ctx, &written)
	progress.Phase = "recovering_intent"
	var refreshed Handle
	if err := workflow.ExecuteActivity(base, RefreshActivity, RecoveryInput{Previous: claim, AttemptID: attempt}).Get(ctx, &refreshed); err != nil {
		return fail("INTENT_RECOVERY_FAILED", err)
	}
	claim = refreshed
	if claim.Status == Synced {
		return Outcome{OperationID: in.OperationID, Status: Synced}, nil
	}
	progress.Phase = "listing_history"
	var plan HistoryResult
	if err := workflow.ExecuteActivity(base, HistoryActivity, claim).Get(ctx, &plan); err != nil {
		return fail("VERSION_HISTORY_UNAVAILABLE", err)
	}
	if plan.Count < 0 || plan.Count > MaxPage {
		return fail("RECONCILIATION_PLAN_INVALID", nil)
	}
	progress.Phase = "hashing_versions"
	checks := []Handle{}
	for first := 0; first < plan.Count; first += 4 {
		count := 4
		if plan.Count-first < count {
			count = plan.Count - first
		}
		futures := make([]workflow.Future, count)
		for i := 0; i < count; i++ {
			futures[i] = workflow.ExecuteActivity(base, HashVersionActivity, HashInput{Plan: plan.Handle, Index: first + i})
		}
		for _, f := range futures {
			var check Handle
			if err := f.Get(ctx, &check); err != nil {
				progress.Failed++
				continue
			}
			checks = append(checks, check)
		}
	}
	progress.Phase = "checking_current_version"
	var current Handle
	if err := workflow.ExecuteActivity(base, CurrentActivity, plan.Handle).Get(ctx, &current); err != nil {
		return fail("CURRENT_VERSION_UNAVAILABLE", err)
	}
	progress.Phase = "acknowledging"
	var out Outcome
	if err := workflow.ExecuteActivity(base, AckActivity, AckInput{Plan: plan.Handle, Checks: checks, Current: current}).Get(ctx, &out); err != nil {
		return fail("POINTER_ACK_FAILED", err)
	}
	progress.Status = out.Status
	progress.Phase = "complete"
	if out.Status != Synced {
		return out, temporal.NewNonRetryableApplicationError("sync retained blocked/conflict/retry outcome", "SyncNotCleared", nil, out)
	}
	return out, nil
}

// WorkflowRegistrar is the existing narrow worker registration surface; inputs: functions/options; effects: registration only.
type WorkflowRegistrar interface {
	RegisterWorkflowWithOptions(any, workflow.RegisterOptions)
}

// RegisterWorkflows installs the exact cycle/write names without editing the shared worker.
// Inputs: parent worker; outputs: none. Effects: registration only; parent owns scheduling and runtime composition.
func RegisterWorkflows(r WorkflowRegistrar) {
	r.RegisterWorkflowWithOptions(ToolkitLibrarySyncCycleWorkflow, workflow.RegisterOptions{Name: CycleWorkflowName})
	r.RegisterWorkflowWithOptions(ToolkitLibrarySyncWriteWorkflow, workflow.RegisterOptions{Name: WriteWorkflowName})
}
