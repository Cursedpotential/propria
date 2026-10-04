package superindex

// The Temporal Schedule that makes the Super Index automatic. One schedule, one workflow per tick, overlap SKIP: a
// cycle still running when the next tick arrives is left alone, and a tick that finds the catalog unchanged ends in
// one cheap Activity (discover) with outcome "no_change". Every tick, changed or not, is a workflow run in Temporal
// history with its Activities, counts and heartbeats.
//
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

import (
	"context"
	"errors"
	"fmt"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const (
	// ScheduleID is the one schedule.
	ScheduleID = "superindex-cycle"
	// DefaultInterval is how often the catalog watermark is checked.
	DefaultInterval = 15 * time.Minute
)

// ScheduleConfig names where the workflow runs. WorkflowTaskQueue is the Go worker's queue (TEMPORAL_TASK_QUEUE of the
// proffer worker that registers CycleWorkflow); the Activities run on TaskQueue ("superindex").
type ScheduleConfig struct {
	WorkflowTaskQueue string
	Interval          time.Duration
	Paused            bool
	Request           CycleRequest
}

func (c ScheduleConfig) validate() error {
	if c.WorkflowTaskQueue == "" {
		return errors.New("superindex: the workflow task queue is required")
	}
	if c.Interval != 0 && c.Interval < time.Minute {
		return fmt.Errorf("superindex: interval %s is shorter than a minute", c.Interval)
	}
	if c.Request.Limit > 0 || c.Request.PathPrefix != "" || c.Request.ImageLocators != nil {
		return errors.New("superindex: a scheduled cycle must not be a bounded proof run")
	}
	return nil
}

func scheduleOptions(c ScheduleConfig) client.ScheduleOptions {
	interval := c.Interval
	if interval == 0 {
		interval = DefaultInterval
	}
	return client.ScheduleOptions{
		ID: ScheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: interval}},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:                       "superindex-cycle",
			Workflow:                 WorkflowName,
			Args:                     []any{c.Request},
			TaskQueue:                c.WorkflowTaskQueue,
			WorkflowExecutionTimeout: 7 * 24 * time.Hour,
		},
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
		Paused:  c.Paused,
		Note:    "Coco Super Index cycle: discover, extract, summarize, embed, publish, commit",
	}
}

// EnsureSchedule creates the schedule, or updates its spec and action when it exists. It reports what it did.
func EnsureSchedule(ctx context.Context, c client.Client, cfg ScheduleConfig) (string, error) {
	if err := cfg.validate(); err != nil {
		return "", err
	}
	options := scheduleOptions(cfg)
	_, err := c.ScheduleClient().Create(ctx, options)
	if err == nil {
		return "created", nil
	}
	if !errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return "", fmt.Errorf("superindex: create schedule: %w", err)
	}
	handle := c.ScheduleClient().GetHandle(ctx, ScheduleID)
	err = handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			in.Description.Schedule.Spec = &options.Spec
			in.Description.Schedule.Action = options.Action
			return &client.ScheduleUpdate{Schedule: &in.Description.Schedule}, nil
		},
	})
	if err != nil {
		return "", fmt.Errorf("superindex: update schedule: %w", err)
	}
	return "updated", nil
}
