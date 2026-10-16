package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	storepg "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/sourcecontext"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// rollbackOperatorDB makes existing store transactions actual PostgreSQL savepoints.
// Input: one outer synthetic transaction. Output: existing DB seam; effects:
// nested commits release savepoints, while the outer rollback removes all fixtures.
type rollbackOperatorDB struct{ pgx.Tx }

// BeginTx creates a real nested savepoint for the existing store write method.
// Inputs: context/options. Output: nested transaction; never commits the outer fixture.
func (db rollbackOperatorDB) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return db.Tx.Begin(ctx)
}

// operatorFixtureBundle carries only the companion script's actual synthetic extraction.
// Inputs: retained JSON fixture. Output: exact messages/page; effects: none.
type operatorFixtureBundle struct {
	Contract string
	Messages []struct {
		RecordID   string    `json:"record_id"`
		Ordinal    int64     `json:"ordinal"`
		OccurredAt time.Time `json:"occurred_at"`
		Body       string
		Sender     string
		Recipients []string
	}
	Page json.RawMessage
}

// TestOperatorResultsRealSemanticaStoreAndCorrections verifies real extractor output and audited writes.
// Inputs: exact dedicated test DSN and real Semantica fixture produced by the companion script.
// Output: grounded compare-only entity/event rows, alias history/reopen and source-context
// supersession/control assertions. Effects: actual PostgreSQL writes within one outer
// rollback only; no Temporal worker, source ingestion, model, production or evidence writes.
func TestOperatorResultsRealSemanticaStoreAndCorrections(t *testing.T) {
	dsn := os.Getenv("PROPRIA_OPERATOR_REPAIR_TEST_DSN")
	fixturePath := os.Getenv("PROPRIA_OPERATOR_SEMANTICA_FIXTURE")
	if dsn == "" || fixturePath == "" {
		t.Skip("dedicated DB and actual deterministic fixture required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	config, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "backend_package_review_operator_results_20261015", config.ConnConfig.Database, "never use another database")
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	defer pool.Close()
	var current string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&current))
	require.Equal(t, "backend_package_review_operator_results_20261015", current)
	var schemas int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM pg_namespace WHERE nspname IN ('context','working','registry')").Scan(&schemas))
	require.Zero(t, schemas, "fixture DB must be empty")
	outer, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer outer.Rollback(context.Background())
	ddl, err := os.ReadFile("testdata/operator_results_fixture_20261015.sql")
	require.NoError(t, err)
	_, err = outer.Exec(ctx, string(ddl))
	require.NoError(t, err)
	db := rollbackOperatorDB{outer}
	raw, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	var bundle operatorFixtureBundle
	require.NoError(t, json.Unmarshal(raw, &bundle))
	require.Equal(t, "synthetic-semantic-store-proof/1", bundle.Contract)
	require.Len(t, bundle.Messages, 4)
	digest := sha256.Sum256(raw)
	t.Logf("real Semantica fixture sha256=%x messages=%d", digest, len(bundle.Messages))
	const sourceID = "10000000-0000-4000-8000-000000000001"
	const sourceVersion = "10000000-0000-4000-8000-000000000002"
	const rawGeneration = "10000000-0000-4000-8000-000000000003"
	const generation = "10000000-0000-4000-8000-000000000004"
	const handle = "synthetic-human-semantic-operation"
	const sourceRef = "b2://synthetic-operator-fixtures/human-thread.ndjson?versionId=fixture-v1"
	_, err = outer.Exec(ctx, `INSERT INTO context.source VALUES($1::uuid,$2);
 INSERT INTO context.source_version(id,source_id,workflow_id,matter_id,court_case_id,declared_format,version_ordinal) VALUES($3::uuid,$1::uuid,'synthetic-human-request',$4::uuid,$5::uuid,'sms_backup_restore_xml',1);
 INSERT INTO context.raw_generation VALUES($6::uuid,$3::uuid);
 INSERT INTO context.normalized_generation(id,source_version_id,raw_generation_id) VALUES($7::uuid,$3::uuid,$6::uuid);
 INSERT INTO context.proffer_preview_binding(preview_handle,request_id,source_ref,workflow_id,run_id,parser_options_ref) VALUES($8,'synthetic-human-request',$2,'synthetic-human-request','fixture-run','fixture-parser');
 INSERT INTO context.proffer_preview_snapshot VALUES($8,1,$3::uuid,$6::uuid,$7::uuid)`, pgx.QueryExecModeSimpleProtocol, sourceID, sourceRef, sourceVersion, caseidentity.AuthoritativeMatterID, caseidentity.AuthoritativeCourtCaseID, rawGeneration, generation, handle)
	require.NoError(t, err)
	modeReceipt, _ := json.Marshal(map[string]string{"operating_mode": "LIVE", "matter_id": caseidentity.AuthoritativeMatterID, "court_case_id": caseidentity.AuthoritativeCourtCaseID})
	_, err = outer.Exec(ctx, "INSERT INTO context.proffer_preview_event VALUES($1,0,'phase_changed',now(),'starting',$2)", handle, string(modeReceipt))
	require.NoError(t, err)
	for _, message := range bundle.Messages {
		participants := []map[string]string{{"role": "sender", "identifier": message.Sender}}
		for _, recipient := range message.Recipients {
			participants = append(participants, map[string]string{"role": "recipient", "identifier": recipient})
		}
		payload, _ := json.Marshal(map[string]any{"content": map[string]string{"body": message.Body}, "participants": participants, "source_available_from": message.OccurredAt})
		_, err = outer.Exec(ctx, "INSERT INTO context.normalized_record_identity VALUES($1::uuid,$2::uuid,$3::uuid,$4,'message',$5,$6::jsonb)", message.RecordID, sourceVersion, generation, message.Ordinal, message.OccurredAt, payload)
		require.NoError(t, err)
	}
	var originalsBefore string
	require.NoError(t, outer.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(n) ORDER BY id)::text FROM context.normalized_record_identity n").Scan(&originalsBefore))
	extraction, err := storepg.NewEntityExtractionStore(db)
	require.NoError(t, err)
	run, err := extraction.ResolveRun(ctx, handle)
	require.NoError(t, err)
	require.Equal(t, "LIVE", run.MatterMode)
	require.Equal(t, generation, run.GenerationID)
	spec, ok := flow.ExtractorByID("semantica")
	require.True(t, ok)
	require.True(t, spec.CompareOnly)
	input := flow.ExternalRunInput{ExtractionID: uuid.NewString(), Run: run, Extractor: "semantica", Actor: entities.Actor{SubjectUID: "fixture:owner", Username: "fixture-owner"}}
	acts := activities.ConversationActivities{Store: extraction}
	_, err = acts.BeginExternalExtractionRun(ctx, flow.ExternalRunStart{Input: input, Spec: spec})
	require.NoError(t, err)
	stage := flow.StageExternalPage{Input: input, RunID: input.ExternalRunID(), Request: flow.ExternalPageRequest{GenerationID: generation, SourceVersionID: sourceVersion, PreviewHandle: handle, Extractor: "semantica", AfterOrdinal: -1, Limit: 200}, Page: bundle.Page}
	result, err := acts.StageExternalExtractionPage(ctx, stage)
	require.NoError(t, err)
	require.False(t, result.Invalid, result.Reason)
	require.Equal(t, 4, result.Messages)
	require.Greater(t, result.Entities, 0)
	require.Greater(t, result.Events, 0)
	replay, err := acts.StageExternalExtractionPage(ctx, stage)
	require.NoError(t, err)
	require.Equal(t, result.Entities, replay.Entities)
	require.Equal(t, result.Events, replay.Events)
	err = acts.FinishExternalExtractionRun(ctx, flow.FinishExternalRun{Run: run, RunID: input.ExternalRunID(), Status: "completed", Stats: map[string]any{"messages": result.Messages, "entities": result.Entities, "events": result.Events}})
	require.NoError(t, err)
	exported, err := extraction.ConversationExtractions(ctx, []string{generation})
	require.NoError(t, err)
	require.Len(t, exported.Runs, 1)
	require.Equal(t, "completed", exported.Runs[0].Status)
	require.True(t, exported.Runs[0].CompareOnly)
	require.Len(t, exported.Entities, result.Entities)
	require.Len(t, exported.Events, result.Events)
	proposals, err := extraction.CurrentEntities(ctx, generation)
	require.NoError(t, err)
	require.Empty(t, proposals, "compare-only rows must stay out of review proposals")
	eventProposals, err := extraction.CurrentEvents(ctx, generation)
	require.NoError(t, err)
	require.Empty(t, eventProposals)
	var invalidCitations int
	require.NoError(t, outer.QueryRow(ctx, `SELECT count(*) FROM working.candidate_entity c
 CROSS JOIN LATERAL jsonb_array_elements(c.attrs->'model_mentions') mention
 WHERE NOT EXISTS(SELECT 1 FROM context.normalized_record_identity n WHERE n.id::text=mention->>'record_id' AND n.normalized_generation_id=$1::uuid)`, generation).Scan(&invalidCitations))
	require.Zero(t, invalidCitations)
	require.NoError(t, outer.QueryRow(ctx, `SELECT count(*) FROM working.candidate_event c
 CROSS JOIN LATERAL jsonb_array_elements(c.attrs->'source_records') citation
 WHERE NOT EXISTS(SELECT 1 FROM context.normalized_record_identity n WHERE n.id::text=citation->>'record_id' AND n.normalized_generation_id=$1::uuid)`, generation).Scan(&invalidCitations))
	require.Zero(t, invalidCitations)
	var entityCitations, eventCitations int
	require.NoError(t, outer.QueryRow(ctx, "SELECT count(*) FROM working.candidate_entity c CROSS JOIN LATERAL jsonb_array_elements(c.attrs->'model_mentions') mention").Scan(&entityCitations))
	require.Greater(t, entityCitations, 0)
	require.NoError(t, outer.QueryRow(ctx, "SELECT count(*) FROM working.candidate_event c CROSS JOIN LATERAL jsonb_array_elements(c.attrs->'source_records') citation").Scan(&eventCitations))
	require.Greater(t, eventCitations, 0)
	t.Logf("actual citations entity_mentions=%d event_records=%d all resolve to fixture generation", entityCitations, eventCitations)
	t.Logf("actual Activity/store/readback entities=%d events=%d messages=%d replay_no_duplicates=true compare_only=true", len(exported.Entities), len(exported.Events), result.Messages)
	verifyOperatorPreflightCoverage(t, ctx, db, extraction, sourceRef)
	verifyOperatorAliasCorrection(t, ctx, db)
	verifyOperatorSourceRevision(t, ctx, db, sourceRef, raw)
	var originalsAfter string
	require.NoError(t, outer.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(n) ORDER BY id)::text FROM context.normalized_record_identity n").Scan(&originalsAfter))
	require.Equal(t, originalsBefore, originalsAfter, "corrections must not rewrite normalized records")
	require.NoError(t, outer.Rollback(ctx))
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM pg_namespace WHERE nspname IN ('context','working','registry')").Scan(&schemas))
	require.Zero(t, schemas)
	t.Log("outer rollback readback: fixture schemas=0; retained database remains empty")
}

