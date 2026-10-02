-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Go-live case identity (OD-05) and the TEST purge, in one transaction, on the live `platform` database.
-- Owner, 2026-10-01, in the session that runs this: 07:17 "Mint now, I approve"; 07:42 "I don't really want test runs.
-- I want you using real data with real names, and if it fucking works, we fucking leave it."; 08:06 "Yes, delete it"
-- (the DEV placeholder identity and every TEST-run row). Approval payload and facts:
-- sql/bootstrap/case_registry_live_identity_20261001.json (commit 5643178cf0beba05a11bd357544e0c32f170f840,
-- sha256 b39561e9..., 4,532 bytes).
--
-- Purge scope, read 2026-10-02 01:55 EDT: every non-empty table ops.reset_test_data truncates holds TEST-run rows
-- only (36 context.source_version rows, all on the DEV matter deadbeef..., 427 activity executions, 5,389 normalized
-- record identities; the 2 evidence.* rows are legacy Python test rows). No table outside the six reset schemas has
-- a foreign key into them, so the TRUNCATE ... CASCADE reaches nothing in registry, reference, canon or ops.
--
-- Order: (0) remove the 18 append-only triggers on data tables (owner 2026-10-02 02:06 EDT: "Nothing's immutable
-- until it goes to evidence"; audit trails keep theirs) and stop the raw-format registration from adding one;
-- (1) save the format catalog; (2) ops.reset_test_data('RESET'), the one sanctioned purge; (3) restore the catalog;
-- (4) remove the DEV placeholder identity, which the reset re-seeds; (5) widen the person CHECKs to the caption's
-- terms; (6) mint the real rows and the owner receipt; (7) read everything back.
-- Run as the database superuser:
--   docker exec -i <probata-db> psql -U ai -d platform -X [-v do_commit=1] < seed_live_case_registry_20261001.sql
-- Ends with ROLLBACK unless psql variable do_commit=1.

\set ON_ERROR_STOP on
begin;

-- (0) data tables carry no append-only guard until evidence; audit trails keep theirs
drop trigger raw_subtype_append_only on context.raw_xml;
drop trigger raw_subtype_append_only on context.raw_ndjson;
drop trigger raw_subtype_append_only on context.raw_sms_export_xml;
drop trigger raw_subtype_append_only on context.raw_callsbackuprestore_xml;
drop trigger record_context_review_revision_append_only on context.record_context_review_revision;
drop trigger record_context_review_revision_no_truncate on context.record_context_review_revision;
drop trigger record_foreshadowing_flag_append_only on context.record_foreshadowing_flag;
drop trigger record_foreshadowing_flag_no_truncate on context.record_foreshadowing_flag;
drop trigger source_metadata_correction_append_only on context.source_metadata_correction;
drop trigger source_metadata_correction_no_truncate on context.source_metadata_correction;
drop trigger filenode_immutable on raw.file_node;
drop trigger entity_alias_append_only on registry.entity_alias;
drop trigger entity_alias_no_truncate on registry.entity_alias;
drop trigger discrev_immutable on analysis.discovery_request_revision;
drop trigger export_append_only on analysis.export;
drop trigger finding_version_immutable on analysis.finding_version;
drop trigger redaction_append_only on analysis.redaction;
drop trigger taskrev_immutable on analysis.task_revision;

-- the raw-format registration no longer adds one (same body as the snapshot)
CREATE OR REPLACE FUNCTION context.register_raw_format_subtype(p_format_id text) RETURNS regclass
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'context'
    AS $_$
DECLARE
    v_table_name TEXT;
    v_relation REGCLASS;
    v_relation_kind "char";
    v_raw_record_attnum SMALLINT;
    v_raw_identity_id_attnum SMALLINT;
    v_native_fields_attnum SMALLINT;
    v_native_metadata_attnum SMALLINT;
BEGIN
    IF p_format_id !~ '^[a-z][a-z0-9_]{0,58}$' THEN
        RAISE EXCEPTION 'invalid raw format id %', p_format_id;
    END IF;

    v_table_name := 'raw_' || p_format_id;
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS context.%I (
            raw_record_id UUID PRIMARY KEY
                REFERENCES context.raw_record_identity(id) ON DELETE RESTRICT,
            native_fields JSONB NOT NULL DEFAULT ''{}''::jsonb
                CHECK (jsonb_typeof(native_fields) = ''object''),
            native_metadata JSONB NOT NULL DEFAULT ''{}''::jsonb
                CHECK (jsonb_typeof(native_metadata) = ''object''),
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )',
        v_table_name
    );
    v_relation := to_regclass(format('context.%I', v_table_name));

    SELECT relkind INTO v_relation_kind
    FROM pg_class
    WHERE oid = v_relation;
    IF v_relation_kind IS DISTINCT FROM 'r'::"char" THEN
        RAISE EXCEPTION 'raw subtype % must be an ordinary table, found relation kind %',
            v_relation::TEXT, v_relation_kind;
    END IF;

    SELECT attnum INTO v_raw_identity_id_attnum
    FROM pg_attribute
    WHERE attrelid = 'context.raw_record_identity'::regclass
      AND attname = 'id'
      AND atttypid = 'uuid'::regtype
      AND attnotnull
      AND NOT attisdropped;

    SELECT attnum INTO v_raw_record_attnum
    FROM pg_attribute
    WHERE attrelid = v_relation
      AND attname = 'raw_record_id'
      AND atttypid = 'uuid'::regtype
      AND attnotnull
      AND NOT attisdropped;

    IF v_raw_record_attnum IS NULL OR v_raw_identity_id_attnum IS NULL THEN
        RAISE EXCEPTION 'raw subtype % must have a NOT NULL UUID raw_record_id key column',
            v_relation::TEXT;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = v_relation
          AND contype = 'p'
          AND conkey = ARRAY[v_raw_record_attnum]::SMALLINT[]
    ) THEN
        RAISE EXCEPTION 'raw subtype % must have raw_record_id as its exact primary key',
            v_relation::TEXT;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = v_relation
          AND contype = 'f'
          AND confrelid = 'context.raw_record_identity'::regclass
          AND conkey = ARRAY[v_raw_record_attnum]::SMALLINT[]
          AND confkey = ARRAY[v_raw_identity_id_attnum]::SMALLINT[]
          AND confdeltype = 'r'
    ) THEN
        RAISE EXCEPTION 'raw subtype % must have an exact raw_record_id FK to context.raw_record_identity(id) ON DELETE RESTRICT',
            v_relation::TEXT;
    END IF;

    SELECT attnum INTO v_native_fields_attnum
    FROM pg_attribute
    WHERE attrelid = v_relation
      AND attname = 'native_fields'
      AND atttypid = 'jsonb'::regtype
      AND attnotnull
      AND NOT attisdropped;
    SELECT attnum INTO v_native_metadata_attnum
    FROM pg_attribute
    WHERE attrelid = v_relation
      AND attname = 'native_metadata'
      AND atttypid = 'jsonb'::regtype
      AND attnotnull
      AND NOT attisdropped;
    IF v_native_fields_attnum IS NULL OR v_native_metadata_attnum IS NULL THEN
        RAISE EXCEPTION 'raw subtype % must have NOT NULL JSONB native_fields and native_metadata columns',
            v_relation::TEXT;
    END IF;

    INSERT INTO context.raw_format_registry (format_id, subtype_relation)
    VALUES (p_format_id, v_relation)
    ON CONFLICT (format_id) DO NOTHING;
    IF (SELECT subtype_relation FROM context.raw_format_registry WHERE format_id = p_format_id)
       IS DISTINCT FROM v_relation THEN
        RAISE EXCEPTION 'raw format % is already registered to a different subtype relation', p_format_id;
    END IF;

    -- raw_subtype_append_only REMOVED 2026-10-02 (Claude Code · Opus 5.5). Owner, 02:06 EDT: "Nothing's
    -- immutable until it goes to evidence, which is completely separate from context."
    -- raw_subtype_open_generation_gate REMOVED 2026-09-20 (Claude Code · Fable 5.1):
    -- it called context.guard_raw_subtype_insert(), one of the 27 custody guard
    -- functions the D-152 rebuild (2026-09-07, "get the database rebuilt without it")
    -- left out of this snapshot. The leftover made every first registration of a raw
    -- format fail ("function ... does not exist"), so no raw generation could ever
    -- be persisted. The gate returns with the custody guards at promotion time.
    --
    -- Grants ADDED 2026-09-20: this SECURITY DEFINER function creates the table as its
    -- owner, so the engine role (member of context_import_writer) could register a
    -- subtype and then not insert into it ("permission denied for table raw_ndjson").
    -- Same grants as the sibling context.raw_record_identity.
    EXECUTE format('GRANT SELECT, INSERT ON TABLE context.%I TO context_import_writer', v_table_name);
    EXECUTE format('GRANT SELECT ON TABLE context.%I TO context_reader', v_table_name);
    EXECUTE format('GRANT ALL ON TABLE context.%I TO platform_app', v_table_name);
    RETURN v_relation;
