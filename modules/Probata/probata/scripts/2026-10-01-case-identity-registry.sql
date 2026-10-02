-- Case identity: one identity store in registry, edited from the Workbench Case page.
--
-- Byline: Claude Code · Opus 5.5 · 2026-10-01; reworked 2026-10-02 to the owner's 02:12 EDT decision.
-- Owner order 2026-10-01 07:56/07:57: registry is the ONE identity store; a Case page in the Workbench
-- edits it; ingest, search, the legal desk and the toolkit read it.
-- Owner 2026-10-02 02:06: nothing is immutable until it is promoted to evidence; audit trails are always
-- locked. 02:12: registry.entity_alias is a plain editable table: a wrong identifier is fixed in place
-- with UPDATE, a bad row is removed with DELETE, and every add, edit and delete writes one
-- registry.identity_change row (before, after, who, why). That log stays append-only.
--
-- What this file leaves in the database
--   registry.norm_identifier(text)   one identifier key, the same rule as the Case Bible catalog's
--                                    raw_duck.norm_phone (Consignatio/casebible/tools/msg_identity_20260924.sql)
--   registry.forbid_mutation()       the append-only guard of the two audit tables
--   registry.entity_alias            THE per-person identifier table: names, phones, emails, accounts, every raw
--                                    spelling its own row; columns normalized (generated), status, period, basis,
--                                    recorded_by. Editable (UPDATE, DELETE); every change is logged.
--   registry.entity_alias_current    plain view over entity_alias (the name the engine readers use)
--   registry.person.short_name       the label the Case Bible tools use for a person ("Matt", "Katrina")
--   registry.court_case.presiding_judge  the judge on the case header
--   registry.identity_change         append-only before/after log of every identifier, header and person change
--   registry.identifier_triage       append-only owner decisions on identifiers tied to nobody (dismiss / reopen)
--   registry.vw_case_identifier      read-only projection the Case Bible catalog reads over postgres_fdw
--   registry.vw_identifier_dismissed identifiers whose newest triage decision is "dismissed"
--
-- The snapshot (sql/bootstrap/schema_snapshot_20260907.sql) carries the same objects in final form.
-- Idempotent and convergent: on a database holding the 2026-10-01 version (supersede chain,
-- append-only entity_alias) it removes that design; run again it changes nothing.

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
    RAISE EXCEPTION 'registry.% is an audit trail: % blocked (append a new row instead)',
        TG_TABLE_NAME, TG_OP;
END
$$;

-- The two views depend on entity_alias columns; they are rebuilt below.
DROP VIEW IF EXISTS registry.vw_case_identifier;
DROP VIEW IF EXISTS registry.entity_alias_current;

-- entity_alias: identifiers of every kind, edited in place.
DROP TRIGGER IF EXISTS entity_alias_append_only ON registry.entity_alias;
DROP TRIGGER IF EXISTS entity_alias_no_truncate ON registry.entity_alias;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_supersedes_fk;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_chain_key;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_supersedes_key;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_idempotency_key_key;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_text_bounds_check;
ALTER TABLE registry.entity_alias DROP COLUMN IF EXISTS supersedes_id;
ALTER TABLE registry.entity_alias DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE registry.entity_alias DROP COLUMN IF EXISTS change_reason;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_alias_kind_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_alias_kind_check CHECK ((alias_kind = ANY (ARRAY['name'::text, 'phone'::text, 'email'::text, 'account'::text, 'nickname'::text, 'legal'::text, 'maiden'::text, 'handle'::text, 'misspelling'::text, 'phonetic'::text, 'initials'::text, 'other'::text])));
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS normalized text GENERATED ALWAYS AS (registry.norm_identifier((alias_text)::text)) STORED;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS status text DEFAULT 'candidate'::text NOT NULL;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS period text;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS basis text;
ALTER TABLE registry.entity_alias ADD COLUMN IF NOT EXISTS recorded_by text DEFAULT 'system'::text NOT NULL;
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_status_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_status_check CHECK ((status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'retired'::text])));
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_text_bounds_check CHECK (((length(btrim((alias_text)::text)) > 0) AND (octet_length((alias_text)::text) <= 512) AND ((period IS NULL) OR (octet_length(period) <= 200)) AND ((basis IS NULL) OR (octet_length(basis) <= 4000))));
ALTER TABLE registry.entity_alias DROP CONSTRAINT IF EXISTS entity_alias_recorded_by_check;
ALTER TABLE registry.entity_alias ADD CONSTRAINT entity_alias_recorded_by_check CHECK ((length(btrim(recorded_by)) > 0));
CREATE INDEX IF NOT EXISTS idx_alias_normalized ON registry.entity_alias USING btree (normalized);

COMMENT ON COLUMN registry.entity_alias.alias_text IS 'The identifier exactly as it was seen (raw spelling kept: 810-295-9302, Matthew Salem, Me). Each spelling is its own row.';
COMMENT ON COLUMN registry.entity_alias.normalized IS 'registry.norm_identifier(alias_text): the key every reader matches on.';
COMMENT ON COLUMN registry.entity_alias.status IS 'confirmed = the owner said so; candidate = the data suggests it, owner to confirm; retired = no longer believed.';
COMMENT ON COLUMN registry.entity_alias.period IS 'When the identifier was in use, as known (free text, e.g. 2021-2024).';
COMMENT ON COLUMN registry.entity_alias.basis IS 'Why we believe it.';
COMMENT ON COLUMN registry.entity_alias.recorded_by IS 'Who last wrote this row. Every add, edit and delete is logged in registry.identity_change with before, after, who and why.';

