// Byline: Codex · GPT-6 · 2026-10-05. Dedicated Case Bible working catalog; parent owns DDL/grants/runtime.
package postgres

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ToolkitWorkingCatalog uses the existing separate Case Bible transaction seam with a dedicated guarded writer.
// Inputs: injected transaction provider. Outputs: registration and independent readback methods.
// Effects: none at construction; choose over recovery storage or the read-only CatalogVersionStore.
type ToolkitWorkingCatalog struct {
	db    ToolkitRecoveryDB
	close func()
}

const toolkitWorkingAdmissionSQL = `SELECT current_database()='casebible'
 AND pg_has_role(current_user,'casebible_toolkit_working_writer','USAGE')
 AND NOT pg_has_role(current_user,'casebible_toolkit_working_owner','MEMBER')
 AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication
 AND NOT has_table_privilege(current_user,'raw_duck.source_occurrences','INSERT,UPDATE,DELETE,TRUNCATE')
 AND NOT has_table_privilege(current_user,'library_catalog.toolkit_working_operation','INSERT,UPDATE,DELETE,TRUNCATE')
 AND NOT has_table_privilege(current_user,'library_catalog.toolkit_working_object','INSERT,UPDATE,DELETE,TRUNCATE')
 AND has_function_privilege(current_user,'library_catalog.register_toolkit_working(text,text)','EXECUTE')
 AND has_function_privilege(current_user,'library_catalog.read_toolkit_working(text)','EXECUTE')
 FROM pg_catalog.pg_roles WHERE rolname=current_user`

// NewToolkitWorkingCatalog constructs an injectable transaction repository without I/O or schema application.
// Inputs: Case Bible transaction provider. Outputs: repository or nil-seam rejection. Effects: none.
// Choose for tests or parent-owned admitted connections; runtime file opening performs principal admission.
func NewToolkitWorkingCatalog(db ToolkitRecoveryDB) (*ToolkitWorkingCatalog, error) {
	if db == nil {
		return nil, errors.New("separate Case Bible working connection required")
	}
	return &ToolkitWorkingCatalog{db: db}, nil
}

// OpenToolkitWorkingCatalogFromFile opens an explicit CASEBIBLE_WORKING_DATABASE_URL_FILE and admits its principal.
// Inputs: absolute regular mounted DSN file, at most 16KiB. Outputs: dedicated pool/repository.
// Effects: bounded file and privilege reads, no DDL/writes or credential logging; choose instead of sharing recovery/platform credentials.
func OpenToolkitWorkingCatalogFromFile(ctx context.Context, path string) (*ToolkitWorkingCatalog, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if !filepath.IsAbs(path) {
		return nil, errors.New("CASEBIBLE_WORKING_DATABASE_URL_FILE must be absolute")
	}
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 16<<10 {
		return nil, errors.New("working DSN file missing or outside bounds")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, errors.New("working DSN file unavailable")
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		return nil, errors.New("working DSN file changed")
	}
	raw, e := io.ReadAll(io.LimitReader(f, (16<<10)+1))
	if e != nil || len(raw) > 16<<10 {
		return nil, errors.New("working DSN file read failed")
	}
	cfg, e := pgxpool.ParseConfig(strings.TrimSpace(string(raw)))
	if e != nil {
		return nil, errors.New("invalid working DSN configuration")
	}
	if cfg.ConnConfig.Database != "casebible" || cfg.ConnConfig.User == "" {
		return nil, errors.New("working writer must target casebible with own login")
	}
	cfg.MaxConns = 2
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["application_name"] = "toolkit-working-catalog"
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "15000"
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "2000"
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		return nil, errors.New("working pool unavailable")
	}
	admittedCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var admitted bool
	if e = pool.QueryRow(admittedCtx, toolkitWorkingAdmissionSQL).Scan(&admitted); e != nil || !admitted {
		pool.Close()
		return nil, errors.New("working principal/schema admission failed")
	}
	return &ToolkitWorkingCatalog{db: pool, close: pool.Close}, nil
}

// Close releases only the dedicated working catalog pool owned by this repository.
// Inputs/outputs: none. Effects: pool close if configured; choose during parent worker shutdown.
func (s *ToolkitWorkingCatalog) Close() {
	if s.close != nil {
		s.close()
	}
}

// RegisterToolkitWorking invokes guarded immutable admission inside its own transaction.
// Inputs: fully verified metadata batch. Outputs: nil only after exact returned hash and committed registration.
// Effects: function-scoped INSERTs, no direct corpus DML or inventory writes; choose for retryable additive registration.
func (s *ToolkitWorkingCatalog) RegisterToolkitWorking(ctx context.Context, b activities.ToolkitWorkingCatalogBatch) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	raw, digest, e := activities.ToolkitWorkingCatalogCanonical(b)
	if e != nil {
		return e
	}
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return errors.New("working registration transaction unavailable")
	}
	defer tx.Rollback(ctx)
	var admitted string
	if e = tx.QueryRow(ctx, `SELECT library_catalog.register_toolkit_working($1,$2)`, string(raw), digest).Scan(&admitted); e != nil {
		return errors.New("working catalog admission failed or conflicting receipt; no overwrite")
	}
	if admitted != digest {
		return errors.New("working registration hash mismatch")
	}
	if e = tx.Commit(ctx); e != nil {
		return errors.New("working registration commit outcome unknown; retry identical operation")
	}
	return nil
}

// ReadToolkitWorking reconstructs ledger and fixed-source occurrences in a separate read-only repeatable-read transaction.
// Inputs: bounded operation ID. Outputs: complete batch only after recomputed hash/count validation.
// Effects: SELECT only; choose for the independently tracked readback Activity, never repair data here.
func (s *ToolkitWorkingCatalog) ReadToolkitWorking(ctx context.Context, operation string) (activities.ToolkitWorkingCatalogBatch, error) {
	var b activities.ToolkitWorkingCatalogBatch
	if e := ctx.Err(); e != nil {
		return b, e
	}
	if len(operation) < 1 || len(operation) > 64 {
		return b, errors.New("bounded working operation required")
	}
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return b, errors.New("working independent readback transaction unavailable")
	}
	defer tx.Rollback(ctx)
	var raw []byte
	var expected string
	if e = tx.QueryRow(ctx, `SELECT metadata_sha256,payload FROM library_catalog.read_toolkit_working($1)`, operation).Scan(&expected, &raw); e != nil {
		return b, errors.New("working independent occurrence readback failed")
	}
	if len(raw) > 2<<20 {
		return b, errors.New("working database metadata exceeds read bound")
	}
	if e = decodeToolkitRecoveryJSON(raw, &b); e != nil {
		return b, e
	}
	_, actual, e := activities.ToolkitWorkingCatalogCanonical(b)
	if e != nil || b.Request.OperationID != operation || actual != expected {
		return activities.ToolkitWorkingCatalogBatch{}, errors.New("working independent metadata/count/hash mismatch")
	}
	if e = tx.Commit(ctx); e != nil {
		return activities.ToolkitWorkingCatalogBatch{}, errors.New("working independent readback completion failed")
	}
	return b, nil
}
