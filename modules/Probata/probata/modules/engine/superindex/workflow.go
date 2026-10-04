// Package superindex orchestrates the Coco Super Index as a Temporal workflow driven by a schedule.
//
// Owner order 2026-10-02: the index runs automatically and every pass is traceable. The Go engine owns the sequence;
// every stage is its own Activity on the Python worker that ships with the superindex image (queue "superindex"), so
// each pass shows up in Temporal history with its counts, heartbeats and retries, and nothing runs as a daemon loop
// nobody can see. The Activities are thin wrappers over casebible_index/stage_runner.py (Consignatio/Intake/backend);
// the workflow decides what runs next, never the Activity.
//
//	discover      catalog watermark + classification; ends the cycle when nothing changed
//	extract       CocoIndex incremental pass (extract + chunk + document rows), time-sliced; loops via More
//	summarize     the separate summary pass over documents the index picks
//	embed         one Activity per vector slot, run in parallel, until nothing is left without a vector
//	publish       chunks + vectors -> Weaviate (CaseBibleChunks20261002); follows moved files
//	graph         the active snapshot -> the surreal-intake file graph; decides itself to skip (at most daily)
//	commit        advance the catalog watermark, last, only after every stage of the cycle succeeded
//
// Extract and chunk are one Activity on purpose: extracted text streams into the chunker and is never materialized
// (that is what keeps a 1.3 GB export bounded).
//
// The image stage (owner 2026-10-03, off unless the worker has INTAKE_IMAGE_STAGE=on) extends Intake's image index
// with a catalog source. It runs as a loop of bounded SLICES, after the text passes and also on a tick where the
// catalog did not change (it is driven by its own ledger, not by the watermark, so a backlog keeps draining):
//
//	image_fetch    a slice of images or scanned-PDF pages, bucket -> spool; answers the slots the worker has enabled
//	image_facts    original time, device, screenshot / photo / scan
//	image_ocr      the text in the pictures
//	image_embed    one Activity per enabled slot (image_maxsim, image_single, image_colqwen), in parallel
//	image_publish  slice -> Weaviate IntakeImageV1, ledger, release the spool
//
// They are plain Python Activities, not n8n flows: a flow is one workflow execution per file in n8n and one Activity
// pair per file in Temporal history, which cannot carry hundreds of thousands of images; one Activity per bounded
// slice can. The per-file PDF/image tools on the tool gateway stay available as selectable OCR engines.
//
// Byline: Claude Code · Sonnet 5.5 · 2026-10-03 (image stage; the rest 2026-10-02)
package superindex

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// WorkflowName is the exact Temporal workflow type the Go worker registers.
	WorkflowName = "SuperIndexCycleWorkflow"
	// StatusQueryName returns the run's Status while it runs.
	StatusQueryName = "superindex_status"
	// TaskQueue is the Python worker's queue (casebible_index/temporal_worker.py).
	TaskQueue = "superindex"

	DiscoverActivityName  = "superindex_discover_activity"
	ExtractActivityName   = "superindex_extract_activity"
	SummarizeActivityName = "superindex_summarize_activity"
	EmbedActivityName     = "superindex_embed_activity"
	PublishActivityName   = "superindex_publish_activity"
	GraphActivityName     = "superindex_graph_activity"
	CommitActivityName    = "superindex_commit_activity"

	ImageFetchActivityName   = "superindex_image_fetch_activity"
	ImageFactsActivityName   = "superindex_image_facts_activity"
	ImageOcrActivityName     = "superindex_image_ocr_activity"
	ImageEmbedActivityName   = "superindex_image_embed_activity"
	ImagePublishActivityName = "superindex_image_publish_activity"

	// ImageStageVersion is the workflow.GetVersion change id of the image stage: executions that started before it
	// existed replay without the stage, new ones run it.
	ImageStageVersion = "superindex-image-stage"

	// defaultImageSlices bounds one cycle's image loop so Temporal history stays small (about 25 events per slice);
	// the next cycle resumes from the ledger.
	defaultImageSlices = 150

	defaultSlot          = "text_nim"
	defaultMaxPasses     = 200
	defaultExtractBudget = 4 * 3600
	maxLoopsPerStage     = 500
)

