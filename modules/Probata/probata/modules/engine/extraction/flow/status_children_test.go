package flow

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"testing"
)

type childStatusClient struct {
	client.Client
	t         *testing.T
	snapshots map[string]*workflowservice.DescribeWorkflowExecutionResponse
	results   map[string]Progress
	failures  map[string]error
	events    []*historypb.HistoryEvent
	reads     map[string]int
}

// DescribeWorkflowExecution supplies actual-shaped server snapshots and asserts recorded child run pinning.
func (c *childStatusClient) DescribeWorkflowExecution(_ context.Context, id, run string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	s, ok := c.snapshots[id]
	if !ok {
		return nil, errors.New("unknown execution")
	}
	if id != "parent" {
		require.Equal(c.t, s.GetWorkflowExecutionInfo().GetExecution().GetRunId(), run)
	}
	c.reads[id]++
	return s, nil
}

// QueryWorkflow supplies current child/parent progress, refusing unpinned query execution coordinates.
func (c *childStatusClient) QueryWorkflow(_ context.Context, id, run, name string, args ...interface{}) (converter.EncodedValue, error) {
	require.Equal(c.t, c.snapshots[id].GetWorkflowExecutionInfo().GetExecution().GetRunId(), run)
	return childStatusValue{value: c.results[id]}, nil
}

// GetWorkflow supplies terminal result or typed failure for exactly the recorded child run.
func (c *childStatusClient) GetWorkflow(_ context.Context, id, run string) client.WorkflowRun {
	require.Equal(c.t, c.snapshots[id].GetWorkflowExecutionInfo().GetExecution().GetRunId(), run)
	return childStatusRun{value: c.results[id], err: c.failures[id]}
}

// GetWorkflowHistory supplies only recorded parent events; no child IDs or source coordinates are manufactured by the adapter.
func (c *childStatusClient) GetWorkflowHistory(_ context.Context, id, run string, _ bool, _ enumspb.HistoryEventFilterType) client.HistoryEventIterator {
	require.Equal(c.t, "parent", id)
	require.Equal(c.t, "parent-run", run)
	return &childStatusHistory{events: c.events}
}

type childStatusValue struct {
	converter.EncodedValue
	value Progress
}

// Get decodes the actual-shaped queried result without mutating fixture rows.
func (v childStatusValue) Get(out interface{}) error {
	payloads, _ := converter.GetDefaultDataConverter().ToPayloads(v.value)
	return converter.GetDefaultDataConverter().FromPayloads(payloads, out)
}

type childStatusRun struct {
	client.WorkflowRun
	value Progress
	err   error
}

// Get returns the terminal result or actual typed application failure.
func (r childStatusRun) Get(_ context.Context, out interface{}) error {
	if r.err != nil {
		return r.err
	}
	return childStatusValue{value: r.value}.Get(out)
}

type childStatusHistory struct {
	events []*historypb.HistoryEvent
	index  int
}

// HasNext bounds the fixture's history iteration.
func (h *childStatusHistory) HasNext() bool { return h.index < len(h.events) }

// Next returns the next recorded event, preserving its actual IDs and payloads.
func (h *childStatusHistory) Next() (*historypb.HistoryEvent, error) {
	e := h.events[h.index]
	h.index++
	return e, nil
}

// childSnapshot creates actual-shaped Describe data for a pinned test execution; it performs no service calls.
func childSnapshot(id, run, name string, status enumspb.WorkflowExecutionStatus) *workflowservice.DescribeWorkflowExecutionResponse {
	return &workflowservice.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{Execution: &commonpb.WorkflowExecution{WorkflowId: id, RunId: run}, Type: &commonpb.WorkflowType{Name: name}, Status: status}}
}

