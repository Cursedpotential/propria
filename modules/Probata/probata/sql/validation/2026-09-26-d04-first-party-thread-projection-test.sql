-- D04 live proof: the minimum satisfying write for the first-party context-thread family.
--
-- Byline: Claude Code · Opus 5 · 2026-09-26
--
-- WHY THIS EXISTS. server/evidence/message_projection.py::_write_first_party inserts into
-- working.conversation, which schema_snapshot_20260907.sql deleted (its replacement's table
-- comment says so outright: "working.conversation and conversation_group are deleted; do not
-- rebuild parallel models"). The 2026-09-24 review recorded the mismatch but could not validate
-- anything, because it found no PostgreSQL to run against. This file is that missing proof: it
-- establishes empirically what the current schema demands of a first-party projection write,
-- so the repair is built against measured constraints rather than a reading of the DDL.
--
-- Proves, against a real PostgreSQL holding the platform schema:
--   1. the old statement is dead — working.conversation does not exist, so no name substitution
--      can repair the writer;
--   2. the four-table family (first_party_context_thread, _version, _message, _source) plus the
--      spine working.message row is satisfiable in ONE transaction, and the rows read back joined;
--   3. working.message.id must EQUAL the corresponding working.normalized_record.id — the
--      uuid.uuid4() in the current writer is rejected by message_id_fkey;
--   4. the version's first/last_occurred_at must equal its membership's min/max(occurred_at);
--   5. a version with no membership is rejected (so _version/_message/_source CANNOT be split
--      across transactions — the completeness triggers are DEFERRABLE INITIALLY DEFERRED and
--      fire at COMMIT, which is why one activity must own the whole thread-version transaction);
--   6. knowledge_available_from must equal the greatest required member availability;
--   7. _message.source_available_from must equal its occurred_at;
--   8. _source requires an existing context.source_version — the evidence-side artifact_id the
--      Python ingest path carries cannot satisfy it;
--   9. the thread's (court_case_id, matter_id) pair must resolve in registry.court_case, so a
--      non-UUID matter literal such as 'primary' can never be stored;
--  10. thread_ordinal is unique per version.
--
-- Identity uses the D-126 pre-launch DEV sentinels (matter deadbeef-…, court case cafebabe-…)
-- seeded by sql/0069_dev_case_registry_identity.sql and documented in
-- modules/engine/postgres/proffer_schema_probe.go. Nothing here is derived or invented: the
-- probe supplies identity explicitly, which is the point — a projection that cannot be handed
-- an owner, matter, court case and perspective person must fail, not guess.
--
-- Run as a superuser on a THROWAWAY instance or disposable database. Upstream fixtures are
-- inserted with session_replication_role = replica so no ingest chain is needed; the projection
-- write itself runs with replication role 'origin' and the derived-write guard ARMED, so every
-- foreign key, CHECK and constraint trigger really fires. SET CONSTRAINTS ALL IMMEDIATE forces
-- the deferred validators to run inside the transaction, because the closing ROLLBACK means
-- COMMIT never happens. Every row is discarded by that ROLLBACK.
-- NEVER run this against the live platform database.

\set ON_ERROR_STOP on
BEGIN;

-- ---------------------------------------------------------------------------
-- Proof 1: the table the current writer targets is gone.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'working' AND table_name = 'conversation') THEN
        RAISE EXCEPTION 'PROBE FAILED: working.conversation still exists; this schema is not the 2026-09-07 snapshot';
    END IF;
    RAISE NOTICE 'proof 1 OK: working.conversation does not exist';
END $$;

-- ---------------------------------------------------------------------------
-- Upstream fixtures. Not the subject of the proof: inserted with triggers and
-- FK checks off so the probe does not have to replay custody, parse and
-- normalize just to reach the projection.
-- ---------------------------------------------------------------------------
SET LOCAL session_replication_role = replica;

INSERT INTO registry.matter (id, title, created_by)
VALUES ('deadbeef-dead-beef-dead-beefdeadbeef', 'D04 probe matter (DEV sentinel)', 'd04-probe');

INSERT INTO registry.court_case (id, matter_id, caption, created_by)
VALUES ('cafebabe-cafe-babe-cafe-babecafebabe', 'deadbeef-dead-beef-dead-beefdeadbeef',
        'D04 probe case (DEV sentinel)', 'd04-probe');

