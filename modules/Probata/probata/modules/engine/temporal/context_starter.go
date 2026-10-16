package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// ContextStarter uses the existing Proffer Temporal client and task queue for context-first runs.
// Inputs: shared client and queue. Outputs: start/status methods. Effects: no independent scheduler.
// Choose for the new context route while legacy import starter behavior stays intact.
type ContextStarter struct {
	client    client.Client
	taskQueue string
}

// NewContextStarter validates the existing Temporal connection and queue.
// Inputs: shared client and task queue. Outputs: ready context starter or error.
// Effects: none; choose at the mounted starter composition boundary.
func NewContextStarter(c client.Client, taskQueue string) (*ContextStarter, error) {
	if c == nil || strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("context starter requires the shared Temporal client and task queue")
	}
	return &ContextStarter{client: c, taskQueue: taskQueue}, nil
}

// Start starts or joins an actor-bound context-first Proffer run.
// Inputs: bounded context-v1 request with deterministic workflow ID. Outputs: actual Temporal IDs.
// Effects: one Temporal start/describe; choose over the legacy parser-options start endpoint.
func (s *ContextStarter) Start(ctx context.Context, in proffer.WorkflowInput) (proffer.ContextStarted, error) {
	if in.ContextContract != proffer.ContextContractVersion || !strings.HasPrefix(in.RequestID, proffer.ContextWorkflowIDPrefix) || in.ActorSubjectUID == "" {
		return proffer.ContextStarted{}, errors.New("context starter requires an actor-bound context-v1 request")
	}
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: in.RequestID, TaskQueue: s.taskQueue,
		Memo:                     map[string]interface{}{"actor_subject_uid": in.ActorSubjectUID, "context_contract": in.ContextContract},
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, proffer.ProfferWorkflow, in)
	if err != nil {
		return proffer.ContextStarted{}, fmt.Errorf("start context source workflow: %w", err)
	}
	described, err := s.client.DescribeWorkflowExecution(ctx, run.GetID(), run.GetRunID())
	if err != nil {
		return proffer.ContextStarted{}, fmt.Errorf("verify context source workflow: %w", err)
	}
	fields := described.GetWorkflowExecutionInfo().GetMemo().GetFields()
	actor, actorErr := contextMemoField(fields, "actor_subject_uid")
	contract, contractErr := contextMemoField(fields, "context_contract")
	if actorErr != nil || contractErr != nil || actor != in.ActorSubjectUID || contract != proffer.ContextContractVersion {
		return proffer.ContextStarted{}, errors.New("context workflow ID is owned by a different request")
	}
	return proffer.ContextStarted{WorkflowID: run.GetID(), RunID: run.GetRunID()}, nil
}

// Status reads an actor-bound running query or completed result from Temporal.
// Inputs: one context workflow ID. Outputs: reference-only progress or error.
// Effects: Temporal describe/query/read; choose for context status polling, not content reads.
func (s *ContextStarter) Status(ctx context.Context, workflowID string) (proffer.ContextProgress, error) {
	if !strings.HasPrefix(workflowID, proffer.ContextWorkflowIDPrefix) || len(workflowID) > 160 {
		return proffer.ContextProgress{}, errors.New("unknown context workflow")
	}
	described, err := s.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return proffer.ContextProgress{}, err
	}
	actor, err := contextMemoField(described.GetWorkflowExecutionInfo().GetMemo().GetFields(), "actor_subject_uid")
	if err != nil {
		return proffer.ContextProgress{}, err
	}
	switch described.GetWorkflowExecutionInfo().GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		value, err := s.client.QueryWorkflow(ctx, workflowID, "", proffer.ContextStatusQueryName)
		if err != nil {
			return proffer.ContextProgress{}, err
		}
		var progress proffer.ContextProgress
		if err := value.Get(&progress); err != nil {
			return proffer.ContextProgress{}, err
		}
		if progress.ActorSubjectUID != actor {
			return proffer.ContextProgress{}, errors.New("context workflow actor mismatch")
		}
		return progress, nil
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		var result proffer.WorkflowResult
		if err := s.client.GetWorkflow(ctx, workflowID, "").Get(ctx, &result); err != nil {
			return proffer.ContextProgress{}, err
		}
		if result.Context == nil || result.Context.ActorSubjectUID != actor {
			return proffer.ContextProgress{}, errors.New("context workflow result actor mismatch")
		}
		return proffer.ContextProgress{ContextSummary: *result.Context}, nil
	default:
		return proffer.ContextProgress{ContextSummary: proffer.ContextSummary{
			ActorSubjectUID: actor, Status: "failed", Reason: "workflow_" + strings.ToLower(strings.TrimPrefix(described.GetWorkflowExecutionInfo().GetStatus().String(), "WORKFLOW_EXECUTION_STATUS_")),
		}}, nil
	}
}

// contextMemoField decodes a required existing Temporal identity memo.
// Inputs: existing contract values. Output: validated value or error. Effects: none.
// Choose at this adapter boundary rather than reading or changing source content.
func contextMemoField(fields map[string]*commonpb.Payload, key string) (string, error) {
	var value string
	if fields[key] == nil || converter.GetDefaultDataConverter().FromPayload(fields[key], &value) != nil || value == "" {
		return "", errors.New("context workflow identity memo is missing")
	}
	return value, nil
}
