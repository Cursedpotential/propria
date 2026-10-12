package runtimeapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// retryWorkflowFixture records the exact server-bound retry input without starting a real workflow.
// Inputs: actual-shaped retained parent summary. Output: recorded input and synthetic execution refs.
// Effects: memory only. Pick for HTTP source/actor/predecessor validation tests.
type retryWorkflowFixture struct {
	reviewWorkflowFixture
	starts int
	input  proffer.ContextExtractionRetryInput
}

// StartExtractionRetry records an authenticated attempt request for assertions.
// Inputs: deterministic attempt ID and server pins. Output: actual-shaped execution IDs.
// Effects: test counter only; choose for retry HTTP tests.
func (f *retryWorkflowFixture) StartExtractionRetry(_ context.Context, id string, in proffer.ContextExtractionRetryInput) (proffer.ContextStarted, error) {
	f.starts++
	f.input = in
	return proffer.ContextStarted{WorkflowID: id, RunID: "synthetic-attempt-run"}, nil
}

// TestNativeRetryHTTPRejectsDriftAndPreservesParent exercises the real retry handler with immutable predecessor files.
// Inputs: valid source bundle and wrong actor/version/model variants. Output: exact start input and refused starts.
// Effects: synthetic temporary files and memory only; run on the existing VPS test environment.
func TestNativeRetryHTTPRejectsDriftAndPreservesParent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AI_CONTEXT_ROOT", root)
	ref := "b2://synthetic/native.json"
	preparedRef := "file://" + filepath.ToSlash(filepath.Join(root, "prepared.json"))
	productsRef := "file://" + filepath.ToSlash(filepath.Join(root, "work_products.json"))
	source := map[string]any{"source_ref": ref, "source_format": "chatgpt"}
	for name, value := range map[string]map[string]any{"prepared.json": {"stage": "prepared", "source": source}, "work_products.json": {"stage": "work_products", "source": source, "prepared_ref": preparedRef}} {
		raw, _ := json.Marshal(value)
		if err := os.WriteFile(filepath.Join(root, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fixture := &retryWorkflowFixture{reviewWorkflowFixture: reviewWorkflowFixture{progress: proffer.ContextProgress{ContextSummary: proffer.ContextSummary{
		ActorSubjectUID: "owner", SourceVersionRef: "registered-version", PreparedRef: preparedRef, WorkProductsRef: productsRef,
		ReviewSource: &proffer.ContextVerifiedReviewSource{ContextReviewSource: proffer.ContextReviewSource{SourceVersionID: "registered-version", SourceRef: ref}, PreparedRef: preparedRef}}}}}
	h := &ContextSourceHTTPHandler{workflows: fixture}
	for _, test := range []struct {
		actor, version, model string
		code                  int
	}{{"other", "registered-version", "configured", 404}, {"owner", "stale-version", "configured", 409}, {"owner", "registered-version", "unconfigured-model", 422}, {"owner", "registered-version", "configured", 202}} {
		raw, _ := json.Marshal(map[string]any{"source_version_ref": test.version, "model_id": test.model})
		request := httptest.NewRequest("POST", "/retry", strings.NewReader(string(raw)))
		request.SetPathValue("workflow_id", "context-source:parent")
		request.Header.Set("X-authentik-uid", test.actor)
		request.Header.Set("X-authentik-username", "owner")
		request.Header.Set("Idempotency-Key", "click")
		response := httptest.NewRecorder()
		h.retryExtraction(response, request)
		if response.Code != test.code {
			t.Fatalf("%+v got %d %s", test, response.Code, response.Body.String())
		}
	}
	if fixture.starts != 1 || fixture.input.Source.SourceVersionID != "registered-version" || fixture.input.PreparedRef != preparedRef || fixture.input.ParentWorkflowID != "context-source:parent" {
		t.Fatalf("retry pins %+v starts %d", fixture.input, fixture.starts)
	}
}
