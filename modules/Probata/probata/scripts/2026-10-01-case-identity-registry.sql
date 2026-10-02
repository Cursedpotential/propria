-- Case identity: one identity store in registry, edited from the Workbench Case page.
--
-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- Owner order 2026-10-01 07:56: "There needs to be a case identity page. Can update things. People,
-- profiles, aliases, phone numbers, view what's been extracted ... And that data flows throughout the
-- entire application." 07:57 design approved: Probata registry is the ONE identity store; a Case page in
-- the Workbench edits it; ingest, search, the legal desk and the toolkit read from it; edits version,
-- never overwrite. Owner 2026-09-24: "entities preserved, aliases preserved -- keep every raw form";
-- an alias is a row, never an overwrite.
--
-- What changes
--   registry.norm_identifier(text)   one identifier key, the same rule as the Case Bible catalog's
--                                    raw_duck.norm_phone (Consignatio/casebible/tools/msg_identity_20260924.sql)
--   registry.entity_alias            becomes THE per-person identifier table: names, phones, emails,
--                                    accounts. New columns: normalized (generated), status, period, basis,
--                                    change_reason, recorded_by, idempotency_key, supersedes_id. Append-only:
--                                    an edit is a new row that supersedes exactly one earlier row.
--   registry.entity_alias_current    the newest row of every alias chain (any status)
--   registry.court_case.presiding_judge  the judge on the case header
--   registry.person.short_name       the label the Case Bible tools use for a person ("Matt", "Katrina")
--   registry.identity_change         append-only before/after log for every case-header and person edit
--   registry.identifier_triage       append-only owner decisions on identifiers tied to nobody (dismiss / reopen)
--   registry.vw_case_identifier      read-only projection the Case Bible catalog reads over postgres_fdw
--   registry.vw_identifier_dismissed identifiers whose newest triage decision is "dismissed"
--
-- The snapshot (sql/bootstrap/schema_snapshot_20260907.sql) carries the same objects in final form:
-- the function and the two table definitions are edited in place, and the new objects sit in the
-- "case-identity registry" block at the end of the file.
--
-- Idempotent: IF NOT EXISTS / OR REPLACE / DROP CONSTRAINT IF EXISTS + ADD / repeatable GRANT.

BEGIN;

CREATE OR REPLACE FUNCTION registry.norm_identifier(p text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$
  select case
    when p is null or btrim(p) = '' then null
    when d ~ '^1[2-9][0-9]{9}$' then substr(d, 2)
    when d ~ '^[2-9][0-9]{9}$' then d
    when length(d) > 11 and right(d, 10) ~ '^[2-9][0-9]{9}$' and p ~ '^\s*[*#]' then right(d, 10)
    when d <> '' and p !~ '[A-Za-z@]' then d
    else lower(btrim(p))
  end
  from (select regexp_replace(p, '[^0-9]', '', 'g') as d) x
$$;

COMMENT ON FUNCTION registry.norm_identifier(text) IS 'The canonical key for an identifier, whatever the spelling: a North American number from +1 (810) 268-9630, 18102689630, 810.268.9630 or 8102689630 is 8102689630; dial prefixes (*67) are dropped; short codes stay digits; names, emails and handles are lowercased. Same rule as the Case Bible catalog raw_duck.norm_phone (2026-09-24).';

CREATE OR REPLACE FUNCTION registry.forbid_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'registry.% is append-only: % blocked (an edit is a new row that supersedes the old one)',
        TG_TABLE_NAME, TG_OP;
END
$$;

-- entity_alias: identifiers of every kind, versioned by supersession.
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_alias_kind_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_alias_kind_check CHECK ((alias_kind = ANY (ARRAY['name'::text, 'phone'::text, 'email'::text, 'account'::text, 'nickname'::text, 'legal'::text, 'maiden'::text, 'handle'::text, 'misspelling'::text, 'phonetic'::text, 'initials'::text, 'other'::text])));
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS normalized text GENERATED ALWAYS AS (registry.norm_identifier((alias_text)::text)) STORED;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS status text DEFAULT 'candidate'::text NOT NULL;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS period text;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS basis text;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS change_reason text;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS recorded_by text DEFAULT 'system'::text NOT NULL;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS idempotency_key text;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS supersedes_id uuid;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_status_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_status_check CHECK ((status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'retired'::text])));
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_text_bounds_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_text_bounds_check CHECK (((length(btrim((alias_text)::text)) > 0) AND (octet_length((alias_text)::text) <= 512) AND ((period IS NULL) OR (octet_length(period) <= 200)) AND ((basis IS NULL) OR (octet_length(basis) <= 4000)) AND ((change_reason IS NULL) OR (octet_length(change_reason) <= 4000))));
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_recorded_by_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_recorded_by_check CHECK ((length(btrim(recorded_by)) > 0));
-- the supersedes FK depends on the chain key, so it is dropped first on a re-run
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_supersedes_fk;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_chain_key;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_chain_key UNIQUE (id, entity_id, alias_text);
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_supersedes_key;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_supersedes_key UNIQUE (supersedes_id);
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_idempotency_key_key;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_idempotency_key_key UNIQUE (idempotency_key);
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_supersedes_fk FOREIGN KEY (supersedes_id, entity_id, alias_text) REFERENCES registry.entity_alias(id, entity_id, alias_text) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_alias_normalized ON registry.entity_alias USING btree (normalized);

