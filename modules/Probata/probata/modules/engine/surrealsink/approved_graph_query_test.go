// Byline: Codex · GPT-6 · 2026-10-07
package surrealsink

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
)

func TestApprovedHistoricalPrefilterUsesSourceClockNotApprovalTime(t *testing.T) {
	cutoff := time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC)
	known := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	later := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	if !eligibleAt("as_lived", &known, cutoff) || eligibleAt("as_lived", &later, cutoff) || eligibleAt("as_lived", nil, cutoff) {
		t.Fatal("as-lived source availability filter is wrong")
	}
	if !eligibleAt("hindsight", &later, cutoff) || !eligibleAt("hindsight", nil, cutoff) {
		t.Fatal("hindsight excluded a later or explicitly unknown source")
	}
	where := approvedWhere("as_lived")
	for _, field := range []string{"matter_id=$matter", "case_id=$case", "approved_revision_id=$revision", "approval_digest=$digest", "generation_id=$generation", "source_available_from != NONE", "source_available_from <= <datetime> $horizon"} {
		if !strings.Contains(where, field) {
			t.Fatalf("missing prefilter %q", field)
		}
	}
	if strings.Contains(where, "approved_at") || strings.Contains(approvedWhere("hindsight"), "$horizon") || strings.Contains(approvedWhere("hindsight"), "source_available_from != NONE") {
		t.Fatal("approval time leaked into historical cutoff or hindsight excluded unknown availability")
	}
	if validApprovedLimit(0) || !validApprovedLimit(100) || validApprovedLimit(101) {
		t.Fatal("query fanout limit changed")
	}
}

func TestApprovedCursorBindsScopeAndTotalOrder(t *testing.T) {
	known := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	q := ApprovedQueryScope{MatterID: "matter", CourtCaseID: "case", AccessPolicyID: "policy",
		ApprovedRevisionID: "revision", ApprovalDigest: strings.Repeat("a", 64),
		ProjectionGenerationID: "generation", ProjectionHash: strings.Repeat("b", 64),
		Perspective: "hindsight", Limit: 2}
	claims := []approvedgraph.Claim{
		{ID: "same", Kind: "statement", SourceAvailableFrom: &known},
		{ID: "same", Kind: "entity_mention", SourceAvailableFrom: &known},
		{ID: "unknown", Kind: "event_account"},
	}
	sort.Slice(claims, func(i, j int) bool { return approvedClaimLess(claims[i], claims[j]) })
	if claims[0].Kind != "entity_mention" || claims[1].Kind != "statement" || claims[2].ID != "unknown" {
		t.Fatal("approved claim order lost typed-table tie or placed unknown before known")
	}
	encoded := encodeApprovedCursor(q, claims[0])
	cursor, err := decodeApprovedCursor(q, encoded)
	if err != nil || cursor.Table != "ctx_entity_mention" || cursor.ID != "same" || !strings.Contains(approvedAfterCursor(cursor, "ctx_statement"), "'ctx_statement' > $cursor_table") {
		t.Fatalf("typed-table continuation failed: cursor=%+v err=%v", cursor, err)
	}
	q.Limit = 3
	if _, err := decodeApprovedCursor(q, encoded); err == nil {
		t.Fatal("cursor reused with a different page scope")
	}
	q.Limit = 2
	q.ApprovedRevisionID = "other-revision"
	if _, err := decodeApprovedCursor(q, encoded); err == nil {
		t.Fatal("cursor reused with a different approved revision")
	}
	q.ApprovedRevisionID = "revision"
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var altered approvedCursor
	if err := json.Unmarshal(raw, &altered); err != nil {
		t.Fatal(err)
	}
	altered.ID = "other"
	raw, err = json.Marshal(altered)
	if err != nil {
		t.Fatal(err)
	}
	tampered := base64.RawURLEncoding.EncodeToString(raw)
	if _, err := decodeApprovedCursor(q, tampered); err == nil {
		t.Fatal("tampered cursor accepted")
	}
	unknownCursor, err := decodeApprovedCursor(q, encodeApprovedCursor(q, claims[2]))
	if err != nil || !strings.Contains(approvedAfterCursor(unknownCursor, "ctx_event_account"), "source_available_from = NONE") {
		t.Fatal("unknown-availability continuation lost")
	}
}

func TestApprovedQueryScopeDecodesHTTPFieldNames(t *testing.T) {
	var scope ApprovedQueryScope
	raw := `{"matter_id":"matter","court_case_id":"case","access_policy_id":"policy","approved_revision_id":"revision","approval_digest":"digest","projection_generation_id":"generation","projection_hash":"hash","perspective":"hindsight","limit":2,"cursor":"continuation"}`
	if err := json.Unmarshal([]byte(raw), &scope); err != nil {
		t.Fatal(err)
	}
	if scope.MatterID != "matter" || scope.CourtCaseID != "case" || scope.AccessPolicyID != "policy" || scope.ApprovedRevisionID != "revision" || scope.ApprovalDigest != "digest" || scope.ProjectionGenerationID != "generation" || scope.ProjectionHash != "hash" || scope.Perspective != "hindsight" || scope.Limit != 2 || scope.Cursor != "continuation" {
		t.Fatalf("snake-case HTTP query scope was not decoded: %+v", scope)
	}
}

// TestApprovedNativePayloadRoundTrip keeps a zero-start citation and typed predicate inside immutable graph hashes.
// Inputs: one native AI claim JSON and column-shaped reconstruction. Outputs: hash agreement or tamper rejection.
// Effects: none. Pick for the native payload extension; SMS optional fields stay absent.
func TestApprovedNativePayloadRoundTrip(t *testing.T) {
	start, end := 0, 4
	claim := approvedgraph.Claim{ID: "claim", Kind: "statement", Text: "typed statement", Predicate: "direct_account",
		NativeSpanStart: &start, NativeSpanEnd: &end, NativeSpanUnit: "unicode_codepoint", NativeSpanSHA256: strings.Repeat("a", 64)}
	raw, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	copy := claim
	copy.Predicate, copy.NativeSpanStart, copy.NativeSpanEnd, copy.NativeSpanUnit, copy.NativeSpanSHA256 = "", nil, nil, "", ""
	if err := restoreApprovedNativeFields(string(raw), &copy); err != nil {
		t.Fatal(err)
	}
	if copy.NativeSpanStart == nil || *copy.NativeSpanStart != 0 || graphHash(copy) != graphHash(claim) {
		t.Fatal("native locator was lost in query reconstruction")
	}
	copy.Predicate = "changed"
	if graphHash(copy) == graphHash(claim) {
		t.Fatal("typed predicate tampering kept immutable hash")
	}
	if err := restoreApprovedNativeFields("{", &copy); err == nil {
		t.Fatal("malformed stored payload admitted")
	}
}
