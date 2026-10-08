package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// TestAISourceSpanProof checks exact native JSON pointer, Unicode codepoint span, hash and quote agreement.
// Inputs: synthetic original JSON. Outputs: success only for its exact compass-rune excerpt.
// Effects: none. Choose as the retained-source verifier regression.
func TestAISourceSpanProof(t *testing.T) {
	const excerpt = "🧭"
	digest := sha256.Sum256([]byte(excerpt))
	candidate := service.AICandidate{
		NativeJSONPointer: "/turns/0/text", SpanUnit: "unicode_codepoint",
		SourceSpan:    service.AISourceSpan{Start: 1, End: 2, SHA256: hex.EncodeToString(digest[:])},
		EvidenceQuote: excerpt,
	}
	document := map[string]any{"turns": []any{map[string]any{"text": "A🧭B"}}}
	if err := verifyAISpans(document, []service.AICandidate{candidate}); err != nil {
		t.Fatalf("valid Unicode span: %v", err)
	}
	bad := candidate
	bad.NativeJSONPointer = "/turns/1/text"
	if err := verifyAISpans(document, []service.AICandidate{bad}); err == nil {
		t.Fatal("missing pointer admitted")
	}
	bad = candidate
	bad.SourceSpan.End = 4
	if err := verifyAISpans(document, []service.AICandidate{bad}); err == nil {
		t.Fatal("out-of-bounds span admitted")
	}
	bad = candidate
	bad.SourceSpan.Start = -1
	if err := verifyAISpans(document, []service.AICandidate{bad}); err == nil {
		t.Fatal("negative span admitted")
	}
	bad = candidate
	bad.EvidenceQuote = "B"
	if err := verifyAISpans(document, []service.AICandidate{bad}); err == nil {
		t.Fatal("unsupported quote admitted")
	}
	bad = candidate
	bad.SourceSpan.SHA256 = strings.Repeat("0", 64)
	if err := verifyAISpans(document, []service.AICandidate{bad}); err == nil {
		t.Fatal("wrong span hash admitted")
	}
}

