// Byline: Claude Code · Opus 5 · 2026-09-21
//
// The Temporal client side of batch-by-folder intake. It is a separate seam
// from WorkflowStarter so the single-start surface and its test doubles are
// untouched.

package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// BatchStarter starts one folder batch and reads its status.
type BatchStarter struct {
	client    client.Client
	taskQueue string
}

// NewBatchStarter wraps an already-dialed client; it does not own its lifecycle.
func NewBatchStarter(c client.Client, taskQueue string) (*BatchStarter, error) {
	if c == nil {
		return nil, errors.New("temporal: batch starter requires a Temporal client")
	}
	if strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("temporal: batch starter requires a task queue")
	}
	return &BatchStarter{client: c, taskQueue: taskQueue}, nil
}

// StartBatch uses the batch id as the Temporal workflow ID, so re-submitting
// the same batch joins the existing run instead of starting a second one.
func (s *BatchStarter) StartBatch(ctx context.Context, in proffer.BatchInput) (string, string, error) {
	if strings.TrimSpace(in.BatchID) == "" {
		return "", "", errors.New("temporal: batch_id is required to start a batch")
	}
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        in.BatchID,
		TaskQueue: s.taskQueue,
	}, proffer.BatchWorkflowName, in)
	if err != nil {
		if runID, ok := alreadyStartedRunID(err); ok {
			return in.BatchID, runID, nil
		}
		return "", "", fmt.Errorf("temporal: start proffer batch workflow: %w", err)
	}
	return run.GetID(), run.GetRunID(), nil
}

// BatchStatus queries the running batch for per-item status and counts.
func (s *BatchStarter) BatchStatus(ctx context.Context, batchID string) (proffer.BatchStatus, error) {
	if strings.TrimSpace(batchID) == "" {
		return proffer.BatchStatus{}, errors.New("temporal: batch_id is required to query batch status")
	}
	value, err := s.client.QueryWorkflow(ctx, batchID, "", proffer.BatchStatusQueryName)
	if err != nil {
		return proffer.BatchStatus{}, fmt.Errorf("temporal: query batch status: %w", err)
	}
	var status proffer.BatchStatus
	if err := value.Get(&status); err != nil {
		return proffer.BatchStatus{}, fmt.Errorf("temporal: decode batch status: %w", err)
	}
	return status, nil
}
