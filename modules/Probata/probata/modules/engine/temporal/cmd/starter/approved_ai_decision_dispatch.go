// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"

	"github.com/Cursedpotential/probata/engine/approvedaigraphprojectionflow"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// approvedAIWorkflowClient is the existing Temporal client's narrow enqueue/readback seam.
// Inputs: start options or execution identity. Outputs: actual execution metadata.
// Effects: Temporal RPCs only. Choose instead of creating a second client or scheduler.
type approvedAIWorkflowClient interface {
	ExecuteWorkflow(context.Context, client.StartWorkflowOptions, interface{}, ...interface{}) (client.WorkflowRun, error)
	DescribeWorkflowExecution(context.Context, string, string) (*workflowservice.DescribeWorkflowExecutionResponse, error)
}

type approvedAIProjectionDispatcher struct {
	client    approvedAIWorkflowClient
	taskQueue string
}

type extractionWithApprovedAIDispatch struct {
	runtimeapi.ExtractionWorkflows
	*approvedAIProjectionDispatcher
}

// withApprovedAIProjectionDispatch attaches the optional approved graph seam to existing extraction workflows.
// Inputs: existing extraction starter, Temporal client/queue and existing APPROVED_CONTEXT_GRAPH_ENABLED flag.
// Outputs: the original interface with optional dispatch capability. Effects: construction only.
// Choose in entityExtractionHandler; disabled deployments retain a committed/pending approval response.
func withApprovedAIProjectionDispatch(existing runtimeapi.ExtractionWorkflows, c approvedAIWorkflowClient, taskQueue string) (runtimeapi.ExtractionWorkflows, error) {
	if existing == nil {
		return nil, errors.New("approved AI projection requires existing extraction workflows")
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APPROVED_CONTEXT_GRAPH_ENABLED")), "true") {
		return existing, nil
	}
	if c == nil || strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("approved AI projection requires the existing Temporal client and task queue")
	}
	return &extractionWithApprovedAIDispatch{ExtractionWorkflows: existing, approvedAIProjectionDispatcher: &approvedAIProjectionDispatcher{client: c, taskQueue: strings.TrimSpace(taskQueue)}}, nil
}

// approvedAIScopeDigest binds a deterministic execution to every exact source and review selector.
// Inputs: validated Scope. Outputs: stable SHA256. Effects: none.
// Choose for Temporal identity memo verification when an idempotent retry joins an execution.
func approvedAIScopeDigest(scope approvedgraphai.Scope) string {
	raw, _ := json.Marshal(scope)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// StartApprovedAIProjection starts or joins the exact workflow for an already committed native AI approval.
// Inputs: canonical scope from the successful decision receipt. Outputs: verified actual workflow/run IDs.
// Effects: one Temporal start/join and identity readback; no source, review or graph writes here.
// Choose after approval; already-completed executions are joined without starting a duplicate projection.
func (s *approvedAIProjectionDispatcher) StartApprovedAIProjection(ctx context.Context, scope approvedgraphai.Scope) (runtimeapi.ApprovedAIProjectionStarted, error) {
	if s == nil || s.client == nil || s.taskQueue == "" {
		return runtimeapi.ApprovedAIProjectionStarted{}, errors.New("approved AI projection dispatcher is unavailable")
	}
	id, err := approvedgraphai.WorkflowID(scope)
	if err != nil {
		return runtimeapi.ApprovedAIProjectionStarted{}, err
	}
	digest := approvedAIScopeDigest(scope)
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: id, TaskQueue: s.taskQueue,
		Memo:                     map[string]interface{}{"approved_ai_scope_digest": digest},
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, approvedaigraphprojectionflow.WorkflowName, scope)
	runID := ""
	if err != nil {
		var duplicate *serviceerror.WorkflowExecutionAlreadyStarted
		if !errors.As(err, &duplicate) {
			return runtimeapi.ApprovedAIProjectionStarted{}, err
		}
		runID = duplicate.RunId
	} else {
		if run == nil || run.GetID() != id || run.GetRunID() == "" {
			return runtimeapi.ApprovedAIProjectionStarted{}, errors.New("approved AI projection returned an invalid execution identity")
		}
		runID = run.GetRunID()
	}
	described, err := s.client.DescribeWorkflowExecution(ctx, id, runID)
	if err != nil {
		return runtimeapi.ApprovedAIProjectionStarted{}, err
	}
	info := described.GetWorkflowExecutionInfo()
	execution := info.GetExecution()
	var storedDigest string
	payload := info.GetMemo().GetFields()["approved_ai_scope_digest"]
	if payload == nil || converter.GetDefaultDataConverter().FromPayload(payload, &storedDigest) != nil || storedDigest != digest || execution.GetWorkflowId() != id || execution.GetRunId() == "" || (runID != "" && execution.GetRunId() != runID) {
		return runtimeapi.ApprovedAIProjectionStarted{}, errors.New("approved AI projection execution does not match the committed decision")
	}
	return runtimeapi.ApprovedAIProjectionStarted{WorkflowID: id, RunID: execution.GetRunId()}, nil
}