COMMENT ON COLUMN registry.entity_alias.alias_text IS 'The identifier exactly as it was seen (raw spelling kept: 810-295-9302, Matthew Salem, Me).';
COMMENT ON COLUMN registry.entity_alias.normalized IS 'registry.norm_identifier(alias_text): the key every reader matches on.';
COMMENT ON COLUMN registry.entity_alias.status IS 'confirmed = the owner said so; candidate = the data suggests it, owner to confirm; retired = no longer believed (kept, never deleted).';
COMMENT ON COLUMN registry.entity_alias.period IS 'When the identifier was in use, as known (free text, e.g. 2021-2024).';
COMMENT ON COLUMN registry.entity_alias.basis IS 'Why we believe it.';
COMMENT ON COLUMN registry.entity_alias.supersedes_id IS 'The earlier row of the same person and raw spelling this row replaces. Rows are never updated or deleted; the newest row of a chain is the current one (registry.entity_alias_current).';

CREATE OR REPLACE TRIGGER entity_alias_append_only
    BEFORE UPDATE OR DELETE ON registry.entity_alias
    FOR EACH ROW EXECUTE FUNCTION registry.forbid_mutation();
CREATE OR REPLACE TRIGGER entity_alias_no_truncate
    BEFORE TRUNCATE ON registry.entity_alias
    FOR EACH STATEMENT EXECUTE FUNCTION registry.forbid_mutation();

CREATE OR REPLACE VIEW registry.entity_alias_current AS
 SELECT a.id, a.entity_id, a.alias_text, a.alias_kind, a.normalized, a.status, a.period, a.basis,
    a.change_reason, a.confidence, a.recorded_by, a.created_at, a.supersedes_id, a.provenance
   FROM registry.entity_alias a
  WHERE NOT EXISTS (SELECT 1 FROM registry.entity_alias s WHERE s.supersedes_id = a.id);

COMMENT ON VIEW registry.entity_alias_current IS 'Newest row of every alias chain, any status. Participant resolution matches normalized against rows whose status is not retired.';

-- person: the short label the Case Bible tools use.
ALTER TABLE registry.person ADD COLUMN IF NOT EXISTS short_name text;
ALTER TABLE registry.person DROP CONSTRAINT IF EXISTS person_short_name_check;
ALTER TABLE registry.person ADD CONSTRAINT person_short_name_check CHECK (((short_name IS NULL) OR ((length(btrim(short_name)) > 0) AND (length(short_name) <= 64))));
COMMENT ON COLUMN registry.person.short_name IS 'The label the Case Bible catalog tools filter on (raw_duck.msg_identity_20260924.person: Matt, Katrina).';

-- court_case: the presiding judge belongs to the case header (owner 2026-10-01: Judge Dawn M. Weier).
ALTER TABLE registry.court_case ADD COLUMN IF NOT EXISTS presiding_judge text;
ALTER TABLE registry.court_case DROP CONSTRAINT IF EXISTS court_case_presiding_judge_check;
ALTER TABLE registry.court_case ADD CONSTRAINT court_case_presiding_judge_check CHECK (((presiding_judge IS NULL) OR ((length(btrim(presiding_judge)) > 0) AND (length(presiding_judge) <= 200))));
COMMENT ON COLUMN registry.court_case.presiding_judge IS 'The judge assigned to the case, as captioned (e.g. Hon. Dawn M. Weier).';

-- >>> BEGIN case-identity registry block (Claude Code · Opus 5.5 · 2026-10-01) >>>

