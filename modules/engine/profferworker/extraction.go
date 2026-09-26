// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Entity/event extraction wiring for the proffer worker. Kept beside, not
// inside, RegisterAll: the extraction workflows are standalone and their
// Activities are none of the 26 Proffer stages.
package profferworker

import (
	"errors"
	"fmt"

	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
)

// ExtractionRegistrar is the worker seam the extraction wiring needs.
type ExtractionRegistrar interface {
	activities.ActivityRegistrar
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
}

// RegisterExtraction installs the two extraction workflows and their
// Activities under their exact names.
func RegisterExtraction(registrar ExtractionRegistrar, acts activities.EntityExtractionActivities) {
	flow.RegisterWorkflows(registrar)
	activities.RegisterEntityExtractionActivities(registrar, acts)
}

// buildExtraction constructs the extraction Activities. The model is
// configuration: an absent key file disables model extraction with a visible
// reason (rule-based proposals still run); a present but invalid
// configuration — including any GLM model id — is a loud boot failure.
func buildExtraction(db platformpostgres.DB) (activities.EntityExtractionActivities, error) {
	store, err := platformpostgres.NewEntityExtractionStore(db)
	if err != nil {
		return activities.EntityExtractionActivities{}, err
	}
	cfg, err := model.ConfigFromEnv()
	switch {
	case errors.Is(err, model.ErrDisabled):
		reason := "no extraction model is configured on the proffer worker (" + model.EnvAPIKeyFile + " is not mounted); rule-based proposals only"
		return activities.NewEntityExtractionActivities(store, nil, "", reason), nil
	case err != nil:
		return activities.EntityExtractionActivities{}, fmt.Errorf("proffer worker: entity model configuration: %w", err)
	}
	client, err := model.NewClient(cfg)
	if err != nil {
		return activities.EntityExtractionActivities{}, fmt.Errorf("proffer worker: entity model client: %w", err)
	}
	return activities.NewEntityExtractionActivities(store, client, cfg.ModelID, ""), nil
}
