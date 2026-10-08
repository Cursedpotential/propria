// Byline: Codex · GPT-6 · 2026-10-07
package surrealsink

import (
	"strings"
	"testing"
	"time"
)

func TestApprovedHistoricalPrefilterUsesSourceClockNotApprovalTime(t *testing.T) {
	cutoff := time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC)
	known := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	later := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	if !eligibleAt("as_lived", known, cutoff) || eligibleAt("as_lived", later, cutoff) || eligibleAt("as_lived", time.Time{}, cutoff) {
		t.Fatal("as-lived source availability filter is wrong")
	}
	if !eligibleAt("hindsight", later, cutoff) || eligibleAt("hindsight", time.Time{}, cutoff) {
		t.Fatal("hindsight included an unknown source or excluded later source")
	}
	where := approvedWhere("as_lived")
	for _, field := range []string{"matter_id=$matter", "case_id=$case", "approved_revision_id=$revision", "approval_digest=$digest", "generation_id=$generation", "source_available_from <= <datetime> $horizon"} {
		if !strings.Contains(where, field) {
			t.Fatalf("missing prefilter %q", field)
		}
	}
	if strings.Contains(where, "approved_at") || strings.Contains(approvedWhere("hindsight"), "$horizon") {
		t.Fatal("approval time leaked into historical cutoff or hindsight was horizon-filtered")
	}
	if validApprovedLimit(0) || !validApprovedLimit(100) || validApprovedLimit(101) {
		t.Fatal("query fanout limit changed")
	}
}
