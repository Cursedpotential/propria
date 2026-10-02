//go:build d04live

// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (participant resolution; owner-participation split)
//
// Live proof of the first-party context import (D04) against a real
// PostgreSQL 18 built from sql/bootstrap/schema_snapshot_20260907.sql. Never
// run against the platform database: it seeds fixtures. Run it against a
// disposable database created from the snapshot and dropped afterwards:
//
//	D04_LIVE_ADMIN_DSN=postgres://ai:...@100.91.190.107:5432/d04_probe_20261001 \
//	  go test -tags d04live -run TestFirstPartyContextImportLive ./postgres/
//
// The fixtures are written by the superuser with triggers suspended; every
// D04 write runs as platform_runtime (SET ROLE on every pooled connection),
// so the snapshot's grants, guard triggers and deferred validators are what
// is being proven.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/temporal"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
)

const (
	liveOwner       = "0d040000-0000-7000-8000-00000000f001"
	livePerspective = "0d040000-0000-7000-8000-00000000f002"
	livePrefix      = "b2://d04-probe/vault/sms-2024-11-24.xml.derived/"
	liveOwnerPhone  = "+18105550100"
	liveOtherPhone  = "+18105550199"
)

type liveChunk struct {
	requestID, sourceVersion, generation, verification string
	records                                            []liveRecord
}

type liveRecord struct {
	id, sender string
	recipients []string
	at         time.Time
	body       string
}

