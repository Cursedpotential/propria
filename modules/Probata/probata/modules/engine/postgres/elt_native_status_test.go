// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	"context"
	"io"
	"testing"

	"github.com/Cursedpotential/probata/engine/parser"
)

// TestNativeStructuredRowsTransportStatuses verifies optional five-column scanning.
// Inputs are synthetic PostgreSQL rows; outputs are unchanged raw statuses and
// reasons. No I/O occurs. Pick for native v2 accounting and legacy isolation.
func TestNativeStructuredRowsTransportStatuses(t *testing.T) {
	for _, status := range []string{"parsed", "envelope", "unknown", "malformed", "rejected", "unparsed"} {
		r := &structuredELTRows{rows: &reviewFakeRows{rows: [][]any{{"null", "{}", `{"duckdb_template":"chatgpt_json_array_v2"}`, status, "source explanation"}}}, extendedStatus: true}
		row, err := r.Next(context.Background())
		if err != nil || row.RecordStatus != parser.RecordStatus(status) || row.StatusReason != "source explanation" {
			t.Fatalf("status dropped: %+v %v", row, err)
		}
		if _, err := r.Next(context.Background()); err != io.EOF {
			t.Fatalf("end stream=%v", err)
		}
	}
	legacy := &structuredELTRows{rows: &reviewFakeRows{rows: [][]any{{"native", "{}", `{"duckdb_template":"sms_xml_v1"}`}}}}
	row, err := legacy.Next(context.Background())
	if err != nil || row.RecordStatus != "" || row.StatusReason != "" {
		t.Fatalf("legacy shape changed: %+v %v", row, err)
	}
}
