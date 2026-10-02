package proffer

// Conversation chunks for Weaviate, after the first-party threads are committed.
//
// Owner rules 2026-10-02: Postgres holds every single message and is the record of truth. Weaviate holds only
// conversation CHUNKS, never one entry per message; each chunk links back to the Postgres message ids it covers.
// Calls are one Weaviate entry per call-log FILE; the individual calls stay in Postgres. The Go engine orchestrates:
// this file schedules three Activities on the Python worker's task queue, the same way the extraction commit workflow
// (extraction/flow/workflows.go) schedules build_timeline_generation_activity:
//
//	chunk_context_threads_activity    CHUNK   the threads the source version touched -> per-thread plans
//	publish_context_chunks_activity   PUBLISH one thread: embed its chunks, write them, replace its old chunks
//	publish_call_log_files_activity   PUBLISH one entry per call-log file of the source version
//
// One unit, one job; references and counts only cross the workflow boundary. A plan carries message INDEXES and a
// digest, never message text. The Activities live in server/temporal/chunk_activities.py; their request and result
// shapes are the JSON below (snake_case names match the Python dataclasses).
//
// They are not stages of the stage graph: the graph is the Go worker's 26 canon stages and every stage there is
// registered on the Go worker, while these run on the Python queue like the extraction flow's projection step.
//
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/workflow"
)

const (
	// contextChunksChangeID gates the chunk path: a history recorded before it still publishes one search object
	// per message and call in the Weaviate-first stage and runs no chunk Activity.
	contextChunksChangeID = "proffer-conversation-chunks-v1"
	contextChunksVersion  = workflow.Version(1)

	// ChunkContextThreadsActivityName and its siblings are the Python Activities' registered names.
	ChunkContextThreadsActivityName  = "chunk_context_threads_activity"
	PublishContextChunksActivityName = "publish_context_chunks_activity"
	PublishCallLogFilesActivityName  = "publish_call_log_files_activity"

	// ContextChunksTaskQueue is the Python worker's queue (server/temporal/worker.py, TASK_QUEUE_DEFAULT).
	ContextChunksTaskQueue = "evidence-pipeline"

	// SkipRecordKindsRefKey is the publish_context_search StageRequest reference that names the record kinds the
	// Weaviate-first stage leaves to the chunk path; skippedRecordKinds is its value.
	SkipRecordKindsRefKey = "skip_record_kinds"
	skippedRecordKinds    = "message,call"

	// maxConcurrentChunkPublishes bounds how many threads embed at once (NIM rate limits).
	maxConcurrentChunkPublishes = 3
)

// ChunkThreadsRequest is the chunk Activity's input: a reference, never a payload. Chunker and Overlap are empty
// and zero to take the platform defaults (Neural distilbert, overlap 2).
type ChunkThreadsRequest struct {
	RequestID       string `json:"request_id"`
	SourceVersionID string `json:"source_version_id"`
	Chunker         string `json:"chunker,omitempty"`
	Overlap         int    `json:"overlap,omitempty"`
}

// ChunkThreadPlan is one thread's chunking: inclusive [first, last] message indexes in thread order and a digest
// that binds the plan to the exact ordered messages it was cut from.
type ChunkThreadPlan struct {
	Corpus         string  `json:"corpus"`
	ThreadID       string  `json:"thread_id"`
	MessageCount   int     `json:"message_count"`
	Digest         string  `json:"digest"`
	Chunker        string  `json:"chunker"`
	ChunkerVersion string  `json:"chunker_version"`
	Overlap        int     `json:"overlap"`
	Spans          [][]int `json:"spans"`
}

// ChunkThreadsResult is the chunk Activity's output.
type ChunkThreadsResult struct {
	Chunker        string            `json:"chunker"`
	ChunkerVersion string            `json:"chunker_version"`
	Overlap        int               `json:"overlap"`
	Threads        []ChunkThreadPlan `json:"threads"`
}

// PublishChunksRequest is one thread's embed-and-publish input: the plan the chunk Activity returned.
type PublishChunksRequest struct {
	RequestID string          `json:"request_id"`
	Plan      ChunkThreadPlan `json:"plan"`
}

// PublishChunksResult is one thread's outcome.
type PublishChunksResult struct {
	Corpus          string `json:"corpus"`
	ThreadID        string `json:"thread_id"`
	ChunksWritten   int    `json:"chunks_written"`
	StaleDeleted    int    `json:"stale_deleted"`
	EmbedTexts      int    `json:"embed_texts"`
	EmbedRequests   int    `json:"embed_requests"`
	SkippedExisting bool   `json:"skipped_existing"`
}

// PublishCallLogFilesRequest is the call-log publish input.
type PublishCallLogFilesRequest struct {
	RequestID       string `json:"request_id"`
	SourceVersionID string `json:"source_version_id"`
}

// PublishCallLogFilesResult counts the call-log files written (one entry each) and the calls they cover.
type PublishCallLogFilesResult struct {
	Files         int `json:"files"`
	Calls         int `json:"calls"`
	EmbedRequests int `json:"embed_requests"`
}

// ContextChunksSummary is what the workflow keeps of the chunk step.
type ContextChunksSummary struct {
	Threads       int
	Chunks        int
	StaleDeleted  int
	EmbedRequests int
	CallFiles     int
	Calls         int
}

