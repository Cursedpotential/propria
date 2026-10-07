// Byline: Codex · GPT-6 · 2026-10-06.
package proffer

import (
	"context"
	"fmt"
	"testing"

	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// registerAIContentMocks models compact stage receipts, with optional provenance corruption.
// Inputs: a Temporal test environment and stage to corrupt. Outputs: registered activity bodies.
// Effects: test history only. Choose to validate ordering and fail-closed provenance without real case data.
func registerAIContentMocks(env *testsuite.TestWorkflowEnvironment, badStage string) {
	for _, step := range []struct{name,stage string}{
		{AIPrepareContentActivityName,"prepared"},{AIExtractCandidatesActivityName,"candidates"},
		{AIEmbedContentActivityName,"embedded"},{AIPublishContentActivityName,"published"},
		{AIVerifyContentPublicationActivityName,"verified"},
	} {
		stage:=step.stage
		env.RegisterActivityWithOptions(func(_ context.Context, req AIContentRequest)(AIContentResult,error){
			if stage!="prepared" && req.PreparedRef=="" { return AIContentResult{},fmt.Errorf("missing prepared reference") }
			if (stage=="published" || stage=="verified") && (req.CandidatesRef=="" || req.EmbeddingsRef=="") { return AIContentResult{},fmt.Errorf("missing extraction/embed references") }
			if stage=="verified" && req.PublicationRef=="" { return AIContentResult{},fmt.Errorf("missing publication reference") }
			out:=AIContentResult{Stage:stage,BundleRef:Ref("file:///retained/"+stage+".json"),SourceVersionID:req.SourceVersionID,NormalizedGenerationID:req.NormalizedGenerationID,VerificationID:req.VerificationID,Conversations:8,Records:132,Chunks:24,Candidates:7,ObjectsWritten:24,ObjectsVerified:24}
			if stage==badStage { out.NormalizedGenerationID="wrong-generation" }
			return out,nil
		},activity.RegisterOptions{Name:step.name})
	}
}

// TestAIContentPathReplacesPerMessageSearchAndRejectsDrift validates routing and independent stage boundaries.
// Inputs: verified AI applicability, legacy replay marker, and malformed stage receipts. Outputs: assertions.
// Effects: mocked Temporal history only. Choose as regression protection against AI per-message publication.
func TestAIContentPathReplacesPerMessageSearchAndRejectsDrift(t *testing.T) {
	for _,bad:=range []string{"","prepared","candidates","embedded","published","verified","legacy"} {
		t.Run("stage-"+bad,func(t *testing.T){
			var suite testsuite.WorkflowTestSuite
			env:=suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(ProfferWorkflow)
			mockAllStagesSucceed(env)
			registerAIContentMocks(env,bad)
			env.OnActivity(string(stagegraph.ResolveContextParticipants),mock.Anything,mock.Anything).Return(StageResult{Stage:stagegraph.ResolveContextParticipants,Status:StatusNotApplicable,ReceiptRef:"verified-ai-receipt",AIChatSource:true},nil).Once()
			if bad=="legacy" {env.OnGetVersion(aiContentChangeID,workflow.DefaultVersion,aiContentVersion).Return(workflow.DefaultVersion).Once()}
			order:=newOrderRecorder(env)
			approveHold(env)
			env.ExecuteWorkflow(ProfferWorkflow,testInput())
			if bad=="legacy" {
				if env.GetWorkflowError()!=nil || !order.contains(string(stagegraph.PublishContextSearch)) || order.contains(AIPrepareContentActivityName) {t.Fatalf("legacy command history changed: %v error=%v",order.snapshot(),env.GetWorkflowError())}
				return
			}
			if order.contains(string(stagegraph.PublishContextSearch)) || order.contains(ChunkContextThreadsActivityName) || order.contains(string(stagegraph.CommitFirstPartyMessages)) {t.Fatalf("AI entered message pipeline: %v",order.snapshot())}
			if bad!="" {
				if env.GetWorkflowError()==nil || order.contains(string(stagegraph.PublishPreview)) || order.contains(string(stagegraph.PublishGeneration)) {t.Fatalf("bad %s receipt did not halt: %v",bad,order.snapshot())}
				return
			}
			if err:=env.GetWorkflowError();err!=nil{t.Fatal(err)}
			prior:=-1
			for _,name:=range []string{AIPrepareContentActivityName,AIExtractCandidatesActivityName,AIEmbedContentActivityName,AIPublishContentActivityName,AIVerifyContentPublicationActivityName,string(stagegraph.PublishPreview)} {
				at:=order.indexOf(name);if at<=prior{t.Fatalf("stage order %v",order.snapshot())};prior=at
			}
			var result WorkflowResult
			if err:=env.GetWorkflowResult(&result);err!=nil{t.Fatal(err)}
			if result.AIContent==nil || result.AIContent.Chunks!=24 || result.AIContent.Candidates!=7 || result.AIContent.VerificationRef==""{t.Fatalf("missing verified content result: %+v",result.AIContent)}
		})
	}
}
