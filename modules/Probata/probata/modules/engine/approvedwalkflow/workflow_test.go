package approvedwalkflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestWorkflowSequencesReferencesAndIndependentHindsight(t *testing.T) {
	in, _, _ := fixture()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	preparedIndices := []int{}
	reasonCalls := 0
	ref := func(name string) libraryvalidation.ArtifactRef {
		return libraryvalidation.ArtifactRef{URI: "b2://salem-data/fixture/" + name, VersionID: "fixture", SHA256: "sha256:" + strings.Repeat("f", 64), Bytes: 1}
	}
	env.RegisterActivityWithOptions(func(_ context.Context, step StepRequest) (Prepared, error) {
		preparedIndices = append(preparedIndices, step.Index)
		if (step.Index == 0 || step.Index == len(in.Cutoffs)) && step.Previous != nil {
			t.Fatal("initial or hindsight inherited analyst memory")
		}
		if step.Index > 0 && step.Index < len(in.Cutoffs) && step.Previous == nil {
			t.Fatal("historical continuation lost its prior checkpoint")
		}
		raw, _ := json.Marshal(step)
		if strings.Contains(string(raw), `"claims"`) || strings.Contains(string(raw), `"findings"`) {
			t.Fatal("source/model content entered workflow input")
		}
		return Prepared{Reference: Reference{Artifact: ref("input")}}, nil
	}, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(func(_ context.Context, input Reference) (Reference, error) {
		reasonCalls++
		return Reference{Artifact: ref("checkpoint")}, nil
	}, activity.RegisterOptions{Name: ReasonActivity})
	env.RegisterActivityWithOptions(func(_ context.Context, input FinalizeRequest) (Result, error) {
		if len(input.Steps) != len(in.Cutoffs)+1 || len(input.Inputs) != len(input.Steps) {
			t.Fatal("missing input/checkpoint references")
		}
		return Result{Artifact: ref("result"), ReviewStatus: "pending"}, nil
	}, activity.RegisterOptions{Name: FinalizeActivity})
	env.ExecuteWorkflow(Workflow, in)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result Progress
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "completed" || result.Result.ReviewStatus != "pending" || reasonCalls != 3 || len(preparedIndices) != 3 || preparedIndices[2] != 2 {
		t.Fatal("walk did not execute chronological steps then independent hindsight")
	}
}
