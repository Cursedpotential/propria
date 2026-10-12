package runtimeapi

import (
	"context"
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/proffer"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestReviewBundleContainmentAndLosslessFields verifies the reader cannot escape native roots.
// Inputs: synthetic derivative bundles. Output: source/unknown-field preservation and containment assertions.
// Effects: temporary test files only; choose before connecting actual workflow stage references.
func TestReviewBundleContainmentAndLosslessFields(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AI_CONTEXT_ROOT", root)
	t.Setenv("AI_CONTENT_ROOT", root)
	value := map[string]any{"stage": "prepared", "source": map[string]any{"source_version_id": "actual-version"}, "chunks": []any{map[string]any{"chunk_ref": "native:1", "content": "retained in bundle", "unrecognized": true}}}
	raw, _ := json.Marshal(value)
	path := filepath.Join(root, "prepared.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readContextReviewBundle("file://" + filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	rows := contextReviewArrays(got, "chunks")
	if len(rows) != 1 || rows[0].(map[string]any)["unrecognized"] != true {
		t.Fatalf("native fields lost: %#v", rows)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err = os.WriteFile(outside, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = readContextReviewBundle("file://" + filepath.ToSlash(outside)); err == nil {
		t.Fatal("outside derivative root admitted")
	}
	if _, err = readContextReviewBundle("file://other-host" + filepath.ToSlash(path)); err == nil {
		t.Fatal("remote file host admitted")
	}
}

// reviewWorkflowFixture supplies actual-shaped retained status without scheduling Temporal work.
// Inputs: synthetic summary. Output: in-memory workflow seam; effects: none.
type reviewWorkflowFixture struct {
	nativeContextStartRecorder
	progress proffer.ContextProgress
}

// Status returns fixture progress so package reader tests exercise the real HTTP body handler.
// Input: unused workflow ID. Output: retained summary; effects: none.
func (f *reviewWorkflowFixture) Status(context.Context, string) (proffer.ContextProgress, error) {
	return f.progress, nil
}

// TestPackageReaderPagesEveryNativeArray verifies replies and newly added native artifacts stay reachable.
// Inputs: sixty retained model-reply rows and actor-bound native status. Outputs: lossless fifty/ten paging.
// Effects: temporary files and in-memory HTTP only; no provider, database or graph operations.
func TestPackageReaderPagesEveryNativeArray(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AI_CONTEXT_ROOT", root)
	t.Setenv("AI_CONTENT_ROOT", root)
	rows := []any{}
	for i := 0; i < 60; i++ {
		rows = append(rows, map[string]any{"ordinal": i, "unknown_native_field": true})
	}
	path := filepath.Join(root, "candidates.json")
	ref := "file://" + filepath.ToSlash(path)
	bundle := map[string]any{"stage": "candidates", "source": map[string]any{"source_ref": "b2://synthetic/unit.json"}, "replies": rows, "candidates": []any{}}
	raw, _ := json.Marshal(bundle)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	handler := &ContextSourceHTTPHandler{workflows: &reviewWorkflowFixture{progress: proffer.ContextProgress{ContextSummary: proffer.ContextSummary{ActorSubjectUID: "synthetic-actor", SourceVersionRef: "actual-version", CandidatesRef: ref}}}}
	read := func(query string) map[string]any {
		request := httptest.NewRequest(http.MethodGet, "/package?"+query, nil)
		request.SetPathValue("workflow_id", "context-source:synthetic")
		request.Header.Set("X-authentik-uid", "synthetic-actor")
		request.Header.Set("X-authentik-username", "synthetic-owner")
		response := httptest.NewRecorder()
		handler.packageContent(response, request)
		if response.Code != 200 {
			t.Fatalf("reader %d %s", response.Code, response.Body.String())
		}
		value := map[string]any{}
		if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	first := read("section=native:replies")
	if len(first["items"].([]any)) != 50 || first["next_offset"] != float64(50) {
		t.Fatalf("first page %#v", first)
	}
	item := first["items"].([]any)[0].(map[string]any)
	if item["id"] != ref+"#native:replies/0" || item["fields"].(map[string]any)["unknown_native_field"] != true {
		t.Fatal("native identity or field lost")
	}
	last := read("section=native:replies&offset=50")
	if len(last["items"].([]any)) != 10 || last["next_offset"] != nil {
		t.Fatal("native continuation lost")
	}
	metadata := read("section=allmetadata")
	entry := metadata["items"].([]any)[0].(map[string]any)
	if entry["replies"].(map[string]any)["section"] != "native:replies" {
		t.Fatal("native array has no explicit metadata link")
	}
}

// TestReviewSectionsPreserveActualEdges verifies relationship rows are retained without synthesis.
// Inputs: staged candidates and actual edge. Output: source-kind and edge selection assertions; effects: none.
func TestReviewSectionsPreserveActualEdges(t *testing.T) {
	edge := map[string]any{"kind": "actual-new-kind", "from_node_id": "a", "to_node_id": "b", "sourcebasis": "native:3"}
	bundle := map[string]any{"candidates": []any{map[string]any{"kind": "entity"}, map[string]any{"kind": "strategy"}}, "edges": []any{edge}}
	if rows := contextReviewArrays(bundle, "entities"); len(rows) != 1 {
		t.Fatalf("entity count %d", len(rows))
	}
	if rows := contextReviewArrays(bundle, "strategy"); len(rows) != 1 {
		t.Fatalf("strategy count %d", len(rows))
	}
	if rows := contextReviewArrays(bundle, "edges"); len(rows) != 1 || rows[0].(map[string]any)["kind"] != "actual-new-kind" {
		t.Fatal("actual edge changed")
	}
	if rows := contextReviewArrays(map[string]any{}, "edges"); rows != nil {
		t.Fatal("invented absent edge collection")
	}
}
