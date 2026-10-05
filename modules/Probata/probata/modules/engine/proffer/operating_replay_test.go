// Byline: Codex · GPT-5 · 2026-10-05.
package proffer

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// These synthetic pre-upgrade histories contain neither operating-mode input
// nor the new version marker. The SDK replayer must still issue the recorded
// first command, rather than failing at the new entry admission fence.
// Pending Activity payload admission is tested separately in profferworker.
func TestOperatingFenceReplaysLegacySingleAndBatchFirstCommand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    interface{}
		fn       interface{}
		activity string
	}{
		{"ProfferWorkflow", testInput(), ProfferWorkflow, "register_source_activity"},
		{BatchWorkflowName, batchInput(), BatchWorkflow, listBatchFolderActivityName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.input)
			require.NoError(t, err)
			var legacy map[string]interface{}
			require.NoError(t, json.Unmarshal(raw, &legacy))
			delete(legacy, "OperatingMode")
			delete(legacy, "operating_mode")
			input, err := converter.GetDefaultDataConverter().ToPayloads(legacy)
			require.NoError(t, err)
			now := timestamppb.New(time.Unix(1700000000, 0))
			event := func(id int64, kind enumspb.EventType) *historypb.HistoryEvent {
				return &historypb.HistoryEvent{EventId: id, EventType: kind, EventTime: now}
			}
			started := event(1, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED)
			started.Attributes = &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				WorkflowType: &commonpb.WorkflowType{Name: tc.name}, TaskQueue: &taskqueuepb.TaskQueue{Name: "legacy-import"}, Input: input,
				WorkflowTaskTimeout: durationpb.New(time.Minute),
			}}
			scheduled := event(2, enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED)
			scheduled.Attributes = &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{TaskQueue: &taskqueuepb.TaskQueue{Name: "legacy-import"}, StartToCloseTimeout: durationpb.New(time.Minute)}}
			taskStarted := event(3, enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED)
			taskStarted.Attributes = &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{ScheduledEventId: 2, Identity: "legacy-worker"}}
			completed := event(4, enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
			completed.Attributes = &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{ScheduledEventId: 2, StartedEventId: 3, Identity: "legacy-worker"}}
			activity := event(5, enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED)
			activity.Attributes = &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
				ActivityId: "1", ActivityType: &commonpb.ActivityType{Name: tc.activity}, TaskQueue: &taskqueuepb.TaskQueue{Name: "legacy-import"},
				WorkflowTaskCompletedEventId: 4, StartToCloseTimeout: durationpb.New(time.Minute),
			}}
			replay := worker.NewWorkflowReplayer()
			replay.RegisterWorkflowWithOptions(tc.fn, workflow.RegisterOptions{Name: tc.name})
			require.NoError(t, replay.ReplayWorkflowHistory(nil, &historypb.History{Events: []*historypb.HistoryEvent{started, scheduled, taskStarted, completed, activity}}))
		})
	}
}
