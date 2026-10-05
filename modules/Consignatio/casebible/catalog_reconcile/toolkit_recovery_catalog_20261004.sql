-- Byline: Codex · GPT-6 · 2026-10-04.
-- Inputs: DBA execution in casebible after source review. Outputs: separate append-only toolkit recovery ledger/view and NOLOGIN roles.
-- Effects: additive DDL only; no corpus, dated catalogs, current_generation or existing roles are modified.
-- Choose for verified whole-original preservation receipts; publication/selection/import remain separate operations.
BEGIN;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';
DO $$ BEGIN
 IF current_database() <> 'casebible' THEN RAISE EXCEPTION 'toolkit recovery migration requires casebible'; END IF;
 IF to_regnamespace('catalog_reconcile') IS NULL THEN RAISE EXCEPTION 'existing Case Bible catalog_reconcile schema required'; END IF;
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='casebible_toolkit_recovery_reader') THEN CREATE ROLE casebible_toolkit_recovery_reader NOLOGIN; END IF;
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='casebible_toolkit_recovery_writer') THEN CREATE ROLE casebible_toolkit_recovery_writer NOLOGIN; END IF;
END $$;
CREATE SCHEMA IF NOT EXISTS casebible_recovery;
CREATE TABLE IF NOT EXISTS casebible_recovery.toolkit_registration (
 operation_id text PRIMARY KEY CHECK (operation_id ~ '^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$'),
 metadata_sha256 text NOT NULL CHECK (metadata_sha256 ~ '^[a-f0-9]{64}$'),
 payload jsonb NOT NULL CHECK (octet_length(payload::text) <= 524288),
 registered_at timestamptz NOT NULL DEFAULT now(),
 CHECK (jsonb_typeof(payload)='object'),
 CHECK ((payload->>'schema' = 'toolkit-recovery-catalog/v1') IS TRUE),
 CHECK ((payload->'request'->>'operation_id' = operation_id) IS TRUE),
 CHECK ((jsonb_typeof(payload->'packages')='array' AND jsonb_array_length(payload->'packages')=15) IS TRUE)
);
CREATE TABLE IF NOT EXISTS casebible_recovery.toolkit_occurrence (
 operation_id text NOT NULL REFERENCES casebible_recovery.toolkit_registration(operation_id),
 original_ref text NOT NULL CHECK (length(original_ref) BETWEEN 1 AND 4096),
 package_name text NOT NULL CHECK (length(package_name) BETWEEN 1 AND 512),
 payload jsonb NOT NULL CHECK (octet_length(payload::text) <= 32768),
 PRIMARY KEY(operation_id,original_ref), UNIQUE(operation_id,package_name),
 CHECK (jsonb_typeof(payload)='object'),
 CHECK ((payload->>'original_ref'=original_ref AND payload->>'package_name'=package_name) IS TRUE),
 CHECK ((payload->>'storage_mode'='versioned-recovery') IS TRUE),
 CHECK ((payload->>'sha256' ~ '^[a-f0-9]{64}$') IS TRUE),
 CHECK (((payload->>'bytes')::bigint BETWEEN 1 AND 5000000000) IS TRUE),
 CHECK ((length(payload->>'archive_version_id') BETWEEN 1 AND 2048 AND payload->>'archive_version_id'<>'null') IS TRUE),
 CHECK ((length(payload->>'receipt_version_id') BETWEEN 1 AND 2048 AND payload->>'receipt_version_id'<>'null') IS TRUE),
 CHECK ((length(payload->>'verification_version_id') BETWEEN 1 AND 2048 AND payload->>'verification_version_id'<>'null') IS TRUE),
 CHECK ((payload->>'receipt_sha256' ~ '^[a-f0-9]{64}$' AND payload->>'verification_receipt_sha256' ~ '^[a-f0-9]{64}$') IS TRUE)
);
COMMENT ON TABLE casebible_recovery.toolkit_registration IS 'Immutable scoped metadata operations; independent of dated catalog current_generation. No format/legal/selection claim.';
COMMENT ON TABLE casebible_recovery.toolkit_occurrence IS 'Every original occurrence remains distinct even for identical archive SHA; JSON retains exact archive and receipt refs/VersionIds.';
CREATE OR REPLACE VIEW catalog_reconcile.toolkit_recovered_originals AS
 SELECT r.operation_id,r.metadata_sha256,r.registered_at,
 r.payload->'request'->>'result_ref' AS preservation_result_ref,
 r.payload->'request'->>'result_sha256' AS preservation_result_sha256,
 r.payload->'request'->>'result_version_id' AS preservation_result_version_id,
 r.payload->>'inventory_ref' AS inventory_ref,r.payload->>'inventory_sha256' AS inventory_sha256,
 o.original_ref,o.package_name,o.payload->>'preserved_ref' AS preserved_ref,
 o.payload->>'archive_version_id' AS archive_version_id,o.payload->>'sha256' AS archive_sha256,
 (o.payload->>'bytes')::bigint AS archive_bytes,
 o.payload->>'receipt_ref' AS preservation_receipt_ref,o.payload->>'receipt_version_id' AS preservation_receipt_version_id,
 o.payload->>'receipt_sha256' AS preservation_receipt_sha256,
 o.payload->>'verification_ref' AS verification_receipt_ref,o.payload->>'verification_version_id' AS verification_receipt_version_id,
 o.payload->>'verification_receipt_sha256' AS verification_receipt_sha256,
 'preservation_evidence_verified'::text AS preservation_status,
 'not_assessed'::text AS format_completeness_status,'not_assessed'::text AS final_material_status,
 'not_refreshed'::text AS projection_status,o.payload AS metadata_receipt
 FROM casebible_recovery.toolkit_registration r JOIN casebible_recovery.toolkit_occurrence o USING(operation_id);
COMMENT ON VIEW catalog_reconcile.toolkit_recovered_originals IS 'Scoped recovery metadata and exact retained-version references; does not replace current_generation or establish corpus/final-material validation.';
GRANT USAGE ON SCHEMA casebible_recovery,catalog_reconcile TO casebible_toolkit_recovery_reader,casebible_toolkit_recovery_writer;
GRANT SELECT ON casebible_recovery.toolkit_registration,casebible_recovery.toolkit_occurrence,catalog_reconcile.toolkit_recovered_originals TO casebible_toolkit_recovery_reader,casebible_toolkit_recovery_writer;
GRANT INSERT ON casebible_recovery.toolkit_registration,casebible_recovery.toolkit_occurrence TO casebible_toolkit_recovery_writer;
-- Parent DBA provisions a separate LOGIN and grants only the writer role; credentials never belong in this migration.
-- Writer runtime admits no superuser/BYPASSRLS principal. No UPDATE/DELETE/TRUNCATE/DDL privilege is granted.
COMMIT;
