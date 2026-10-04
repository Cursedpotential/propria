// Command superindex-schedule creates or updates the Temporal Schedule that runs the Coco Super Index cycle, and
// can trigger one cycle now or pause and resume the schedule. It starts nothing else and writes nothing else.
//
// Environment (names only; nothing here is a secret):
//
//	TEMPORAL_ADDRESS           frontend address, e.g. 100.91.190.107:7233
//	TEMPORAL_NAMESPACE         default "default"
//	TEMPORAL_TASK_QUEUE        the proffer (Go) worker's queue, where CycleWorkflow is registered
//	SUPERINDEX_INTERVAL        Go duration between watermark checks, default 15m
//
// Usage: superindex-schedule ensure | trigger | pause | resume | describe
//
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/superindex"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: superindex-schedule ensure|trigger|pause|resume|describe")
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "superindex-schedule:", err)
		os.Exit(1)
	}
}

// run applies one explicitly selected Super Index schedule command.
// Inputs: context, command and Temporal environment settings. Output: an error or a status on stdout.
// Side effects: ensure creates or updates a paused schedule; trigger, pause and resume alter its execution state.
// Pick ensure for deployment preparation and resume only after the bounded live proof succeeds.
func run(ctx context.Context, command string) error {
	address := os.Getenv("TEMPORAL_ADDRESS")
	if address == "" {
		return fmt.Errorf("TEMPORAL_ADDRESS is required")
	}
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	c, err := client.Dial(client.Options{HostPort: address, Namespace: namespace})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer c.Close()

	handle := c.ScheduleClient().GetHandle(ctx, superindex.ScheduleID)
	switch command {
	case "ensure":
		queue := os.Getenv("TEMPORAL_TASK_QUEUE")
		interval := superindex.DefaultInterval
		if raw := os.Getenv("SUPERINDEX_INTERVAL"); raw != "" {
			if interval, err = time.ParseDuration(raw); err != nil {
				return fmt.Errorf("SUPERINDEX_INTERVAL: %w", err)
			}
		}
		outcome, err := superindex.EnsureSchedule(ctx, c, superindex.ScheduleConfig{
			WorkflowTaskQueue: queue,
			Interval:          interval,
			Paused:            true,
		})
		if err != nil {
			return err
		}
		fmt.Println("schedule", superindex.ScheduleID, outcome)
	case "trigger":
		if err := handle.Trigger(ctx, client.ScheduleTriggerOptions{}); err != nil {
			return err
		}
		fmt.Println("triggered one cycle now")
	case "pause":
		if err := handle.Pause(ctx, client.SchedulePauseOptions{Note: "paused by superindex-schedule"}); err != nil {
			return err
		}
		fmt.Println("paused")
	case "resume":
		if err := handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: "resumed by superindex-schedule"}); err != nil {
			return err
		}
		fmt.Println("resumed")
	case "describe":
		description, err := handle.Describe(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("paused=%v recent_actions=%d next=%v\n", description.Schedule.State.Paused,
			len(description.Info.RecentActions), description.Info.NextActionTimes)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	return nil
}
