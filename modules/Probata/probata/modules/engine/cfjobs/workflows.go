// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// jobSpec is what differs between the three jobs; runJob does the rest.
type jobSpec struct {
	job             Job
	activity        string
	defaultBatch    int
	maxBatch        int
	defaultParallel int
	timeout         time.Duration
	heartbeat       time.Duration
	// call builds the argument of the Worker-call Activity for one batch.
	call func(runID string, scope Scope, items []WorkItem) any
}

func retryPolicy() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:        10 * time.Second,
		BackoffCoefficient:     2,
		MaximumInterval:        5 * time.Minute,
		MaximumAttempts:        5,
		NonRetryableErrorTypes: []string{ErrorTypeBadRequest},
	}
}

func catalogOptions(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: 5 * time.Second, MaximumAttempts: 4, NonRetryableErrorTypes: []string{ErrorTypeBadRequest}},
	})
}

// runJob pages a work list out of the catalog and runs it through a Worker-call Activity, `parallel` batches at a time.
//
// With DryRun it stops after the plan. A batch whose Activity fails after its retries fails the workflow: the catalog already
// holds everything recorded so far, so starting the job again resumes where it stopped.
func runJob(ctx workflow.Context, spec jobSpec, in JobInput, ruleset string) (JobResult, error) {
	scope := in.Scope.withDefaults()
	batchSize := clampBatch(in.BatchSize, spec.defaultBatch, spec.maxBatch)
	parallel := in.Parallel
	if parallel <= 0 {
		parallel = spec.defaultParallel
	}
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	result := JobResult{RunID: runID, DryRun: in.DryRun}

	catalog := catalogOptions(ctx)
	if err := workflow.ExecuteActivity(catalog, PlanActivityName, PlanInput{
		Job: spec.job, Scope: scope, BatchSize: batchSize, Ruleset: ruleset, MaxItems: in.MaxItems,
	}).Get(ctx, &result.Plan); err != nil {
		return result, fmt.Errorf("plan %s: %w", spec.job, err)
	}
	if in.DryRun || result.Plan.Objects == 0 {
		return result, nil
	}

	work := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: spec.timeout,
		HeartbeatTimeout:    spec.heartbeat,
		RetryPolicy:         retryPolicy(),
	})
	after := ""
	remaining := in.MaxItems
	exhausted := false
	for !exhausted {
		var window [][]WorkItem
		for len(window) < parallel {
			limit := batchSize
			if in.MaxItems > 0 && remaining < limit {
				limit = remaining
			}
			if limit <= 0 {
				exhausted = true
				break
			}
			var page NextBatchResult
			if err := workflow.ExecuteActivity(catalog, NextBatchActivityName, NextBatchInput{
				Job: spec.job, Scope: scope, After: after, Limit: limit, Ruleset: ruleset,
			}).Get(ctx, &page); err != nil {
				return result, fmt.Errorf("next batch %s: %w", spec.job, err)
			}
			if len(page.Items) == 0 {
				exhausted = true
				break
			}
			after = page.Last
			remaining -= len(page.Items)
			window = append(window, page.Items)
		}
		futures := make([]workflow.Future, len(window))
		for i, items := range window {
			futures[i] = workflow.ExecuteActivity(work, spec.activity, spec.call(runID, scope, items))
		}
		for _, future := range futures {
			var counts Counts
			if err := future.Get(ctx, &counts); err != nil {
				return result, fmt.Errorf("%s batch: %w", spec.job, err)
			}
			result.Totals.add(counts)
			result.Batches++
		}
	}
	return result, nil
}

// SniffFormatsWorkflow runs the format sniffer Worker over the AI-chat candidate list in the catalog.
//
// Input: SniffInput (scope, ruleset, head size, batch size, dry-run flag). Output: the plan and the totals. Each batch of up to
// 200 objects becomes one SniffFormats Activity; results land in raw_duck.cf_format_sniff_20261002. A dry run only counts the
// list. Pick it to learn what format thousands of objects are without downloading them.
func SniffFormatsWorkflow(ctx workflow.Context, in SniffInput) (JobResult, error) {
	ruleset := in.Ruleset
	if ruleset == "" {
		ruleset = DefaultRuleset
	}
	spec := jobSpec{
		job: JobSniff, activity: SniffFormatsActivityName, defaultBatch: SniffBatchDefault, maxBatch: SniffBatchMax,
		defaultParallel: 3, timeout: sniffTimeout, heartbeat: 0,
		call: func(runID string, scope Scope, items []WorkItem) any {
			return SniffBatch{RunID: runID, Scope: scope, Ruleset: ruleset, HeadBytes: in.HeadBytes, Items: items}
		},
	}
	return runJob(ctx, spec, in.JobInput, ruleset)
}

// ListZipMembersWorkflow runs the ZIP lister Worker over the ZIP candidate list in the catalog.
//
// Input: ZipInput (scope, member cap, batch size, dry-run flag). Output: the plan and the totals, including members recorded.
// Each batch of up to 50 archives becomes one ListZipMembers Activity; results land in raw_duck.cf_zip_archives_20261002 and
// cf_zip_members_20261002. Only each archive's end and central directory are read. A dry run only counts the list. Pick it to
// see inside ZIPs without extracting them.
func ListZipMembersWorkflow(ctx workflow.Context, in ZipInput) (JobResult, error) {
	spec := jobSpec{
		job: JobZip, activity: ListZipMembersActivityName, defaultBatch: ZipBatchDefault, maxBatch: ZipBatchMax,
		defaultParallel: 3, timeout: zipTimeout, heartbeat: 2 * time.Minute,
		call: func(runID string, scope Scope, items []WorkItem) any {
			return ZipBatch{RunID: runID, Scope: scope, MaxMembers: in.MaxMembers, Items: items}
		},
	}
	return runJob(ctx, spec, in.JobInput, "")
}

// BackfillB2HashesWorkflow runs the B2 hasher Worker over the B2 objects that have no SHA-1 in the catalog listing.
//
// Input: HashInput (scope, batch size, dry-run flag). Output: the plan and the totals, including bytes hashed. Each batch
// becomes one HashB2Objects Activity (one object per call by default, since one object can be tens of gigabytes); digests land
// in raw_duck.cf_b2_hashes_20261002. A dry run only counts the list. Pick it to give every B2 object a hash before a transfer
// or a comparison relies on it.
func BackfillB2HashesWorkflow(ctx workflow.Context, in HashInput) (JobResult, error) {
	if in.Parallel <= 0 {
		in.Parallel = 1
	}
	spec := jobSpec{
		job: JobHash, activity: HashB2ObjectsActivityName, defaultBatch: HashBatchDefault, maxBatch: HashBatchMax,
		defaultParallel: 1, timeout: hashTimeout, heartbeat: hashHeartbeat,
		call: func(runID string, scope Scope, items []WorkItem) any {
			return HashBatch{RunID: runID, Scope: scope, Items: items}
		},
	}
	return runJob(ctx, spec, in.JobInput, "")
}
