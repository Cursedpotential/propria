package surrealsink

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

type projectionRPCFixture struct {
	ledger        []map[string]any
	pins          [][]ApprovedSourcePin
	seen          []map[string]any
	missingSchema bool
}

// serve answers only the existing schema, checkpoint and source-pin query shapes.
// Inputs: scoped Surreal RPC request. Outputs: fixture metadata. Effects: records query variables.
// Pick for bounded repository behavior tests without a live analytical database.
func (f *projectionRPCFixture) serve(w http.ResponseWriter, r *http.Request) {
	var request rpcRequest
	if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.Params) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sql, _ := request.Params[0].(string)
	vars, _ := request.Params[1].(map[string]any)
	f.seen = append(f.seen, map[string]any{"sql": sql, "vars": vars})
	var values []any
	switch {
	case strings.HasPrefix(sql, "INFO FOR DB"):
		tables := map[string]string{}
		if !f.missingSchema {
			for _, name := range []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account", "ctx_approved_source_version", "ctx_approved_record", "ctx_approved_provenance", "ctx_approved_record_source", "ana_approved_projection"} {
				tables[name] = "DEFINE TABLE " + name + " TYPE NORMAL SCHEMAFULL"
			}
			tables["ctx_approved_provenance"] = "TYPE RELATION SCHEMAFULL"
			tables["ctx_approved_record_source"] = "TYPE RELATION SCHEMAFULL"
		}
		values = []any{map[string]any{"tables": tables}}
	case strings.HasPrefix(sql, "INFO FOR TABLE"):
		fields := map[string]string{}
		for name, typ := range graphFields("ctx_statement") {
			fields[name] = "DEFINE FIELD " + name + " TYPE " + typ
		}
		for name, typ := range approvedClaimFields {
			fields[name] = "DEFINE FIELD " + name + " TYPE " + typ
		}
		for _, name := range []string{"matter_id", "case_id", "source_id", "source_version_id", "source_object_id", "source_object_sha256", "source_object_uri", "record_id", "record_sha256", "approved_revision_id", "approval_digest", "candidate_id", "access_policy_id", "control_generation_id", "claims_hash", "node_set_hash"} {
			fields[name] = "DEFINE FIELD " + name + " TYPE string"
		}
		fields["in"], fields["out"] = "TYPE record", "TYPE record"
		fields["occurred_at"], fields["source_available_from"] = "TYPE option<datetime>", "TYPE option<datetime>"
		fields["claim_count"], fields["completed_at"] = "TYPE int", "TYPE datetime"
		for range 8 {
			values = append(values, map[string]any{"fields": fields})
		}
	case strings.Contains(sql, "FROM ana_approved_projection"):
		var selected []map[string]any
		for _, row := range f.ledger {
			if row["control_generation_id"].(string) > vars["after"].(string) {
				selected = append(selected, row)
			}
		}
		limit := int(vars["scan_limit"].(float64))
		if len(selected) > limit {
			selected = selected[:limit]
		}
		values = []any{selected}
	case strings.Contains(sql, "GROUP BY source_id"):
		for _, pins := range f.pins {
			values = append(values, pins)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	response := rpcResponse{}
	for _, value := range values {
		raw, _ := json.Marshal(value)
		response.Result = append(response.Result, statementResult{Status: "OK", Result: raw})
	}
	_ = json.NewEncoder(w).Encode(response)
}

// projectionFixture builds a real Client over fixture HTTP without bypassing schema checks.
// Inputs: fixture responses. Outputs: configured sink and close function. Effects: local test server.
// Pick for scoped listing and source-pin tests.
func projectionFixture(t *testing.T, fixture *projectionRPCFixture) (*ApprovedClaimsSink, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(fixture.serve))
	client, err := NewAnalysis(Config{URL: server.URL, Namespace: "fct", Database: "analysis", AuthLevel: "database", User: "test", Password: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return &ApprovedClaimsSink{Client: client, AccessPolicyID: "policy", CreatedByService: "service"}, server.Close
}

// projectionRow supplies a completed checkpoint with source-free control fields.
// Inputs: generation ID. Outputs: fixture ledger row. Effects: none.
// Pick for pagination tests; never as live projection evidence.
func projectionRow(generation string) map[string]any {
	return map[string]any{"matter_id": caseidentity.AuthoritativeMatterID, "case_id": caseidentity.AuthoritativeCourtCaseID,
		"access_policy_id": "policy", "approved_revision_id": "receipt", "approval_digest": strings.Repeat("a", 64),
		"control_generation_id": generation, "claims_hash": strings.Repeat("b", 64), "node_set_hash": strings.Repeat("c", 64),
		"claim_count": 2, "completed_at": time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)}
}

