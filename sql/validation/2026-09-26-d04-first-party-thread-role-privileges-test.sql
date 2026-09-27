-- D04 live proof: what the ENGINE'S OWN ROLE may do to the first-party thread family.
--
-- Byline: Claude Code · Opus 5 · 2026-09-26
--
-- WHY THIS EXISTS, AND WHY IT IS SEPARATE. The companion file
-- 2026-09-26-d04-first-party-thread-projection-test.sql proves the write CONTRACT, but it
-- runs as the database owner. That proves the SQL is valid; it does NOT prove the engine is
-- allowed to execute it. Those are different claims, and conflating them is how a design
-- passes validation and then fails on first deploy with "permission denied".
--
-- deploy/docker/postgres/Dockerfile states it outright: "the engine connects as
-- platform_runtime". So this file runs the projection write AS platform_runtime and records
-- the privilege boundary that role actually has.
--
-- Proves:
--   1. platform_runtime CAN insert the whole four-table family, and the DEFERRED completeness
--      validator executes successfully under that role — so it holds EXECUTE on
--      working.validate_first_party_context_thread_version and SELECT on every relation that
--      function reads, including the context.*_relative_time_anchor tables;
--   2. platform_runtime CANNOT UPDATE working.first_party_context_thread_version. The four
--      tables are append-only BY PRIVILEGE, deliberately: contrast working.extraction_run and
--      working.content_chunk_generation, which carry explicit column-scoped UPDATE grants, so
--      the absence here is a decision and not an oversight;
--   3. therefore an approval is an INSERT of a NEW version row carrying review_state='approved'
--      with reviewed_by/reviewed_at and supersedes_id — never a mutation of the proposed row.
--      Currency is derived (the row nothing supersedes), never stamped onto an old row;
--   4. a superseding SOURCE assertion is likewise appended, and its predecessor is left as it
--      was rather than being marked superseded;
--   5. platform_runtime CANNOT write working.message or working.normalized_record at all — it
--      holds no privilege on either. The spine write therefore cannot live in the engine's
--      role, which is why it stays with the Python side that already owns those tables.
--
-- Run as a superuser on a THROWAWAY instance or disposable database: the fixtures need
-- session_replication_role, and SET ROLE needs membership. The closing ROLLBACK discards
-- everything. NEVER run this against the live platform database.

\set ON_ERROR_STOP on
BEGIN;

-- ---------------------------------------------------------------------------
-- Fixtures, as the owner. Not the subject of this proof.
-- ---------------------------------------------------------------------------
SET LOCAL session_replication_role = replica;

INSERT INTO registry.matter (id, title, created_by)
VALUES ('deadbeef-dead-beef-dead-beefdeadbeef', 'D04 role probe matter (DEV sentinel)', 'd04-probe');
INSERT INTO registry.court_case (id, matter_id, caption, created_by)
VALUES ('cafebabe-cafe-babe-cafe-babecafebabe', 'deadbeef-dead-beef-dead-beefdeadbeef',
        'D04 role probe case (DEV sentinel)', 'd04-probe');
INSERT INTO registry.entity (id, entity_type, display_name)
VALUES ('0d040000-0000-7000-8000-00000000e1e1', 'person', 'D04 probe owner');
INSERT INTO registry.person (id, role_in_case, connection_to)
VALUES ('0d040000-0000-7000-8000-00000000e1e1', 'user', 'petitioner');
INSERT INTO evidence.source (id, sha256, byte_size, source_type, acquisition_source, original_filename)
VALUES ('0d040000-0000-7000-8000-00000000ab01', decode(repeat('11', 32), 'hex'), 4096,
        'chat_export', 'probe', 'thread-export.xml');
INSERT INTO evidence.evidence_hash (id, source_ref, digest, source_id)
VALUES ('0d040000-0000-7000-8000-00000000aaa1', 'probe://d04/thread-export.xml',
        decode(repeat('11', 32), 'hex'), '0d040000-0000-7000-8000-00000000ab01');
INSERT INTO context.source (id, source_key, provenance_class)
VALUES ('0d040000-0000-7000-8000-000000000cd1', 'probe://d04/thread-export.xml', 'first_party_authored');
INSERT INTO context.source_version
    (id, source_id, version_ordinal, workflow_id, submission_idempotency_key, declared_format,
     acquired_at, status, matter_id, court_case_id)
