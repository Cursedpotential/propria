package contextgraphflow

import (
	"strings"
	"testing"
)

// formatPreparation supplies reference-only boundary values, never a corpus/projection proof.
// Inputs none; outputs small typed values. No external effects occur.
func formatPreparation() (Request, PreparationResult) {
	r := Request{RequestID: "format-request", SourceVersionID: "format-version", NormalizedGenerationID: "format-generation", VerificationID: "format-verification", OperatingMode: "LIVE", MatterID: "format-matter", CourtCaseID: "format-case", PreparedRef: "file:///format/prepared.json", WorkProductsRef: "file:///format/works.json", AccessPolicyID: "format-policy", CreatedByService: "format-service", ExpectedSourceTurns: 1, ExpectedCreatedWorks: 1, ExpectedConversations: 1}
	p := PreparationResult{RequestID: r.RequestID, SourceVersionID: r.SourceVersionID, NormalizedGenerationID: r.NormalizedGenerationID, VerificationID: r.VerificationID, OperatingMode: r.OperatingMode, MatterID: r.MatterID, CourtCaseID: r.CourtCaseID, ManifestRef: "file:///format/manifest.json", ManifestHash: strings.Repeat("a", 64), SourceTurns: 1, CreatedWorks: 1, Conversations: 1, Batches: 1, BatchRefs: []BatchRef{{BundleRef: "file:///format/batch.json", BundleSHA256: strings.Repeat("b", 64), GenerationID: "format-projection", ExtractionRunRef: r.PreparedRef, SourceTurns: 1, CreatedWorks: 1, Nodes: 2, Edges: 1}}, TemporalWorkflowID: "format-workflow", TemporalRunID: "format-run"}
	return r, p
}

// TestContextGraphPreparationRequiresExactBindingsAndCompleteManifest checks scheduling boundaries.
// Inputs are reference-only format values; output is assertions. No Activities,
// database, corpus or model calls occur; this is not live workflow delivery proof.
func TestContextGraphPreparationRequiresExactBindingsAndCompleteManifest(t *testing.T) {
	r, p := formatPreparation()
	if e := validateRequest(r); e != nil {
		t.Fatal(e)
	}
	if e := validatePreparation(r, p, "format-workflow", "format-run"); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*PreparationResult){func(p *PreparationResult) { p.VerificationID = "different" }, func(p *PreparationResult) { p.TemporalRunID = "different" }, func(p *PreparationResult) { p.SourceTurns = 2 }, func(p *PreparationResult) { p.BatchRefs[0].SourceTurns = 0 }, func(p *PreparationResult) { p.BatchRefs[0].ExtractionRunRef = "different" }, func(p *PreparationResult) { p.Batches = 2; p.BatchRefs = append(p.BatchRefs, p.BatchRefs[0]) }} {
		_, changed := formatPreparation()
		mutate(&changed)
		if validatePreparation(r, changed, "format-workflow", "format-run") == nil {
			t.Fatal("changed bindings/completeness admitted")
		}
	}
	r.ExpectedSourceTurns = 0
	if validateRequest(r) == nil {
		t.Fatal("implicit expected count admitted")
	}
}