// TestAINativeMarkdownSpanProof verifies UTF-8 whole-text codepoint locators without a JSON wrapper.
// Inputs: one exact native string and its quoted span. Outputs: agreement or fail-closed error.
// Effects: none. Choose for the declared Markdown verifier branch.
func TestAINativeMarkdownSpanProof(t *testing.T) {
	const native = "A🧭B"
	sum := sha256.Sum256([]byte("🧭"))
	candidate := service.AICandidate{SpanUnit: "unicode_codepoint", SourceSpan: service.AISourceSpan{Start: 1, End: 2, SHA256: hex.EncodeToString(sum[:])}, EvidenceQuote: "🧭"}
	if err := verifyAISpans(native, []service.AICandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	candidate.NativeJSONPointer = "/text"
	if err := verifyAISpans(native, []service.AICandidate{candidate}); err == nil {
		t.Fatal("native text accepted a fabricated JSON pointer")
	}
	candidate.NativeJSONPointer = ""
	if err := verifyAISpans(map[string]any{"text": native}, []service.AICandidate{candidate}); err == nil {
		t.Fatal("JSON source accepted an unlocated whole-text span")
	}
}

// TestAINativeClockRequiresExactMessage checks source availability only against a cited native JSON message.
// Inputs: ChatGPT and Claude native fields with verified clocks. Outputs: exact match or rejection.
// Effects: none. Pick for first-party knowledge clocks, never candidate event time.
func TestAINativeClockRequiresExactMessage(t *testing.T) {
	chatClock := time.Unix(1735689600, 125000000).UTC()
	chat := map[string]any{"mapping": map[string]any{"node": map[string]any{"message": map[string]any{"create_time": float64(1735689600.125), "content": map[string]any{"parts": []any{"source"}}}}}}
	candidate := service.AICandidate{NativeJSONPointer: "/mapping/node/message/content/parts/0", SourceAvailableFrom: &chatClock}
	if err := verifyAINativeClock(chat, "chatgpt_official_json", candidate); err != nil {
		t.Fatal(err)
	}
	wrong := chatClock.Add(time.Second)
	candidate.SourceAvailableFrom = &wrong
	if err := verifyAINativeClock(chat, "chatgpt_official_json", candidate); err == nil {
		t.Fatal("different message clock admitted")
	}
	claudeClock := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	claude := map[string]any{"chat_messages": []any{map[string]any{"created_at": claudeClock.Format(time.RFC3339Nano), "text": "source"}}}
	candidate.NativeJSONPointer, candidate.SourceAvailableFrom = "/chat_messages/0/text", &claudeClock
	if err := verifyAINativeClock(claude, "claude_ai_export_json", candidate); err != nil {
		t.Fatal(err)
	}
	candidate.NativeJSONPointer = "/other/text"
	if err := verifyAINativeClock(claude, "claude_ai_export_json", candidate); err == nil {
		t.Fatal("unlocated clock admitted")
	}
}

// TestAIReplayAndUndatedAccount checks exact digest/source replays and event-time preservation.
// Inputs: synthetic digests and an undated event account. Outputs: only exact replay and fact conversion.
// Effects: none. Choose before exercising the working-table transaction on a disposable schema.
func TestAIReplayAndUndatedAccount(t *testing.T) {
	digest := strings.Repeat("a", 64)
	if !sameAIRunDigest(digest, digest) || sameAIRunDigest(digest, strings.Repeat("b", 64)) {
		t.Fatal("run digest replay is not exact")
	}
	bytes := sha256.Sum256([]byte("candidate"))
	if !sameAIExistingCandidate("context.source_version", "version-1", bytes[:], "version-1", bytes[:]) {
		t.Fatal("identical candidate rejected")
	}
	if sameAIExistingCandidate("context.normalized_generation", "version-1", bytes[:], "version-1", bytes[:]) ||
		sameAIExistingCandidate("context.source_version", "version-2", bytes[:], "version-1", bytes[:]) ||
		sameAIExistingCandidate("context.source_version", "version-1", bytes[:], "version-1", make([]byte, 32)) {
		t.Fatal("mismatched candidate replay admitted")
	}
	staged := aiCandidateForStaging(service.AICandidate{Kind: "event", EventType: "communication", Statement: "Time unknown"})
	if staged.Kind != "fact" || staged.Predicate != "undated_event_account" || staged.OccurredAt != nil {
		t.Fatal("undated event gained a fabricated time")
	}
	strategy := aiCandidateForStaging(service.AICandidate{Kind: "strategy", Statement: "A documented plan"})
	if strategy.Kind != "fact" || strategy.Predicate != "strategy_account" {
		t.Fatal("strategy did not retain its reviewable account kind")
	}
	if aiLedgerDecision("needs_info") != "needs_context" || aiLedgerDecision("approved") != "approved" {
		t.Fatal("decision vocabulary drift")
	}
}

// TestAIOriginalAvailability rejects missing local opener and unsupported remote original storage.
// Inputs: absent opener and remote class. Outputs: fail-closed errors.
// Effects: none. Choose before wiring the approved opener into the starter.
func TestAIOriginalAvailability(t *testing.T) {
	version := "unverified-provider-version"
	if _, err := (&EntityExtractionStore{}).VerifyAISource(context.Background(), "preview", service.AISourcePin{VersionID: &version}); err == nil {
		t.Fatal("unverified provider version admitted")
	}
	if err := aiOpenerAvailable(nil); err == nil {
		t.Fatal("missing opener admitted")
	}
	if err := aiStorageAvailable("immutable_object_store"); err == nil {
		t.Fatal("remote original admitted without version-aware opener")
	}
	if err := aiStorageAvailable("filesystem"); err != nil {
		t.Fatal(err)
	}
	if err := aiStorageAvailable("inline"); err != nil {
		t.Fatal(err)
	}
	if _, err := aiStringAtPointer(map[string]any{"a/b": "x"}, "/a~1b"); err != nil {
		t.Fatal(err)
	}
	if _, err := aiStringAtPointer(map[string]any{"a": "x"}, "/a~2"); err == nil {
		t.Fatal("invalid pointer escape admitted")
	}
}

// TestAIGraphPromotion requires the existing owner decision for attributed analysis context.
// Inputs: typed AI proposals, mapped accounts and review outcomes. Outputs: exact eligibility labels.
// Effects: none. Choose to prevent a blanket hold or accidental promotion of an unreviewed claim.
func TestAIGraphPromotion(t *testing.T) {
	for _, kind := range []string{"entity", "event", "fact"} {
		candidate := service.AICandidate{Kind: kind, ReviewDomain: "ai_chat_content"}
		if got := aiGraphPromotion("approved", candidate); got != "approved_context" || aiProjectionScope(got) != "context" {
			t.Fatalf("approved %s stayed held: %s", kind, got)
		}
		for _, decision := range []string{"rejected", "needs_info"} {
			if got := aiGraphPromotion(decision, candidate); got != "held" || aiProjectionScope(got) != "none" {
				t.Fatalf("%s %s was promoted: %s", decision, kind, got)
			}
		}
	}
	account := aiCandidateForStaging(service.AICandidate{Kind: "history", ReviewDomain: "ai_chat_account"})
	if got := aiGraphPromotion("approved", account); got != "approved_context" {
		t.Fatalf("approved attributed account stayed held: %s", got)
	}
	ledger := aiReviewRationale("source", strings.Repeat("a", 64), "owner", strings.Repeat("b", 64), "working.candidate_fact", "history", "ai_chat_account", aiGraphPromotion("approved", account))
	if ledger["candidate_kind"] != "fact" || ledger["reported_kind"] != "history" ||
		ledger["review_domain"] != "ai_chat_account" || ledger["graph_promotion"] != "approved_context" || ledger["projection_scope"] != "context" {
		t.Fatalf("owner ledger lost attributed AI account projection: %v", ledger)
	}
	for _, candidate := range []service.AICandidate{
		{Kind: "work_product", ReviewDomain: "ai_chat_content"},
		{Kind: "fact", ReviewDomain: "unknown"},
	} {
		if got := aiGraphPromotion("approved", candidate); got != "held" {
			t.Fatalf("ineligible candidate was promoted: %s", got)
		}
	}
}
