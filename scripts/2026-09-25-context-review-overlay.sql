-- Context review overlays: to / about / about-the-child / relevant, and the
-- hindsight-only foreshadowing flag, per normalized record (Review messages).
--
-- Byline: Claude Code · Opus 5.5 · 2026-09-25
-- Owner requirement 2026-09-25 19:13 (docs/pending-review/2026-09-25-metadata-context-review-entities.md §2).
--
-- ⚠ NOT APPLIED. The parent session applies it after review: dry-run inside a
--   transaction that ends in ROLLBACK first, then this file as written.
--
-- The block between the BEGIN/END markers is copied verbatim into
-- sql/bootstrap/schema_snapshot_20260907.sql (the database; D-142 §3, D-152),
-- so a rebuild from the snapshot and this apply script produce the same objects.
--
-- Idempotent: every statement is IF NOT EXISTS / OR REPLACE / a repeatable
-- GRANT or COMMENT. Re-running changes nothing. No existing object is altered.

BEGIN;

-- >>> BEGIN context-review-overlay block (Claude Code · Opus 5.5 · 2026-09-25) >>>
--
-- Corrections are OVERLAYS. The annotated record (context.normalized_record_identity,
-- which the Review message projects) is never written. Every row is actor-bound,
-- timestamped, receipt-addressed and append-only (context.forbid_mutation): a
-- correction is a new revision that supersedes exactly the newest one.
--
-- Two tables, two horizons:
--   context.record_context_review_revision  to / about / about the child / relevant.
--       As-lived-safe: it holds no hindsight-only fact and is visible wherever
--       the annotated record is visible.
--   context.record_foreshadowing_flag  the hindsight-only system flag. Separate
--       rows, horizon = 'hindsight' on every row; never a column on the record,
--       on the message table or on the authored spine. Readers other than the
--       engine reach it only through context.vw_record_foreshadowing_current,
--       which applies the spine's horizon PRE-filter (app.horizon set = as-lived /
--       ignorant => zero rows; unset or empty = hindsight), the same convention as
--       working.vw_spine_horizon. AGENTS.md "WHY THIS EXISTS": one leaked future
--       fact silently spoils the ignorant walk.

CREATE TABLE IF NOT EXISTS context.record_context_review_revision (
    review_ref uuid NOT NULL,
    normalized_record_id uuid NOT NULL,
    preview_handle text NOT NULL,
    matter_id uuid NOT NULL,
    court_case_id uuid NOT NULL,
    revision integer NOT NULL,
    supersedes_ref uuid,
    addressed_to jsonb DEFAULT '[]'::jsonb NOT NULL,
    about jsonb DEFAULT '[]'::jsonb NOT NULL,
    about_child text,
    relevant boolean,
    change_reason text NOT NULL,
    actor_subject_uid text NOT NULL,
    actor_username text NOT NULL,
    idempotency_key text NOT NULL,
    content_digest bytea NOT NULL,
    receipt_ref text NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT record_context_review_revision_pkey PRIMARY KEY (review_ref),
    CONSTRAINT record_context_review_revision_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT record_context_review_revision_receipt_ref_key UNIQUE (receipt_ref),
    CONSTRAINT record_context_review_revision_subject_revision_key UNIQUE (normalized_record_id, matter_id, revision),
    CONSTRAINT record_context_review_revision_scope_key UNIQUE (review_ref, normalized_record_id, matter_id),
    CONSTRAINT record_context_review_revision_revision_check CHECK ((revision >= 1)),
    CONSTRAINT record_context_review_revision_chain_check CHECK ((((revision = 1) AND (supersedes_ref IS NULL)) OR ((revision > 1) AND (supersedes_ref IS NOT NULL)))),
    CONSTRAINT record_context_review_revision_addressed_to_check CHECK (((jsonb_typeof(addressed_to) = 'array'::text) AND (jsonb_array_length(addressed_to) <= 32))),
    CONSTRAINT record_context_review_revision_about_check CHECK (((jsonb_typeof(about) = 'array'::text) AND (jsonb_array_length(about) <= 32))),
    CONSTRAINT record_context_review_revision_about_child_check CHECK (((about_child IS NULL) OR (about_child = ANY (ARRAY['yes'::text, 'no'::text, 'unsure'::text])))),
    CONSTRAINT record_context_review_revision_preview_handle_check CHECK ((preview_handle ~ '^[A-Za-z0-9_-]{32,128}$'::text)),
    CONSTRAINT record_context_review_revision_change_reason_check CHECK (((length(btrim(change_reason)) > 0) AND (octet_length(change_reason) <= 4000))),
    CONSTRAINT record_context_review_revision_actor_subject_uid_check CHECK ((length(btrim(actor_subject_uid)) > 0)),
    CONSTRAINT record_context_review_revision_actor_username_check CHECK ((length(btrim(actor_username)) > 0)),
    CONSTRAINT record_context_review_revision_idempotency_key_check CHECK ((length(btrim(idempotency_key)) > 0)),
    CONSTRAINT record_context_review_revision_content_digest_check CHECK ((octet_length(content_digest) = 32)),
    CONSTRAINT record_context_review_revision_receipt_ref_check CHECK ((length(btrim(receipt_ref)) > 0)),
    CONSTRAINT record_context_review_revision_record_fk FOREIGN KEY (normalized_record_id) REFERENCES context.normalized_record_identity(id) ON DELETE RESTRICT,
    CONSTRAINT record_context_review_revision_preview_fk FOREIGN KEY (preview_handle) REFERENCES context.proffer_preview_binding(preview_handle) ON DELETE RESTRICT,
    CONSTRAINT record_context_review_revision_matter_fk FOREIGN KEY (matter_id) REFERENCES registry.matter(id) ON DELETE RESTRICT,
    CONSTRAINT record_context_review_revision_court_case_fk FOREIGN KEY (court_case_id, matter_id) REFERENCES registry.court_case(id, matter_id) ON DELETE RESTRICT,
    CONSTRAINT record_context_review_revision_supersedes_fk FOREIGN KEY (supersedes_ref, normalized_record_id, matter_id) REFERENCES context.record_context_review_revision(review_ref, normalized_record_id, matter_id) ON DELETE RESTRICT
);

