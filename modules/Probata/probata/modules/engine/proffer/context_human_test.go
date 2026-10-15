package proffer

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Cursedpotential/probata/engine/humancontext"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// humanContextFixture constructs metadata-only workflow input for SDK composition tests.
// Inputs: none. Outputs: synthetic coordinates without source content. Effects: none.
// Choose for routing/receipt tests; it never opens a corpus or a service connection.
func humanContextFixture() WorkflowInput {
	return WorkflowInput{ContextContract: ContextContractVersion, RequestID: ContextWorkflowIDPrefix + "synthetic-human",
		ActorSubjectUID: "synthetic-actor", SourceRef: "b2://synthetic/source.xml", ProviderVersionID: "native-version",
		SourceKind: "sms", DeclaredFormat: "smsbackuprestore_xml", HumanSource: &humancontext.Source{
			SourceRef: "untrusted-input-source", SourceVersionRef: "untrusted-input-version", RegistrationReceiptRef: "untrusted-input-receipt",
			OwnerPersonRef: "synthetic-owner-reference", PerspectivePersonRef: "synthetic-perspective-reference",
			Perspective: "source-device", ByteLength: 128, ETag: "synthetic-etag"}}
}

// TestHumanContextBridgePreservesPinsAndSeparateStages exercises the registered native workflow seam.
// Inputs: synthetic receipts, including one pending record. Outputs: stage, pin and summary assertions.
// Effects: Temporal SDK memory environment only; choose to catch AI contamination and lost partial coverage.
func TestHumanContextBridgePreservesPinsAndSeparateStages(t *testing.T) {
	for _, pending := range []int64{0, 1} {
		t.Run(map[int64]string{0: "complete", 1: "pending"}[pending], func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			input := humanContextFixture()
			var order []string
			env.RegisterActivityWithOptions(func(_ context.Context, req ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
				order = append(order, contextRegisterActivityName)
				if req.SourceRef != string(input.SourceRef) || req.DeclaredFormat != "smsbackuprestore_xml" {
					t.Error("registration lost source coordinates")
				}
				if req.ByteLength != input.HumanSource.ByteLength || req.ETag != input.HumanSource.ETag {
					t.Error("registration lost original length or ETag before streaming")
				}
				return ContextSourceRegistrationResult{SourceVersionRef: "registered-version", ReceiptRef: "registration-receipt", Status: "registered"}, nil
			}, activity.RegisterOptions{Name: contextRegisterActivityName})
			previous := ""
			for _, spec := range []struct{ name, stage string }{{humancontext.FrameActivity, "framed"}, {humancontext.ExtractActivity, "extracted"}, {humancontext.ChunkActivity, "chunked"}, {humancontext.IndexActivity, "published"}, {humancontext.VerifyActivity, "verified"}} {
				spec := spec
				env.RegisterActivityWithOptions(func(_ctx context.Context, req humancontext.Request) (humancontext.Result, error) {
					order = append(order, spec.name)
					want := *input.HumanSource
					want.SourceRef = string(input.SourceRef)
					want.ProviderVersionID = input.ProviderVersionID
					want.SourceVersionRef = "registered-version"
					want.RegistrationReceiptRef = "registration-receipt"
					want.Format = input.DeclaredFormat
					if req.ContractVersion != humancontext.Contract || req.WorkflowID == "" || req.Source != want || req.ManifestRef != previous {
						t.Error("human stage lost actual registration, attribution or predecessor pin")
					}
					if spec.stage == "chunked" && activity.GetInfo(_ctx).TaskQueue != ContextChunksTaskQueue {
						t.Error("native chunk planner did not use the existing Python queue")
					}
					if spec.stage == "published" || spec.stage == "verified" {
						if req.ChunkManifestRef != "file:///synthetic/chunked.json" {
							t.Error("chunk plan reference was lost")
						}
					}
					previous = "file:///synthetic/" + spec.stage + ".json"
					result := humancontext.Result{ContractVersion: humancontext.Contract, Stage: spec.stage, Status: "complete", ManifestRef: previous, ReceiptRef: spec.stage + "-receipt", Records: 3}
					if spec.stage != "framed" {
						result.Pending = pending
						if pending > 0 {
							result.Status = "partial"
						}
					}
					if spec.stage == "chunked" || spec.stage == "published" || spec.stage == "verified" {
						result.Chunks = 2
					}
					if spec.stage == "published" || spec.stage == "verified" {
						result.Published = 2
						result.IndexKind = humancontext.IndexKind
					}
					if spec.stage == "verified" {
						result.Verified = result.Published
					}
					return result, nil
				}, activity.RegisterOptions{Name: spec.name})
			}
			env.ExecuteWorkflow(contextFirstWorkflow, input)
			if err := env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			var result WorkflowResult
			if err := env.GetWorkflowResult(&result); err != nil {
				t.Fatal(err)
			}
			wantOrder := []string{contextRegisterActivityName, humancontext.FrameActivity, humancontext.ExtractActivity, humancontext.ChunkActivity, humancontext.IndexActivity, humancontext.VerifyActivity}
			if !reflect.DeepEqual(order, wantOrder) {
				t.Fatalf("unexpected activities: %v", order)
			}
			if result.Context == nil || result.Context.Human == nil {
				t.Fatal("native summary absent")
			}
			c := result.Context
			if c.ActorSubjectUID != input.ActorSubjectUID || c.SourceVersionRef != "registered-version" || c.Human.Verified != 2 || c.Human.Chunks != 2 || c.Human.Pending != pending || c.Human.ExtractedRef == "" || c.Human.ChunkManifestRef == "" || c.Human.VerificationRef == "" {
				t.Fatalf("native outcome lost: %+v", c)
			}
			if c.Human.IndexKind != humancontext.IndexKind {
				t.Fatal("native index kind was lost from the receipt summary")
			}
			if c.Package != nil || c.PreparedRef != "" || c.WorkProductsRef != "" || c.PublicationRef != "" || c.VerificationRef != "" || result.AIContent != nil {
				t.Fatal("human output populated AI fields")
			}
			if (pending == 0 && (c.Status != "complete" || result.Status != StatusSuccess)) || (pending > 0 && (c.Status != "partial" || c.Reason != "native_records_pending")) {
				t.Fatalf("false completion: %+v", c)
			}
		})
	}
}

