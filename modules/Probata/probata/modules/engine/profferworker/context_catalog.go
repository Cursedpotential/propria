// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package profferworker

import (
	"context"
	"errors"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
)

// nativeContextCatalog opens only the existing restricted writer during an explicit package Activity.
// Inputs: optional AI_CONTEXT_CATALOG_DATABASE_URL_FILE plain-DSN path. Outputs: lazy opener and pool cleanup.
// Effects: bounded credential/role admission only when invoked; no worker boot probe, broad writes or toolkit activation.
// Choose for native package cataloging without enabling the inventory-root or F:/Downloads recovery lanes.
func nativeContextCatalog(file string) func(context.Context) (activities.ContextPackageCatalog, func(), error) {
	file = strings.TrimSpace(file)
	return func(ctx context.Context) (activities.ContextPackageCatalog, func(), error) {
		if file == "" {
			return nil, nil, errors.New("native context catalog DSN not configured; package remains pending")
		}
		repo, err := platformpostgres.OpenToolkitRecoveryCatalogFromFile(ctx, file)
		if err != nil {
			return nil, nil, errors.New("native context catalog writer unavailable; package remains pending")
		}
		return repo, repo.Close, nil
	}
}