-- The owner person. registry.person.id is an FK to registry.entity.id.
INSERT INTO registry.entity (id, entity_type, display_name)
VALUES ('0d040000-0000-7000-8000-00000000e1e1', 'person', 'D04 probe owner');
INSERT INTO registry.person (id, role_in_case, connection_to)
VALUES ('0d040000-0000-7000-8000-00000000e1e1', 'user', 'petitioner');

-- The artifact the normalized records hang off.
INSERT INTO evidence.evidence_hash (id, source_ref, digest)
VALUES ('0d040000-0000-7000-8000-00000000aaa1', 'probe://d04/thread-export.xml',
        decode(repeat('11', 32), 'hex'));

-- The selected source and its version: the FK _source actually requires.
INSERT INTO context.source (id, source_key, provenance_class)
VALUES ('0d040000-0000-7000-8000-000000000cd1', 'probe://d04/thread-export.xml', 'first_party_authored');
INSERT INTO context.source_version
    (id, source_id, version_ordinal, workflow_id, submission_idempotency_key, declared_format,
     acquired_at, status, matter_id, court_case_id)
VALUES ('0d040000-0000-7000-8000-000000000cd2', '0d040000-0000-7000-8000-000000000cd1', 1,
        'd04-probe-workflow', 'd04-probe-submission', 'smsbackuprestore_xml',
        '2026-03-02T00:00:00Z', 'registered',
        'deadbeef-dead-beef-dead-beefdeadbeef', 'cafebabe-cafe-babe-cafe-babecafebabe');

-- Two normalized records: the authored spine the projection is rebuilt from.
INSERT INTO working.normalized_record
    (id, artifact_id, record_type, source, content, occurred_at, message_corpus, sender, recipients)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-00000000aaa1', 'message',
     'probe', 'first probe message', '2026-03-01T10:00:00Z', 'first_party',
     '+15550000001', '[{"identity": "+15550000002", "role": "to"}]'),
    ('0d040000-0000-7000-8000-000000000dd2', '0d040000-0000-7000-8000-00000000aaa1', 'message',
     'probe', 'second probe message', '2026-03-01T11:30:00Z', 'first_party',
     '+15550000002', '[{"identity": "+15550000001", "role": "to"}]'),
    -- dd3 is deliberately NOT a member of the version below; proof 10 uses it to
    -- isolate the thread_ordinal unique key from the (version, message) primary key.
    ('0d040000-0000-7000-8000-000000000dd3', '0d040000-0000-7000-8000-00000000aaa1', 'message',
     'probe', 'third probe message', '2026-03-01T12:45:00Z', 'first_party',
     '+15550000001', '[{"identity": "+15550000002", "role": "to"}]');

SET LOCAL session_replication_role = origin;

-- Arm the derived-write guard and declare this session the deriver, so the
-- projection write is proven to pass the guard rather than to dodge it.
SET LOCAL app.enforce_derived_guard = 'on';
SET LOCAL app.deriving = 'on';

-- ---------------------------------------------------------------------------
-- Proof 2: the whole write, in one transaction, with everything armed.
-- ---------------------------------------------------------------------------
SET CONSTRAINTS ALL DEFERRED;

-- The route comes first: working.message.message_route_fk points at it.
INSERT INTO working.message_projection_route
    (normalized_record_id, projection_kind, decision_state, basis, proposed_by,
     approved_by, approved_at, deriver_version)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', 'first_party', 'approved',
     '{"source_parties_present": true}', 'd04-probe', 'd04-probe', now(), 'd04-probe@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd2', 'first_party', 'approved',
     '{"source_parties_present": true}', 'd04-probe', 'd04-probe', now(), 'd04-probe@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd3', 'first_party', 'approved',
     '{"source_parties_present": true}', 'd04-probe', 'd04-probe', now(), 'd04-probe@2026-09-26');

