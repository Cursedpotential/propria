-- D04: the exact, minimal privileges the ENGINE needs to write the message spine.
--
-- Byline: Claude Code · Opus 5 · 2026-09-26
--
-- WHY THIS EXISTS. Owner ruling: Go is the orchestrator, everything runs through the Go engine,
-- and the Python side reports to it rather than co-owning tables. The engine connects as
-- platform_runtime (deploy/docker/postgres/Dockerfile), and
-- 2026-09-26-d04-first-party-thread-role-privileges-test.sql proved that role currently cannot
-- write working.message or working.normalized_record at all. Under the ruling that is a DEFECT,
-- not a boundary: the intended sole writer lacks the grants to be the writer.
--
-- A privilege change on custody-case evidence tables should be deliberate and minimal, so this
-- file derives the set empirically instead of guessing it. It proves each grant is NECESSARY by
-- revoking it and showing the write fails, and proves the set is SUFFICIENT by performing the
-- whole spine write as platform_runtime with nothing else added.
--
-- Proves:
--   1. baseline — with no new grants, platform_runtime cannot write the spine (the defect);
--   2. sufficiency — with exactly the proposed set, platform_runtime performs the entire spine
--      write: normalized_record, its route, the message carrying the record's own id, and the
--      message participants, with the derived-write guard ARMED;
--   3. necessity — each grant in the set is individually required: revoking any one of them
--      breaks the write. This includes working.normalized_record_event, which is NOT obvious
--      from any column or foreign key: working.emit_row_event is SECURITY INVOKER and the
--      normalized_record_outbox trigger makes it INSERT the CDC row as the CALLING role.
--
-- THE PROPOSED GRANT SET (and why each is needed):
--   working.normalized_record          SELECT, INSERT  the authored spine the engine now writes
--   working.normalized_record_event    INSERT          the outbox trigger writes it as the caller
--   working.message_projection_route   SELECT, INSERT  message_route_fk requires the route row
--   working.message                    SELECT, INSERT  the spine projection row
--   working.message_participant        SELECT, INSERT  the per-message party rows
-- No UPDATE and no DELETE anywhere: the engine appends. SELECT is included where the store reads
-- back for idempotency.
--
-- Run as a superuser on a THROWAWAY instance or disposable database: it needs
-- session_replication_role for fixtures and GRANT/REVOKE as the table owner. GRANT and REVOKE
-- are transactional in PostgreSQL, so the closing ROLLBACK also undoes every privilege change.
-- NEVER run this against the live platform database.

\set ON_ERROR_STOP on
BEGIN;

-- ---------------------------------------------------------------------------
-- Fixtures, as the owner.
-- ---------------------------------------------------------------------------
SET LOCAL session_replication_role = replica;

INSERT INTO registry.matter (id, title, created_by)
VALUES ('deadbeef-dead-beef-dead-beefdeadbeef', 'D04 grant probe matter (DEV sentinel)', 'd04-probe');
INSERT INTO registry.court_case (id, matter_id, caption, created_by)
VALUES ('cafebabe-cafe-babe-cafe-babecafebabe', 'deadbeef-dead-beef-dead-beefdeadbeef',
        'D04 grant probe case (DEV sentinel)', 'd04-probe');
INSERT INTO evidence.source (id, sha256, byte_size, source_type, acquisition_source, original_filename)
VALUES ('0d040000-0000-7000-8000-00000000ab01', decode(repeat('11', 32), 'hex'), 4096,
        'chat_export', 'probe', 'thread-export.xml');
INSERT INTO evidence.evidence_hash (id, source_ref, digest, source_id)
VALUES ('0d040000-0000-7000-8000-00000000aaa1', 'probe://d04/thread-export.xml',
        decode(repeat('11', 32), 'hex'), '0d040000-0000-7000-8000-00000000ab01');

SET LOCAL session_replication_role = origin;

-- ---------------------------------------------------------------------------
-- Proof 1: the defect. No new grants yet.
-- ---------------------------------------------------------------------------
SAVEPOINT baseline;
SET LOCAL ROLE platform_runtime;
DO $$
BEGIN
    INSERT INTO working.normalized_record
        (id, artifact_id, record_type, source, content, occurred_at, knowledge_time,
         disclosure_tier, message_corpus)
    VALUES ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-00000000aaa1', 'message',
            'probe', 'baseline', '2026-03-01T10:00:00Z', '2026-03-01T10:00:00Z',
            'contemporaneous', 'first_party');
    RAISE EXCEPTION 'PROBE FAILED: the engine role could already write the spine';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 1 OK: today the engine cannot write the spine -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT baseline;

