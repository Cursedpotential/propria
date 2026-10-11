// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
package proffer

// The re-chunk of what is already committed, and the removal of the per-message search objects, as Go workflows over
// Python Activities (owner order 2026-10-02: "Make sure all these things get run as Temporal activities and are
// traceable"). Dry-run is a workflow input. Each thread is its own chunk Activity and its own publish Activity; the
// workflow result is the receipt, and a query handler shows progress while it runs. The scripts are thin starters
// (server/context_chunks/start.py).
//
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"go.temporal.io/sdk/workflow"
)

const (
	ConversationChunksBackfillWorkflowName = "proffer_conversation_chunks_backfill_workflow"
	ConversationChunksRemovalWorkflowName  = "proffer_conversation_chunks_removal_workflow"
	// ConversationChunksProgressQueryName answers with a ConversationChunksProgress while a backfill runs.
	ConversationChunksProgressQueryName = "conversation_chunks_progress"

	ListContextThreadsActivityName            = "list_context_threads_activity"
	EstimateContextChunksActivityName         = "estimate_context_chunks_activity"
	VerifyChunkCoverageActivityName           = "verify_chunk_coverage_activity"
	RemovePerMessageObjectsActivityName       = "remove_per_message_objects_activity"
	backfillThreadsInFlight                   = 3
	backfillEstimateTimeout                   = 8 * time.Hour
	backfillRemovalTimeout                    = 4 * time.Hour
	backfillListTimeout                       = 30 * time.Minute
	backfillSingleRemovalAttempts       int32 = 2
)

// ConversationChunksBackfillInput is the workflow input (JSON names match server/context_chunks/start.py).
type ConversationChunksBackfillInput struct {
	OperatingMode string `json:"operating_mode,omitempty"`
	MatterID      string `json:"matter_id,omitempty"`
	CourtCaseID   string `json:"court_case_id,omitempty"`
	RequestID     string `json:"request_id"`
	// DryRun only counts: threads, messages, estimated (or, with Exact, exact) chunks and embed calls.
	DryRun      bool   `json:"dry_run"`
	Exact       bool   `json:"exact"`
	Chunker     string `json:"chunker,omitempty"`
	Overlap     int    `json:"overlap,omitempty"`
	SkipCovered bool   `json:"skip_covered"`
	ThreadLimit int    `json:"thread_limit,omitempty"`
	NoCalls     bool   `json:"no_calls"`
	Sample      int    `json:"sample,omitempty"`
	SampleCap   int    `json:"sample_cap,omitempty"`
}

// ContextThreadRef is one thread to re-chunk.
type ContextThreadRef struct {
	Corpus   string `json:"corpus"`
	ThreadID string `json:"thread_id"`
	Messages int    `json:"messages"`
}

// ListContextThreadsResult is the list Activity's output.
type ListContextThreadsResult struct {
	Threads            []ContextThreadRef `json:"threads"`
	CallSourceVersions []string           `json:"call_source_versions"`
}

// ConversationChunksProgress is the progress query's answer.
type ConversationChunksProgress struct {
	Stage          string `json:"stage"`
	ThreadsTotal   int    `json:"threads_total"`
	ThreadsDone    int    `json:"threads_done"`
	ThreadsFailed  int    `json:"threads_failed"`
	ChunksWritten  int    `json:"chunks_written"`
	CallFilesTotal int    `json:"call_files_total"`
	CallFilesDone  int    `json:"call_files_done"`
}

// ConversationChunksBackfillResult is the workflow's receipt.
type ConversationChunksBackfillResult struct {
	OperatingMode string                     `json:"operating_mode,omitempty"`
	MatterID      string                     `json:"matter_id,omitempty"`
	CourtCaseID   string                     `json:"court_case_id,omitempty"`
	RequestID     string                     `json:"request_id"`
	DryRun        bool                       `json:"dry_run"`
	Estimate      map[string]interface{}     `json:"estimate,omitempty"`
	Progress      ConversationChunksProgress `json:"progress"`
	// Failed names each thread that did not complete, with the Activity's reason; the workflow also fails when any did.
	Failed        []string `json:"failed,omitempty"`
	StaleDeleted  int      `json:"stale_deleted"`
	EmbedRequests int      `json:"embed_requests"`
}

