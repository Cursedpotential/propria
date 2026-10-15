package activities

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// TestNativeContextCandidatePolicyKeepsAccountsExternal verifies precise held counts without converting undated events to facts.
// Inputs: external synthetic native bundle with one entity and seven external-only kinds. Outputs: one staged entity and seven held candidates.
// Effects: retained fixture files and memory-only store; choose to protect the PostgreSQL storage boundary.
func TestNativeContextCandidatePolicyKeepsAccountsExternal(t *testing.T) {
	base := os.Getenv("AI_WORKPRODUCT_TEST_ROOT")
	if base == "" {
		t.Skip("requires retained VPS fixture root")
	}
	unit, err := os.MkdirTemp(base, "native-policy-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_CONTEXT_ROOT", unit)
	ref := func(name string) string {
		return (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(unit, name))}).String()
	}
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(unit, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	original := []byte("External synthetic source excerpt")
	write("original.md", original)
	source := aicontextsource.Identity{SourceRef: "b2://fixture/original.md", SourceFormat: "gemini_markdown", SourceVersionID: "11111111-1111-4111-8111-111111111111", SourceSHA256: aicontextsource.Hash(original)}
	prepared := aicontextsource.Prepared{ContractVersion: "ai-context-v1", Stage: "prepared", Source: source, OriginalRef: ref("original.md"), OriginalSHA256: aicontextsource.Hash(original)}
	raw, _ := json.Marshal(prepared)
	write("prepared.json", raw)
	candidates := []map[string]any{}
	for _, kind := range []string{"fact", "strategy", "history", "document", "work_product", "artifact", "event", "entity"} {
		candidates = append(candidates, map[string]any{"kind": kind, "name": "Fixture Person", "entity_type": "person", "quote": string(original), "status": "unreviewed_context_candidate", "confidence": 0.8, "occurrences": []map[string]any{{"native_json_pointer": "", "start": 0, "end": len(original), "unit": "unicode_codepoint"}}})
	}
	raw, _ = json.Marshal(map[string]any{"contract_version": "ai-context-v1", "stage": "candidates", "source": source, "prepared_ref": ref("prepared.json"), "candidates": candidates})
	write("candidates.json", raw)
	store := &aiStageStore{}
	a := AICandidateStagingActivities{DerivedRoot: base, Store: store}
	result, err := a.StageAICandidateBundle(context.Background(), AICandidateStageInput{ContractVersion: "ai-context-v1", OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "context-source:fixture", Source: service.AISourcePin{SourceVersionID: source.SourceVersionID, SourceRef: source.SourceRef, SourceSHA256: source.SourceSHA256}, PreparedRef: ref("prepared.json"), BundleRef: ref("candidates.json")})
	if err != nil || result.Candidates != 8 || result.Staged != 1 || result.HeldExternal != 7 || len(store.staged) != 1 || store.staged[0].Kind != "entity" || store.staged[0].EvidenceBundleRef != ref("candidates.json") {
		t.Fatalf("native policy/status mismatch: %+v %v", result, err)
	}
}
