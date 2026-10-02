// Byline: Claude Code · Opus 5.5 · 2026-10-02
package activities

import (
	"context"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

type recordedDecision struct {
	handle, reason, actor string
	approved              bool
	selection, options    proffer.Ref
}

type decisionRecorder struct{ calls []recordedDecision }

func (d *decisionRecorder) RecordDecision(_ context.Context, handle string, approved bool, reason, actor string, selection, options proffer.Ref) error {
	d.calls = append(d.calls, recordedDecision{handle, reason, actor, approved, selection, options})
	return nil
}

func cleanApprovalRequest() proffer.AutoApprovalRequest {
	checks := make([]proffer.AutoApprovalCheck, 0, len(proffer.AutoApprovalChecks))
	for _, id := range proffer.AutoApprovalChecks {
		checks = append(checks, proffer.AutoApprovalCheck{Stage: id, Status: proffer.StatusSuccess, ReceiptRef: proffer.Ref(string(id) + "-receipt")})
	}
	return proffer.AutoApprovalRequest{RequestID: "req-1", PreviewHandle: "handle-1", SelectionRef: "sel-1", ParserOptionsRef: "opts-1", Checks: checks}
}

func TestAutoApprovalRecordsTheSameDecisionAHumanWritesAsAutomatic(t *testing.T) {
	store := &decisionRecorder{}
	result, err := AutoApprovalActivity{Store: store}.Record(context.Background(), cleanApprovalRequest())
	if err != nil {
		t.Fatalf("Record error = %v", err)
	}
	if result.Status != proffer.StatusSuccess || result.ReceiptRef == "" || result.Stage != stagegraph.RecordAutoApproval {
		t.Fatalf("result = %+v", result)
	}
	if len(store.calls) != 1 {
		t.Fatalf("decisions recorded = %d, want 1", len(store.calls))
	}
	call := store.calls[0]
	if !call.approved || call.actor != stagegraph.AutoApprovalActor || call.handle != "handle-1" || call.selection != "sel-1" || call.options != "opts-1" {
		t.Fatalf("decision = %+v, want an approval by %s", call, stagegraph.AutoApprovalActor)
	}
	for _, id := range proffer.AutoApprovalChecks {
		if !strings.Contains(call.reason, string(id)+"=success("+string(id)+"-receipt)") {
			t.Fatalf("reason %q does not name check %s by receipt", call.reason, id)
		}
	}
	// A retry must produce the identical decision so the insert stays idempotent.
	if again := AutoApprovalReason(cleanApprovalRequest().Checks); again != call.reason || len(call.reason) > 4000 {
		t.Fatalf("reason is not deterministic or exceeds the column bound: %q", call.reason)
	}
}

func TestAutoApprovalRefusesAnyCheckThatDidNotPass(t *testing.T) {
	for _, mutate := range []func(*proffer.AutoApprovalRequest){
		func(r *proffer.AutoApprovalRequest) { r.Checks[1].Status = proffer.StatusNotApplicable },
		func(r *proffer.AutoApprovalRequest) { r.Checks[4].ReceiptRef = "" },
		func(r *proffer.AutoApprovalRequest) { r.Checks = r.Checks[:3] },
		func(r *proffer.AutoApprovalRequest) { r.PreviewHandle = "" },
	} {
		request := cleanApprovalRequest()
		mutate(&request)
		store := &decisionRecorder{}
		if _, err := (AutoApprovalActivity{Store: store}).Record(context.Background(), request); err == nil {
			t.Fatalf("request %+v was approved", request)
		}
		if len(store.calls) != 0 {
			t.Fatalf("a refused request recorded a decision")
		}
	}
}

// Owner 2026-10-02 option A. Byline: Claude Code · Opus 5.5 · 2026-10-02
func TestAutoApprovalPassesNotApplicableByteCoverageOnlyForLocatorlessFormats(t *testing.T) {
	byteCoverageNA := func(format string) proffer.AutoApprovalRequest {
		request := cleanApprovalRequest()
		request.Checks[1].Status = proffer.StatusNotApplicable
		request.DetectedFormat = format
		return request
	}
	for _, format := range []string{"ndjson", "facebook_messenger_json", "facebook_messenger_html", "generic_html_document"} {
		store := &decisionRecorder{}
		if _, err := (AutoApprovalActivity{Store: store}).Record(context.Background(), byteCoverageNA(format)); err != nil {
			t.Fatalf("%s: a receipted not_applicable byte-coverage check was refused: %v", format, err)
		}
		if len(store.calls) != 1 || !strings.Contains(store.calls[0].reason, "reconcile_byte_coverage_activity=not_applicable(") {
			t.Fatalf("%s: decision = %+v", format, store.calls)
		}
	}
	for _, format := range []string{"smsbackuprestore_xml", "xml", "pdf", ""} {
		store := &decisionRecorder{}
		if _, err := (AutoApprovalActivity{Store: store}).Record(context.Background(), byteCoverageNA(format)); err == nil || len(store.calls) != 0 {
			t.Fatalf("%s: not_applicable byte coverage passed for a format that has byte locators", format)
		}
	}
	// Every other check must still be a success, even for a locator-less format.
	for _, index := range []int{0, 2, 3, 4} {
		request := cleanApprovalRequest()
		request.DetectedFormat = "ndjson"
		request.Checks[index].Status = proffer.StatusNotApplicable
		store := &decisionRecorder{}
		if _, err := (AutoApprovalActivity{Store: store}).Record(context.Background(), request); err == nil || len(store.calls) != 0 {
			t.Fatalf("check %s not_applicable passed for ndjson", request.Checks[index].Stage)
		}
	}
	// A not_applicable byte-coverage check still needs its receipt.
	request := byteCoverageNA("ndjson")
	request.Checks[1].ReceiptRef = ""
	if _, err := (AutoApprovalActivity{Store: &decisionRecorder{}}).Record(context.Background(), request); err == nil {
		t.Fatalf("an unreceipted not_applicable check passed")
	}
}
