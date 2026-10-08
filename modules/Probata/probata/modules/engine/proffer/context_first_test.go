// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package proffer

import (
	"context"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"reflect"
	"testing"
)

// TestContextFirstGeminiUsesExistingSequentialActivities verifies actual workflow composition.
// Inputs: one bounded Gemini request and in-memory Activity receipts. Output: sequence/pin assertions.
// Effects: SDK test environment only; choose to detect phantom tools, import drift and lost budgets.
func TestContextFirstGeminiUsesExistingSequentialActivities(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	limits := ContextResourceBounds{MaxSourceBytes: 32768, MaxRecords: 8, MaxTextBytes: 32768, MaxChunks: 8, MaxModelCalls: 8}
	var order []string
	register := func(context.Context, ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
		return ContextSourceRegistrationResult{}, nil
	}
	env.RegisterActivityWithOptions(register, activity.RegisterOptions{Name: contextRegisterActivityName})
	env.OnActivity(contextRegisterActivityName, mock.Anything, mock.Anything).Return(func(_ context.Context, in ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
		order = append(order, contextRegisterActivityName)
		if in.SourceRef != "file:///private/original.md" || in.DeclaredFormat != "gemini_markdown" {
			t.Fatal("registration lost exact source coordinates")
		}
		return ContextSourceRegistrationResult{SourceVersionRef: "version", ReceiptRef: "receipt", Status: "registered"}, nil
	})
	steps := []struct{ name, stage string }{
		{contextPrepareActivityName, "prepared"}, {contextWorkProductsActivityName, "work_products"},
		{contextCandidatesActivityName, "candidates"}, {contextEmbedActivityName, "embedded"},
		{contextPublishActivityName, "published"}, {contextVerifyActivityName, "verified"},
	}
	for _, item := range steps {
		step := item
		fn := func(context.Context, aiContextRequest) (aiContextReceipt, error) { return aiContextReceipt{}, nil }
		env.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: step.name})
		env.OnActivity(step.name, mock.Anything, mock.Anything).Return(func(_ context.Context, in aiContextRequest) (aiContextReceipt, error) {
			order = append(order, step.name)
			if in.ContextResourceBounds != limits || in.SourceFormat != "gemini_markdown" {
				t.Fatal("Activity lost exact format/resource bounds")
			}
			if step.stage != "prepared" && in.PreparedRef != "file:///private/prepared.json" {
				t.Fatal("Activity lacks prepared predecessor")
			}
			if step.stage != "prepared" && step.stage != "work_products" && in.WorkProductsRef != "file:///private/work_products.json" {
				t.Fatal("Activity lacks work-product predecessor")
			}
			return aiContextReceipt{ContractVersion: "ai-context-v1", Stage: step.stage, BundleRef: "file:///private/" + step.stage + ".json", Status: "complete", Records: 8, Conversations: 1, Chunks: 2, WorkProducts: 1, Candidates: 1, ObjectsWritten: 2, ObjectsVerified: 2}, nil
		})
	}
	env.ExecuteWorkflow(contextFirstWorkflow, WorkflowInput{ContextContract: ContextContractVersion, RequestID: ContextWorkflowIDPrefix + "bounded", ActorSubjectUID: "owner", SourceRef: "file:///private/original.md", SourceKind: "ai_chat", DeclaredFormat: "gemini_markdown", ContextResourceBounds: limits})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result WorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	want := []string{contextRegisterActivityName, contextPrepareActivityName, contextWorkProductsActivityName, contextCandidatesActivityName, contextEmbedActivityName, contextPublishActivityName, contextVerifyActivityName}
	if !reflect.DeepEqual(order, want) || result.Context == nil || result.Context.Status != "complete" || result.Context.Records != 8 {
		t.Fatalf("unexpected context completion/sequence: %+v %v", result.Context, order)
	}
}

// TestContextBoundsRejectExcessWithoutTruncation checks requested and reported coverage limits.
// Inputs: over-limit counts and omitted model limit. Output: validation/default assertions.
// Effects: none; choose to keep bounded runs honest rather than trimming source records.
func TestContextBoundsRejectExcessWithoutTruncation(t *testing.T) {
	if (ContextResourceBounds{}).defaults().MaxModelCalls != 8 {
		t.Fatal("default model budget must stay bounded")
	}
	if (ContextResourceBounds{MaxRecords: 1025}).Validate() == nil || (ContextResourceBounds{MaxModelCalls: -1}).Validate() == nil {
		t.Fatal("invalid resource request accepted")
	}
	receipt := aiContextReceipt{ContractVersion: "ai-context-v1", Stage: "prepared", BundleRef: "file:///private/prepared.json", Records: 9, Chunks: 1}
	if receipt.validate("prepared", ContextResourceBounds{MaxRecords: 8, MaxChunks: 8}) == nil {
		t.Fatal("over-bound receipt accepted as complete coverage")
	}
}
