// Byline: Codex | GPT-6.1-sol | 2026-10-07
package activities

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/contextgraphflow"
	"github.com/Cursedpotential/probata/engine/surrealsink"
	"go.temporal.io/sdk/activity"
)

// contextGraphFormatInput creates tiny retained format values without source/corpus data.
// Inputs are the test handle; outputs are a configured group and sealed request.
// Effects create temporary format JSON only; files remain for owner cleanup.
func contextGraphFormatInput(t *testing.T) (ContextGraphActivities, ContextGraphActivityRequest) {
	t.Helper()
	root, e := os.MkdirTemp("", "context-graph-activity-format-")
	if e != nil {
		t.Fatal(e)
	}
	scope := surrealsink.ContextGraphScope{MatterID: caseidentity.AuthoritativeMatterID, CaseID: caseidentity.AuthoritativeCourtCaseID, AccessPolicyID: "format-policy", CreatedByService: "format-service"}
	a, e := NewContextGraphActivities(ContextGraphActivityConfig{Root: root, Scope: scope, Surreal: surrealsink.Config{URL: "http://127.0.0.1:1", Namespace: "fct", Database: "analysis", AuthLevel: "database", User: "format-user", Password: "format-password"}})
	if e != nil {
		t.Fatal(e)
	}
	pins := []surrealsink.ContextSourcePin{{SourceID: "format-source", SourceVersionID: "format-version", SourceHash: strings.Repeat("a", 64), Locator: "format:span"}}
	b := surrealsink.ContextGraphBundle{Scope: scope, GenerationID: "format-graph", ExtractionRunRef: "file:///format/prepared.json", Nodes: []surrealsink.ContextGraphNode{{NodeID: "format-turn", Kind: "ctx_content_unit", DerivativeKind: "ai_source_turn", SourcePins: pins}, {NodeID: "format-work", Kind: "ctx_content_unit", DerivativeKind: "created_work", SourcePins: pins, Body: "format value", InputNodeIDs: []string{"format-turn"}}}, Edges: []surrealsink.ContextGraphEdge{{EdgeID: "format-dependency", Kind: "depends_on", FromNodeID: "format-work", ToNodeID: "format-turn", SourcePins: pins}}}
	raw, _ := json.Marshal(b)
	path := filepath.Join(root, "bundle.json")
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(raw)
	request := ContextGraphActivityRequest{Request: contextgraphflow.Request{RequestID: "format-request", SourceVersionID: "format-version", NormalizedGenerationID: "format-normalized", VerificationID: "format-verification", OperatingMode: "LIVE", MatterID: scope.MatterID, CourtCaseID: scope.CaseID, PreparedRef: b.ExtractionRunRef, WorkProductsRef: "file:///format/works.json", AccessPolicyID: scope.AccessPolicyID, CreatedByService: scope.CreatedByService}, BundleRef: string(toolkitFileRef(path)), BundleSHA256: hex.EncodeToString(hash[:]), CaseID: scope.CaseID, GenerationID: b.GenerationID, ExtractionRunRef: b.ExtractionRunRef}
	return a, request
}

// TestContextGraphSealedInputRejectsHashScopeRunAndEscapedPaths verifies pre-query admission.
// Inputs are small format-only files; output is assertions. No network/database,
// corpus or model calls occur; accepted packets are never projected by this test.
func TestContextGraphSealedInputRejectsHashScopeRunAndEscapedPaths(t *testing.T) {
	a, r := contextGraphFormatInput(t)
	r.SourcePins = []surrealsink.ContextSourcePin{{SourceID: "format-source", SourceVersionID: "format-version", SourceHash: strings.Repeat("a", 64), Locator: "format:original-root"}}
	if _, e := a.ValidateSealedInput(r); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*ContextGraphActivityRequest){func(r *ContextGraphActivityRequest) { r.BundleSHA256 = strings.Repeat("0", 64) }, func(r *ContextGraphActivityRequest) { r.AccessPolicyID = "different" }, func(r *ContextGraphActivityRequest) { r.GenerationID = "different" }, func(r *ContextGraphActivityRequest) { r.ExtractionRunRef = "different" }, func(r *ContextGraphActivityRequest) { r.BundleRef = "file:///outside/bundle.json" }, func(r *ContextGraphActivityRequest) { r.CourtCaseID = "different" }, func(r *ContextGraphActivityRequest) { r.SourcePins = nil }, func(r *ContextGraphActivityRequest) {
		r.SourcePins = []surrealsink.ContextSourcePin{{SourceID: "wrong-root"}}
	}} {
		changed := r
		mutate(&changed)
		if _, e := a.load(changed); e == nil {
			t.Fatal("changed sealed input admitted")
		}
	}
	if _, e := (ContextGraphActivities{}).load(r); e == nil {
		t.Fatal("unconfigured Activity admitted")
	}
}

type contextGraphRegistrar struct{ names map[string]bool }

func (r *contextGraphRegistrar) RegisterActivityWithOptions(_ interface{}, options activity.RegisterOptions) {
	r.names[options.Name] = true
}

// TestContextGraphNamedRegistrationIncludesIndependentVerify checks production seam names.
// Input is an in-memory registrar; output is assertions, with no runtime scheduling.
func TestContextGraphNamedRegistrationIncludesIndependentVerify(t *testing.T) {
	r := &contextGraphRegistrar{names: map[string]bool{}}
	RegisterContextGraphActivities(r, ContextGraphActivities{})
	for _, name := range []string{ProjectContextGraphActivityName, VerifyContextGraphActivityName, TraverseContextGraphActivityName} {
		if !r.names[name] {
			t.Fatal("missing named Activity", name)
		}
	}
	if len(r.names) != 3 {
		t.Fatal("unexpected registration")
	}
}

// TestContextGraphOfflineAdmissionRejectsNonLiveMode keeps existing canonical-write admission.
// Inputs are tiny format values; output is assertions; no network or graph write occurs.
func TestContextGraphOfflineAdmissionRejectsNonLiveMode(t *testing.T) {
	a, r := contextGraphFormatInput(t)
	r.SourcePins = []surrealsink.ContextSourcePin{{SourceID: "format-source", SourceVersionID: "format-version", SourceHash: strings.Repeat("a", 64), Locator: "format:root"}}
	for _, mode := range []string{"DEV", "REAL", ""} {
		r.OperatingMode = mode
		if _, e := a.ValidateSealedInput(r); e == nil {
			t.Fatal("non-LIVE durable mode admitted")
		}
	}
}