// childEvents records the child initiation input and server-started execution, independent of naming conventions.
func childEvents(id, run, name string, input any, eventID int64) []*historypb.HistoryEvent {
	payloads, _ := converter.GetDefaultDataConverter().ToPayloads(input)
	return []*historypb.HistoryEvent{{EventId: eventID, Attributes: &historypb.HistoryEvent_StartChildWorkflowExecutionInitiatedEventAttributes{StartChildWorkflowExecutionInitiatedEventAttributes: &historypb.StartChildWorkflowExecutionInitiatedEventAttributes{WorkflowId: id, WorkflowType: &commonpb.WorkflowType{Name: name}, Input: payloads}}}, {EventId: eventID + 1, Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionStartedEventAttributes{ChildWorkflowExecutionStartedEventAttributes: &historypb.ChildWorkflowExecutionStartedEventAttributes{InitiatedEventId: eventID, WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: id, RunId: run}}}}}
}

// TestStarterOverlaysClosedSiblingsWhileParentWaitsForFirst verifies immediate real child status without workflow changes.
// Inputs: blocked parent, running Go child, closed successful Semantica and closed failed LangExtract with recorded inputs.
// Outputs: exactly the existing rows with pinned IDs, actual results/retries/failure details, and parent still running.
// Effects: client fixtures only; choose for the sequential future Get regression.
func TestStarterOverlaysClosedSiblingsWhileParentWaitsForFirst(t *testing.T) {
	pin := RunRef{PreviewHandle: "handle", SourceVersionID: "version", GenerationID: "generation", MatterMode: "LIVE"}
	c := &childStatusClient{t: t, snapshots: map[string]*workflowservice.DescribeWorkflowExecutionResponse{}, results: map[string]Progress{}, failures: map[string]error{}, reads: map[string]int{}}
	c.snapshots["parent"] = childSnapshot("parent", "parent-run", RequestWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
	c.results["parent"] = Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "opaque-row-go", Conversation: "display only", Extractor: DefaultExtractorID, Status: StepRunning, WorkflowID: "recorded-go", RunID: "recorded-go-run"}, {Step: "opaque-row-sem", Conversation: "display only", Extractor: "semantica", Status: StepRunning}, {Step: "opaque-row-lang", Conversation: "display only", Extractor: "langextract", Status: StepRunning}}}
	c.snapshots["recorded-go"] = childSnapshot("recorded-go", "recorded-go-run", ExtractionWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
	c.snapshots["recorded-go"].PendingActivities = []*workflowpb.PendingActivityInfo{{ActivityId: "13", ActivityType: &commonpb.ActivityType{Name: ExtractModelActivity}, Attempt: 2, LastFailure: &failurepb.Failure{Message: "actual timeout"}}}
	c.results["recorded-go"] = Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "rules", Status: StepCompleted}, {Step: "model", Status: StepRunning}}}
	c.snapshots["recorded-sem"] = childSnapshot("recorded-sem", "recorded-sem-run", ExternalWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
	c.results["recorded-sem"] = Progress{Outcome: OutcomeCompleted, Steps: []StepResult{{Step: "extract", Status: StepCompleted, Detail: "183 stored entities", Counts: map[string]int{"entities": 183}}}}
	c.snapshots["recorded-lang"] = childSnapshot("recorded-lang", "recorded-lang-run", ExternalWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_FAILED)
	failed := Progress{Outcome: OutcomeFailed, Steps: []StepResult{{Step: "extract", Status: StepFailed, Detail: "actual exhausted failure"}}}
	c.failures["recorded-lang"] = temporal.NewNonRetryableApplicationError("exhausted", "StandaloneExtractionFailed", nil, failed)
	c.events = append(c.events, childEvents("recorded-go", "recorded-go-run", ExtractionWorkflowName, ExtractionRequest{Run: pin}, 10)...)
	c.events = append(c.events, childEvents("recorded-sem", "recorded-sem-run", ExternalWorkflowName, ExternalRunInput{Run: pin, Extractor: "semantica"}, 20)...)
	c.events = append(c.events, childEvents("recorded-lang", "recorded-lang-run", ExternalWorkflowName, ExternalRunInput{Run: pin, Extractor: "langextract"}, 30)...)
	starter, _ := NewStarter(c, "unused-read-only")
	out, err := starter.Status(context.Background(), "parent")
	require.NoError(t, err)
	require.Len(t, out.Steps, 3)
	require.Equal(t, OutcomeRunning, out.Outcome)
	require.Equal(t, "retrying", out.Steps[0].Status)
	require.EqualValues(t, 2, out.Steps[0].Attempt)
	require.Equal(t, "actual timeout", out.Steps[0].LastFailure)
	require.Equal(t, StepCompleted, out.Steps[1].Status)
	require.Equal(t, 183, out.Steps[1].Counts["entities"])
	require.Equal(t, "recorded-sem", out.Steps[1].WorkflowID)
	require.Equal(t, "recorded-sem-run", out.Steps[1].RunID)
	require.Equal(t, StepFailed, out.Steps[2].Status)
	require.Contains(t, out.Steps[2].Detail, "actual exhausted failure")
	require.Equal(t, "recorded-lang-run", out.Steps[2].RunID)
}

