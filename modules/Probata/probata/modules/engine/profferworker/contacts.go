// Byline: Claude Code · Sonnet · 2026-10-02
//
// Wiring for ContactsImportWorkflow (package contacts). The registry is reached only through the case
// identity store; the catalog is a separate read-only connection (INTAKE_DISCOVERY_PG_*, the metabase_ro login) and the files come from the configured B2 object
// store. Anything not configured leaves its Activity answering with a permanent "not configured" error
// instead of stopping the worker.
package profferworker

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/objectstores"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
)

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func buildContacts(ctx context.Context, pool *pgxpool.Pool, stores objectstores.Stores, cfg Config) (activities.ContactsActivities, error) {
	identity, err := platformpostgres.NewCaseIdentityStore(pool)
	if err != nil {
		return activities.ContactsActivities{}, err
	}
	registry, err := platformpostgres.NewContactsRegistry(pool)
	if err != nil {
		return activities.ContactsActivities{}, err
	}
	// The catalog is read with the same read-only login the Workbench uses for Sources (metabase_ro):
	// INTAKE_DISCOVERY_PG_* name the host, database and user; the password is read from its mounted file.
	var catalog activities.ContactsCatalog
	if file := strings.TrimSpace(os.Getenv("INTAKE_DISCOVERY_PG_PASSWORD_FILE")); file != "" {
		raw, err := os.ReadFile(file)
		if err != nil {
			return activities.ContactsActivities{}, err
		}
		catalogCfg, err := pgxpool.ParseConfig("postgres://")
		if err != nil {
			return activities.ContactsActivities{}, err
		}
		catalogCfg.ConnConfig.Host = envOr("INTAKE_DISCOVERY_PG_HOST", "100.91.190.107")
		port, _ := strconv.Atoi(envOr("INTAKE_DISCOVERY_PG_PORT", "5433"))
		catalogCfg.ConnConfig.Port = uint16(port)
		catalogCfg.ConnConfig.Database = envOr("INTAKE_DISCOVERY_PG_DATABASE", "casebible")
		catalogCfg.ConnConfig.User = envOr("INTAKE_DISCOVERY_PG_USER", "metabase_ro")
		catalogCfg.ConnConfig.Password = strings.TrimSpace(string(raw))
		catalogCfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
		catalogCfg.ConnConfig.RuntimeParams["statement_timeout"] = "60000"
		catalogPool, err := pgxpool.NewWithConfig(ctx, catalogCfg)
		if err != nil {
			return activities.ContactsActivities{}, err
		}
		reader, err := platformpostgres.NewContactsCatalog(catalogPool)
		if err != nil {
			return activities.ContactsActivities{}, err
		}
		catalog = reader
	}
	var fetcher activities.ContactsFetcher
	if credentialFile, ok := stores["b2"]; ok {
		storeCfg, err := acquisition.LoadObjectStorageConfigFile(credentialFile)
		if err != nil {
			return activities.ContactsActivities{}, err
		}
		client, err := acquisition.NewS3Client(storeCfg)
		if err != nil {
			return activities.ContactsActivities{}, err
		}
		fetcher = activities.S3ContactsFetcher{Client: client}
	}
	return activities.NewContactsActivities(catalog, fetcher, registry, identity, filepath.Join(cfg.DeriveScratchDir, "contacts")), nil
}