COMMENT ON TABLE context.record_context_review_revision IS 'Append-only operator context review of one normalized record (to, about, about the child, relevant). An overlay: the record is never written. As-lived-safe; hindsight-only facts live in context.record_foreshadowing_flag. Byline: Claude Code · Opus 5.5 · 2026-09-25.';
COMMENT ON COLUMN context.record_context_review_revision.addressed_to IS 'Who the record is TO: JSON array of {"label": text, "entity_id": uuid|null}; entity_id links registry.entity once entity extraction lands.';
COMMENT ON COLUMN context.record_context_review_revision.about IS 'Who the record is ABOUT: JSON array of {"label": text, "entity_id": uuid|null}.';
COMMENT ON COLUMN context.record_context_review_revision.about_child IS 'yes / no / unsure; NULL = not reviewed.';
COMMENT ON COLUMN context.record_context_review_revision.relevant IS 'Owner relevance mark; NULL = not reviewed.';
COMMENT ON COLUMN context.record_context_review_revision.preview_handle IS 'The Review view the correction was made in (audit); the subject is normalized_record_id.';

CREATE TABLE IF NOT EXISTS context.record_foreshadowing_flag (
    flag_ref uuid NOT NULL,
    normalized_record_id uuid NOT NULL,
    preview_handle text NOT NULL,
    matter_id uuid NOT NULL,
    court_case_id uuid NOT NULL,
    revision integer NOT NULL,
    supersedes_ref uuid,
    foreshadowing boolean NOT NULL,
    horizon text DEFAULT 'hindsight'::text NOT NULL,
    knowledge_time timestamp with time zone DEFAULT now() NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    change_reason text NOT NULL,
    actor_subject_uid text NOT NULL,
    actor_username text NOT NULL,
    idempotency_key text NOT NULL,
    content_digest bytea NOT NULL,
    receipt_ref text NOT NULL,
    CONSTRAINT record_foreshadowing_flag_pkey PRIMARY KEY (flag_ref),
    CONSTRAINT record_foreshadowing_flag_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT record_foreshadowing_flag_receipt_ref_key UNIQUE (receipt_ref),
    CONSTRAINT record_foreshadowing_flag_subject_revision_key UNIQUE (normalized_record_id, matter_id, revision),
    CONSTRAINT record_foreshadowing_flag_scope_key UNIQUE (flag_ref, normalized_record_id, matter_id),
    CONSTRAINT record_foreshadowing_flag_horizon_check CHECK ((horizon = 'hindsight'::text)),
    CONSTRAINT record_foreshadowing_flag_revision_check CHECK ((revision >= 1)),
    CONSTRAINT record_foreshadowing_flag_chain_check CHECK ((((revision = 1) AND (supersedes_ref IS NULL)) OR ((revision > 1) AND (supersedes_ref IS NOT NULL)))),
    CONSTRAINT record_foreshadowing_flag_note_check CHECK ((octet_length(note) <= 4000)),
    CONSTRAINT record_foreshadowing_flag_preview_handle_check CHECK ((preview_handle ~ '^[A-Za-z0-9_-]{32,128}$'::text)),
    CONSTRAINT record_foreshadowing_flag_change_reason_check CHECK (((length(btrim(change_reason)) > 0) AND (octet_length(change_reason) <= 4000))),
    CONSTRAINT record_foreshadowing_flag_actor_subject_uid_check CHECK ((length(btrim(actor_subject_uid)) > 0)),
    CONSTRAINT record_foreshadowing_flag_actor_username_check CHECK ((length(btrim(actor_username)) > 0)),
    CONSTRAINT record_foreshadowing_flag_idempotency_key_check CHECK ((length(btrim(idempotency_key)) > 0)),
    CONSTRAINT record_foreshadowing_flag_content_digest_check CHECK ((octet_length(content_digest) = 32)),
    CONSTRAINT record_foreshadowing_flag_receipt_ref_check CHECK ((length(btrim(receipt_ref)) > 0)),
    CONSTRAINT record_foreshadowing_flag_record_fk FOREIGN KEY (normalized_record_id) REFERENCES context.normalized_record_identity(id) ON DELETE RESTRICT,
    CONSTRAINT record_foreshadowing_flag_preview_fk FOREIGN KEY (preview_handle) REFERENCES context.proffer_preview_binding(preview_handle) ON DELETE RESTRICT,
    CONSTRAINT record_foreshadowing_flag_matter_fk FOREIGN KEY (matter_id) REFERENCES registry.matter(id) ON DELETE RESTRICT,
    CONSTRAINT record_foreshadowing_flag_court_case_fk FOREIGN KEY (court_case_id, matter_id) REFERENCES registry.court_case(id, matter_id) ON DELETE RESTRICT,
    CONSTRAINT record_foreshadowing_flag_supersedes_fk FOREIGN KEY (supersedes_ref, normalized_record_id, matter_id) REFERENCES context.record_foreshadowing_flag(flag_ref, normalized_record_id, matter_id) ON DELETE RESTRICT
);

