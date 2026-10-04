package superindex

// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

import (
	"context"
	"sync"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type recorder struct {
	mu    sync.Mutex
	calls []string
	// perStage counts calls to a stage, so a stage can answer "more" a fixed number of times.
	perStage    map[string]int
	imageParams []map[string]any
}

func (r *recorder) note(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name)
	if r.perStage == nil {
		r.perStage = map[string]int{}
	}
	r.perStage[name]++
	return r.perStage[name]
}

// imageSlices is how many slices the fake image backlog holds; each fetch hands one out until it is empty.
func registerImages(env *testsuite.TestWorkflowEnvironment, r *recorder, imageSlices int, slots []string) {
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (ImageFetchResult, error) {
		n := r.note(ImageFetchActivityName)
		r.imageParams = append(r.imageParams, req.Params)
		if n > imageSlices {
			return ImageFetchResult{}, nil
		}
		count := 40
		if remaining, ok := req.Params["max_items"].(float64); ok {
			count = min(2, int(remaining))
		}
		return ImageFetchResult{SliceID: "slice-" + string(rune('a'+n-1)), Count: count, More: n < imageSlices, Slots: slots}, nil
	}, activity.RegisterOptions{Name: ImageFetchActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (map[string]any, error) {
		r.note(ImageFactsActivityName)
		return map[string]any{"images": 40}, nil
	}, activity.RegisterOptions{Name: ImageFactsActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (map[string]any, error) {
		r.note(ImageOcrActivityName)
		return map[string]any{"images": 40}, nil
	}, activity.RegisterOptions{Name: ImageOcrActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (map[string]any, error) {
		r.note(ImageEmbedActivityName + ":" + req.Params["slot"].(string))
		return map[string]any{"embedded_ok": 40}, nil
	}, activity.RegisterOptions{Name: ImageEmbedActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (ImagePublishResult, error) {
		r.note(ImagePublishActivityName)
		return ImagePublishResult{Published: 39, Partial: 1, Failed: 1}, nil
	}, activity.RegisterOptions{Name: ImagePublishActivityName})
}

func register(env *testsuite.TestWorkflowEnvironment, r *recorder, changed bool, extractMore int) {
	registerImages(env, r, 0, nil)
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (DiscoverResult, error) {
		r.note(DiscoverActivityName)
		return DiscoverResult{
			Changed:   changed,
			Watermark: []map[string]any{{"provider": "b2", "bucket": "salem-data", "objects": 3}},
		}, nil
	}, activity.RegisterOptions{Name: DiscoverActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (ExtractResult, error) {
		n := r.note(ExtractActivityName)
		return ExtractResult{More: n <= extractMore}, nil
	}, activity.RegisterOptions{Name: ExtractActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (SummarizeResult, error) {
		r.note(SummarizeActivityName)
		return SummarizeResult{Summarized: 2}, nil
	}, activity.RegisterOptions{Name: SummarizeActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (EmbedResult, error) {
		n := r.note(EmbedActivityName + ":" + req.Params["slot"].(string))
		return EmbedResult{Slot: req.Params["slot"].(string), EmbeddedOK: 10, More: n == 1}, nil
	}, activity.RegisterOptions{Name: EmbedActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (PublishResult, error) {
		r.note(PublishActivityName)
		return PublishResult{Published: 5}, nil
	}, activity.RegisterOptions{Name: PublishActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (GraphResult, error) {
		r.note(GraphActivityName)
		return GraphResult{Projected: false, Reason: "snapshot unchanged since the last projection"}, nil
	}, activity.RegisterOptions{Name: GraphActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (map[string]any, error) {
		r.note(CommitActivityName)
		return map[string]any{"committed": true}, nil
	}, activity.RegisterOptions{Name: CommitActivityName})
}

func run(t *testing.T, req CycleRequest, r *recorder, changed bool, extractMore int) CycleResult {
	t.Helper()
	return runWithImages(t, req, r, changed, extractMore, 0, nil)
}

func runWithImages(t *testing.T, req CycleRequest, r *recorder, changed bool, extractMore, imageSlices int, slots []string) CycleResult {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(CycleWorkflow)
	register(env, r, changed, extractMore)
	registerImages(env, r, imageSlices, slots)
	env.ExecuteWorkflow(CycleWorkflow, req)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
	var result CycleResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestUnchangedCatalogEndsAfterDiscover(t *testing.T) {
	r := &recorder{}
	result := run(t, CycleRequest{}, r, false, 0)
	if result.Outcome != "no_change" {
		t.Fatalf("outcome = %q", result.Outcome)
	}
	if len(r.calls) != 2 || r.calls[0] != DiscoverActivityName || r.calls[1] != ImageFetchActivityName {
		t.Fatalf("an unchanged catalog must run only discover and the (empty) image fetch, ran %v", r.calls)
	}
}

func TestChangedCatalogRunsEveryStageAndCommitsLast(t *testing.T) {
	r := &recorder{}
	result := run(t, CycleRequest{}, r, true, 0)
	if result.Outcome != "committed" {
		t.Fatalf("outcome = %q", result.Outcome)
	}
	want := []string{
		DiscoverActivityName, ExtractActivityName, SummarizeActivityName,
		EmbedActivityName + ":text_nim", EmbedActivityName + ":text_nim", // more=true on the first call
		PublishActivityName, ImageFetchActivityName, GraphActivityName, CommitActivityName,
	}
	if len(r.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", r.calls, want)
	}
	for i := range want {
		if r.calls[i] != want[i] {
			t.Fatalf("call %d = %q, want %q (all: %v)", i, r.calls[i], want[i], r.calls)
		}
	}
	if result.Embedded != 20 || result.Published != 5 || result.Summarized != 2 {
		t.Fatalf("counts = %+v", result)
	}
}

func TestExtractSlicesLoopThroughEmbedAndPublishBeforeCommit(t *testing.T) {
	r := &recorder{}
	result := run(t, CycleRequest{SkipSummary: true}, r, true, 2) // extract says "more" twice, then done
	if result.Passes != 3 {
		t.Fatalf("passes = %d, want 3", result.Passes)
	}
	if last := r.calls[len(r.calls)-1]; last != CommitActivityName {
		t.Fatalf("commit must be last, got %q", last)
	}
	commits := 0
	for _, c := range r.calls {
		if c == CommitActivityName {
			commits++
		}
		if c == SummarizeActivityName {
			t.Fatal("summary ran although SkipSummary was set")
		}
	}
	if commits != 1 {
		t.Fatalf("commit ran %d times", commits)
	}
}

func TestEverySlotGetsItsOwnEmbedActivity(t *testing.T) {
	r := &recorder{}
	run(t, CycleRequest{Slots: []string{"text_nim", "legal"}, SkipSummary: true}, r, true, 0)
	if r.perStage[EmbedActivityName+":text_nim"] == 0 || r.perStage[EmbedActivityName+":legal"] == 0 {
		t.Fatalf("each slot needs an embed Activity, got %v", r.perStage)
	}
}

func TestBoundedProofRunNeverCommits(t *testing.T) {
	r := &recorder{}
	result := run(t, CycleRequest{Limit: 5, SkipSummary: true}, r, true, 0)
	if result.Outcome != "proof_run_not_committed" {
		t.Fatalf("outcome = %q", result.Outcome)
	}
	for _, c := range r.calls {
		if c == CommitActivityName {
			t.Fatal("a bounded proof run advanced the watermark")
		}
	}
}

func TestBadSlotListFailsBeforeAnyActivity(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(CycleWorkflow)
	r := &recorder{}
	register(env, r, true, 0)
	env.ExecuteWorkflow(CycleWorkflow, CycleRequest{Slots: []string{"text_nim", "text_nim"}})
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("a repeated slot must be refused")
	}
	if len(r.calls) != 0 {
		t.Fatalf("no Activity may run for a bad request, ran %v", r.calls)
	}
}

func TestScheduleConfigRefusesProofRuns(t *testing.T) {
	if err := (ScheduleConfig{WorkflowTaskQueue: "q", Request: CycleRequest{Limit: 3}}).validate(); err == nil {
		t.Fatal("a scheduled proof run must be refused")
	}
	if err := (ScheduleConfig{}).validate(); err == nil {
		t.Fatal("a schedule without a workflow task queue must be refused")
	}
	if err := (ScheduleConfig{WorkflowTaskQueue: "q"}).validate(); err != nil {
		t.Fatalf("default schedule refused: %v", err)
	}
}

func TestImageBacklogDrainsOnAQuietCatalogWithEverySlotInParallel(t *testing.T) {
	r := &recorder{}
	slots := []string{"image_maxsim", "image_single"}
	result := runWithImages(t, CycleRequest{}, r, false, 0, 2, slots)
	if result.Outcome != "no_change" {
		t.Fatalf("outcome = %q", result.Outcome)
	}
	if result.ImageSlices != 2 || result.ImagesPublished != 78 || result.ImagesFailed != 2 {
		t.Fatalf("image counts = %+v", result)
	}
	for _, slot := range slots {
		if r.perStage[ImageEmbedActivityName+":"+slot] != 2 {
			t.Fatalf("slot %s embedded %d times, want once per slice: %v", slot, r.perStage[ImageEmbedActivityName+":"+slot], r.perStage)
		}
	}
	if r.perStage[ImageFactsActivityName] != 2 || r.perStage[ImageOcrActivityName] != 2 || r.perStage[ImagePublishActivityName] != 2 {
		t.Fatalf("each slice runs facts, ocr and publish once: %v", r.perStage)
	}
	for _, c := range r.calls {
		if c == CommitActivityName || c == ExtractActivityName {
			t.Fatalf("a quiet catalog must not run %s", c)
		}
	}
}

func TestImageStageRunsBeforeCommitAndCanBeSkippedOrRestricted(t *testing.T) {
	r := &recorder{}
	runWithImages(t, CycleRequest{SkipSummary: true, ImageSlots: []string{"image_single"}}, r, true, 0, 1, []string{"image_maxsim", "image_single"})
	if r.perStage[ImageEmbedActivityName+":image_maxsim"] != 0 || r.perStage[ImageEmbedActivityName+":image_single"] != 1 {
		t.Fatalf("ImageSlots must restrict the slots: %v", r.perStage)
	}
	last := r.calls[len(r.calls)-1]
	if last != CommitActivityName {
		t.Fatalf("commit must stay last, got %q", last)
	}
	skipped := &recorder{}
	runWithImages(t, CycleRequest{SkipImages: true}, skipped, true, 0, 3, []string{"image_single"})
	if skipped.perStage[ImageFetchActivityName] != 0 {
		t.Fatalf("SkipImages must not run the image stage: %v", skipped.perStage)
	}
}

func TestImageLoopIsBoundedPerCycle(t *testing.T) {
	r := &recorder{}
	result := runWithImages(t, CycleRequest{ImageMaxSlices: 3}, r, false, 0, 50, []string{"image_single"})
	if result.ImageSlices != 3 {
		t.Fatalf("the loop must stop at ImageMaxSlices, ran %d", result.ImageSlices)
	}
}

// TestImageProofBudgetIsSharedAcrossSlices verifies global fencing, not only each fetch call.
// Inputs are a synthetic backlog and exact locator; output is an assertion with no external side effects.
func TestImageProofBudgetIsSharedAcrossSlices(t *testing.T) {
	r := &recorder{}
	locator := ImageLocator{Provider: "b2", Bucket: "proof", Key: "proof/one.png"}
	result := runWithImages(t, CycleRequest{Limit: 5, ImageLocators: []ImageLocator{locator}}, r, false, 0, 100, nil)
	if result.ImageSlices != 3 || r.perStage[ImageFetchActivityName] != 3 {
		t.Fatalf("five-image budget must stop after 2+2+1, got %+v", result)
	}
	for i, want := range []float64{5, 3, 1} {
		if r.imageParams[i]["max_items"] != want || r.imageParams[i]["locators"] == nil {
			t.Fatalf("scope or remaining budget missing: %+v", r.imageParams)
		}
	}
}

// TestFailedOnlyImageSlicePublishesItsLedgerWithoutEmbedding verifies terminal failure handling.
// Inputs are three rejected originals; outputs are assertions with no external side effects.
func TestFailedOnlyImageSlicePublishesItsLedgerWithoutEmbedding(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(CycleWorkflow)
	r := &recorder{}
	register(env, r, false, 0)
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (ImageFetchResult, error) {
		r.note(ImageFetchActivityName)
		return ImageFetchResult{SliceID: "rejected", Failed: 3, More: true}, nil
	}, activity.RegisterOptions{Name: ImageFetchActivityName})
	env.RegisterActivityWithOptions(func(ctx context.Context, req StageRequest) (ImagePublishResult, error) {
		r.note(ImagePublishActivityName)
		return ImagePublishResult{Failed: 3}, nil
	}, activity.RegisterOptions{Name: ImagePublishActivityName})
	env.ExecuteWorkflow(CycleWorkflow, CycleRequest{Limit: 3})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result CycleResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.ImagesFailed != 3 || r.perStage[ImagePublishActivityName] != 1 || r.perStage[ImageFetchActivityName] != 1 {
		t.Fatalf("failures must spend the budget and be ledgered once: %+v %v", result, r.calls)
	}
	if r.perStage[ImageFactsActivityName] != 0 || r.perStage[ImageOcrActivityName] != 0 {
		t.Fatalf("failed originals must not run image processing: %v", r.calls)
	}
}
