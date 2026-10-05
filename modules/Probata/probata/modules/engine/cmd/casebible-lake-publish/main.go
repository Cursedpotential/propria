// Byline: Codex, 2026-10-04. Tracked catalog publication runner.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// publication sequences independently tracked server-side catalog publication stages.
// Inputs: none; outputs: generation; effects: invokes bounded stages on a dedicated queue.
// Choose for the B2 Parquet projection, not source ingestion or custody.
func publication(ctx workflow.Context) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 4 * time.Hour, HeartbeatTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}})
	for _, stage := range []string{"export", "schema", "upload", "readback", "finalize"} {
		if err := workflow.ExecuteActivity(ctx, "CaseBibleLakeStage", stage).Get(ctx, nil); err != nil {
			return "", err
		}
	}
	return "2026-10-04", nil
}

// stage runs one allowlisted publication operation and heartbeats its bounded identity.
// Inputs: operation name; outputs: success/error; effects: the named server-side publisher phase.
// Choose as the publication Workflow's Activity; shell text is never supplied by the caller.
func stage(ctx context.Context, name string) error {
	switch name {
	case "export", "schema", "upload", "readback", "finalize":
	default:
		return fmt.Errorf("invalid stage")
	}
	cmd := exec.CommandContext(ctx, "bash", "/data/consignatio/lake-publish-20261004/lake_publish_20261004.sh", name)
	cmd.Env = os.Environ()
	log, err := os.OpenFile("/data/consignatio/lake-publish-20261004/"+name+".activity.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd.Stdout = log
	cmd.Stderr = log
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx, name)
			}
		}
	}()
	if err := cmd.Run(); err != nil {
		return err
	}
	if _, err := os.Stat("/data/consignatio/lake-publish-20261004/markers/" + name + ".done"); err != nil {
		return fmt.Errorf("stage %s has no success receipt: %w", name, err)
	}
	return nil
}

// main hosts the existing engine SDK publication Workflow on ovh-files and waits for its receipt.
// Inputs: trusted process environment; outputs: Temporal run/result; effects: worker polling and publication start.
// Choose for an explicitly authorized one-shot publication until deployed worker wiring is complete.
func main() {
	c, err := client.Dial(client.Options{HostPort: os.Getenv("TEMPORAL_HOST_PORT")})
	if err != nil {
		panic(err)
	}
	defer c.Close()
	w := worker.New(c, "casebible-lake-publication", worker.Options{MaxConcurrentActivityExecutionSize: 1})
	w.RegisterWorkflowWithOptions(publication, workflow.RegisterOptions{Name: "CaseBibleLakePublication"})
	w.RegisterActivityWithOptions(stage, activity.RegisterOptions{Name: "CaseBibleLakeStage"})
	if err = w.Start(); err != nil {
		panic(err)
	}
	defer w.Stop()
	run, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{ID: "casebible-lake-20261004", TaskQueue: "casebible-lake-publication"}, "CaseBibleLakePublication")
	if err != nil {
		panic(err)
	}
	fmt.Println("workflow", run.GetID(), "run", run.GetRunID())
	var result string
	if err = run.Get(context.Background(), &result); err != nil {
		panic(err)
	}
	fmt.Println("verified", result)
}