// ConversationChunksBackfillWorkflow is ConversationChunksBackfillWorkflowName.
func ConversationChunksBackfillWorkflow(ctx workflow.Context, in ConversationChunksBackfillInput) (ConversationChunksBackfillResult, error) {
	if !in.DryRun && workflow.GetVersion(ctx, operatingContextChangeID, workflow.DefaultVersion, 1) != workflow.DefaultVersion {
		if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(in.OperatingMode)); err != nil {
			return ConversationChunksBackfillResult{}, err
		}
		if !caseidentity.AdmittedIdentity(in.MatterID, in.CourtCaseID) {
			return ConversationChunksBackfillResult{}, errors.New("chunk backfill requires the approved case identity")
		}
	}
	result := ConversationChunksBackfillResult{OperatingMode: in.OperatingMode, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, DryRun: in.DryRun}
	if strings.TrimSpace(in.RequestID) == "" {
		return result, errors.New("conversation chunks back-fill requires a request id")
	}
	progress := ConversationChunksProgress{Stage: "starting"}
	if err := workflow.SetQueryHandler(ctx, ConversationChunksProgressQueryName, func() (ConversationChunksProgress, error) {
		return progress, nil
	}); err != nil {
		return result, fmt.Errorf("proffer: register progress query: %w", err)
	}

	if in.DryRun {
		progress.Stage = "estimating"
		sample, sampleCap := in.Sample, in.SampleCap
		if sample <= 0 {
			sample = 6
		}
		if sampleCap <= 0 {
			sampleCap = 5000
		}
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(backfillEstimateTimeout)),
			EstimateContextChunksActivityName, map[string]interface{}{
				"request_id": in.RequestID, "chunker": in.Chunker, "overlap": in.Overlap, "exact": in.Exact,
				"sample": sample, "sample_cap": sampleCap,
			}).Get(ctx, &result.Estimate)
		progress.Stage = "done"
		result.Progress = progress
		return result, wrapActivity(EstimateContextChunksActivityName, err)
	}

	progress.Stage = "listing"
	var listed ListContextThreadsResult
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(backfillListTimeout)),
		ListContextThreadsActivityName, map[string]interface{}{
			"request_id": in.RequestID, "skip_covered": in.SkipCovered, "limit": in.ThreadLimit,
		}).Get(ctx, &listed); err != nil {
		return result, wrapActivity(ListContextThreadsActivityName, err)
	}
	progress.ThreadsTotal = len(listed.Threads)
	if !in.NoCalls {
		progress.CallFilesTotal = len(listed.CallSourceVersions)
	}

	progress.Stage = "chunking"
	for start := 0; start < len(listed.Threads); start += backfillThreadsInFlight {
		end := min(start+backfillThreadsInFlight, len(listed.Threads))
		batch := listed.Threads[start:end]
		chunkFutures := make([]workflow.Future, len(batch))
		for i, thread := range batch {
			chunkFutures[i] = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(4*time.Hour)),
				ChunkContextThreadsActivityName, ChunkThreadsRequest{
					OperatingMode: in.OperatingMode, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, Chunker: in.Chunker, Overlap: in.Overlap,
					ThreadRefs: []ContextThreadSelector{{Corpus: thread.Corpus, ThreadID: thread.ThreadID}},
				})
		}
		publishFutures := make([]workflow.Future, len(batch))
		failed := make([]string, len(batch))
		for i, future := range chunkFutures {
			var plan ChunkThreadsResult
			if err := future.Get(ctx, &plan); err != nil {
				failed[i] = fmt.Sprintf("%s %s: chunk: %v", batch[i].Corpus, batch[i].ThreadID, err)
				continue
			}
			if len(plan.Threads) != 1 {
				failed[i] = fmt.Sprintf("%s %s: chunk returned %d plans, want 1", batch[i].Corpus, batch[i].ThreadID, len(plan.Threads))
				continue
			}
			if err := plan.Threads[0].validate(); err != nil {
				failed[i] = fmt.Sprintf("%s %s: unusable plan: %v", batch[i].Corpus, batch[i].ThreadID, err)
				continue
			}
			publishFutures[i] = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(3*time.Hour)),
				PublishContextChunksActivityName, PublishChunksRequest{OperatingMode: in.OperatingMode, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, Plan: plan.Threads[0]})
		}
		for i, future := range publishFutures {
			if future == nil {
				continue
			}
			var published PublishChunksResult
			if err := future.Get(ctx, &published); err != nil {
				failed[i] = fmt.Sprintf("%s %s: publish: %v", batch[i].Corpus, batch[i].ThreadID, err)
				continue
			}
			progress.ChunksWritten += published.ChunksWritten
			result.StaleDeleted += published.StaleDeleted
			result.EmbedRequests += published.EmbedRequests
		}
		for _, reason := range failed {
			if reason != "" {
				result.Failed = append(result.Failed, reason)
				progress.ThreadsFailed++
			}
		}
		progress.ThreadsDone += len(batch)
	}

	if !in.NoCalls {
		progress.Stage = "call-log files"
		for _, version := range listed.CallSourceVersions {
			var files PublishCallLogFilesResult
			err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(time.Hour)),
				PublishCallLogFilesActivityName, PublishCallLogFilesRequest{OperatingMode: in.OperatingMode, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, SourceVersionID: version}).Get(ctx, &files)
			if err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("call-log file %s: %v", version, err))
				continue
			}
			progress.CallFilesDone += files.Files
			result.EmbedRequests += files.EmbedRequests
		}
	}
	progress.Stage = "done"
	result.Progress = progress
	if len(result.Failed) > 0 {
		return result, fmt.Errorf("proffer: conversation chunks back-fill finished with %d failure(s); first: %s", len(result.Failed), result.Failed[0])
	}
	return result, nil
}

