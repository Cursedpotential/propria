// Byline: Codex · GPT-6 · 2026-10-05.
package profferworker

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/Cursedpotential/probata/engine/activities"
)

// aiWorkproductCatalogRepository combines the narrow metadata API and explicit read-only admission check.
// Inputs: verified source-occurrence metadata. Outputs: admitted registration/readback operations.
// Effects: implementation-specific catalog metadata operations only; choose the existing pool lifecycle instead of another writer.
type aiWorkproductCatalogRepository interface {
	activities.AIWorkproductOccurrenceCatalog
	AdmitAIWorkproductCatalog(context.Context) error
}

// configureAIWorkproductCatalog reuses the configured catalog connection and existing placement adapters.
// Inputs: optional toolkit catalog group and existing derive-scratch placement group. Outputs: optional admitted AI catalog Activities.
// Effects: read-only database admission when configured; no new pool, DDL, source transfer or catalog write.
// Choose beside placement registration; an absent catalog remains disabled and a misconfigured one fails visibly.
func configureAIWorkproductCatalog(ctx context.Context, toolkit *activities.ToolkitCatalogRegistrationActivities, placement *activities.AIWorkproductPlacementActivities) (*activities.AIWorkproductCatalogActivities, error) {
	if toolkit == nil {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if placement == nil || !filepath.IsAbs(placement.AllowedRoot) || placement.Stores == nil {
		return nil, errors.New("proffer worker: AI workproduct catalog placement adapters unavailable")
	}
	repo, ok := toolkit.Catalog.(aiWorkproductCatalogRepository)
	if !ok || repo == nil {
		return nil, errors.New("proffer worker: configured Case Bible writer lacks AI source-occurrence API")
	}
	if err := repo.AdmitAIWorkproductCatalog(ctx); err != nil {
		return nil, errors.New("proffer worker: AI source-occurrence catalog admission failed")
	}
	return activities.NewAIWorkproductCatalogActivities(placement.AllowedRoot, placement.Stores, repo), nil
}
