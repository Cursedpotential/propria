package approvedgraphai

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestResolveAttributedAvailabilityBindsLedgerAndReceipt checks the owner overlay without rehashing original proposals.
// Inputs: original approved fixture and matching/disagreeing correction metadata. Outputs: attributed clock or error.
// Effects: none; choose for unknown native discovery and tampered ledger/source-value regressions.
func TestResolveAttributedAvailabilityBindsLedgerAndReceipt(t *testing.T) {
	scope, row := approvedFixture(t)
	var attrs, ledger map[string]any
	_ = json.Unmarshal(row.Attrs, &attrs)
	_ = json.Unmarshal(row.LedgerRationale, &ledger)
	c := map[string]any{"field_key": "source:source_available_from", "source_value": nil, "corrected_value": "2026-10-07T12:00:00Z", "action": "correct", "change_reason": "Owner supplied actual discovery date"}
	receipt := attrs["review_decision"].(map[string]any)
	receipt["metadata_correction"] = c
	receipt["revision"] = 1
	ledger["metadata_correction"] = c
	ledger["revision"] = 1
	row.Attrs, _ = json.Marshal(attrs)
	row.LedgerRationale, _ = json.Marshal(ledger)
	claim, err := Resolve(context.Background(), staticAIReader{row: row}, scope)
	if err != nil || claim.SourceAvailableFrom == nil || claim.SourceAvailableFrom.Format(time.RFC3339) != "2026-10-07T12:00:00Z" || claim.CandidateSHA256 != row.CandidateSHA256 {
		t.Fatalf("original hash / owner availability: %+v %v", claim, err)
	}
	c["change_reason"] = "altered"
	row.LedgerRationale, _ = json.Marshal(ledger)
	if _, err = Resolve(context.Background(), staticAIReader{row: row}, scope); err == nil {
		t.Fatal("ledger and receipt disagreement admitted")
	}
	c["change_reason"] = "Owner supplied actual discovery date"
	c["source_value"] = "2026-10-01T00:00:00Z"
	row.Attrs, _ = json.Marshal(attrs)
	row.LedgerRationale, _ = json.Marshal(ledger)
	if _, err = Resolve(context.Background(), staticAIReader{row: row}, scope); err == nil {
		t.Fatal("invented original clock admitted")
	}
}
