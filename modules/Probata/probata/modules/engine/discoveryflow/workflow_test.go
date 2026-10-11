package discoveryflow

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"strings"
	"testing"
)

func TestWorkflowSchedulesOnlyExternalReferencesOnPythonQueue(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	assemble := func(context.Context, Request) (discovery.ScoreRequest, error) { return discovery.ScoreRequest{}, nil }
	score := func(context.Context, discovery.ScoreRequest) (discovery.ScoreReceipt, error) {
		return discovery.ScoreReceipt{}, nil
	}
	persist := func(context.Context, discovery.PersistRequest) (discovery.Completion, error) {
		return discovery.Completion{}, nil
	}
	env.RegisterActivityWithOptions(assemble, activity.RegisterOptions{Name: AssembleActivity})
	env.RegisterActivityWithOptions(score, activity.RegisterOptions{Name: ScoreActivity})
	env.RegisterActivityWithOptions(persist, activity.RegisterOptions{Name: PersistActivity})
	req := Request{RunID: "11111111-1111-4111-8111-111111111111", Scope: discovery.Scope{Mode: "LIVE", MatterID: "22222222-2222-4222-8222-222222222222", CourtCaseID: "33333333-3333-4333-8333-333333333333"}, FeedbackRevision: 3}
	batch := discovery.ScoreRequest{RunID: req.RunID, FeedbackRevision: 3, BatchRef: discovery.Ref{URI: "file:///existing/immutable.json", SHA256: strings.Repeat("a", 64), Bytes: 100}}
	receipt := discovery.ScoreReceipt{RunID: req.RunID, FeedbackRevision: 3, ResultRef: discovery.Ref{URI: "file:///existing/result.json", SHA256: strings.Repeat("b", 64), Bytes: 200}, CandidateCount: 1, ModelVersion: "local-v1"}
	env.OnActivity(AssembleActivity, mock.Anything, req).Return(batch, nil).Once()
	env.OnActivity(ScoreActivity, mock.Anything, mock.MatchedBy(func(in discovery.ScoreRequest) bool {
		raw, _ := json.Marshal(in)
		return in == batch && !strings.Contains(string(raw), "records") && !strings.Contains(string(raw), "text")
	})).Return(func(ctx context.Context, in discovery.ScoreRequest) (discovery.ScoreReceipt, error) {
		if activity.GetInfo(ctx).TaskQueue != PythonQueue {
			t.Fatal("Python queue differs")
		}
		return receipt, nil
	}).Once()
	env.OnActivity(PersistActivity, mock.Anything, discovery.PersistRequest{WorkRequest: req, BatchRef: batch.BatchRef, Receipt: receipt}).Return(discovery.Completion{RunID: req.RunID, Revision: 5, ScoreRevision: 2, FeedbackRevision: 3, CandidateCount: 1}, nil).Once()
	env.ExecuteWorkflow(Workflow, req)
	if e := env.GetWorkflowError(); e != nil {
		t.Fatal(e)
	}
	var result discovery.Completion
	if e := env.GetWorkflowResult(&result); e != nil || result.FeedbackRevision != 3 {
		t.Fatal(e, result)
	}
	env.AssertExpectations(t)
}

func TestWorkflowExhaustionRecordsProtectedFailure(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	assemble := func(context.Context, Request) (discovery.ScoreRequest, error) { return discovery.ScoreRequest{}, nil }
	fail := func(context.Context, discovery.FailureRequest) error { return nil }
	env.RegisterActivityWithOptions(assemble, activity.RegisterOptions{Name: AssembleActivity})
	env.RegisterActivityWithOptions(fail, activity.RegisterOptions{Name: FailActivity})
	req := Request{RunID: "11111111-1111-4111-8111-111111111111", Scope: discovery.Scope{Mode: "LIVE", MatterID: "22222222-2222-4222-8222-222222222222", CourtCaseID: "33333333-3333-4333-8333-333333333333"}}
	env.OnActivity(AssembleActivity, mock.Anything, req).Return(discovery.ScoreRequest{}, errors.New("sensitive original activity failure")).Times(3)
	env.OnActivity(FailActivity, mock.Anything, mock.MatchedBy(func(in discovery.FailureRequest) bool {
		return in.WorkRequest == req && in.Reason == "assembly" && !strings.Contains(in.Reason, "sensitive")
	})).Return(nil).Once()
	env.ExecuteWorkflow(Workflow, req)
	if env.GetWorkflowError() == nil {
		t.Fatal("exhaustion returned success")
	}
	env.AssertExpectations(t)
}