func mustV7(t *testing.T) string {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

func seedLive(t *testing.T, ctx context.Context, admin *pgxpool.Pool, chunks []liveChunk) {
	t.Helper()
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	exec(`SET LOCAL session_replication_role = replica`)
	exec(`INSERT INTO registry.matter (id, title, created_by) VALUES ($1, 'D04 live probe matter', 'd04-live-test') ON CONFLICT DO NOTHING`, devMatterID)
	exec(`INSERT INTO registry.court_case (id, matter_id, caption, created_by) VALUES ($1, $2, 'D04 live probe case', 'd04-live-test') ON CONFLICT DO NOTHING`, devCourtCaseID, devMatterID)
	exec(`INSERT INTO registry.entity (id, entity_type) VALUES ($1, 'person'), ($2, 'person') ON CONFLICT DO NOTHING`, liveOwner, livePerspective)
	exec(`INSERT INTO registry.person (id, role_in_case) VALUES ($1, 'user'), ($2, 'partner') ON CONFLICT DO NOTHING`, liveOwner, livePerspective)
	// The owner's phone is a CONFIRMED identifier; the other number is unknown.
	exec(`INSERT INTO registry.entity_alias (id, entity_id, alias_text, alias_kind, status, recorded_by)
	      VALUES ($1, $2, $3, 'phone', 'confirmed', 'd04-live-test')`, mustV7(t), liveOwner, liveOwnerPhone)

	// The derivation that published the chunks: one derive receipt naming the
	// derived prefix, as derive_sms_threads_activity records it.
	parentSource, parentVersion := mustV7(t), mustV7(t)
	exec(`INSERT INTO context.source (id, source_key, provenance_class) VALUES ($1, $2, 'unknown')`, parentSource, "b2://d04-probe/vault/sms-2024-11-24.xml")
	exec(`INSERT INTO context.source_version (id, source_id, version_ordinal, workflow_id, submission_idempotency_key, declared_format, acquired_at, status, matter_id, court_case_id, original_object_id)
	      VALUES ($1, $2, 1, 'd04-probe-parent', 'd04-probe-parent', 'smsbackuprestore_xml', now(), 'retained', $3, $4, $5)`, parentVersion, parentSource, devMatterID, devCourtCaseID, mustV7(t))
	deriveExecution, deriveReceipt := mustV7(t), mustV7(t)
	exec(`INSERT INTO context.activity_execution (id, source_version_id, workflow_id, activity_name, idempotency_key) VALUES ($1, $2, 'd04-probe-parent', 'derive_sms_threads_activity', 'd04-probe')`, deriveExecution, parentVersion)
	result, _ := json.Marshal(map[string]any{"ref_kind": "derived_structured_text", "ref_id": mustV7(t), "derived_prefix": livePrefix})
	exec(`INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref) VALUES ($1, $2, 1, 'success', now(), now(), $3::jsonb)`, deriveReceipt, deriveExecution, result)

	for index, chunk := range chunks {
		source := mustV7(t)
		exec(`INSERT INTO context.source (id, source_key, provenance_class) VALUES ($1, $2, 'unknown')`,
			source, livePrefix+"threads/8105550100.000"+string(rune('1'+index))+".ndjson")
		exec(`INSERT INTO context.source_version (id, source_id, version_ordinal, workflow_id, submission_idempotency_key, declared_format, acquired_at, status, matter_id, court_case_id, original_object_id)
		      VALUES ($1, $2, 1, $3, $3, 'ndjson', now(), 'retained', $4, $5, $6)`, chunk.sourceVersion, source, chunk.requestID, devMatterID, devCourtCaseID, mustV7(t))
		raw := mustV7(t)
		exec(`INSERT INTO context.raw_generation (id, source_version_id, generation_ordinal, format_id, parser_id, parser_version) VALUES ($1, $2, 1, 'ndjson', 'duckdb_structured_elt', '1.0.0')`, raw, chunk.sourceVersion)
		exec(`INSERT INTO context.normalized_generation (id, source_version_id, raw_generation_id, generation_ordinal, normalizer_id, normalizer_version) VALUES ($1, $2, $3, 1, 'generic_message', '1.0.0')`, chunk.generation, chunk.sourceVersion, raw)
		for ordinal, record := range chunk.records {
			participants := []map[string]string{{"role": "sender", "identifier": record.sender}}
			for _, recipient := range record.recipients {
				participants = append(participants, map[string]string{"role": "recipient", "identifier": recipient})
			}
			payload, _ := json.Marshal(map[string]any{
				"content": map[string]string{"body": record.body}, "occurred_at": record.at.Format(time.RFC3339),
				"record_type": "message", "participants": participants,
			})
			exec(`INSERT INTO context.normalized_record_identity (id, normalized_generation_id, source_version_id, record_ordinal, record_type, occurred_at, canonical_bytes, canonicalization, normalized_payload)
			      VALUES ($1, $2, $3, $4, 'message', $5, convert_to($6::jsonb::text, 'UTF8'), 'normalized-record-postgresql18-jsonb-text-utf8-sha256-v1', $6::jsonb)`,
				record.id, chunk.generation, chunk.sourceVersion, ordinal, record.at, string(payload))
		}
		verifyExecution, verifyReceipt := mustV7(t), mustV7(t)
		exec(`INSERT INTO context.activity_execution (id, source_version_id, workflow_id, activity_name, idempotency_key) VALUES ($1, $2, $3, 'verify_normalized_generation_activity', 'd04-probe')`, verifyExecution, chunk.sourceVersion, chunk.requestID)
		exec(`INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref) VALUES ($1, $2, 1, 'success', now(), now(), '{"ref_kind":"probe"}'::jsonb)`, verifyReceipt, verifyExecution)
		exec(`INSERT INTO context.reconciliation_receipt (id, activity_receipt_id, reconciliation_kind, status, verified_at, normalized_generation_id, expected, observed, discrepancies)
		      VALUES ($1, $2, 'normalized_generation_verification', 'success', now(), $3,
		              '{"normalized_generation_manifest_digest":"probe","verification_mode":"independent_recomputation"}',
		              '{"normalized_generation_manifest_digest":"probe","verification_mode":"independent_recomputation"}', '[]')`, chunk.verification, verifyReceipt, chunk.generation)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func runImport(t *testing.T, ctx context.Context, acts activities.FirstPartyContextActivities, chunk liveChunk, owner, perspective string) error {
	t.Helper()
	base := func(refs map[string]proffer.Ref) proffer.StageRequest {
		if owner != "" {
			refs["owner_person"] = proffer.Ref(owner)
		}
		if perspective != "" {
			refs["perspective_person"] = proffer.Ref(perspective)
		}
		return proffer.StageRequest{
			RequestID: chunk.requestID, MatterID: devMatterID, CourtCaseID: devCourtCaseID,
			SourceVersionRef: proffer.Ref(chunk.sourceVersion), DeclaredFormat: "ndjson", Refs: refs,
		}
	}
	resolution, err := acts.ResolveContextParticipants(ctx, base(map[string]proffer.Ref{
		"normalized_generation": proffer.Ref(chunk.generation), "normalized_verification": proffer.Ref(chunk.verification),
	}))
	if err != nil {
		return err
	}
	if resolution.Status != proffer.StatusSuccess {
		return errors.New("resolution not successful: " + resolution.Reason)
	}
	proposal, err := acts.ProposeFirstPartyContext(ctx, base(map[string]proffer.Ref{
		"normalized_generation": proffer.Ref(chunk.generation), "normalized_verification": proffer.Ref(chunk.verification),
		"participant_resolution": resolution.Ref,
	}))
	if err != nil {
		return err
	}
	if proposal.Status != proffer.StatusSuccess {
		return errors.New("proposal not successful: " + proposal.Reason)
	}
	confirmation, err := acts.ConfirmFirstPartyContext(ctx, base(map[string]proffer.Ref{
		"context_proposal": proposal.Ref, "normalized_verification": proffer.Ref(chunk.verification),
	}))
	if err != nil {
		return err
	}
	messages, err := acts.CommitFirstPartyMessages(ctx, base(map[string]proffer.Ref{
		"context_confirmation": confirmation.Ref, "normalized_verification": proffer.Ref(chunk.verification),
	}))
	if err != nil {
		return err
	}
	// A retried spine commit is a no-op returning the same receipt.
	again, err := acts.CommitFirstPartyMessages(ctx, base(map[string]proffer.Ref{
		"context_confirmation": confirmation.Ref, "normalized_verification": proffer.Ref(chunk.verification),
	}))
	if err != nil || again.ReceiptRef != messages.ReceiptRef {
		t.Fatalf("retried spine commit = %+v, %v; want the first receipt %s", again, err, messages.ReceiptRef)
	}
	_, err = acts.CommitFirstPartyContextThreads(ctx, base(map[string]proffer.Ref{
		"context_messages": messages.Ref, "context_confirmation": confirmation.Ref,
		"normalized_verification": proffer.Ref(chunk.verification),
	}))
	return err
}

func TestFirstPartyContextImportLive(t *testing.T) {
	dsn := os.Getenv("D04_LIVE_ADMIN_DSN")
	if dsn == "" {
		t.Skip("D04_LIVE_ADMIN_DSN is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var database string
	if err := admin.QueryRow(ctx, `SELECT current_database()`).Scan(&database); err != nil || database == "platform" {
		t.Fatalf("refusing to seed database %q (err %v): use a disposable database", database, err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET ROLE platform_runtime`)
		return err
	}
	engine, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	var role string
	if err := engine.QueryRow(ctx, `SELECT current_user`).Scan(&role); err != nil || role != "platform_runtime" {
		t.Fatalf("engine pool runs as %q (err %v), want platform_runtime", role, err)
	}

	day := time.Date(2024, 11, 20, 9, 0, 0, 0, time.UTC)
	first := liveChunk{requestID: "d04-live-" + mustV7(t), sourceVersion: mustV7(t), generation: mustV7(t), verification: mustV7(t),
		records: []liveRecord{
			{id: mustV7(t), sender: "self", recipients: []string{liveOwnerPhone}, at: day, body: "Can you take her Saturday?"},
			{id: mustV7(t), sender: liveOwnerPhone, recipients: []string{"self"}, at: day.Add(time.Hour), body: "Yes, 10am."},
			{id: mustV7(t), sender: "self", recipients: []string{liveOtherPhone}, at: day.Add(2 * time.Hour), body: "He said yes."},
		}}
	second := liveChunk{requestID: "d04-live-" + mustV7(t), sourceVersion: mustV7(t), generation: mustV7(t), verification: mustV7(t),
		records: []liveRecord{
			{id: mustV7(t), sender: "self", recipients: []string{liveOwnerPhone}, at: day.Add(48 * time.Hour), body: "Running late."},
			{id: mustV7(t), sender: liveOtherPhone, recipients: []string{"self"}, at: day.Add(49 * time.Hour), body: "ok"},
		}}
	seedLive(t, ctx, admin, []liveChunk{first, second})

	store, err := NewFirstPartyContextStore(engine)
	if err != nil {
		t.Fatal(err)
	}
	acts := activities.FirstPartyContextActivities{Store: store}

	// Missing identity fails loudly and permanently.
	err = runImport(t, ctx, acts, first, liveOwner, "")
	var application *temporal.ApplicationError
	if err == nil || !errors.As(err, &application) || !application.NonRetryable() {
		t.Fatalf("missing perspective person = %v, want a non-retryable failure", err)
	}
	// The owner must be the registry's one case owner.
	if err := runImport(t, ctx, acts, first, livePerspective, liveOwner); err == nil {
		t.Fatal("a perspective person was accepted as the case owner")
	}

	if err := runImport(t, ctx, acts, first, liveOwner, livePerspective); err != nil {
		t.Fatalf("first chunk import: %v", err)
	}
	// Owner 2026-10-02: nothing is immutable until promoted to evidence, so an
	// APPROVED version is extended in place by a later chunk too.
	if _, err := admin.Exec(ctx, `
		UPDATE working.first_party_context_thread_version
		SET review_state = 'approved', reviewed_by = 'd04-live-test', reviewed_at = now()
		WHERE context_thread_id IN (SELECT context_thread_id FROM working.first_party_context_thread WHERE owner_person_id = $1::uuid)`, liveOwner); err != nil {
		t.Fatalf("approve version 1: %v", err)
	}
	if err := runImport(t, ctx, acts, second, liveOwner, livePerspective); err != nil {
		t.Fatalf("second chunk import (extends the thread): %v", err)
	}

	ids := []string{first.records[0].id, first.records[1].id, second.records[0].id}
	thirdIDs := []string{first.records[2].id, second.records[1].id}
	var spine, sameIDs, participants, routes int
	if err := admin.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE message.id = record.id AND message.derived_from_record_id = record.id
		                        AND record.id = identity.id AND record.artifact_id IS NULL
		                        AND record.source_version_id = identity.source_version_id
		                        AND record.disclosure_tier = 'contemporaneous' AND record.message_corpus = 'first_party'),
		       (SELECT count(*) FROM working.message_participant WHERE message_id = ANY($1::uuid[]) AND entity_id IS NOT NULL),
		       (SELECT count(*) FROM working.message_projection_route WHERE normalized_record_id = ANY($1::uuid[]) AND decision_state = 'approved')
		FROM working.normalized_record record
		JOIN working.message message ON message.id = record.id
		JOIN context.normalized_record_identity identity ON identity.id = record.id
		WHERE record.id = ANY($1::uuid[])`, ids).Scan(&spine, &sameIDs, &participants, &routes); err != nil {
		t.Fatal(err)
	}
	if spine != 3 || sameIDs != 3 || participants != 6 || routes != 3 {
		t.Fatalf("spine rows %d, matching ids %d, participants %d, approved routes %d; want 3, 3, 6, 3", spine, sameIDs, participants, routes)
	}
	var threads, versions, members, sources int
	var firstAt, lastAt, horizon time.Time
	var state string
	if err := admin.QueryRow(ctx, `
		SELECT count(DISTINCT thread.context_thread_id), count(DISTINCT version.id),
		       (SELECT count(*) FROM working.first_party_context_thread_message m WHERE m.thread_version_id = min(version.id::text)::uuid),
		       (SELECT count(*) FROM working.first_party_context_thread_source s WHERE s.thread_version_id = min(version.id::text)::uuid),
		       min(version.first_occurred_at), max(version.last_occurred_at), max(version.knowledge_available_from), min(version.review_state)
		FROM working.first_party_context_thread thread
		JOIN working.first_party_context_thread_version version ON version.context_thread_id = thread.context_thread_id
		WHERE thread.owner_person_id = $1::uuid`, liveOwner).Scan(&threads, &versions, &members, &sources, &firstAt, &lastAt, &horizon, &state); err != nil {
		t.Fatal(err)
	}
	if threads != 1 || versions != 1 || members != 3 || sources != 2 || state != "approved" {
		t.Fatalf("threads %d versions %d members %d sources %d state %s; want one approved thread version extended in place to 3 members from 2 sources", threads, versions, members, sources, state)
	}
	if !firstAt.Equal(day) || !lastAt.Equal(day.Add(48*time.Hour)) || !horizon.Equal(day.Add(48*time.Hour)) {
		t.Fatalf("bounds %s..%s horizon %s", firstAt, lastAt, horizon)
	}
	var conversationMatches int
	if err := admin.QueryRow(ctx, `
		SELECT count(*) FROM working.message message
		JOIN working.first_party_context_thread_message member ON member.message_id = message.id
		WHERE message.conversation_id = member.context_thread_id AND message.id = ANY($1::uuid[])`, ids).Scan(&conversationMatches); err != nil {
		t.Fatal(err)
	}
	if conversationMatches != 3 {
		t.Fatalf("%d of 3 messages carry their thread as conversation_id", conversationMatches)
	}

	// The messages between the device owner and somebody else are acquired
	// third-party material: discovered, proposed routes, one conversation
	// filed under the acquired backup, no first-party rows.
	var thirdRecords, thirdMessages, proposedRoutes, conversations, counted, unresolved, firstPartyLeak int
	if err := admin.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE record.message_corpus = 'acquired_third_party' AND record.disclosure_tier = 'discovered'),
		       (SELECT count(*) FROM working.third_party_message WHERE id = ANY($1::uuid[]) AND normalized_record_id = id),
		       (SELECT count(*) FROM working.message_projection_route WHERE normalized_record_id = ANY($1::uuid[])
		          AND projection_kind = 'acquired_third_party' AND decision_state = 'proposed'),
		       (SELECT count(DISTINCT conversation_id) FROM working.third_party_message WHERE id = ANY($1::uuid[])),
		       (SELECT max(c.message_count) FROM working.third_party_conversation c
		          JOIN working.third_party_message m ON m.conversation_id = c.id WHERE m.id = ANY($1::uuid[])
		          AND c.source_artifact_id IS NULL AND c.source_version_id IS NOT NULL AND c.review_status = 'pending'),
		       (SELECT count(*) FROM working.third_party_message_participant WHERE message_id = ANY($1::uuid[])
		          AND participant_raw = $2 AND entity_id IS NULL),
		       (SELECT count(*) FROM working.message WHERE id = ANY($1::uuid[]))
		FROM working.normalized_record record WHERE record.id = ANY($1::uuid[])`, thirdIDs, liveOtherPhone).
		Scan(&thirdRecords, &thirdMessages, &proposedRoutes, &conversations, &counted, &unresolved, &firstPartyLeak); err != nil {
		t.Fatal(err)
	}
	if thirdRecords != 2 || thirdMessages != 2 || proposedRoutes != 2 || conversations != 1 || counted != 2 || unresolved != 2 || firstPartyLeak != 0 {
		t.Fatalf("third-party: records %d messages %d proposed routes %d conversations %d count %d unresolved %d first-party leak %d; want 2,2,2,1,2,2,0",
			thirdRecords, thirdMessages, proposedRoutes, conversations, counted, unresolved, firstPartyLeak)
	}
	t.Logf("live D04 proof: 3 first-party spine rows (contemporaneous, id = normalized record id, 6 resolved participants), 1 thread / 1 approved version extended in place / 3 members / 2 sources, horizon %s; 2 third-party messages (discovered, proposed routes) in 1 conversation", horizon)
}
