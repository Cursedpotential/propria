package postgres

import (
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// TestPackageCandidateEditPreservesNativePins validates bounded assertion edits without permitting source rewrites.
// Inputs: synthetic source-pinned fact and account candidates. Outputs: edit and immutable-pin assertions.
// Effects: none. Pick before PostgreSQL integration to catch untyped or source-field edits.
func TestPackageCandidateEditPreservesNativePins(t *testing.T) {
	candidate := service.AICandidate{AISourcePin: service.AISourcePin{SourceVersionID: "11111111-1111-4111-8111-111111111111", SourceRef: "b2://synthetic/source.json", PreparedRef: "file:///synthetic/prepared.json", SourceSHA256: strings.Repeat("a", 64)},
		NativeJSONPointer: "/turn/0", SourceSpan: service.AISourceSpan{Start: 0, End: 5, SHA256: strings.Repeat("b", 64)}, SpanUnit: "unicode_codepoint",
		Kind: "fact", ReportedKind: "fact", ReviewDomain: "ai_chat_content", Predicate: "reports", Statement: "Proposed assertion", EvidenceQuote: "quote", Confidence: 0.5}
	edited, err := packageCandidateEdit(candidate, map[string]any{"statement": "Owner assertion", "confidence": 0.8})
	if err != nil || edited.Statement != "Owner assertion" || edited.AISourcePin != candidate.AISourcePin || edited.SourceSpan != candidate.SourceSpan || edited.NativeJSONPointer != candidate.NativeJSONPointer || edited.EvidenceQuote != candidate.EvidenceQuote {
		t.Fatalf("source drift: %+v %v", edited, err)
	}
	for _, fields := range []map[string]any{{"native_json_pointer": "/invented"}, {"source_ref": "b2://other/source.json"}, {"evidence_quote": "changed"}, {"confidence": 2.0}, {"statement": strings.Repeat("x", 2001)}} {
		if _, err := packageCandidateEdit(candidate, fields); err == nil {
			t.Fatalf("accepted invalid/source edit %+v", fields)
		}
	}
	candidate.ReportedKind, candidate.ReviewDomain = "strategy", "ai_chat_account"
	if edited, err := packageCandidateEdit(candidate, map[string]any{"statement": "Reviewed strategy"}); err != nil || edited.Kind != "fact" || edited.ReportedKind != "strategy" {
		t.Fatalf("staged strategy type lost: %+v %v", edited, err)
	}
}
