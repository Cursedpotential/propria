package proffer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/humancontext"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const contextHumanChangeID = "proffer-context-human-source-v1"

// HumanContextSummary preserves native-human lineage separately from AI package fields.
// Inputs: registered source and bounded stage receipts. Outputs: reference-only progress.
// Effects: none; choose for human-source-v1 viewing, publication and independent readback.
type HumanContextSummary struct {
	Source humancontext.Source `json:"source"`
	humancontext.Result
	FramedRef        string `json:"framed_ref,omitempty"`
	ExtractedRef     string `json:"extracted_ref,omitempty"`
	ChunkManifestRef string `json:"chunk_manifest_ref,omitempty"`
	PublicationRef   string `json:"publication_ref,omitempty"`
	VerificationRef  string `json:"verification_ref,omitempty"`
}

// contextHumanWorkflow sequences five native-human Activities after source registration.
// Inputs: source metadata, actual registration receipts and the actor-bound progress object.
// Outputs: native reference summary, including retained partial outcomes on failure.
// Effects: framing, DuckDB extraction, Python chunk planning, chunk publication and independent verification Activities;
// choose for SMS XML context without scheduling legacy ingest or AI package retention.
func contextHumanWorkflow(ctx workflow.Context, in WorkflowInput, registered ContextSourceRegistrationResult, summary *ContextSummary) (WorkflowResult, error) {
	source := humancontext.Source{}
	if in.HumanSource != nil {
		source = *in.HumanSource
	}
	// Registration and workflow coordinates always override caller-supplied identity pins.
	source.SourceRef, source.ProviderVersionID = string(in.SourceRef), in.ProviderVersionID
	source.SourceVersionRef, source.RegistrationReceiptRef = registered.SourceVersionRef, registered.ReceiptRef
	source.Format = in.DeclaredFormat
	human := &HumanContextSummary{Source: source}
	summary.Human = human
	finish := func(status, reason string) (WorkflowResult, error) {
		summary.Status, summary.Reason = status, reason
		resultStatus := StatusNotApplicable
		if status == "complete" {
			resultStatus = StatusSuccess
		}
		return WorkflowResult{SourceVersionRef: Ref(registered.SourceVersionRef), Status: resultStatus, Context: summary}, nil
	}
	request := humancontext.Request{ContractVersion: humancontext.Contract, WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID, Source: source}
	steps := []struct{ name, stage string }{
		{humancontext.FrameActivity, "framed"},
		{humancontext.ExtractActivity, "extracted"},
		{humancontext.ChunkActivity, "chunked"},
		{humancontext.IndexActivity, "published"},
		{humancontext.VerifyActivity, "verified"},
	}
	var extracted, chunked, published humancontext.Result
	partial := false
	for _, step := range steps {
		summary.Status = step.stage
		// Stay on the existing Go queue; Activities wrappers heartbeat every 15 seconds.
		options := workflow.ActivityOptions{ActivityID: step.name, StartToCloseTimeout: 2 * time.Hour,
			ScheduleToCloseTimeout: 6 * time.Hour, HeartbeatTimeout: time.Minute, WaitForCancellation: true,
			RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}}
		if step.stage == "chunked" {
			// Reuse the existing Python queue/options; the native planner receives refs only.
			options = chunkActivityOptions(2 * time.Hour)
			options.ActivityID = step.name
			options.WaitForCancellation = true
		}
		var result humancontext.Result
		if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), step.name, request).Get(ctx, &result); err != nil {
			return finish("partial", step.stage+"_failed")
		}
		if err := validateHumanContextResult(result, step.stage); err != nil {
			return finish("partial", step.stage+"_receipt_invalid")
		}
		human.Result = result
		request.ManifestRef = result.ManifestRef
		partial = partial || result.Status == "partial" || result.Pending > 0
		switch step.stage {
		case "framed":
			human.FramedRef = result.ManifestRef
		case "extracted":
			human.ExtractedRef, extracted = result.ManifestRef, result
		case "chunked":
			human.ChunkManifestRef, chunked = result.ManifestRef, result
			request.ChunkManifestRef = result.ManifestRef
			if result.Records != extracted.Records || result.Pending != extracted.Pending {
				return finish("partial", "chunk_source_count_mismatch")
			}
		case "published":
			human.PublicationRef, published = result.ManifestRef, result
			if result.IndexKind != humancontext.IndexKind || result.IndexKind == "native-human-message-projections-v1" {
				return finish("partial", "chunk_index_kind_mismatch")
			}
			if result.Records != chunked.Records || result.Pending != chunked.Pending || result.Chunks != chunked.Chunks || result.Published != chunked.Chunks {
				return finish("partial", "publication_count_mismatch")
			}
		case "verified":
			human.VerificationRef = result.ManifestRef
			if result.IndexKind != published.IndexKind || result.Records != published.Records || result.Pending != published.Pending || result.Chunks != published.Chunks || result.Published != published.Published || result.Verified != published.Published {
				return finish("partial", "verification_count_mismatch")
			}
		}
	}
	if partial {
		reason := "native_source_partial"
		if human.Pending > 0 {
			reason = "native_records_pending"
		}
		return finish("partial", reason)
	}
	return finish("complete", "")
}

// validateHumanContextResult validates a bounded native receipt without reading its bundle.
// Inputs: Activity result and expected stage. Output: error or nil. Effects: none.
// Choose after each human Activity; exact object comparison remains VerifyActivity's responsibility.
func validateHumanContextResult(result humancontext.Result, stage string) error {
	if result.ContractVersion != humancontext.Contract || result.Stage != stage ||
		(result.Status != "complete" && result.Status != "partial") {
		return errors.New("native human receipt contract, stage or status differs")
	}
	for _, ref := range []string{result.ManifestRef, result.ReceiptRef} {
		if strings.TrimSpace(ref) == "" || len(ref) > 4096 || strings.ContainsAny(ref, "\x00\r\n") {
			return errors.New("native human manifest and persisted receipt references are required within bounds")
		}
	}
	if strings.TrimSpace(result.ManifestRef) == "" || result.Records < 0 || result.Pending < 0 || result.Pending > result.Records ||
		result.Chunks < 0 || result.Published < 0 || result.Published > result.Chunks || result.Verified < 0 || result.Verified > result.Published {
		return fmt.Errorf("native human %s receipt has invalid reference or counts", stage)
	}
	return nil
}