-- The spine message rows. PROOF 3 IS ENCODED HERE: id is the normalized_record
-- id, not a fresh uuid4. conversation_id is the per-source conversation grain;
-- it is NOT NULL and, since working.conversation was deleted, carries no FK.
INSERT INTO working.message
    (id, conversation_id, ts_utc, platform, external_id, sender_raw, recipient_raw, direction,
     message_type, content_sha256, derived_from_record_id, deriver_version, derived_at, projection_kind)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-000000000cc1',
     '2026-03-01T10:00:00Z', 'sms', 'probe-ext-1', '+15550000001', '+15550000002', 'outbound',
     'text', decode(repeat('a1', 32), 'hex'), '0d040000-0000-7000-8000-000000000dd1',
     'd04-probe@2026-09-26', now(), 'first_party'),
    ('0d040000-0000-7000-8000-000000000dd2', '0d040000-0000-7000-8000-000000000cc1',
     '2026-03-01T11:30:00Z', 'sms', 'probe-ext-2', '+15550000002', '+15550000001', 'inbound',
     'text', decode(repeat('a2', 32), 'hex'), '0d040000-0000-7000-8000-000000000dd2',
     'd04-probe@2026-09-26', now(), 'first_party'),
    ('0d040000-0000-7000-8000-000000000dd3', '0d040000-0000-7000-8000-000000000cc1',
     '2026-03-01T12:45:00Z', 'sms', 'probe-ext-3', '+15550000001', '+15550000002', 'outbound',
     'text', decode(repeat('a3', 32), 'hex'), '0d040000-0000-7000-8000-000000000dd3',
     'd04-probe@2026-09-26', now(), 'first_party');

-- The thread: identity-scoped, matter and court case paired.
INSERT INTO working.first_party_context_thread
    (context_thread_id, owner_person_id, matter_id, court_case_id)
VALUES ('0d040000-0000-7000-8000-000000000ff1', '0d040000-0000-7000-8000-00000000e1e1',
        'deadbeef-dead-beef-dead-beefdeadbeef', 'cafebabe-cafe-babe-cafe-babecafebabe');

-- The version snapshot. Bounds and horizon must match the membership and the
-- source exactly; the validator recomputes both.
INSERT INTO working.first_party_context_thread_version
    (id, context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
     confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from, rationale)
VALUES ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1', 1,
        'd04-probe-classifier', '1.0.0', decode(repeat('bb', 32), 'hex'), 0.9, 'proposed',
        '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z',
        'synthetic probe version');

-- Ordered membership. source_available_from must equal occurred_at (CHECK).
INSERT INTO working.first_party_context_thread_message
    (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
     source_available_from, required_for_horizon, membership_confidence)
VALUES
    ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1',
     '0d040000-0000-7000-8000-000000000dd1', 0, '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z', true, 1.0),
    ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1',
     '0d040000-0000-7000-8000-000000000dd2', 1, '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z', true, 1.0);

-- The source assertion. source_available_from must equal coverage_last_occurred_at (CHECK).
INSERT INTO working.first_party_context_thread_source
    (id, thread_version_id, context_thread_id, source_version_id, source_anchor_ordinal, platform,
     platform_conversation_key, representation_kind, capture_kind, declared_format,
     perspective_person_id, coverage_first_occurred_at, coverage_last_occurred_at,
     coverage_message_count, source_available_from, required_for_horizon, metadata_clock_kind,
     metadata_timestamp, metadata_clock_basis, metadata_confidence, metadata_review_state,
     metadata_extractor_id, metadata_extractor_version, assertion_version, confidence,
     review_state, provenance_digest, asserted_by)
VALUES ('0d040000-0000-7000-8000-000000000fd1', '0d040000-0000-7000-8000-000000000fe1',
        '0d040000-0000-7000-8000-000000000ff1', '0d040000-0000-7000-8000-000000000cd2', 0, 'sms',
        'probe-thread-key-1', 'native_export', 'device_export', 'smsbackuprestore_xml',
        '0d040000-0000-7000-8000-00000000e1e1', '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z',
        2, '2026-03-01T11:30:00Z', true, 'export_created', '2026-03-02T00:00:00Z',
        'export header declared creation time', 0.9, 'unreviewed', 'd04-probe-extractor', '1.0.0',
        1, 0.9, 'proposed', decode(repeat('cc', 32), 'hex'), 'd04-probe');

