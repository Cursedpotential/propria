package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestNativeRepairRegistrationAndQuoteFreeDurableReadback validates source lineage and actual PostgreSQL writes in isolation.
// Inputs: explicitly named disposable database and retained VPS fixture root. Outputs: exact retry/readback assertions.
// Effects: rollback-only fixture DDL/data and retained synthetic external files; never uses production or chat-generation tables.
func TestNativeRepairRegistrationAndQuoteFreeDurableReadback(t *testing.T) {
	dsn := os.Getenv("AI_NATIVE_REPAIR_TEST_DSN")
	root := os.Getenv("AI_NATIVE_REPAIR_TEST_ROOT")
	if dsn == "" || root == "" {
		t.Skip("requires explicit disposable native repair DB and retained fixture root")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var database string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		t.Fatal(err)
	}
	if database != "backend_package_review_operator_contracts_20261015" {
		t.Fatal("refusing a database outside the exact disposable repair fixture")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	// Existing disposable schemas remain intact by OID and are restored by the outer rollback.
	for _, schema := range []string{"context", "working"} {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, schema).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			saved := "native_repair_saved_" + schema + "_" + strings.ReplaceAll(uuid.NewString(), "-", "")
			if _, err = tx.Exec(ctx, "ALTER SCHEMA "+pgx.Identifier{schema}.Sanitize()+" RENAME TO "+pgx.Identifier{saved}.Sanitize()); err != nil {
				t.Fatal(err)
			}
		}
	}
	ddl := `CREATE SCHEMA IF NOT EXISTS context; CREATE SCHEMA IF NOT EXISTS working;
 CREATE TABLE context.source(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),source_key text UNIQUE,provenance_class text);
 CREATE TABLE context.retained_object(id uuid PRIMARY KEY,content_sha256 bytea,byte_length bigint);
 CREATE TABLE context.source_version(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),source_id uuid,version_ordinal int,workflow_id text UNIQUE,submission_idempotency_key text UNIQUE,declared_format text,acquired_at timestamptz,matter_id uuid,court_case_id uuid,original_object_id uuid);
 CREATE TABLE context.activity_execution(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),source_version_id uuid,workflow_id text,activity_name text,idempotency_key text,request_digest bytea,UNIQUE(source_version_id,activity_name,idempotency_key));
 CREATE TABLE context.activity_receipt(id uuid PRIMARY KEY,activity_execution_id uuid,attempt int,status text,started_at timestamptz,completed_at timestamptz,result_ref jsonb);
 CREATE TABLE context.source_metadata(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),source_version_id uuid,metadata_class text,metadata jsonb,extractor_id text,extractor_version text,extraction_activity_receipt_id uuid,generated_at timestamptz);
 CREATE TABLE working.extraction_run(id uuid PRIMARY KEY,extractor text,extractor_version text,source_summary text,status text,finished_at timestamptz,stats jsonb);
 CREATE TABLE working.candidate_entity(id uuid PRIMARY KEY,extraction_run_id uuid,source_raw_table text,source_raw_id text,entity_type text,name text,normalized_name text,confidence float8,attrs jsonb,content_sha256 bytea,review_state text,domain text,ontology_version text);
 CREATE TABLE working.candidate_event(id uuid PRIMARY KEY,extraction_run_id uuid,source_raw_table text,source_raw_id text,event_type text,summary text,occurred_at timestamptz,temporal_confidence float8,confidence float8,attrs jsonb,content_sha256 bytea,review_state text,domain text,ontology_version text);
 CREATE TABLE working.candidate_fact(id uuid PRIMARY KEY,extraction_run_id uuid,source_raw_table text,source_raw_id text,predicate text,statement text,evidence_quote text,confidence float8,attrs jsonb,content_sha256 bytea,review_state text,domain text,ontology_version text);`
	if _, err = tx.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	quote := "Synthetic original stays exclusively outside PostgreSQL"
	original, _ := json.Marshal([]any{map[string]any{"mapping": map[string]any{"node": map[string]any{"message": map[string]any{"content": map[string]any{"parts": []string{quote}}}}}}})
	hash := sha256.Sum256(original)
	digest := hex.EncodeToString(hash[:])
	sourceID, oldID, objectID := uuid.New(), uuid.New(), uuid.New()
	ref := "b2://fixture/historical.json"
	provider := "exact-original-version"
	_, err = tx.Exec(ctx, `INSERT INTO context.source(id,source_key,provenance_class)VALUES($1,$2,'unknown');`, sourceID, ref+"?versionId="+provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.retained_object VALUES($1,$2,$3)`, objectID, hash[:], len(original)); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.source_version(id,source_id,version_ordinal,workflow_id,submission_idempotency_key,declared_format,matter_id,court_case_id,original_object_id) VALUES($1,$2,1,'historical-fixture','historical-fixture','chatgpt_official_json',$3::uuid,$4::uuid,$5)`, oldID, sourceID, authoritativeMatterID, authoritativeCourtCaseID, objectID); err != nil {
		t.Fatal(err)
	}
	db := nestedPreviewTestDB{tx: tx}
	registration, _ := NewContextSourceRegistrationStore(db)
	req := activities.ContextSourceRegistrationRequest{RequestID: "context-source:fixture", WorkflowID: "context-source:fixture", SourceRef: ref, ProviderVersionID: provider, SourceKind: "ai_chat", DeclaredFormat: "chatgpt", HistoricalSourceVersionID: oldID.String(), SourceSHA256: digest, MatterID: authoritativeMatterID, CourtCaseID: authoritativeCourtCaseID}
	result, err := registration.RegisterContextSource(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := registration.RegisterContextSource(ctx, req)
	if err != nil || retry != result {
		t.Fatalf("exact native retry mismatch: %+v %+v %v", result, retry, err)
	}
	if result.SourceVersionRef == oldID.String() || result.HistoricalSourceVersionID != oldID.String() {
		t.Fatal("native registration replaced historical identity")
	}
	bad := req
	bad.RequestID, bad.WorkflowID = "context-source:mismatch", "context-source:mismatch"
	bad.ProviderVersionID = "different"
	if _, err = registration.RegisterContextSource(ctx, bad); err == nil {
		t.Fatal("historical provider substitution admitted")
	}
	var versions int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM context.source_version`).Scan(&versions); err != nil || versions != 2 {
		t.Fatal("refused request performed registration writes")
	}
	unit := filepath.Join(root, uuid.NewString())
	if err = os.MkdirAll(unit, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_CONTEXT_ROOT", root)
	fileRef := func(name string) string { return "file://" + filepath.ToSlash(filepath.Join(unit, name)) }
	write := func(name string, b []byte) {
		if e := os.WriteFile(filepath.Join(unit, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	write("original.json", original)
	prepared := aicontextsource.Prepared{ContractVersion: "ai-context-v1", Stage: "prepared", Source: aicontextsource.Identity{SourceRef: ref, ProviderVersionID: &provider, SourceFormat: "chatgpt", SourceSHA256: digest, SourceVersionID: result.SourceVersionRef, HistoricalSourceVersionID: oldID.String()}, OriginalRef: fileRef("original.json"), OriginalSHA256: digest}
	raw, _ := json.Marshal(prepared)
	write("prepared.json", raw)
	external, _ := json.Marshal(map[string]any{"contract_version": "ai-context-v1", "stage": "candidates", "quote": quote})
	write("candidates.json", external)
	externalHash := sha256.Sum256(external)
	span := sha256.Sum256([]byte(quote))
	candidate := service.AICandidate{AISourcePin: service.AISourcePin{SourceVersionID: result.SourceVersionRef, VersionID: &provider, SourceSHA256: digest, SourceRef: ref, PreparedRef: fileRef("prepared.json")}, NativeJSONPointer: "/0/mapping/node/message/content/parts/0", SourceSpan: service.AISourceSpan{Start: 0, End: len(quote), SHA256: hex.EncodeToString(span[:])}, SpanUnit: "unicode_codepoint", Kind: "fact", ReportedKind: "fact", ReviewDomain: "ai_chat_content", Predicate: "derived_fixture", Statement: "A typed derived assertion", EvidenceQuote: quote, EvidenceBundleRef: fileRef("candidates.json"), EvidenceBundleSHA256: hex.EncodeToString(externalHash[:]), Confidence: 0.8}
	store := &EntityExtractionStore{db: db}
	runID := flow.DeterministicID("native-fixture", result.SourceVersionRef)
	candidate.Kind, candidate.ReportedKind, candidate.Name, candidate.EntityType = "entity", "entity", "Fixture Person", "person"
	event := candidate
	event.Kind, event.ReportedKind, event.EventType = "event", "event", "fixture_event"
	occurred := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	event.OccurredAt = &occurred
	rows := []service.AICandidate{candidate, event}
	ids, err := store.StageAICandidates(ctx, "", runID, strings.Repeat("a", 64), rows)
	if err != nil || len(ids) != 2 {
		t.Fatalf("native stage: %v %v", ids, err)
	}
	var attrs []byte
	if err = tx.QueryRow(ctx, `SELECT attrs FROM working.candidate_entity WHERE id=$1::uuid`, ids[0]).Scan(&attrs); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(attrs), quote) || !strings.Contains(string(attrs), candidate.EvidenceBundleSHA256) || !strings.Contains(string(attrs), candidate.SourceSpan.SHA256) {
		t.Fatal("PostgreSQL contains native excerpt or omitted external evidence pins")
	}
	var summary string
	if err = tx.QueryRow(ctx, `SELECT attrs,summary FROM working.candidate_event WHERE id=$1::uuid`, ids[1]).Scan(&attrs, &summary); err != nil || summary != event.Statement || strings.Contains(string(attrs), quote) {
		t.Fatal("dated event lost typed summary or retained chat quote")
	}
	var facts int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM working.candidate_fact`).Scan(&facts); err != nil || facts != 0 {
		t.Fatal("native import created a fact row")
	}
	again, err := store.StageAICandidates(ctx, "", runID, strings.Repeat("a", 64), rows)
	if err != nil || len(again) != 2 || again[0] != ids[0] || again[1] != ids[1] {
		t.Fatal("quote-free native staging retry changed identity")
	}
}
