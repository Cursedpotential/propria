package proffer

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

// repairFixture supplies explicit current authority with synthetic source references only.
// Inputs: none. Output: bounded repair input. Effects: none; choose for standalone retry regressions.
func repairFixture() PreviewAdmissionRepairInput {
	return PreviewAdmissionRepairInput{PreviewHandle: "synthetic-existing-handle", SourceVersionRef: "11111111-1111-4111-8111-111111111111", OperatingMode: "LIVE", MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, ActorSubjectUID: "owner", ActorUsername: "owner", IdempotencyKey: "repair-click", Reason: "Explicit current owner readmission"}
}

// TestPreviewAdmissionRepairStandaloneRetriesAndFails tracks actual Activity attempts in an independent Temporal workflow.
// Inputs: transient success, exhausted transient failure and invalid authority. Output: bounded attempt/failure/detail assertions.
// Effects: SDK test environment only; choose to prove no direct HTTP processing or unbounded retry.
func TestPreviewAdmissionRepairStandaloneRetriesAndFails(t *testing.T) {
	for _, scenario := range []string{"transient", "exhausted", "invalid"} {
		t.Run(scenario, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			attempts := 0
			in := repairFixture()
			if scenario == "invalid" {
				in.OperatingMode = ""
			}
			env.RegisterActivityWithOptions(func(ctx context.Context, got PreviewAdmissionRepairInput) (PreviewAdmissionRepairResult, error) {
				attempts++
				if got != in {
					t.Fatal("repair changed owner/source pins")
				}
				if scenario == "exhausted" || attempts < 3 {
					return PreviewAdmissionRepairResult{}, errors.New("synthetic transient store failure")
				}
				if activity.GetInfo(ctx).Attempt != 3 {
					t.Fatal("Temporal attempt number lost")
				}
				return PreviewAdmissionRepairResult{ReceiptRef: "preview-event://synthetic/1", EventID: 1}, nil
			}, activity.RegisterOptions{Name: PreviewAdmissionRepairActivityName})
			env.ExecuteWorkflow(PreviewAdmissionRepairWorkflow, in)
			if scenario == "transient" {
				var out PreviewAdmissionRepairResult
				if err := env.GetWorkflowResult(&out); err != nil || out.Status != "completed" || out.WorkflowID == "" || out.RunID == "" || attempts != 3 {
					t.Fatalf("result %+v attempts%d err%v", out, attempts, err)
				}
			} else {
				err := env.GetWorkflowError()
				if err == nil {
					t.Fatal("failure completed successfully")
				}
				var app *temporal.ApplicationError
				if !errors.As(err, &app) {
					t.Fatal(err)
				}
				var out PreviewAdmissionRepairResult
				if app.Details(&out) != nil || out.Status != "failed" || out.Reason == "" || out.WorkflowID == "" {
					t.Fatalf("failure details %+v", out)
				}
				want := 3
				if scenario == "invalid" {
					want = 0
				}
				if attempts != want {
					t.Fatalf("attempts%d want%d", attempts, want)
				}
			}
		})
	}
}
