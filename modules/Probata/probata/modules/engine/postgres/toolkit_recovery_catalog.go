// Byline: Codex · GPT-6 · 2026-10-04. Separate Case Bible recovery writer; no automatic schema mutation.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrToolkitRecoveryCollision = errors.New("recovery operation or occurrence metadata collision")
	ErrToolkitRecoveryMissing   = errors.New("recovery operation is absent")
)

// ToolkitRecoveryDB is the separate metadata transaction seam, implemented by a Case Bible pgx pool.
// Inputs: context and transaction options. Outputs: an isolated transaction. Effects: database operations only when called.
// Choose instead of widening CatalogDB, whose existing client remains read-only.
type ToolkitRecoveryDB interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// ToolkitRecoveryCatalog stores immutable operation metadata and occurrence rows beside dated catalogs.
// Inputs: separate configured transaction provider. Outputs: registration and independent readback methods.
// Effects: metadata INSERT/SELECT only; never schema application or current-generation changes.
type ToolkitRecoveryCatalog struct {
	db    ToolkitRecoveryDB
	close func()
}

// NewToolkitRecoveryCatalog injects a writer/readback transaction provider without opening it.
// Inputs: Case Bible transaction seam. Outputs: repository or nil-seam error. Effects: none.
// Choose for tests or parent-owned connection construction; OpenToolkitRecoveryCatalogFromFile handles explicit runtime configuration.
func NewToolkitRecoveryCatalog(db ToolkitRecoveryDB) (*ToolkitRecoveryCatalog, error) {
	if db == nil {
		return nil, errors.New("separate Case Bible recovery connection required")
	}
	return &ToolkitRecoveryCatalog{db: db}, nil
}

// OpenToolkitRecoveryCatalogFromFile opens the explicit CASEBIBLE_RECOVERY_DATABASE_URL_FILE configuration.
// Inputs: absolute mounted file path containing only a PostgreSQL DSN (at most 16KiB). Outputs: admitted writer repository.
// Effects: reads the file and checks database/principal privileges; no migrations, secret sourcing or logging of credentials.
// Choose over OpenCatalogPool: this writer requires casebible and membership in casebible_toolkit_recovery_writer, without superuser/BYPASSRLS.
func OpenToolkitRecoveryCatalogFromFile(ctx context.Context, path string) (*ToolkitRecoveryCatalog, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("CASEBIBLE_RECOVERY_DATABASE_URL_FILE must be an explicit absolute mounted file")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 16<<10 {
		return nil, errors.New("Case Bible recovery DSN file is missing or outside bounds")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open Case Bible recovery DSN file")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("Case Bible recovery DSN file changed")
	}
	raw, err := io.ReadAll(io.LimitReader(f, (16<<10)+1))
	if err != nil || len(raw) > 16<<10 {
		return nil, errors.New("cannot read bounded Case Bible recovery DSN file")
	}
	cfg, err := pgxpool.ParseConfig(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, errors.New("invalid Case Bible recovery DSN configuration")
	}
	if cfg.ConnConfig.Database != "casebible" || cfg.ConnConfig.User == "" {
		return nil, errors.New("recovery writer must explicitly target casebible with its own login")
	}
	cfg.MaxConns = 2
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["application_name"] = "toolkit-recovery-catalog"
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "5000"
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "2000"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("Case Bible recovery pool unavailable")
	}
	var admitted bool
	admissionCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	err = pool.QueryRow(admissionCtx, `SELECT current_database()='casebible' AND pg_has_role(current_user,'casebible_toolkit_recovery_writer','USAGE') AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication FROM pg_catalog.pg_roles WHERE rolname=current_user`).Scan(&admitted)
	if err != nil || !admitted {
		pool.Close()
		return nil, errors.New("Case Bible recovery principal admission failed")
	}
	return &ToolkitRecoveryCatalog{db: pool, close: pool.Close}, nil
}

// Close releases the dedicated recovery writer pool when the worker shuts down.
// Inputs/outputs: none. Effects: closes only this repository's pool, if owned.
// Choose instead of closing the existing read-only catalog or other application connections.
func (s *ToolkitRecoveryCatalog) Close() {
	if s.close != nil {
		s.close()
	}
}

// decodeToolkitRecoveryBatch converts one bounded database payload into the shared admission type.
// Inputs: stored JSON. Outputs: metadata batch or explicit malformed error. Effects: none.
// Choose for readback rather than trusting a stored digest without reconstructing all occurrence metadata.
func decodeToolkitRecoveryBatch(raw []byte) (activities.ToolkitRecoveryCatalogBatch, error) {
	var batch activities.ToolkitRecoveryCatalogBatch
	if len(raw) > 512<<10 {
		return batch, errors.New("database recovery metadata exceeds read bound")
	}
	if err := decodeToolkitRecoveryJSON(raw, &batch); err != nil {
		return batch, err
	}
	return batch, nil
}

// decodeToolkitRecoveryJSON rejects unknown or trailing stored metadata before replay/readback comparison.
// Inputs: bounded JSON and destination. Outputs: decoded value or explicit malformed error. Effects: none.
// Choose over permissive unmarshalling so unexpected database fields cannot disappear during canonical hashing.
func decodeToolkitRecoveryJSON(raw []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(dst) != nil {
		return errors.New("malformed database recovery metadata")
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("trailing database recovery metadata")
	}
	return nil
}

