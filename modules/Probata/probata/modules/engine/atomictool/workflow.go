package atomictool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Progress is the bounded, actor-bound status exposed to Workbench. The
// source-derived payload lives in ContentStore, never in workflow history.
type Progress struct {
	ActorSubjectUID string  `json:"actor_subject_uid"`
	Outcome         string  `json:"outcome"`
	Result          *Result `json:"result,omitempty"`
	Error           string  `json:"error,omitempty"`
}

// Workflow executes one pinned read-only tool Activity. Input is a validated
// request; output is bounded metadata with a content ref and audit chain head.
// It may create one ContentStore result and tool-runtime audit record.
func Workflow(ctx workflow.Context, in Request) (Progress, error) {
	progress := Progress{ActorSubjectUID: in.Actor.SubjectUID, Outcome: "running"}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	if err := in.Validate(); err != nil {
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	var result Result
	if err := workflow.ExecuteActivity(activityCtx, ActivityName, in).Get(ctx, &result); err != nil {
		progress.Outcome, progress.Error = "failed", err.Error()
		return progress, err
	}
	progress.Outcome, progress.Result = "completed", &result
	return progress, nil
}

// Started identifies the real Temporal workflow and run for a Workbench click.
type Started struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// Starter uses the existing Proffer Temporal client and task queue. It holds
// no independent scheduler, actor registry or workflow state.
type Starter struct {
	Client    client.Client
	TaskQueue string
}

// NewStarter validates the shared Temporal seam, without touching runtime.
func NewStarter(c client.Client, taskQueue string) (*Starter, error) {
	if c == nil || strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("atomic tool starter requires a Temporal client and task queue")
	}
	return &Starter{Client: c, TaskQueue: taskQueue}, nil
}

// requestDigest binds an idempotency key to the complete action input except
// that key itself. Inputs: validated request. Output: SHA-256 hex. Effects: none;
// use to reject a reused key with a different source, tool, actor or options.
func requestDigest(in Request) (string, error) {
	in.RequestID = ""
	data, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// memoField decodes one bounded Temporal memo string without trusting a client
// status. Inputs: execution memo and key. Output: value or error. Effects: none;
// use for actor-bound status and idempotency checks even after failure.
func memoField(fields map[string]*commonpb.Payload, key string) (string, error) {
	var value string
	if fields[key] == nil || converter.GetDefaultDataConverter().FromPayload(fields[key], &value) != nil || value == "" {
		return "", errors.New("atomic tool workflow identity memo is missing")
	}
	return value, nil
}

// Start starts or joins one deterministic source-backed tool workflow; its
// side effect is a Temporal execution and it returns actual workflow/run IDs.
func (s *Starter) Start(ctx context.Context, in Request) (Started, error) {
	if err := in.Validate(); err != nil {
		return Started{}, err
	}
	digest, err := requestDigest(in)
	if err != nil {
		return Started{}, err
	}
	run, err := s.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: WorkflowIDPrefix + in.RequestID, TaskQueue: s.TaskQueue,
		Memo:                     map[string]interface{}{"actor_subject_uid": in.Actor.SubjectUID, "input_digest": digest},
		WorkflowExecutionTimeout: 20 * time.Minute,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, WorkflowName, in)
	if err != nil {
		return Started{}, fmt.Errorf("start atomic tool action: %w", err)
	}
	described, err := s.Client.DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
	if err != nil {
		return Started{}, fmt.Errorf("verify atomic tool action identity: %w", err)
	}
	fields := described.GetWorkflowExecutionInfo().GetMemo().GetFields()
	storedActor, actorErr := memoField(fields, "actor_subject_uid")
	storedDigest, digestErr := memoField(fields, "input_digest")
	if actorErr != nil || digestErr != nil || storedActor != in.Actor.SubjectUID || storedDigest != digest {
		return Started{}, errors.New("atomic tool idempotency key was already used for a different action")
	}
	return Started{WorkflowID: run.GetID(), RunID: run.GetRunID()}, nil
}

// Status reads a running query or completed result from the same Temporal
// workflow; it never accepts a client-supplied status or fabricated run ID.
func (s *Starter) Status(ctx context.Context, workflowID string) (Progress, error) {
	if !strings.HasPrefix(workflowID, WorkflowIDPrefix) || len(workflowID) > 160 {
		return Progress{}, errors.New("unknown atomic tool workflow")
	}
	described, err := s.Client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return Progress{}, err
	}
	actor, err := memoField(described.GetWorkflowExecutionInfo().GetMemo().GetFields(), "actor_subject_uid")
	if err != nil {
		return Progress{}, err
	}
	switch described.GetWorkflowExecutionInfo().GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		value, err := s.Client.QueryWorkflow(ctx, workflowID, "", StatusQuery)
		if err != nil {
			return Progress{}, err
		}
		var progress Progress
		if err := value.Get(&progress); err != nil {
			return Progress{}, err
		}
		if progress.ActorSubjectUID != actor {
			return Progress{}, errors.New("atomic tool workflow actor mismatch")
		}
		return progress, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		var progress Progress
		if err := s.Client.GetWorkflow(ctx, workflowID, "").Get(ctx, &progress); err != nil {
			return Progress{}, err
		}
		if progress.ActorSubjectUID != actor {
			return Progress{}, errors.New("atomic tool workflow actor mismatch")
		}
		return progress, nil
	default:
		return Progress{ActorSubjectUID: actor, Outcome: "failed", Error: "workflow " + strings.ToLower(strings.TrimPrefix(described.GetWorkflowExecutionInfo().GetStatus().String(), "WORKFLOW_EXECUTION_STATUS_"))}, nil
	}
}