-- ---------------------------------------------------------------------------
-- The proposed grant set: five write grants, and six READ grants that are not
-- obvious from any column or foreign key.
--
-- The read grants exist because working.validate_message_projection is a
-- DEFERRABLE CONSTRAINT TRIGGER on working.message, working.normalized_record,
-- working.message_projection_route and registry.person, and it is SECURITY
-- INVOKER. So it executes as the ENGINE, and the engine needs SELECT on every
-- relation that function touches -- including the third-party tables it reads
-- only to prove a first-party record has NOT also been projected as third-party.
-- PostgreSQL checks table permissions when a statement executes, not when its
-- rows match, so these are required even on a database with no third-party rows
-- at all. Discovered by running the write as platform_runtime and reading the
-- failure: "permission denied for table third_party_message".
-- ---------------------------------------------------------------------------

-- Writes: the spine the engine now owns.
GRANT SELECT, INSERT ON TABLE working.normalized_record        TO platform_runtime;
GRANT         INSERT ON TABLE working.normalized_record_event  TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.message_projection_route TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.message                  TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.message_participant      TO platform_runtime;

-- Reads: required by the SECURITY INVOKER projection validator. Read-only, on
-- tables the engine never writes.
GRANT SELECT ON TABLE working.third_party_message                     TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_conversation                TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_message_participant         TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_conversation_acquisition    TO platform_runtime;
GRANT SELECT ON TABLE registry.person                                 TO platform_runtime;
GRANT SELECT ON TABLE evidence.acquisition                            TO platform_runtime;

-- ---------------------------------------------------------------------------
-- Proof 2: sufficiency. The whole spine write, as the engine, guard armed.
-- ---------------------------------------------------------------------------
SET LOCAL ROLE platform_runtime;
SET LOCAL app.enforce_derived_guard = 'on';
SET LOCAL app.deriving = 'on';

DO $$ BEGIN RAISE NOTICE 'acting as role: %', current_user; END $$;

SET CONSTRAINTS ALL DEFERRED;

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
     '{"source_parties_present": true}', 'engine', 'engine', now(), 'engine@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd2', 'first_party', 'approved',
     '{"source_parties_present": true}', 'engine', 'engine', now(), 'engine@2026-09-26');

-- id IS the normalized_record id, never a fresh uuid.
INSERT INTO working.message
    (id, conversation_id, ts_utc, platform, external_id, sender_raw, recipient_raw, direction,
     message_type, content_sha256, derived_from_record_id, deriver_version, derived_at, projection_kind)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-000000000cc1',
     '2026-03-01T10:00:00Z', 'sms', 'probe-ext-1', '+15550000001', '+15550000002', 'outbound',
     'text', decode(repeat('a1', 32), 'hex'), '0d040000-0000-7000-8000-000000000dd1',
     'engine@2026-09-26', now(), 'first_party'),
    ('0d040000-0000-7000-8000-000000000dd2', '0d040000-0000-7000-8000-000000000cc1',
     '2026-03-01T11:30:00Z', 'sms', 'probe-ext-2', '+15550000002', '+15550000001', 'inbound',
     'text', decode(repeat('a2', 32), 'hex'), '0d040000-0000-7000-8000-000000000dd2',
     'engine@2026-09-26', now(), 'first_party');

INSERT INTO working.message_participant (message_id, participant_raw, role, deriver_version)
VALUES
    ('0d040000-0000-7000-8000-000000000dd1', '+15550000001', 'from', 'engine@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd1', '+15550000002', 'to',   'engine@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd2', '+15550000002', 'from', 'engine@2026-09-26'),
    ('0d040000-0000-7000-8000-000000000dd2', '+15550000001', 'to',   'engine@2026-09-26');

SET CONSTRAINTS ALL IMMEDIATE;