VALUES ('0d040000-0000-7000-8000-000000000cd2', '0d040000-0000-7000-8000-000000000cd1', 1,
        'd04-probe-workflow', 'd04-probe-submission', 'smsbackuprestore_xml',
        '2026-03-02T00:00:00Z', 'registered',
        'deadbeef-dead-beef-dead-beefdeadbeef', 'cafebabe-cafe-babe-cafe-babecafebabe');
INSERT INTO working.normalized_record
    (id, artifact_id, record_type, source, content, occurred_at, knowledge_time,
     disclosure_tier, message_corpus, sender, recipients)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-00000000aaa1', 'message',
     'probe', 'first probe message', '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z',
     'contemporaneous', 'first_party', '+15550000001', '[{"identity": "+15550000002", "role": "to"}]'),
    ('0d040000-0000-7000-8000-000000000dd2', '0d040000-0000-7000-8000-00000000aaa1', 'message',
     'probe', 'second probe message', '2026-03-01T11:30:00Z', '2026-03-03T09:00:00Z',
     'discovered', 'first_party', '+15550000002', '[{"identity": "+15550000001", "role": "to"}]');
INSERT INTO working.message_projection_route
    (normalized_record_id, projection_kind, decision_state, basis, proposed_by,
     approved_by, approved_at, deriver_version)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', 'first_party', 'approved',
     '{"source_parties_present": true}', 'd04-probe', 'd04-probe', now(), 'd04-probe@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd2', 'first_party', 'approved',
     '{"source_parties_present": true}', 'd04-probe', 'd04-probe', now(), 'd04-probe@2026-09-26');
-- The spine messages are written here, by the owner, standing in for the Python-side
-- activity that owns them. Proof 5 below shows the engine's role could not have.
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
     'd04-probe@2026-09-26', now(), 'first_party');

SET LOCAL session_replication_role = origin;

-- ---------------------------------------------------------------------------
-- From here on we are the engine.
-- ---------------------------------------------------------------------------
SET LOCAL ROLE platform_runtime;

DO $$ BEGIN RAISE NOTICE 'acting as role: %', current_user; END $$;

-- Proof 1: the engine can write the whole family, and the deferred validator runs under it.
SET CONSTRAINTS ALL DEFERRED;

INSERT INTO working.first_party_context_thread
    (context_thread_id, owner_person_id, matter_id, court_case_id)
VALUES ('0d040000-0000-7000-8000-000000000ff1', '0d040000-0000-7000-8000-00000000e1e1',
        'deadbeef-dead-beef-dead-beefdeadbeef', 'cafebabe-cafe-babe-cafe-babecafebabe');

INSERT INTO working.first_party_context_thread_version
    (id, context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
     confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from, rationale)
VALUES ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1', 1,
        'd04-probe-classifier', '1.0.0', decode(repeat('bb', 32), 'hex'), 0.9, 'proposed',
        '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z',
        'proposed by the engine, awaiting the owner');

INSERT INTO working.first_party_context_thread_message
    (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
     source_available_from, required_for_horizon, membership_confidence)
VALUES
    ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1',
     '0d040000-0000-7000-8000-000000000dd1', 0, '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z', true, 1.0),
    ('0d040000-0000-7000-8000-000000000fe1', '0d040000-0000-7000-8000-000000000ff1',
     '0d040000-0000-7000-8000-000000000dd2', 1, '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z', true, 1.0);

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

SET CONSTRAINTS ALL IMMEDIATE;

DO $$ BEGIN RAISE NOTICE 'proof 1 OK: platform_runtime wrote the family and the deferred validator executed under it'; END $$;

-- Proof 2: the engine cannot mutate a version. Append-only by privilege.
SAVEPOINT neg_update_version;
DO $$
BEGIN
    UPDATE working.first_party_context_thread_version
       SET review_state = 'approved', reviewed_by = 'owner', reviewed_at = now()
     WHERE id = '0d040000-0000-7000-8000-000000000fe1';
    RAISE EXCEPTION 'PROBE FAILED: platform_runtime updated a thread version';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 2 OK: platform_runtime cannot UPDATE a version -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_update_version;

