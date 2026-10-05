// Byline: Codex · GPT-6 · 2026-10-04. Synthetic transaction tests; no remote database writes.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// recoveryTestBatch builds distinct original occurrences with identical content hashes.
// Inputs: none. Outputs: bounded synthetic batch. Effects: none; choose to prove occurrence-preserving registration.
func recoveryTestBatch() activities.ToolkitRecoveryCatalogBatch {
	b := activities.ToolkitRecoveryCatalogBatch{Schema: activities.ToolkitRecoveryCatalogSchema, Request: activities.ToolkitCatalogRegistrationInput{OperationID: "test-recovery", PreservationNamespace: "test-preservation", ResultRef: "file:///synthetic/result.json", ResultSHA256: strings.Repeat("a", 64), MetadataRef: "file:///synthetic/metadata.json"}, InventoryRef: "file:///synthetic/inventory.json", InventorySHA256: strings.Repeat("b", 64)}
	for i := 0; i < 15; i++ {
		name := fmt.Sprintf("synthetic-%02d.zip", i)
		ref := proffer.Ref("b2://salem-data/consignatio/casevault/recovery/library-sources/test-preservation/" + b.InventorySHA256 + "/" + name)
		b.Packages = append(b.Packages, activities.ToolkitPackagePreservationReceipt{StorageMode: activities.ToolkitPackagePreservationModeVersioned, PackageName: name, OriginalRef: proffer.Ref("file:///synthetic/" + name), PreservedRef: ref, ReceiptRef: ref + ".preservation.json", VerificationRef: ref + ".preservation.json.verified.json", SHA256: strings.Repeat("c", 64), Bytes: 100, ArchiveVersionID: "archive-v1", ReceiptVersionID: "receipt-v1", VerificationVersionID: "verification-v1", ReceiptSHA256: strings.Repeat("d", 64), ReceiptBytes: 100, VerificationReceiptSHA256: strings.Repeat("e", 64), VerificationReceiptBytes: 100})
	}
	return b
}

// recoveryTestRow models bounded pgx scan results without a database.
// Inputs: values/error. Outputs: Scan outcome. Effects: destination assignment only; choose for transaction contracts.
type recoveryTestRow struct {
	values []any
	err    error
}

// Scan copies the exact scalar/JSON metadata types used by the repository.
// Inputs: destinations. Outputs: scan/error. Effects: test memory only; choose over generic SQL emulation.
func (r recoveryTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan arity")
	}
	for i, d := range dest {
		switch p := d.(type) {
		case *string:
			*p = r.values[i].(string)
		case *[]byte:
			*p = append([]byte(nil), r.values[i].([]byte)...)
		default:
			return errors.New("unexpected scan type")
		}
	}
	return nil
}

// recoveryTestRows supplies a finite occurrence result set.
// Inputs: synthetic rows. Outputs: pgx iteration. Effects: cursor only; choose to exercise count/tamper readback.
type recoveryTestRows struct {
	pgx.Rows
	data  []recoveryTestRow
	index int
}

// Next advances the bounded fixture cursor.
// Inputs: none. Outputs: remaining-row flag. Effects: index; choose for occurrence iteration.
func (r *recoveryTestRows) Next() bool { r.index++; return r.index <= len(r.data) }

// Scan reads the current fixture occurrence.
// Inputs: destinations. Outputs: error. Effects: assignments; choose after Next.
func (r *recoveryTestRows) Scan(d ...any) error { return r.data[r.index-1].Scan(d...) }

// Err reports the fixture's completed iteration status.
// Inputs: none. Outputs: nil. Effects: none; choose for successful row iteration.
func (r *recoveryTestRows) Err() error { return nil }

// Close retains synthetic rows without filesystem cleanup.
// Inputs/outputs: none. Effects: none; choose for repository deferred Close.
func (r *recoveryTestRows) Close() {}

// recoveryTestDB holds only committed synthetic metadata and transaction observations.
// Inputs: repository operations. Outputs: isolated fixtures. Effects: process memory; choose without DB credentials.
type recoveryTestDB struct {
	header  []byte
	hash    string
	rows    []recoveryTestRow
	options []pgx.TxOptions
	failAt  int
	inserts int
}

// BeginTx copies committed state and records isolation/access mode.
// Inputs: context/options. Outputs: transaction or cancellation. Effects: observations; choose for atomicity and independent readback tests.
func (d *recoveryTestDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.options = append(d.options, opts)
	return &recoveryTestTx{db: d, header: append([]byte(nil), d.header...), hash: d.hash, rows: append([]recoveryTestRow(nil), d.rows...), opts: opts}, nil
}

// recoveryTestTx implements only repository SQL and refuses unexpected statements.
// Inputs: transaction calls. Outputs: bounded metadata. Effects: local staged state; choose to detect accidental DDL/update paths.
type recoveryTestTx struct {
	pgx.Tx
	db     *recoveryTestDB
	header []byte
	hash   string
	rows   []recoveryTestRow
	opts   pgx.TxOptions
	closed bool
}

