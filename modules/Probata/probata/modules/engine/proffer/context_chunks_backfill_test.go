// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
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
)

// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// The re-chunk and the per-message removal as workflows over Python Activities (context_chunks_backfill.go).

func anyRequestActivity(context.Context, map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("placeholder ran unmocked")
}

func listPlaceholder(context.Context, map[string]interface{}) (ListContextThreadsResult, error) {
	return ListContextThreadsResult{}, errors.New("placeholder ran unmocked")
}

func backfillEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ConversationChunksBackfillWorkflow)
	env.RegisterWorkflow(ConversationChunksRemovalWorkflow)
	env.RegisterActivityWithOptions(listPlaceholder, activity.RegisterOptions{Name: ListContextThreadsActivityName})
	for _, name := range []string{EstimateContextChunksActivityName, VerifyChunkCoverageActivityName, RemovePerMessageObjectsActivityName} {
		env.RegisterActivityWithOptions(anyRequestActivity, activity.RegisterOptions{Name: name})
	}
	env.RegisterActivityWithOptions(emptyChunkThreadsActivity, activity.RegisterOptions{Name: ChunkContextThreadsActivityName})
	env.RegisterActivityWithOptions(emptyPublishChunksActivity, activity.RegisterOptions{Name: PublishContextChunksActivityName})
	env.RegisterActivityWithOptions(emptyPublishCallFilesActivity, activity.RegisterOptions{Name: PublishCallLogFilesActivityName})
	return env
}

func TestBackfillDryRunOnlyEstimatesAndDoesNotTouchAThread(t *testing.T) {
	env := backfillEnv(t)
	var seen map[string]interface{}
	env.OnActivity(EstimateContextChunksActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req map[string]interface{}) (map[string]interface{}, error) {
			seen = req
			return map[string]interface{}{"threads": map[string]interface{}{"total": 201}, "chunks": 47000, "embed_calls": 1700}, nil
		}).Once()
	order := newOrderRecorder(env)

	env.ExecuteWorkflow(ConversationChunksBackfillWorkflow, ConversationChunksBackfillInput{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "r1", DryRun: true, Exact: true})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error = %v", err)
	}
	var result ConversationChunksBackfillResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || result.Estimate["chunks"] != float64(47000) || seen["exact"] != true || seen["sample"] != float64(6) {
		t.Fatalf("result %+v, request %v; want the estimate returned and the exact flag and a default sample passed", result, seen)
	}
	for _, name := range []string{ListContextThreadsActivityName, ChunkContextThreadsActivityName, PublishContextChunksActivityName, PublishCallLogFilesActivityName} {
		if order.contains(name) {
			t.Errorf("%s ran in a dry run", name)
		}
	}
}

func TestBackfillChunksAndPublishesEachThreadAsItsOwnActivitiesThenTheCallFiles(t *testing.T) {
	env := backfillEnv(t)
	threads := []ContextThreadRef{{"first_party", "t1", 10}, {"first_party", "t2", 8}, {"acquired_third_party", "t3", 6}, {"first_party", "t4", 12}}
	env.OnActivity(ListContextThreadsActivityName, mock.Anything, mock.Anything).Return(
		ListContextThreadsResult{Threads: threads, CallSourceVersions: []string{"sv-a", "sv-b"}}, nil).Once()
	var mu sync.Mutex
	var chunkReqs []ChunkThreadsRequest
	var published []string
	var calls []string
	env.OnActivity(ChunkContextThreadsActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req ChunkThreadsRequest) (ChunkThreadsResult, error) {
			mu.Lock()
			defer mu.Unlock()
			chunkReqs = append(chunkReqs, req)
			return ChunkThreadsResult{Threads: []ChunkThreadPlan{chunkPlan(req.ThreadRefs[0].ThreadID, 10)}}, nil
		})
	env.OnActivity(PublishContextChunksActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req PublishChunksRequest) (PublishChunksResult, error) {
			mu.Lock()
			defer mu.Unlock()
			published = append(published, req.Plan.ThreadID)
			return PublishChunksResult{ChunksWritten: 2, StaleDeleted: 1, EmbedRequests: 1}, nil
		})
	env.OnActivity(PublishCallLogFilesActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req PublishCallLogFilesRequest) (PublishCallLogFilesResult, error) {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, req.SourceVersionID)
			if req.NormalizedGenerationID != "" {
				t.Errorf("a back-fill call-log request carries a generation: %+v", req)
			}
			return PublishCallLogFilesResult{Files: 1, Calls: 5, EmbedRequests: 1}, nil
		})

	env.ExecuteWorkflow(ConversationChunksBackfillWorkflow, ConversationChunksBackfillInput{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "r2"})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error = %v", err)
	}
	var result ConversationChunksBackfillResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if len(chunkReqs) != 4 || len(published) != 4 || len(calls) != 2 {
		t.Fatalf("chunk %d, publish %d, call files %d; want 4, 4, 2", len(chunkReqs), len(published), len(calls))
	}
	for _, req := range chunkReqs {
		if len(req.ThreadRefs) != 1 || req.NormalizedGenerationID != "" || req.SourceVersionID != "" {
			t.Errorf("a back-fill chunk request must name one committed thread and no generation: %+v", req)
		}
	}
	p := result.Progress
	if p.Stage != "done" || p.ThreadsTotal != 4 || p.ThreadsDone != 4 || p.ChunksWritten != 8 || p.CallFilesDone != 2 || result.StaleDeleted != 4 || result.EmbedRequests != 6 {
		t.Fatalf("receipt = %+v", result)
	}
}