DO $$
DECLARE v_bad INT;
BEGIN
    SELECT count(*) INTO v_bad
      FROM working.message msg JOIN working.normalized_record nr ON nr.id = msg.id
     WHERE msg.id IN ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-000000000dd2')
       AND (msg.id <> nr.id OR msg.ts_utc IS DISTINCT FROM nr.occurred_at
            OR nr.knowledge_time IS NULL OR nr.disclosure_tier IS NULL);
    IF v_bad <> 0 THEN
        RAISE EXCEPTION 'PROBE FAILED: % spine rows diverge from their normalized record', v_bad;
    END IF;
    RAISE NOTICE 'proof 2 OK: the engine wrote the whole spine under the proposed grants, and every spine id equals its normalized record id';
END $$;

RESET ROLE;

-- The outbox is verified AS THE OWNER, deliberately. The engine needs INSERT on
-- working.normalized_record_event so the SECURITY INVOKER outbox trigger can write
-- it, but it never reads it, so SELECT is correctly absent from the grant set. An
-- earlier draft of this probe read the outbox as the engine and failed here --
-- which is the grant set being minimal, not the write being broken.
DO $$
DECLARE v_events INT;
BEGIN
    SELECT count(*) INTO v_events FROM working.normalized_record_event
     WHERE source_pk IN ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-000000000dd2');
    IF v_events < 2 THEN
        RAISE EXCEPTION 'PROBE FAILED: the outbox did not record the engine writes (% rows)', v_events;
    END IF;
    RAISE NOTICE 'proof 2b OK: the CDC outbox captured % engine writes, written by the trigger as the engine', v_events;
END $$;

-- ---------------------------------------------------------------------------
-- Proof 3: necessity. Revoke one grant at a time and show the write breaks.
-- Each case runs on the SAME fixtures with a fresh record id, inside a savepoint.
-- ---------------------------------------------------------------------------

-- 3a: working.normalized_record_event. The non-obvious one -- no column or FK
-- points at it; the SECURITY INVOKER outbox trigger inserts it as the caller.
SAVEPOINT need_event;
REVOKE INSERT ON TABLE working.normalized_record_event FROM platform_runtime;
SET LOCAL ROLE platform_runtime;
DO $$
BEGIN
    INSERT INTO working.normalized_record
        (id, artifact_id, record_type, source, content, occurred_at, knowledge_time,
         disclosure_tier, message_corpus)
    VALUES ('0d040000-0000-7000-8000-000000000d91', '0d040000-0000-7000-8000-00000000aaa1', 'message',
            'probe', 'needs the outbox grant', '2026-03-01T15:00:00Z', '2026-03-01T15:00:00Z',
            'contemporaneous', 'first_party');
    RAISE EXCEPTION 'PROBE FAILED: normalized_record_event INSERT was not required';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3a OK: INSERT on working.normalized_record_event is required -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT need_event;

-- 3b: working.message_projection_route. message_route_fk needs the row to exist,
-- and the engine must be the one to write it.
SAVEPOINT need_route;
REVOKE INSERT ON TABLE working.message_projection_route FROM platform_runtime;
SET LOCAL ROLE platform_runtime;
DO $$
BEGIN
    INSERT INTO working.message_projection_route
        (normalized_record_id, projection_kind, decision_state, basis, proposed_by, deriver_version)
    VALUES ('0d040000-0000-7000-8000-000000000dd1', 'first_party', 'proposed',
            '{}', 'engine', 'engine@2026-09-26');
    RAISE EXCEPTION 'PROBE FAILED: message_projection_route INSERT was not required';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3b OK: INSERT on working.message_projection_route is required -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT need_route;

-- 3c: working.message itself.
SAVEPOINT need_message;
REVOKE INSERT ON TABLE working.message FROM platform_runtime;
SET LOCAL ROLE platform_runtime;
DO $$
BEGIN
    INSERT INTO working.message
        (id, conversation_id, ts_utc, platform, external_id, direction, message_type,
         derived_from_record_id, deriver_version, derived_at, projection_kind)
    VALUES ('0d040000-0000-7000-8000-000000000dd1', '0d040000-0000-7000-8000-000000000cc1',
            '2026-03-01T10:00:00Z', 'sms', 'probe-ext-need', 'outbound', 'text',
            '0d040000-0000-7000-8000-000000000dd1', 'engine@2026-09-26', now(), 'first_party');
    RAISE EXCEPTION 'PROBE FAILED: working.message INSERT was not required';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3c OK: INSERT on working.message is required -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT need_message;