// QueryRow emulates exact operation insert/select and immutable replay.
// Inputs: SQL/arguments. Outputs: one row. Effects: staged header only; choose to validate collision behavior.
func (tx *recoveryTestTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.HasPrefix(sql, "INSERT INTO casebible_recovery.toolkit_registration") {
		if tx.opts.AccessMode == pgx.ReadOnly {
			return recoveryTestRow{err: errors.New("write in read-only transaction")}
		}
		if len(tx.header) != 0 {
			return recoveryTestRow{err: pgx.ErrNoRows}
		}
		tx.hash = args[1].(string)
		tx.header = []byte(args[2].(string))
		return recoveryTestRow{values: []any{args[0].(string)}}
	}
	if strings.HasPrefix(sql, "SELECT metadata_sha256,payload FROM casebible_recovery.toolkit_registration") {
		if len(tx.header) == 0 {
			return recoveryTestRow{err: pgx.ErrNoRows}
		}
		return recoveryTestRow{values: []any{tx.hash, tx.header}}
	}
	return recoveryTestRow{err: errors.New("unexpected query")}
}

// Query enforces exact-operation bounded occurrence selection.
// Inputs: SQL/arguments. Outputs: at most sixteen rows. Effects: none; choose to prove bounded readback.
func (tx *recoveryTestTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if !strings.Contains(sql, "WHERE operation_id=$1 ORDER BY original_ref LIMIT 16") {
		return nil, errors.New("unbounded or unexpected query")
	}
	rows := tx.rows
	if len(rows) > 16 {
		rows = rows[:16]
	}
	return &recoveryTestRows{data: rows}, nil
}

// Exec emulates occurrence inserts with an optional mid-transaction failure.
// Inputs: SQL/arguments. Outputs: tag/error. Effects: staged metadata; choose for rollback proof.
func (tx *recoveryTestTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx.opts.AccessMode == pgx.ReadOnly || !strings.HasPrefix(sql, "INSERT INTO casebible_recovery.toolkit_occurrence") {
		return pgconn.CommandTag{}, errors.New("unexpected write")
	}
	tx.db.inserts++
	if tx.db.failAt > 0 && tx.db.inserts == tx.db.failAt {
		return pgconn.CommandTag{}, errors.New("synthetic failure")
	}
	tx.rows = append(tx.rows, recoveryTestRow{values: []any{args[1].(string), []byte(args[3].(string))}})
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

// Commit exposes staged writes only for successful write transactions.
// Inputs: context. Outputs: error. Effects: memory commit; choose for atomic registration evidence.
func (tx *recoveryTestTx) Commit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	if tx.opts.AccessMode != pgx.ReadOnly {
		tx.db.header = tx.header
		tx.db.hash = tx.hash
		tx.db.rows = tx.rows
	}
	return nil
}

// Rollback discards staged state without deleting test artifacts.
// Inputs: context. Outputs: nil or closed. Effects: marks closed; choose for deferred rollback.
func (tx *recoveryTestTx) Rollback(context.Context) error {
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	return nil
}