CREATE TABLE IF NOT EXISTS registry.identity_change (
    id uuid NOT NULL,
    subject_table text NOT NULL,
    subject_id uuid NOT NULL,
    before_state jsonb NOT NULL,
    after_state jsonb NOT NULL,
    change_reason text NOT NULL,
    recorded_by text NOT NULL,
    recorded_by_uid text NOT NULL,
    idempotency_key text NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT identity_change_pkey PRIMARY KEY (id),
    CONSTRAINT identity_change_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT identity_change_subject_table_check CHECK ((subject_table = ANY (ARRAY['registry.matter'::text, 'registry.court_case'::text, 'registry.person'::text, 'registry.entity'::text]))),
    CONSTRAINT identity_change_state_check CHECK (((jsonb_typeof(before_state) = 'object'::text) AND (jsonb_typeof(after_state) = 'object'::text))),
    CONSTRAINT identity_change_change_reason_check CHECK (((length(btrim(change_reason)) > 0) AND (octet_length(change_reason) <= 4000))),
    CONSTRAINT identity_change_recorded_by_check CHECK (((length(btrim(recorded_by)) > 0) AND (length(btrim(recorded_by_uid)) > 0))),
    CONSTRAINT identity_change_idempotency_key_check CHECK ((length(btrim(idempotency_key)) > 0))
);

COMMENT ON TABLE registry.identity_change IS 'Append-only version log for case-header and person edits from the Workbench Case page: the row before, the row after, who and why. The registry row holds the current value; this table holds every earlier one. Byline: Claude Code · Opus 5.5 · 2026-10-01.';

CREATE OR REPLACE TRIGGER identity_change_append_only
    BEFORE UPDATE OR DELETE ON registry.identity_change
    FOR EACH ROW EXECUTE FUNCTION registry.forbid_mutation();
CREATE OR REPLACE TRIGGER identity_change_no_truncate
    BEFORE TRUNCATE ON registry.identity_change
    FOR EACH STATEMENT EXECUTE FUNCTION registry.forbid_mutation();
