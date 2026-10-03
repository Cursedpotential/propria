// Byline: Claude Code · Sonnet · 2026-10-02
package contacts

import (
	"context"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestWorkflowRunsTheSixStepsInOrderCarryingReferencesAndTheDryRunFlag(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	var order []string
	var seen = map[string]StepRequest{}
	steps := []struct{ name, ref string }{
		{ManifestActivity, "/w/r1/manifest.jsonl"}, {FetchActivity, "/w/r1/fetched.jsonl"}, {ParseActivity, "/w/r1/people.json"},
		{PeopleActivity, ""}, {PlaceholdersActivity, ""}, {RelinkActivity, ""},
	}
	for _, step := range steps {
		step := step
		env.RegisterActivityWithOptions(func(_ context.Context, req StepRequest) (Receipt, error) {
			order = append(order, step.name)
			seen[step.name] = req
			return Receipt{Step: step.name, DryRun: req.DryRun, Status: "success", Ref: step.ref, At: time.Unix(0, 0).UTC(), Counts: map[string]int64{"n": 1}}, nil
		}, activity.RegisterOptions{Name: step.name})
	}
	env.ExecuteWorkflow(ContactsImportWorkflow, Input{RunID: "run-1", DryRun: true, Actor: Actor{SubjectUID: "uid", Username: "matt"}})
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
		t.Fatalf("workflow error = %v", env.GetWorkflowError())
	}
	var result Result
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	want := []string{ManifestActivity, FetchActivity, ParseActivity, PeopleActivity, PlaceholdersActivity, RelinkActivity}
	if len(order) != len(want) || len(result.Receipts) != len(want) {
		t.Fatalf("order = %v receipts = %d", order, len(result.Receipts))
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v", order)
		}
		if !seen[want[i]].DryRun || seen[want[i]].Actor.Username != "matt" {
			t.Fatalf("%s did not get the dry-run flag and actor: %+v", want[i], seen[want[i]])
		}
	}
	if seen[FetchActivity].Refs["manifest"] != "/w/r1/manifest.jsonl" || seen[ParseActivity].Refs["files"] != "/w/r1/fetched.jsonl" ||
		seen[PeopleActivity].Refs["people"] != "/w/r1/people.json" || seen[PlaceholdersActivity].Refs["people"] != "/w/r1/people.json" {
		t.Fatalf("references were not carried: %+v", seen)
	}
	value, err := env.QueryWorkflow(StatusQueryName)
	if err != nil {
		t.Fatal(err)
	}
	var status RunStatus
	if err := value.Get(&status); err != nil || status.State != "completed" || len(status.Steps) != 6 {
		t.Fatalf("status = %+v %v", status, err)
	}
}

func TestWorkflowStopsAtTheFirstFailedStepAndReportsIt(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	ran := map[string]bool{}
	for _, name := range []string{ManifestActivity, FetchActivity, ParseActivity, PeopleActivity, PlaceholdersActivity, RelinkActivity} {
		name := name
		env.RegisterActivityWithOptions(func(_ context.Context, req StepRequest) (Receipt, error) {
			ran[name] = true
			if name == ParseActivity {
				return Receipt{}, context.DeadlineExceeded
			}
			return Receipt{Step: name, Status: "success", Ref: "/x", At: time.Unix(0, 0).UTC()}, nil
		}, activity.RegisterOptions{Name: name})
	}
	env.ExecuteWorkflow(ContactsImportWorkflow, Input{RunID: "run-2", Actor: Actor{SubjectUID: "u", Username: "m"}})
	if env.GetWorkflowError() == nil || ran[PeopleActivity] || ran[PlaceholdersActivity] || ran[RelinkActivity] {
		t.Fatalf("a failed parse must stop the run before any registry write: ran = %v err = %v", ran, env.GetWorkflowError())
	}
}

func TestWorkflowRefusesARunWithoutAnOwnerActorOrRunID(t *testing.T) {
	for name, input := range map[string]Input{
		"no actor":   {RunID: "run-3"},
		"no run id":  {Actor: Actor{SubjectUID: "u", Username: "m"}},
		"bad run id": {RunID: "../escape", Actor: Actor{SubjectUID: "u", Username: "m"}},
	} {
		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()
		env.ExecuteWorkflow(ContactsImportWorkflow, input)
		if env.GetWorkflowError() == nil {
			t.Fatalf("%s must be refused", name)
		}
	}
}