// ConversationChunksRemovalInput is the removal workflow's input.
type ConversationChunksRemovalInput struct {
	OperatingMode string `json:"operating_mode,omitempty"`
	MatterID      string `json:"matter_id,omitempty"`
	CourtCaseID   string `json:"court_case_id,omitempty"`
	RequestID     string `json:"request_id"`
	DryRun        bool   `json:"dry_run"`
	OnlyCovered   bool   `json:"only_covered"`
	OldCollection string `json:"old_collection,omitempty"`
}

// ConversationChunksRemovalResult is the removal workflow's receipt: the verification report and what was (or, in a
// dry run, would be) deleted.
type ConversationChunksRemovalResult struct {
	OperatingMode string                 `json:"operating_mode,omitempty"`
	MatterID      string                 `json:"matter_id,omitempty"`
	CourtCaseID   string                 `json:"court_case_id,omitempty"`
	RequestID     string                 `json:"request_id"`
	DryRun        bool                   `json:"dry_run"`
	Verification  map[string]interface{} `json:"verification"`
	Removal       map[string]interface{} `json:"removal,omitempty"`
}

// ConversationChunksRemovalWorkflow is ConversationChunksRemovalWorkflowName. It verifies first, as its own Activity,
// so the report is in the history even when the removal is refused.
func ConversationChunksRemovalWorkflow(ctx workflow.Context, in ConversationChunksRemovalInput) (ConversationChunksRemovalResult, error) {
	if !in.DryRun && workflow.GetVersion(ctx, operatingContextChangeID, workflow.DefaultVersion, 1) != workflow.DefaultVersion {
		if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(in.OperatingMode)); err != nil {
			return ConversationChunksRemovalResult{}, err
		}
		if !caseidentity.AdmittedIdentity(in.MatterID, in.CourtCaseID) {
			return ConversationChunksRemovalResult{}, errors.New("chunk removal requires the approved case identity")
		}
	}
	result := ConversationChunksRemovalResult{OperatingMode: in.OperatingMode, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, RequestID: in.RequestID, DryRun: in.DryRun}
	if strings.TrimSpace(in.RequestID) == "" {
		return result, errors.New("per-message removal requires a request id")
	}
	request := map[string]interface{}{
		"operating_mode": in.OperatingMode, "matter_id": in.MatterID, "court_case_id": in.CourtCaseID,
		"request_id": in.RequestID, "old_collection": in.OldCollection, "dry_run": in.DryRun, "only_covered": in.OnlyCovered,
	}
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(backfillRemovalTimeout)),
		VerifyChunkCoverageActivityName, request).Get(ctx, &result.Verification); err != nil {
		return result, wrapActivity(VerifyChunkCoverageActivityName, err)
	}
	if verified, _ := result.Verification["verified"].(bool); !verified && !in.OnlyCovered {
		return result, errors.New("proffer: per-message removal refused: the chunks do not cover every message and call; nothing was deleted (see the verification in this result, or run with only_covered)")
	}
	options := chunkActivityOptions(backfillRemovalTimeout)
	options.RetryPolicy = retryPolicy(30*time.Second, backfillSingleRemovalAttempts)
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options),
		RemovePerMessageObjectsActivityName, request).Get(ctx, &result.Removal)
	return result, wrapActivity(RemovePerMessageObjectsActivityName, err)
}

func wrapActivity(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("proffer: %s: %w", name, err)
}
