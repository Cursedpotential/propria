package flow

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
	"strings"
	"testing"
)

// rateLimitFailureFixture models the observed server wrapper without retaining private provider text.
// Inputs: none. Output: synthetic structural match; effects: none, use for safe status regression tests.
func rateLimitFailureFixture() *failurepb.Failure {
	return &failurepb.Failure{Message: extractionFailureSizeWrapper, Cause: &failurepb.Failure{
		Message: "NIM request failed: HTTP 429 private-provider-sentinel", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "InferenceRuntimeError"}},
		Cause: &failurepb.Failure{Message: "Error code: 429 private-provider-sentinel", StackTrace: "private-stack", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "RateLimitError"}}},
	}}
}

// TestPendingExtractionFailureRecognizesOnlyBoundedStructuredRateLimit checks the real wrapper shape and refusal boundaries.
func TestPendingExtractionFailureRecognizesOnlyBoundedStructuredRateLimit(t *testing.T) {
	require.Equal(t, extractionRateLimitReason, pendingExtractionFailure(rateLimitFailureFixture()))
	require.Empty(t, pendingExtractionFailure(nil))
	ordinary := rateLimitFailureFixture()
	ordinary.Message = "ordinary failure"
	require.Equal(t, "ordinary failure", pendingExtractionFailure(ordinary))
	ordinary.Message = extractionFailureSizeWrapper + " extra"
	require.Equal(t, ordinary.Message, pendingExtractionFailure(ordinary))
	unknown := rateLimitFailureFixture()
	unknown.Cause.Cause.GetApplicationFailureInfo().Type = "UnknownError"
	require.Equal(t, extractionFailureSizeWrapper, pendingExtractionFailure(unknown))
	wrongCode := rateLimitFailureFixture()
	wrongCode.Cause.Cause.Message = "HTTP 4290"
	require.Equal(t, extractionFailureSizeWrapper, pendingExtractionFailure(wrongCode))
	for depth := 2; depth <= 7; depth++ {
		root := &failurepb.Failure{Message: extractionFailureSizeWrapper}
		tail := root
		for n := 1; n < depth; n++ {
			tail.Cause = &failurepb.Failure{}
			tail = tail.Cause
		}
		tail.Message = "HTTP 429 private-provider-sentinel"
		tail.FailureInfo = rateLimitFailureFixture().Cause.Cause.FailureInfo
		expected := extractionFailureSizeWrapper
		if depth <= 6 {
			expected = extractionRateLimitReason
		}
		require.Equal(t, expected, pendingExtractionFailure(root), "depth %d", depth)
	}
	cycle := &failurepb.Failure{Message: extractionFailureSizeWrapper}
	cycle.Cause = cycle
	require.Equal(t, extractionFailureSizeWrapper, pendingExtractionFailure(cycle))
	require.Len(t, pendingExtractionFailure(&failurepb.Failure{Message: strings.Repeat("x", 1500)}), 1200)
}

// TestPendingExtractionRetryUsesSafeCause preserves actual attempts while excluding nested private text.
func TestPendingExtractionRetryUsesSafeCause(t *testing.T) {
	progress := decoratePendingExtractionActivities(Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "extract", Status: StepRunning}}}, []*workflowpb.PendingActivityInfo{{ActivityId: "229", ActivityType: &commonpb.ActivityType{Name: LangExtractExtractActivity}, Attempt: 2, LastFailure: rateLimitFailureFixture()}})
	require.Equal(t, int32(2), progress.Steps[0].Attempt)
	require.Equal(t, "retrying", progress.Steps[0].Status)
	require.Equal(t, extractionRateLimitReason, progress.Steps[0].LastFailure)
	require.Equal(t, extractionRateLimitReason, progress.Steps[0].Detail)
}

// TestTerminalExtractionFailurePreservesProgress checks SDK conversion and the actual Starter failed-status seam.
func TestTerminalExtractionFailurePreservesProgress(t *testing.T) {
	cause := temporal.GetDefaultFailureConverter().FailureToError(rateLimitFailureFixture())
	original := Progress{Outcome: OutcomeFailed, Steps: []StepResult{{Step: "extract", Status: StepFailed, Detail: "original durable Progress detail", Counts: map[string]int{"messages": 3600}}, {Step: "rules", Status: StepCompleted, Detail: "kept"}}}
	failure := temporal.NewApplicationErrorWithCause("extraction failed", "StandaloneExtractionFailed", cause, original)
	c := &childStatusClient{t: t, snapshots: map[string]*workflowservice.DescribeWorkflowExecutionResponse{"child": childSnapshot("child", "actual-run", ExternalWorkflowName, enumspb.WORKFLOW_EXECUTION_STATUS_FAILED)}, failures: map[string]error{"child": failure}, reads: map[string]int{}}
	actual, err := (&Starter{Client: c}).statusForExecution(context.Background(), "child", "actual-run")
	require.NoError(t, err)
	require.Equal(t, "actual-run", actual.RunID)
	require.Equal(t, original.Steps[0].Detail, actual.Steps[0].Detail)
	require.Equal(t, original.Steps[0].Counts, actual.Steps[0].Counts)
	require.Equal(t, original.Steps[1], actual.Steps[1])
	require.Equal(t, extractionRateLimitReason, actual.Steps[0].LastFailure)
	require.Equal(t, OutcomeFailed, actual.Outcome)
	ordinary := temporal.NewApplicationErrorWithCause("ordinary", "StandaloneExtractionFailed", nil, original)
	c.failures["child"] = ordinary
	actual, err = (&Starter{Client: c}).statusForExecution(context.Background(), "child", "actual-run")
	require.NoError(t, err)
	require.Equal(t, original.Steps, actual.Steps)
	for depth := 3; depth <= 7; depth++ {
		bounded := cause
		for n := 3; n < depth; n++ {
			bounded = fmt.Errorf("wrapper: %w", bounded)
		}
		expected := ""
		if depth <= 6 {
			expected = extractionRateLimitReason
		}
		require.Equal(t, expected, terminalExtractionFailureReason(bounded), "depth %d", depth)
	}
	require.Empty(t, terminalExtractionFailureReason(temporal.NewApplicationError("HTTP 429", "RateLimitError")))
	require.Empty(t, terminalExtractionFailureReason(fmt.Errorf("ordinary: %w", temporal.NewApplicationError("HTTP 429", "RateLimitError"))))
	deep := cause
	for n := 0; n < 6; n++ {
		deep = fmt.Errorf("wrapper: %w", deep)
	}
	require.Empty(t, terminalExtractionFailureReason(deep))
	c.failures["child"] = cause
	actual, err = (&Starter{Client: c}).statusForExecution(context.Background(), "child", "actual-run")
	require.NoError(t, err)
	require.Equal(t, extractionRateLimitReason, actual.Steps[0].Detail)
}
