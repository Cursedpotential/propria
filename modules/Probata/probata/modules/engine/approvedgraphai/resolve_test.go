package approvedgraphai

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

type staticAIReader struct {
	row Snapshot
	err error
}

// ReadApprovedAI supplies one persisted fixture without performing database writes.
// Inputs: exact scope. Outputs: fixture snapshot. Effects: none.
// Pick for resolver tests; production uses postgres.ApprovedAIGraphReader.
func (r staticAIReader) ReadApprovedAI(context.Context, Scope) (Snapshot, error) { return r.row, r.err }

// TestResolveApprovedNativeFact proves an actual digest-bound decision yields the typed fact and native citation.
// Inputs: persisted candidate, owner receipt, ledger and retained source pin. Outputs: one source-cited claim.
// Effects: none. Pick for the native bridge, including legacy transformed history accounts.
func TestResolveApprovedNativeFact(t *testing.T) {
	scope, row := approvedFixture(t)
	claim, err := Resolve(context.Background(), staticAIReader{row: row}, scope)
	if err != nil {
		t.Fatal(err)
	}
	if claim.Kind != "statement" || claim.Text != "A documented account" || claim.Predicate != "history_account" || claim.NativeSpanStart == nil || *claim.NativeSpanStart != 0 || claim.RecordID == "" || claim.RecordSHA256 == "" || claim.SourceAvailableFrom != nil || claim.ReceiptID != scope.DecisionID {
		t.Fatalf("typed native citation lost: %+v", claim)
	}
	if id, err := WorkflowID(scope); err != nil || id == "" {
		t.Fatalf("workflow ID: %q %v", id, err)
	}
}

// TestResolveApprovedContextWithoutRetainedObject binds an approved claim to an actual native source URI.
// Inputs: digest-bound approved fixture with a registered context source and empty object ID.
// Outputs: cited claim and a rejection for changed URI. Effects: none.
// Pick for context registrations that have no retained-object or preview row.
func TestResolveApprovedContextWithoutRetainedObject(t *testing.T) {
	scope, row := approvedFixture(t)
	var attrs map[string]any
	if err := json.Unmarshal(row.Attrs, &attrs); err != nil {
		t.Fatal(err)
	}
	var candidate service.AICandidate
	raw, _ := json.Marshal(attrs["candidate"])
	if err := json.Unmarshal(raw, &candidate); err != nil {
		t.Fatal(err)
	}
	candidate.SourceObjectID = ""
	candidate.SourceRef = "b2://original/native.md"
	candidate.PreparedRef = "file:///data/proffer/derive-scratch/ai-content/context/scope/prepared.json"
	attrs["candidate"] = candidate
	digest := service.AICandidateDigest(candidate)
	row.CandidateSHA256 = hex.EncodeToString(digest[:])
	attrs["review_decision"].(map[string]any)["approved_content_sha256"] = row.CandidateSHA256
	row.Attrs, _ = json.Marshal(attrs)
	var rationale map[string]any
	if err := json.Unmarshal(row.LedgerRationale, &rationale); err != nil {
		t.Fatal(err)
	}
	rationale["candidate_content_sha256"] = row.CandidateSHA256
	row.LedgerRationale, _ = json.Marshal(rationale)
	row.SourceObjectID = ""
	row.SourceObjectURI = candidate.SourceRef
	claim, err := Resolve(context.Background(), staticAIReader{row: row}, scope)
	if err != nil {
		t.Fatal(err)
	}
	if claim.SourceObjectID != "" || claim.SourceObjectURI != candidate.SourceRef || claim.NativeSpanStart == nil || *claim.NativeSpanStart != 0 {
		t.Fatalf("native source identity lost: %+v", claim)
	}
	row.SourceObjectURI = "b2://other/original.md"
	if _, err := Resolve(context.Background(), staticAIReader{row: row}, scope); err == nil {
		t.Fatal("cross-source URI admitted")
	}
}

