// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Read-only access to the Case Bible catalog (PostgreSQL `casebible`, schema
// raw_duck, on ovh-files) for repair.find_other_version. The catalog is the
// source of truth for what exists where (owner rule 2026-09-16); this reads
// it and never re-scans a bucket. The connection is the Workbench's own
// convention (INTAKE_DISCOVERY_PG_*, role metabase_ro), opened with
// default_transaction_read_only=on so no statement here can write.
//
// Snapshots read (dated tables; update the names when newer snapshots land,
// the step summary reports which were read):
//
//	raw_duck.vault_index_source_20260918  vault/v1 objects: key, size, sha1, name
//	raw_duck.vault_moves_20260924         keys moved since that index (old_key -> new_key)
//	raw_duck.bucket_objects               every object of B2 salem-data, newest whole-bucket
//	                                      listing (provider 'b2'): key, size, sha1
//
// 2026-10-02 (Claude Code · Opus 5.5, owner "go" 20:18 EDT): the B2 part used to read
// raw_duck.b2_objects, the 2026-09-14 listing of consignatio/intake/ only (emptied
// 2026-09-16), so keys outside intake/ read as missing. It now reads the newest
// whole-bucket listing loaded by Consignatio casebible/tools/bucket_objects_load.py.
//
// Probed read-only 2026-09-25: the by-name query takes about 1.1 s over
// ~1.04 M rows, inside the 8 s statement timeout.

package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cursedpotential/probata/engine/activities"
)

// CatalogDB is the read surface the catalog store needs; *pgxpool.Pool has it.
type CatalogDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// CatalogVersionStore implements activities.CatalogVersionFinder.
type CatalogVersionStore struct {
	db CatalogDB
}

// NewCatalogVersionStore wraps a read-only catalog connection.
func NewCatalogVersionStore(db CatalogDB) (*CatalogVersionStore, error) {
	if db == nil {
		return nil, errors.New("postgres catalog store: database is required")
	}
	return &CatalogVersionStore{db: db}, nil
}

const (
	catalogVaultSnapshot = "raw_duck.vault_index_source_20260918"
	catalogB2Snapshot    = "raw_duck.bucket_objects"
)

const catalogVersionsByName = `
	SELECT key, size, sha1, snapshot FROM (
	    SELECT COALESCE(moved.new_key, vault.key) AS key, vault.size, COALESCE(vault.sha1, '') AS sha1,
	           '` + catalogVaultSnapshot + `' AS snapshot
	    FROM raw_duck.vault_index_source_20260918 vault
	    LEFT JOIN raw_duck.vault_moves_20260924 moved ON moved.old_key = vault.key
	    WHERE vault.name = $1
	    UNION ALL
	    SELECT objects.key, objects.size, COALESCE(objects.sha1, ''), '` + catalogB2Snapshot + `'
	    FROM raw_duck.bucket_objects objects
	    WHERE objects.provider = 'b2' AND objects.bucket = 'salem-data'
	      AND objects.listed_at = (SELECT max(listed_at) FROM raw_duck.bucket_objects
	                               WHERE provider = 'b2' AND bucket = 'salem-data')
	      AND (objects.key = $1 OR objects.key LIKE $2 ESCAPE '\')
	) found
	ORDER BY size DESC, key
	LIMIT $3`

const catalogVersionByKey = `
	SELECT key, size, sha1, snapshot FROM (
	    SELECT vault.key, vault.size, COALESCE(vault.sha1, '') AS sha1, '` + catalogVaultSnapshot + `' AS snapshot
	    FROM raw_duck.vault_index_source_20260918 vault WHERE vault.key = $1
	    UNION ALL
	    SELECT moved.new_key, vault.size, COALESCE(vault.sha1, ''), '` + catalogVaultSnapshot + `'
	    FROM raw_duck.vault_moves_20260924 moved
	    JOIN raw_duck.vault_index_source_20260918 vault ON vault.key = moved.old_key
	    WHERE moved.new_key = $1
	    UNION ALL
	    SELECT objects.key, objects.size, COALESCE(objects.sha1, ''), '` + catalogB2Snapshot + `'
	    FROM raw_duck.bucket_objects objects
	    WHERE objects.provider = 'b2' AND objects.bucket = 'salem-data'
	      AND objects.listed_at = (SELECT max(listed_at) FROM raw_duck.bucket_objects
	                               WHERE provider = 'b2' AND bucket = 'salem-data')
	      AND objects.key = $1
	) found
	LIMIT 1`

