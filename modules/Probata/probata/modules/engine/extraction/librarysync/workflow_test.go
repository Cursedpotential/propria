// Byline: Codex · GPT-6.1 · 2026-10-05.
package librarysync

import (
	"context"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func registerTestUnits(env *testsuite.TestWorkflowEnvironment, s *Service) {
	env.RegisterActivityWithOptions(s.ListPage, activity.RegisterOptions{Name: ListActivity})
	env.RegisterActivityWithOptions(s.CheckObservation, activity.RegisterOptions{Name: SeenActivity})
	env.RegisterActivityWithOptions(s.HashSource, activity.RegisterOptions{Name: HashSourceActivity})
	env.RegisterActivityWithOptions(s.RetainSource, activity.RegisterOptions{Name: RetainActivity})
	env.RegisterActivityWithOptions(s.ExtractSource, activity.RegisterOptions{Name: ExtractActivity})
	env.RegisterActivityWithOptions(s.HydrateObservation, activity.RegisterOptions{Name: HydrateActivity})
	env.RegisterActivityWithOptions(s.StageObservation, activity.RegisterOptions{Name: ObserveActivity})
	env.RegisterActivityWithOptions(s.ClaimOperation, activity.RegisterOptions{Name: ClaimActivity})
	env.RegisterActivityWithOptions(s.PreparePayload, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(func(ctx context.Context, in WriteRequest) (Handle, error) {
		return s.WriteVersion(ctx, in.Prepared, in.AttemptID)
	}, activity.RegisterOptions{Name: WriteActivity})
	env.RegisterActivityWithOptions(s.RefreshOperation, activity.RegisterOptions{Name: RefreshActivity})
	env.RegisterActivityWithOptions(s.PlanReconciliation, activity.RegisterOptions{Name: HistoryActivity})
	env.RegisterActivityWithOptions(s.HashReconciliation, activity.RegisterOptions{Name: HashVersionActivity})
	env.RegisterActivityWithOptions(s.CheckCurrent, activity.RegisterOptions{Name: CurrentActivity})
	env.RegisterActivityWithOptions(s.Acknowledge, activity.RegisterOptions{Name: AckActivity})
	env.RegisterActivityWithOptions(func(ctx context.Context, in FailureInput) (Outcome, error) {
		return s.RecordFailure(ctx, in.Claim, in.Status, in.Code)
	}, activity.RegisterOptions{Name: FailureActivity})
}

func TestWriteWorkflowRunsTrackedUnitsAndReconcilesLostReply(t *testing.T) {
	s, store, back := fixtureService(t)
	store.lostReply = true
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	registerTestUnits(env, s)
	env.ExecuteWorkflow(ToolkitLibrarySyncWriteWorkflow, WriteInput{fixtureOperation})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var out Outcome
	requireNoError(t, env.GetWorkflowResult(&out))
	if out.Status != Synced || store.putCalls != 1 || back.completed.IntentID == "" {
		t.Fatalf("workflow outcome %+v", out)
	}
}

func TestWriteWorkflowPersistsUnpreparedFailureAndRetainsPartialOutcome(t *testing.T) {
	for _, kind := range []string{"bad-payload", "partial-history"} {
		t.Run(kind, func(t *testing.T) {
			s, store, back := fixtureService(t)
			if kind == "bad-payload" {
				back.payload = []byte("different immutable bytes")
			} else {
				store.partial = true
			}
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			registerTestUnits(env, s)
			env.ExecuteWorkflow(ToolkitLibrarySyncWriteWorkflow, WriteInput{fixtureOperation})
			if env.GetWorkflowError() == nil {
				t.Fatal("incomplete workflow reported success")
			}
			if kind == "bad-payload" && (store.putCalls != 0 || back.failureCalls == 0) {
				t.Fatal("prewrite failure did not persist")
			}
			if kind == "partial-history" && back.completed.Status == Synced {
				t.Fatal("partial history cleared")
			}
		})
	}
}

func TestCycleWorkflowSkipsKnownVersionsAndStagesUnknownWithoutPublication(t *testing.T) {
	s, store, back := fixtureService(t)
	key := fixtureScope.LegalRoot + "reference-data/fixture.md"
	raw := []byte("full synthetic private Markdown")
	obj := Object{Bucket: fixtureScope.Bucket, Key: key, VersionID: "v", Latest: true, UploadedAt: fixtureNow, ContentType: "text/markdown", Size: int64(len(raw))}
	store.add(raw, obj)
	store.listFunc = func(child string, _ Cursor) (Page, error) {
		p := Page{Objects: []Object{}, Complete: true}
		if child == "reference-data" {
			p.Objects = append(p.Objects, obj)
		}
		return p, nil
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	registerTestUnits(env, s)
	env.ExecuteWorkflow(ToolkitLibrarySyncCycleWorkflow, CycleInput{})
	requireNoError(t, env.GetWorkflowError())
	var out CycleResult
	requireNoError(t, env.GetWorkflowResult(&out))
	if !out.Complete || out.Observed != 1 || len(back.observations) != 1 || back.observations[0].Status != CitationRequired {
		t.Fatal("cycle failed source staging")
	}
	if len(back.incoming) != 1 || back.uploadCalls != 1 {
		t.Fatal("cycle observed before retaining full incoming body")
	}
	back.seen = true
	env = suite.NewTestWorkflowEnvironment()
	registerTestUnits(env, s)
	env.ExecuteWorkflow(ToolkitLibrarySyncCycleWorkflow, CycleInput{})
	requireNoError(t, env.GetWorkflowError())
	requireNoError(t, env.GetWorkflowResult(&out))
	if out.Skipped != 1 || len(back.observations) != 1 {
		t.Fatal("known version reprocessed")
	}
}

func TestCycleWorkflowRejectsNonAdjacentCursorCycle(t *testing.T) {
	s, store, _ := fixtureService(t)
	root := fixtureScope.LegalRoot + "case-law/"
	calls := 0
	store.listFunc = func(_ string, _ Cursor) (Page, error) {
		calls++
		key := root + "A"
		if calls == 2 {
			key = root + "B"
		}
		return Page{Objects: []Object{}, Next: Cursor{Key: key, VersionID: "cursor"}, Complete: false}, nil
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	registerTestUnits(env, s)
	env.ExecuteWorkflow(ToolkitLibrarySyncCycleWorkflow, CycleInput{})
	if env.GetWorkflowError() == nil || calls != 3 {
		t.Fatalf("cursor cycle continued %d", calls)
	}
}
