// Byline: Codex · GPT-6.1 · 2026-10-05.
package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrAIWorkproductCatalogAdmission = errors.New("narrow Case Bible source-occurrence API admission failed")
	ErrAIWorkproductCatalogCollision = errors.New("existing source occurrence or operation differs; nothing replaced")
)

// AdmitAIWorkproductCatalog checks only the narrow function capabilities on the existing writer connection.
// Inputs: context and already configured recovery writer lifecycle. Outputs: explicit admission error or nil. Effects: a read-only transaction; never GRANT, DDL or direct source-occurrence SELECT. Choose before parent-owned worker registration instead of granting broad raw_duck access.
func (s *ToolkitRecoveryCatalog) AdmitAIWorkproductCatalog(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ErrAIWorkproductCatalogAdmission
	}
	defer tx.Rollback(ctx)
	var admitted bool
	err = tx.QueryRow(ctx, `SELECT current_database()='casebible' AND pg_has_role(current_user,'casebible_toolkit_recovery_writer','USAGE') AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND has_schema_privilege(current_user,'source_occurrence_api','USAGE') AND has_function_privilege(current_user,'source_occurrence_api.register_ai_workproduct(text,text)','EXECUTE') AND has_function_privilege(current_user,'source_occurrence_api.read_ai_workproduct(text,text)','EXECUTE') FROM pg_catalog.pg_roles WHERE rolname=current_user`).Scan(&admitted)
	if err != nil || !admitted {
		return ErrAIWorkproductCatalogAdmission
	}
	if err = tx.Commit(ctx); err != nil {
		return ErrAIWorkproductCatalogAdmission
	}
	return nil
}

// RegisterAIWorkproductOccurrences atomically inserts or compares authenticated occurrences through the narrow database function.
// Inputs: complete canonical batch. Outputs: nil only for exact row-count admission. Effects: one serializable function call on the existing pool; no direct table writes, deletes, updates, invented source IDs or inventory generations. Choose instead of the fifteen-package recovery ledger.
func (s *ToolkitRecoveryCatalog) RegisterAIWorkproductOccurrences(ctx context.Context, batch activities.AIWorkproductCatalogBatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, sha, err := activities.AIWorkproductCatalogCanonical(batch)
	if err != nil {
		return err
	}
	if err = s.AdmitAIWorkproductCatalog(ctx); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errors.New("cannot begin source-occurrence admission")
	}
	defer tx.Rollback(ctx)
	var count int
	err = tx.QueryRow(ctx, `SELECT source_occurrence_api.register_ai_workproduct($1::text,$2::text)`, string(raw), sha).Scan(&count)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAIWorkproductCatalogCollision
		}
		return errors.New("narrow source-occurrence admission failed")
	}
	if count != len(batch.Rows) {
		return errors.New("source-occurrence admission count mismatch")
	}
	if err = tx.Commit(ctx); err != nil {
		return errors.New("source-occurrence commit outcome unknown; retry identical pinned batch")
	}
	return nil
}

// ReadAIWorkproductOccurrences independently reads only one fixed-scope operation through its pinned batch hash.
// Inputs: bounded operation and metadata SHA-256. Outputs: at most sixteen existing occurrences. Effects: repeatable-read read-only transaction through the restricted function; never repairs rows or selects the whole catalog. Choose after registration without interpreting commit as readback success.
func (s *ToolkitRecoveryCatalog) ReadAIWorkproductOccurrences(ctx context.Context, operation, sha string) ([]activities.AIWorkproductOccurrence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(operation) < 1 || len(operation) > 64 || len(sha) != 64 {
		return nil, errors.New("bounded operation and batch pin required")
	}
	if err := s.AdmitAIWorkproductCatalog(ctx); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, errors.New("cannot begin independent source-occurrence readback")
	}
	defer tx.Rollback(ctx)
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT source_occurrence_api.read_ai_workproduct($1::text,$2::text)`, operation, sha).Scan(&raw); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("restricted source-occurrence readback failed")
	}
	if len(raw) > 256<<10 {
		return nil, errors.New("source-occurrence readback metadata ceiling exceeded")
	}
	var rows []activities.AIWorkproductOccurrence
	if err = decodeToolkitRecoveryJSON(raw, &rows); err != nil || len(rows) > 16 {
		return nil, errors.New("malformed bounded source-occurrence readback")
	}
	// JSON is parsed strictly above; this marshal check bounds the decoded representation too.
	if encoded, e := json.Marshal(rows); e != nil || len(encoded) > 256<<10 {
		return nil, errors.New("decoded source-occurrence metadata ceiling exceeded")
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, errors.New("independent source-occurrence readback transaction failed")
	}
	return rows, nil
}