// escapeLike makes a file name a literal LIKE pattern.
func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

// FindByBasename lists catalog objects whose file name is basename.
func (s *CatalogVersionStore) FindByBasename(ctx context.Context, basename string, limit int) ([]activities.CatalogObject, error) {
	basename = strings.TrimSpace(basename)
	if basename == "" || strings.ContainsAny(basename, "/\x00") || len(basename) > 1024 {
		return nil, errors.New("catalog lookup requires one file name")
	}
	if limit < 1 || limit > 200 {
		return nil, errors.New("catalog lookup limit must be between 1 and 200")
	}
	rows, err := s.db.Query(ctx, catalogVersionsByName, basename, "%/"+escapeLike(basename), limit)
	if err != nil {
		return nil, fmt.Errorf("query the catalog by name: %w", err)
	}
	defer rows.Close()
	found := make([]activities.CatalogObject, 0, limit)
	for rows.Next() {
		var object activities.CatalogObject
		if err := rows.Scan(&object.Key, &object.Size, &object.SHA1, &object.Snapshot); err != nil {
			return nil, fmt.Errorf("scan a catalog row: %w", err)
		}
		found = append(found, object)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read catalog rows: %w", err)
	}
	return found, nil
}

// LookupKey reads one catalog object by its exact key.
func (s *CatalogVersionStore) LookupKey(ctx context.Context, key string) (activities.CatalogObject, bool, error) {
	var object activities.CatalogObject
	err := s.db.QueryRow(ctx, catalogVersionByKey, key).Scan(&object.Key, &object.Size, &object.SHA1, &object.Snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return activities.CatalogObject{}, false, nil
	}
	if err != nil {
		return activities.CatalogObject{}, false, fmt.Errorf("look up %s in the catalog: %w", key, err)
	}
	return object, true, nil
}

// CatalogConnection is the catalog's read-only connection, in the
// Workbench's INTAKE_DISCOVERY_PG_* convention. The password is read from a
// mounted file by the caller and never logged.
type CatalogConnection struct {
	Host, Database, User, Password string
	Port                           int
}

// OpenCatalogPool opens a small read-only pool. No connection is made until
// the first query, so an unreachable catalog cannot stop a worker starting.
func OpenCatalogPool(ctx context.Context, connection CatalogConnection) (*pgxpool.Pool, error) {
	if connection.Host == "" || connection.Database == "" || connection.User == "" || connection.Password == "" || connection.Port <= 0 {
		return nil, errors.New("postgres catalog: host, port, database, user and password are required")
	}
	dsn := url.URL{
		Scheme: "postgres", User: url.UserPassword(connection.User, connection.Password),
		Host: connection.Host + ":" + strconv.Itoa(connection.Port), Path: "/" + connection.Database,
		RawQuery: "connect_timeout=5",
	}
	config, err := pgxpool.ParseConfig(dsn.String())
	if err != nil {
		return nil, errors.New("postgres catalog: invalid connection configuration")
	}
	config.MaxConns = 2
	config.MinConns = 0
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	config.ConnConfig.RuntimeParams["statement_timeout"] = "8000"
	config.ConnConfig.RuntimeParams["lock_timeout"] = "1000"
	config.ConnConfig.RuntimeParams["application_name"] = "proffer-worker/repair.find_other_version"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.New("postgres catalog: could not create the connection pool")
	}
	return pool, nil
}
