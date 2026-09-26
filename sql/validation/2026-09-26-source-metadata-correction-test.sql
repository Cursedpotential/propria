-- Overlay proof for owner metadata corrections (scripts/2026-09-26-source-metadata-correction-overlay.sql).
--
-- Byline: Claude Code · Opus 5.5 · 2026-09-26
--
-- Proves, against a real PostgreSQL holding the platform schema plus the overlay block:
--   1. a correction chain (correct -> correct -> retract) is accepted in order and the
--      current view shows the newest revision, including a retract;
--   2. the observed metadata row is untouched by any correction (it is never written);
--   3. the table is append-only (UPDATE, DELETE and TRUNCATE all raise);
--   4. a revision can only supersede a revision of the same file and field;
--   5. a first revision cannot be a retract, a correct needs a value and a retract cannot carry one;
--   6. context_reader can read corrections but cannot write them.
--
-- Run as a superuser (fixtures for upstream tables are inserted with
-- session_replication_role = replica so no ingest chain is needed). Every row
-- is created inside this transaction and discarded by the final ROLLBACK.
-- NEVER run against the live platform database; it is a throwaway-instance proof.

\set ON_ERROR_STOP on
BEGIN;

SET LOCAL session_replication_role = replica;
INSERT INTO registry.matter (id, title, created_by)
VALUES ('0190a000-0000-7000-8000-00000000aa02', 'metadata proof matter', 'proof');
INSERT INTO registry.court_case (id, matter_id, caption, created_by)
VALUES ('0190a000-0000-7000-8000-00000000cc02', '0190a000-0000-7000-8000-00000000aa02', 'metadata proof case', 'proof');
INSERT INTO context.proffer_preview_binding (preview_handle, request_id, source_ref, workflow_id, run_id, parser_options_ref)
VALUES ('proofmetadata_0123456789abcdefABCDEF', 'proof-metadata-request', 'b2://proof-bucket/proof/IMG_0001.jpg',
        'proof-metadata-request', 'proof-run', 'proof-options');
INSERT INTO context.source (id, source_key, provenance_class)
VALUES ('0190a000-0000-7000-8000-0000000000d1', 'b2://proof-bucket/proof/IMG_0001.jpg', 'first_party_authored');
INSERT INTO context.source_version
    (id, source_id, version_ordinal, workflow_id, submission_idempotency_key, declared_format, acquired_at,
     status, matter_id, court_case_id)
VALUES ('0190a000-0000-7000-8000-0000000000d2', '0190a000-0000-7000-8000-0000000000d1', 1, 'proof-metadata-request',
        'proof-metadata-submission', 'image', '2026-09-26T00:00:00Z', 'registered',
        '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02');
INSERT INTO context.source_metadata
    (id, source_version_id, metadata_class, metadata, extractor_id, extractor_version,
     extraction_activity_receipt_id, generated_at)
VALUES ('0190a000-0000-7000-8000-0000000000d3', '0190a000-0000-7000-8000-0000000000d2', 'embedded',
        '{"EXIF:DateTimeOriginal":"2021:05:04 10:11:12"}', 'proof-extractor', '1',
        '0190a000-0000-7000-8000-0000000000d4', '2026-09-26T00:00:00Z');
SET LOCAL session_replication_role = origin;

