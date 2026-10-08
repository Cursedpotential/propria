package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"

	"github.com/Cursedpotential/probata/engine/approvedaigraphprojectionflow"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"github.com/stretchr/testify/require"
)

type aiDispatchTestRun struct {
	client.WorkflowRun
	id, runID string
}

// GetID returns the fixture workflow identity without Temporal I/O.
// Inputs: none. Outputs: test ID. Effects: none. Choose for dispatcher tests.
func (r aiDispatchTestRun) GetID() string { return r.id }

// GetRunID returns the fixture execution identity without Temporal I/O.
// Inputs: none. Outputs: test run ID. Effects: none. Choose for dispatcher tests.
func (r aiDispatchTestRun) GetRunID() string { return r.runID }

type aiDispatchTestClient struct {
	options      []client.StartWorkflowOptions
	inputs       []approvedgraphai.Scope
	workflowName interface{}
	startErr     error
	badMemo      bool
}

// ExecuteWorkflow captures one start attempt and returns a fixture execution or injected failure.
// Inputs: options, workflow name and Scope. Outputs: stub run. Effects: test recorder only.
// Choose to assert the existing queue and deterministic retry identity without scheduling work.
func (c *aiDispatchTestClient) ExecuteWorkflow(_ context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
	c.options = append(c.options, options)
	c.inputs = append(c.inputs, args[0].(approvedgraphai.Scope))
	c.workflowName = workflow
	if c.startErr != nil {
		return nil, c.startErr
	}
	return aiDispatchTestRun{id: options.ID, runID: "test-run"}, nil
}

// DescribeWorkflowExecution returns the actual-like identity and matching scope memo for the captured request.
// Inputs: expected workflow and run IDs. Outputs: bounded fixture metadata. Effects: no I/O.
// Choose to test identity verification, including rejection of mismatched existing executions.
func (c *aiDispatchTestClient) DescribeWorkflowExecution(_ context.Context, id, _ string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	digest := approvedAIScopeDigest(c.inputs[len(c.inputs)-1])
	if c.badMemo {
		digest = "different"
	}
	payload, err := converter.GetDefaultDataConverter().ToPayload(digest)
	if err != nil {
		return nil, err
	}
	return &workflowservice.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{Execution: &commonpb.WorkflowExecution{WorkflowId: id, RunId: "test-run"}, Memo: &commonpb.Memo{Fields: map[string]*commonpb.Payload{"approved_ai_scope_digest": payload}}}}, nil
}

// TestApprovedAIDispatchUsesExistingQueueAndStableExecution checks retry and completed-duplicate semantics.
// Inputs: synthetic unit scope and a stub client. Outputs: assertions. Effects: no I/O.
// Choose for the post-approval composition without constructing another Temporal client.
func TestApprovedAIDispatchUsesExistingQueueAndStableExecution(t *testing.T) {
	scope := approvedgraphai.Scope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, SourceVersionID: "11111111-1111-4111-8111-111111111111", CandidateID: "22222222-2222-4222-8222-222222222222", DecisionID: "33333333-3333-4333-8333-333333333333", RequestDigest: strings.Repeat("a", 64)}
	c := &aiDispatchTestClient{startErr: errors.New("transport unavailable")}
	s := &approvedAIProjectionDispatcher{client: c, taskQueue: "proffer-v1"}
	_, err := s.StartApprovedAIProjection(context.Background(), scope)
	require.Error(t, err)
	c.startErr = nil
	started, err := s.StartApprovedAIProjection(context.Background(), scope)
	require.NoError(t, err)
	require.Equal(t, c.options[0].ID, started.WorkflowID)
	require.Equal(t, c.options[0], c.options[1])
	require.Equal(t, "proffer-v1", c.options[1].TaskQueue)
	require.Equal(t, enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE, c.options[1].WorkflowIDReusePolicy)
	require.Equal(t, enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING, c.options[1].WorkflowIDConflictPolicy)
	require.Equal(t, approvedaigraphprojectionflow.WorkflowName, c.workflowName)
	require.Equal(t, scope, c.inputs[1])
	c.startErr = serviceerror.NewWorkflowExecutionAlreadyStarted("already completed", "request", "test-run")
	joined, err := s.StartApprovedAIProjection(context.Background(), scope)
	require.NoError(t, err)
	require.Equal(t, started, joined)
	c.badMemo = true
	_, err = s.StartApprovedAIProjection(context.Background(), scope)
	require.ErrorContains(t, err, "does not match")
}

type aiDispatchExtractionStub struct{ runtimeapi.ExtractionWorkflows }

// TestApprovedAIDispatchCompositionIsOptional checks the existing feature flag and shared client admission.
// Inputs: enabled/disabled configuration and existing workflow interface. Outputs: assertions.
// Effects: test environment only. Choose to keep legacy extraction composition compatible.
func TestApprovedAIDispatchCompositionIsOptional(t *testing.T) {
	existing := &aiDispatchExtractionStub{}
	t.Setenv("APPROVED_CONTEXT_GRAPH_ENABLED", "false")
	disabled, err := withApprovedAIProjectionDispatch(existing, nil, "")
	require.NoError(t, err)
	require.Same(t, existing, disabled)
	t.Setenv("APPROVED_CONTEXT_GRAPH_ENABLED", "true")
	_, err = withApprovedAIProjectionDispatch(existing, nil, "proffer-v1")
	require.Error(t, err)
	enabled, err := withApprovedAIProjectionDispatch(existing, &aiDispatchTestClient{}, "proffer-v1")
	require.NoError(t, err)
	_, ok := enabled.(runtimeapi.ApprovedAIProjectionDispatcher)
	require.True(t, ok)
}
