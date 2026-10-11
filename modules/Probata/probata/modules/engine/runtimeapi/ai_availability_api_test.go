package runtimeapi

import (
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"testing"
)

// TestAIAvailabilityRequestDigestPreservesLegacyAndBindsCorrection checks exact old request serialization and new metadata binding.
// Inputs: same owner/key and unchanged native pin with optional correction. Outputs: stable legacy and distinct corrected digests.
// Effects: none; choose for backward retry compatibility and correction idempotency.
func TestAIAvailabilityRequestDigestPreservesLegacyAndBindsCorrection(t *testing.T) {
	legacy := struct {
		runRequest
		service.AISourcePin
		CandidateID           string `json:"candidate_id"`
		ExpectedContentSHA256 string `json:"expected_content_sha256"`
		Decision              string `json:"decision"`
	}{Decision: "approved"}
	current := aiDecisionRequest{Decision: "approved"}
	actor := entities.Actor{SubjectUID: "owner", Username: "owner"}
	old := requestDigest("key", actor, legacy)
	if old != requestDigest("key", actor, current) {
		t.Fatal("legacy absent correction changed request digest")
	}
	current.MetadataCorrection = &service.AIAvailabilityCorrection{FieldKey: "source:source_available_from", SourceValue: json.RawMessage(`null`), CorrectedValue: json.RawMessage(`"2026-10-01T00:00:00Z"`), Action: "correct", ChangeReason: "Owner receipt date"}
	first := requestDigest("key", actor, current)
	if first == old {
		t.Fatal("correction not bound")
	}
	current.MetadataCorrection.ChangeReason = "Corrected owner receipt date"
	if requestDigest("key", actor, current) == first {
		t.Fatal("reason not bound")
	}
}
