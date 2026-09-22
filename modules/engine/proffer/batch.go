// Byline: Claude Code · Opus 5 · 2026-09-21
//
// BatchWorkflow imports every object under one folder, one at a time.
//
// Owner 2026-09-20 23:52: "it gets batched by folder. Being careful not to
// overload any systems and process them one at a time ... batching is
// absolutely part of this." Build order step 4 in
// docs/decisions/2026-09-20-bulk-intake-owner-requirements.md.
//
// Why a workflow and not a goroutine in the HTTP start handler: a handler
// goroutine dies on redeploy and takes the rest of the folder with it. The
// sequencing, the in-flight bound, the skip rule and the per-item failure
// record are workflow state, so a redeploy resumes exactly where it stopped.
//
// Each item is an ordinary child ProfferWorkflow with its own preview
// binding, so it appears in Review exactly like a hand-started run. The child
// is started with an abandon close policy on purpose: an item parked at a
// human gate must outlive the batch that started it.

package proffer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// BatchWorkflowName is the exact Temporal name the worker registers.
	BatchWorkflowName = "proffer_batch_workflow"
	// BatchStatusQueryName returns per-item status and counts.
	BatchStatusQueryName = "batch_status"

	// ListBatchFolderActivityName and friends are spelled here rather than
	// imported: proffer must not depend on the activities package.
	listBatchFolderActivityName     = "list_batch_folder_activity"
	bindImportOperationActivityName = "bind_import_operation_activity"
	readImportOperationActivityName = "read_import_operation_activity"
	findImportBindingsActivityName  = "find_import_bindings_activity"
)

const (
	// maxBatchItems bounds one batch run. A folder with more members than
	// this is imported by more than one batch; the status says so rather than
	// silently importing a prefix of it.
	maxBatchItems = 2000
	// maxReportedBatchItems bounds the query response so a status read stays
	// small. Counts always cover every item.
	maxReportedBatchItems = 500
	// batchItemPollInterval is how often a running item's lifecycle is read.
	batchItemPollInterval = 15 * time.Second
	// batchItemPollLimit stops an item holding its slot forever when durable
	// state never becomes readable. The run itself is not cancelled.
	batchItemPollLimit = 8 * time.Hour
)

// BatchItemStatus is one item's position in the batch.
type BatchItemStatus string

const (
	BatchItemQueued        BatchItemStatus = "queued"
	BatchItemRunning       BatchItemStatus = "running"
	BatchItemWaitingOnGate BatchItemStatus = "waiting_on_gate"
	BatchItemDone          BatchItemStatus = "done"
	BatchItemFailed        BatchItemStatus = "failed"
	BatchItemSkipped       BatchItemStatus = "skipped"
)

