// Byline: Codex · GPT-6 · 2026-10-05. Synthetic transaction/admission tests; no production writes.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/jackc/pgx/v5"
)

// workingRepositoryFixture creates a complete metadata-only batch with fixed approved evidence pins.
// Inputs: none. Outputs: 443 synthetic rows. Effects: none; choose for transaction/replay tests without private source payloads.
func workingRepositoryFixture() activities.ToolkitWorkingCatalogBatch {
	b := activities.ToolkitWorkingCatalogBatch{Schema: activities.ToolkitWorkingCatalogSchema, Request: activities.ToolkitWorkingCatalogInput{OperationID: "synthetic-working", ReceiptRef: "file:///synthetic/receipt.json", ReceiptSHA256: activities.ToolkitWorkingCatalogReceiptSHA256, ManifestRef: "file:///synthetic/manifest.json", ManifestSHA256: activities.ToolkitWorkingCatalogManifestSHA256, ReferenceMapRef: "file:///synthetic/map.json", ReferenceMapSHA256: activities.ToolkitWorkingCatalogReferenceMapSHA256, ExpectedObjects: 443}}
	for i := 0; i < 443; i++ {
		key := fmt.Sprintf("consignatio/casevault/KnowledgeBase/legal/reference-data/synthetic/member-%03d.md", i)
		b.Objects = append(b.Objects, activities.ToolkitWorkingCatalogObject{Placement: activities.ToolkitContentPlacementObject{UnitID: "synthetic-unit", SourceRef: "file:///synthetic/unit.zip", Path: fmt.Sprintf("member-%03d.md", i), ObjectKey: key, ObjectRef: proffer.Ref("b2://salem-data/" + key + "?versionId=synthetic-v1"), VersionID: "synthetic-v1", LatestVersionID: "synthetic-v1", SHA256: strings.Repeat("a", 64), Bytes: 10, BeforeVersions: []string{}, AfterVersions: []string{"synthetic-v1"}, Status: "verified"}, ArchiveSHA256: strings.Repeat("b", 64), ArchiveBytes: 4430, LogicalSourcePath: fmt.Sprintf("member-%03d.md", i), LogicalCategory: "reference-data", ResourceRole: "synthetic-reference", LinkPolicy: "Resolve through pinned private map", ReferenceRecords: []activities.ToolkitWorkingCatalogLink{}, SourceRecords: []activities.ToolkitWorkingCatalogLink{}})
	}
	return b
}

// workingMockDB retains only committed synthetic metadata and observes transaction boundaries.
// Inputs: mock repository operations. Outputs: isolated transaction. Effects: memory only; choose without database credentials.
type workingMockDB struct {
	raw                                  []byte
	sha                                  string
	options                              []pgx.TxOptions
	failAdmission, failCommit, wrongHash bool
	commits                              int
}

// BeginTx snapshots committed metadata and records isolation/access mode.
// Inputs: context/options. Outputs: mock transaction or cancellation. Effects: observations only; choose for independent readback proof.
func (d *workingMockDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	d.options = append(d.options, opts)
	return &workingMockTx{db: d, raw: append([]byte(nil), d.raw...), sha: d.sha, opts: opts}, nil
}

// workingMockTx refuses every statement except the two narrowly guarded working-catalog functions.
// Inputs: SQL calls. Outputs: row result. Effects: staged memory only; choose to catch accidental direct corpus writes.
type workingMockTx struct {
	pgx.Tx
	db   *workingMockDB
	raw  []byte
	sha  string
	opts pgx.TxOptions
}

// QueryRow emulates immutable function admission and independent readback, rejecting conflicting operation payloads.
// Inputs: function SQL/args. Outputs: result or visible conflict. Effects: staged metadata only; choose over permissive SQL mocks.
func (tx *workingMockTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	switch sql {
	case "SELECT library_catalog.register_toolkit_working($1,$2)":
		if tx.opts.AccessMode == pgx.ReadOnly || tx.db.failAdmission {
			return recoveryTestRow{err: errors.New("admission refused")}
		}
		raw := []byte(args[0].(string))
		sha := args[1].(string)
		if len(tx.raw) > 0 && (string(tx.raw) != string(raw) || tx.sha != sha) {
			return recoveryTestRow{err: errors.New("immutable operation conflict")}
		}
		tx.raw = raw
		tx.sha = sha
		if tx.db.wrongHash {
			sha = strings.Repeat("f", 64)
		}
		return recoveryTestRow{values: []any{sha}}
	case "SELECT metadata_sha256,payload FROM library_catalog.read_toolkit_working($1)":
		if len(tx.raw) == 0 {
			return recoveryTestRow{err: pgx.ErrNoRows}
		}
		return recoveryTestRow{values: []any{tx.sha, tx.raw}}
	default:
		return recoveryTestRow{err: errors.New("unexpected broad SQL")}
	}
}

// Commit promotes staged metadata only for writable transactions with a known successful outcome.
// Inputs: context. Outputs: commit outcome. Effects: mock committed state; choose to test unknown-outcome handling.
func (tx *workingMockTx) Commit(context.Context) error {
	if tx.db.failCommit {
		return errors.New("commit interrupted")
	}
	if tx.opts.AccessMode != pgx.ReadOnly {
		tx.db.raw = tx.raw
		tx.db.sha = tx.sha
	}
	tx.db.commits++
	return nil
}

