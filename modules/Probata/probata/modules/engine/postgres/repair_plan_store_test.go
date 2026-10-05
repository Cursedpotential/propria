// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cursedpotential/probata/engine/repairplan"
)

type repairPlanRow struct {
	values []any
	err    error
}

func (r repairPlanRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for index, value := range r.values {
		target := reflect.ValueOf(dest[index]).Elem()
		if value == nil {
			target.Set(reflect.Zero(target.Type()))
			continue
		}
		target.Set(reflect.ValueOf(value))
	}
	return nil
}

type repairPlanExec struct {
	sql  string
	args []any
}

type repairPlanTx struct {
	pgx.Tx
	rows      []pgx.Row
	queries   []string
	execs     []repairPlanExec
	committed bool
}

func (t *repairPlanTx) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	t.queries = append(t.queries, query)
	row := t.rows[0]
	t.rows = t.rows[1:]
	return row
}

func (t *repairPlanTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	t.execs = append(t.execs, repairPlanExec{sql: query, args: args})
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (t *repairPlanTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *repairPlanTx) Rollback(context.Context) error { return nil }

type repairPlanDB struct {
	rows    []pgx.Row
	queries []string
	args    [][]any
	tx      *repairPlanTx
}

func (db *repairPlanDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) { return db.tx, nil }

func (db *repairPlanDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (db *repairPlanDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queries = append(db.queries, query)
	db.args = append(db.args, args)
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

const testPreviewHandle = "HandleHandleHandleHandleHandle0123456789_-"

func TestResolveAnchorReadsTheReviewRunAndItsAssessment(t *testing.T) {
	version := uuid.MustParse("0199aaaa-0000-7000-8000-000000000001")
	matter := uuid.MustParse(devMatterID)
	court := uuid.MustParse(devCourtCaseID)
	db := &repairPlanDB{rows: []pgx.Row{
		repairPlanRow{values: []any{testPreviewHandle, "req-1", "b2://salem-data/v/sms-1.xml", "req-1",
			"pending-handler-selection/v1", &version, "smsbackuprestore_xml", &matter, &court, `{"operating_mode":"LIVE"}`}},
		repairPlanRow{err: pgx.ErrNoRows},
		repairPlanRow{values: []any{[]byte(`{"detection":{"fmt":"xml"}}`), []byte(`{"clean":false,"truncated":true}`)}},
	}}
	store, err := NewRepairPlanStore(db)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := store.ResolveAnchor(context.Background(), "b2://salem-data/v/sms-1.xml", testPreviewHandle)
	if err != nil {
		t.Fatal(err)
	}
	if anchor.PreviewHandle != testPreviewHandle || anchor.SourceVersionID != version.String() ||
		anchor.MatterID != devMatterID || anchor.CourtCaseID != devCourtCaseID || anchor.DetectedFormat != "" ||
		anchor.ParserOptionsRef != "pending-handler-selection/v1" || string(anchor.RepairReport) != `{"clean":false,"truncated":true}` {
		t.Fatalf("anchor = %+v", anchor)
	}
	if !strings.Contains(db.queries[0], "binding.preview_handle = $1") || db.args[0][0] != testPreviewHandle {
		t.Fatalf("anchor query = %s args=%v", db.queries[0], db.args[0])
	}
	if !strings.Contains(db.queries[1], "handler_detected_format") || !strings.Contains(db.queries[2], "repair_assessment") {
		t.Fatalf("follow-up queries = %v", db.queries[1:])
	}
}

func TestResolveAnchorBySourceUsesTheNewestRunAndFailsClosed(t *testing.T) {
	db := &repairPlanDB{rows: []pgx.Row{repairPlanRow{err: pgx.ErrNoRows}}}
	store, _ := NewRepairPlanStore(db)
	if _, err := store.ResolveAnchor(context.Background(), "b2://salem-data/v/none.xml", ""); !errors.Is(err, repairplan.ErrAnchorNotFound) {
		t.Fatalf("err = %v, want ErrAnchorNotFound", err)
	}
	if !strings.Contains(db.queries[0], "binding.source_ref = $1") || !strings.Contains(db.queries[0], "ORDER BY binding.created_at DESC") {
		t.Fatalf("source query = %s", db.queries[0])
	}

	// A run that has not registered its source yet anchors nothing further.
	var noVersion *uuid.UUID
	db = &repairPlanDB{rows: []pgx.Row{repairPlanRow{values: []any{testPreviewHandle, "r", "b2://b/k.xml", "r", "p", noVersion, "", noVersion, noVersion}}}}
	store, _ = NewRepairPlanStore(db)
	anchor, err := store.ResolveAnchor(context.Background(), "b2://b/k.xml", testPreviewHandle)
	if err != nil || anchor.SourceVersionID != "" || len(db.queries) != 1 {
		t.Fatalf("anchor = %+v err=%v queries=%d", anchor, err, len(db.queries))
	}

	db = &repairPlanDB{rows: []pgx.Row{repairPlanRow{err: errors.New("connection refused")}}}
	store, _ = NewRepairPlanStore(db)
	if _, err := store.ResolveAnchor(context.Background(), "b2://b/k.xml", testPreviewHandle); err == nil || errors.Is(err, repairplan.ErrAnchorNotFound) {
		t.Fatalf("an outage must not read as 'not found': %v", err)
	}
}

func receiptRequest(status string) repairplan.ReceiptRequest {
	request := repairplan.ReceiptRequest{
		WorkflowID: "repair-plan-p-1", RunID: "run-1", PlanID: "p-1", StepID: "s1", StepIndex: 0,
		Activity: "repair.salvage_truncated_xml", SourceVersionID: "0199aaaa-0000-7000-8000-000000000001",
		InputRef: "b2://salem-data/v/sms-1.xml", Status: status,
	}
	if status == repairplan.ReceiptSuccess {
		request.Result = &repairplan.StepResult{OutputRef: "b2://salem-data/v/sms-1.xml.derived/salvaged/sms-1.xml",
			OutputType: "sms_backup_xml", OutputKind: "derived_object", OutputSHA256: strings.Repeat("a", 64),
			Summary: json.RawMessage(`{"records_kept":2}`)}
	} else {
		request.Error = "no complete record precedes the cut-off"
	}
	return request
}

func TestRecordRepairStepReceiptWritesOneAppendOnlyReceipt(t *testing.T) {
	execution := uuid.New()
	tx := &repairPlanTx{rows: []pgx.Row{repairPlanRow{values: []any{execution}}, repairPlanRow{err: pgx.ErrNoRows}}}
	store, _ := NewRepairPlanStore(&repairPlanDB{tx: tx})
	ref, err := store.RecordRepairStepReceipt(context.Background(), receiptRequest(repairplan.ReceiptSuccess), 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(ref); err != nil || !tx.committed || len(tx.execs) != 1 {
		t.Fatalf("ref=%q committed=%t execs=%d", ref, tx.committed, len(tx.execs))
	}
	if !strings.Contains(tx.queries[0], "INSERT INTO context.activity_execution") {
		t.Fatalf("execution query = %s", tx.queries[0])
	}
	insert := tx.execs[0]
	if insert.args[2] != int32(2) || insert.args[3] != repairplan.ReceiptSuccess || insert.args[6] != nil && len(insert.args[6].([]byte)) != 0 {
		t.Fatalf("insert args = %v", insert.args)
	}
	var result map[string]any
	if err := json.Unmarshal(insert.args[5].([]byte), &result); err != nil {
		t.Fatal(err)
	}
	if result["ref"] != "b2://salem-data/v/sms-1.xml.derived/salvaged/sms-1.xml" || result["ref_kind"] != "repair_plan_step" ||
		result["output_sha256"] != strings.Repeat("a", 64) || result["step_id"] != "s1" {
		t.Fatalf("result_ref = %v", result)
	}
}

func TestRecordRepairStepReceiptIsIdempotentAndRecordsFailures(t *testing.T) {
	prior := uuid.New()
	tx := &repairPlanTx{rows: []pgx.Row{repairPlanRow{values: []any{uuid.New()}}, repairPlanRow{values: []any{prior}}}}
	store, _ := NewRepairPlanStore(&repairPlanDB{tx: tx})
	ref, err := store.RecordRepairStepReceipt(context.Background(), receiptRequest(repairplan.ReceiptSuccess), 3)
	if err != nil || ref != prior.String() || len(tx.execs) != 0 {
		t.Fatalf("a retried receipt must return the first one: ref=%q err=%v execs=%d", ref, err, len(tx.execs))
	}

	tx = &repairPlanTx{rows: []pgx.Row{repairPlanRow{values: []any{uuid.New()}}, repairPlanRow{err: pgx.ErrNoRows}}}
	store, _ = NewRepairPlanStore(&repairPlanDB{tx: tx})
	if _, err := store.RecordRepairStepReceipt(context.Background(), receiptRequest(repairplan.ReceiptFailed), 1); err != nil {
		t.Fatal(err)
	}
	insert := tx.execs[0]
	if insert.args[3] != repairplan.ReceiptFailed || insert.args[5] != nil && len(insert.args[5].([]byte)) != 0 {
		t.Fatalf("a failure receipt carries no result_ref: %v", insert.args)
	}
	var detail map[string]any
	if err := json.Unmarshal(insert.args[6].([]byte), &detail); err != nil || detail["message"] != "no complete record precedes the cut-off" {
		t.Fatalf("error_detail = %v err=%v", detail, err)
	}

	if _, err := store.RecordRepairStepReceipt(context.Background(), repairplan.ReceiptRequest{SourceVersionID: "not-a-uuid"}, 1); err == nil {
		t.Fatal("an invalid source version was accepted")
	}
}

func TestRepairStepIdempotencyKeyIsPerRunAndStep(t *testing.T) {
	request := receiptRequest(repairplan.ReceiptSuccess)
	if got := repairStepIdempotencyKey(request); got != "repair-plan:repair-plan-p-1:run-1:00:s1" {
		t.Fatalf("key = %q", got)
	}
	request.RunID = "run-2"
	if repairStepIdempotencyKey(request) == repairStepIdempotencyKey(receiptRequest(repairplan.ReceiptSuccess)) {
		t.Fatal("a new run of the same plan must record its own receipts")
	}
}

func TestMatterModeForIdentityUsesTheAdmittedIdentitiesOnly(t *testing.T) {
	for _, tc := range []struct {
		matter, court, want string
		known               bool
	}{
		{devMatterID, devCourtCaseID, "", false},
		{strings.ToUpper(devMatterID), devCourtCaseID, "", false},
		{authoritativeMatterID, authoritativeCourtCaseID, "", true},
		{devMatterID, authoritativeCourtCaseID, "", false},
		{"00000000-0000-0000-0000-000000000000", devCourtCaseID, "", false},
		{"", "", "", false},
	} {
		if known := AdmittedCaseIdentity(tc.matter, tc.court); known != tc.known {
			mode := ""
			t.Fatalf("%s/%s -> %q %t", tc.matter, tc.court, mode, known)
		}
	}
}

type catalogRows struct {
	pgx.Rows
	rows  [][]any
	index int
}

func (r *catalogRows) Next() bool { r.index++; return r.index <= len(r.rows) }
func (r *catalogRows) Scan(dest ...any) error {
	return repairPlanRow{values: r.rows[r.index-1]}.Scan(dest...)
}
func (r *catalogRows) Err() error { return nil }
func (r *catalogRows) Close()     {}

type catalogDB struct {
	rows  [][]any
	query string
	args  []any
	row   pgx.Row
}

func (db *catalogDB) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	db.query, db.args = query, args
	return &catalogRows{rows: db.rows}, nil
}

func (db *catalogDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.query, db.args = query, args
	return db.row
}

func TestCatalogVersionStoreQueriesByLiteralFileName(t *testing.T) {
	db := &catalogDB{rows: [][]any{{"consignatio/vault/v1/a_b%.xml", int64(10), "", "raw_duck.bucket_objects"}}}
	store, err := NewCatalogVersionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	found, err := store.FindByBasename(context.Background(), "a_b%.xml", 50)
	if err != nil || len(found) != 1 || found[0].Size != 10 {
		t.Fatalf("found = %+v err=%v", found, err)
	}
	if db.args[1] != `%/a\_b\%.xml` || db.args[2] != 50 {
		t.Fatalf("LIKE must treat the name literally: %v", db.args)
	}
	for _, bad := range []string{"", "dir/name.xml", strings.Repeat("a", 2000)} {
		if _, err := store.FindByBasename(context.Background(), bad, 5); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	db.row = repairPlanRow{err: pgx.ErrNoRows}
	if _, found, err := store.LookupKey(context.Background(), "k"); found || err != nil {
		t.Fatalf("missing key = %t %v", found, err)
	}
}