COMMENT ON TABLE context.record_foreshadowing_flag IS 'HINDSIGHT-ONLY system flag: the owner now knows, in hindsight, this record was significant. Separate append-only rows with horizon = hindsight; never a column on the record, the message table or the authored spine. Non-engine readers use context.vw_record_foreshadowing_current, which is horizon PRE-filtered. Byline: Claude Code · Opus 5.5 · 2026-09-25.';
COMMENT ON COLUMN context.record_foreshadowing_flag.horizon IS 'Explicit horizon marker; always hindsight. The as-lived / ignorant read path must never see these rows.';
COMMENT ON COLUMN context.record_foreshadowing_flag.knowledge_time IS 'When the owner flagged it (row-write audit time). Never a horizon predicate: visibility is decided by horizon, not by this clock.';
COMMENT ON COLUMN context.record_foreshadowing_flag.foreshadowing IS 'true = flagged as foreshadowing; false = a later revision cleared the flag.';

CREATE OR REPLACE TRIGGER record_context_review_revision_append_only
    BEFORE UPDATE OR DELETE ON context.record_context_review_revision
    FOR EACH ROW EXECUTE FUNCTION context.forbid_mutation();
CREATE OR REPLACE TRIGGER record_context_review_revision_no_truncate
    BEFORE TRUNCATE ON context.record_context_review_revision
    FOR EACH STATEMENT EXECUTE FUNCTION context.forbid_mutation();
CREATE OR REPLACE TRIGGER record_foreshadowing_flag_append_only
    BEFORE UPDATE OR DELETE ON context.record_foreshadowing_flag
    FOR EACH ROW EXECUTE FUNCTION context.forbid_mutation();
CREATE OR REPLACE TRIGGER record_foreshadowing_flag_no_truncate
    BEFORE TRUNCATE ON context.record_foreshadowing_flag
    FOR EACH STATEMENT EXECUTE FUNCTION context.forbid_mutation();

CREATE OR REPLACE VIEW context.vw_record_context_review_current AS
 SELECT DISTINCT ON (r.normalized_record_id, r.matter_id)
    r.normalized_record_id,
    r.matter_id,
    r.court_case_id,
    r.review_ref,
    r.revision,
    r.addressed_to,
    r.about,
    r.about_child,
    r.relevant,
    r.change_reason,
    r.actor_username,
    r.receipt_ref,
    r.recorded_at
   FROM context.record_context_review_revision r
  ORDER BY r.normalized_record_id, r.matter_id, r.revision DESC;

COMMENT ON VIEW context.vw_record_context_review_current IS 'Newest context-review revision per record and matter. As-lived-safe by construction: it never reads context.record_foreshadowing_flag.';

