-- Synthetic store-integration fixture only; all objects live in an outer rolled-back transaction.
-- Inputs: empty dedicated backend_package_review_operator_results_20261015 database.
-- Outputs: minimal existing-store table/view columns, not a production migration or schema proof.
CREATE SCHEMA context;
CREATE SCHEMA working;
CREATE SCHEMA registry;
CREATE TABLE context.source(id uuid PRIMARY KEY,source_key text);
CREATE TABLE context.source_version(id uuid PRIMARY KEY,source_id uuid,workflow_id text,matter_id uuid,court_case_id uuid,declared_format text,version_ordinal int,created_at timestamptz DEFAULT now());
CREATE TABLE context.raw_generation(id uuid PRIMARY KEY,source_version_id uuid);
CREATE TABLE context.normalized_generation(id uuid PRIMARY KEY,source_version_id uuid,raw_generation_id uuid);
CREATE TABLE context.normalized_record_identity(id uuid PRIMARY KEY,source_version_id uuid,normalized_generation_id uuid,record_ordinal bigint,record_type text,occurred_at timestamptz,normalized_payload jsonb);
CREATE TABLE context.proffer_preview_binding(preview_handle text PRIMARY KEY,request_id text,source_ref text,workflow_id text,run_id text,parser_options_ref text,created_at timestamptz DEFAULT now());
CREATE TABLE context.proffer_preview_snapshot(preview_handle text,snapshot_seq int,source_version_id uuid,raw_generation_id uuid,normalized_generation_id uuid);
CREATE TABLE context.proffer_preview_decision(id uuid,preview_handle text,selection_ref text,parser_options_ref text,recorded_at timestamptz);
CREATE TABLE context.proffer_preview_event(preview_handle text,event_id bigint,event_type text,occurred_at timestamptz,phase text,detail text,PRIMARY KEY(preview_handle,event_id));
CREATE TABLE context.activity_execution(id uuid PRIMARY KEY,source_version_id uuid,workflow_id text,activity_name text,created_at timestamptz DEFAULT now());
CREATE TABLE context.activity_receipt(id uuid PRIMARY KEY,activity_execution_id uuid,status text,result_ref jsonb,not_applicable_reason text,error_detail jsonb,attempt int,started_at timestamptz,completed_at timestamptz);
CREATE TABLE context.proffer_source_context_revision(source_context_ref uuid PRIMARY KEY,request_id text,revision int,supersedes_ref uuid,matter_id uuid,court_case_id uuid,source_ref text,observed_source jsonb,previous_assertions jsonb,assertions jsonb,change_reason text,actor_subject_uid text,actor_username text,idempotency_key text UNIQUE,content_digest bytea,receipt_ref text,recorded_at timestamptz);
CREATE TABLE working.extraction_run(id uuid PRIMARY KEY,extractor text,extractor_version text,model_id text,source_summary text,status text,stats jsonb,prompt_version text,started_at timestamptz DEFAULT now(),finished_at timestamptz,error text);
CREATE TABLE working.candidate_entity(id uuid PRIMARY KEY,extraction_run_id uuid REFERENCES working.extraction_run(id),source_raw_table text,source_raw_id text,entity_type text,name text,normalized_name text,confidence float8,attrs jsonb,content_sha256 bytea,review_state text,domain text,ontology_version text,knowledge_time timestamptz,created_at timestamptz DEFAULT now(),promoted_to_id text);
CREATE TABLE working.candidate_event(id uuid PRIMARY KEY,extraction_run_id uuid REFERENCES working.extraction_run(id),source_raw_table text,source_raw_id text,event_type text,summary text,occurred_at timestamptz,temporal_confidence float8,confidence float8,attrs jsonb,content_sha256 bytea,review_state text,domain text,ontology_version text,knowledge_time timestamptz,created_at timestamptz DEFAULT now(),promoted_to_id text);
CREATE TABLE registry.entity(id uuid PRIMARY KEY,display_name text,merged_into_id uuid);
CREATE TABLE registry.person(id uuid PRIMARY KEY REFERENCES registry.entity(id));
CREATE TABLE registry.entity_alias(id uuid PRIMARY KEY,entity_id uuid REFERENCES registry.entity(id),alias_text text,alias_kind text,normalized text,status text,period text,basis text,recorded_by text,created_at timestamptz DEFAULT now());
CREATE TABLE registry.identity_change(id uuid PRIMARY KEY,subject_table text,subject_id uuid,before_state jsonb,after_state jsonb,change_reason text,recorded_by text,recorded_by_uid text,idempotency_key text UNIQUE,recorded_at timestamptz);
CREATE FUNCTION registry.norm_identifier(text) RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT right(regexp_replace($1,''[^0-9]'','''',''g''),10)';
CREATE VIEW registry.entity_alias_current AS SELECT * FROM registry.entity_alias;
CREATE VIEW registry.vw_case_identifier AS SELECT a.id AS alias_id,e.display_name AS person,a.alias_kind AS kind FROM registry.entity_alias a JOIN registry.entity e ON e.id=a.entity_id;
