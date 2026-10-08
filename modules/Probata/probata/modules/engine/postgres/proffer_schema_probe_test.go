// Byline: Codex · GPT-6.1-sol · 2026-10-08 (fresh snapshot regression coverage)
// Byline: Codex · GPT-5.6-Sol · 2026-08-30
// Extended · Claude Code · Sonnet 5 · 2026-09-02 (BUILD LANE S2): ledger
// retarget coverage (public.schema_version -> ops.migration_ledger, D-109).
// Extended · Claude Code · Sonnet 5 · 2026-09-02 (BUILD LANE S3, D-126):
// dev-bypass sentinel-identity gating coverage.
package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type probeDB struct{ row probeRow }

func (db probeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return db.row
}

type capturingProbeDB struct {
	query string
	args  []any
	row   probeRow
}

func (db *capturingProbeDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.query = query
	db.args = args
	return db.row
}

type probeRow struct {
	database, user, owner                                                 string
	tables, columns                                                       int
	constraintsExact, substrateExact, roleSafe, grantsExact, receiptExact bool
	err                                                                   error
}

func (row probeRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = row.database, row.user, row.owner
	*dest[3].(*int), *dest[4].(*int) = row.tables, row.columns
	*dest[5].(*bool), *dest[6].(*bool), *dest[7].(*bool), *dest[8].(*bool), *dest[9].(*bool) =
		row.constraintsExact, row.substrateExact, row.roleSafe, row.grantsExact, row.receiptExact
	return nil
}

func admittedProbeRow() probeRow {
	return probeRow{
		database: "platform", user: "platform_runtime", owner: "platform_admin",
		tables: len(requiredProfferTables), columns: len(requiredProfferColumns),
		constraintsExact: true, substrateExact: true, roleSafe: true, grantsExact: true, receiptExact: true,
	}
}

func TestProbeProfferSchemaAdmitsExactPlatformContract(t *testing.T) {
	if err := ProbeProfferSchema(context.Background(), probeDB{row: admittedProbeRow()}); err != nil {
		t.Fatal(err)
	}
}

func TestProbeProfferSchemaCastsCatalogNamesBeforeTextArrayComparison(t *testing.T) {
	db := &capturingProbeDB{row: admittedProbeRow()}
	if err := ProbeProfferSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if strings.Count(db.query, "a.attname::text") != 3 || strings.Contains(db.query, "ARRAY(SELECT a.attname FROM") {
		t.Fatal("catalog attribute arrays must be text[] before comparison with the required text[] contract")
	}
}

// TestProbeProfferSchemaAdmitsSnapshotWithoutMigrationHistory checks D-153 admission.
// Inputs: complete catalog/receipt result. Output: test assertions. Effects: none.
// Use to prevent retired DDL-history requirements while retaining history write protection.
func TestProbeProfferSchemaAdmitsSnapshotWithoutMigrationHistory(t *testing.T) {
	db := &capturingProbeDB{row: admittedProbeRow()}
	if err := ProbeProfferSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(db.query, "FROM ops.migration_ledger") || strings.Contains(db.query, "migration_id") || strings.Contains(db.query, "public.schema_version") {
		t.Fatal("fresh snapshot admission must not depend on retired migration or contract-version history")
	}
	if !strings.Contains(db.query, "NOT has_table_privilege('platform_runtime','ops.migration_ledger','INSERT')") {
		t.Fatal("retained history must remain protected from runtime forgery")
	}
}

// TestProbeProfferSchemaFreshSnapshotReadOnly checks the actual fresh platform catalog.
// Inputs: optional PROFFER_SCHEMA_PROBE_TEST_DSN for platform_runtime on an existing
// canonical snapshot with the exact real registry receipt and no August migration rows.
// Output: live admission assertions. Effects: bounded SELECTs only, no DDL/data writes.
// Choose after the parent restores the canonical database; no fixture is created here.
func TestProbeProfferSchemaFreshSnapshotReadOnly(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("PROFFER_SCHEMA_PROBE_TEST_DSN"))
	if dsn == "" {
		t.Skip("PROFFER_SCHEMA_PROBE_TEST_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("fresh snapshot validation connection unavailable")
	}
	defer conn.Close(ctx)
	var historicalRows int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM ops.migration_ledger WHERE migration_id=ANY($1::text[])",
		[]string{"0036", "0037", "0038", "0039", "0042", "0050", "0051", "0053", "0054"}).Scan(&historicalRows); err != nil {
		t.Fatal("fresh snapshot history verification unavailable")
	}
	if historicalRows != 0 {
		t.Fatal("fresh snapshot proof requires no retired August migration history")
	}
	if err := ProbeProfferSchema(ctx, conn); err != nil {
		t.Fatal(err)
	}
}

