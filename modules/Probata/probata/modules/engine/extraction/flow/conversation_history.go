package flow

import (
	"context"
	"fmt"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
)

// ConversationAttempt is an exact retained parent execution and its existing status.
// Inputs: recorded RequestInput and server execution identity. Output: bounded metadata.
// Effects: none; choose for failed attempts that created no extraction output rows.
type ConversationAttempt struct {
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id"`
	Extractors []string `json:"extractors"`
	Progress   Progress `json:"progress"`
}

// ConversationHistory states the bounded visibility/retention coverage of attempts.
// Inputs: Temporal visibility scan. Output: matching executions and honest coverage.
// Effects: none; never equate no retained attempts with never attempted.
type ConversationHistory struct {
	Available bool                  `json:"available"`
	Attempts  []ConversationAttempt `json:"attempts"`
	Truncated bool                  `json:"truncated"`
	Scanned   int                   `json:"scanned"`
	Coverage  string                `json:"coverage"`
}

// ConversationHistory reads retained parent inputs and matches exact target coordinates.
// Inputs: approved matter and one conversation. Output: up to 20 pinned statuses from
// at most 100 visible executions. Effects: bounded Temporal reads only; no dispatch.
func (s *Starter) ConversationHistory(ctx context.Context, matter string, ref ConversationRef) (ConversationHistory, error) {
	out := ConversationHistory{Available: true, Attempts: []ConversationAttempt{}, Coverage: "Retained Temporal parent requests only; older expired history and automatic run-only requests are not covered"}
	page, err := s.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{PageSize: 100, Query: fmt.Sprintf("WorkflowType = '%s'", RequestWorkflowName)})
	if err != nil {
		return out, err
	}
	out.Truncated = len(page.GetNextPageToken()) > 0
	for _, info := range page.GetExecutions() {
		if out.Scanned >= 100 {
			out.Truncated = true
			break
		}
		out.Scanned++
		execution := info.GetExecution()
		iterator := s.Client.GetWorkflowHistory(ctx, execution.GetWorkflowId(), execution.GetRunId(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		if !iterator.HasNext() {
			continue
		}
		event, err := iterator.Next()
		if err != nil {
			out.Truncated = true
			continue
		}
		started := event.GetWorkflowExecutionStartedEventAttributes()
		if started == nil {
			continue
		}
		var request RequestInput
		if converter.GetDefaultDataConverter().FromPayloads(started.GetInput(), &request) != nil || request.MatterID != matter {
			continue
		}
		matched := false
		for _, target := range append(request.Conversations, request.RequestedConversations...) {
			if target == ref {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if len(out.Attempts) >= 20 {
			out.Truncated = true
			break
		}
		status, err := s.statusForExecution(ctx, execution.GetWorkflowId(), execution.GetRunId())
		if err != nil {
			status = Progress{Outcome: "unavailable", Steps: []StepResult{{Step: "status", Status: StepFailed, Detail: "Retained attempt status unavailable"}}}
		}
		out.Attempts = append(out.Attempts, ConversationAttempt{WorkflowID: execution.GetWorkflowId(), RunID: execution.GetRunId(), Extractors: request.Extractors, Progress: status})
	}
	return out, nil
}