// verifyOperatorPreflightCoverage distinguishes real counted, unqueried and partial workloads.
// Inputs: real four-message fixture and canonical store. Output: actual SQL coverage assertions.
// Effects: synthetic fixture savepoints only; never fabricates counts or queries AI chat generations.
func verifyOperatorPreflightCoverage(t *testing.T, ctx context.Context, db rollbackOperatorDB, store *storepg.EntityExtractionStore, sourceRef string) {
	ref := flow.ConversationRef{ExportKey: sourceRef, Conv: "human-thread.ndjson?versionId=fixture-v1"}
	check, err := store.PreflightConversations(ctx, caseidentity.AuthoritativeMatterID, []flow.ConversationRef{ref}, []string{"semantica"})
	require.NoError(t, err)
	require.Len(t, check.Targets, 1)
	available := check.Targets[0]
	require.Equal(t, "available", available.WorkloadStatus)
	require.EqualValues(t, 4, available.Messages)
	require.Equal(t, 1, available.QueriedGenerations)
	require.Equal(t, 1, available.ResolvedGenerations)
	_, err = db.Exec(ctx, `INSERT INTO context.source VALUES('30000000-0000-4000-8000-000000000001','b2://synthetic-operator-fixtures/ai.json'),('30000000-0000-4000-8000-000000000002','b2://synthetic-operator-fixtures/empty-human.xml');
 INSERT INTO context.source_version(id,source_id,workflow_id,matter_id,court_case_id,declared_format,version_ordinal) VALUES
 ('30000000-0000-4000-8000-000000000003','30000000-0000-4000-8000-000000000001','synthetic-ai-only',$1::uuid,$2::uuid,'chatgpt',1),
 ('30000000-0000-4000-8000-000000000004','30000000-0000-4000-8000-000000000002','synthetic-no-generation',$1::uuid,$2::uuid,'sms_backup_restore_xml',1)`, pgx.QueryExecModeSimpleProtocol, caseidentity.AuthoritativeMatterID, caseidentity.AuthoritativeCourtCaseID)
	require.NoError(t, err)
	for _, unqueried := range []flow.ConversationRef{{ExportKey: "b2://synthetic-operator-fixtures/ai.json", Conv: "ai.json"}, {ExportKey: "b2://synthetic-operator-fixtures/empty-human.xml", Conv: "empty-human.xml"}} {
		check, err = store.PreflightConversations(ctx, caseidentity.AuthoritativeMatterID, []flow.ConversationRef{unqueried}, []string{"semantica"})
		require.NoError(t, err)
		require.False(t, check.Eligible)
		require.Equal(t, "not_queried", check.Targets[0].WorkloadStatus)
		require.Zero(t, check.Targets[0].QueriedGenerations)
		for _, blocker := range check.Blockers {
			require.NotContains(t, blocker, "no message records available", "unqueried is not a queried zero")
		}
	}
	derivedKey := sourceRef + ".derived/threads/" + ref.Conv + ".ndjson"
	_, err = db.Exec(ctx, `INSERT INTO context.source VALUES('30000000-0000-4000-8000-000000000005',$1);
 INSERT INTO context.source_version(id,source_id,workflow_id,matter_id,court_case_id,declared_format,version_ordinal) VALUES('30000000-0000-4000-8000-000000000006','30000000-0000-4000-8000-000000000005','synthetic-unresolved-operation',$2::uuid,$3::uuid,'sms_backup_restore_xml',2);
 INSERT INTO context.normalized_generation(id,source_version_id,raw_generation_id) VALUES('30000000-0000-4000-8000-000000000007','30000000-0000-4000-8000-000000000006','30000000-0000-4000-8000-000000000008')`, pgx.QueryExecModeSimpleProtocol, derivedKey, caseidentity.AuthoritativeMatterID, caseidentity.AuthoritativeCourtCaseID)
	require.NoError(t, err)
	check, err = store.PreflightConversations(ctx, caseidentity.AuthoritativeMatterID, []flow.ConversationRef{ref}, []string{"semantica"})
	require.NoError(t, err)
	require.False(t, check.Eligible)
	partial := check.Targets[0]
	require.Equal(t, "partial", partial.WorkloadStatus)
	require.Equal(t, 1, partial.QueriedGenerations)
	require.Equal(t, 2, partial.ResolvedGenerations)
	require.EqualValues(t, 4, partial.Messages)
	require.Equal(t, available.Characters, partial.Characters)
	require.Equal(t, available.UTF8Bytes, partial.UTF8Bytes)
	tokenBefore := flow.PreflightToken(flow.ExtractionPreflight{Targets: []flow.ExtractionPreflightTarget{available}})
	tokenAfter := flow.PreflightToken(flow.ExtractionPreflight{Targets: []flow.ExtractionPreflightTarget{partial}})
	require.NotEqual(t, tokenBefore, tokenAfter, "coverage is part of the confirmed workload")
	t.Log("actual preflight coverage: complete4messages; AI/no-generation not_queried; unresolved sibling partial observed4of2generations")
}