// Rollback discards the staged fixture without deleting filesystem or database data.
// Inputs: context. Outputs: nil. Effects: none; choose for repository deferred rollback.
func (tx *workingMockTx) Rollback(context.Context) error { return nil }

// TestToolkitWorkingRepository verifies exact replay, collision rejection, independent isolation and tamper detection.
// Inputs: complete synthetic metadata. Outputs: assertions. Effects: process memory only; choose before production integration.
func TestToolkitWorkingRepository(t *testing.T) {
	db := &workingMockDB{}
	repo, _ := NewToolkitWorkingCatalog(db)
	ctx := context.Background()
	b := workingRepositoryFixture()
	if e := repo.RegisterToolkitWorking(ctx, b); e != nil {
		t.Fatal(e)
	}
	if e := repo.RegisterToolkitWorking(ctx, b); e != nil {
		t.Fatal("identical retry", e)
	}
	got, e := repo.ReadToolkitWorking(ctx, b.Request.OperationID)
	if e != nil || len(got.Objects) != 443 {
		t.Fatal("readback", e)
	}
	opt := db.options[len(db.options)-1]
	if opt.AccessMode != pgx.ReadOnly || opt.IsoLevel != pgx.RepeatableRead {
		t.Fatal("readback isolation missing")
	}
	b.Objects[0].Placement.SHA256 = strings.Repeat("c", 64)
	prior := string(db.raw)
	if e = repo.RegisterToolkitWorking(ctx, b); e == nil || string(db.raw) != prior {
		t.Fatal("conflicting operation overwritten")
	}
	var changed activities.ToolkitWorkingCatalogBatch
	if e = json.Unmarshal(db.raw, &changed); e != nil {
		t.Fatal(e)
	}
	changed.Objects[0].Placement.Bytes++
	db.raw, _ = json.Marshal(changed)
	if _, e = repo.ReadToolkitWorking(ctx, b.Request.OperationID); e == nil {
		t.Fatal("tampered row hash accepted")
	}
}

// TestToolkitWorkingRepositoryFailureBoundaries checks admission/returned-hash/commit failure and pre-I/O rejection.
// Inputs: targeted mock faults. Outputs: zero committed writes for rejected admission. Effects: memory only.
// Choose to prove rollback safety and bounded error messages without credentials.
func TestToolkitWorkingRepositoryFailureBoundaries(t *testing.T) {
	for _, fault := range []string{"admission", "returned-hash", "commit"} {
		t.Run(fault, func(t *testing.T) {
			db := &workingMockDB{failAdmission: fault == "admission", wrongHash: fault == "returned-hash", failCommit: fault == "commit"}
			repo, _ := NewToolkitWorkingCatalog(db)
			if e := repo.RegisterToolkitWorking(context.Background(), workingRepositoryFixture()); e == nil || len(db.raw) > 0 {
				t.Fatal("failure committed")
			}
		})
	}
	db := &workingMockDB{}
	repo, _ := NewToolkitWorkingCatalog(db)
	b := workingRepositoryFixture()
	b.Objects[0].Placement.VersionID = ""
	if e := repo.RegisterToolkitWorking(context.Background(), b); e == nil || len(db.options) > 0 {
		t.Fatal("invalid version reached SQL")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := repo.RegisterToolkitWorking(ctx, workingRepositoryFixture()); !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation")
	}
	if _, e := repo.ReadToolkitWorking(ctx, "synthetic-working"); !errors.Is(e, context.Canceled) {
		t.Fatal("read cancellation")
	}
	if _, e := OpenToolkitWorkingCatalogFromFile(context.Background(), "relative.env"); e == nil {
		t.Fatal("relative DSN")
	}
	if _, e := NewToolkitWorkingCatalog(nil); e == nil {
		t.Fatal("nil connection")
	}
}

// TestToolkitWorkingSQLContract checks additive SQL and narrow principal admission without claiming a live SQL proof.
// Inputs: bootstrap source. Outputs: structural scope assertions. Effects: file read only; choose alongside synthetic VPS execution.
func TestToolkitWorkingSQLContract(t *testing.T) {
	raw, e := os.ReadFile("../../../sql/bootstrap/toolkit_working_catalog_20261005.sql")
	if e != nil {
		t.Fatal(e)
	}
	sql := string(raw)
	for _, need := range []string{"SECURITY DEFINER SET search_path=pg_catalog,library_catalog", "ON CONFLICT(source,scope,path,source_id) DO NOTHING", "'family-court-library'", "PRIMARY KEY(operation_id,object_key)", "working operation receipt collision", "REVOKE ALL ON FUNCTION", "source_id is the pinned provider version", activities.ToolkitWorkingCatalogReceiptSHA256, "encode(sha256(convert_to(p_payload,'UTF8')),'hex')"} {
		if !strings.Contains(sql, need) {
			t.Fatal("missing SQL admission contract", need)
		}
	}
	for _, bad := range []string{"DROP ", "DELETE FROM ", "UPDATE raw_duck.", "bucket_objects", "current_generation", "GRANT ALL", "casebible_recovery."} {
		if strings.Contains(sql, bad) {
			t.Fatal("forbidden scope", bad)
		}
	}
	for _, need := range []string{"NOT rolbypassrls", "NOT rolsuper", "NOT rolcreatedb", "NOT rolcreaterole", "NOT rolreplication", "has_function_privilege", "NOT has_table_privilege", "working_owner','MEMBER'"} {
		if !strings.Contains(toolkitWorkingAdmissionSQL, need) {
			t.Fatal("missing principal admission", need)
		}
	}
}
