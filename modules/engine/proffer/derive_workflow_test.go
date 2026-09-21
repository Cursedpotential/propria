// Byline: Claude Code · Opus 5 · 2026-09-20

package proffer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

func placeholderDeriveActivity(_ context.Context, _ StageRequest) (DeriveResult, error) {
	return DeriveResult{}, errors.New("proffer: placeholder derive activity ran unmocked")
}

// stagesBeforeExtraction are the stages that run before the handler decision
// is known. The derive route terminates immediately after that decision, so
// these are the only canon stages a derive run schedules.
var stagesBeforeExtraction = []stagegraph.StageID{
	stagegraph.RegisterSource, stagegraph.RetainOriginal,
	stagegraph.AssessSourceRepair, stagegraph.ResolveSourceRepair,
	stagegraph.CaptureFilesystemMetadata, stagegraph.FingerprintSource,
	stagegraph.InventoryContainer, stagegraph.ExtractEmbeddedMetadata,
}

func deriveStageStub() DeriveResult {
	return DeriveResult{
		Result: StageResult{
			Status: StatusSuccess, Ref: "derived-result-ref", ReceiptRef: "derived-receipt-ref",
		},
		SourceLocator: "b2://bkt/vault/sms.xml", SourceSHA256: "aa", SourceBytes: 584_000_000,
		ManifestURI:    "b2://bkt/vault/sms.xml.derived/manifest.json",
		ManifestSHA256: "bb", DerivedPrefix: "b2://bkt/vault/sms.xml.derived/",
		Schema: "smsthreads/v2", Records: 12389, MediaObjects: 564,
		ChunkCount: 132, ThreadCount: 41,
		Chunks: []DerivedChunkRef{{
			Thread: "8105550101", Chunk: 1, URI: "b2://bkt/vault/sms.xml.derived/threads/8105550101.0001.ndjson",
			SHA256: "cc", Records: 100, Bytes: 4096, DeclaredFormat: "ndjson",
		}},
	}
}

func mockDeriveRoute(env *testsuite.TestWorkflowEnvironment, derived DeriveResult, deriveErr error) {
	registerAllStages(env)
	mockHandlerActivities(env, "smsbackuprestore_xml", HandlerPathDerive)
	for _, id := range stagesBeforeExtraction {
		env.OnActivity(string(id), mock.Anything, mock.Anything).Return(stageStub(id), nil).Once()
	}
	env.OnActivity(string(stagegraph.DeriveStructuredText), mock.Anything, mock.Anything).
		Return(derived, deriveErr).Once()
}

// TestDeriveRouteEndsAfterDerivingStructuredText is the branch proof: a
// derive-routed source runs the observation stages, derives, and stops. It
// schedules no parser selection, no parser execution, no raw generation, and
// no seal or publish — there is nothing to seal, because derive produces no
// parser bundle.
func TestDeriveRouteEndsAfterDerivingStructuredText(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	order := newOrderRecorder(env)
	mockDeriveRoute(env, deriveStageStub(), nil)
	approveHold(env)

	input := testInput()
	input.DeclaredFormat = "smsbackuprestore_xml"
	env.ExecuteWorkflow(ProfferWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("derive route failed: %v", err)
	}

	var result WorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("decode workflow result: %v", err)
	}
	if result.Status != StatusSuccess {
		t.Fatalf("status = %q, want success", result.Status)
	}
	if result.PublicationRef != "" {
		t.Fatalf("PublicationRef = %q; a derive run publishes no sealed generation", result.PublicationRef)
	}
	if result.Derived == nil {
		t.Fatal("terminal result carries no derivation summary")
	}
	if result.Derived.ManifestURI != "b2://bkt/vault/sms.xml.derived/manifest.json" {
		t.Fatalf("manifest reference = %q", result.Derived.ManifestURI)
	}
	if result.Derived.ChunkCount != 132 || len(result.Derived.Chunks) != 1 {
		t.Fatalf("derived chunk references = %d of %d", len(result.Derived.Chunks), result.Derived.ChunkCount)
	}
	if got := result.Derived.Chunks[0].DeclaredFormat; got != "ndjson" {
		t.Fatalf("successor declared format = %q, want ndjson", got)
	}

	for _, forbidden := range []stagegraph.StageID{
		stagegraph.SelectParser, stagegraph.ExecuteParser, stagegraph.PersistRawGeneration,
		stagegraph.SealGeneration, stagegraph.PublishGeneration, stagegraph.PublishPreview,
	} {
		if order.contains(string(forbidden)) {
			t.Errorf("derive route scheduled %q; it has no parser bundle to persist", forbidden)
		}
	}
	if !order.contains(string(stagegraph.DeriveStructuredText)) {
		t.Fatal("derive route never scheduled derive_structured_text_activity")
	}

	state := queryOperation(t, env)
	if state.Lifecycle != OperationCompleted || !state.Terminal {
		t.Fatalf("operation lifecycle = %q terminal=%v, want a completed terminal run", state.Lifecycle, state.Terminal)
	}
	if state.DeriveManifestRef != "derived-result-ref" || state.DeriveManifestURI == "" || state.DerivedChunkCount != 132 {
		t.Fatalf("operation state does not report the derivation: %+v", state)
	}
}

// TestDeriveRouteFailsClosedOnUnusableDerivation proves the workflow refuses
// a derivation a successor run could not ingest, rather than reporting a
// success with nothing behind it.
func TestDeriveRouteFailsClosedOnUnusableDerivation(t *testing.T) {
	for name, mutate := range map[string]func(*DeriveResult){
		"no manifest":  func(d *DeriveResult) { d.ManifestURI = "" },
		"no digest":    func(d *DeriveResult) { d.ManifestSHA256 = "" },
		"no chunks":    func(d *DeriveResult) { d.ChunkCount = 0 },
		"no prefix":    func(d *DeriveResult) { d.DerivedPrefix = "" },
		"chunk format": func(d *DeriveResult) { d.Chunks[0].DeclaredFormat = "" },
	} {
		t.Run(name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(ProfferWorkflow)
			derived := deriveStageStub()
			mutate(&derived)
			mockDeriveRoute(env, derived, nil)
			approveHold(env)

			input := testInput()
			input.DeclaredFormat = "smsbackuprestore_xml"
			env.ExecuteWorkflow(ProfferWorkflow, input)

			if !env.IsWorkflowCompleted() {
				t.Fatal("workflow did not complete")
			}
			if env.GetWorkflowError() == nil {
				t.Fatal("an unusable derivation was accepted")
			}
		})
	}
}

// TestDeriveRouteFailsClosedWhenTheActivityFails keeps the ordinary
// fail-closed contract: a failed derive is a failed run, not a partial
// success that quietly proceeds.
func TestDeriveRouteFailsClosedWhenTheActivityFails(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	mockDeriveRoute(env, DeriveResult{}, errors.New("object store unavailable"))
	approveHold(env)

	input := testInput()
	input.DeclaredFormat = "smsbackuprestore_xml"
	env.ExecuteWorkflow(ProfferWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if env.GetWorkflowError() == nil {
		t.Fatal("a failed derivation was reported as success")
	}
	state := queryOperation(t, env)
	if state.Lifecycle != OperationFailed {
		t.Fatalf("operation lifecycle = %q, want failed", state.Lifecycle)
	}
}