CREATE OR REPLACE VIEW context.vw_record_foreshadowing_current AS
 SELECT DISTINCT ON (f.normalized_record_id, f.matter_id)
    f.normalized_record_id,
    f.matter_id,
    f.court_case_id,
    f.flag_ref,
    f.revision,
    f.foreshadowing,
    f.horizon,
    f.knowledge_time,
    f.note,
    f.actor_username,
    f.receipt_ref
   FROM context.record_foreshadowing_flag f
  WHERE f.horizon = 'hindsight'::text
    AND NULLIF(current_setting('app.horizon'::text, true), ''::text) IS NULL
  ORDER BY f.normalized_record_id, f.matter_id, f.revision DESC;

COMMENT ON VIEW context.vw_record_foreshadowing_current IS 'Newest foreshadowing revision per record and matter, horizon PRE-filtered: with app.horizon set (as-lived / ignorant session) it returns zero rows; unset or empty = hindsight. Same convention as working.vw_spine_horizon. The filter runs before DISTINCT ON and before any caller LIMIT.';

CREATE OR REPLACE VIEW context.vw_record_context_review_horizon AS
 SELECT COALESCE(r.normalized_record_id, f.normalized_record_id) AS normalized_record_id,
    COALESCE(r.matter_id, f.matter_id) AS matter_id,
    r.review_ref,
    r.revision AS review_revision,
    r.addressed_to,
    r.about,
    r.about_child,
    r.relevant,
    r.recorded_at AS reviewed_at,
    f.flag_ref AS foreshadowing_ref,
    f.foreshadowing,
    f.horizon AS foreshadowing_horizon,
    f.knowledge_time AS foreshadowing_flagged_at
   FROM (context.vw_record_context_review_current r
     FULL JOIN context.vw_record_foreshadowing_current f ON (((f.normalized_record_id = r.normalized_record_id) AND (f.matter_id = r.matter_id))));

COMMENT ON VIEW context.vw_record_context_review_horizon IS 'One row per reviewed record for agents and walks. The foreshadowing columns come only from the horizon-filtered view, so an as-lived session (app.horizon set) sees them NULL on every row.';

GRANT SELECT,INSERT ON TABLE context.record_context_review_revision TO context_import_writer;
GRANT SELECT ON TABLE context.record_context_review_revision TO context_reader;
GRANT SELECT ON TABLE context.record_context_review_revision TO platform_runtime;
GRANT ALL ON TABLE context.record_context_review_revision TO platform_app;
-- The base foreshadowing table: the engine (writer, serving the owner's hindsight
-- Review surface) and the owner role only. Every other reader goes through the
-- horizon-filtered views below.
GRANT SELECT,INSERT ON TABLE context.record_foreshadowing_flag TO context_import_writer;
GRANT ALL ON TABLE context.record_foreshadowing_flag TO platform_app;
GRANT SELECT ON TABLE context.vw_record_context_review_current TO context_import_writer;
GRANT SELECT ON TABLE context.vw_record_context_review_current TO context_reader;
GRANT SELECT ON TABLE context.vw_record_context_review_current TO platform_runtime;
GRANT ALL ON TABLE context.vw_record_context_review_current TO platform_app;
GRANT SELECT ON TABLE context.vw_record_foreshadowing_current TO context_import_writer;
GRANT SELECT ON TABLE context.vw_record_foreshadowing_current TO context_reader;
GRANT SELECT ON TABLE context.vw_record_foreshadowing_current TO platform_runtime;
GRANT ALL ON TABLE context.vw_record_foreshadowing_current TO platform_app;
GRANT SELECT ON TABLE context.vw_record_context_review_horizon TO context_import_writer;
GRANT SELECT ON TABLE context.vw_record_context_review_horizon TO context_reader;
GRANT SELECT ON TABLE context.vw_record_context_review_horizon TO platform_runtime;
GRANT ALL ON TABLE context.vw_record_context_review_horizon TO platform_app;

-- <<< END context-review-overlay block <<<

-- Read-back, printed before the transaction ends.
SELECT c.relname AS object, CASE c.relkind WHEN 'r' THEN 'table' WHEN 'v' THEN 'view' END AS kind,
       (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid = c.oid AND NOT t.tgisinternal) AS guard_triggers,
       pg_catalog.array_to_string(c.relacl, ' ') AS grants
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'context'
  AND c.relname IN ('record_context_review_revision', 'record_foreshadowing_flag',
                    'vw_record_context_review_current', 'vw_record_foreshadowing_current',
                    'vw_record_context_review_horizon')
ORDER BY c.relkind, c.relname;

COMMIT;
