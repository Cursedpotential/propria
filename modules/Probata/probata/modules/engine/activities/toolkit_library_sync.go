// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"context"
	"errors"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	"go.temporal.io/sdk/activity"
)

// ToolkitLibrarySyncActivities supplies independently tracked original-observation and outbox-export units.
// Inputs: configured current B2/private backend service. Outputs: references, metadata and explicit statuses only.
// Effects: each method performs its named unit; workflows own retries/sequencing and the parent owns worker wiring.
type ToolkitLibrarySyncActivities struct{ Service *librarysync.Service }

// RegisterToolkitLibrarySyncActivities installs fifteen exact workflow-facing names without changing the shared registrar.
// Inputs: parent worker/service. Outputs: none. Effects: registration only; use alongside librarysync.RegisterWorkflows.
func RegisterToolkitLibrarySyncActivities(r ActivityRegistrar, a ToolkitLibrarySyncActivities) {
	r.RegisterActivityWithOptions(a.ListPage, activity.RegisterOptions{Name: librarysync.ListActivity})
	r.RegisterActivityWithOptions(a.Seen, activity.RegisterOptions{Name: librarysync.SeenActivity})
	r.RegisterActivityWithOptions(a.HashSource, activity.RegisterOptions{Name: librarysync.HashSourceActivity})
	r.RegisterActivityWithOptions(a.Retain, activity.RegisterOptions{Name: librarysync.RetainActivity})
	r.RegisterActivityWithOptions(a.Extract, activity.RegisterOptions{Name: librarysync.ExtractActivity})
	r.RegisterActivityWithOptions(a.Observe, activity.RegisterOptions{Name: librarysync.ObserveActivity})
	r.RegisterActivityWithOptions(a.Claim, activity.RegisterOptions{Name: librarysync.ClaimActivity})
	r.RegisterActivityWithOptions(a.Prepare, activity.RegisterOptions{Name: librarysync.PrepareActivity})
	r.RegisterActivityWithOptions(a.Write, activity.RegisterOptions{Name: librarysync.WriteActivity})
	r.RegisterActivityWithOptions(a.Refresh, activity.RegisterOptions{Name: librarysync.RefreshActivity})
	r.RegisterActivityWithOptions(a.History, activity.RegisterOptions{Name: librarysync.HistoryActivity})
	r.RegisterActivityWithOptions(a.HashVersion, activity.RegisterOptions{Name: librarysync.HashVersionActivity})
	r.RegisterActivityWithOptions(a.Current, activity.RegisterOptions{Name: librarysync.CurrentActivity})
	r.RegisterActivityWithOptions(a.Acknowledge, activity.RegisterOptions{Name: librarysync.AckActivity})
	r.RegisterActivityWithOptions(a.Failure, activity.RegisterOptions{Name: librarysync.FailureActivity})
}

// ListPage reads one provider version page; inputs: child/cursor; outputs: metadata only; effects: bounded listing.
func (a ToolkitLibrarySyncActivities) ListPage(ctx context.Context, in librarysync.ListInput) (librarysync.Page, error) {
	return syncUnit(ctx, a.Service, in.Child, "list", func() (librarysync.Page, error) { return a.Service.ListPage(ctx, in) })
}

// Seen checks a durable exact-version observation; inputs: object; outputs: bool; effects: backend metadata read.
func (a ToolkitLibrarySyncActivities) Seen(ctx context.Context, in librarysync.Object) (bool, error) {
	return syncUnit(ctx, a.Service, in.VersionID, "seen", func() (bool, error) { return a.Service.CheckObservation(ctx, in) })
}

// HashSource hashes one complete original version; inputs: metadata; outputs: evidence ref; effects: pinned GET/hash and descriptor write.
func (a ToolkitLibrarySyncActivities) HashSource(ctx context.Context, in librarysync.Object) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.VersionID, "hash_source", func() (librarysync.Handle, error) { return a.Service.HashSource(ctx, in) })
}

// Retain stages independently hashed bytes for existing extraction; inputs: evidence ref; outputs: raw ref; effects: versioned derivative retention.
func (a ToolkitLibrarySyncActivities) Retain(ctx context.Context, in librarysync.Handle) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.Ref.VersionID, "retain", func() (librarysync.Handle, error) { return a.Service.RetainSource(ctx, in) })
}

// Extract invokes the existing input-pinned extractor; inputs: raw ref; outputs: extraction ref/status; effects: parser/descriptor only.
func (a ToolkitLibrarySyncActivities) Extract(ctx context.Context, in librarysync.Handle) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.Ref.VersionID, "extract", func() (librarysync.Handle, error) { return a.Service.ExtractSource(ctx, in) })
}

