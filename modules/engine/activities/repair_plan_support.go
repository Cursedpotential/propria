// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// The repair-plan Activities (owner ratification 2026-09-25, option A;
// docs/pending-review/2026-09-25-repair-workflow-builder.md). Five units, each
// one job, each scheduled by RepairPlanWorkflow (engine/repairplan):
//
//   repair_validate_plan_activity        re-run the plan validator on the worker
//   repair.find_other_version            name another existing copy (reads only)
//   repair.salvage_truncated_xml         publish a salvaged derived copy
//   repair.lenient_decode                publish lenient derived NDJSON threads
//   repair_record_step_receipt_activity  record one append-only step receipt
//
// None orchestrates, none writes an original, and none moves source bytes
// through Temporal: every source travels as a locator and is streamed by the
// Activity that reads it (AGENTS.md ATOMICITY rules 1, 5 and 6).

package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// RepairPlanValidateActivity re-validates a plan with THIS worker's storage
// configuration before any step runs. The starter validates for the
// Workbench; the writer's own configuration has the final word.
type RepairPlanValidateActivity struct {
	Environment repairplan.Environment
}

// ValidatePlan returns the full validation. A refused plan is a successful
// Activity with OK=false (the workflow reports the checks); only an outage
// reading the plan's Review run is an error, and it is retried.
func (a RepairPlanValidateActivity) ValidatePlan(ctx context.Context, request repairplan.ValidatePlanRequest) (repairplan.ValidatedPlan, error) {
	if a.Environment.Anchors == nil {
		return repairplan.ValidatedPlan{}, errors.New("repair plan validation: no anchor resolver is configured on this worker")
	}
	if err := request.Plan.ShapeError(); err != nil {
		return repairplan.ValidatedPlan{}, stopRetryingPermanent(permanent(err))
	}
	return a.Environment.Validate(ctx, request.Plan)
}

// RepairStepReceiptStore persists one append-only step receipt and returns
// its reference. A retried identical request returns the first receipt.
type RepairStepReceiptStore interface {
	RecordRepairStepReceipt(ctx context.Context, request repairplan.ReceiptRequest, attempt int32) (receiptRef string, err error)
}

// RepairStepReceiptActivity records one receipt per plan step.
type RepairStepReceiptActivity struct {
	Store   RepairStepReceiptStore
	Attempt Attempt
}

// RecordStepReceipt validates the request and records it.
func (a RepairStepReceiptActivity) RecordStepReceipt(ctx context.Context, request repairplan.ReceiptRequest) (repairplan.ReceiptResult, error) {
	if a.Store == nil {
		return repairplan.ReceiptResult{}, errors.New("repair step receipt: store is required")
	}
	if strings.TrimSpace(request.WorkflowID) == "" || strings.TrimSpace(request.RunID) == "" ||
		strings.TrimSpace(request.StepID) == "" || strings.TrimSpace(request.Activity) == "" ||
		strings.TrimSpace(request.SourceVersionID) == "" || request.StepIndex < 0 {
		return repairplan.ReceiptResult{}, stopRetryingPermanent(permanent(errors.New(
			"repair step receipt requires workflow, run, step, activity and source version references")))
	}
	switch request.Status {
	case repairplan.ReceiptSuccess:
		if request.Result == nil || strings.TrimSpace(request.Result.OutputRef) == "" {
			return repairplan.ReceiptResult{}, stopRetryingPermanent(permanent(errors.New("a success receipt requires the step's output reference")))
		}
	case repairplan.ReceiptFailed:
		if strings.TrimSpace(request.Error) == "" {
			return repairplan.ReceiptResult{}, stopRetryingPermanent(permanent(errors.New("a failure receipt requires the failure reason")))
		}
	default:
		return repairplan.ReceiptResult{}, stopRetryingPermanent(permanent(fmt.Errorf("unknown receipt status %q", request.Status)))
	}
	attempt := int32(1)
	if a.Attempt != nil {
		if value := a.Attempt(ctx); value >= 1 {
			attempt = value
		}
	}
	ref, err := a.Store.RecordRepairStepReceipt(ctx, request, attempt)
	if err != nil {
		return repairplan.ReceiptResult{}, fmt.Errorf("record repair step receipt: %w", err)
	}
	if strings.TrimSpace(ref) == "" {
		return repairplan.ReceiptResult{}, errors.New("the receipt store returned no reference")
	}
	return repairplan.ReceiptResult{ReceiptRef: ref}, nil
}