// TestListApprovedProjectionsEmptyAndPagedPins verifies no fake empty item and exact continuation.
// Inputs: empty then two completed fixture checkpoints. Outputs: [] then two one-item pages.
// Effects: local test RPC only; choose when scoped ledger listing changes.
func TestListApprovedProjectionsEmptyAndPagedPins(t *testing.T) {
	fixture := &projectionRPCFixture{pins: [][]ApprovedSourcePin{{}, {}, {}}}
	sink, closeServer := projectionFixture(t, fixture)
	defer closeServer()
	scope := ApprovedProjectionScope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, Limit: 1}
	empty, err := sink.ListApprovedProjections(context.Background(), scope)
	if err != nil || empty.Items == nil || len(empty.Items) != 0 || empty.HasMore || empty.NextCursor != "" {
		t.Fatalf("empty ledger was misreported: %+v err=%v", empty, err)
	}
	fixture.ledger = []map[string]any{projectionRow("generation-1"), projectionRow("generation-2")}
	fixture.pins = [][]ApprovedSourcePin{{{SourceID: "source", SourceVersionID: "version", SourceObjectID: "object", SourceObjectSHA256: strings.Repeat("d", 64), SourceObjectURI: "b2://salem-data/original"}}, {}, {}}
	first, err := sink.ListApprovedProjections(context.Background(), scope)
	if err != nil || len(first.Items) != 1 || !first.HasMore || first.NextCursor == "" || len(first.Items[0].SourcePins) != 1 {
		t.Fatalf("first completed projection page differs: %+v err=%v", first, err)
	}
	scope.Cursor = first.NextCursor
	second, err := sink.ListApprovedProjections(context.Background(), scope)
	if err != nil || len(second.Items) != 1 || second.Items[0].ProjectionGenerationID != "generation-2" || second.HasMore || second.NextCursor != "" {
		t.Fatalf("second completed projection page differs: %+v err=%v", second, err)
	}
	seenScopedPins := false
	for _, query := range fixture.seen {
		sql := query["sql"].(string)
		if strings.Contains(sql, "GROUP BY source_id") {
			vars := query["vars"].(map[string]any)
			seenScopedPins = strings.Contains(sql, "access_policy_id=$policy") && strings.Contains(sql, "approval_digest=$digest") && strings.Contains(sql, "control_generation_id=$generation") && vars["policy"] == "policy"
		}
	}
	if !seenScopedPins {
		t.Fatal("source pins were not prefiltered by case, policy and completed generation")
	}
}

// TestListApprovedProjectionsRejectsCrossScopeAndUnboundedPins rejects forged continuation and partial pin sets.
// Inputs: altered cursor and 101 distinct pin rows. Outputs: visible errors. Effects: local test RPC only.
// Pick when cursor or source pin bounds change.
func TestListApprovedProjectionsRejectsCrossScopeAndUnboundedPins(t *testing.T) {
	fixture := &projectionRPCFixture{ledger: []map[string]any{projectionRow("generation-1")}, pins: [][]ApprovedSourcePin{{}, {}, {}}}
	sink, closeServer := projectionFixture(t, fixture)
	defer closeServer()
	scope := ApprovedProjectionScope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, Limit: 1,
		Cursor: encodeProjectionCursor(ApprovedProjectionScope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID}, "other-policy", "generation-0")}
	if _, err := sink.ListApprovedProjections(context.Background(), scope); err == nil {
		t.Fatal("cross-policy cursor was accepted")
	}
	scope.Cursor = ""
	for i := range 101 {
		fixture.pins[0] = append(fixture.pins[0], ApprovedSourcePin{SourceID: "source", SourceVersionID: fmt.Sprintf("version-%03d", i), SourceObjectID: "object", SourceObjectSHA256: strings.Repeat("d", 64), SourceObjectURI: "b2://salem-data/original"})
	}
	if _, err := sink.ListApprovedProjections(context.Background(), scope); err == nil {
		t.Fatal("101 source pins were silently truncated")
	}
	fixture.pins[0] = []ApprovedSourcePin{{SourceID: "source", SourceVersionID: "version", SourceObjectID: "object", SourceObjectSHA256: "missing-hash", SourceObjectURI: "b2://salem-data/original"}}
	if _, err := sink.ListApprovedProjections(context.Background(), scope); err == nil {
		t.Fatal("incomplete retained-source pin was exposed")
	}
}

// TestListApprovedProjectionsDoesNotTreatMissingSchemaAsEmpty rejects an unadmitted graph.
// Inputs: absent typed table metadata. Outputs: visible error. Effects: local test RPC only.
// Pick when empty-store behavior changes; an admitted empty ledger must remain distinguishable.
func TestListApprovedProjectionsDoesNotTreatMissingSchemaAsEmpty(t *testing.T) {
	fixture := &projectionRPCFixture{missingSchema: true}
	sink, closeServer := projectionFixture(t, fixture)
	defer closeServer()
	_, err := sink.ListApprovedProjections(context.Background(), ApprovedProjectionScope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID})
	if err == nil {
		t.Fatal("missing analytical schema was reported as an empty completed ledger")
	}
}
