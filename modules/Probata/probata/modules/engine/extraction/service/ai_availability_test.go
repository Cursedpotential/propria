package service

import (
	"encoding/json"
	"testing"
	"time"
)

// TestAIAvailabilityCorrectionPreservesUnknownAndOriginal checks explicit owner dates without altering native clocks.
// Inputs: unknown and first-party native availability. Outputs: bounded effective clock or validation error.
// Effects: none; choose for discovery-date, null and future-date regressions.
func TestAIAvailabilityCorrectionPreservesUnknownAndOriginal(t *testing.T) {
	at := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	var absent *AIAvailabilityCorrection
	if got, err := absent.Effective(nil, at); err != nil || got != nil {
		t.Fatal("unknown availability inferred")
	}
	c := AIAvailabilityCorrection{FieldKey: "source:source_available_from", SourceValue: json.RawMessage(`null`), CorrectedValue: json.RawMessage(`"2026-10-09T12:00:00-04:00"`), ChangeReason: "Owner recalls receiving this material then", Action: "correct"}
	got, err := c.Effective(nil, at)
	if err != nil || got == nil || got.Hour() != 16 {
		t.Fatalf("attributed clock: %v %v", got, err)
	}
	original := at.Add(-48 * time.Hour)
	c.SourceValue, _ = json.Marshal(original)
	if _, err = c.Effective(&original, at); err != nil {
		t.Fatal(err)
	}
	c.SourceValue = json.RawMessage(`null`)
	if _, err = c.Effective(&original, at); err == nil {
		t.Fatal("original source clock mismatch admitted")
	}
	c.SourceValue = nil
	if c.Validate() == nil {
		t.Fatal("omitted original unknown admitted")
	}
	c.SourceValue = json.RawMessage(`null`)
	c.CorrectedValue = json.RawMessage(`"2027-01-01T00:00:00Z"`)
	if _, err = c.Effective(nil, at); err == nil {
		t.Fatal("future discovery admitted")
	}
	c.CorrectedValue = json.RawMessage(`"2026-10-09"`)
	if c.Validate() == nil {
		t.Fatal("date without timezone admitted")
	}
	c.FieldKey = "source:occurred_at"
	if c.Validate() == nil {
		t.Fatal("event clock correction admitted")
	}
}

// TestAIAvailabilityRetractAndDecisionPrecision preserves unknown withdrawal and PG microsecond readback.
// Inputs: a withdrawn owner overlay and clocks within one database microsecond. Outputs: stable effective values.
// Effects: none; choose for retraction and ledger timestamp precision regressions.
func TestAIAvailabilityRetractAndDecisionPrecision(t *testing.T) {
	at := time.Date(2026, 10, 10, 12, 0, 0, 123456000, time.UTC)
	c := AIAvailabilityCorrection{FieldKey: "source:source_available_from", SourceValue: json.RawMessage(`null`), CorrectedValue: json.RawMessage(`"2026-10-10T12:00:00.123456789Z"`), ChangeReason: "Owner recorded receipt", Action: "correct"}
	if _, err := c.Effective(nil, at); err != nil {
		t.Fatalf("PG precision changed acceptance: %v", err)
	}
	c.Action = "retract"
	c.CorrectedValue = json.RawMessage(`null`)
	c.SupersedesRef = "00000000-0000-0000-0000-000000000009"
	if got, err := c.Effective(nil, at); err != nil || got != nil {
		t.Fatal("withdrawn unknown was inferred")
	}
}