// TestChildStatusCorrelationRefusesUnknownOrAmbiguousSources verifies source identity cannot be guessed from display text.
// Inputs: source mismatch, duplicate anchors, malformed child linkage. Outputs: refusal with no fabricated row or ID.
// Effects: none; choose for source/version and multi-conversation regression.
func TestChildStatusCorrelationRefusesUnknownOrAmbiguousSources(t *testing.T) {
	pin := RunRef{PreviewHandle: "handle", SourceVersionID: "version", GenerationID: "generation"}
	goChild := recordedExtractionChild{workflowID: "real-go", runID: "run-go", extractor: DefaultExtractorID, source: pin}
	sem := recordedExtractionChild{workflowID: "real-sem", runID: "run-sem", extractor: "semantica", source: pin}
	p := Progress{Steps: []StepResult{{WorkflowID: "real-go", Conversation: "same display", Extractor: DefaultExtractorID}, {Conversation: "same display", Extractor: "semantica"}}}
	require.Equal(t, 1, childProgressRow(p, []recordedExtractionChild{goChild, sem}, sem))
	sem.source.SourceVersionID = "other-version"
	require.Equal(t, -1, childProgressRow(p, []recordedExtractionChild{goChild, sem}, sem))
	sem.source = pin
	duplicate := recordedExtractionChild{workflowID: "another-actual-child", runID: "another-run", extractor: "langextract", source: pin}
	p.Steps = append(p.Steps, StepResult{WorkflowID: duplicate.workflowID, Conversation: "different display", Extractor: "langextract"})
	require.Equal(t, -1, childProgressRow(p, []recordedExtractionChild{goChild, duplicate, sem}, sem))
	events := childEvents("real-sem", "run-sem", ExternalWorkflowName, ExternalRunInput{Run: pin, Extractor: "semantica"}, 2)
	events[1].GetChildWorkflowExecutionStartedEventAttributes().WorkflowExecution.WorkflowId = "different-unverified-child"
	require.Empty(t, recordedExtractionChildren(events))
}

// TestChildStatusHistoryLimitIsExplicit bounds reads and retains the authoritative parent business outcome.
// Inputs: parent history exceeding the fixed observation limit. Outputs: explicit limitation and no child reads.
// Effects: fake Temporal reads only; choose to prevent unbounded status polling.
func TestChildStatusHistoryLimitIsExplicit(t *testing.T) {
	c := &childStatusClient{t: t, snapshots: map[string]*workflowservice.DescribeWorkflowExecutionResponse{}, reads: map[string]int{}}
	d := childSnapshot("parent", "parent-run", RequestWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
	for i := 0; i <= childStatusHistoryLimit; i++ {
		c.events = append(c.events, &historypb.HistoryEvent{EventId: int64(i + 1)})
	}
	s, _ := NewStarter(c, "unused")
	out := s.overlayRecordedExtractionChildren(context.Background(), Progress{Outcome: OutcomeRunning}, d)
	require.Equal(t, OutcomeRunning, out.Outcome)
	require.Len(t, out.Steps, 1)
	require.Contains(t, out.Steps[0].Detail, "history limit reached")
	require.Empty(t, c.reads)
}
