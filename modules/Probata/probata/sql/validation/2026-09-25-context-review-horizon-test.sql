-- Horizon proof for the context review overlays (scripts/2026-09-25-context-review-overlay.sql).
--
-- Byline: Claude Code · Opus 5.5 · 2026-09-25
--
-- Proves, against a real PostgreSQL holding the platform schema plus the overlay block:
--   1. an as-lived session (app.horizon set) reads ZERO foreshadowing rows, through
--      both the flag view and the combined review view, while the review itself
--      stays visible;
--   2. a hindsight session (app.horizon unset/empty) reads the same flag;
--   3. context_reader cannot read the base foreshadowing table at all, and through
--      the view it sees the flag only under hindsight;
--   4. both tables are append-only (UPDATE, DELETE and TRUNCATE all raise);
--   5. a foreshadowing row cannot carry any horizon but 'hindsight';
--   6. a revision can only supersede a revision of the same record and matter.
--
-- Run as a superuser (fixtures for upstream tables are inserted with
-- session_replication_role = replica so no ingest chain is needed). Every row
-- is created inside this transaction and discarded by the final ROLLBACK.
-- NEVER run against the live platform database; it is a throwaway-instance proof.

\set ON_ERROR_STOP on
BEGIN;

-- Fixture identities (fixed so the assertions below can name them).
SET LOCAL session_replication_role = replica;
INSERT INTO registry.matter (id, title, created_by)
VALUES ('0190a000-0000-7000-8000-00000000aa01', 'horizon proof matter', 'proof');
INSERT INTO registry.court_case (id, matter_id, caption, created_by)
VALUES ('0190a000-0000-7000-8000-00000000cc01', '0190a000-0000-7000-8000-00000000aa01', 'horizon proof case', 'proof');
INSERT INTO context.proffer_preview_binding (preview_handle, request_id, source_ref, workflow_id, run_id, parser_options_ref)
VALUES ('proofhandle_0123456789abcdefABCDEF', 'proof-request', 'r2://casebible-sorted/proof.xml', 'proof-request', 'proof-run', 'proof-options');
INSERT INTO context.normalized_record_identity
    (id, normalized_generation_id, source_version_id, record_ordinal, record_type, canonical_bytes, canonicalization, normalized_payload)
VALUES
    ('0190a000-0000-7000-8000-0000000000e1', '0190a000-0000-7000-8000-0000000000f1', '0190a000-0000-7000-8000-0000000000f2', 0, 'message',
     convert_to('{}', 'UTF8'), 'normalized-record-postgresql18-jsonb-text-utf8-sha256-v1', '{}'::jsonb),
    ('0190a000-0000-7000-8000-0000000000e2', '0190a000-0000-7000-8000-0000000000f1', '0190a000-0000-7000-8000-0000000000f2', 1, 'message',
     convert_to('{}', 'UTF8'), 'normalized-record-postgresql18-jsonb-text-utf8-sha256-v1', '{}'::jsonb);
SET LOCAL session_replication_role = origin;

-- The overlays themselves are inserted with every constraint and trigger live.
INSERT INTO context.record_context_review_revision
    (review_ref, normalized_record_id, preview_handle, matter_id, court_case_id, revision, supersedes_ref,
     addressed_to, about, about_child, relevant, change_reason, actor_subject_uid, actor_username,
     idempotency_key, content_digest, receipt_ref)
VALUES
    ('0190a000-0000-7000-8000-0000000000a1', '0190a000-0000-7000-8000-0000000000e1', 'proofhandle_0123456789abcdefABCDEF',
     '0190a000-0000-7000-8000-00000000aa01', '0190a000-0000-7000-8000-00000000cc01', 1, NULL,
     '[{"label":"Recipient","entity_id":null}]', '[{"label":"The child","entity_id":null}]', 'yes', true,
     'proof review', 'proof-uid', 'proof', 'proof-review-1', decode(repeat('ab', 32), 'hex'), 'context-review://proof-1');