CREATE INDEX IF NOT EXISTS identity_change_subject_idx ON registry.identity_change USING btree (subject_table, subject_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS registry.identifier_triage (
    id uuid NOT NULL,
    normalized text NOT NULL,
    raw_value text NOT NULL,
    decision text NOT NULL,
    basis text NOT NULL,
    recorded_by text NOT NULL,
    idempotency_key text NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT identifier_triage_pkey PRIMARY KEY (id),
    CONSTRAINT identifier_triage_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT identifier_triage_decision_check CHECK ((decision = ANY (ARRAY['dismissed'::text, 'reopened'::text]))),
    CONSTRAINT identifier_triage_text_check CHECK (((length(btrim(normalized)) > 0) AND (octet_length(normalized) <= 512) AND (length(btrim(raw_value)) > 0) AND (octet_length(raw_value) <= 512) AND (length(btrim(basis)) > 0) AND (octet_length(basis) <= 4000) AND (length(btrim(recorded_by)) > 0) AND (length(btrim(idempotency_key)) > 0)))
);

COMMENT ON TABLE registry.identifier_triage IS 'Append-only owner decisions on identifiers seen in the data but tied to no person: dismissed (not case-relevant) or reopened. The newest decision per normalized identifier stands. Assigning an identifier to a person is a registry.entity_alias row instead.';

CREATE OR REPLACE TRIGGER identifier_triage_append_only
    BEFORE UPDATE OR DELETE ON registry.identifier_triage
    FOR EACH ROW EXECUTE FUNCTION registry.forbid_mutation();
CREATE OR REPLACE TRIGGER identifier_triage_no_truncate
    BEFORE TRUNCATE ON registry.identifier_triage
    FOR EACH STATEMENT EXECUTE FUNCTION registry.forbid_mutation();
CREATE INDEX IF NOT EXISTS identifier_triage_normalized_idx ON registry.identifier_triage USING btree (normalized, recorded_at DESC);

CREATE OR REPLACE VIEW registry.vw_identifier_dismissed AS
 SELECT t.normalized, t.raw_value, t.basis, t.recorded_by, t.recorded_at
   FROM (SELECT DISTINCT ON (normalized) normalized, raw_value, decision, basis, recorded_by, recorded_at
           FROM registry.identifier_triage ORDER BY normalized, recorded_at DESC, id DESC) t
  WHERE t.decision = 'dismissed';

COMMENT ON VIEW registry.vw_identifier_dismissed IS 'Identifiers whose newest owner triage decision is dismissed; the unknowns queue leaves them out.';

CREATE OR REPLACE VIEW registry.vw_case_identifier AS
 SELECT a.entity_id,
    COALESCE(p.short_name, (e.display_name)::text, (e.canonical_name)::text) AS person,
    COALESCE((e.display_name)::text, (e.canonical_name)::text) AS display_name,
    p.role_in_case,
    a.normalized AS identifier,
    (a.alias_text)::text AS raw_value,
    CASE
        WHEN a.alias_kind = ANY (ARRAY['phone'::text, 'email'::text, 'account'::text]) THEN a.alias_kind
        WHEN a.alias_kind = 'handle'::text THEN 'account'::text
        WHEN a.alias_kind = 'other'::text OR a.alias_kind IS NULL THEN 'other'::text
        ELSE 'name'::text
    END AS kind,
    a.alias_kind,
    a.status,
    a.period,
    COALESCE(a.basis, ''::text) AS basis,
    a.created_at AS added_at,
    a.recorded_by,
    a.id AS alias_id
   FROM registry.entity_alias_current a
     JOIN registry.entity e ON e.id = a.entity_id
     JOIN registry.person p ON p.id = a.entity_id
  WHERE e.merged_into_id IS NULL;

COMMENT ON VIEW registry.vw_case_identifier IS 'Read-only projection of every person identifier (current version of each alias chain, retired rows included with their status). The Case Bible catalog reads it over postgres_fdw as raw_duck.msg_identity_20260924, so the catalog never holds a second editable copy.';

-- Grants. The Proffer starter connects as platform_runtime: it reads the case and its people,
-- inserts alias / change / triage rows, and updates only the documented editable columns.
GRANT SELECT, INSERT ON TABLE registry.identity_change TO platform_runtime;
GRANT SELECT, INSERT ON TABLE registry.identifier_triage TO platform_runtime;
GRANT SELECT ON TABLE registry.entity_alias_current TO platform_runtime;
GRANT SELECT ON TABLE registry.vw_identifier_dismissed TO platform_runtime;
GRANT SELECT ON TABLE registry.vw_case_identifier TO platform_runtime;
GRANT SELECT ON TABLE registry.person TO platform_runtime;
GRANT INSERT ON TABLE registry.person TO platform_runtime;
GRANT UPDATE (title, description, status, verification_state) ON TABLE registry.matter TO platform_runtime;
GRANT UPDATE (caption, docket_number, court_name, jurisdiction, case_type, presiding_judge, status, filed_on, closed_on, verification_state) ON TABLE registry.court_case TO platform_runtime;
GRANT SELECT ON TABLE working.call_log TO platform_runtime;
GRANT SELECT ON TABLE working.entity_resolution TO platform_runtime;
GRANT UPDATE (role_in_case, connection_to, relationship_type, short_name, is_minor, notes, verification_state) ON TABLE registry.person TO platform_runtime;
GRANT UPDATE (display_name, canonical_name, is_party) ON TABLE registry.entity TO platform_runtime;
GRANT SELECT ON TABLE working.message TO platform_runtime;
GRANT SELECT ON TABLE working.message_participant TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_message TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_message_participant TO platform_runtime;
GRANT ALL ON TABLE registry.identity_change TO platform_app;
GRANT ALL ON TABLE registry.identifier_triage TO platform_app;
GRANT ALL ON TABLE registry.entity_alias_current TO platform_app;
GRANT ALL ON TABLE registry.vw_identifier_dismissed TO platform_app;
GRANT ALL ON TABLE registry.vw_case_identifier TO platform_app;
GRANT SELECT ON TABLE registry.entity_alias_current TO platform_reader;
GRANT SELECT ON TABLE registry.vw_case_identifier TO platform_reader;
GRANT SELECT ON TABLE registry.vw_identifier_dismissed TO platform_reader;
GRANT SELECT ON TABLE registry.identity_change TO platform_reader;
GRANT EXECUTE ON FUNCTION registry.norm_identifier(text) TO PUBLIC;

-- The catalog's postgres_fdw login reads the projection and nothing else. The role itself is
-- created (with its password) by the apply step, outside this file: roles are cluster objects
-- and the password never enters git.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'registry_catalog_reader') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA registry TO registry_catalog_reader';
    EXECUTE 'GRANT SELECT ON TABLE registry.vw_case_identifier TO registry_catalog_reader';
    EXECUTE 'GRANT SELECT ON TABLE registry.vw_identifier_dismissed TO registry_catalog_reader';
  END IF;
END $$;

-- <<< END case-identity registry block <<<

-- Read-back, printed before the transaction ends.
SELECT c.relname AS object, CASE c.relkind WHEN 'r' THEN 'table' WHEN 'v' THEN 'view' END AS kind,
       (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid = c.oid AND NOT t.tgisinternal) AS guard_triggers
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'registry'
  AND c.relname IN ('entity_alias', 'entity_alias_current', 'identity_change', 'identifier_triage',
                    'vw_case_identifier', 'vw_identifier_dismissed')
ORDER BY c.relkind, c.relname;
SELECT column_name, data_type, is_generated FROM information_schema.columns
WHERE table_schema = 'registry' AND table_name = 'entity_alias' ORDER BY ordinal_position;
SELECT registry.norm_identifier('+1 (810) 268-9630') AS a, registry.norm_identifier('18102689630') AS b,
       registry.norm_identifier('810.268.9630') AS c, registry.norm_identifier('*678103533592') AS d,
       registry.norm_identifier('34428') AS e, registry.norm_identifier('Katrina Kinzel') AS f;

COMMIT;