END;
$_$;

-- (1) format catalog: reference data, not test traffic
create temp table keep_raw_format_registry as select * from context.raw_format_registry;

-- (2) the sanctioned purge
select * from ops.reset_test_data('RESET');

-- (3) restore the catalog
insert into context.raw_format_registry select * from keep_raw_format_registry;

-- (4) retire the DEV placeholder identity (re-seeded by the reset's self-heal)
delete from analysis.case_registry_import_receipt where approved_by = 'dev-mode-placeholder';
delete from analysis.matter_knowledge_partition where matter_id = 'deadbeef-dead-beef-dead-beefdeadbeef';
delete from registry.court_case where id = 'cafebabe-cafe-babe-cafe-babecafebabe';
delete from registry.matter where id = 'deadbeef-dead-beef-dead-beefdeadbeef';

-- (5) the caption is "Matthew S. Salem v Katrina Kinzel": plaintiff and defendant; she is the child's other parent
alter table registry.person drop constraint person_connection_to_check;
alter table registry.person add constraint person_connection_to_check check (connection_to = any (array[
  'petitioner','respondent','plaintiff','defendant','child','mutual','third_party','unknown']));
alter table registry.person drop constraint person_role_in_case_check;
alter table registry.person add constraint person_role_in_case_check check (role_in_case = any (array[
  'user','partner','co_parent','child','witness','evaluator','attorney','third_party','neutral','unknown']));

-- (6) the real case
insert into registry.matter (id, title, description, status, created_by, verification_state)
values ('01a0f751-e07b-75cc-9ad5-63ad9449a8ba', 'Salem v. Kinzel (custody)',
        'Matthew S. Salem v Katrina Kinzel, 2025-53985-DC, Genesee County Circuit Court, Family Division.',
        'active', 'owner', 'confirmed');
insert into registry.court_case (id, matter_id, caption, docket_number, court_name, jurisdiction, case_type, status,
                                 filed_on, is_primary, created_by, verification_state, presiding_judge)
values ('01a0f751-e07b-76a1-a738-eb3e3aa3e68c', '01a0f751-e07b-75cc-9ad5-63ad9449a8ba',
        'Matthew S. Salem v Katrina Kinzel', '2025-53985-DC',
        'Genesee County Circuit Court, Family Division (7th Judicial Circuit)', 'Genesee County, Michigan',
        'custody', 'active', null, true, 'owner', 'confirmed', 'Dawn M. Weier');

insert into registry.entity (id, entity_type, display_name, canonical_name, normalized_name, is_party, data_tier,
                             requires_human_review, review_status, safe_for_legal_use)
values ('01a0f751-e07b-76b6-afcb-63acfbba373e', 'person', 'Matthew S. Salem', 'Matthew S. Salem', 'matthew s. salem',
        true, 'extracted', false, 'approved', true),
       ('01a0f751-e07b-76c7-8c0f-65692ad656b8', 'person', 'Katrina Kinzel', 'Katrina Kinzel', 'katrina kinzel',
        true, 'extracted', false, 'approved', true);
insert into registry.person (id, connection_to, role_in_case, is_minor, verification_state, identification_signal, notes)
values ('01a0f751-e07b-76b6-afcb-63acfbba373e', 'plaintiff', 'user', false, 'confirmed', 'court caption',
        'The owner. Plaintiff in Matthew S. Salem v Katrina Kinzel, 2025-53985-DC.'),
       ('01a0f751-e07b-76c7-8c0f-65692ad656b8', 'defendant', 'co_parent', false, 'confirmed', 'court caption',
        'Defendant in Matthew S. Salem v Katrina Kinzel, 2025-53985-DC; the child''s other parent.');

insert into analysis.matter_knowledge_partition (partition_key, matter_id, default_court_case_id, created_by)
values ('live', '01a0f751-e07b-75cc-9ad5-63ad9449a8ba', '01a0f751-e07b-76a1-a738-eb3e3aa3e68c', 'owner');

insert into analysis.case_registry_import_receipt (
  manifest_sha256, source_migration_uri, source_migration_sha256, source_git_commit, payload_schema_version,
  payload_byte_length, canonical_payload_sha256, api_payload_sha256, source_observed_at, matter_id, court_case_id,
  partition_key, approved_by, approved_on, imported_by)
values (
  decode('b39561e99a111c55f86d258e97cdbffec3fa0f84efb1d97fb548aa301b8fe544', 'hex'),
  'sql/bootstrap/case_registry_live_identity_20261001.json',
  decode('b39561e99a111c55f86d258e97cdbffec3fa0f84efb1d97fb548aa301b8fe544', 'hex'),
  '5643178cf0beba05a11bd357544e0c32f170f840',
  'live-case-registry-identity-v1',
  4532,
  decode('e51c7fcfcc59d422255e173cbad9455ff010e8e052807190b7872b49cfaf113a', 'hex'),
  decode('b39561e99a111c55f86d258e97cdbffec3fa0f84efb1d97fb548aa301b8fe544', 'hex'),
  timestamptz '2026-10-01 11:47:00+00',
  '01a0f751-e07b-75cc-9ad5-63ad9449a8ba', '01a0f751-e07b-76a1-a738-eb3e3aa3e68c', 'live',
  'owner', date '2026-10-01', 'Claude Code · Opus 5.5 (owner session 2026-10-01/02)');

-- (6b) third-party option A (owner 2026-10-02 02:12 EDT, "store them fully"; snapshot in main 9d5365e7, d04)
ALTER TABLE working.third_party_conversation ALTER COLUMN source_artifact_id DROP NOT NULL;
ALTER TABLE working.third_party_conversation ADD COLUMN source_version_id uuid;
ALTER TABLE working.third_party_conversation ADD CONSTRAINT third_party_conversation_custody_or_context_ck CHECK (((source_artifact_id IS NULL) <> (source_version_id IS NULL)));
ALTER TABLE ONLY working.third_party_conversation ADD CONSTRAINT third_party_conversation_source_version_id_fkey FOREIGN KEY (source_version_id) REFERENCES context.source_version(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX third_party_conversation_source_version_key_uq ON working.third_party_conversation USING btree (source_version_id, platform, external_thread_key) WHERE (source_version_id IS NOT NULL);
GRANT SELECT, INSERT ON TABLE working.third_party_conversation TO platform_runtime;
GRANT UPDATE (started_at, ended_at, message_count) ON TABLE working.third_party_conversation TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.third_party_message TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.third_party_message_participant TO platform_runtime;

-- (7) read back
select 'third-party grants' as t, string_agg(table_name || ':' || privilege_type, ', ' order by table_name, privilege_type)
  from information_schema.role_table_grants where grantee = 'platform_runtime' and table_schema = 'working'
   and table_name in ('third_party_conversation', 'third_party_message', 'third_party_message_participant');
select 'third-party column update' as t, string_agg(column_name, ', ' order by column_name)
  from information_schema.column_privileges where grantee = 'platform_runtime' and table_schema = 'working'
   and table_name = 'third_party_conversation' and privilege_type = 'UPDATE';
select 'third-party delete granted' as t, has_table_privilege('platform_runtime', 'working.third_party_message', 'DELETE');
select 'matter' as t, id::text, title, status, created_by, verification_state from registry.matter order by created_at;
select 'court_case' as t, id::text, caption, docket_number, court_name, case_type, status, presiding_judge from registry.court_case;
select 'person' as t, e.id::text, e.display_name, p.role_in_case, p.connection_to, p.verification_state
  from registry.person p join registry.entity e on e.id = p.id order by e.display_name;
select 'partition' as t, partition_key, matter_id::text, default_court_case_id::text from analysis.matter_knowledge_partition;
select 'receipt' as t, approved_by, approved_on, source_git_commit, payload_schema_version, payload_byte_length
  from analysis.case_registry_import_receipt;
select 'raw_format_registry' as t, count(*) from context.raw_format_registry;
select 'left in context.source_version' as t, count(*) from context.source_version;
select 'left in context.activity_execution' as t, count(*) from context.activity_execution;
select 'left in evidence.source' as t, count(*) from evidence.source;
select 'append-only triggers left' as t, string_agg(n.nspname||'.'||c.relname||':'||t.tgname, ', ' order by 1)
  from pg_trigger t join pg_class c on c.oid = t.tgrelid join pg_namespace n on n.oid = c.relnamespace
  join pg_proc p on p.oid = t.tgfoid
 where not t.tgisinternal and p.proname in ('forbid_mutation', 'source_immutable_core');

\if :{?do_commit}
  commit;
  \echo COMMITTED
\else
  rollback;
  \echo ROLLED BACK (dry run; rerun with -v do_commit=1)
\endif
