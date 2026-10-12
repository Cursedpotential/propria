package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testNativePackageCandidate proves real working edits, held acceptance and canonical decision commit in disposable PostgreSQL.
// Inputs: the verified backend_package_review_* fixture pool. Output: replacement/hash/ledger/provenance assertions.
// Effects: synthetic native files and tables in that isolated test database only. Pick after base correction fixtures exist.
func testNativePackageCandidate(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	ddl := `CREATE SCHEMA working; CREATE SCHEMA analysis;
ALTER TABLE context.source_version ADD COLUMN declared_format text,ADD COLUMN original_object_id uuid;
ALTER TABLE context.source_metadata ADD COLUMN source_version_id uuid,ADD COLUMN metadata_class text,ADD COLUMN extractor_id text,ADD COLUMN metadata jsonb,ADD COLUMN generated_at timestamptz DEFAULT now();
CREATE TABLE working.extraction_run(id uuid PRIMARY KEY,extractor text,extractor_version text,source_summary text,status text,finished_at timestamptz,stats jsonb);
CREATE TABLE working.candidate_fact(id uuid PRIMARY KEY,extraction_run_id uuid,source_raw_table text,source_raw_id text,attrs jsonb,content_sha256 bytea,review_state text,confidence double precision,
 statement text,predicate text,evidence_quote text,domain text,ontology_version text,created_at timestamptz DEFAULT now(),promoted_to_table text,promoted_to_id uuid,promoted_at timestamptz);
CREATE TABLE working.candidate_entity(LIKE working.candidate_fact INCLUDING ALL);
CREATE TABLE working.candidate_event(LIKE working.candidate_fact INCLUDING ALL);
CREATE TABLE analysis.review_decision(decision_id uuid PRIMARY KEY,target_kind text,target_id uuid,reviewer text,decision text,court_readiness text,rationale text,decided_at timestamptz);`
	if _, err := pool.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("AI_CONTEXT_ROOT", root)
	original := []byte("A source statement.")
	originalPath, preparedPath := filepath.Join(root, "original.md"), filepath.Join(root, "prepared.json")
	if err := os.WriteFile(originalPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	ref := "b2://synthetic/SourceCorpus/documents/native.md"
	version := "66666666-6666-4666-8666-666666666666"
	preparedRef := "file://" + filepath.ToSlash(preparedPath)
	prepared, _ := json.Marshal(aicontextsource.Prepared{ContractVersion: "ai-context-v1", Stage: "prepared",
		Source:      aicontextsource.Identity{SourceRef: ref, SourceFormat: "gemini_markdown"},
		OriginalRef: "file://" + filepath.ToSlash(originalPath), OriginalSHA256: aicontextsource.Hash(original)})
	if err := os.WriteFile(preparedPath, prepared, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO registry.matter VALUES($1::uuid)`, caseidentity.AuthoritativeMatterID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO registry.court_case VALUES($1::uuid,$2::uuid)`, caseidentity.AuthoritativeCourtCaseID, caseidentity.AuthoritativeMatterID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO context.source VALUES('77777777-7777-4777-8777-777777777777',$1)`, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO context.source_version VALUES($1::uuid,'77777777-7777-4777-8777-777777777777',$2::uuid,$3::uuid,'gemini_markdown',NULL)`, version, caseidentity.AuthoritativeMatterID, caseidentity.AuthoritativeCourtCaseID); err != nil {
		t.Fatal(err)
	}
	metadata, _ := json.Marshal(map[string]any{"source_ref": ref, "declared_format": "gemini_markdown"})
	if _, err := pool.Exec(ctx, `INSERT INTO context.source_metadata(source_version_id,metadata_class,extractor_id,metadata) VALUES($1::uuid,'record_native','context-source-registration',$2::jsonb)`, version, metadata); err != nil {
		t.Fatal(err)
	}
	candidate := service.AICandidate{AISourcePin: service.AISourcePin{SourceVersionID: version, SourceRef: ref, PreparedRef: preparedRef, SourceSHA256: aicontextsource.Hash(original)},
		SourceSpan: service.AISourceSpan{Start: 2, End: 8, SHA256: aicontextsource.Hash([]byte("source"))}, SpanUnit: "unicode_codepoint",
		Kind: "fact", ReportedKind: "fact", ReviewDomain: "ai_chat_content", Predicate: "reports", Statement: "Source assertion", EvidenceQuote: "source", Confidence: 0.5}
	hash := service.AICandidateDigest(candidate)
	id := flow.DeterministicID("ai_candidate", version, hex.EncodeToString(hash[:]))
	attrs, _ := json.Marshal(map[string]any{"candidate": candidate, "reported_kind": "fact", "review_domain": "ai_chat_content", "graph_promotion": "held", "projection_scope": "none",
		"placement": map[string]any{"category": "documents", "package_ref": "b2://synthetic/unit"}})
	if _, err := pool.Exec(ctx, `INSERT INTO working.candidate_fact(id,extraction_run_id,source_raw_table,source_raw_id,attrs,content_sha256,review_state,confidence,statement,predicate,evidence_quote,domain,ontology_version)
 VALUES($1::uuid,'88888888-8888-4888-8888-888888888888','context.source_version',$2,$3::jsonb,$4,'pending',0.5,'Source assertion','reports','source','context','probata.ai_content/v1')`, id, version, attrs, hash[:]); err != nil {
		t.Fatal(err)
	}
	store, _ := NewSourceMetadataStore(pool)
	spec := PackageReviewSpec{SourceVersionRef: version, ItemID: "candidate_fact:" + id, Operation: "edit", CandidateContentSHA256: hex.EncodeToString(hash[:]),
		Fields: map[string]any{"statement": "Owner corrected assertion"}, SourceRefs: []map[string]any{{"source_ref": ref, "source_version_ref": version, "locator": map[string]any{"prepared_ref": preparedRef, "start": 2, "end": 8}}},
		Reason: "Owner correction", ActorSubjectUID: "synthetic-owner", ActorUsername: "owner", IdempotencyKey: "native-edit"}
	first, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	var newID, newHash, originalState, originalStatement, category string
	if err := pool.QueryRow(ctx, `SELECT corrected_value->'canonical'->>'candidate_id',corrected_value->'canonical'->>'content_sha256' FROM context.source_metadata_correction WHERE correction_ref=$1::uuid`, first.CorrectionRef).Scan(&newID, &newHash); err != nil {
		t.Fatal(err)
	}
	if newID == id || newHash == spec.CandidateContentSHA256 {
		t.Fatal("edit did not create a real replacement candidate")
	}
	if err := pool.QueryRow(ctx, `SELECT review_state,statement FROM working.candidate_fact WHERE id=$1::uuid`, id).Scan(&originalState, &originalStatement); err != nil || originalState != "superseded" || originalStatement != "Source assertion" {
		t.Fatalf("original changed: %s %s %v", originalState, originalStatement, err)
	}
	if err := pool.QueryRow(ctx, `SELECT attrs->'placement'->>'category' FROM working.candidate_fact WHERE id=$1::uuid`, newID).Scan(&category); err != nil || category != "documents" {
		t.Fatal("replacement lost category inheritance")
	}
	spec.Operation, spec.Revision, spec.IdempotencyKey, spec.Fields, spec.CandidateContentSHA256 = "accept", first.CorrectionRef, "native-accept", nil, newHash
	accepted, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	var promotion string
	var decisions int
	if err := pool.QueryRow(ctx, `SELECT attrs->>'graph_promotion',(SELECT count(*) FROM analysis.review_decision) FROM working.candidate_fact WHERE id=$1::uuid`, newID).Scan(&promotion, &decisions); err != nil || promotion != "held" || decisions != 0 {
		t.Fatal("accept crossed canonical commit boundary")
	}
	spec.Operation, spec.Revision, spec.IdempotencyKey = "commit", accepted.CorrectionRef, "native-commit"
	if _, err := pool.Exec(ctx, `CREATE FUNCTION analysis.refuse_test_decision() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN RAISE EXCEPTION ''synthetic decision failure''; END'; CREATE TRIGGER refuse_test_decision BEFORE INSERT ON analysis.review_decision FOR EACH ROW EXECUTE FUNCTION analysis.refuse_test_decision()`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PersistPackageReview(ctx, spec); err == nil {
		t.Fatal("forced canonical writer failure succeeded")
	}
	var head string
	if err := pool.QueryRow(ctx, `SELECT correction_ref::text FROM context.source_metadata_correction WHERE source_version_id=$1::uuid ORDER BY revision DESC LIMIT 1`, version).Scan(&head); err != nil || head != accepted.CorrectionRef {
		t.Fatal("failed decision advanced the correction chain")
	}
	if err := pool.QueryRow(ctx, `SELECT attrs->>'graph_promotion' FROM working.candidate_fact WHERE id=$1::uuid`, newID).Scan(&promotion); err != nil || promotion != "held" {
		t.Fatal("failed transaction changed canonical eligibility")
	}
	if _, err := pool.Exec(ctx, `DROP TRIGGER refuse_test_decision ON analysis.review_decision`); err != nil {
		t.Fatal(err)
	}
	committed, err := store.PersistPackageReview(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT attrs->>'graph_promotion',(SELECT count(*) FROM analysis.review_decision) FROM working.candidate_fact WHERE id=$1::uuid`, newID).Scan(&promotion, &decisions); err != nil || promotion != "approved_context" || decisions != 1 {
		t.Fatalf("canonical decision not written: %s %d %v", promotion, decisions, err)
	}
	replay, err := store.PersistPackageReview(ctx, spec)
	if err != nil || replay.CorrectionRef != committed.CorrectionRef {
		t.Fatalf("canonical replay %v %v", replay, err)
	}
	// A later model attempt may repeat the old candidate; preserve its occurrence without reviving owner-superseded content.
	stageStore := &EntityExtractionStore{db: pool}
	attemptID := flow.DeterministicID("synthetic-model-attempt", version)
	ids, err := stageStore.StageAICandidates(ctx, "", attemptID, strings.Repeat("c", 64), []service.AICandidate{candidate})
	if err != nil || len(ids) != 1 || ids[0] != id {
		t.Fatalf("repeat model occurrence lost: %v %v", ids, err)
	}
	var attemptCandidate string
	if err := pool.QueryRow(ctx, `SELECT stats->'candidate_ids'->>0 FROM working.extraction_run WHERE id=$1::uuid`, attemptID).Scan(&attemptCandidate); err != nil || attemptCandidate != id {
		t.Fatal("retry occurrence was not durably retained")
	}
	if err := pool.QueryRow(ctx, `SELECT review_state FROM working.candidate_fact WHERE id=$1::uuid`, id).Scan(&originalState); err != nil || originalState != "superseded" {
		t.Fatal("model retry revived owner-superseded candidate")
	}
	if strings.Contains(string(prepared), "Owner corrected assertion") {
		t.Fatal("source bundle was edited")
	}
}