// Observe stages evidence through existing proposal gates; inputs: evidence ref; outputs: IDs/status; effects: guarded backend import.
func (a ToolkitLibrarySyncActivities) Observe(ctx context.Context, in librarysync.Handle) (librarysync.Outcome, error) {
	return syncUnit(ctx, a.Service, in.Ref.VersionID, "observe", func() (librarysync.Outcome, error) { return a.Service.StageObservation(ctx, in) })
}

// Claim leases one durable outbox operation; inputs: UUID/attempt; outputs: immutable ref; effects: backend claim/descriptor retention.
func (a ToolkitLibrarySyncActivities) Claim(ctx context.Context, in librarysync.OperationInput) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.OperationID, "claim", func() (librarysync.Handle, error) { return a.Service.ClaimOperation(ctx, in) })
}

// Prepare seals exact immutable payload bytes; inputs: claim ref; outputs: prepared ref; effects: backend read/hash and derivative retention.
func (a ToolkitLibrarySyncActivities) Prepare(ctx context.Context, in librarysync.Handle) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.OperationID, "prepare", func() (librarysync.Handle, error) { return a.Service.PreparePayload(ctx, in) })
}

// Write consumes a durable intent and appends at most one version; inputs: prepared ref/attempt; outputs: safe status/ref; effects: one optional PUT.
func (a ToolkitLibrarySyncActivities) Write(ctx context.Context, in librarysync.WriteRequest) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.Prepared.OperationID, "write", func() (librarysync.Handle, error) { return a.Service.WriteVersion(ctx, in.Prepared, in.AttemptID) })
}

// Refresh recovers persisted intent and lease without rebasing content; inputs: prior ref/attempt; outputs: fresh ref; effects: backend recovery.
func (a ToolkitLibrarySyncActivities) Refresh(ctx context.Context, in librarysync.RecoveryInput) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.Previous.OperationID, "refresh", func() (librarysync.Handle, error) { return a.Service.RefreshOperation(ctx, in) })
}

// History lists bounded exact-key retained versions; inputs: claim ref; outputs: hash-task plan; effects: listing/descriptor retention.
func (a ToolkitLibrarySyncActivities) History(ctx context.Context, in librarysync.Handle) (librarysync.HistoryResult, error) {
	return syncUnit(ctx, a.Service, in.OperationID, "history", func() (librarysync.HistoryResult, error) { return a.Service.PlanReconciliation(ctx, in) })
}

// HashVersion independently hashes one retained reconciliation candidate; inputs: plan/index; outputs: hash ref; effects: pinned GET/hash.
func (a ToolkitLibrarySyncActivities) HashVersion(ctx context.Context, in librarysync.HashInput) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.Plan.OperationID, "hash_version", func() (librarysync.Handle, error) { return a.Service.HashReconciliation(ctx, in) })
}

// Current observes the current version after hashes; inputs: plan ref; outputs: HEAD evidence ref; effects: one HEAD/descriptor retention.
func (a ToolkitLibrarySyncActivities) Current(ctx context.Context, in librarysync.Handle) (librarysync.Handle, error) {
	return syncUnit(ctx, a.Service, in.OperationID, "current", func() (librarysync.Handle, error) { return a.Service.CheckCurrent(ctx, in) })
}

// Acknowledge submits complete evidence for independent backend CAS; inputs: plan/hash/current refs; outputs: durable outcome; effects: pointer transaction.
func (a ToolkitLibrarySyncActivities) Acknowledge(ctx context.Context, in librarysync.AckInput) (librarysync.Outcome, error) {
	return syncUnit(ctx, a.Service, in.Plan.OperationID, "acknowledge", func() (librarysync.Outcome, error) { return a.Service.Acknowledge(ctx, in) })
}

// Failure persists explicit incomplete state; inputs: claim/status/code; outputs: safe outcome; effects: backend failure transaction.
func (a ToolkitLibrarySyncActivities) Failure(ctx context.Context, in librarysync.FailureInput) (librarysync.Outcome, error) {
	return syncUnit(ctx, a.Service, in.Claim.OperationID, "failure", func() (librarysync.Outcome, error) { return a.Service.RecordFailure(ctx, in.Claim, in.Status, in.Code) })
}

// syncUnit checks configuration and reports safe identity/phase heartbeats around one independent service unit.
// Inputs: Activity context and unit closure; outputs: that unit's result. Effects: heartbeat only, never logs bodies or credentials.
func syncUnit[T any](ctx context.Context, service *librarysync.Service, id, phase string, run func() (T, error)) (T, error) {
	if service == nil {
		var zero T
		return zero, errors.New("library sync worker is not configured")
	}
	if !activity.IsActivity(ctx) {
		return run()
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	pulse := func() { activity.RecordHeartbeat(ctx, map[string]string{"reference_id": id, "phase": phase}) }
	pulse()
	go func() {
		defer close(finished)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				pulse()
			}
		}
	}()
	defer func() { close(done); <-finished }()
	return run()
}