// TestHumanContextLegacyMarkerKeepsParserOutcome checks the pre-bridge version branch.
// Inputs: SMS metadata and DefaultVersion marker. Outputs: original needs_parser result.
// Effects: SDK memory environment only; choose for compatibility, not full recorded-history replay proof.
func TestHumanContextLegacyMarkerKeepsParserOutcome(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.OnGetVersion(contextHumanChangeID, workflow.DefaultVersion, workflow.Version(1)).Return(workflow.DefaultVersion)
	// Register the exact signature before mocking the string-named Activity.
	env.RegisterActivityWithOptions(func(context.Context, ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
		return ContextSourceRegistrationResult{}, nil
	}, activity.RegisterOptions{Name: contextRegisterActivityName})
	env.OnActivity(contextRegisterActivityName, mock.Anything, mock.Anything).Return(ContextSourceRegistrationResult{Status: "registered", SourceVersionRef: "registered-version", ReceiptRef: "registration-receipt"}, nil)
	env.ExecuteWorkflow(contextFirstWorkflow, humanContextFixture())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result WorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.Context == nil || result.Context.Status != "needs_parser" || result.Context.Human != nil {
		t.Fatalf("legacy route changed: %+v", result.Context)
	}
	env.AssertExpectations(t)
}

// TestHumanContextGuardPreservesAICommands checks AI routing never evaluates the human marker.
// Inputs: Gemini metadata and an unavailable synthetic preparation Activity.
// Outputs: unchanged registration/preparation and no human marker evaluation under either mock version.
// Effects: SDK memory environment only; choose to detect new human commands on old AI branches.
func TestHumanContextGuardPreservesAICommands(t *testing.T) {
	for _, version := range []workflow.Version{workflow.DefaultVersion, 1} {
		t.Run(map[workflow.Version]string{workflow.DefaultVersion: "legacy", 1: "current"}[version], func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.OnGetVersion(contextHumanChangeID, workflow.DefaultVersion, workflow.Version(1)).Return(version).Maybe().Run(func(mock.Arguments) {
				t.Error("AI workflow evaluated the native-human version marker")
			})
			input := humanContextFixture()
			input.DeclaredFormat, input.HumanSource = "gemini_markdown", nil
			env.OnGetVersion(contextAtomicAIChangeID, workflow.DefaultVersion, workflow.Version(1)).Return(workflow.DefaultVersion)
			var order []string
			env.RegisterActivityWithOptions(func(context.Context, ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
				order = append(order, contextRegisterActivityName)
				return ContextSourceRegistrationResult{Status: "registered", SourceVersionRef: "registered-version", ReceiptRef: "registration-receipt"}, nil
			}, activity.RegisterOptions{Name: contextRegisterActivityName})
			env.RegisterActivityWithOptions(func(context.Context, aiContextRequest) (aiContextReceipt, error) {
				order = append(order, contextPrepareActivityName)
				return aiContextReceipt{}, temporal.NewNonRetryableApplicationError("synthetic preparation unavailable", "synthetic", nil)
			}, activity.RegisterOptions{Name: contextPrepareActivityName})
			env.ExecuteWorkflow(contextFirstWorkflow, input)
			if err := env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			var result WorkflowResult
			if err := env.GetWorkflowResult(&result); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(order, []string{contextRegisterActivityName, contextPrepareActivityName}) || result.Context == nil || result.Context.Human != nil || result.Context.Reason != "preparation_failed" {
				t.Fatalf("AI commands changed: %v %+v", order, result.Context)
			}
			env.AssertExpectations(t)
		})
	}
}