-- Fire every deferred validator now; ROLLBACK means COMMIT never will.
SET CONSTRAINTS ALL IMMEDIATE;

DO $$ BEGIN RAISE NOTICE 'proof 2 OK: the four-table family plus spine message committed all deferred checks'; END $$;

-- ---------------------------------------------------------------------------
-- Read the rows back, joined, and assert the shape.
-- ---------------------------------------------------------------------------
SELECT 'READBACK thread' AS what, t.context_thread_id, t.owner_person_id, t.matter_id, t.court_case_id, t.case_key
FROM working.first_party_context_thread t
WHERE t.context_thread_id = '0d040000-0000-7000-8000-000000000ff1';

SELECT 'READBACK version' AS what, v.id, v.version_ordinal, v.review_state,
       v.first_occurred_at, v.last_occurred_at, v.knowledge_available_from,
       octet_length(v.assertion_digest) AS digest_bytes
FROM working.first_party_context_thread_version v
WHERE v.context_thread_id = '0d040000-0000-7000-8000-000000000ff1';

-- The join that matters: membership -> spine message -> normalized record.
-- id_matches_normalized_record proves the fix for the uuid4 bug.
SELECT 'READBACK membership' AS what, m.thread_ordinal, m.message_id, m.occurred_at,
       m.source_available_from, msg.platform, msg.direction, msg.external_id,
       (msg.id = nr.id) AS id_matches_normalized_record, nr.message_corpus
FROM working.first_party_context_thread_message m
JOIN working.message msg ON msg.id = m.message_id
JOIN working.normalized_record nr ON nr.id = msg.id
WHERE m.thread_version_id = '0d040000-0000-7000-8000-000000000fe1'
ORDER BY m.thread_ordinal;

SELECT 'READBACK source' AS what, s.source_anchor_ordinal, s.platform, s.platform_conversation_key,
       s.representation_kind, s.review_state, s.assertion_version, s.perspective_person_id,
       s.coverage_first_occurred_at, s.coverage_last_occurred_at, s.source_available_from,
       sv.declared_format AS source_version_format, octet_length(s.provenance_digest) AS digest_bytes
FROM working.first_party_context_thread_source s
JOIN context.source_version sv ON sv.id = s.source_version_id
WHERE s.thread_version_id = '0d040000-0000-7000-8000-000000000fe1';

DO $$
DECLARE
    v_members INTEGER;
    v_sources INTEGER;
    v_mismatched INTEGER;
BEGIN
    SELECT count(*) INTO v_members FROM working.first_party_context_thread_message
     WHERE thread_version_id = '0d040000-0000-7000-8000-000000000fe1';
    SELECT count(*) INTO v_sources FROM working.first_party_context_thread_source
     WHERE thread_version_id = '0d040000-0000-7000-8000-000000000fe1';
    SELECT count(*) INTO v_mismatched
      FROM working.first_party_context_thread_message m
      JOIN working.message msg ON msg.id = m.message_id
      LEFT JOIN working.normalized_record nr ON nr.id = msg.id
     WHERE m.thread_version_id = '0d040000-0000-7000-8000-000000000fe1'
       AND (nr.id IS NULL OR nr.id <> msg.id);
    IF v_members <> 2 THEN RAISE EXCEPTION 'PROBE FAILED: expected 2 membership rows, found %', v_members; END IF;
    IF v_sources <> 1 THEN RAISE EXCEPTION 'PROBE FAILED: expected 1 source assertion, found %', v_sources; END IF;
    IF v_mismatched <> 0 THEN RAISE EXCEPTION 'PROBE FAILED: % spine messages do not carry the normalized-record id', v_mismatched; END IF;
    RAISE NOTICE 'readback OK: 2 members, 1 source assertion, every spine message id = its normalized_record id';
END $$;

-- ---------------------------------------------------------------------------
-- Negative proofs. Each offending write must be REJECTED. A savepoint isolates
-- each one so the positive rows above survive for comparison.
-- ---------------------------------------------------------------------------

