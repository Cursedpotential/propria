// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package proffer

import (
	"strings"
	"testing"

	"go.temporal.io/sdk/testsuite"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// TestCommittedMessagesStartTheDefaultExtractionAsAChild proves the owner's rule that extraction
// is part of the workflow: once the first-party messages are committed, extraction_request_workflow
// starts once, with only the default extractor, the run's own preview handle and generation, and
// the workflow's own actor, after the thread commit and before the seal; the result records it.
func TestCommittedMessagesStartTheDefaultExtractionAsAChild(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	mockAllStagesSucceed(env)
	collected := &autoExtractionInputs{}
	mockFirstPartyContextCollecting(env, nil, collected)
	order := newOrderRecorder(env)
	approveHold(env)

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error = %v", err)
	}
	var result WorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.AutoExtraction, "started: "+flow.AutoExtractionWorkflowIDPrefix) {
		t.Fatalf("AutoExtraction = %q, want the child started", result.AutoExtraction)
	}
	inputs := collected.snapshot()
	if len(inputs) != 1 {
		t.Fatalf("extraction started %d times, want once", len(inputs))
	}
	in := inputs[0]
	if len(in.Extractors) != 1 || in.Extractors[0] != flow.DefaultExtractorID || !in.Auto {
		t.Fatalf("child input = %+v, want only the default extractor, marked automatic", in)
	}
	if len(in.Runs) != 1 || in.Runs[0].GenerationID != string(stageStub(stagegraph.PersistNormalizedGeneration).Ref) || in.Runs[0].PreviewHandle == "" {
		t.Fatalf("child runs = %+v, want this run's generation and preview", in.Runs)
	}
	if in.Actor.Username != "proffer-auto-extraction" {
		t.Fatalf("actor = %+v, want the workflow's own actor", in.Actor)
	}
	if order.indexOf(string(stagegraph.CommitFirstPartyContextThreads)) < 0 || order.indexOf(string(stagegraph.SealGeneration)) < 0 {
		t.Fatalf("the commit or the seal never ran; order = %v", order.snapshot())
	}
}

// TestRejectedRunStartsNoExtraction proves nothing is extracted when the owner rejects the preview.
func TestRejectedRunStartsNoExtraction(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	collected := &autoExtractionInputs{}
	mockAutoExtractionChild(env, collected)
	mockAllStagesSucceed(env)
	rejectHold(env, "wrong thread")

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	if env.GetWorkflowError() == nil {
		t.Fatal("a rejected preview completed without error")
	}
	if got := collected.snapshot(); len(got) != 0 {
		t.Fatalf("extraction started %d time(s) for a rejected run", len(got))
	}
}