// TestHumanContextReceiptRejectsFalseCompletion checks external receipt admission.
// Inputs: corrupted contracts, references and counts. Outputs: validation errors.
// Effects: none; choose to prevent invalid stage receipts from authorizing descendants.
func TestHumanContextReceiptRejectsFalseCompletion(t *testing.T) {
	base := humancontext.Result{ContractVersion: humancontext.Contract, Stage: "verified", Status: "complete", ManifestRef: "file:///synthetic/verified.json", ReceiptRef: "synthetic-persisted-verified-receipt", Records: 3, Chunks: 2, Published: 2, Verified: 2}
	if err := validateHumanContextResult(base, "verified"); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*humancontext.Result){
		func(r *humancontext.Result) { r.ContractVersion = "ai-context-v1" },
		func(r *humancontext.Result) { r.Stage = "published" },
		func(r *humancontext.Result) { r.ManifestRef = "" },
		func(r *humancontext.Result) { r.ReceiptRef = "" },
		func(r *humancontext.Result) { r.ReceiptRef = "   " },
		func(r *humancontext.Result) { r.ReceiptRef = "bad\nref" },
		func(r *humancontext.Result) { r.Pending = 4 },
		func(r *humancontext.Result) { r.Verified = 4 },
		func(r *humancontext.Result) { r.Records = -1 },
		func(r *humancontext.Result) { r.Chunks = -1 },
		func(r *humancontext.Result) { r.Published = 3 },
	} {
		r := base
		mutate(&r)
		if validateHumanContextResult(r, "verified") == nil {
			t.Fatalf("invalid receipt accepted: %+v", r)
		}
	}
}