func TestProbeProfferSchemaRejectsWrongIdentityOrScope(t *testing.T) {
	for name, mutate := range map[string]func(*probeRow){
		"legacy database": func(row *probeRow) { row.database = "ai" },
		"wrong role":      func(row *probeRow) { row.user = "ai" },
		"wrong owner":     func(row *probeRow) { row.owner = "ai" },
		"missing table":   func(row *probeRow) { row.tables-- },
		"missing column":  func(row *probeRow) { row.columns-- },
		"unsafe role":     func(row *probeRow) { row.roleSafe = false },
		"bad fk":          func(row *probeRow) { row.constraintsExact = false },
		"bad substrate":   func(row *probeRow) { row.substrateExact = false },
		"excess grant":    func(row *probeRow) { row.grantsExact = false },
		"missing receipt": func(row *probeRow) { row.receiptExact = false },
	} {
		t.Run(name, func(t *testing.T) {
			row := admittedProbeRow()
			mutate(&row)
			if err := ProbeProfferSchema(context.Background(), probeDB{row: row}); err == nil {
				t.Fatal("expected fail-closed admission")
			}
		})
	}
}

// TestProbeProfferSchemaDefaultBindsStrictAuthoritativeIdentity locks in the
// fail-closed default (D-125, D-126): with PLATFORM_DEV_AUTH_BYPASS unset,
// the probe must bind the REAL authoritative identity and the STRICT
// (approved_by='owner') receipt expectation -- unmet until go-live, exactly
// as before this build lane.
func TestProbeProfferSchemaDefaultBindsStrictAuthoritativeIdentity(t *testing.T) {
	db := &capturingProbeDB{row: admittedProbeRow()}
	if err := ProbeProfferSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if len(db.args) != 13 {
		t.Fatalf("expected 13 bound query args, got %d", len(db.args))
	}
	if db.args[2] != authoritativeMatterID || db.args[3] != authoritativeCourtCaseID {
		t.Fatalf("flag unset must bind the real authoritative identity, got matter=%v court_case=%v", db.args[2], db.args[3])
	}
	if db.args[10] != registryReceiptPayloadByteLength || db.args[11] != registryReceiptApprovedBy || db.args[12] != registryReceiptApprovedOn {
		t.Fatalf("flag unset must bind the STRICT receipt expectation, got payload_byte_length=%v approved_by=%v approved_on=%v",
			db.args[10], db.args[11], db.args[12])
	}
}

// TestProbeProfferSchemaDevFlagDoesNotChangeIdentity: the dev flag only governs login (owner
// 2026-10-02). With it on, the probe still binds the real go-live identity and receipt.
func TestProbeProfferSchemaDevFlagDoesNotChangeIdentity(t *testing.T) {
	for _, value := range []string{"", "1", "true"} {
		t.Setenv("PLATFORM_DEV_AUTH_BYPASS", value)
		db := &capturingProbeDB{row: admittedProbeRow()}
		if err := ProbeProfferSchema(context.Background(), db); err != nil {
			t.Fatal(err)
		}
		if db.args[2] != authoritativeMatterID || db.args[3] != authoritativeCourtCaseID || db.args[11] != registryReceiptApprovedBy {
			t.Fatalf("PLATFORM_DEV_AUTH_BYPASS=%q bound matter=%v court_case=%v approved_by=%v, want the real identity",
				value, db.args[2], db.args[3], db.args[11])
		}
	}
}

func TestProbeProfferSchemaHidesCatalogError(t *testing.T) {
	err := ProbeProfferSchema(context.Background(), probeDB{row: probeRow{err: errors.New("secret dsn detail")}})
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error = %v, want generic catalog failure", err)
	}
}
