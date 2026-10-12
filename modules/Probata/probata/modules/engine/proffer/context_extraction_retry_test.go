package proffer

import (
	"context"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// TestNativeExtractionRetryPreservesSourceAndPredecessors verifies additive attempts use the existing extraction/staging seam.
// Inputs: synthetic registered source and predecessor refs. Output: exact request and partial-result assertions.
// Effects: Temporal SDK test environment only. Pick to catch fake generations, reingestion or overwritten owner review.
func TestNativeExtractionRetryPreservesSourceAndPredecessors(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()
		in := ContextExtractionRetryInput{ParentWorkflowID: "context-source:parent", ActorSubjectUID: "actor",
			Source:      ContextReviewSource{SourceVersionID: "registered-version", SourceRef: "b2://synthetic/native.json"},
			PreparedRef: "file:///synthetic/prepared.json", WorkProductsRef: "file:///synthetic/work_products.json"}
		staged := false
		env.RegisterActivityWithOptions(func(_ context.Context, req aiContextRequest) (aiContextReceipt, error) {
			if req.SourceRef != in.Source.SourceRef || req.PreparedRef != in.PreparedRef || req.WorkProductsRef != in.WorkProductsRef || req.RequestID != in.ParentWorkflowID || req.TemporalRunID == "" {
				t.Fatalf("retry lost original source/predecessors or real attempt: %+v", req)
			}
			if fail {
				return aiContextReceipt{}, temporal.NewNonRetryableApplicationError("provider unavailable", "provider", nil)
			}
			return aiContextReceipt{ContractVersion: "ai-context-v1", Stage: "candidates", Status: "partial_enrichment", BundleRef: "file:///new-attempt/candidates.json", Candidates: 1}, nil
		}, activity.RegisterOptions{Name: contextCandidatesActivityName})
		env.RegisterActivityWithOptions(func(_ context.Context, req ContextReviewRequest) (ContextReviewResult, error) {
			staged = true
			if req.Source != in.Source || req.PreparedRef != in.PreparedRef || req.RequestID != in.ParentWorkflowID || req.BundleRef != "file:///new-attempt/candidates.json" {
				t.Fatalf("staging lost native pins: %+v", req)
			}
			return ContextReviewResult{Candidates: 1, Staged: 1, Source: &ContextVerifiedReviewSource{ContextReviewSource: in.Source, PreparedRef: in.PreparedRef}}, nil
		}, activity.RegisterOptions{Name: "stage_ai_candidate_bundle_activity"})
		env.ExecuteWorkflow(ContextExtractionRetryWorkflow, in)
		var result WorkflowResult
		if err := env.GetWorkflowResult(&result); err != nil {
			t.Fatal(err)
		}
		if result.Context == nil || result.Context.SourceVersionRef != in.Source.SourceVersionID || result.Context.PreparedRef != in.PreparedRef {
			t.Fatalf("lost source result: %+v", result)
		}
		if fail {
			if staged || result.Context.Status != "failed" {
				t.Fatalf("failed retry staged or succeeded: %+v", result.Context)
			}
		} else if !staged || result.Context.Status != "partial_enrichment" {
			t.Fatalf("partial extraction was reported complete: %+v", result.Context)
		}
	}
}
