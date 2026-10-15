package proffer

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// TestAtomicAISeparatesDecodeAndPreservesSearchBeforeEnrichment verifies real sequential SDK command composition.
// Inputs: successful or failed enrichment plus exact registration refs. Outputs: durable upstream refs and truthful counts.
// Effects: in-memory Activities only; choose to detect a failed model stage hiding a successful index publication.
func TestAtomicAISeparatesDecodeAndPreservesSearchBeforeEnrichment(t *testing.T) {
	for _, fixture := range []struct {
		name string
		fail bool
		held int
	}{{"grounded", false, 0}, {"model_failure", true, 0}, {"external_only_candidates", false, 1}} {
		failCandidate := fixture.fail
		t.Run(fixture.name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			order := []string{}
			env.RegisterActivityWithOptions(func(_ context.Context, in ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
				order = append(order, contextRegisterActivityName)
				return ContextSourceRegistrationResult{Status: "registered", SourceVersionRef: "registered", ReceiptRef: "receipt", HistoricalSourceVersionID: in.HistoricalSourceVersionID, SourceSHA256: in.SourceSHA256}, nil
			}, activity.RegisterOptions{Name: contextRegisterActivityName})
			for _, item := range []struct{ name, stage string }{{contextDecodeActivityName, "decoded"}, {contextWorkProductsActivityName, "work_products"}, {contextPrepareActivityName, "prepared"}, {contextEmbedActivityName, "embedded"}, {contextPublishActivityName, "published"}, {contextVerifyActivityName, "verified"}, {contextCandidatesActivityName, "candidates"}} {
				step := item
				env.RegisterActivityWithOptions(func(_ context.Context, in aiContextRequest) (aiContextReceipt, error) {
					order = append(order, step.name)
					if in.SourceVersionID != "registered" || in.HistoricalSourceVersionID != "historical" || in.SourceSHA256 != strings.Repeat("a", 64) {
						t.Fatal("registered immutable lineage lost")
					}
					if step.stage != "decoded" && in.DecodedRef != "file:///unit/decoded.json" {
						t.Fatal("missing completed decode predecessor")
					}
					if step.stage == "work_products" && in.PreparedRef != "" {
						t.Fatal("work products blocked behind chunker")
					}
					if (step.stage == "published" || step.stage == "verified") && in.CandidatesRef != "" {
						t.Fatal("index depends on model candidate bundle")
					}
					if step.stage == "candidates" && failCandidate {
						return aiContextReceipt{}, temporal.NewNonRetryableApplicationError("synthetic model failure", "fixture", nil)
					}
					return aiContextReceipt{ContractVersion: "ai-context-v1", Stage: step.stage, BundleRef: "file:///unit/" + step.stage + ".json", Status: "complete", Records: 2, Conversations: 1, Chunks: 1, WorkProducts: 1, Candidates: 1, ObjectsWritten: 1, ObjectsVerified: 1, ModelID: "actual-fixture-model", ModelCalls: 1, GroundedCandidates: 1}, nil
				}, activity.RegisterOptions{Name: step.name})
			}
			env.RegisterActivityWithOptions(func(_ context.Context, in ContextReviewRequest) (ContextReviewResult, error) {
				order = append(order, "stage_ai_candidate_bundle_activity")
				return ContextReviewResult{Source: &ContextVerifiedReviewSource{ContextReviewSource: in.Source}, Candidates: 1, Staged: 1, HeldExternal: fixture.held}, nil
			}, activity.RegisterOptions{Name: "stage_ai_candidate_bundle_activity"})
			env.RegisterActivityWithOptions(func(_ context.Context, in ContextPackageRequest) (ContextPackageResult, error) {
				order = append(order, "retain_ai_context_package_activity")
				if in.DecodedRef == "" || in.VerificationRef == "" {
					t.Fatal("retention omitted successful upstream result")
				}
				return ContextPackageResult{Complete: true, ManifestRef: "b2://fixture/package", ManifestSHA256: "manifest", CatalogStatus: "complete", CatalogReceiptRef: "catalog", CatalogReceiptSHA256: "cataloghash"}, nil
			}, activity.RegisterOptions{Name: "retain_ai_context_package_activity"})
			env.RegisterActivityWithOptions(func(_ context.Context, in ContextCatalogRequest) (ContextPackageResult, error) {
				order = append(order, "catalog_ai_context_package_activity")
				return in.Package, nil
			}, activity.RegisterOptions{Name: "catalog_ai_context_package_activity"})
			env.ExecuteWorkflow(contextFirstWorkflow, WorkflowInput{ContextContract: ContextContractVersion, RequestID: "context-source:fixture", ActorSubjectUID: "actor", SourceRef: "b2://fixture/source", ProviderVersionID: "exact-version", SourceKind: "ai_chat", DeclaredFormat: "chatgpt", HistoricalSourceVersionID: "historical", SourceSHA256: strings.Repeat("a", 64)})
			if err := env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			var result WorkflowResult
			if err := env.GetWorkflowResult(&result); err != nil {
				t.Fatal(err)
			}
			prefix := []string{contextRegisterActivityName, contextDecodeActivityName, contextWorkProductsActivityName, contextPrepareActivityName, contextEmbedActivityName, contextPublishActivityName, contextVerifyActivityName, contextCandidatesActivityName}
			if len(order) < len(prefix) || !reflect.DeepEqual(order[:len(prefix)], prefix) {
				t.Fatalf("order=%v", order)
			}
			if result.Context == nil || result.Context.DecodedRef == "" || result.Context.ObjectsVerified != 1 || result.Context.VerificationRef == "" {
				t.Fatalf("lost upstream refs: %+v", result.Context)
			}
			if failCandidate {
				if result.Context.Status != "partial" || result.Context.EnrichmentStatus != "failed" || !strings.Contains(result.Context.Reason, "candidate_extraction_failed") {
					t.Fatal("model failure claimed success")
				}
			} else if fixture.held > 0 {
				if result.Context.Status != "partial" || result.Context.CandidateStage == nil || result.Context.CandidateStage.Held != fixture.held || result.Context.CandidateStage.Status != "partially_staged" {
					t.Fatal("held external candidates claimed complete staging")
				}
			} else if result.Context.Status != "complete" || result.Context.ModelID != "actual-fixture-model" || result.Context.ModelCalls != 1 {
				t.Fatalf("actual model/stage status lost: %+v", result.Context)
			}
		})
	}
}