INSERT INTO context.source_metadata_correction
    (correction_ref, source_version_id, subject_sha256, field_key, preview_handle, matter_id, court_case_id,
     revision, supersedes_ref, action, source_value, corrected_value, change_reason,
     actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
VALUES
    ('0190a000-0000-7000-8000-0000000000c1', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
     'embedded:EXIF:DateTimeOriginal', 'proofmetadata_0123456789abcdefABCDEF',
     '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
     1, NULL, 'correct', '"2021:05:04 10:11:12"', '"2021:05:04 22:11:12"', 'camera clock was twelve hours off',
     'proof-uid', 'proof', 'proof-correction-1', decode(repeat('01', 32), 'hex'), 'metadata-correction://proof-1'),
    ('0190a000-0000-7000-8000-0000000000c2', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
     'embedded:EXIF:DateTimeOriginal', 'proofmetadata_0123456789abcdefABCDEF',
     '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
     2, '0190a000-0000-7000-8000-0000000000c1', 'correct', '"2021:05:04 10:11:12"', '"2021:05:04 21:11:12"',
     'daylight saving', 'proof-uid', 'proof', 'proof-correction-2', decode(repeat('02', 32), 'hex'), 'metadata-correction://proof-2'),
    ('0190a000-0000-7000-8000-0000000000c3', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
     'embedded:EXIF:DateTimeOriginal', 'proofmetadata_0123456789abcdefABCDEF',
     '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
     3, '0190a000-0000-7000-8000-0000000000c2', 'retract', '"2021:05:04 10:11:12"', NULL,
     'the device clock was right after all', 'proof-uid', 'proof', 'proof-correction-3', decode(repeat('03', 32), 'hex'),
     'metadata-correction://proof-3');

DO $$
DECLARE
    n bigint;
    current_action text;
    observed jsonb;
BEGIN
    SELECT action INTO current_action FROM context.vw_source_metadata_correction_current
     WHERE source_version_id = '0190a000-0000-7000-8000-0000000000d2' AND field_key = 'embedded:EXIF:DateTimeOriginal';
    IF current_action IS DISTINCT FROM 'retract' THEN
        RAISE EXCEPTION 'FAIL 1: current view shows %, expected the newest revision (retract)', current_action;
    END IF;
    RAISE NOTICE 'PASS 1: correct -> correct -> retract chain accepted; the current view shows the newest revision';

    SELECT metadata INTO observed FROM context.source_metadata WHERE id = '0190a000-0000-7000-8000-0000000000d3';
    IF observed ->> 'EXIF:DateTimeOriginal' IS DISTINCT FROM '2021:05:04 10:11:12' THEN
        RAISE EXCEPTION 'FAIL 2: the observed metadata row changed';
    END IF;
    RAISE NOTICE 'PASS 2: the observed metadata row is untouched';

    BEGIN
        UPDATE context.source_metadata_correction SET change_reason = 'rewrite';
        RAISE EXCEPTION 'FAIL 3a: UPDATE was allowed';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM NOT LIKE '%append-only%' THEN RAISE; END IF;
    END;
    BEGIN
        DELETE FROM context.source_metadata_correction;
        RAISE EXCEPTION 'FAIL 3b: DELETE was allowed';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM NOT LIKE '%append-only%' THEN RAISE; END IF;
    END;
    BEGIN
        TRUNCATE context.source_metadata_correction;
        RAISE EXCEPTION 'FAIL 3c: TRUNCATE was allowed';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM NOT LIKE '%append-only%' THEN RAISE; END IF;
    END;
    RAISE NOTICE 'PASS 3: UPDATE, DELETE and TRUNCATE are refused (append-only)';

    BEGIN
        INSERT INTO context.source_metadata_correction
            (correction_ref, source_version_id, subject_sha256, field_key, preview_handle, matter_id, court_case_id,
             revision, supersedes_ref, action, corrected_value, change_reason,
             actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
        VALUES ('0190a000-0000-7000-8000-0000000000c9', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
                'embedded:EXIF:Make', 'proofmetadata_0123456789abcdefABCDEF',
                '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
                2, '0190a000-0000-7000-8000-0000000000c1', 'correct', '"Other"', 'must fail',
                'proof-uid', 'proof', 'proof-correction-bad', decode(repeat('09', 32), 'hex'), 'metadata-correction://bad');
        RAISE EXCEPTION 'FAIL 4: a revision superseded another field''s revision';
    EXCEPTION WHEN foreign_key_violation THEN
        RAISE NOTICE 'PASS 4: a revision can only supersede a revision of the same file and field';
    END;

    BEGIN
        INSERT INTO context.source_metadata_correction
            (correction_ref, source_version_id, subject_sha256, field_key, preview_handle, matter_id, court_case_id,
             revision, action, change_reason, actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
        VALUES ('0190a000-0000-7000-8000-0000000000ca', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
                'embedded:EXIF:Model', 'proofmetadata_0123456789abcdefABCDEF',
                '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
                1, 'retract', 'must fail', 'proof-uid', 'proof', 'proof-correction-bad-2', decode(repeat('0a', 32), 'hex'),
                'metadata-correction://bad-2');
        RAISE EXCEPTION 'FAIL 5a: a first revision retract was accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
    BEGIN
        INSERT INTO context.source_metadata_correction
            (correction_ref, source_version_id, subject_sha256, field_key, preview_handle, matter_id, court_case_id,
             revision, action, change_reason, actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
        VALUES ('0190a000-0000-7000-8000-0000000000cb', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
                'embedded:EXIF:Model', 'proofmetadata_0123456789abcdefABCDEF',
                '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
                1, 'correct', 'must fail', 'proof-uid', 'proof', 'proof-correction-bad-3', decode(repeat('0b', 32), 'hex'),
                'metadata-correction://bad-3');
        RAISE EXCEPTION 'FAIL 5b: a correct without a value was accepted';
    EXCEPTION WHEN check_violation THEN
        RAISE NOTICE 'PASS 5: a first retract and a value-less correct are refused';
    END;
END
$$;

SET LOCAL ROLE context_reader;
DO $$
DECLARE
    n bigint;
BEGIN
    SELECT count(*) INTO n FROM context.source_metadata_correction;
    IF n <> 3 THEN RAISE EXCEPTION 'FAIL 6a: context_reader read % correction row(s), expected 3', n; END IF;
    BEGIN
        INSERT INTO context.source_metadata_correction
            (correction_ref, source_version_id, subject_sha256, field_key, preview_handle, matter_id, court_case_id,
             revision, action, corrected_value, change_reason, actor_subject_uid, actor_username, idempotency_key,
             content_digest, receipt_ref)
        VALUES ('0190a000-0000-7000-8000-0000000000cc', '0190a000-0000-7000-8000-0000000000d2', decode(repeat('ab', 32), 'hex'),
                'embedded:EXIF:Software', 'proofmetadata_0123456789abcdefABCDEF',
                '0190a000-0000-7000-8000-00000000aa02', '0190a000-0000-7000-8000-00000000cc02',
                1, 'correct', '"x"', 'must fail', 'proof-uid', 'proof', 'proof-correction-bad-4', decode(repeat('0c', 32), 'hex'),
                'metadata-correction://bad-4');
        RAISE EXCEPTION 'FAIL 6b: context_reader wrote a correction';
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE NOTICE 'PASS 6: context_reader reads corrections and cannot write them';
    END;
END
$$;
RESET ROLE;

ROLLBACK;
