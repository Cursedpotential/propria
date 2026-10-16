package flow

import (
	"context"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"testing"
)

type retainedHistoryClient struct {
	*childStatusClient
	inputs map[string]RequestInput
}

// ListWorkflow supplies bounded retained parents and unrelated controls.
// Inputs: visibility request. Output: synthetic executions; effects: none.
func (c *retainedHistoryClient) ListWorkflow(_ context.Context, in *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	require.Equal(c.t, int32(100), in.PageSize)
	require.Equal(c.t, "WorkflowType = 'extraction_request_workflow'", in.Query)
	out := &workflowservice.ListWorkflowExecutionsResponse{NextPageToken: []byte("more")}
	for _, id := range []string{"other-matter", "other-conversation", "matched"} {
		out.Executions = append(out.Executions, &workflowpb.WorkflowExecutionInfo{Execution: &commonpb.WorkflowExecution{WorkflowId: id, RunId: id + "-run"}})
	}
	return out, nil
}

// GetWorkflowHistory returns exact stored input and asserts execution pinning.
// Inputs: execution IDs. Output: one start event; no effects or output rows.
func (c *retainedHistoryClient) GetWorkflowHistory(_ context.Context, id, run string, _ bool, _ enumspb.HistoryEventFilterType) client.HistoryEventIterator {
	require.Equal(c.t, id+"-run", run)
	payloads, _ := converter.GetDefaultDataConverter().ToPayloads(c.inputs[id])
	return &childStatusHistory{events: []*historypb.HistoryEvent{{Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{Input: payloads}}}}}
}

// TestConversationHistoryPinsFailedParentsAndExcludesUnrelatedTargets proves reopen history.
// Inputs: failed parent without output rows and unrelated controls. Output: exact
// execution and truthful truncation. Effects: fake reads only, never workflow starts.
func TestConversationHistoryPinsFailedParentsAndExcludesUnrelatedTargets(t *testing.T) {
	ref := ConversationRef{ExportKey: "fixture.xml", Conv: "five-messages"}
	base := &childStatusClient{t: t, snapshots: map[string]*workflowservice.DescribeWorkflowExecutionResponse{"matched": childSnapshot("matched", "matched-run", RequestWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)}, results: map[string]Progress{"matched": {Outcome: OutcomeFailed, Steps: []StepResult{{Step: "rules", Status: StepFailed, Detail: "operation has no verified operating_mode"}}}}, reads: map[string]int{}}
	fixture := &retainedHistoryClient{childStatusClient: base, inputs: map[string]RequestInput{
		"other-matter":       {MatterID: "other", Conversations: []ConversationRef{ref}},
		"other-conversation": {MatterID: "matter", Conversations: []ConversationRef{{ExportKey: "fixture.xml", Conv: "control"}}},
		"matched":            {MatterID: "matter", Conversations: []ConversationRef{ref}, Extractors: []string{"semantica"}},
	}}
	starter := &Starter{Client: fixture}
	out, err := starter.ConversationHistory(context.Background(), "matter", ref)
	require.NoError(t, err)
	require.True(t, out.Truncated)
	require.Len(t, out.Attempts, 1)
	require.Equal(t, "matched-run", out.Attempts[0].RunID)
	require.Equal(t, OutcomeFailed, out.Attempts[0].Progress.Outcome)
	require.Equal(t, 1, base.reads["matched"])
	require.Zero(t, base.reads["other-matter"])
}