// readToolkitRecoveryTx independently reconstructs and verifies one operation and all its occurrence rows.
// Inputs: transaction and bounded operation ID. Outputs: canonical batch; missing/damaged rows fail visibly.
// Effects: exact operation SELECT and at most sixteen occurrence rows. Choose for replay validation and readback, not table-wide scans.
func readToolkitRecoveryTx(ctx context.Context, tx pgx.Tx, operation string) (activities.ToolkitRecoveryCatalogBatch, error) {
	var empty activities.ToolkitRecoveryCatalogBatch
	var raw []byte
	var expectedHash string
	err := tx.QueryRow(ctx, `SELECT metadata_sha256,payload FROM casebible_recovery.toolkit_registration WHERE operation_id=$1`, operation).Scan(&expectedHash, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, ErrToolkitRecoveryMissing
	}
	if err != nil {
		return empty, errors.New("cannot read recovery operation metadata")
	}
	batch, err := decodeToolkitRecoveryBatch(raw)
	if err != nil {
		return empty, err
	}
	_, headerHash, err := activities.ToolkitRecoveryCatalogCanonical(batch)
	if err != nil || headerHash != expectedHash || batch.Request.OperationID != operation {
		return empty, ErrToolkitRecoveryCollision
	}
	rows, err := tx.Query(ctx, `SELECT original_ref,payload FROM casebible_recovery.toolkit_occurrence WHERE operation_id=$1 ORDER BY original_ref LIMIT 16`, operation)
	if err != nil {
		return empty, errors.New("cannot read bounded recovery occurrences")
	}
	defer rows.Close()
	actual := batch
	actual.Packages = nil
	for rows.Next() {
		var original string
		var payload []byte
		var item activities.ToolkitPackagePreservationReceipt
		if err = rows.Scan(&original, &payload); err != nil || len(payload) > 32<<10 {
			return empty, errors.New("invalid recovery occurrence row")
		}
		if decodeToolkitRecoveryJSON(payload, &item) != nil || string(item.OriginalRef) != original {
			return empty, ErrToolkitRecoveryCollision
		}
		actual.Packages = append(actual.Packages, item)
	}
	if rows.Err() != nil {
		return empty, errors.New("recovery occurrence read failed")
	}
	_, actualHash, err := activities.ToolkitRecoveryCatalogCanonical(actual)
	if err != nil || actualHash != expectedHash {
		return empty, ErrToolkitRecoveryCollision
	}
	return actual, nil
}

// RegisterToolkitRecovery atomically inserts one immutable operation and fifteen distinct original occurrences.
// Inputs: verified bounded metadata batch. Outputs: nil for first registration or identical replay; collisions reject.
// Effects: INSERTs within one transaction, never UPDATE/DELETE/DDL. Choose instead of legacy loaders with cleanup/upsert overwrite behavior.
func (s *ToolkitRecoveryCatalog) RegisterToolkitRecovery(ctx context.Context, batch activities.ToolkitRecoveryCatalogBatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, digest, err := activities.ToolkitRecoveryCatalogCanonical(batch)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return errors.New("cannot begin recovery registration")
	}
	defer tx.Rollback(ctx)
	var inserted string
	err = tx.QueryRow(ctx, `INSERT INTO casebible_recovery.toolkit_registration(operation_id,metadata_sha256,payload) VALUES ($1,$2,$3::jsonb) ON CONFLICT (operation_id) DO NOTHING RETURNING operation_id`, batch.Request.OperationID, digest, string(raw)).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, e := readToolkitRecoveryTx(ctx, tx, batch.Request.OperationID)
		if e != nil {
			return e
		}
		_, actual, e := activities.ToolkitRecoveryCatalogCanonical(existing)
		if e != nil || actual != digest {
			return ErrToolkitRecoveryCollision
		}
		return tx.Commit(ctx)
	}
	if err != nil {
		return errors.New("recovery operation insertion failed")
	}
	for _, item := range batch.Packages {
		if err = ctx.Err(); err != nil {
			return err
		}
		payload, e := json.Marshal(item)
		if e != nil {
			return e
		}
		_, err = tx.Exec(ctx, `INSERT INTO casebible_recovery.toolkit_occurrence(operation_id,original_ref,package_name,payload) VALUES ($1,$2,$3,$4::jsonb)`, batch.Request.OperationID, string(item.OriginalRef), item.PackageName, string(payload))
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrToolkitRecoveryCollision
			}
			return errors.New("recovery occurrence insertion failed")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return errors.New("recovery registration commit outcome unknown; retry identical operation")
	}
	return nil
}

// ReadToolkitRecovery performs a separate read-only repeatable-read transaction for catalog receipt verification.
// Inputs: bounded operation ID. Outputs: all fifteen verified metadata rows. Effects: SELECTs only; never repairs missing rows.
// Choose after registration rather than interpreting its commit as independent readback or projection refresh.
func (s *ToolkitRecoveryCatalog) ReadToolkitRecovery(ctx context.Context, operation string) (activities.ToolkitRecoveryCatalogBatch, error) {
	var empty activities.ToolkitRecoveryCatalogBatch
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if len(operation) < 1 || len(operation) > 64 {
		return empty, errors.New("bounded recovery operation ID required")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return empty, errors.New("cannot begin independent recovery readback")
	}
	defer tx.Rollback(ctx)
	batch, err := readToolkitRecoveryTx(ctx, tx, operation)
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(ctx); err != nil {
		return empty, errors.New("independent recovery readback failed")
	}
	return batch, nil
}