// verifyOperatorAliasCorrection exercises the real global alias edit and read lookup.
// Inputs: rolled-back fixture DB. Output: history/replay/control assertions; effects: synthetic savepoints only.
func verifyOperatorAliasCorrection(t *testing.T, ctx context.Context, db rollbackOperatorDB) {
	const first = "20000000-0000-4000-8000-000000000001"
	const corrected = "20000000-0000-4000-8000-000000000002"
	const unrelated = "20000000-0000-4000-8000-000000000003"
	const alias = "20000000-0000-4000-8000-000000000004"
	const control = "20000000-0000-4000-8000-000000000005"
	_, err := db.Exec(ctx, `INSERT INTO registry.entity VALUES($1::uuid,'Initial fixture person',NULL),($2::uuid,'Corrected fixture person',NULL),($3::uuid,'Unrelated fixture person',NULL);
 INSERT INTO registry.person VALUES($1::uuid),($2::uuid),($3::uuid);
 INSERT INTO registry.entity_alias(id,entity_id,alias_text,alias_kind,normalized,status,basis,recorded_by) VALUES($4::uuid,$1::uuid,'5555550101','phone','5555550101','confirmed','synthetic initial','fixture-owner'),($5::uuid,$3::uuid,'5555550102','phone','5555550102','confirmed','unrelated control','fixture-owner')`, pgx.QueryExecModeSimpleProtocol, first, corrected, unrelated, alias, control)
	require.NoError(t, err)
	store, err := storepg.NewCaseIdentityStore(db)
	require.NoError(t, err)
	before, err := store.Lookup(ctx, []string{"5555550101", "5555550102"})
	require.NoError(t, err)
	require.Len(t, before, 2)
	target, basis, period := corrected, "Owner-confirmed synthetic correction", "2025 descriptive"
	actor := caseidentity.Actor{SubjectUID: "fixture:owner", Username: "fixture-owner", IdempotencyKey: "fixture-alias-correction"}
	spec := caseidentity.IdentifierEditSpec{ID: alias, Fields: map[string]*string{"entity_id": &target, "basis": &basis, "period": &period}, ChangeReason: "Correct selected synthetic alias only"}
	receipt, err := store.EditIdentifier(ctx, spec, actor)
	require.NoError(t, err)
	repeated, err := store.EditIdentifier(ctx, spec, actor)
	require.NoError(t, err)
	require.True(t, repeated.Replayed)
	require.Equal(t, receipt.Ref, repeated.Ref)
	reopened, err := store.Lookup(ctx, []string{"5555550101", "5555550102"})
	require.NoError(t, err)
	require.NotNil(t, reopened[0].EntityID)
	require.NotNil(t, reopened[1].EntityID)
	require.Equal(t, corrected, *reopened[0].EntityID)
	require.Equal(t, unrelated, *reopened[1].EntityID)
	require.Equal(t, before[1], reopened[1])
	var changes int
	var oldOwner, newOwner, reason, uid, username string
	var recordedAt time.Time
	require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM registry.identity_change").Scan(&changes))
	require.Equal(t, 1, changes)
	require.NoError(t, db.QueryRow(ctx, "SELECT before_state->>'entity_id',after_state->>'entity_id',change_reason,recorded_by_uid,recorded_by,recorded_at FROM registry.identity_change").Scan(&oldOwner, &newOwner, &reason, &uid, &username, &recordedAt))
	require.Equal(t, first, oldOwner)
	require.Equal(t, corrected, newOwner)
	require.Equal(t, spec.ChangeReason, reason)
	require.Equal(t, actor.SubjectUID, uid)
	require.Equal(t, actor.Username, username)
	require.False(t, recordedAt.IsZero())
	require.WithinDuration(t, receipt.RecordedAt, recordedAt, time.Microsecond)
	t.Log("actual EditIdentifier+Lookup: one audited global alias revision, replay stable, unrelated alias unchanged")
	t.Logf("alias receipt=%s actor=%s uid=%s rationale=%q at=%s before_person=%s after_person=%s unrelated_person=%s", receipt.Ref, username, uid, reason, recordedAt.UTC().Format(time.RFC3339Nano), oldOwner, newOwner, unrelated)
}

