-- Owner corrections to a file's metadata, as append-only attributed overlays.
--
-- Byline: Claude Code · Opus 5.5 · 2026-09-26
-- Owner requirement 2026-09-25 19:13 / 19:15 (docs/pending-review/2026-09-25-metadata-context-review-entities.md §1):
-- "the metadata review and correction ... ALL metadata for ALL files, including any available
-- accompanying sidecar". Source values stay immutable; a correction is a new revision.
--
-- ⚠ NOT APPLIED. The parent session applies it after review: dry-run inside a
--   transaction that ends in ROLLBACK first, then this file as written.
--
-- The block between the BEGIN/END markers is copied verbatim into
-- sql/bootstrap/schema_snapshot_20260907.sql (the database; D-142 §3, D-152),
-- directly after the context-review-overlay block, so a rebuild from the
-- snapshot and this apply script produce the same objects.
--
-- Idempotent: every statement is IF NOT EXISTS / OR REPLACE / a repeatable
-- GRANT or COMMENT. Re-running changes nothing. No existing object is altered.
--
-- Roles: the engine connects as platform_runtime and writes through its
-- context_import_writer membership (the same path it uses for
-- context.proffer_source_context_revision). context_reader reads; platform_app
-- keeps its schema-wide ALL.

BEGIN;

-- >>> BEGIN source-metadata-correction-overlay block (Claude Code · Opus 5.5 · 2026-09-26) >>>
--
-- Corrections are OVERLAYS. The observed value -- a context.source_metadata row,
-- a sidecar beside the object, an attachment's recorded facts -- is never
-- written. Every row is actor-bound, timestamped, receipt-addressed and
-- append-only (context.forbid_mutation): a correction is a new revision that
-- supersedes exactly the newest revision of the same field on the same file.
--
-- The file is named by its content digest inside one source version: the
-- retained original, a retained member, or an attachment the run projected.
-- field_key names where the value came from and its path, for example
-- 'embedded:EXIF:DateTimeOriginal', 'sidecar:takeout:photoTakenTime.timestamp'
-- or 'filesystem:modified_at'. source_value echoes what the owner saw when he
-- corrected it; it is audit, never a replacement of the observation.

CREATE TABLE IF NOT EXISTS context.source_metadata_correction (
    correction_ref uuid NOT NULL,
    source_version_id uuid NOT NULL,
    subject_sha256 bytea NOT NULL,
    field_key text NOT NULL,
    preview_handle text NOT NULL,
    matter_id uuid NOT NULL,
    court_case_id uuid NOT NULL,
    revision integer NOT NULL,
    supersedes_ref uuid,
    action text NOT NULL,
    source_value jsonb,
    corrected_value jsonb,
    change_reason text NOT NULL,
    actor_subject_uid text NOT NULL,
    actor_username text NOT NULL,
    idempotency_key text NOT NULL,
    content_digest bytea NOT NULL,
    receipt_ref text NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT source_metadata_correction_pkey PRIMARY KEY (correction_ref),
    CONSTRAINT source_metadata_correction_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT source_metadata_correction_receipt_ref_key UNIQUE (receipt_ref),
    CONSTRAINT source_metadata_correction_subject_revision_key UNIQUE (source_version_id, subject_sha256, field_key, revision),
    CONSTRAINT source_metadata_correction_scope_key UNIQUE (correction_ref, source_version_id, subject_sha256, field_key),
    CONSTRAINT source_metadata_correction_revision_check CHECK ((revision >= 1)),
    CONSTRAINT source_metadata_correction_chain_check CHECK ((((revision = 1) AND (supersedes_ref IS NULL)) OR ((revision > 1) AND (supersedes_ref IS NOT NULL)))),
    CONSTRAINT source_metadata_correction_subject_sha256_check CHECK ((octet_length(subject_sha256) = 32)),
    CONSTRAINT source_metadata_correction_field_key_check CHECK (((length(field_key) >= 3) AND (length(field_key) <= 512) AND (field_key ~ '^[a-z][a-z_]{0,31}:.'::text) AND (field_key !~ '[[:cntrl:]]'::text))),
    CONSTRAINT source_metadata_correction_action_check CHECK ((action = ANY (ARRAY['correct'::text, 'retract'::text]))),
    CONSTRAINT source_metadata_correction_action_value_check CHECK ((((action = 'correct'::text) AND (corrected_value IS NOT NULL)) OR ((action = 'retract'::text) AND (corrected_value IS NULL)))),
    CONSTRAINT source_metadata_correction_first_revision_check CHECK ((NOT ((revision = 1) AND (action = 'retract'::text)))),
    CONSTRAINT source_metadata_correction_source_value_check CHECK (((source_value IS NULL) OR (octet_length((source_value)::text) <= 65536))),
    CONSTRAINT source_metadata_correction_corrected_value_check CHECK (((corrected_value IS NULL) OR (octet_length((corrected_value)::text) <= 65536))),
    CONSTRAINT source_metadata_correction_preview_handle_check CHECK ((preview_handle ~ '^[A-Za-z0-9_-]{32,128}$'::text)),
    CONSTRAINT source_metadata_correction_change_reason_check CHECK (((length(btrim(change_reason)) > 0) AND (octet_length(change_reason) <= 4000))),
    CONSTRAINT source_metadata_correction_actor_subject_uid_check CHECK ((length(btrim(actor_subject_uid)) > 0)),
    CONSTRAINT source_metadata_correction_actor_username_check CHECK ((length(btrim(actor_username)) > 0)),
    CONSTRAINT source_metadata_correction_idempotency_key_check CHECK ((length(btrim(idempotency_key)) > 0)),
    CONSTRAINT source_metadata_correction_content_digest_check CHECK ((octet_length(content_digest) = 32)),
    CONSTRAINT source_metadata_correction_receipt_ref_check CHECK ((length(btrim(receipt_ref)) > 0)),
    CONSTRAINT source_metadata_correction_source_version_fk FOREIGN KEY (source_version_id) REFERENCES context.source_version(id) ON DELETE RESTRICT,
    CONSTRAINT source_metadata_correction_preview_fk FOREIGN KEY (preview_handle) REFERENCES context.proffer_preview_binding(preview_handle) ON DELETE RESTRICT,
    CONSTRAINT source_metadata_correction_matter_fk FOREIGN KEY (matter_id) REFERENCES registry.matter(id) ON DELETE RESTRICT,
    CONSTRAINT source_metadata_correction_court_case_fk FOREIGN KEY (court_case_id, matter_id) REFERENCES registry.court_case(id, matter_id) ON DELETE RESTRICT,
    CONSTRAINT source_metadata_correction_supersedes_fk FOREIGN KEY (supersedes_ref, source_version_id, subject_sha256, field_key) REFERENCES context.source_metadata_correction(correction_ref, source_version_id, subject_sha256, field_key) ON DELETE RESTRICT
);

