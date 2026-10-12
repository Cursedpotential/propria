package flow

import (
	"context"
	"fmt"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"time"
)

const childStatusHistoryLimit = 512
const childStatusReadLimit = 16

type recordedExtractionChild struct {
	workflowID, runID, extractor string
	source                       RunRef
}

// childSourceKey compares recorded source coordinates, never parsed display labels or constructed workflow IDs.
// Inputs: child RunRef. Output: comparable source/version/generation triple. Effects: none; pick for sibling correlation.
func childSourceKey(run RunRef) [3]string {
	return [3]string{run.PreviewHandle, run.SourceVersionID, run.GenerationID}
}

// decodeRecordedChild reads the source-pinned input on a real child initiation event.
// Inputs: Temporal initiation attributes. Output: recorded workflow ID, extractor and complete source coordinates.
// Effects: none; unknown types or incomplete source identities are refused rather than guessed.
func decodeRecordedChild(a *historypb.StartChildWorkflowExecutionInitiatedEventAttributes) (recordedExtractionChild, bool) {
	child := recordedExtractionChild{workflowID: a.GetWorkflowId()}
	switch a.GetWorkflowType().GetName() {
	case ExtractionWorkflowName, "EntityExtractionWorkflow":
		var in ExtractionRequest
		if converter.GetDefaultDataConverter().FromPayloads(a.GetInput(), &in) != nil {
			return child, false
		}
		child.source, child.extractor = in.Run, DefaultExtractorID
	case ExternalWorkflowName:
		var in ExternalRunInput
		if converter.GetDefaultDataConverter().FromPayloads(a.GetInput(), &in) != nil {
			return child, false
		}
		spec, ok := ExtractorByID(in.Extractor)
		if !ok || spec.Kind != KindExternal {
			return child, false
		}
		child.source, child.extractor = in.Run, in.Extractor
	default:
		return child, false
	}
	key := childSourceKey(child.source)
	return child, child.workflowID != "" && key[0] != "" && key[1] != "" && key[2] != ""
}

// recordedExtractionChildren links initiation inputs to server-assigned child run IDs.
// Inputs: bounded parent history. Output: only actually started source-pinned children, including closed children.
// Effects: none; initiation IDs and workflow IDs must agree, so pending children and naming conventions cannot fabricate identity.
func recordedExtractionChildren(events []*historypb.HistoryEvent) []recordedExtractionChild {
	initiated := map[int64]recordedExtractionChild{}
	children := []recordedExtractionChild{}
	for _, event := range events {
		if a := event.GetStartChildWorkflowExecutionInitiatedEventAttributes(); a != nil {
			if child, ok := decodeRecordedChild(a); ok {
				initiated[event.GetEventId()] = child
			}
		}
		if a := event.GetChildWorkflowExecutionStartedEventAttributes(); a != nil {
			child, ok := initiated[a.GetInitiatedEventId()]
			if !ok || a.GetWorkflowExecution().GetWorkflowId() != child.workflowID || a.GetWorkflowExecution().GetRunId() == "" {
				continue
			}
			child.runID = a.GetWorkflowExecution().GetRunId()
			children = append(children, child)
		}
	}
	return children
}

// childProgressRow finds one existing row using an actual child ID or a source-equal anchored sibling.
// Inputs: recorded children and queried parent rows. Output: unique row index or refusal.
// Effects: none; labels only correlate existing rows after identity is proven by history, never supply source coordinates.
func childProgressRow(progress Progress, children []recordedExtractionChild, child recordedExtractionChild) int {
	exact := -1
	for i, row := range progress.Steps {
		if row.WorkflowID == child.workflowID {
			if exact >= 0 {
				return -1
			}
			exact = i
		}
	}
	if exact >= 0 {
		return exact
	}
	anchors := map[string]bool{}
	for _, sibling := range children {
		if childSourceKey(sibling.source) != childSourceKey(child.source) {
			continue
		}
		for _, row := range progress.Steps {
			if row.WorkflowID == sibling.workflowID && row.Conversation != "" {
				anchors[row.Conversation] = true
			}
		}
	}
	if len(anchors) != 1 {
		return -1
	}
	match := -1
	for i, row := range progress.Steps {
		if anchors[row.Conversation] && row.Extractor == child.extractor {
			if match >= 0 {
				return -1
			}
			match = i
		}
	}
	return match
}

// overlayChildProgress replaces one anchored row with actual child query/result semantics and retry metadata.
// Inputs: existing row, pinned execution, actual child Progress. Output: updated row preserving parent display coordinates.
// Effects: none; running child stage completion cannot mark its whole extractor complete.
func overlayChildProgress(row StepResult, child recordedExtractionChild, result Progress) StepResult {
	row.WorkflowID, row.RunID = child.workflowID, child.runID
	row.ActivityID, row.ActivityType, row.Attempt, row.LastFailure = "", "", 0, ""
	if result.Outcome == OutcomeRunning || result.Outcome == "" {
		row.Status = StepRunning
		for _, step := range result.Steps {
			if step.Attempt > 0 {
				row.ActivityID, row.ActivityType, row.Attempt, row.LastFailure = step.ActivityID, step.ActivityType, step.Attempt, step.LastFailure
				row.Detail = step.Detail
				if step.Status == "retrying" {
					row.Status = "retrying"
				}
				break
			}
		}
		return row
	}
	row.Status, row.Detail = childStatus(result)
	row.Counts = sumCounts(result)
	return row
}

// overlayRecordedExtractionChildren reads closed and running siblings without waiting for the parent's sequential Get.
// Inputs: actual parent execution and queried rows. Output: bounded observational overlay; parent outcome remains authoritative.
// Effects: at most 512 history events and 16 child status reads within five seconds; no worker, queue, database or history writes.
func (s *Starter) overlayRecordedExtractionChildren(ctx context.Context, progress Progress, described *workflowservice.DescribeWorkflowExecutionResponse) Progress {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	execution := described.GetWorkflowExecutionInfo().GetExecution()
	iter := s.Client.GetWorkflowHistory(ctx, execution.GetWorkflowId(), execution.GetRunId(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	events := []*historypb.HistoryEvent{}
	for len(events) < childStatusHistoryLimit && iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return progress
		}
		events = append(events, event)
	}
	if iter.HasNext() {
		return childStatusObservationLimit(progress, "child status history limit reached; original parent progress retained")
	}
	children := recordedExtractionChildren(events)
	if len(children) > childStatusReadLimit {
		return childStatusObservationLimit(progress, fmt.Sprintf("child status read limit %d reached; original parent progress retained", childStatusReadLimit))
	}
	for _, child := range children {
		index := childProgressRow(progress, children, child)
		if index < 0 {
			continue
		}
		result, err := s.statusForExecution(ctx, child.workflowID, child.runID)
		if err != nil {
			continue
		}
		progress.Steps[index] = overlayChildProgress(progress.Steps[index], child, result)
	}
	return progress
}

// childStatusObservationLimit exposes a bounded-read limitation without rewriting business outcome or inventing child rows.
// Inputs: unchanged parent progress and limit reason. Output: one explicit observation note. Effects: none.
func childStatusObservationLimit(progress Progress, reason string) Progress {
	progress.Steps = append(progress.Steps, StepResult{Step: "status_observation", Status: StepSkipped, Detail: reason})
	return progress
}
