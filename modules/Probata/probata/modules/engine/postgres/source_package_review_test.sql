-- Synthetic disposable-database fixture for the existing metadata correction schema.
-- Inputs: a new backend_package_review_* database. Outputs: real original correction constraints.
-- Effects: creates only synthetic fixture objects; never applies to the live platform database.
CREATE SCHEMA context;
CREATE SCHEMA registry;
CREATE TABLE registry.matter(id uuid PRIMARY KEY);
CREATE TABLE registry.court_case(id uuid,matter_id uuid,PRIMARY KEY(id,matter_id));
CREATE TABLE context.source(id uuid PRIMARY KEY,source_key text);
CREATE TABLE context.source_version(id uuid PRIMARY KEY,source_id uuid,matter_id uuid,court_case_id uuid);
CREATE TABLE context.proffer_preview_binding(preview_handle text PRIMARY KEY);
CREATE TABLE context.source_metadata(id uuid);
CREATE TABLE context.source_version_object(id uuid);
CREATE TABLE context.retained_object(id uuid);
INSERT INTO registry.matter VALUES('33333333-3333-4333-8333-333333333333');
INSERT INTO registry.court_case VALUES('44444444-4444-4444-8444-444444444444','33333333-3333-4333-8333-333333333333');
INSERT INTO context.source VALUES('55555555-5555-4555-8555-555555555555','b2://case/SourceCorpus/sms/unit.xml');
INSERT INTO context.source_version VALUES('11111111-1111-4111-8111-111111111111','55555555-5555-4555-8555-555555555555','33333333-3333-4333-8333-333333333333','44444444-4444-4444-8444-444444444444');
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


CREATE FUNCTION context.forbid_mutation() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN RAISE EXCEPTION ''append-only''; END';
CREATE TRIGGER source_metadata_correction_append_only BEFORE UPDATE OR DELETE ON context.source_metadata_correction FOR EACH ROW EXECUTE FUNCTION context.forbid_mutation();