// TestResolveApprovedNativeRejectsDrift rejects held decisions, altered source digests and ledger mismatches.
// Inputs: same persisted fixture with one field changed. Outputs: fail-closed error.
// Effects: none. Pick to guard against fabricated approval and cross-source projection.
func TestResolveApprovedNativeRejectsDrift(t *testing.T) {
	scope, row := approvedFixture(t)
	for name, mutate := range map[string]func(*Snapshot){
		"held":    func(r *Snapshot) { r.ReviewState = "pending" },
		"ledger":  func(r *Snapshot) { r.LedgerDecision = "rejected" },
		"source":  func(r *Snapshot) { r.SourceSHA256 = strings.Repeat("f", 64) },
		"content": func(r *Snapshot) { r.CandidateSHA256 = strings.Repeat("f", 64) },
		"target":  func(r *Snapshot) { r.LedgerTargetID = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := row
			mutate(&bad)
			if _, err := Resolve(context.Background(), staticAIReader{row: bad}, scope); err == nil {
				t.Fatal("mismatched approval admitted")
			}
		})
	}
}

// TestUTCClockPreservesNativeInstant ensures analytical hashing is stable across offset readbacks.
// Inputs: a verified first-party clock with a non-UTC offset. Outputs: UTC same instant.
// Effects: none. Pick for native AI graph claim materialization after candidate digest verification.
func TestUTCClockPreservesNativeInstant(t *testing.T) {
	local := time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.FixedZone("offset", -5*3600))
	got := utcClock(&local)
	if got == nil || got.Location() != time.UTC || !got.Equal(local) || utcClock(nil) != nil {
		t.Fatal("native source clock changed instant or lost nil")
	}
}

// approvedFixture constructs a persisted transformed fact with the actual receipt and rationale vocabulary.
// Inputs: test handle. Outputs: exact scope and snapshot. Effects: none.
// Pick for native resolver tests, not graph or source-store fixtures.
func approvedFixture(t *testing.T) (Scope, Snapshot) {
	t.Helper()
	scope := Scope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, SourceVersionID: "00000000-0000-0000-0000-000000000003", CandidateID: "00000000-0000-0000-0000-000000000004", RequestDigest: strings.Repeat("a", 64)}
	scope.DecisionID = flow.DeterministicID("ai_review_decision", scope.CandidateID, scope.RequestDigest)
	at := time.Date(2026, 10, 8, 12, 0, 0, 123456789, time.UTC)
	candidate := service.AICandidate{AISourcePin: service.AISourcePin{SourceVersionID: scope.SourceVersionID, SourceObjectID: "00000000-0000-0000-0000-000000000005", SourceSHA256: strings.Repeat("b", 64)}, SourceSpan: service.AISourceSpan{Start: 0, End: 4, SHA256: strings.Repeat("c", 64)}, SpanUnit: "unicode_codepoint", Kind: "fact", ReportedKind: "history", ReviewDomain: "ai_chat_account", Predicate: "history_account", Statement: "A documented account", EvidenceQuote: "text"}
	digest := service.AICandidateDigest(candidate)
	hash := hex.EncodeToString(digest[:])
	attrs, _ := json.Marshal(map[string]any{"candidate": candidate, "reported_kind": "history", "review_domain": "ai_chat_account", "graph_promotion": "approved_context", "projection_scope": "context", "review_decision": map[string]any{"decision_id": scope.DecisionID, "actor": map[string]any{"subject_uid": "owner", "username": "owner@example"}, "request_digest": scope.RequestDigest, "approved_content_sha256": hash, "at": at, "decision": "approved"}})
	rationale, _ := json.Marshal(map[string]any{"source_version_id": scope.SourceVersionID, "candidate_content_sha256": hash, "actor_subject_uid": "owner", "request_digest": scope.RequestDigest, "candidate_kind": "fact", "reported_kind": "history", "review_domain": "ai_chat_account", "graph_promotion": "approved_context", "projection_scope": "context"})
	return scope, Snapshot{Table: "working.candidate_fact", ReviewState: "approved", SourceRawTable: "context.source_version", SourceRawID: scope.SourceVersionID, Attrs: attrs, CandidateSHA256: hash, LedgerDecision: "approved", LedgerTargetKind: "working.candidate_fact", LedgerTargetID: scope.CandidateID, LedgerReviewer: "owner@example", LedgerRationale: rationale, DecidedAt: at.Truncate(time.Microsecond), SourceID: "00000000-0000-0000-0000-000000000006", SourceObjectID: candidate.SourceObjectID, SourceObjectURI: "file:///retained/original.md", SourceSHA256: candidate.SourceSHA256}
}
