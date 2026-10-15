package postgres

import (
	"context"
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"strings"
	"testing"
)

// TestNativePostgresKindAdmissionRefusesExternalOnlyCandidates rejects unsupported native kinds before any store call.
// Inputs: known external-only kinds and an undated event. Outputs: refusal without DB access.
// Effects: none; choose to preserve the owner boundary of entities and dated timeline events in PostgreSQL.
func TestNativePostgresKindAdmissionRefusesExternalOnlyCandidates(t *testing.T) {
	for _, kind := range []string{"fact", "strategy", "history", "document", "work_product", "artifact", "event"} {
		candidate := service.AICandidate{AISourcePin: service.AISourcePin{SourceRef: "b2://fixture/source"}, Kind: kind}
		if _, err := (&EntityExtractionStore{}).StageAICandidates(context.Background(), "", "fixture", "", []service.AICandidate{candidate}); err == nil {
			t.Fatalf("native external-only kind admitted: %s", kind)
		}
	}
}

// TestHistoricalContextCoordinatesRejectIdentitySubstitution checks exact existing version/provider/hash lineage.
// Inputs: immutable synthetic registration and mismatches. Outputs: match or refusal. Effects: none.
func TestHistoricalContextCoordinatesRejectIdentitySubstitution(t *testing.T) {
	req := activities.ContextSourceRegistrationRequest{SourceRef: "b2://fixture/source.json", ProviderVersionID: "version-a", DeclaredFormat: "chatgpt", SourceSHA256: strings.Repeat("a", 64)}
	if err := historicalContextCoordinates(req, req.SourceRef+"?versionId=version-a", "", "chatgpt_official_json", req.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*activities.ContextSourceRegistrationRequest){func(r *activities.ContextSourceRegistrationRequest) { r.SourceRef += "-other" }, func(r *activities.ContextSourceRegistrationRequest) { r.ProviderVersionID = "version-b" }, func(r *activities.ContextSourceRegistrationRequest) { r.SourceSHA256 = strings.Repeat("b", 64) }, func(r *activities.ContextSourceRegistrationRequest) { r.DeclaredFormat = "claude" }} {
		bad := req
		change(&bad)
		if historicalContextCoordinates(bad, req.SourceRef+"?versionId=version-a", "", "chatgpt_official_json", req.SourceSHA256) == nil {
			t.Fatal("mismatched immutable lineage admitted")
		}
	}
	if historicalContextCoordinates(req, req.SourceRef+"?versionId=version-a&versionId=version-b", "", "chatgpt_official_json", req.SourceSHA256) == nil {
		t.Fatal("ambiguous version accepted")
	}
}

// TestNativeCandidatePersistenceNeverRetainsLiteralQuote verifies attrs and fact-column projection share the safe value.
// Inputs: externally pinned synthetic candidate. Outputs: typed fields, hashes and external refs without native text.
// Effects: none; choose before durable readback tests for every native candidate kind.
func TestNativeCandidatePersistenceNeverRetainsLiteralQuote(t *testing.T) {
	const quote = "private synthetic original passage"
	for _, kind := range []string{"entity", "event", "fact"} {
		candidate := service.AICandidate{AISourcePin: service.AISourcePin{SourceRef: "b2://fixture/source"}, Kind: kind, Statement: "Derived assertion", EvidenceQuote: quote, EvidenceBundleRef: "file:///unit/candidates.json", EvidenceBundleSHA256: strings.Repeat("a", 64), SourceSpan: service.AISourceSpan{SHA256: strings.Repeat("b", 64), Start: 1, End: 2}}
		projected, err := aiCandidateForPersistence(candidate)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"candidate": projected})
		if projected.EvidenceQuote != "" || strings.Contains(string(raw), quote) || (kind != "entity" && projected.Statement != candidate.Statement) || (kind == "entity" && projected.Statement != "") || projected.SourceSpan != candidate.SourceSpan || projected.EvidenceBundleRef == "" {
			t.Fatal("native quote leaked or external proof lost")
		}
	}
	if _, err := aiCandidateForPersistence(service.AICandidate{AISourcePin: service.AISourcePin{SourceRef: "b2://fixture/source"}, EvidenceQuote: quote}); err == nil {
		t.Fatal("missing external evidence accepted")
	}
	legacy, err := aiCandidateForPersistence(service.AICandidate{EvidenceQuote: quote})
	if err != nil || legacy.EvidenceQuote != quote {
		t.Fatal("legacy history projection changed")
	}
}