// CycleRequest is the schedule's (or an operator's) input. Every field is optional.
type CycleRequest struct {
	// Force runs the whole cycle even when the catalog watermark has not moved (a re-check or a backfill).
	Force bool `json:"force,omitempty"`
	// Slots are the embedding slots to fill; empty means text_nim. The Python worker refuses a slot that is not
	// enabled there (INTAKE_VECTOR_SLOTS), so this can only ask for less than is configured.
	Slots []string `json:"slots,omitempty"`
	// ExtractBudgetSeconds slices the extract pass: it stops after this long, publishes what it finished, and runs
	// again. Zero takes the default (4 hours).
	ExtractBudgetSeconds int `json:"extract_budget_seconds,omitempty"`
	// Limit and PathPrefix bound a proof run (catalog rows taken, key prefix). Never set by the schedule.
	Limit      int    `json:"limit,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	// MaxPasses bounds the extract/embed/publish loop; zero takes the default.
	MaxPasses int `json:"max_passes,omitempty"`
	// SkipSummary leaves the summary pass out of this cycle.
	SkipSummary bool `json:"skip_summary,omitempty"`
	// SkipImages leaves the image stage out of this cycle. The stage is also off in the worker unless
	// INTAKE_IMAGE_STAGE=on, in which case its Activities answer "skipped" and the loop ends at once.
	SkipImages bool `json:"skip_images,omitempty"`
	// ImageSlots restricts the image embedding slots to a subset of what the worker enabled; empty takes them all.
	ImageSlots []string `json:"image_slots,omitempty"`
	// ImageMaxSlices bounds the image loop of one cycle; zero takes the default.
	ImageMaxSlices int `json:"image_max_slices,omitempty"`
	// ImageLocators confines image reads to an explicit proof manifest; nil leaves the normal source unchanged.
	ImageLocators []ImageLocator `json:"image_locators,omitempty"`
}

// ImageLocator identifies one approved bucket object for a bounded image proof.
// Inputs are the source provider, bucket and exact key; output is an Activity selector with no side effects.
// Pick this instead of a prefix when proof files share a namespace with unrelated originals.
type ImageLocator struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
}

// StageRequest is every Activity's input: a reference and small parameters, never a payload.
type StageRequest struct {
	CycleID string         `json:"cycle_id"`
	Params  map[string]any `json:"params,omitempty"`
}

// DiscoverResult is what the workflow keeps of the discovery receipt.
type DiscoverResult struct {
	Changed         bool             `json:"changed"`
	Forced          bool             `json:"forced"`
	ChangedBuckets  []string         `json:"changed_buckets"`
	Watermark       []map[string]any `json:"watermark"`
	Kinds           map[string]int   `json:"kinds"`
	RepresentedRows int              `json:"represented_rows"`
}

// ExtractResult carries the extract pass's counts.
type ExtractResult struct {
	More             bool           `json:"more"`
	ExitCode         int            `json:"exit_code"`
	DurationSeconds  float64        `json:"duration_s"`
	PeakRSSMB        int            `json:"peak_rss_mb"`
	StoppedForBudget bool           `json:"stopped_for_time_budget"`
	RunStatus        map[string]any `json:"run_status"`
	B2Requests       int            `json:"b2_requests"`
	B2BytesRead      int64          `json:"b2_bytes_read"`
	IndexReceipt     string         `json:"index_receipt"`
}

// SummarizeResult carries the summary pass's counts.
type SummarizeResult struct {
	More       bool `json:"more"`
	Selected   int  `json:"selected"`
	Summarized int  `json:"summarized"`
	Failed     int  `json:"failed"`
}

// EmbedResult carries one slot's counts.
type EmbedResult struct {
	More           bool   `json:"more"`
	Slot           string `json:"slot"`
	Selected       int    `json:"selected"`
	EmbeddedOK     int    `json:"embedded_ok"`
	EmbeddedFailed int    `json:"embedded_failed"`
	Requests       int    `json:"requests"`
}

// PublishResult carries the publish pass's counts.
type PublishResult struct {
	More            bool `json:"more"`
	Published       int  `json:"published"`
	CollectionCount int  `json:"collection_count"`
	Relocated       struct {
		Patched int `json:"patched"`
		Missing int `json:"missing"`
	} `json:"relocated"`
}

// GraphResult says whether the file graph was projected this cycle, and why not when it was not.
type GraphResult struct {
	Projected bool   `json:"projected"`
	Reason    string `json:"reason"`
	Documents int    `json:"documents"`
}

// ImageFetchResult is what the workflow keeps of an image_fetch answer. An empty SliceID means nothing is left.
type ImageFetchResult struct {
	SliceID string   `json:"slice_id"`
	Kind    string   `json:"kind"`
	Count   int      `json:"count"`
	Failed  int      `json:"failed"`
	More    bool     `json:"more"`
	Resumed bool     `json:"resumed"`
	Slots   []string `json:"slots"`
}

// ImagePublishResult carries one slice's publish counts.
type ImagePublishResult struct {
	Published int `json:"published"`
	Partial   int `json:"partial"`
	Failed    int `json:"failed"`
}

// Status is what the query handler returns while a cycle runs.
type Status struct {
	CycleID     string `json:"cycle_id"`
	Stage       string `json:"stage"`
	Pass        int    `json:"pass"`
	Extracted   int    `json:"extracted_passes"`
	Embedded    int    `json:"embedded_texts"`
	Published   int    `json:"published_objects"`
	Summarized  int    `json:"summarized_documents"`
	ImageSlices int    `json:"image_slices"`
	NoChange    bool   `json:"no_change"`
	Committed   bool   `json:"committed"`
}

// CycleResult is the workflow's result and its history summary.
type CycleResult struct {
	CycleID    string         `json:"cycle_id"`
	Outcome    string         `json:"outcome"` // no_change | committed
	Discover   DiscoverResult `json:"discover"`
	Passes     int            `json:"passes"`
	Summarized int            `json:"summarized"`
	Embedded   int            `json:"embedded_texts"`
	Failed     int            `json:"embed_failed"`
	Published  int            `json:"published_objects"`
	Relocated  int            `json:"relocated_objects"`
	// GraphProjected is true when this cycle wrote the file graph (it skips unless the snapshot changed and a day passed).
	GraphProjected bool `json:"graph_projected"`
	// ImageSlices, ImagesPublished and ImagesFailed count the image stage's work this cycle.
	ImageSlices     int `json:"image_slices"`
	ImagesPublished int `json:"images_published"`
	ImagesFailed    int `json:"images_failed"`
}

func cycleID(ctx workflow.Context) string {
	return workflow.Now(ctx).UTC().Format("20060102T150405Z")
}

func activityOptions(startToClose, heartbeat time.Duration, attempts int32) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		TaskQueue: TaskQueue,
		// The Python worker is its own Coolify service; a deploy or restart must not fail a cycle.
		ScheduleToStartTimeout: 30 * time.Minute,
		StartToCloseTimeout:    startToClose,
		HeartbeatTimeout:       heartbeat,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Minute,
			MaximumAttempts:    attempts,
		},
	}
}

func normalizeSlots(slots []string) ([]string, error) {
	if len(slots) == 0 {
		return []string{defaultSlot}, nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(slots))
	for _, slot := range slots {
		slot = strings.TrimSpace(slot)
		if slot == "" || seen[slot] {
			return nil, fmt.Errorf("superindex: slots must be distinct and non-empty, got %v", slots)
		}
		seen[slot] = true
		out = append(out, slot)
	}
	return out, nil
}

// CycleWorkflow runs one traceable pass of the Super Index.
// Inputs: workflow context and bounded cycle settings. Output: stage counts or an error.
// Side effects: schedules Python Activities for catalog reads, derived indexing, publication and receipts.
// Pick this for scheduled indexing or a fenced proof; individual Activities perform only their own stage.
func CycleWorkflow(ctx workflow.Context, req CycleRequest) (CycleResult, error) {
	slots, err := normalizeSlots(req.Slots)
	if err != nil {
		return CycleResult{}, temporal.NewNonRetryableApplicationError(err.Error(), "superindex_bad_request", nil)
	}
	maxPasses := req.MaxPasses
	if maxPasses <= 0 {
		maxPasses = defaultMaxPasses
	}
	budget := req.ExtractBudgetSeconds
	if budget <= 0 {
		budget = defaultExtractBudget
	}
	id := cycleID(ctx)
	result := CycleResult{CycleID: id}
	status := Status{CycleID: id, Stage: "discover"}
	if err := workflow.SetQueryHandler(ctx, StatusQueryName, func() (Status, error) { return status, nil }); err != nil {
		return result, err
	}

	stage := func(name string, startToClose, heartbeat time.Duration, attempts int32, params map[string]any, out any) error {
		status.Stage = name
		actCtx := workflow.WithActivityOptions(ctx, activityOptions(startToClose, heartbeat, attempts))
		return workflow.ExecuteActivity(actCtx, name, StageRequest{CycleID: id, Params: params}).Get(ctx, out)
	}

	discoverParams := map[string]any{"force": req.Force}
	var discovered DiscoverResult
	if err := stage(DiscoverActivityName, 2*time.Hour, 5*time.Minute, 3, discoverParams, &discovered); err != nil {
		return result, fmt.Errorf("superindex: %s: %w", DiscoverActivityName, err)
	}
	result.Discover = discovered
	if !discovered.Changed {
		status.NoChange = true
		result.Outcome = "no_change"
		// The image backlog is driven by its own ledger, not by the watermark: a quiet catalog still drains it.
		if err := runImageStage(ctx, &req, id, &status, &result, stage); err != nil {
			return result, err
		}
		return result, nil
	}

	extractParams := map[string]any{"time_budget_s": budget}
	if req.Limit > 0 {
		extractParams["limit"] = req.Limit
	}
	if req.PathPrefix != "" {
		extractParams["path_prefix"] = req.PathPrefix
	}

	for pass := 1; pass <= maxPasses; pass++ {
		status.Pass = pass

		var extracted ExtractResult
		// The Activity enforces the time budget itself; the Temporal timeout is the budget plus the shutdown grace.
		extractTimeout := time.Duration(budget)*time.Second + 30*time.Minute
		if err := stage(ExtractActivityName, extractTimeout, 3*time.Minute, 3, extractParams, &extracted); err != nil {
			return result, fmt.Errorf("superindex: %s (pass %d): %w", ExtractActivityName, pass, err)
		}
		status.Extracted++

		if !req.SkipSummary {
			for loop := 0; loop < maxLoopsPerStage; loop++ {
				var summarized SummarizeResult
				if err := stage(SummarizeActivityName, 2*time.Hour, 5*time.Minute, 3, nil, &summarized); err != nil {
					return result, fmt.Errorf("superindex: %s (pass %d): %w", SummarizeActivityName, pass, err)
				}
				result.Summarized += summarized.Summarized
				status.Summarized += summarized.Summarized
				if !summarized.More || summarized.Summarized == 0 {
					break
				}
			}
		}

		// One embed Activity per slot, in parallel; each repeats until its slot has nothing left.
		for loop := 0; loop < maxLoopsPerStage; loop++ {
			futures := make([]workflow.Future, 0, len(slots))
			for _, slot := range slots {
				actCtx := workflow.WithActivityOptions(ctx, activityOptions(3*time.Hour, 5*time.Minute, 5))
				futures = append(futures, workflow.ExecuteActivity(actCtx, EmbedActivityName,
					StageRequest{CycleID: id, Params: map[string]any{"slot": slot}}))
			}
			status.Stage = EmbedActivityName
			again := false
			for i, future := range futures {
				var embedded EmbedResult
				if err := future.Get(ctx, &embedded); err != nil {
					return result, fmt.Errorf("superindex: %s slot %s (pass %d): %w", EmbedActivityName, slots[i], pass, err)
				}
				result.Embedded += embedded.EmbeddedOK
				result.Failed += embedded.EmbeddedFailed
				status.Embedded += embedded.EmbeddedOK
				again = again || embedded.More
			}
			if !again {
				break
			}
		}

		for loop := 0; loop < maxLoopsPerStage; loop++ {
			var published PublishResult
			if err := stage(PublishActivityName, 3*time.Hour, 5*time.Minute, 5, nil, &published); err != nil {
				return result, fmt.Errorf("superindex: %s (pass %d): %w", PublishActivityName, pass, err)
			}
			result.Published += published.Published
			result.Relocated += published.Relocated.Patched
			status.Published += published.Published
			if !published.More {
				break
			}
		}

		result.Passes = pass
		if !extracted.More {
			break
		}
		if pass == maxPasses {
			// Not an error: the next scheduled cycle resumes from the same watermark (nothing was committed).
			result.Outcome = "max_passes_reached"
			return result, nil
		}
	}

	if err := runImageStage(ctx, &req, id, &status, &result, stage); err != nil {
		return result, err
	}

	if req.Limit > 0 || req.PathPrefix != "" {
		// A bounded proof run never advances the watermark: the catalog has not been caught up to.
		status.Stage = "done"
		result.Outcome = "proof_run_not_committed"
		return result, nil
	}
	var graph GraphResult
	if err := stage(GraphActivityName, 3*time.Hour, 5*time.Minute, 3, nil, &graph); err != nil {
		return result, fmt.Errorf("superindex: %s: %w", GraphActivityName, err)
	}
	result.GraphProjected = graph.Projected
	var committed struct {
		Committed bool `json:"committed"`
	}
	commitParams := map[string]any{
		"watermark": discovered.Watermark,
		"summary": map[string]any{
			"passes": result.Passes, "published": result.Published, "embedded": result.Embedded,
			"summarized": result.Summarized,
		},
	}
	if err := stage(CommitActivityName, 10*time.Minute, time.Minute, 5, commitParams, &committed); err != nil {
		return result, fmt.Errorf("superindex: %s: %w", CommitActivityName, err)
	}
	status.Committed = committed.Committed
	status.Stage = "done"
	result.Outcome = "committed"
	return result, nil
}

type stageFunc func(name string, startToClose, heartbeat time.Duration, attempts int32, params map[string]any, out any) error

// runImageStage drains up to ImageMaxSlices slices of the image backlog. Each slice runs five Activities in a fixed
// order (fetch, facts, ocr, embed per slot in parallel, publish); the Python worker decides what a slice holds and
// which slots exist, the workflow only sequences and bounds. A worker with the stage off answers fetch with an empty
// slice id and the loop ends after one cheap Activity.
func runImageStage(ctx workflow.Context, req *CycleRequest, id string, status *Status, result *CycleResult, stage stageFunc) error {
	if req.SkipImages || workflow.GetVersion(ctx, ImageStageVersion, workflow.DefaultVersion, 1) < 1 {
		return nil
	}
	maxSlices := req.ImageMaxSlices
	if maxSlices <= 0 {
		maxSlices = defaultImageSlices
	}
	fetchParams := map[string]any{}
	if req.ImageLocators != nil {
		fetchParams["locators"] = req.ImageLocators
	}
	if req.PathPrefix != "" {
		fetchParams["path_prefix"] = req.PathPrefix
	}
	remaining := req.Limit
	for slice := 0; slice < maxSlices; slice++ {
		if req.Limit > 0 {
			if remaining <= 0 {
				break
			}
			fetchParams["max_files"] = remaining
			fetchParams["max_items"] = remaining
		}
		var fetched ImageFetchResult
		if err := stage(ImageFetchActivityName, 3*time.Hour, 5*time.Minute, 3, fetchParams, &fetched); err != nil {
			return fmt.Errorf("superindex: %s: %w", ImageFetchActivityName, err)
		}
		if fetched.SliceID == "" {
			break
		}
		if req.Limit > 0 {
			used := fetched.Count + fetched.Failed
			if used <= 0 || used > remaining {
				return fmt.Errorf("superindex: image fetch exceeded remaining proof budget: count=%d remaining=%d", used, remaining)
			}
			remaining -= used
		}
		chain := map[string]any{"slice_id": fetched.SliceID}
		if fetched.Count > 0 {
			var facts, ocr map[string]any
			if err := stage(ImageFactsActivityName, 2*time.Hour, 5*time.Minute, 3, chain, &facts); err != nil {
				return fmt.Errorf("superindex: %s (slice %s): %w", ImageFactsActivityName, fetched.SliceID, err)
			}
			if err := stage(ImageOcrActivityName, 2*time.Hour, 5*time.Minute, 3, chain, &ocr); err != nil {
				return fmt.Errorf("superindex: %s (slice %s): %w", ImageOcrActivityName, fetched.SliceID, err)
			}
			slots := fetched.Slots
			if len(req.ImageSlots) > 0 {
				slots = req.ImageSlots
			}
			futures := make([]workflow.Future, 0, len(slots))
			for _, slot := range slots {
				actCtx := workflow.WithActivityOptions(ctx, activityOptions(2*time.Hour, 5*time.Minute, 3))
				futures = append(futures, workflow.ExecuteActivity(actCtx, ImageEmbedActivityName,
					StageRequest{CycleID: id, Params: map[string]any{"slice_id": fetched.SliceID, "slot": slot}}))
			}
			status.Stage = ImageEmbedActivityName
			for i, future := range futures {
				var embedded map[string]any
				if err := future.Get(ctx, &embedded); err != nil {
					return fmt.Errorf("superindex: %s slot %s (slice %s): %w", ImageEmbedActivityName, slots[i], fetched.SliceID, err)
				}
			}
		}
		var published ImagePublishResult
		if err := stage(ImagePublishActivityName, 3*time.Hour, 5*time.Minute, 5, chain, &published); err != nil {
			return fmt.Errorf("superindex: %s (slice %s): %w", ImagePublishActivityName, fetched.SliceID, err)
		}
		result.ImageSlices++
		result.ImagesPublished += published.Published
		result.ImagesFailed += published.Failed
		status.ImageSlices++
		if !fetched.More {
			break
		}
	}
	return nil
}