INSERT INTO context.record_foreshadowing_flag
    (flag_ref, normalized_record_id, preview_handle, matter_id, court_case_id, revision, supersedes_ref,
     foreshadowing, note, change_reason, actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
VALUES
    ('0190a000-0000-7000-8000-0000000000b1', '0190a000-0000-7000-8000-0000000000e1', 'proofhandle_0123456789abcdefABCDEF',
     '0190a000-0000-7000-8000-00000000aa01', '0190a000-0000-7000-8000-00000000cc01', 1, NULL,
     true, 'known in hindsight', 'proof flag', 'proof-uid', 'proof', 'proof-flag-1', decode(repeat('cd', 32), 'hex'), 'foreshadowing://proof-1');

DO $$
DECLARE
    rec constant uuid := '0190a000-0000-7000-8000-0000000000e1';
    n bigint;
BEGIN
    -- 1. As-lived / ignorant session: the horizon is a cutoff timestamp.
    PERFORM set_config('app.horizon', '2023-06-01T00:00:00+00', true);
    SELECT count(*) INTO n FROM context.vw_record_foreshadowing_current WHERE normalized_record_id = rec;
    IF n <> 0 THEN RAISE EXCEPTION 'FAIL 1a: as-lived read returned % foreshadowing row(s)', n; END IF;
    SELECT count(*) INTO n FROM context.vw_record_context_review_horizon
        WHERE normalized_record_id = rec AND (foreshadowing IS NOT NULL OR foreshadowing_ref IS NOT NULL);
    IF n <> 0 THEN RAISE EXCEPTION 'FAIL 1b: as-lived combined view leaked % foreshadowing value(s)', n; END IF;
    SELECT count(*) INTO n FROM context.vw_record_context_review_current WHERE normalized_record_id = rec;
    IF n <> 1 THEN RAISE EXCEPTION 'FAIL 1c: as-lived session lost the review itself (% rows)', n; END IF;
    RAISE NOTICE 'PASS 1: as-lived session reads 0 foreshadowing rows (flag view and combined view); the review stays visible';

    -- 2. Hindsight session: app.horizon empty (the spine convention).
    PERFORM set_config('app.horizon', '', true);
    SELECT count(*) INTO n FROM context.vw_record_foreshadowing_current WHERE normalized_record_id = rec AND foreshadowing;
    IF n <> 1 THEN RAISE EXCEPTION 'FAIL 2a: hindsight read returned % foreshadowing row(s), expected 1', n; END IF;
    SELECT count(*) INTO n FROM context.vw_record_context_review_horizon
        WHERE normalized_record_id = rec AND foreshadowing AND foreshadowing_horizon = 'hindsight';
    IF n <> 1 THEN RAISE EXCEPTION 'FAIL 2b: hindsight combined view returned % flagged row(s), expected 1', n; END IF;
    RAISE NOTICE 'PASS 2: hindsight session reads the flag (horizon = hindsight) through both views';
END
$$;

-- 3. Privileges: context_reader never touches the base table.
SET LOCAL ROLE context_reader;
DO $$
DECLARE
    n bigint;
BEGIN
    BEGIN
        SELECT count(*) INTO n FROM context.record_foreshadowing_flag;
        RAISE EXCEPTION 'FAIL 3a: context_reader read the base foreshadowing table (% rows)', n;
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE NOTICE 'PASS 3a: context_reader is denied the base foreshadowing table';
    END;
    PERFORM set_config('app.horizon', '2023-06-01T00:00:00+00', true);
    SELECT count(*) INTO n FROM context.vw_record_foreshadowing_current;
    IF n <> 0 THEN RAISE EXCEPTION 'FAIL 3b: context_reader as-lived read returned % foreshadowing row(s)', n; END IF;
    PERFORM set_config('app.horizon', '', true);
    SELECT count(*) INTO n FROM context.vw_record_foreshadowing_current;
    IF n <> 1 THEN RAISE EXCEPTION 'FAIL 3c: context_reader hindsight read returned % row(s), expected 1', n; END IF;
    RAISE NOTICE 'PASS 3b/3c: through the view, context_reader sees the flag only under hindsight';
END
$$;
RESET ROLE;

-- 4-6. Guards.
DO $$
BEGIN
    BEGIN
        UPDATE context.record_context_review_revision SET relevant = false;
        RAISE EXCEPTION 'FAIL 4a: review UPDATE was allowed';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM NOT LIKE '%append-only%' THEN RAISE; END IF;
    END;
    BEGIN
        DELETE FROM context.record_foreshadowing_flag;
        RAISE EXCEPTION 'FAIL 4b: foreshadowing DELETE was allowed';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM NOT LIKE '%append-only%' THEN RAISE; END IF;
    END;
    BEGIN
        TRUNCATE context.record_foreshadowing_flag;
        RAISE EXCEPTION 'FAIL 4c: foreshadowing TRUNCATE was allowed';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM NOT LIKE '%append-only%' THEN RAISE; END IF;
    END;
    RAISE NOTICE 'PASS 4: UPDATE, DELETE and TRUNCATE are refused (append-only)';

    BEGIN
        INSERT INTO context.record_foreshadowing_flag
            (flag_ref, normalized_record_id, preview_handle, matter_id, court_case_id, revision, supersedes_ref,
             foreshadowing, horizon, change_reason, actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
        VALUES ('0190a000-0000-7000-8000-0000000000b9', '0190a000-0000-7000-8000-0000000000e2', 'proofhandle_0123456789abcdefABCDEF',
                '0190a000-0000-7000-8000-00000000aa01', '0190a000-0000-7000-8000-00000000cc01', 1, NULL,
                true, 'as_lived', 'must fail', 'proof-uid', 'proof', 'proof-flag-bad', decode(repeat('ef', 32), 'hex'), 'foreshadowing://bad');
        RAISE EXCEPTION 'FAIL 5: a foreshadowing row with horizon as_lived was accepted';
    EXCEPTION WHEN check_violation THEN
        RAISE NOTICE 'PASS 5: foreshadowing rows cannot carry any horizon but hindsight';
    END;

    BEGIN
        INSERT INTO context.record_context_review_revision
            (review_ref, normalized_record_id, preview_handle, matter_id, court_case_id, revision, supersedes_ref,
             about_child, change_reason, actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
        VALUES ('0190a000-0000-7000-8000-0000000000a9', '0190a000-0000-7000-8000-0000000000e2', 'proofhandle_0123456789abcdefABCDEF',
                '0190a000-0000-7000-8000-00000000aa01', '0190a000-0000-7000-8000-00000000cc01', 2,
                '0190a000-0000-7000-8000-0000000000a1', 'no', 'must fail', 'proof-uid', 'proof', 'proof-review-bad',
                decode(repeat('01', 32), 'hex'), 'context-review://bad');
        RAISE EXCEPTION 'FAIL 6: a revision superseded another record''s revision';
    EXCEPTION WHEN foreign_key_violation THEN
        RAISE NOTICE 'PASS 6: a revision can only supersede a revision of the same record and matter';
    END;
END
$$;

ROLLBACK;