-- Proof 3: a fresh uuid4 spine message id (what the current writer does).
SAVEPOINT neg_uuid4;
DO $$
BEGIN
    INSERT INTO working.message
        (id, conversation_id, ts_utc, platform, external_id, direction, message_type,
         derived_from_record_id, deriver_version, derived_at, projection_kind)
    VALUES ('0d040000-0000-7000-8000-00000000dead', '0d040000-0000-7000-8000-000000000cc1',
            '2026-03-01T12:00:00Z', 'sms', 'probe-ext-uuid4', 'outbound', 'text',
            '0d040000-0000-7000-8000-000000000dd1', 'd04-probe@2026-09-26', now(), 'first_party');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: a random working.message.id was accepted; message_id_fkey did not fire';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3 OK: random working.message.id rejected -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_uuid4;

-- Proof 4: version bounds that disagree with the membership.
SAVEPOINT neg_bounds;
DO $$
BEGIN
    EXECUTE 'SET CONSTRAINTS ALL DEFERRED';
    INSERT INTO working.first_party_context_thread_version
        (id, context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
         confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from)
    VALUES ('0d040000-0000-7000-8000-000000000fe2', '0d040000-0000-7000-8000-000000000ff1', 2,
            'd04-probe-classifier', '1.0.0', decode(repeat('bd', 32), 'hex'), 0.9, 'proposed',
            '2026-03-01T09:00:00Z', '2026-03-01T23:00:00Z', '2026-03-01T23:00:00Z');
    INSERT INTO working.first_party_context_thread_message
        (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
         source_available_from, required_for_horizon, membership_confidence)
    VALUES ('0d040000-0000-7000-8000-000000000fe2', '0d040000-0000-7000-8000-000000000ff1',
            '0d040000-0000-7000-8000-000000000dd1', 0, '2026-03-01T10:00:00Z',
            '2026-03-01T10:00:00Z', true, 1.0);
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: wrong version bounds were accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 4 OK: version bounds must equal membership bounds -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_bounds;

-- Proof 5: a version with no membership. THIS is why _version/_message/_source
-- cannot be split across activities: each would commit on its own and fail here.
SAVEPOINT neg_no_members;
DO $$
BEGIN
    EXECUTE 'SET CONSTRAINTS ALL DEFERRED';
    INSERT INTO working.first_party_context_thread_version
        (id, context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
         confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from)
    VALUES ('0d040000-0000-7000-8000-000000000fe3', '0d040000-0000-7000-8000-000000000ff1', 3,
            'd04-probe-classifier', '1.0.0', decode(repeat('be', 32), 'hex'), 0.9, 'proposed',
            '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: a membership-free version was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 5 OK: a version with no membership is rejected -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_no_members;

-- Proof 6: knowledge_available_from that is not the greatest required availability.
SAVEPOINT neg_horizon;
DO $$
BEGIN
    EXECUTE 'SET CONSTRAINTS ALL DEFERRED';
    INSERT INTO working.first_party_context_thread_version
        (id, context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
         confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from)
    VALUES ('0d040000-0000-7000-8000-000000000fe4', '0d040000-0000-7000-8000-000000000ff1', 4,
            'd04-probe-classifier', '1.0.0', decode(repeat('bf', 32), 'hex'), 0.9, 'proposed',
            '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z', '2026-03-01T10:00:00Z');
    INSERT INTO working.first_party_context_thread_message
        (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
         source_available_from, required_for_horizon, membership_confidence)
    VALUES
        ('0d040000-0000-7000-8000-000000000fe4', '0d040000-0000-7000-8000-000000000ff1',
         '0d040000-0000-7000-8000-000000000dd1', 0, '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z', true, 1.0),
        ('0d040000-0000-7000-8000-000000000fe4', '0d040000-0000-7000-8000-000000000ff1',
         '0d040000-0000-7000-8000-000000000dd2', 1, '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z', true, 1.0);
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: a wrong knowledge horizon was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 6 OK: knowledge_available_from must be the greatest required availability -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_horizon;

-- Proof 7: membership whose source_available_from disagrees with occurred_at.
SAVEPOINT neg_clock;
DO $$
BEGIN
    INSERT INTO working.first_party_context_thread_message
        (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
         source_available_from, required_for_horizon, membership_confidence)
    VALUES ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1',
            '0d040000-0000-7000-8000-000000000dd1', 9, '2026-03-01T10:00:00Z',
            '2026-03-05T10:00:00Z', true, 1.0);
    RAISE EXCEPTION 'PROBE FAILED: a membership clock mismatch was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 7 OK: membership source_available_from must equal occurred_at -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_clock;