// verifyOperatorSourceRevision exercises real append/reopen and source-scope refusal.
// Inputs: immutable synthetic observation. Output: prior/current/control assertions.
// Effects: fixture savepoints only; neither normalized history nor unrelated request changes.
func verifyOperatorSourceRevision(t *testing.T, ctx context.Context, db rollbackOperatorDB, sourceRef string, fixtureBytes []byte) {
	store, err := storepg.NewSourceContextStore(db)
	require.NoError(t, err)
	spec := sourcecontext.Spec{OperatingMode: "LIVE", RequestID: "synthetic-source-context", MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, SourceRef: sourceRef,
		ObservedSource: sourcecontext.ObservedSource{Key: sourceRef, Name: "human-thread.ndjson", ByteLength: int64(len(fixtureBytes)), ETag: "fixture-etag", PreviewSHA256: fmt.Sprintf("%x", sha256.Sum256(fixtureBytes)), VerificationState: "fixture"},
		Assertions:     sourcecontext.HumanAssertions{SourceClass: "unknown", Notes: "initial synthetic context"}, ChangeReason: "Initial synthetic context", ActorSubjectUID: "fixture:owner", ActorUsername: "fixture-owner", IdempotencyKey: "fixture-context-initial", ContentDigest: sha256.Sum256([]byte("initial fixture context"))}
	first, err := store.PersistSourceContext(ctx, spec)
	require.NoError(t, err)
	control := spec
	control.RequestID = "unrelated-source-context"
	control.IdempotencyKey = "fixture-context-control"
	control.Assertions.Notes = "unrelated control"
	_, err = store.PersistSourceContext(ctx, control)
	require.NoError(t, err)
	controlBefore, found, err := store.CurrentSourceContext(ctx, control.RequestID, sourceRef)
	require.NoError(t, err)
	require.True(t, found)
	second := spec
	second.SupersedesRef = first.SourceContextRef
	second.IdempotencyKey = "fixture-context-corrected"
	second.Assertions.Notes = "owner-confirmed synthetic correction"
	second.ChangeReason = "Correct selected source context only"
	second.ContentDigest = sha256.Sum256([]byte("corrected fixture context"))
	receipt, err := store.PersistSourceContext(ctx, second)
	require.NoError(t, err)
	require.Equal(t, 2, receipt.Revision)
	reopened, found, err := store.CurrentSourceContext(ctx, spec.RequestID, sourceRef)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 2, reopened.Revision)
	require.Equal(t, second.Assertions, reopened.Assertions)
	require.Equal(t, second.ActorUsername, reopened.ActorUsername)
	require.Equal(t, second.ChangeReason, reopened.ChangeReason)
	require.False(t, reopened.RecordedAt.IsZero())
	var oldNotes string
	require.NoError(t, db.QueryRow(ctx, "SELECT assertions->>'notes' FROM context.proffer_source_context_revision WHERE source_context_ref=$1::uuid", first.SourceContextRef).Scan(&oldNotes))
	require.Equal(t, spec.Assertions.Notes, oldNotes)
	controlAfter, found, err := store.CurrentSourceContext(ctx, control.RequestID, sourceRef)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, controlBefore, controlAfter)
	bad := second
	bad.RequestID = "another-request"
	bad.IdempotencyKey = "fixture-context-wrong-scope"
	_, err = store.PersistSourceContext(ctx, bad)
	require.Error(t, err)
	t.Log("actual PersistSourceContext+CurrentSourceContext: revision2 reopened, prior/control unchanged, cross-request supersession refused")
	t.Logf("source context prior_ref=%s current_ref=%s revision=%d actor=%s rationale=%q at=%s", first.SourceContextRef, reopened.SourceContextRef, reopened.Revision, reopened.ActorUsername, reopened.ChangeReason, reopened.RecordedAt.UTC().Format(time.RFC3339Nano))
}