-- Proof 3: so an approval is an appended version. This is the confirm step's real write:
-- a new row, attributed, superseding the proposal. The proposal is left exactly as it was.
SAVEPOINT pos_approval_insert;
DO $$
DECLARE v_state TEXT; v_by TEXT; v_super UUID; v_old TEXT;
BEGIN
    EXECUTE 'SET CONSTRAINTS ALL DEFERRED';
    INSERT INTO working.first_party_context_thread_version
        (id, context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
         confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from,
         supersedes_id, reviewed_by, reviewed_at, rationale)
    VALUES ('0d040000-0000-7000-8000-000000000fe2', '0d040000-0000-7000-8000-000000000ff1', 2,
            'd04-probe-classifier', '1.0.0', decode(repeat('bb', 32), 'hex'), 0.9, 'approved',
            '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z',
            '0d040000-0000-7000-8000-000000000fe1', 'owner', now(),
            'owner validated the thread in the Workbench and approved the commit');
    INSERT INTO working.first_party_context_thread_message
        (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
         source_available_from, required_for_horizon, membership_confidence)
    VALUES
        ('0d040000-0000-7000-8000-000000000fe2', '0d040000-0000-7000-8000-000000000ff1',
         '0d040000-0000-7000-8000-000000000dd1', 0, '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z', true, 1.0),
        ('0d040000-0000-7000-8000-000000000fe2', '0d040000-0000-7000-8000-000000000ff1',
         '0d040000-0000-7000-8000-000000000dd2', 1, '2026-03-01T11:30:00Z', '2026-03-01T11:30:00Z', true, 1.0);
    INSERT INTO working.first_party_context_thread_source
        (id, thread_version_id, context_thread_id, source_version_id, source_anchor_ordinal, platform,
         platform_conversation_key, representation_kind, capture_kind, declared_format,
         perspective_person_id, coverage_first_occurred_at, coverage_last_occurred_at,
         coverage_message_count, source_available_from, required_for_horizon, metadata_clock_kind,
         metadata_timestamp, metadata_clock_basis, metadata_confidence, metadata_review_state,
         metadata_extractor_id, metadata_extractor_version, assertion_version, confidence,
         review_state, provenance_digest, asserted_by)
    VALUES ('0d040000-0000-7000-8000-000000000fd2', '0d040000-0000-7000-8000-000000000fe2',
            '0d040000-0000-7000-8000-000000000ff1', '0d040000-0000-7000-8000-000000000cd2', 0, 'sms',
            'probe-thread-key-1', 'native_export', 'device_export', 'smsbackuprestore_xml',
            '0d040000-0000-7000-8000-00000000e1e1', '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z',
            2, '2026-03-01T11:30:00Z', true, 'export_created', '2026-03-02T00:00:00Z',
            'export header declared creation time', 0.9, 'approved', 'd04-probe-extractor', '1.0.0',
            1, 0.9, 'approved', decode(repeat('cc', 32), 'hex'), 'owner');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    SELECT review_state, reviewed_by, supersedes_id INTO v_state, v_by, v_super
      FROM working.first_party_context_thread_version
     WHERE id = '0d040000-0000-7000-8000-000000000fe2';
    SELECT review_state INTO v_old FROM working.first_party_context_thread_version
     WHERE id = '0d040000-0000-7000-8000-000000000fe1';
    IF v_state <> 'approved' OR v_by IS NULL
       OR v_super <> '0d040000-0000-7000-8000-000000000fe1' OR v_old <> 'proposed' THEN
        RAISE EXCEPTION 'PROBE FAILED: the appended approval did not read back as expected (state=%, old=%)', v_state, v_old;
    END IF;
    RAISE NOTICE 'proof 3 OK: approval appended as version 2 (approved by %), proposal left untouched as %', v_by, v_old;
END $$;
ROLLBACK TO SAVEPOINT pos_approval_insert;

