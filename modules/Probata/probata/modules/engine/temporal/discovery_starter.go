// Byline: Codex · GPT-6 · 2026-10-11
package temporal

import (
	"context"
	"errors"
	"fmt"

	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/Cursedpotential/probata/engine/discoveryflow"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// DiscoveryStarter dispatches the exact saved PG feedback revision on the existing Go queue.
// Inputs: configured Temporal client and queue. Outputs: reference-only execution identity.
// Effects: starts optional discovery; it never starts or changes the intake workflow.
type DiscoveryStarter struct {
	client    client.Client
	taskQueue string
}

// NewDiscoveryStarter builds a dispatcher without creating a separate worker or queue.
// Inputs: existing Temporal connection and Go task queue. Outputs: dispatcher.
// Effects: none. Choose after canonical discovery persistence has been configured.
func NewDiscoveryStarter(c client.Client, taskQueue string) (*DiscoveryStarter, error) {
	if c == nil || taskQueue == "" {
		return nil, errors.New("discovery dispatcher requires Temporal client and queue")
	}
	return &DiscoveryStarter{client: c, taskQueue: taskQueue}, nil
}

// Start makes repeated attempts for the same saved revision converge on one workflow.
// Inputs: persisted PG run. Outputs: workflow and Temporal run IDs. Effects: dispatch.
// Choose after a run/correction commit; durable pending state handles retry after outages.
func (s *DiscoveryStarter) Start(ctx context.Context, run discovery.Run) (string, string, error) {
	id := fmt.Sprintf("discovery-%s-f%d", run.Request.RunID, run.FeedbackRevision)
	input := discoveryflow.Request{RunID: run.Request.RunID, Scope: run.Request.Scope, FeedbackRevision: run.FeedbackRevision}
	execution, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: id, TaskQueue: s.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	}, discoveryflow.WorkflowName, input)
	if err != nil {
		var exists *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &exists) && exists.RunId != "" {
			return id, exists.RunId, nil
		}
		return "", "", err
	}
	return execution.GetID(), execution.GetRunID(), nil
}