// TestHumanContextStopsDescendantsAndRetainsNativeLineage checks failure isolation and chunk-only publication.
// Inputs: synthetic extraction/chunk failures, short publication or the retired message index kind.
// Outputs: partial status and last usable bundle references. Effects: SDK memory environment only.
// Choose to ensure extraction stays viewable independently of chunk planning and publication status.
func TestHumanContextStopsDescendantsAndRetainsNativeLineage(t *testing.T) {
	for _, failure := range []string{"extract", "chunk", "missing_chunk_receipt", "publication_count", "message_index"} {
		t.Run(failure, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			input := humanContextFixture()
			input.HumanSource = nil // Optional attribution must never block registration.
			env.RegisterActivityWithOptions(func(context.Context, ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
				return ContextSourceRegistrationResult{Status: "registered", SourceVersionRef: "registered-version", ReceiptRef: "registration-receipt"}, nil
			}, activity.RegisterOptions{Name: contextRegisterActivityName})
			var order []string
			for _, spec := range []struct{ name, stage string }{{humancontext.FrameActivity, "framed"}, {humancontext.ExtractActivity, "extracted"}, {humancontext.ChunkActivity, "chunked"}, {humancontext.IndexActivity, "published"}} {
				spec := spec
				env.RegisterActivityWithOptions(func(_ context.Context, req humancontext.Request) (humancontext.Result, error) {
					order = append(order, spec.stage)
					if (failure == "extract" && spec.stage == "extracted") || (failure == "chunk" && spec.stage == "chunked") {
						return humancontext.Result{}, temporal.NewNonRetryableApplicationError("synthetic stage unavailable", "synthetic", errors.New("unavailable"))
					}
					result := humancontext.Result{ContractVersion: humancontext.Contract, Stage: spec.stage, Status: "complete", ManifestRef: "file:///synthetic/" + spec.stage + ".json", ReceiptRef: "synthetic-" + spec.stage + "-receipt", Records: 3}
					if spec.stage == "chunked" || spec.stage == "published" {
						result.Chunks = 2
					}
					if failure == "missing_chunk_receipt" && spec.stage == "chunked" {
						result.ReceiptRef = ""
					}
					if spec.stage == "published" {
						result.IndexKind = humancontext.IndexKind
						if failure == "message_index" {
							result.IndexKind = "native-human-message-projections-v1"
							result.Published = 2
						}
						// publication_count intentionally reports zero published chunks.
					}
					return result, nil
				}, activity.RegisterOptions{Name: spec.name})
			}
			env.ExecuteWorkflow(contextFirstWorkflow, input)
			if err := env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			var result WorkflowResult
			if err := env.GetWorkflowResult(&result); err != nil {
				t.Fatal(err)
			}
			if result.Context == nil || result.Context.Human == nil {
				t.Fatal("native partial summary missing")
			}
			c := result.Context
			if c.Status != "partial" || c.Human.FramedRef == "" || c.Human.VerificationRef != "" || c.Package != nil {
				t.Fatalf("invalid failure outcome: %+v", c)
			}
			wantOrder, wantReason := []string{"framed", "extracted"}, "extracted_failed"
			if failure != "extract" {
				if c.Human.ExtractedRef != "file:///synthetic/extracted.json" {
					t.Fatal("viewer lost extraction after downstream failure")
				}
				wantOrder, wantReason = append(wantOrder, "chunked"), "chunked_failed"
			}
			if failure == "missing_chunk_receipt" {
				wantReason = "chunked_receipt_invalid"
			}
			if failure == "publication_count" || failure == "message_index" {
				wantOrder, wantReason = append(wantOrder, "published"), "publication_count_mismatch"
				if failure == "message_index" {
					wantReason = "chunk_index_kind_mismatch"
				}
			}
			if !reflect.DeepEqual(order, wantOrder) || c.Reason != wantReason {
				t.Fatalf("unexpected descendants: %v, %s", order, c.Reason)
			}
		})
	}
}
