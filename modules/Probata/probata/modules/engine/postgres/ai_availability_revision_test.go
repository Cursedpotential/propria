package postgres

import (
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"testing"
)

// TestAIAvailabilityRevisionRequiresCurrentDecision checks explicit append-only approval revision links.
// Inputs: pending, approved, rejected and needs-info states with current receipt. Outputs: conflict or accepted revision.
// Effects: none; choose to prevent stale revisions, implicit reapproval and repeated withdrawal.
func TestAIAvailabilityRevisionRequiresCurrentDecision(t *testing.T) {
	id := "00000000-0000-0000-0000-000000000009"
	c := &service.AIAvailabilityCorrection{Action: "correct"}
	if err := validateAIAvailabilityRevision("pending", "", nil, c); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"approved", "rejected", "needs_info"} {
		if validateAIAvailabilityRevision(state, id, nil, c) == nil {
			t.Fatal("revision without explicit current ref admitted")
		}
		c.SupersedesRef = id
		if err := validateAIAvailabilityRevision(state, id, nil, c); err != nil {
			t.Fatal(err)
		}
		c.SupersedesRef = "other"
		if validateAIAvailabilityRevision(state, id, nil, c) == nil {
			t.Fatal("stale revision admitted")
		}
		c.SupersedesRef = ""
	}
	if validateAIAvailabilityRevision("approved", id, nil, nil) == nil {
		t.Fatal("implicit reapproval admitted")
	}
	c.Action = "retract"
	c.SupersedesRef = id
	if validateAIAvailabilityRevision("approved", id, []byte(`{}`), c) == nil {
		t.Fatal("withdrawal without active correction admitted")
	}
	receipt, _ := json.Marshal(map[string]any{"metadata_correction": map[string]string{"action": "correct"}})
	if err := validateAIAvailabilityRevision("approved", id, receipt, c); err != nil {
		t.Fatal(err)
	}
}
