package weaviate

import (
	"context"
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/humancontext"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHumanTargetUsesBatchUpsertAndIndependentGet verifies the existing idempotent transport and per-object rejection.
// Inputs: synthetic server/object only. Outputs: matching GET or error; effects: loopback HTTP, no real search writes.
func TestHumanTargetUsesBatchUpsertAndIndependentGet(t *testing.T) {
	const id = "018f0000-0000-7000-8000-000000000001"
	fail := false
	writes, reads := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/batch/objects":
			if r.Method != http.MethodPost {
				t.Error("not batch POST")
			}
			writes++
			var body struct {
				Objects []map[string]any `json:"objects"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body.Objects) != 1 || body.Objects[0]["id"] != id {
				t.Errorf("batch %+v", body)
			}
			status := "SUCCESS"
			if fail {
				status = "FAILED"
			}
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"id": id, "result": map[string]any{"status": status}}})
		case "/v1/objects/ProfferChunks20261002/" + id:
			reads++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "class": "ProfferChunks20261002", "properties": map[string]any{"text": "synthetic", "record_kind": "conversation_chunk"}, "vectors": map[string]any{"text_nim": []float32{.25, .5}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	target := &HumanSourceTarget{Store: Store{BaseURL: server.URL, Collection: "ProfferChunks20261002", VectorName: "text_nim"}}
	object := humancontext.SearchObject{ID: id, Properties: map[string]any{"text": "synthetic", "record_kind": "conversation_chunk"}, Vector: []float32{.25, .5}}
	for n := 0; n < 2; n++ {
		if err := target.Put(context.Background(), object); err != nil {
			t.Fatal(err)
		}
	}
	if writes != 2 || reads != 0 {
		t.Fatalf("publication performed readback %d %d", writes, reads)
	}
	actual, err := target.Get(context.Background(), id)
	if err != nil || actual.ID != id || len(actual.Vector) != 2 || reads != 1 {
		t.Fatalf("readback %+v %v", actual, err)
	}
	fail = true
	if err := target.Put(context.Background(), object); err == nil {
		t.Fatal("per-object failure accepted")
	}
}

// TestHumanTargetRejectsWrongDestination verifies both operations reject message collections before transport.
// Inputs: synthetic targets. Outputs: explicit errors; effects: no HTTP requests.
func TestHumanTargetRejectsWrongDestination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	for _, store := range []Store{{BaseURL: server.URL, Collection: "MsgEvents20260918", VectorName: "text_nim"}, {BaseURL: server.URL, Collection: "ProfferChunks20261002", VectorName: "other"}} {
		target := &HumanSourceTarget{Store: store}
		if err := target.Put(context.Background(), humancontext.SearchObject{ID: "id", Vector: []float32{1}}); err == nil {
			t.Fatal("wrong destination accepted")
		}
		if _, err := target.Get(context.Background(), "id"); err == nil {
			t.Fatal("wrong read destination accepted")
		}
	}
	if calls != 0 {
		t.Fatalf("unexpected HTTP calls %d", calls)
	}
}

// TestHumanTargetRejectsOversizeResponses verifies native writes and independent reads never buffer unbounded bodies.
// Inputs: synthetic response above four MiB. Outputs: budget error; effects: loopback HTTP only.
func TestHumanTargetRejectsOversizeResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 4*1024*1024+1)))
	}))
	defer server.Close()
	target := &HumanSourceTarget{Store: Store{BaseURL: server.URL, Collection: "ProfferChunks20261002", VectorName: "text_nim"}}
	if err := target.Put(context.Background(), humancontext.SearchObject{ID: "id", Vector: []float32{1}}); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("write budget: %v", err)
	}
	if _, err := target.Get(context.Background(), "id"); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("read budget: %v", err)
	}
}
