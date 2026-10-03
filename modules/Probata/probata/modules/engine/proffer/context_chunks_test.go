package proffer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// The conversation-chunk path (context_chunks.go): chunks are cut from the run's normalized generation and published
// BEFORE the preview and the owner's decision (everything goes to Weaviate first), one entry per call-log file,
// nothing per message in the Weaviate-first stage, a bad plan or a failed Activity stops the run before the preview.
// The Python Activities are mocked by name, as the stage Activities are.

func emptyChunkThreadsActivity(context.Context, ChunkThreadsRequest) (ChunkThreadsResult, error) {
	return ChunkThreadsResult{}, nil
}

func emptyPublishChunksActivity(context.Context, PublishChunksRequest) (PublishChunksResult, error) {
	return PublishChunksResult{}, nil
}

func emptyPublishCallFilesActivity(context.Context, PublishCallLogFilesRequest) (PublishCallLogFilesResult, error) {
	return PublishCallLogFilesResult{}, nil
}

func registerContextChunkActivities(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(emptyChunkThreadsActivity, activity.RegisterOptions{Name: ChunkContextThreadsActivityName})
	env.RegisterActivityWithOptions(emptyPublishChunksActivity, activity.RegisterOptions{Name: PublishContextChunksActivityName})
	env.RegisterActivityWithOptions(emptyPublishCallFilesActivity, activity.RegisterOptions{Name: PublishCallLogFilesActivityName})
}

func chunkPlan(thread string, messages int) ChunkThreadPlan {
	return ChunkThreadPlan{
		Corpus: "first_party", ThreadID: thread, MessageCount: messages, Digest: "digest-" + thread,
		Chunker: "neural_distilbert", ChunkerVersion: "neural_distilbert|mirth|overlap=2", Overlap: 2,
		Spans: [][]int{{0, 4}, {3, messages - 1}},
	}
}

func TestChunkPlanValidationFailsClosed(t *testing.T) {
	good := chunkPlan("t1", 10)
	if err := good.validate(); err != nil {
		t.Fatalf("a good plan was refused: %v", err)
	}
	cases := map[string]func(*ChunkThreadPlan){
		"no thread id":            func(p *ChunkThreadPlan) { p.ThreadID = "" },
		"no digest":               func(p *ChunkThreadPlan) { p.Digest = " " },
		"no messages":             func(p *ChunkThreadPlan) { p.MessageCount = 0 },
		"no spans":                func(p *ChunkThreadPlan) { p.Spans = nil },
		"span past the end":       func(p *ChunkThreadPlan) { p.Spans = [][]int{{0, 4}, {3, 10}} },
		"span reversed":           func(p *ChunkThreadPlan) { p.Spans = [][]int{{0, 4}, {9, 5}} },
		"span of the wrong shape": func(p *ChunkThreadPlan) { p.Spans = [][]int{{0, 4, 5}} },
		"spans out of order":      func(p *ChunkThreadPlan) { p.Spans = [][]int{{3, 9}, {0, 4}} },
		"a message in no chunk":   func(p *ChunkThreadPlan) { p.Spans = [][]int{{0, 3}, {5, 9}} },
		"negative overlap":        func(p *ChunkThreadPlan) { p.Overlap = -1 },
	}
	for name, mutate := range cases {
		p := chunkPlan("t1", 10)
		mutate(&p)
		if err := p.validate(); err == nil {
			t.Errorf("%s: plan accepted", name)
		}
	}
}

// chunkRun records what the mocked Python Activities were asked.
type chunkRun struct {
	mu        sync.Mutex
	chunkReqs []ChunkThreadsRequest
	published []PublishChunksRequest
	callReqs  []PublishCallLogFilesRequest
}

func mockChunkActivities(env *testsuite.TestWorkflowEnvironment, plans []ChunkThreadPlan, calls PublishCallLogFilesResult) *chunkRun {
	run := &chunkRun{}
	env.OnActivity(ChunkContextThreadsActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req ChunkThreadsRequest) (ChunkThreadsResult, error) {
			run.mu.Lock()
			defer run.mu.Unlock()
			run.chunkReqs = append(run.chunkReqs, req)
			return ChunkThreadsResult{Chunker: "neural_distilbert", ChunkerVersion: "v", Overlap: 2, Threads: plans}, nil
		}).Once()
	env.OnActivity(PublishContextChunksActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req PublishChunksRequest) (PublishChunksResult, error) {
			run.mu.Lock()
			defer run.mu.Unlock()
			run.published = append(run.published, req)
			return PublishChunksResult{Corpus: req.Plan.Corpus, ThreadID: req.Plan.ThreadID, ChunksWritten: len(req.Plan.Spans), EmbedRequests: 1}, nil
		})
	env.OnActivity(PublishCallLogFilesActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req PublishCallLogFilesRequest) (PublishCallLogFilesResult, error) {
			run.mu.Lock()
			defer run.mu.Unlock()
			run.callReqs = append(run.callReqs, req)
			return calls, nil
		}).Maybe()
	return run
}