// TestToolkitRecoveryRegistration proves atomic insert, identical replay and separate immutable readback.
// Inputs: synthetic batch/DB. Outputs: assertions. Effects: memory only; choose for catalog write contract coverage.
func TestToolkitRecoveryRegistration(t *testing.T) {
	ctx := context.Background()
	db := &recoveryTestDB{}
	repo, _ := NewToolkitRecoveryCatalog(db)
	batch := recoveryTestBatch()
	if err := repo.RegisterToolkitRecovery(ctx, batch); err != nil {
		t.Fatal(err)
	}
	if len(db.rows) != 15 {
		t.Fatal("identical hashes collapsed original occurrences")
	}
	if err := repo.RegisterToolkitRecovery(ctx, batch); err != nil {
		t.Fatal("identical replay", err)
	}
	if db.inserts != 15 {
		t.Fatal("replay rewrote occurrences")
	}
	got, err := repo.ReadToolkitRecovery(ctx, batch.Request.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	_, expected, _ := activities.ToolkitRecoveryCatalogCanonical(batch)
	_, actual, _ := activities.ToolkitRecoveryCatalogCanonical(got)
	if actual != expected {
		t.Fatal("independent hash mismatch")
	}
	opts := db.options[len(db.options)-1]
	if opts.AccessMode != pgx.ReadOnly || opts.IsoLevel != pgx.RepeatableRead {
		t.Fatal("readback isolation missing")
	}
	batch.Request.ResultSHA256 = strings.Repeat("f", 64)
	if err = repo.RegisterToolkitRecovery(ctx, batch); !errors.Is(err, ErrToolkitRecoveryCollision) {
		t.Fatalf("operation pin collision: %v", err)
	}
	if db.hash != expected {
		t.Fatal("collision overwrote header")
	}
}

// TestToolkitRecoveryRollbackAndReadbackDamage rejects partial transactions and stored-row damage.
// Inputs: synthetic failures. Outputs: assertions. Effects: memory only; choose to distinguish registration from verified readback.
func TestToolkitRecoveryRollbackAndReadbackDamage(t *testing.T) {
	for _, kind := range []string{"rollback", "missing", "extra", "changed", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			db := &recoveryTestDB{}
			repo, _ := NewToolkitRecoveryCatalog(db)
			batch := recoveryTestBatch()
			ctx := context.Background()
			if kind == "rollback" {
				db.failAt = 8
				if repo.RegisterToolkitRecovery(ctx, batch) == nil || len(db.header) > 0 || len(db.rows) > 0 {
					t.Fatal("partial transaction committed")
				}
				return
			}
			if err := repo.RegisterToolkitRecovery(ctx, batch); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing":
				db.rows = db.rows[:14]
			case "extra":
				db.rows = append(db.rows, db.rows[0])
			case "changed":
				var p activities.ToolkitPackagePreservationReceipt
				_ = json.Unmarshal(db.rows[0].values[1].([]byte), &p)
				p.ArchiveVersionID = "changed"
				raw, _ := json.Marshal(p)
				db.rows[0].values[1] = raw
			case "unknown":
				raw := db.rows[0].values[1].([]byte)
				db.rows[0].values[1] = append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"unexpected":true}`)...)
			}
			if _, err := repo.ReadToolkitRecovery(ctx, batch.Request.OperationID); !errors.Is(err, ErrToolkitRecoveryCollision) {
				t.Fatalf("damaged readback accepted: %v", err)
			}
		})
	}
}

// TestToolkitRecoveryAdmission tests bounded cancellation, duplicate originals and configuration rejection before networking.
// Inputs: synthetic metadata and retained DSN fixtures. Outputs: assertions. Effects: retained tiny files only; choose instead of live credential tests.
func TestToolkitRecoveryAdmission(t *testing.T) {
	db := &recoveryTestDB{}
	repo, _ := NewToolkitRecoveryCatalog(db)
	batch := recoveryTestBatch()
	batch.Packages[1].OriginalRef = batch.Packages[0].OriginalRef
	if repo.RegisterToolkitRecovery(context.Background(), batch) == nil || len(db.options) > 0 {
		t.Fatal("invalid batch reached database")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(repo.RegisterToolkitRecovery(ctx, recoveryTestBatch()), context.Canceled) {
		t.Fatal("register cancellation")
	}
	if _, err := repo.ReadToolkitRecovery(ctx, "test-recovery"); !errors.Is(err, context.Canceled) {
		t.Fatal("read cancellation")
	}
	if _, err := OpenToolkitRecoveryCatalogFromFile(context.Background(), "relative.env"); err == nil {
		t.Fatal("relative config accepted")
	}
	root := "E:/AI_Workspace/Projects/Propria/_worktrees/toolkit-catalog-registration-20261004/to_be_deleted/catalog-config-tests"
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "dsn-")
	if err != nil {
		t.Fatal(err)
	}
	for i, value := range []string{"postgres://synthetic:DO_NOT_ECHO@localhost/wrong_database", strings.Repeat("x", (16<<10)+1), "invalid DSN = secret"} {
		path := filepath.Join(dir, fmt.Sprintf("test-%d.env", i))
		if err = os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		_, err = OpenToolkitRecoveryCatalogFromFile(context.Background(), path)
		if err == nil || strings.Contains(err.Error(), "DO_NOT_ECHO") || strings.Contains(err.Error(), "secret") {
			t.Fatal("configuration not safely rejected")
		}
	}
}

// TestToolkitRecoveryMigrationContract checks the additive least-privilege source migration without applying it.
// Inputs: explicit migration source. Outputs: structural assertions. Effects: file read only; choose before parent DBA runtime validation.
func TestToolkitRecoveryMigrationContract(t *testing.T) {
	raw, err := os.ReadFile("../../../../../Consignatio/casebible/catalog_reconcile/toolkit_recovery_catalog_20261004.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, required := range []string{"CREATE TABLE IF NOT EXISTS casebible_recovery.toolkit_registration", "PRIMARY KEY(operation_id,original_ref)", "catalog_reconcile.toolkit_recovered_originals", "GRANT INSERT ON casebible_recovery.toolkit_registration", "IS TRUE", "NOLOGIN", "not_assessed", "not_refreshed"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("missing migration contract: %s", required)
		}
	}
	for _, forbidden := range []string{"UPDATE catalog_reconcile.", "DELETE FROM ", "DROP ", "GRANT ALL", "ALTER TABLE catalog_reconcile.", "CREATE TABLE IF NOT EXISTS catalog_reconcile.current_generation"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("unexpected mutation: %s", forbidden)
		}
	}
}
