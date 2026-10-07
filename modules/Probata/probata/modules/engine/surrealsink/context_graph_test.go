package surrealsink

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// graphFormatPacket supplies tiny format values, never a corpus or delivery proof.
// Inputs none; output is an in-memory boundary fixture. No external effects occur.
func graphFormatPacket() ContextGraphBundle {
	pin := ContextSourcePin{SourceID: "format-source", SourceVersionID: "format-version", SourceHash: strings.Repeat("a", 64), Locator: "format:turn:1"}
	pins := []ContextSourcePin{pin}
	return ContextGraphBundle{Scope: ContextGraphScope{MatterID: "format-matter", CaseID: "format-case", AccessPolicyID: "format-policy", CreatedByService: "format-service"}, GenerationID: "format-generation", ExtractionRunRef: "file:///" + strings.Repeat("long-existing-ref/", 20), Nodes: []ContextGraphNode{{NodeID: "turn", Kind: "ctx_content_unit", DerivativeKind: "source_turn", SourceOrigin: "assistant", SourcePins: pins, Body: "format only"}, {NodeID: "work", Kind: "ctx_content_unit", DerivativeKind: "created_work", SourceOrigin: "assistant", SourcePins: pins, Body: "complete format value", InputNodeIDs: []string{"turn"}}}, Edges: []ContextGraphEdge{{EdgeID: "dependency", Kind: "depends_on", FromNodeID: "work", ToNodeID: "turn", SourcePins: pins}}}
}

// TestGraphContractBoundsDependenciesAndNativeOrder checks only packet invariants.
// Inputs are small format values; outputs assertions. No database/corpus effects occur.
func TestGraphContractBoundsDependenciesAndNativeOrder(t *testing.T) {
	b := graphFormatPacket()
	if e := b.Validate(); e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(b)
	if _, e := DecodeContextGraphBundle(strings.NewReader(string(raw) + " {}")); e == nil {
		t.Fatal("trailing JSON accepted")
	}
	bad := strings.Replace(string(raw), `"scope":{`, `"scope":{"unknown":1,`, 1)
	if _, e := DecodeContextGraphBundle(strings.NewReader(bad)); e == nil {
		t.Fatal("unknown field accepted")
	}
	b.Edges[0].Kind = "derived_from"
	if b.Validate() == nil {
		t.Fatal("missing complete dependency accepted")
	}
	b = graphFormatPacket()
	b.Nodes[1].Body = ""
	if b.Validate() == nil {
		t.Fatal("missing full work accepted")
	}
	b = graphFormatPacket()
	b.Nodes[1].DerivativeKind = "source_turn"
	b.Nodes[1].InputNodeIDs = nil
	b.Edges[0].Kind = "before"
	b.Edges[0].FromNodeID = "turn"
	b.Edges[0].ToNodeID = "work"
	first := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	second := first.Add(time.Second)
	b.Nodes[0].OccurredAt = &first
	b.Nodes[1].OccurredAt = &second
	if e := b.Validate(); e != nil {
		t.Fatal(e)
	}
	b.Nodes[1].OccurredAt = &first
	if b.Validate() == nil {
		t.Fatal("equal native time accepted")
	}
	b.Nodes[1].OccurredAt = nil
	if b.Validate() == nil {
		t.Fatal("missing native time accepted")
	}
}

// TestMinimalGraphSchemaContainsOnlyUsedFamilies checks generated typed DDL boundaries.
// Input is a format packet; output is assertions, with no DDL execution or live proof.
func TestMinimalGraphSchemaContainsOnlyUsedFamilies(t *testing.T) {
	schema, e := MinimalContextGraphSchema(graphFormatPacket())
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{"ana_projection_generation", "ctx_content_unit", "depends_on TYPE RELATION", "payload_json", "source_pins.*.source_version_id"} {
		if !strings.Contains(schema, s) {
			t.Fatalf("missing %s", s)
		}
	}
	for _, s := range []string{"FLEXIBLE", "OVERWRITE", "DEFINE DATABASE", "DEFINE USER", "ctx_event_account", "completed_artifact", "DELETE"} {
		if strings.Contains(schema, s) {
			t.Fatalf("unexpected %s", s)
		}
	}
}

// TestAnalysisTargetAdmissionAndErrorSanitation checks identity and wire failure boundaries.
// Inputs are explicit configs and a local transport; outputs assertions. Effects are
// local HTTP only, with no graph corpus, SQL engine or live-database proof.
func TestAnalysisTargetAdmissionAndErrorSanitation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("surreal-db") != "analysis" || r.Header.Get("surreal-auth-db") != "analysis" {
			t.Error("analysis database identity missing")
		}
		w.WriteHeader(500)
		_, _ = w.Write([]byte("private-case-body must never escape"))
	}))
	defer server.Close()
	cfg := Config{URL: server.URL, Namespace: "fct", Database: "analysis", AuthLevel: "database", User: "format-user", Password: "format-password"}
	client, e := NewAnalysis(cfg)
	if e != nil {
		t.Fatal(e)
	}
	_, e = client.ProjectContextGraph(context.Background(), graphFormatPacket())
	if e == nil || strings.Contains(e.Error(), "private-case-body") {
		t.Fatal("raw database failure escaped")
	}
	cfg.Database = "case"
	if _, e := NewAnalysis(cfg); e == nil {
		t.Fatal("case fallback accepted")
	}
	cfg.Database = "analysis"
	cfg.AuthLevel = "root"
	if _, e := NewAnalysis(cfg); e == nil {
		t.Fatal("root identity accepted")
	}
}

// TestGraphMetadataFailsBeforeWrites proves missing admitted schema cannot auto-create tables.
// Inputs are a local metadata response and small packet; output is assertions.
// Effects are local HTTP only; this does not claim live SurrealQL validation.
func TestGraphMetadataFailsBeforeWrites(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request.Params[0] != "INFO FOR DB;" {
			t.Error("write attempted without schema")
		}
		_, _ = w.Write([]byte(`{"result":[{"status":"OK","result":{"tables":{}}}]}`))
	}))
	defer server.Close()
	client, e := NewAnalysis(Config{URL: server.URL, Namespace: "fct", Database: "analysis", AuthLevel: "database", User: "format-user", Password: "format-password"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = client.ProjectContextGraph(context.Background(), graphFormatPacket()); e == nil {
		t.Fatal("missing metadata admitted")
	}
	if calls != 1 {
		t.Fatal("unexpected write")
	}
}