// RepairPlanActivities groups every repair-plan Activity for registration.
type RepairPlanActivities struct {
	Validate RepairPlanValidateActivity
	Receipts RepairStepReceiptActivity
	Find     RepairFindOtherVersionActivity
	Salvage  RepairSalvageTruncatedXMLActivity
	Lenient  RepairLenientDecodeActivity
}

// RegisterRepairPlanActivities installs the five repair-plan Activities under
// their stage-graph names. They are standalone: RepairPlanWorkflow is their
// only scheduler and none belongs to the 26-stage Proffer graph.
func RegisterRepairPlanActivities(registrar ActivityRegistrar, repair RepairPlanActivities) {
	registrar.RegisterActivityWithOptions(repair.Validate.ValidatePlan, activity.RegisterOptions{Name: string(stagegraph.RepairValidatePlan)})
	registrar.RegisterActivityWithOptions(repair.Receipts.RecordStepReceipt, activity.RegisterOptions{Name: string(stagegraph.RepairRecordStepReceipt)})
	registrar.RegisterActivityWithOptions(repair.Find.FindOtherVersion, activity.RegisterOptions{Name: string(stagegraph.RepairFindOtherVersion)})
	registrar.RegisterActivityWithOptions(repair.Salvage.SalvageTruncatedXML, activity.RegisterOptions{Name: string(stagegraph.RepairSalvageTruncatedXML)})
	registrar.RegisterActivityWithOptions(repair.Lenient.LenientDecode, activity.RegisterOptions{Name: string(stagegraph.RepairLenientDecode)})
}

// repairLocator is one object-store coordinate; the scheme names a configured
// store, never a provider in code.
type repairLocator struct {
	Scheme, Bucket, Key string
}

func (l repairLocator) URI() string { return fmt.Sprintf("%s://%s/%s", l.Scheme, l.Bucket, l.Key) }

// nonStoreSchemes name locators that are not a publishable object store.
var nonStoreSchemes = map[string]bool{"upload": true, "file": true, "http": true, "https": true, "s3": true}

// parseRepairLocator accepts only <scheme>://<bucket>/<key> in an object store.
func parseRepairLocator(ref string) (repairLocator, error) {
	parsed, err := url.Parse(strings.TrimSpace(ref))
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
		return repairLocator{}, fmt.Errorf("%q is not an object-store locator", ref)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "" || nonStoreSchemes[scheme] {
		return repairLocator{}, fmt.Errorf("%q is not in an object store; repairs read and publish through a configured store", ref)
	}
	key, err := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/"))
	if err != nil || key == "" || strings.HasSuffix(key, "/") || strings.Contains(key, `\`) {
		return repairLocator{}, fmt.Errorf("%q does not name one object", ref)
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return repairLocator{}, fmt.Errorf("%q has a dot segment", ref)
		}
	}
	return repairLocator{Scheme: scheme, Bucket: parsed.Host, Key: key}, nil
}

// decodeStepParams reads a step's params into dest, refusing unknown knobs.
func decodeStepParams(raw json.RawMessage, dest any) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		trimmed = "{}"
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("step params: %w", err)
	}
	return nil
}

// startIdleHeartbeat keeps a long Activity alive across a quiet stretch (a
// slow first byte, one huge upload). Progress callbacks cover the rest.
func startIdleHeartbeat(ctx context.Context, heartbeat Heartbeat, stage stagegraph.StageID, every time.Duration) func() {
	if heartbeat == nil {
		return func() {}
	}
	if every <= 0 {
		every = 20 * time.Second
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				heartbeat(ctx, Progress{Stage: stage})
			}
		}
	}()
	return func() { close(done) }
}

// repairSummary encodes a step summary for the Workbench.
func repairSummary(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return encoded
}

// NewRepairPlanActivities binds Temporal heartbeats and attempt numbers.
func NewRepairPlanActivities(repair RepairPlanActivities) RepairPlanActivities {
	heartbeat := func(ctx context.Context, progress Progress) { activity.RecordHeartbeat(ctx, progress) }
	repair.Receipts.Attempt = func(ctx context.Context) int32 { return activity.GetInfo(ctx).Attempt }
	repair.Salvage.Heartbeat = heartbeat
	repair.Lenient.Heartbeat = heartbeat
	return repair
}