-- Proof 4: a superseding source assertion is appended too, predecessor untouched.
SAVEPOINT pos_source_append;
DO $$
DECLARE v_n INT; v_prev TEXT;
BEGIN
    INSERT INTO working.first_party_context_thread_source
        (id, thread_version_id, context_thread_id, source_version_id, source_anchor_ordinal, platform,
         platform_conversation_key, representation_kind, capture_kind, declared_format,
         perspective_person_id, coverage_first_occurred_at, coverage_last_occurred_at,
         coverage_message_count, source_available_from, required_for_horizon, metadata_clock_kind,
         metadata_timestamp, metadata_clock_basis, metadata_confidence, metadata_review_state,
         metadata_extractor_id, metadata_extractor_version, assertion_version, confidence,
         review_state, supersedes_id, provenance_digest, asserted_by)
    VALUES ('0d040000-0000-7000-8000-000000000fd3', '0d040000-0000-7000-8000-000000000fe1',
            '0d040000-0000-7000-8000-000000000ff1', '0d040000-0000-7000-8000-000000000cd2', 1, 'sms',
            'probe-thread-key-1', 'native_export', 'device_export', 'smsbackuprestore_xml',
            '0d040000-0000-7000-8000-00000000e1e1', '2026-03-01T10:00:00Z', '2026-03-01T11:30:00Z',
            2, '2026-03-01T11:30:00Z', true, 'export_created', '2026-03-02T00:00:00Z',
            'owner corrected the export clock basis', 0.99, 'approved', 'd04-probe-extractor', '1.1.0',
            2, 0.99, 'proposed', '0d040000-0000-7000-8000-000000000fd1',
            decode(repeat('ce', 32), 'hex'), 'owner');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    SELECT count(*) INTO v_n FROM working.first_party_context_thread_source
     WHERE thread_version_id = '0d040000-0000-7000-8000-000000000fe1';
    SELECT review_state INTO v_prev FROM working.first_party_context_thread_source
     WHERE id = '0d040000-0000-7000-8000-000000000fd1';
    IF v_n <> 2 OR v_prev <> 'proposed' THEN
        RAISE EXCEPTION 'PROBE FAILED: source append did not behave (n=%, prev=%)', v_n, v_prev;
    END IF;
    RAISE NOTICE 'proof 4 OK: assertion_version 2 appended with supersedes_id; predecessor left as %', v_prev;
END $$;
ROLLBACK TO SAVEPOINT pos_source_append;

-- Proof 5: the engine holds no privilege on the spine tables at all.
SAVEPOINT neg_spine;
DO $$
BEGIN
    INSERT INTO working.message
        (id, conversation_id, ts_utc, platform, external_id, direction, message_type,
         derived_from_record_id, deriver_version, derived_at, projection_kind)
    VALUES ('0d040000-0000-7000-8000-000000000dd2', '0d040000-0000-7000-8000-000000000cc1',
            '2026-03-01T11:30:00Z', 'sms', 'probe-ext-denied', 'inbound', 'text',
            '0d040000-0000-7000-8000-000000000dd2', 'd04-probe@2026-09-26', now(), 'first_party');
    RAISE EXCEPTION 'PROBE FAILED: platform_runtime wrote working.message';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 5a OK: platform_runtime cannot write working.message -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_spine;

SAVEPOINT neg_spine_record;
DO $$
BEGIN
    INSERT INTO working.normalized_record
        (id, artifact_id, record_type, source, content, occurred_at, knowledge_time,
         disclosure_tier, message_corpus)
    VALUES ('0d040000-0000-7000-8000-000000000dd9', '0d040000-0000-7000-8000-00000000aaa1', 'message',
            'probe', 'denied', '2026-03-01T14:00:00Z', '2026-03-01T14:00:00Z',
            'contemporaneous', 'first_party');
    RAISE EXCEPTION 'PROBE FAILED: platform_runtime wrote working.normalized_record';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 5b OK: platform_runtime cannot write working.normalized_record -- %', SQLERRM;
END $$;
ROLLBACK TO SAVEPOINT neg_spine_record;

RESET ROLE;
DO $$ BEGIN RAISE NOTICE 'ALL D04 ROLE-PRIVILEGE PROOFS PASSED (back to %)', current_user; END $$;

-- Everything the probe created is discarded here.
ROLLBACK;
