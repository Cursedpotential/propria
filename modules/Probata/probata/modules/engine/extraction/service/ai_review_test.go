package service

import (
	"math"
	"strings"
	"testing"
)

// TestAICandidateAdmission rejects nonfinite confidence, unknown span units and undated ungrounded accounts.
// Inputs: a synthetic retained-source fact with bounded quote. Outputs: exact admission or error.
// Effects: none. Choose as the public validation boundary regression.
func TestAICandidateAdmission(t *testing.T) {
	candidate := AICandidate{
		AISourcePin:       AISourcePin{SourceVersionID: "11111111-1111-4111-8111-111111111111", SourceObjectID: "22222222-2222-4222-8222-222222222222", SourceSHA256: strings.Repeat("a", 64)},
		NativeJSONPointer: "/messages/0/text", SourceSpan: AISourceSpan{Start: 0, End: 4, SHA256: strings.Repeat("b", 64)},
		SpanUnit: "unicode_codepoint", Kind: "fact", ReportedKind: "fact", ReviewDomain: "ai_chat_account", Predicate: "account", Statement: "A bounded account", EvidenceQuote: "text", Confidence: 0.8,
	}
	if err := ValidateAICandidate(candidate); err != nil {
		t.Fatalf("valid candidate: %v", err)
	}
	candidate.Confidence = math.NaN()
	if err := ValidateAICandidate(candidate); err == nil {
		t.Fatal("NaN confidence admitted")
	}
	candidate.Confidence = math.Inf(1)
	if err := ValidateAICandidate(candidate); err == nil {
		t.Fatal("infinite confidence admitted")
	}
	candidate.Confidence = 0.8
	candidate.SpanUnit = "byte"
	if err := ValidateAICandidate(candidate); err == nil {
		t.Fatal("wrong span unit admitted")
	}
	candidate.SpanUnit = "unicode_codepoint"
	candidate.NativeJSONPointer = ""
	if err := ValidateAICandidate(candidate); err != nil {
		t.Fatalf("native text whole-source locator was rejected: %v", err)
	}
	candidate.NativeJSONPointer = "/messages/0/text"
	candidate.Kind, candidate.ReportedKind = "document", "document"
	if err := ValidateAICandidate(candidate); err == nil {
		t.Fatal("document was silently staged as a fact")
	}
}
