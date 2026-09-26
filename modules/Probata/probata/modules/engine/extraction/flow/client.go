// Byline: Claude Code · Opus 5.5 · 2026-09-25

package flow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// Starter starts the extraction workflows on the Proffer task queue and reads
// their progress back. The starter process holds no workflow state.
type Starter struct {
	Client    client.Client
	TaskQueue string
}

// NewStarter validates the Temporal seam.
func NewStarter(c client.Client, taskQueue string) (*Starter, error) {
	if c == nil || strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("extraction starter requires a Temporal client and task queue")
	}
	return &Starter{Client: c, TaskQueue: taskQueue}, nil
}

// ExtractionWorkflowID is stable per click: a retried request joins the
// running workflow instead of starting a second one.
func ExtractionWorkflowID(request ExtractionRequest) string {
	return "entity-extraction:" + request.Run.PreviewHandle + ":" + request.ExtractionID
}

// CommitWorkflowID is stable per validated proposal set.
func CommitWorkflowID(previewHandle, digest string) string {
	short := digest
	if len(short) > 24 {
		short = short[:24]
	}
	return "extraction-commit:" + previewHandle + ":" + short
}

// Started identifies one workflow execution.
type Started struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

func (s *Starter) start(ctx context.Context, id string, timeout time.Duration, workflowName string, input any) (Started, error) {
	run, err := s.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       id,
		TaskQueue:                s.TaskQueue,
		WorkflowExecutionTimeout: timeout,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, workflowName, input)
	if err != nil {
		return Started{}, fmt.Errorf("start %s: %w", workflowName, err)
	}
	return Started{WorkflowID: run.GetID(), RunID: run.GetRunID()}, nil
}

// StartExtraction starts (or joins) EntityExtractionWorkflow.
func (s *Starter) StartExtraction(ctx context.Context, request ExtractionRequest) (Started, error) {
	return s.start(ctx, ExtractionWorkflowID(request), 6*time.Hour, ExtractionWorkflowName, request)
}

// StartCommit starts (or joins) ExtractionCommitWorkflow.
func (s *Starter) StartCommit(ctx context.Context, request CommitRequest) (Started, error) {
	request.WorkflowID = CommitWorkflowID(request.Run.PreviewHandle, request.Digest)
	return s.start(ctx, request.WorkflowID, 2*time.Hour, CommitWorkflowName, request)
}

// Status reads a workflow's progress: the live query while it runs, its
// result once it completed, and the failure when it did not.
func (s *Starter) Status(ctx context.Context, workflowID string) (Progress, error) {
	described, err := s.Client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return Progress{}, fmt.Errorf("describe %s: %w", workflowID, err)
	}
	info := described.GetWorkflowExecutionInfo()
	switch info.GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		value, err := s.Client.QueryWorkflow(ctx, workflowID, "", StatusQuery)
		if err != nil {
			return Progress{Outcome: OutcomeRunning}, nil
		}
		var progress Progress
		if err := value.Get(&progress); err != nil {
			return Progress{}, err
		}
		return progress, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		var progress Progress
		if err := s.Client.GetWorkflow(ctx, workflowID, "").Get(ctx, &progress); err != nil {
			return Progress{}, err
		}
		return progress, nil
	default:
		return Progress{Outcome: OutcomeFailed, Steps: []StepResult{{
			Step: "workflow", Status: StepFailed,
			Detail: "workflow " + strings.ToLower(strings.TrimPrefix(info.GetStatus().String(), "WORKFLOW_EXECUTION_STATUS_")),
		}}}, nil
	}
}
