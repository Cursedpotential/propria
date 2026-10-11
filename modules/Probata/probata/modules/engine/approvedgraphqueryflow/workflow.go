// Byline: Codex · GPT-6 · 2026-10-08.
package approvedgraphqueryflow

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

// Progress exposes actor-bound status and an exact-version result reference without claim bodies.
type Progress struct {
	ActorSubjectUID string  `json:"actor_subject_uid"`
	Outcome         string  `json:"outcome"`
	Result          *Result `json:"result,omitempty"`
	Error           string  `json:"error,omitempty"`
}

// Workflow executes one revision-scoped approved graph query Activity.
// Inputs: validated actor, projection and perspective. Outputs: bounded status with an object-store ref.
// Effects: one graph read and versioned CaseVault derivative through the Activity.
// Pick for approved context pages, not candidate review, graph projection or raw-turn retrieval.
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

// Started identifies the real Temporal execution for one actor-bound query page.
type Started struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// Starter uses the existing Temporal client and task queue without a second scheduler.
type Starter struct {
	Client    client.Client
	TaskQueue string
}

// NewStarter validates the shared Temporal seam without starting a workflow.
// Inputs: existing Temporal client and queue. Outputs: starter or error. Effects: none.
// Pick in root-owned composition for approved context queries.
func NewStarter(c client.Client, taskQueue string) (*Starter, error) {
	if c == nil || strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("approved graph query starter requires a Temporal client and task queue")
	}
	return &Starter{Client: c, TaskQueue: taskQueue}, nil
}

func requestDigest(in Request) (string, error) {
	in.RequestID = ""
	data, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func memoField(fields map[string]*commonpb.Payload, key string) (string, error) {
	var value string
	if fields[key] == nil || converter.GetDefaultDataConverter().FromPayload(fields[key], &value) != nil || value == "" {
		return "", errors.New("approved graph query workflow identity memo is missing")
	}
	return value, nil
}

// Start starts or joins one idempotent query only when the actor and full input digest match.
// Inputs: validated Request. Outputs: actual Temporal workflow/run IDs. Effects: Temporal start.
// Pick for actor-scoped Workbench query actions; never trust a caller-supplied workflow status.
func (s *Starter) Start(ctx context.Context, in Request) (Started, error) {
	if s == nil || s.Client == nil || strings.TrimSpace(s.TaskQueue) == "" {
		return Started{}, errors.New("approved graph query starter is unavailable")
	}
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
		return Started{}, fmt.Errorf("start approved graph query: %w", err)
	}
	described, err := s.Client.DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
	if err != nil {
		return Started{}, fmt.Errorf("verify approved graph query identity: %w", err)
	}
	fields := described.GetWorkflowExecutionInfo().GetMemo().GetFields()
	storedActor, actorErr := memoField(fields, "actor_subject_uid")
	storedDigest, digestErr := memoField(fields, "input_digest")
	if actorErr != nil || digestErr != nil || storedActor != in.Actor.SubjectUID || storedDigest != digest {
		return Started{}, errors.New("approved graph query idempotency key was already used for a different request")
	}
	return Started{WorkflowID: run.GetID(), RunID: run.GetRunID()}, nil
}

// Status reads the real workflow state and returns bounded metadata for caller actor checking.
// Inputs: workflow ID. Outputs: Progress with actor identity and optional result ref. Effects: read-only Temporal I/O.
// Pick for root-owned actor-bound status routes, not direct claim-body retrieval.
func (s *Starter) Status(ctx context.Context, workflowID string) (Progress, error) {
	if s == nil || s.Client == nil || !strings.HasPrefix(workflowID, WorkflowIDPrefix) || len(workflowID) > 160 {
		return Progress{}, errors.New("unknown approved graph query workflow")
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
			return Progress{}, errors.New("approved graph query workflow actor mismatch")
		}
		return progress, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		var progress Progress
		if err := s.Client.GetWorkflow(ctx, workflowID, "").Get(ctx, &progress); err != nil {
			return Progress{}, err
		}
		if progress.ActorSubjectUID != actor {
			return Progress{}, errors.New("approved graph query workflow actor mismatch")
		}
		return progress, nil
	default:
		return Progress{ActorSubjectUID: actor, Outcome: "failed", Error: "workflow " + strings.ToLower(strings.TrimPrefix(described.GetWorkflowExecutionInfo().GetStatus().String(), "WORKFLOW_EXECUTION_STATUS_"))}, nil
	}
}
