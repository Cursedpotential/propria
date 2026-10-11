package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// nativeContextStartRecorder records compact workflow inputs without connecting to Temporal.
// Inputs: start requests. Outputs: synthetic identifiers. Effects: in-memory capture only; choose for HTTP bridge tests.
type nativeContextStartRecorder struct{ inputs []proffer.WorkflowInput }

// Start captures the exact HTTP-to-workflow mapping for one synthetic request.
// Inputs: compact workflow input. Outputs: synthetic started receipt. Effects: memory only; never schedules work.
func (s *nativeContextStartRecorder) Start(_ context.Context, in proffer.WorkflowInput) (proffer.ContextStarted, error) {
	s.inputs = append(s.inputs, in)
	return proffer.ContextStarted{WorkflowID: in.RequestID, RunID: "synthetic-run"}, nil
}

// Status satisfies the unchanged context workflow interface without performing reads.
// Inputs: unused synthetic workflow ID. Outputs: empty progress. Effects: none; choose only in these start tests.
func (*nativeContextStartRecorder) Status(context.Context, string) (proffer.ContextProgress, error) {
	return proffer.ContextProgress{}, nil
}

// submitNativeContextBody invokes the body bridge with explicit synthetic actor and idempotency headers.
// Inputs: handler and request metadata. Outputs: HTTP recorder. Effects: in-memory request only; no service-auth bypass claim.
func submitNativeContextBody(t *testing.T, handler *ContextSourceHTTPHandler, body contextSourceBody) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/reference-import/context/sources", bytes.NewReader(payload))
	request.Header.Set("X-authentik-uid", "synthetic-actor")
	request.Header.Set("X-authentik-username", "synthetic-user")
	request.Header.Set("Idempotency-Key", "synthetic-key")
	response := httptest.NewRecorder()
	handler.start(response, request)
	return response
}

// TestNativeContextHTTPPreservesSourcePinsAndAttribution checks native metadata reaches the Go workflow unchanged.
// Inputs: synthetic SMS source with explicit device owner. Outputs: exact mapping and identity assertions.
// Effects: no network, source reads or database writes; choose to prevent lost provenance or implicit case requirements.
func TestNativeContextHTTPPreservesSourcePinsAndAttribution(t *testing.T) {
	store := &nativeContextStartRecorder{}
	handler := &ContextSourceHTTPHandler{workflows: store}
	body := contextSourceBody{ContractVersion: proffer.ContextContractVersion, SourceRef: "b2://synthetic/source.xml", ProviderVersionID: "synthetic-provider-version", SourceKind: "sms", DeclaredFormat: "smsbackuprestore_xml", OwnerPersonRef: "synthetic-owner", PerspectivePersonRef: "synthetic-owner", Perspective: "source-device", ByteLength: 987654321, ETag: "synthetic-etag"}
	if got := submitNativeContextBody(t, handler, body); got.Code != http.StatusAccepted {
		t.Fatalf("start status %d: %s", got.Code, got.Body.String())
	}
	in := store.inputs[0]
	if in.HumanSource == nil {
		t.Fatal("native source missing")
	}
	source := in.HumanSource
	if source.SourceRef != body.SourceRef || source.ProviderVersionID != body.ProviderVersionID || source.Format != body.DeclaredFormat || source.OwnerPersonRef != body.OwnerPersonRef || source.PerspectivePersonRef != body.PerspectivePersonRef || source.Perspective != body.Perspective || source.ByteLength != body.ByteLength || source.ETag != body.ETag || in.MatterID != "" || in.CourtCaseID != "" || in.ActorSubjectUID != "synthetic-actor" {
		t.Fatal("native coordinates or explicit attribution changed")
	}
	body.OwnerPersonRef = "different-synthetic-owner"
	if got := submitNativeContextBody(t, handler, body); got.Code != http.StatusAccepted {
		t.Fatalf("changed-owner status %d", got.Code)
	}
	if store.inputs[1].RequestID == in.RequestID {
		t.Fatal("different attribution reused the prior native workflow identity")
	}
}

// TestNativeContextHTTPPreservesExistingAIIdentity checks absent native metadata retains the established AI key.
// Inputs: ordinary context AI request. Outputs: original deterministic ID and nil native source.
// Effects: memory only; choose to catch accidental key changes on preexisting callers.
func TestNativeContextHTTPPreservesExistingAIIdentity(t *testing.T) {
	store := &nativeContextStartRecorder{}
	handler := &ContextSourceHTTPHandler{workflows: store}
	body := contextSourceBody{ContractVersion: proffer.ContextContractVersion, SourceRef: "b2://synthetic/ai.json", SourceKind: "ai_chat", DeclaredFormat: "chatgpt"}
	if got := submitNativeContextBody(t, handler, body); got.Code != http.StatusAccepted {
		t.Fatalf("start status %d", got.Code)
	}
	want := proffer.ContextWorkflowIDPrefix + flow.DeterministicID("context_source_ingest", "synthetic-key", "synthetic-actor", body.SourceRef, body.ProviderVersionID, body.PackageRef, body.SourceKind, body.DeclaredFormat, body.MatterID, body.CourtCaseID, fmt.Sprintf("%d:%d:%d:%d:%d", 0, 0, 0, 0, 0))
	if store.inputs[0].RequestID != want || store.inputs[0].HumanSource != nil {
		t.Fatal("existing AI identity or shape changed")
	}
}

// TestNativeContextHTTPRejectsInvalidLengthBeforeStart checks malformed metadata cannot schedule work.
// Inputs: negative native byte length. Outputs:422 and no captured start. Effects: memory only.
func TestNativeContextHTTPRejectsInvalidLengthBeforeStart(t *testing.T) {
	store := &nativeContextStartRecorder{}
	handler := &ContextSourceHTTPHandler{workflows: store}
	body := contextSourceBody{ContractVersion: proffer.ContextContractVersion, SourceRef: "b2://synthetic/source.xml", DeclaredFormat: "smsbackuprestore_xml", ByteLength: -1}
	if got := submitNativeContextBody(t, handler, body); got.Code != http.StatusUnprocessableEntity || len(store.inputs) != 0 {
		t.Fatal("invalid length scheduled work")
	}
}
