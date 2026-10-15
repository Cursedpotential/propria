package profferworker

import (
	"reflect"
	"testing"

	"github.com/Cursedpotential/probata/engine/approvedgraphprojectionflow"
	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/approvedwalkflow"
)

func TestApprovedGraphOptInLeavesOrdinaryWorkerUngated(t *testing.T) {
	t.Setenv(approvedGraphEnabledEnv, "")
	group, err := BuildApprovedGraph(nil, nil)
	if err != nil || group != nil {
		t.Fatalf("disabled approved graph changed worker boot: group=%v err=%v", group, err)
	}
	t.Setenv(approvedGraphEnabledEnv, "yes")
	if _, err := BuildApprovedGraph(nil, nil); err == nil {
		t.Fatal("malformed approved graph opt-in was silently ignored")
	}
	t.Setenv(approvedGraphEnabledEnv, "true")
	if _, err := BuildApprovedGraph(nil, nil); err == nil {
		t.Fatal("enabled approved graph accepted missing shared database and version store")
	}
}

// TestRegisterApprovedGraphUsesExactStandaloneNames verifies the optional registry and Python ownership.
// Inputs: disabled/enabled graph groups. Outputs: exact workflow and Activity assertions.
// Effects: recorder only; choose to detect missing walk registration or duplicate Python execution ownership.
func TestRegisterApprovedGraphUsesExactStandaloneNames(t *testing.T) {
	registrar := &registrationRecorder{}
	RegisterApprovedGraph(registrar, nil)
	if registrar.workflowCount != 0 || len(registrar.names) != 0 {
		t.Fatal("disabled approved graph registered work")
	}
	RegisterApprovedGraph(registrar, &ApprovedGraphGroup{})
	if !reflect.DeepEqual(registrar.workflowNames, []string{approvedgraphprojectionflow.WorkflowName, approvedgraphqueryflow.WorkflowName, approvedwalkflow.WorkflowName}) ||
		!reflect.DeepEqual(registrar.names, []string{approvedgraphprojectionflow.ActivityName, approvedgraphqueryflow.ActivityName, approvedwalkflow.PrepareActivity, approvedwalkflow.FinalizeActivity}) {
		t.Fatalf("approved graph registered wrong names: workflows=%v activities=%v", registrar.workflowNames, registrar.names)
	}
}