CREATE VIEW registry.entity_alias_current AS
 SELECT a.id, a.entity_id, a.alias_text, a.alias_kind, a.normalized, a.status, a.period, a.basis,
    a.confidence, a.recorded_by, a.created_at, a.provenance
   FROM registry.entity_alias a;

COMMENT ON VIEW registry.entity_alias_current IS 'Every identifier row (entity_alias is edited in place). Participant resolution matches normalized against rows whose status is confirmed (or not retired).';

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

-- >>> BEGIN case-identity registry block (Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02) >>>

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
    CONSTRAINT identity_change_state_check CHECK (((jsonb_typeof(before_state) = 'object'::text) AND (jsonb_typeof(after_state) = 'object'::text))),
    CONSTRAINT identity_change_change_reason_check CHECK (((length(btrim(change_reason)) > 0) AND (octet_length(change_reason) <= 4000))),
    CONSTRAINT identity_change_recorded_by_check CHECK (((length(btrim(recorded_by)) > 0) AND (length(btrim(recorded_by_uid)) > 0))),
    CONSTRAINT identity_change_idempotency_key_check CHECK ((length(btrim(idempotency_key)) > 0))
);
ALTER TABLE registry.identity_change DROP CONSTRAINT IF EXISTS identity_change_subject_table_check;
ALTER TABLE registry.identity_change ADD CONSTRAINT identity_change_subject_table_check CHECK ((subject_table = ANY (ARRAY['registry.matter'::text, 'registry.court_case'::text, 'registry.person'::text, 'registry.entity'::text, 'registry.entity_alias'::text])));

COMMENT ON TABLE registry.identity_change IS 'Append-only audit trail of every change made from the Workbench Case page (and the seed): an identifier added, edited or deleted, a case-header or person edit. before_state = {} for an add, after_state = {} for a delete. The registry row holds the current value; this table holds every earlier one. Byline: Claude Code · Opus 5.5 · 2026-10-01/02.';

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

CREATE VIEW registry.vw_case_identifier AS
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
   FROM registry.entity_alias a
     JOIN registry.entity e ON e.id = a.entity_id
     JOIN registry.person p ON p.id = a.entity_id
  WHERE e.merged_into_id IS NULL;

COMMENT ON VIEW registry.vw_case_identifier IS 'Read-only projection of every person identifier, retired rows included with their status. The Case Bible catalog reads it over postgres_fdw as raw_duck.msg_identity_20260924, so the catalog never holds a second editable copy.';

-- Grants. The Proffer starter connects as platform_runtime: it reads the case and its people,
-- writes identifiers (insert, update, delete), appends change and triage rows, and updates only the
-- documented editable columns of the case header and the person.
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE registry.entity_alias TO platform_runtime;
GRANT SELECT, INSERT ON TABLE registry.identity_change TO platform_runtime;
GRANT SELECT, INSERT ON TABLE registry.identifier_triage TO platform_runtime;
GRANT SELECT ON TABLE registry.entity_alias_current TO platform_runtime;
GRANT SELECT ON TABLE registry.vw_identifier_dismissed TO platform_runtime;
GRANT SELECT ON TABLE registry.vw_case_identifier TO platform_runtime;
GRANT SELECT, INSERT ON TABLE registry.person TO platform_runtime;
GRANT UPDATE (title, description, status, verification_state) ON TABLE registry.matter TO platform_runtime;
GRANT UPDATE (caption, docket_number, court_name, jurisdiction, case_type, presiding_judge, status, filed_on, closed_on, verification_state) ON TABLE registry.court_case TO platform_runtime;
GRANT UPDATE (role_in_case, connection_to, relationship_type, short_name, is_minor, notes, verification_state) ON TABLE registry.person TO platform_runtime;
GRANT UPDATE (display_name, canonical_name, is_party) ON TABLE registry.entity TO platform_runtime;
GRANT SELECT ON TABLE working.message TO platform_runtime;
GRANT SELECT ON TABLE working.message_participant TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_message TO platform_runtime;
GRANT SELECT ON TABLE working.third_party_message_participant TO platform_runtime;
GRANT SELECT ON TABLE working.call_log TO platform_runtime;
GRANT SELECT ON TABLE working.entity_resolution TO platform_runtime;
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

-- The catalog's postgres_fdw login reads the projection and nothing else. The role itself (with its
-- password) is created outside this file: roles are cluster objects and the password never enters git.
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
SELECT privilege_type FROM information_schema.role_table_grants
WHERE grantee = 'platform_runtime' AND table_schema = 'registry' AND table_name = 'entity_alias' ORDER BY 1;
SELECT registry.norm_identifier('+1 (810) 268-9630') AS a, registry.norm_identifier('18102689630') AS b,
       registry.norm_identifier('810.268.9630') AS c, registry.norm_identifier('*678103533592') AS d,
       registry.norm_identifier('34428') AS e, registry.norm_identifier('Katrina Kinzel') AS f;

COMMIT;