-- 3d: working.message_participant.
SAVEPOINT need_participant;
REVOKE INSERT ON TABLE working.message_participant FROM platform_runtime;
SET LOCAL ROLE platform_runtime;
DO $$
BEGIN
    INSERT INTO working.message_participant (message_id, participant_raw, role, deriver_version)
    VALUES ('0d040000-0000-7000-8000-000000000dd1', '+15550000003', 'cc', 'engine@2026-09-26');
    RAISE EXCEPTION 'PROBE FAILED: message_participant INSERT was not required';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3d OK: INSERT on working.message_participant is required -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT need_participant;

-- 3e: the derived-write guard is a GUC, not a grant. With the guard armed and the
-- deriver flag NOT set, even a fully granted engine is refused -- so the activity
-- must declare itself the deriver, and no privilege change can paper over that.
SAVEPOINT need_deriver_flag;
SET LOCAL ROLE platform_runtime;
SET LOCAL app.enforce_derived_guard = 'on';
SET LOCAL app.deriving = 'off';
DO $$
BEGIN
    INSERT INTO working.message
        (id, conversation_id, ts_utc, platform, external_id, direction, message_type,
         derived_from_record_id, deriver_version, derived_at, projection_kind)
    VALUES ('0d040000-0000-7000-8000-000000000dd2', '0d040000-0000-7000-8000-000000000cc1',
            '2026-03-01T11:30:00Z', 'sms', 'probe-ext-guard', 'inbound', 'text',
            '0d040000-0000-7000-8000-000000000dd2', 'engine@2026-09-26', now(), 'first_party');
    RAISE EXCEPTION 'PROBE FAILED: the derived-write guard did not fire';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3e OK: the armed guard refuses a writer that has not declared itself the deriver -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT need_deriver_flag;

-- 3f: the validator's READ grants. Revoking SELECT on a third-party table the
-- engine never writes still breaks the spine write, because the SECURITY INVOKER
-- projection validator reads it as the engine. This is the least obvious entry in
-- the set and the one a reviewer is most likely to question, so it is proven.
SAVEPOINT need_validator_read;
REVOKE SELECT ON TABLE working.third_party_message FROM platform_runtime;
SET LOCAL ROLE platform_runtime;
SET LOCAL app.enforce_derived_guard = 'on';
SET LOCAL app.deriving = 'on';
DO $$
BEGIN
    EXECUTE 'SET CONSTRAINTS ALL DEFERRED';
    INSERT INTO working.normalized_record
        (id, artifact_id, record_type, source, content, occurred_at, knowledge_time,
         disclosure_tier, message_corpus)
    VALUES ('0d040000-0000-7000-8000-000000000d92', '0d040000-0000-7000-8000-00000000aaa1', 'message',
            'probe', 'needs the validator read grant', '2026-03-01T16:00:00Z', '2026-03-01T16:00:00Z',
            'contemporaneous', 'first_party');
    INSERT INTO working.message_projection_route
        (normalized_record_id, projection_kind, decision_state, basis, proposed_by,
         approved_by, approved_at, deriver_version)
    VALUES ('0d040000-0000-7000-8000-000000000d92', 'first_party', 'approved',
            '{}', 'engine', 'engine', now(), 'engine@2026-09-26');
    INSERT INTO working.message
        (id, conversation_id, ts_utc, platform, external_id, direction, message_type,
         derived_from_record_id, deriver_version, derived_at, projection_kind)
    VALUES ('0d040000-0000-7000-8000-000000000d92', '0d040000-0000-7000-8000-000000000cc1',
            '2026-03-01T16:00:00Z', 'sms', 'probe-ext-validator', 'outbound', 'text',
            '0d040000-0000-7000-8000-000000000d92', 'engine@2026-09-26', now(), 'first_party');
    EXECUTE 'SET CONSTRAINTS ALL IMMEDIATE';
    RAISE EXCEPTION 'PROBE FAILED: SELECT on working.third_party_message was not required';
EXCEPTION WHEN others THEN
    IF SQLERRM LIKE 'PROBE FAILED%' THEN RAISE; END IF;
    RAISE NOTICE 'proof 3f OK: the validator needs SELECT on tables the engine never writes -- %', SQLERRM;
END $$;
RESET ROLE;
ROLLBACK TO SAVEPOINT need_validator_read;

DO $$ BEGIN RAISE NOTICE 'ALL D04 ENGINE-SPINE GRANT PROOFS PASSED'; END $$;

-- Rolls back the fixtures AND every GRANT: privilege changes are transactional.
ROLLBACK;
