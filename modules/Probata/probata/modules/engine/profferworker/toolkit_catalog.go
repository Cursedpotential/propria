// Byline: Codex · GPT-6 · 2026-10-04. Optional separate Case Bible recovery connection on the existing worker.
package profferworker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Cursedpotential/probata/engine/activities"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
)

// toolkitCatalogRepository owns only the separate recovery repository and its shutdown connection lifecycle.
// Inputs: verified metadata operations. Outputs: registration/readback results and Close. Effects: delegated metadata operations only.
// Choose over the existing CatalogVersionFinder, which remains a separate read-only dated-catalog client.
type toolkitCatalogRepository interface {
	activities.ToolkitRecoveryCatalogRepository
	Close()
}

// toolkitCatalogOpener injects connection admission without letting tests or registration code source secrets.
// Inputs: context and absolute DSN-file path. Outputs: admitted repository or error. Effects: implementation-specific connection reads only.
// Choose for the optional worker configuration seam, not for schema application or catalog writes.
type toolkitCatalogOpener func(context.Context, string) (toolkitCatalogRepository, error)

// configureToolkitCatalog optionally opens the explicit recovery writer and reuses the preservation root/store resolver.
// Inputs: context, CASEBIBLE_RECOVERY_DATABASE_URL_FILE value and existing preservation Activity group.
// Outputs: optional catalog Activities and shutdown closure, or visible configuration/admission failure.
// Effects: bounded credential-file/connection admission reads only when enabled; no B2 resolution, source transfer, DB writes or DDL.
// Choose in Run beside the optional dated-catalog pool; an unset file leaves the new registry disabled.
func configureToolkitCatalog(ctx context.Context, file string, preservation activities.ToolkitPackagePreservationActivities) (*activities.ToolkitCatalogRegistrationActivities, func(), error) {
	return configureToolkitCatalogWithOpener(ctx, file, preservation, func(ctx context.Context, path string) (toolkitCatalogRepository, error) {
		return platformpostgres.OpenToolkitRecoveryCatalogFromFile(ctx, path)
	})
}

// configureToolkitCatalogWithOpener validates opt-in configuration before admitting the separately owned connection.
// Inputs: context, optional file path, existing preservation adapters and injected opener. Outputs: group/once-only cleanup or error.
// Effects: directory metadata check, opener admission and eventual pool close only; never repairs config or falls back to the platform DB.
// Choose for focused configuration/lifecycle tests without real credentials or database access.
func configureToolkitCatalogWithOpener(ctx context.Context, file string, preservation activities.ToolkitPackagePreservationActivities, open toolkitCatalogOpener) (*activities.ToolkitCatalogRegistrationActivities, func(), error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !filepath.IsAbs(file) {
		return nil, nil, errors.New("proffer worker: CASEBIBLE_RECOVERY_DATABASE_URL_FILE must be absolute")
	}
	root := strings.TrimSpace(preservation.AllowedRoot)
	if !filepath.IsAbs(root) {
		return nil, nil, errors.New("proffer worker: configured recovery catalog requires the existing absolute TOOLKIT_INVENTORY_ROOT")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, nil, errors.New("proffer worker: recovery catalog inventory root unavailable")
	}
	if preservation.Stores == nil || open == nil {
		return nil, nil, errors.New("proffer worker: recovery catalog adapters missing")
	}
	repo, err := open(ctx, file)
	if err != nil {
		return nil, nil, errors.New("proffer worker: Case Bible recovery catalog configuration/admission failed")
	}
	if repo == nil {
		return nil, nil, errors.New("proffer worker: Case Bible recovery catalog repository missing")
	}
	if err = ctx.Err(); err != nil {
		repo.Close()
		return nil, nil, err
	}
	group := activities.NewToolkitCatalogRegistrationActivities(root, preservation.Stores, repo)
	var once sync.Once
	return &group, func() { once.Do(repo.Close) }, nil
}