// firstPartyChunkEnv is a first-party run whose first-party context stages succeed.
func firstPartyChunkEnv(t *testing.T) (*testsuite.TestWorkflowEnvironment, map[stagegraph.StageID]*StageRequest) {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	mockAllStagesSucceed(env)
	requests := map[stagegraph.StageID]*StageRequest{}
	env.OnActivity(string(stagegraph.PublishContextSearch), mock.Anything, mock.Anything).Return(
		func(_ context.Context, req StageRequest) (StageResult, error) {
			copied := req
			requests[stagegraph.PublishContextSearch] = &copied
			return stageStub(stagegraph.PublishContextSearch), nil
		}).Once()
	mockFirstPartyContextSucceeds(env, requests)
	env.OnActivity(string(stagegraph.CommitCallLog), mock.Anything, mock.Anything).
		Return(stageStub(stagegraph.CommitCallLog), nil).Once()
	return env, requests
}

// TestChunksArePublishedBeforeThePreviewAndTheDecision: the order, the references each Activity gets (the run's
// normalized generation and participant resolution), and that the Weaviate-first stage is told to leave messages and
// calls to the chunk path.
func TestChunksArePublishedBeforeThePreviewAndTheDecision(t *testing.T) {
	env, requests := firstPartyChunkEnv(t)
	plans := []ChunkThreadPlan{chunkPlan("t1", 10), chunkPlan("t2", 8), chunkPlan("t3", 6), chunkPlan("t4", 12), chunkPlan("t5", 9)}
	run := mockChunkActivities(env, plans, PublishCallLogFilesResult{Files: 1, Calls: 40, EmbedRequests: 1})
	order := newOrderRecorder(env)
	approveHold(env)

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error = %v", err)
	}
	at := func(name string) int { return order.indexOf(name) }
	sequence := []string{
		string(stagegraph.ResolveContextParticipants), string(stagegraph.PublishContextSearch), string(stagegraph.ProposeFirstPartyContext),
		ChunkContextThreadsActivityName, PublishContextChunksActivityName, PublishCallLogFilesActivityName,
		string(stagegraph.PublishPreview), string(stagegraph.ConfirmFirstPartyContext), string(stagegraph.CommitFirstPartyContextThreads),
		string(stagegraph.SealGeneration), string(stagegraph.PublishGeneration),
	}
	for i := 1; i < len(sequence); i++ {
		if at(sequence[i-1]) < 0 || at(sequence[i]) < 0 || at(sequence[i-1]) > at(sequence[i]) {
			t.Fatalf("%s must run before %s; order = %v", sequence[i-1], sequence[i], order.snapshot())
		}
	}
	if search := requests[stagegraph.PublishContextSearch]; search == nil || search.Refs[SkipRecordKindsRefKey] != "message,call" {
		t.Fatalf("publish_context_search request = %+v, want skip_record_kinds=message,call", search)
	}
	generation := string(stageStub(stagegraph.PersistNormalizedGeneration).Ref)
	resolution := string(stageStub(stagegraph.ResolveContextParticipants).Ref)
	if len(run.chunkReqs) != 1 || run.chunkReqs[0].SourceVersionID != string(stageStub(stagegraph.RegisterSource).Ref) ||
		run.chunkReqs[0].NormalizedGenerationID != generation || run.chunkReqs[0].ParticipantResolutionID != resolution {
		t.Fatalf("chunk requests = %+v, want one over the run's source version, normalized generation and participant resolution", run.chunkReqs)
	}
	if len(run.published) != len(plans) {
		t.Fatalf("published %d threads, want %d (more than one batch of %d)", len(run.published), len(plans), maxConcurrentChunkPublishes)
	}
	seen := map[string]bool{}
	for _, req := range run.published {
		seen[req.Plan.ThreadID] = true
		if req.Plan.Digest != "digest-"+req.Plan.ThreadID || len(req.Plan.Spans) != 2 ||
			req.NormalizedGenerationID != generation || req.ParticipantResolutionID != resolution {
			t.Errorf("publish request carries a changed plan or lost its references: %+v", req)
		}
	}
	if len(seen) != len(plans) {
		t.Errorf("published threads %v, want all %d", seen, len(plans))
	}
	if len(run.callReqs) != 1 || run.callReqs[0].NormalizedGenerationID != generation {
		t.Fatalf("call-log requests = %+v, want one over the same generation", run.callReqs)
	}
}