// validate rejects a plan the publish Activity must never receive: it fails closed in the workflow, before any embed.
func (p ChunkThreadPlan) validate() error {
	if strings.TrimSpace(p.ThreadID) == "" || strings.TrimSpace(p.Corpus) == "" || strings.TrimSpace(p.Digest) == "" {
		return fmt.Errorf("chunk plan lacks its corpus, thread id or digest")
	}
	if p.MessageCount < 1 || len(p.Spans) == 0 {
		return fmt.Errorf("chunk plan for thread %s covers no message", p.ThreadID)
	}
	if p.Overlap < 0 {
		return fmt.Errorf("chunk plan for thread %s has a negative overlap", p.ThreadID)
	}
	covered := make([]bool, p.MessageCount)
	previousFirst := -1
	for _, span := range p.Spans {
		if len(span) != 2 || span[0] < 0 || span[1] < span[0] || span[1] >= p.MessageCount {
			return fmt.Errorf("chunk plan for thread %s has an out-of-range span %v", p.ThreadID, span)
		}
		if span[0] <= previousFirst {
			return fmt.Errorf("chunk plan for thread %s has spans out of order at %v", p.ThreadID, span)
		}
		previousFirst = span[0]
		for i := span[0]; i <= span[1]; i++ {
			covered[i] = true
		}
	}
	for i, ok := range covered {
		if !ok {
			return fmt.Errorf("chunk plan for thread %s leaves message %d in no chunk", p.ThreadID, i)
		}
	}
	return nil
}

func chunkActivityOptions(startToClose time.Duration) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		TaskQueue: ContextChunksTaskQueue,
		// The Python worker is its own Coolify app; a deploy or restart must not fail a run, so a long queue wait.
		ScheduleToStartTimeout: 30 * time.Minute,
		StartToCloseTimeout:    startToClose,
		// The Activities heartbeat per window and per embed batch; a silent worker is a dead worker.
		HeartbeatTimeout: 5 * time.Minute,
		RetryPolicy:      retryPolicy(30*time.Second, 3),
	}
}

// execContextChunks chunks and publishes the threads the source version touched, then the call-log files.
// Threads and calls are independent: either may be switched off by the caller.
func (r *run) execContextChunks(ctx workflow.Context, sourceVersionID string, threads, calls bool) (ContextChunksSummary, error) {
	var summary ContextChunksSummary
	if strings.TrimSpace(sourceVersionID) == "" {
		return summary, fmt.Errorf("proffer: conversation chunks need the run's source version id")
	}
	if threads {
		r.markStageStarted(ChunkContextThreadsActivityName)
		var plan ChunkThreadsResult
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(4*time.Hour)),
			ChunkContextThreadsActivityName, ChunkThreadsRequest{RequestID: r.requestID, SourceVersionID: sourceVersionID}).Get(ctx, &plan)
		r.markStageSettled(ChunkContextThreadsActivityName)
		if err != nil {
			return summary, fmt.Errorf("proffer: %s: %w", ChunkContextThreadsActivityName, err)
		}
		for _, thread := range plan.Threads {
			if err := thread.validate(); err != nil {
				return summary, fmt.Errorf("proffer: %s returned an unusable plan: %w", ChunkContextThreadsActivityName, err)
			}
		}
		r.markStageStarted(PublishContextChunksActivityName)
		for start := 0; start < len(plan.Threads); start += maxConcurrentChunkPublishes {
			end := min(start+maxConcurrentChunkPublishes, len(plan.Threads))
			futures := make([]workflow.Future, 0, end-start)
			for _, thread := range plan.Threads[start:end] {
				futures = append(futures, workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(3*time.Hour)),
					PublishContextChunksActivityName, PublishChunksRequest{RequestID: r.requestID, Plan: thread}))
			}
			var firstErr error
			for index, future := range futures {
				var result PublishChunksResult
				if err := future.Get(ctx, &result); err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("proffer: %s for thread %s: %w", PublishContextChunksActivityName, plan.Threads[start+index].ThreadID, err)
					}
					continue
				}
				summary.Threads++
				summary.Chunks += result.ChunksWritten
				summary.StaleDeleted += result.StaleDeleted
				summary.EmbedRequests += result.EmbedRequests
			}
			if firstErr != nil {
				r.markStageSettled(PublishContextChunksActivityName)
				return summary, firstErr
			}
		}
		r.markStageSettled(PublishContextChunksActivityName)
	}
	if calls {
		r.markStageStarted(PublishCallLogFilesActivityName)
		var result PublishCallLogFilesResult
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(time.Hour)),
			PublishCallLogFilesActivityName, PublishCallLogFilesRequest{RequestID: r.requestID, SourceVersionID: sourceVersionID}).Get(ctx, &result)
		r.markStageSettled(PublishCallLogFilesActivityName)
		if err != nil {
			return summary, fmt.Errorf("proffer: %s: %w", PublishCallLogFilesActivityName, err)
		}
		summary.CallFiles, summary.Calls, summary.EmbedRequests = result.Files, result.Calls, summary.EmbedRequests+result.EmbedRequests
	}
	return summary, nil
}
