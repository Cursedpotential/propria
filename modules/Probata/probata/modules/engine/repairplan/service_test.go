// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeRuns struct {
	started    []string
	input      RunInput
	status     RunStatus
	startError error
}

func (f *fakeRuns) StartPlan(_ context.Context, workflowID string, input RunInput) (string, error) {
	if f.startError != nil {
		return "", f.startError
	}
	f.started = append(f.started, workflowID)
	f.input = input
	return "run-1", nil
}

func (f *fakeRuns) PlanStatus(_ context.Context, workflowID string) (RunStatus, error) {
	if f.status.WorkflowID != workflowID {
		return RunStatus{}, ErrRunNotFound
	}
	return f.status, nil
}

func handlePtr(value string) *string { return &value }

func TestProposeOffersOnlyWhatCanRunHere(t *testing.T) {
	service := Service{Env: testEnv(t)}
	response, err := service.Propose(context.Background(), ProposeRequest{SourceRef: testSource, PreviewHandle: handlePtr(testHandle)})
	if err != nil {
		t.Fatal(err)
	}
	if response.Signature != "sms_backup_xml:truncated" || len(response.Proposals) != 4 || response.AgentAvailable {
		t.Fatalf("response = %+v", response)
	}
	encoded, _ := json.Marshal(response)
	var shape map[string]json.RawMessage
	_ = json.Unmarshal(encoded, &shape)
	for _, key := range []string{"signature", "proposals", "agent_available"} {
		if _, ok := shape[key]; !ok {
			t.Fatalf("response lacks %q: %s", key, encoded)
		}
	}

	// A run under a matter that is neither TEST nor REAL can run nothing, so
	// only the no-step wait option is offered.
	env := testEnv(t)
	stray := testAnchor()
	stray.MatterID = "00000000-0000-0000-0000-000000000000"
	env.Anchors = &fakeAnchors{anchor: stray}
	response, err = Service{Env: env}.Propose(context.Background(), ProposeRequest{SourceRef: testSource, PreviewHandle: handlePtr(testHandle)})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Proposals) != 1 || len(response.Proposals[0].Steps) != 0 {
		t.Fatalf("infeasible proposals were offered: %+v", response.Proposals)
	}
}

func TestProposeForAnUncoveredSignatureOffersNothing(t *testing.T) {
	env := testEnv(t)
	pdf := testAnchor()
	pdf.SourceRef, pdf.DeclaredFormat, pdf.DetectedFormat = "b2://salem-data/v/a.pdf", "pdf", "pdf"
	env.Anchors = &fakeAnchors{anchor: pdf}
	response, err := Service{Env: env}.Propose(context.Background(), ProposeRequest{SourceRef: pdf.SourceRef, PreviewHandle: handlePtr(testHandle)})
	if err != nil {
		t.Fatal(err)
	}
	if response.Signature != "pdf:truncated" || len(response.Proposals) != 0 || response.AgentAvailable || response.Proposals == nil {
		t.Fatalf("response = %+v", response)
	}
}

func TestProposeRequiresAMatchingReviewRun(t *testing.T) {
	service := Service{Env: testEnv(t)}
	if _, err := service.Propose(context.Background(), ProposeRequest{SourceRef: testSource, PreviewHandle: handlePtr("NotAHandleNotAHandleNotAHandle0123456789")}); !errors.Is(err, ErrAnchorNotFound) {
		t.Fatalf("err = %v, want ErrAnchorNotFound", err)
	}
	if _, err := service.Propose(context.Background(), ProposeRequest{SourceRef: "b2://salem-data/other.xml", PreviewHandle: handlePtr(testHandle)}); !errors.Is(err, ErrAnchorMismatch) {
		t.Fatalf("err = %v, want ErrAnchorMismatch", err)
	}
}

func TestRunStartsOnlyAValidPlan(t *testing.T) {
	runs := &fakeRuns{}
	service := Service{Env: testEnv(t), Runs: runs}
	bad := testPlan(lenientID, salvageID)
	if _, validated, err := service.Run(context.Background(), bad); !errors.Is(err, ErrPlanInvalid) || validated.OK || len(runs.started) != 0 {
		t.Fatalf("an invalid plan must not start: err=%v started=%v", err, runs.started)
	}
	plan := testPlan(salvageID)
	response, _, err := service.Run(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if response.WorkflowID != WorkflowIDFor(plan) || response.RunID != "run-1" || runs.input.Plan.PlanID != plan.PlanID {
		t.Fatalf("response = %+v input = %+v", response, runs.input)
	}
	if !WorkflowIDPattern.MatchString(response.WorkflowID) {
		t.Fatalf("workflow id %q does not match the runs/{workflow_id} pattern", response.WorkflowID)
	}
	runs.startError = errors.New("temporal unavailable")
	if _, _, err := service.Run(context.Background(), testPlan(findID)); err == nil || errors.Is(err, ErrPlanInvalid) {
		t.Fatalf("a start failure is not a validation failure: %v", err)
	}
}

func TestWorkflowIDFollowsExactlyWhatThePlanAsks(t *testing.T) {
	plan := testPlan(salvageID)
	same := testPlan(salvageID)
	same.Steps[0].Params = json.RawMessage(` { } `)
	if WorkflowIDFor(plan) != WorkflowIDFor(same) {
		t.Fatal("whitespace in params changed the run identity")
	}
	edited := testPlan(findID, salvageID)
	if WorkflowIDFor(plan) == WorkflowIDFor(edited) {
		t.Fatal("an edited plan must be a different run")
	}
}

func TestStatusRefusesAnUnknownOrMalformedWorkflowID(t *testing.T) {
	runs := &fakeRuns{status: RunStatus{WorkflowID: "repair-plan-plan-0001-abcd-0123456789ab", Status: RunRunning}}
	service := Service{Env: testEnv(t), Runs: runs}
	if _, err := service.Status(context.Background(), "../etc"); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("err = %v", err)
	}
	status, err := service.Status(context.Background(), "repair-plan-plan-0001-abcd-0123456789ab")
	if err != nil || status.Status != RunRunning {
		t.Fatalf("status = %+v err=%v", status, err)
	}
}