// BatchInput is what the start-batch endpoint hands the workflow. Every field
// is a reference, an identifier or a small scalar: the folder's objects are
// never carried here.
type BatchInput struct {
	BatchID     string `json:"batch_id"`
	MatterID    string `json:"matter_id"`
	CourtCaseID string `json:"court_case_id"`

	// Scheme, Bucket and Prefix are the validated folder locator. Prefix ends
	// in "/" and lies inside a configured source root; the HTTP boundary has
	// already proved that, exactly as it does for a single start.
	Scheme string `json:"scheme"`
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`

	DeclaredFormat   string `json:"declared_format"`
	ParserOptionsRef Ref    `json:"parser_options_ref"`
	SourceContextRef Ref    `json:"source_context_ref,omitempty"`

	// MaxInFlight is how many items may run at once. One is the default and
	// the owner's instruction; more only when a system is proven to take it.
	MaxInFlight int `json:"max_in_flight,omitempty"`
}

func (in BatchInput) validate() error {
	for name, value := range map[string]string{
		"batch_id": in.BatchID, "matter_id": in.MatterID, "court_case_id": in.CourtCaseID,
		"scheme": in.Scheme, "bucket": in.Bucket, "prefix": in.Prefix,
		"declared_format": in.DeclaredFormat, "parser_options_ref": string(in.ParserOptionsRef),
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("batch input requires %s", name)
		}
	}
	if !strings.HasSuffix(in.Prefix, "/") {
		return errors.New("batch input prefix must name a folder and end in /")
	}
	if in.MaxInFlight < 0 {
		return errors.New("batch max_in_flight cannot be negative")
	}
	return nil
}

// BatchItem is one object's outcome, by reference.
type BatchItem struct {
	Key           string          `json:"key"`
	SourceRef     Ref             `json:"source_ref"`
	RequestID     string          `json:"request_id"`
	PreviewHandle string          `json:"preview_handle,omitempty"`
	Status        BatchItemStatus `json:"status"`
	Reason        string          `json:"reason,omitempty"`
}

// BatchCounts is every item, counted. It always covers the whole batch even
// when the reported item list is truncated.
type BatchCounts struct {
	Total         int `json:"total"`
	Queued        int `json:"queued"`
	Running       int `json:"running"`
	WaitingOnGate int `json:"waiting_on_gate"`
	Done          int `json:"done"`
	Failed        int `json:"failed"`
	Skipped       int `json:"skipped"`
}

// BatchStatus is BatchStatusQueryName's response and the workflow's result.
type BatchStatus struct {
	BatchID  string `json:"batch_id"`
	Prefix   string `json:"prefix"`
	Terminal bool   `json:"terminal"`
	// ListingTruncated is true when the folder holds more members than one
	// batch run imports.
	ListingTruncated bool `json:"listing_truncated"`
	// ItemsTruncated is true when more items ran than the status reports.
	ItemsTruncated bool        `json:"items_truncated"`
	Counts         BatchCounts `json:"counts"`
	// Items is never nil: an empty batch encodes as [].
	Items []BatchItem `json:"items"`
}

// batchState is the workflow's mutable bookkeeping. Only the workflow
// goroutine and its item goroutines touch it, and Temporal runs them on one
// thread, so no lock is needed.
type batchState struct {
	status  BatchStatus
	byIndex map[int]*BatchItem
	order   []int
	running int
}

func (s *batchState) snapshot() BatchStatus {
	out := s.status
	out.Counts = BatchCounts{Total: len(s.order)}
	out.Items = make([]BatchItem, 0, len(s.order))
	for _, index := range s.order {
		item := *s.byIndex[index]
		switch item.Status {
		case BatchItemQueued:
			out.Counts.Queued++
		case BatchItemRunning:
			out.Counts.Running++
		case BatchItemWaitingOnGate:
			out.Counts.WaitingOnGate++
		case BatchItemDone:
			out.Counts.Done++
		case BatchItemFailed:
			out.Counts.Failed++
		case BatchItemSkipped:
			out.Counts.Skipped++
		}
		if len(out.Items) < maxReportedBatchItems {
			out.Items = append(out.Items, item)
		}
	}
	out.ItemsTruncated = len(s.order) > len(out.Items)
	return out
}

// BatchWorkflow enumerates the folder page by page and imports each object.
//
// A failure never stops the batch: the item is recorded failed and the next
// one starts. Re-running the same batch is safe — an object whose prior run
// already completed is skipped rather than imported twice.
func BatchWorkflow(ctx workflow.Context, in BatchInput) (BatchStatus, error) {
	state := &batchState{
		status:  BatchStatus{BatchID: in.BatchID, Prefix: in.Prefix, Items: []BatchItem{}},
		byIndex: map[int]*BatchItem{},
	}
	if err := workflow.SetQueryHandler(ctx, BatchStatusQueryName, func() (BatchStatus, error) {
		return state.snapshot(), nil
	}); err != nil {
		return BatchStatus{}, err
	}
	if err := in.validate(); err != nil {
		return BatchStatus{}, temporal.NewNonRetryableApplicationError(err.Error(), "invalid_batch_input", err)
	}
	maxInFlight := in.MaxInFlight
	if maxInFlight < 1 {
		maxInFlight = 1
	}

	listCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         retryPolicy(2*time.Second, 5),
	})
	cursor := ""
	index := 0
	for {
		var page struct {
			Keys       []string `json:"keys"`
			NextCursor string   `json:"next_cursor,omitempty"`
		}
		request := map[string]any{"scheme": in.Scheme, "bucket": in.Bucket, "prefix": in.Prefix, "cursor": cursor}
		if err := workflow.ExecuteActivity(listCtx, listBatchFolderActivityName, request).Get(ctx, &page); err != nil {
			return state.snapshot(), fmt.Errorf("proffer batch: list %s: %w", in.Prefix, err)
		}
		for _, key := range page.Keys {
			if index >= maxBatchItems {
				state.status.ListingTruncated = true
				break
			}
			itemIndex := index
			index++
			item := &BatchItem{
				Key:       key,
				SourceRef: Ref(fmt.Sprintf("%s://%s/%s", in.Scheme, in.Bucket, key)),
				RequestID: fmt.Sprintf("%s-%05d", in.BatchID, itemIndex),
				Status:    BatchItemQueued,
			}
			state.byIndex[itemIndex] = item
			state.order = append(state.order, itemIndex)

			// Bounded in-flight: the next item starts when a running one
			// finishes OR parks at a human gate, never before.
			if err := workflow.Await(ctx, func() bool { return state.running < maxInFlight }); err != nil {
				return state.snapshot(), err
			}
			state.running++
			workflow.Go(ctx, func(itemCtx workflow.Context) {
				defer func() { state.running-- }()
				runBatchItem(itemCtx, in, item)
			})
		}
		if state.status.ListingTruncated || page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	// A workflow that completes while an item goroutine is still blocked
	// abandons it, so wait for every slot to be released first.
	if err := workflow.Await(ctx, func() bool { return state.running == 0 }); err != nil {
		return state.snapshot(), err
	}
	state.status.Terminal = true
	return state.snapshot(), nil
}

// runBatchItem imports one object. It records an outcome for every path and
// returns as soon as the item no longer needs its in-flight slot.
func runBatchItem(ctx workflow.Context, in BatchInput, item *BatchItem) {
	shortCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy:         retryPolicy(2*time.Second, 4),
	})

	// Idempotency: an object whose prior run already completed is skipped, so
	// re-running the same batch is safe.
	var prior struct {
		Bindings []struct {
			PreviewHandle string `json:"preview_handle"`
			RequestID     string `json:"request_id"`
			WorkflowID    string `json:"workflow_id"`
		} `json:"bindings"`
	}
	if err := workflow.ExecuteActivity(shortCtx, findImportBindingsActivityName,
		map[string]any{"source_ref": string(item.SourceRef)}).Get(ctx, &prior); err == nil {
		for _, binding := range prior.Bindings {
			var state struct {
				Lifecycle string `json:"lifecycle"`
				Terminal  bool   `json:"terminal"`
				Available bool   `json:"available"`
			}
			if readErr := workflow.ExecuteActivity(shortCtx, readImportOperationActivityName,
				map[string]any{"workflow_id": binding.WorkflowID}).Get(ctx, &state); readErr != nil {
				continue
			}
			if state.Available && state.Terminal && state.Lifecycle == string(OperationCompleted) {
				item.Status = BatchItemSkipped
				item.PreviewHandle = binding.PreviewHandle
				item.Reason = "an earlier run of this exact source already completed"
				return
			}
		}
	}

	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: item.RequestID,
		// An item parked at a human gate must outlive the batch that started
		// it; the owner answers the gate later, in Review.
		ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_ABANDON,
	})
	child := workflow.ExecuteChildWorkflow(childCtx, ProfferWorkflow, WorkflowInput{
		RequestID: item.RequestID, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID,
		SourceRef: item.SourceRef, DeclaredFormat: in.DeclaredFormat,
		ParserOptionsRef: in.ParserOptionsRef, SourceContextRef: in.SourceContextRef,
	})

	var execution workflow.Execution
	if err := child.GetChildWorkflowExecution().Get(ctx, &execution); err != nil {
		item.Status, item.Reason = BatchItemFailed, err.Error()
		return
	}
	item.Status = BatchItemRunning

	// The binding is what makes this item visible in Review; without it the
	// run is durable but unaddressable, so a binding failure fails the item.
	var bound struct {
		PreviewHandle string `json:"preview_handle"`
	}
	if err := workflow.ExecuteActivity(shortCtx, bindImportOperationActivityName, map[string]any{
		"request_id": item.RequestID, "source_ref": string(item.SourceRef),
		"workflow_id": execution.ID, "run_id": execution.RunID,
		"parser_options_ref": string(in.ParserOptionsRef),
	}).Get(ctx, &bound); err != nil {
		item.Status, item.Reason = BatchItemFailed, "preview binding failed: "+err.Error()
		return
	}
	item.PreviewHandle = bound.PreviewHandle

	awaitBatchItem(ctx, shortCtx, child, execution, item)
}

// awaitBatchItem holds the item's slot until the run finishes or parks at a
// human gate. It never cancels the run.
func awaitBatchItem(
	ctx workflow.Context,
	activityCtx workflow.Context,
	child workflow.ChildWorkflowFuture,
	execution workflow.Execution,
	item *BatchItem,
) {
	deadline := workflow.Now(ctx).Add(batchItemPollLimit)
	for {
		timer := workflow.NewTimer(ctx, batchItemPollInterval)
		selector := workflow.NewSelector(ctx)
		finished := false
		var childErr error
		selector.AddFuture(child, func(f workflow.Future) {
			finished = true
			childErr = f.Get(ctx, nil)
		})
		selector.AddFuture(timer, func(workflow.Future) {})
		selector.Select(ctx)
		if finished {
			if childErr != nil {
				item.Status, item.Reason = BatchItemFailed, childErr.Error()
				return
			}
			item.Status, item.Reason = BatchItemDone, ""
			return
		}

		var state struct {
			Lifecycle string `json:"lifecycle"`
			Wait      string `json:"wait,omitempty"`
			Terminal  bool   `json:"terminal"`
			Reason    string `json:"reason,omitempty"`
			Available bool   `json:"available"`
		}
		if err := workflow.ExecuteActivity(activityCtx, readImportOperationActivityName,
			map[string]any{"workflow_id": execution.ID}).Get(ctx, &state); err == nil && state.Available {
			if strings.TrimSpace(state.Wait) != "" {
				// The owner answers this gate in Review; the slot is released
				// now so the rest of the folder keeps moving.
				item.Status, item.Reason = BatchItemWaitingOnGate, state.Wait
				return
			}
			if state.Terminal {
				if state.Lifecycle == string(OperationCompleted) {
					item.Status, item.Reason = BatchItemDone, ""
				} else {
					item.Status, item.Reason = BatchItemFailed, state.Reason
				}
				return
			}
		}
		if workflow.Now(ctx).After(deadline) {
			item.Status = BatchItemWaitingOnGate
			item.Reason = "the run is still active but its state could not be followed; it continues in Review"
			return
		}
	}
}
