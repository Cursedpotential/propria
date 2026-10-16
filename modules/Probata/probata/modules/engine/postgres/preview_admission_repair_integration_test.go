package postgres

import (
	"context"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
)

// TestPreviewAdmissionRepairDurableReadback verifies append-only readmission and all Messenger admission reads in isolated PostgreSQL.
// Inputs: PREVIEW_ADMISSION_REPAIR_TEST_DSN for a new backend_package_review_* disposable VPS database.
// Output: preserved event zero, idempotency, mismatch refusal and repaired preview/extraction/write admission assertions.
// Effects: synthetic fixtures in that isolated database only; never production repair or source re-registration.
func TestPreviewAdmissionRepairDurableReadback(t *testing.T) {
	dsn := os.Getenv("PREVIEW_ADMISSION_REPAIR_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated VPS repair database not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var db string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil || !strings.HasPrefix(db, "backend_package_review_") {
		t.Fatal("refusing non-disposable repair database")
	}
	ddl := `CREATE SCHEMA context;
 CREATE TABLE context.source(id uuid PRIMARY KEY,source_key text);
 CREATE TABLE context.source_version(id uuid PRIMARY KEY,source_id uuid,workflow_id text,matter_id uuid,court_case_id uuid,version_ordinal int,created_at timestamptz DEFAULT now(),declared_format text DEFAULT 'smsbackuprestore_xml');
 CREATE TABLE context.proffer_preview_binding(preview_handle text PRIMARY KEY,request_id text,source_ref text,workflow_id text,run_id text,parser_options_ref text,created_at timestamptz DEFAULT now());
 CREATE TABLE context.proffer_preview_snapshot(preview_handle text,snapshot_seq int,source_version_id uuid,raw_generation_id uuid,normalized_generation_id uuid);
 CREATE TABLE context.proffer_preview_decision(id uuid,preview_handle text,selection_ref text,parser_options_ref text,recorded_at timestamptz);
 CREATE TABLE context.proffer_preview_event(preview_handle text,event_id bigint,event_type text,occurred_at timestamptz,phase text,detail text CHECK(octet_length(detail)<=4000),PRIMARY KEY(preview_handle,event_id));
 CREATE TABLE context.raw_generation(id uuid PRIMARY KEY,source_version_id uuid);
 CREATE TABLE context.normalized_generation(id uuid PRIMARY KEY,source_version_id uuid,raw_generation_id uuid);
 CREATE TABLE context.activity_execution(id uuid PRIMARY KEY,source_version_id uuid,workflow_id text,activity_name text,created_at timestamptz DEFAULT now());
 CREATE TABLE context.activity_receipt(id uuid PRIMARY KEY,activity_execution_id uuid,attempt int,status text,result_ref jsonb,not_applicable_reason text,error_detail jsonb,started_at timestamptz DEFAULT now(),completed_at timestamptz DEFAULT now());`
	if _, err = pool.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	in := proffer.PreviewAdmissionRepairInput{PreviewHandle: "synthetic-existing-handle", SourceVersionRef: "11111111-1111-4111-8111-111111111111", OperatingMode: "LIVE", MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, ActorSubjectUID: "owner", ActorUsername: "owner", IdempotencyKey: "repair-click", Reason: "Explicit current owner readmission"}
	if _, err = pool.Exec(ctx, `INSERT INTO context.source VALUES('22222222-2222-4222-8222-222222222222','b2://synthetic/message_1.json');
 INSERT INTO context.proffer_preview_binding(preview_handle,request_id,source_ref,workflow_id,run_id,parser_options_ref) VALUES('synthetic-existing-handle','original-request','b2://synthetic/message_1.json','original-workflow','original-run','pending-handler-selection/v1');
 INSERT INTO context.proffer_preview_event VALUES('synthetic-existing-handle',0,'phase_changed',now(),'starting','');`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO context.source_version(id,source_id,workflow_id,matter_id,court_case_id,version_ordinal) VALUES($1::uuid,'22222222-2222-4222-8222-222222222222','original-workflow',$2::uuid,$3::uuid,1)`, in.SourceVersionRef, in.MatterID, in.CourtCaseID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO context.proffer_preview_snapshot VALUES($1,1,$2::uuid,'33333333-3333-4333-8333-333333333333','44444444-4444-4444-8444-444444444444')`, in.PreviewHandle, in.SourceVersionRef); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO context.raw_generation VALUES('33333333-3333-4333-8333-333333333333',$1::uuid);
 INSERT INTO context.normalized_generation VALUES('44444444-4444-4444-8444-444444444444',$1::uuid,'33333333-3333-4333-8333-333333333333')`, pgx.QueryExecModeSimpleProtocol, in.SourceVersionRef); err != nil {
		t.Fatal(err)
	}
	store, _ := NewProfferPreviewStore(pool, nil)
	extraction, _ := NewEntityExtractionStore(pool)
	before, err := store.Binding(ctx, in.PreviewHandle)
	if err != nil || before.OperatingMode != "" {
		t.Fatalf("initial %+v %v", before, err)
	}
	bad := in
	bad.SourceVersionRef = "55555555-5555-4555-8555-555555555555"
	if _, err = store.RepairBindingAdmission(ctx, bad); err == nil {
		t.Fatal("cross-source repair admitted")
	}
	first, err := store.RepairBindingAdmission(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.RepairBindingAdmission(ctx, in)
	if err != nil || again.ReceiptRef != first.ReceiptRef {
		t.Fatalf("replay %+v %v", again, err)
	}
	after, err := store.Binding(ctx, in.PreviewHandle)
	if err != nil || after.OperatingMode != "LIVE" || after.SourceVersionID.String() != in.SourceVersionRef {
		t.Fatalf("readback %+v %v", after, err)
	}
	run, err := extraction.ResolveRun(ctx, in.PreviewHandle)
	if err != nil || run.MatterMode != "LIVE" || run.SourceVersionID != in.SourceVersionRef {
		t.Fatalf("extraction admission %+v %v", run, err)
	}
	if err = requirePreviewWrite(ctx, pool, in.PreviewHandle); err != nil {
		t.Fatal(err)
	}
	var original string
	var count int
	if err = pool.QueryRow(ctx, `SELECT detail FROM context.proffer_preview_event WHERE preview_handle=$1 AND event_id=0`, in.PreviewHandle).Scan(&original); err != nil || original != "" {
		t.Fatal("initial event changed")
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM context.proffer_preview_event WHERE preview_handle=$1`, in.PreviewHandle).Scan(&count)
	if count != 2 {
		t.Fatalf("events%d want2", count)
	}
	bad = in
	bad.Reason = "different action same key"
	if _, err = store.RepairBindingAdmission(ctx, bad); err == nil {
		t.Fatal("idempotency mismatch admitted")
	}
	// A second operation deliberately separates its request and Temporal IDs and never publishes a preview.
	if _, err = pool.Exec(ctx, `
 INSERT INTO context.source_version(id,source_id,workflow_id,matter_id,court_case_id,version_ordinal) VALUES('66666666-6666-4666-8666-666666666666','22222222-2222-4222-8222-222222222222','upstream-request',$1::uuid,$2::uuid,2);
 INSERT INTO context.proffer_preview_binding(preview_handle,request_id,source_ref,workflow_id,run_id,parser_options_ref) VALUES('synthetic-upstream-only','upstream-request','b2://synthetic/message_1.json','upstream-workflow','upstream-run','options');
 INSERT INTO context.raw_generation VALUES('77777777-7777-4777-8777-777777777777','66666666-6666-4666-8666-666666666666');
 INSERT INTO context.normalized_generation VALUES('88888888-8888-4888-8888-888888888888','66666666-6666-4666-8666-666666666666','77777777-7777-4777-8777-777777777777');
 INSERT INTO context.activity_execution(id,source_version_id,workflow_id,activity_name) VALUES
 ('10000000-0000-4000-8000-000000000001','66666666-6666-4666-8666-666666666666','upstream-request','persist_raw_generation_activity'),
 ('10000000-0000-4000-8000-000000000002','66666666-6666-4666-8666-666666666666','upstream-request','persist_normalized_generation_activity'),
 ('10000000-0000-4000-8000-000000000003','66666666-6666-4666-8666-666666666666','upstream-request','publish_context_search_activity');
 INSERT INTO context.activity_receipt(id,activity_execution_id,attempt,status,result_ref,error_detail) VALUES
 ('20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',1,'success','{"ref_kind":"raw_generation","ref_id":"77777777-7777-4777-8777-777777777777"}',NULL),
 ('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000002',1,'success','{"ref_kind":"normalized_generation","ref_id":"88888888-8888-4888-8888-888888888888"}',NULL),
 ('20000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000003',1,'failed',NULL,'{"message":"human projection unavailable bounded fixture"}');`, pgx.QueryExecModeSimpleProtocol, in.MatterID, in.CourtCaseID); err != nil {
		t.Fatal(err)
	}
	results, err := store.ResolveOperationResults(ctx, "synthetic-upstream-only")
	if err != nil || results.Status != "available" || results.PreviewAvailable || results.NormalizedReceiptRef != "20000000-0000-4000-8000-000000000002" {
		t.Fatalf("upstream results %+v %v", results, err)
	}
	// A foreign-generation receipt is retained for diagnosis and refused; no latest result substitutes for it.
	if _, err = pool.Exec(ctx, `UPDATE context.normalized_generation SET raw_generation_id='33333333-3333-4333-8333-333333333333' WHERE id='88888888-8888-4888-8888-888888888888'`); err != nil {
		t.Fatal(err)
	}
	results, err = store.ResolveOperationResults(ctx, "synthetic-upstream-only")
	if err != nil || results.Status != "inconsistent" || results.NormalizedReceiptRef != "20000000-0000-4000-8000-000000000002" {
		t.Fatalf("cross-source result %+v %v", results, err)
	}
}