COMMENT ON TABLE context.source_metadata_correction IS 'Append-only owner corrections to one metadata field of one file (named by content digest inside a source version). An overlay: the observed value is never written. Byline: Claude Code · Opus 5.5 · 2026-09-26.';
COMMENT ON COLUMN context.source_metadata_correction.subject_sha256 IS 'SHA-256 of the file the field belongs to: the retained original, a retained member, or a projected attachment of the same source version.';
COMMENT ON COLUMN context.source_metadata_correction.field_key IS '<origin>:<path>, e.g. embedded:EXIF:DateTimeOriginal, sidecar:takeout:photoTakenTime.timestamp, filesystem:modified_at.';
COMMENT ON COLUMN context.source_metadata_correction.action IS 'correct = corrected_value replaces the observed value in the owner view; retract = the owner withdrew the previous correction (the observed value stands again).';
COMMENT ON COLUMN context.source_metadata_correction.source_value IS 'The observed value as the owner saw it when correcting (audit echo); NULL when the field was absent.';

CREATE OR REPLACE TRIGGER source_metadata_correction_append_only
    BEFORE UPDATE OR DELETE ON context.source_metadata_correction
    FOR EACH ROW EXECUTE FUNCTION context.forbid_mutation();
CREATE OR REPLACE TRIGGER source_metadata_correction_no_truncate
    BEFORE TRUNCATE ON context.source_metadata_correction
    FOR EACH STATEMENT EXECUTE FUNCTION context.forbid_mutation();

CREATE INDEX IF NOT EXISTS source_metadata_correction_subject_idx
    ON context.source_metadata_correction USING btree (source_version_id, subject_sha256, field_key, revision DESC);

CREATE OR REPLACE VIEW context.vw_source_metadata_correction_current AS
 SELECT DISTINCT ON (c.source_version_id, c.subject_sha256, c.field_key)
    c.source_version_id,
    c.subject_sha256,
    c.field_key,
    c.correction_ref,
    c.revision,
    c.action,
    c.source_value,
    c.corrected_value,
    c.change_reason,
    c.actor_username,
    c.receipt_ref,
    c.recorded_at
   FROM context.source_metadata_correction c
  ORDER BY c.source_version_id, c.subject_sha256, c.field_key, c.revision DESC;

COMMENT ON VIEW context.vw_source_metadata_correction_current IS 'Newest correction revision per file field. action = retract means the observed value stands again.';

GRANT SELECT,INSERT ON TABLE context.source_metadata_correction TO context_import_writer;
GRANT SELECT ON TABLE context.source_metadata_correction TO context_reader;
GRANT SELECT ON TABLE context.source_metadata_correction TO platform_runtime;
GRANT ALL ON TABLE context.source_metadata_correction TO platform_app;
GRANT SELECT ON TABLE context.vw_source_metadata_correction_current TO context_import_writer;
GRANT SELECT ON TABLE context.vw_source_metadata_correction_current TO context_reader;
GRANT SELECT ON TABLE context.vw_source_metadata_correction_current TO platform_runtime;
GRANT ALL ON TABLE context.vw_source_metadata_correction_current TO platform_app;

-- <<< END source-metadata-correction-overlay block <<<

-- Read-back, printed before the transaction ends.
SELECT c.relname AS object, CASE c.relkind WHEN 'r' THEN 'table' WHEN 'v' THEN 'view' END AS kind,
       (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid = c.oid AND NOT t.tgisinternal) AS guard_triggers,
       pg_catalog.array_to_string(c.relacl, ' ') AS grants
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'context'
  AND c.relname IN ('source_metadata_correction', 'vw_source_metadata_correction_current')
ORDER BY c.relkind, c.relname;

COMMIT;