-- Proof 8: a source assertion naming a source_version that does not exist.
-- This is the gap the Python ingest path cannot close: it holds an
-- evidence-side artifact_id, not a context.source_version id.
SAVEPOINT neg_source;
DO $$
BEGIN
    INSERT INTO working.first_party_context_thread_source
        (id, thread_version_id, context_thread_id, source_version_id, source_anchor_ordinal, platform,
         platform_conversation_key, representation_kind, capture_kind, declared_format,
         perspective_person_id, coverage_first_occurred_at, coverage_last_occurred_at,
         source_available_from, required_for_horizon, metadata_clock_kind, metadata_clock_basis,
         metadata_review_state, metadata_extractor_id, metadata_extractor_version,
         assertion_version, confidence, review_state, provenance_digest, asserted_by)
    VALUES ('0d040000-0000-7000-8000-000000000fd9', '0d040000-0000-7000-8000-000000000fe1',
            '0d040000-0000-7000-8000-000000000ff1', '0d040000-0000-7000-8000-00000000aaa1', 9, 'sms',
            'probe-thread-key-9', 'native_export', 'device_export', 'smsbackuprestore_xml',
            '0d040000-0000-7000-8000-00000000e1e1', '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z',
            '2026-03-01T11:30:00Z', true, 'export_created', 'probe', 'unreviewed',
            'd04-probe-extractor', '1.0.0', 1, 0.9, 'proposed', decode(repeat('ce', 32), 'hex'), 'd04-probe');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: a source assertion with no context.source_version was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 8 OK: _source requires an existing context.source_version -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_source;

-- Proof 9: a thread whose matter/court-case pair does not resolve. The literal
-- 'primary' that IngestRequest.matter_id defaults to cannot even be cast.
SAVEPOINT neg_identity;
DO $$
BEGIN
    INSERT INTO working.first_party_context_thread
        (context_thread_id, owner_person_id, matter_id, court_case_id)
    VALUES ('0d040000-0000-7000-8000-000000000ff9', '0d040000-0000-7000-8000-00000000e1e1',
            'deadbeef-dead-beef-dead-beefdeadbeef', '0d040000-0000-7000-8000-00000000cc99');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: an unresolvable court case was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 9a OK: (court_case_id, matter_id) must resolve in registry.court_case -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_identity;

SAVEPOINT neg_matter_literal;
DO $$
BEGIN
    EXECUTE $q$INSERT INTO working.first_party_context_thread
                   (context_thread_id, owner_person_id, matter_id, court_case_id)
               VALUES ('0d040000-0000-7000-8000-000000000ff8',
                       '0d040000-0000-7000-8000-00000000e1e1', 'primary',
                       'cafebabe-cafe-babe-cafe-babecafebabe')$q$;
    RAISE EXCEPTION 'PROBE FAILED: the literal matter id ''primary'' was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 9b OK: matter_id ''primary'' is not a uuid and cannot be stored -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_matter_literal;

-- Proof 10: duplicate thread_ordinal inside one version. dd3 is a message that is
-- NOT yet a member, so the (thread_version_id, message_id) primary key cannot be
-- what rejects this -- only the thread_ordinal unique key can. That key is a plain
-- unique index, so it fires at the INSERT, ahead of any deferred check.
SAVEPOINT neg_ordinal;
DO $$
BEGIN
    INSERT INTO working.first_party_context_thread_message
        (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
         source_available_from, required_for_horizon, membership_confidence)
    VALUES ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1',
            '0d040000-0000-7000-8000-000000000dd3', 0, '2026-03-01T12:45:00Z',
            '2026-03-01T12:45:00Z', true, 1.0);
    RAISE EXCEPTION 'PROBE FAILED: a duplicate thread_ordinal was accepted';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 10 OK: thread_ordinal is unique per version -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_ordinal;

DO $$ BEGIN RAISE NOTICE 'ALL D04 PROBE PROOFS PASSED'; END $$;

-- Everything the probe created is discarded here. No synthetic row survives.
ROLLBACK;