func TestBackfillNamesTheThreadsThatFailedAndFailsTheRun(t *testing.T) {
	env := backfillEnv(t)
	env.OnActivity(ListContextThreadsActivityName, mock.Anything, mock.Anything).Return(
		ListContextThreadsResult{Threads: []ContextThreadRef{{"first_party", "ok", 10}, {"first_party", "bad", 10}}}, nil).Once()
	env.OnActivity(ChunkContextThreadsActivityName, mock.Anything, mock.Anything).Return(
		func(_ context.Context, req ChunkThreadsRequest) (ChunkThreadsResult, error) {
			return ChunkThreadsResult{Threads: []ChunkThreadPlan{chunkPlan(req.ThreadRefs[0].ThreadID, 10)}}, nil
		})
	env.OnActivity(PublishContextChunksActivityName, mock.Anything, mock.MatchedBy(func(req PublishChunksRequest) bool { return req.Plan.ThreadID == "ok" })).
		Return(PublishChunksResult{ChunksWritten: 2}, nil)
	env.OnActivity(PublishContextChunksActivityName, mock.Anything, mock.MatchedBy(func(req PublishChunksRequest) bool { return req.Plan.ThreadID == "bad" })).
		Return(PublishChunksResult{}, temporal.NewNonRetryableApplicationError("NIM refused", "embed", nil))

	env.ExecuteWorkflow(ConversationChunksBackfillWorkflow, ConversationChunksBackfillInput{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "r3", NoCalls: true})
	err := env.GetWorkflowError()
	if err == nil || !strings.Contains(err.Error(), "1 failure") || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("workflow error = %v, want the failed thread named", err)
	}
	var result ConversationChunksBackfillResult
	_ = env.GetWorkflowResult(&result)
}

func TestRemovalVerifiesFirstAndRefusesWhenTheChunksDoNotCover(t *testing.T) {
	env := backfillEnv(t)
	env.OnActivity(VerifyChunkCoverageActivityName, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"verified": false, "uncovered_pg_messages": 3}, nil).Once()
	order := newOrderRecorder(env)

	env.ExecuteWorkflow(ConversationChunksRemovalWorkflow, ConversationChunksRemovalInput{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "d1"})
	err := env.GetWorkflowError()
	if err == nil || !strings.Contains(err.Error(), "nothing was deleted") {
		t.Fatalf("workflow error = %v, want a refusal", err)
	}
	if order.contains(RemovePerMessageObjectsActivityName) {
		t.Error("the removal ran on an unverified collection")
	}
}

func TestRemovalDryRunAndExecuteSendTheModeToTheActivity(t *testing.T) {
	for _, dry := range []bool{true, false} {
		env := backfillEnv(t)
		env.OnActivity(VerifyChunkCoverageActivityName, mock.Anything, mock.Anything).
			Return(map[string]interface{}{"verified": true}, nil).Once()
		var seen map[string]interface{}
		env.OnActivity(RemovePerMessageObjectsActivityName, mock.Anything, mock.Anything).Return(
			func(_ context.Context, req map[string]interface{}) (map[string]interface{}, error) {
				seen = req
				return map[string]interface{}{"deleted": map[string]interface{}{"message": 100, "call": 10}}, nil
			}).Once()
		env.ExecuteWorkflow(ConversationChunksRemovalWorkflow, ConversationChunksRemovalInput{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "d2", DryRun: dry})
		if err := env.GetWorkflowError(); err != nil {
			t.Fatalf("dry=%v: workflow error = %v", dry, err)
		}
		var result ConversationChunksRemovalResult
		if err := env.GetWorkflowResult(&result); err != nil {
			t.Fatal(err)
		}
		if seen["dry_run"] != dry || result.DryRun != dry || result.Verification["verified"] != true || result.Removal["deleted"] == nil {
			t.Fatalf("dry=%v: request %v, result %+v", dry, seen, result)
		}
	}
}

func TestOnlyCoveredLetsAnUnverifiedRemovalGoOn(t *testing.T) {
	env := backfillEnv(t)
	env.OnActivity(VerifyChunkCoverageActivityName, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"verified": false}, nil).Once()
	env.OnActivity(RemovePerMessageObjectsActivityName, mock.Anything, mock.Anything).
		Return(map[string]interface{}{"kept_uncovered": map[string]interface{}{"message": 4}}, nil).Once()
	env.ExecuteWorkflow(ConversationChunksRemovalWorkflow, ConversationChunksRemovalInput{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", RequestID: "d3", OnlyCovered: true})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error = %v", err)
	}
}
