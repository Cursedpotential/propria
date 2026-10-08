package atomictool

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// TestWorkflowFailureRemainsFailed proves a failing tool cannot become a
// completed Temporal execution. Inputs: valid source pin and failing Activity.
// Outputs: failed workflow and actor-bound query. Effects: test environment only;
// use as the regression guard for failed source-backed actions.
func TestWorkflowFailureRemainsFailed(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(_ context.Context, _ Request) (Result, error) {
		return Result{}, errors.New("tool unavailable")
	}, activity.RegisterOptions{Name: ActivityName})
	env.ExecuteWorkflow(Workflow, sourceRequest())
	if env.GetWorkflowError() == nil {
		t.Fatal("failed tool completed its workflow")
	}
	query, err := env.QueryWorkflow(StatusQuery)
	if err != nil {
		t.Fatal(err)
	}
	var progress Progress
	if err := query.Get(&progress); err != nil || progress.ActorSubjectUID != "operator-1" || progress.Outcome != "failed" {
		t.Fatalf("failed workflow query = %+v, %v", progress, err)
	}
}

// TestRequestDigestBindsSourceAndOptions proves same-key retries share one
// identity only for the same action. Inputs: changed copies of one request.
// Outputs: digest comparisons. Effects: none; use before Temporal replay.
func TestRequestDigestBindsSourceAndOptions(t *testing.T) {
	request := sourceRequest()
	base, err := requestDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	retry := request
	retry.Args = map[string]any{"sample_limit": 5}
	changed, err := requestDigest(retry)
	if err != nil || base == changed {
		t.Fatalf("changed options reused the same digest: %v", err)
	}
	retry = request
	retry.SourceSHA256 = strings.Repeat("b", 64)
	changed, err = requestDigest(retry)
	if err != nil || base == changed {
		t.Fatalf("changed source reused the same digest: %v", err)
	}
	retry = request
	retry.RequestID = "new-key"
	changed, err = requestDigest(retry)
	if err != nil || base != changed {
		t.Fatalf("idempotency key changed action digest: %v", err)
	}
}
