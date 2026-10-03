// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Register registers the three cf-jobs workflows and all their Activities on a worker.
//
// Use it on a worker that polls QueueName and nothing else: the Activities of one job set must not be split across workers
// of the same queue.
func Register(r worker.Registry, a *Activities) {
	r.RegisterWorkflowWithOptions(SniffFormatsWorkflow, workflow.RegisterOptions{Name: SniffFormatsWorkflowName})
	r.RegisterWorkflowWithOptions(ListZipMembersWorkflow, workflow.RegisterOptions{Name: ListZipMembersWorkflowName})
	r.RegisterWorkflowWithOptions(BackfillB2HashesWorkflow, workflow.RegisterOptions{Name: BackfillB2HashesWorkflowName})
	r.RegisterActivityWithOptions(a.NextBatch, activity.RegisterOptions{Name: NextBatchActivityName})
	r.RegisterActivityWithOptions(a.Plan, activity.RegisterOptions{Name: PlanActivityName})
	r.RegisterActivityWithOptions(a.SniffFormats, activity.RegisterOptions{Name: SniffFormatsActivityName})
	r.RegisterActivityWithOptions(a.ListZipMembers, activity.RegisterOptions{Name: ListZipMembersActivityName})
	r.RegisterActivityWithOptions(a.HashB2Objects, activity.RegisterOptions{Name: HashB2ObjectsActivityName})
}