// TestChunksStayWhenTheOwnerRejectsThePreview: a rejected run keeps the chunks it published, exactly as it keeps its
// other search objects today (they carry ingest_run_id and the generation id); nothing is committed.
func TestChunksStayWhenTheOwnerRejectsThePreview(t *testing.T) {
	env, _ := firstPartyChunkEnv(t)
	run := mockChunkActivities(env, []ChunkThreadPlan{chunkPlan("t1", 10)}, PublishCallLogFilesResult{})
	order := newOrderRecorder(env)
	rejectHold(env, "wrong thread")

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	if env.GetWorkflowError() == nil {
		t.Fatal("a rejected preview completed without error")
	}
	if len(run.published) != 1 {
		t.Errorf("published %d threads before the rejection, want 1", len(run.published))
	}
	for _, later := range []stagegraph.StageID{stagegraph.ConfirmFirstPartyContext, stagegraph.CommitFirstPartyMessages, stagegraph.SealGeneration} {
		if order.contains(string(later)) {
			t.Errorf("%s ran after the owner rejected the preview", later)
		}
	}
}

// TestChunkStepRunsOnARunWithNoMessagesToo: a calls-only generation has no message to propose, but its call-log file
// still goes out; the Activity answers with nothing when the generation holds nothing.
func TestChunkStepRunsOnARunWithNoMessagesToo(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	mockAllStagesSucceed(env)
	order := newOrderRecorder(env)
	approveHold(env)

	env.ExecuteWorkflow(ProfferWorkflow, testInput())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error = %v", err)
	}
	if !order.contains(ChunkContextThreadsActivityName) || !order.contains(PublishCallLogFilesActivityName) {
		t.Errorf("the chunk step did not run: %v", order.snapshot())
	}
	if order.contains(PublishContextChunksActivityName) {
		t.Errorf("a thread was published for a generation with no message")
	}
	if order.indexOf(PublishCallLogFilesActivityName) > order.indexOf(string(stagegraph.PublishPreview)) {
		t.Errorf("the chunk step ran after the preview: %v", order.snapshot())
	}
}

func TestAnUnusablePlanStopsTheRunBeforeAnythingIsEmbeddedOrPreviewed(t *testing.T) {
	env, _ := firstPartyChunkEnv(t)
	bad := chunkPlan("t1", 10)
	bad.Spans = [][]int{{0, 3}, {5, 9}} // message 4 is in no chunk
	mockChunkActivities(env, []ChunkThreadPlan{bad}, PublishCallLogFilesResult{})
	order := newOrderRecorder(env)
	approveHold(env)

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	err := env.GetWorkflowError()
	if err == nil || !strings.Contains(err.Error(), "leaves message 4 in no chunk") {
		t.Fatalf("workflow error = %v, want the plan's own defect", err)
	}
	for _, later := range []string{PublishContextChunksActivityName, PublishCallLogFilesActivityName, string(stagegraph.PublishPreview), string(stagegraph.SealGeneration)} {
		if order.contains(later) {
			t.Errorf("%s ran after an unusable chunk plan", later)
		}
	}
}

func TestAChunkFailureStopsTheRunBeforeThePreviewWithTheActivitysReason(t *testing.T) {
	env, _ := firstPartyChunkEnv(t)
	env.OnActivity(ChunkContextThreadsActivityName, mock.Anything, mock.Anything).
		Return(ChunkThreadsResult{}, temporal.NewNonRetryableApplicationError("chunker model missing", "chunk", nil)).Once()
	order := newOrderRecorder(env)
	approveHold(env)

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	err := env.GetWorkflowError()
	if err == nil || !strings.Contains(err.Error(), "chunker model missing") {
		t.Fatalf("workflow error = %v, want the chunk Activity's reason", err)
	}
	if order.contains(string(stagegraph.PublishPreview)) || order.contains(PublishContextChunksActivityName) {
		t.Errorf("the run went on after the chunk failure: %v", order.snapshot())
	}
}

func TestAPublishFailureNamesTheThreadAndStopsBeforeThePreview(t *testing.T) {
	env, _ := firstPartyChunkEnv(t)
	env.OnActivity(ChunkContextThreadsActivityName, mock.Anything, mock.Anything).
		Return(ChunkThreadsResult{Threads: []ChunkThreadPlan{chunkPlan("t1", 10), chunkPlan("t2", 10)}}, nil).Once()
	env.OnActivity(PublishContextChunksActivityName, mock.Anything, mock.MatchedBy(func(req PublishChunksRequest) bool { return req.Plan.ThreadID == "t1" })).
		Return(PublishChunksResult{ChunksWritten: 2}, nil)
	env.OnActivity(PublishContextChunksActivityName, mock.Anything, mock.MatchedBy(func(req PublishChunksRequest) bool { return req.Plan.ThreadID == "t2" })).
		Return(PublishChunksResult{}, temporal.NewNonRetryableApplicationError("NIM refused the batch", "embed", errors.New("x")))
	order := newOrderRecorder(env)
	approveHold(env)

	env.ExecuteWorkflow(ProfferWorkflow, firstPartyInput())
	err := env.GetWorkflowError()
	if err == nil || !strings.Contains(err.Error(), "thread t2") || !strings.Contains(err.Error(), "NIM refused the batch") {
		t.Fatalf("workflow error = %v, want the failing thread and the Activity's reason", err)
	}
	if order.contains(string(stagegraph.PublishPreview)) {
		t.Errorf("the preview was published after a chunk publish failed")
	}
}
