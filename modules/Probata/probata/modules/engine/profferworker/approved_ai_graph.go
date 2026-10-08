package profferworker

import (
	"errors"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/approvedaigraphprojectionflow"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
)

// BuildApprovedAIGraph binds the existing analysis sink to the native AI decision reader.
// Inputs: platform DB and the already configured approved graph group. Outputs: Activity group.
// Effects: none until Activity execution. Pick beside BuildApprovedGraph without another client or credential.
func BuildApprovedAIGraph(db platformpostgres.DB, existing *ApprovedGraphGroup) (*activities.ApprovedAIContextGraphActivities, error) {
	if existing == nil {
		return nil, nil
	}
	if db == nil || existing.Projection.Sink == nil {
		return nil, errors.New("approved AI graph: database and existing approved graph sink required")
	}
	reader, err := platformpostgres.NewApprovedAIGraphReader(db)
	if err != nil {
		return nil, err
	}
	return &activities.ApprovedAIContextGraphActivities{Reader: reader, Sink: existing.Projection.Sink}, nil
}

// RegisterApprovedAIGraph installs the native AI projection on the existing worker queue.
// Inputs: worker registrar and configured Activity group. Outputs: none.
// Effects: Temporal registration only. Pick beside RegisterApprovedGraph when its group is enabled.
func RegisterApprovedAIGraph(registrar ExtractionRegistrar, group *activities.ApprovedAIContextGraphActivities) {
	if group == nil {
		return
	}
	registrar.RegisterWorkflowWithOptions(approvedaigraphprojectionflow.Workflow, workflow.RegisterOptions{Name: approvedaigraphprojectionflow.WorkflowName})
	registrar.RegisterActivityWithOptions(group.ProjectApprovedAIContextGraph, activity.RegisterOptions{Name: approvedaigraphprojectionflow.ActivityName})
}
