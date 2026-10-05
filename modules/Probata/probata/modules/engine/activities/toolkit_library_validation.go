// Byline: Codex · GPT-6.1 · 2026-10-04.
package activities

import (
	"context"
	"errors"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"go.temporal.io/sdk/activity"
)

// ToolkitLibraryValidationActivities supplies four atomic reference-only library validation bodies.
// Inputs: configured shared FCT/B2/extractor/NIM service. Outputs: durable references and bounded statuses.
// Side effects: preparation, source reads, verification and trusted receipt writes remain independently scheduled.
// Choose for ToolkitLibraryValidationWorkflow rather than widening an existing import/extraction Activity.
type ToolkitLibraryValidationActivities struct{ Service *libraryvalidation.Service }

// RegisterToolkitLibraryValidationActivities registers exact workflow-facing names without altering the shared registrar.
// Inputs: worker/service. Outputs: none. Effects: worker registration only; parent owns worker/start route composition.
func RegisterToolkitLibraryValidationActivities(registrar ActivityRegistrar, acts ToolkitLibraryValidationActivities) {
	registrar.RegisterActivityWithOptions(acts.Prepare, activity.RegisterOptions{Name: libraryvalidation.PrepareActivity})
	registrar.RegisterActivityWithOptions(acts.SourceSnapshot, activity.RegisterOptions{Name: libraryvalidation.SnapshotActivity})
	registrar.RegisterActivityWithOptions(acts.VerifyClaim, activity.RegisterOptions{Name: libraryvalidation.VerifyActivity})
	registrar.RegisterActivityWithOptions(acts.CommitReceipt, activity.RegisterOptions{Name: libraryvalidation.ReceiptActivity})
}

// Prepare fetches the saved shared proposal and persists its authenticated plan outside history.
// Inputs: proposal ID. Outputs: plan reference/count. Effects: case_record read and derivative descriptor write.
func (a ToolkitLibraryValidationActivities) Prepare(ctx context.Context, input libraryvalidation.LibraryValidationInput) (libraryvalidation.Plan, error) {
	if a.Service == nil {
		return libraryvalidation.Plan{}, errors.New("library validation worker is not configured")
	}
	stop := libraryValidationHeartbeat(ctx, input.ProposalID, -1, "prepare")
	defer stop()
	return a.Service.Prepare(ctx, input.ProposalID)
}

// SourceSnapshot acquires one official source and retains its provider-version-pinned raw bytes and descriptor.
// Inputs: plan/index only. Outputs: reference/status. Effects: bounded official HTTP read and derivative B2 writes.
func (a ToolkitLibraryValidationActivities) SourceSnapshot(ctx context.Context, input libraryvalidation.ClaimInput) (libraryvalidation.StepResult, error) {
	if a.Service == nil {
		return libraryvalidation.StepResult{}, errors.New("library validation worker is not configured")
	}
	stop := libraryValidationHeartbeat(ctx, input.Plan.ProposalID, input.Index, "source_snapshot")
	defer stop()
	return a.Service.SourceSnapshot(ctx, input)
}

// VerifyClaim independently checks exact snapshot bytes, pinpoint, quote fit and authenticated currency evidence.
// Inputs: plan/index/snapshot reference. Outputs: signed check reference/status. Effects: existing extractor, existing NIM client, derivative write.
func (a ToolkitLibraryValidationActivities) VerifyClaim(ctx context.Context, input libraryvalidation.ClaimInput) (libraryvalidation.StepResult, error) {
	if a.Service == nil {
		return libraryvalidation.StepResult{}, errors.New("library validation worker is not configured")
	}
	stop := libraryValidationHeartbeat(ctx, input.Plan.ProposalID, input.Index, "claim_verification")
	defer stop()
	return a.Service.VerifyClaim(ctx, input)
}

// CommitReceipt binds every ordered claim result to one proposal and commits only a trusted validation row.
// Inputs: plan/check references. Outputs: receipt identity and normalized aggregate status. Effects: signed derivative receipt and atomic DB commit.
// Choose after all checks; partial failures and provisional currency prevent VERIFIED_PRIMARY and never publish automatically.
func (a ToolkitLibraryValidationActivities) CommitReceipt(ctx context.Context, input libraryvalidation.FinishInput) (libraryvalidation.Result, error) {
	if a.Service == nil {
		return libraryvalidation.Result{}, errors.New("library validation worker is not configured")
	}
	stop := libraryValidationHeartbeat(ctx, input.Plan.ProposalID, -1, "receipt_commit")
	defer stop()
	return a.Service.Finish(ctx, input)
}

// libraryValidationHeartbeat reports only identity/index/phase during bounded external operations.
// Inputs: Activity context and safe coordinates. Outputs: stop function. Effects: Temporal heartbeats; no personal text or credentials.
func libraryValidationHeartbeat(ctx context.Context, id string, index int, phase string) func() {
	if !activity.IsActivity(ctx) {
		return func() {}
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	pulse := func() {
		activity.RecordHeartbeat(ctx, map[string]any{"proposal_id": id, "claim_index": index, "phase": phase})
	}
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
	return func() { close(done); <-finished }
}
