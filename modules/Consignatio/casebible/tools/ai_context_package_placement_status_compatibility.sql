-- Admit native SHA256-readback placements without weakening existing SHA1 status semantics.
-- Inputs: psql -v migration_mode=apply|rollback [-v operation=context-<actual manifest SHA256>].
-- Outputs: constraint definitions/flags and scoped rollback count/before/after digests.
-- Effects: replaces only the placement status CHECK; rollback retires exact derived placements
-- through existing status=unverified, preserving every other field, raw occurrence and source byte.
-- Choose with ai_context_package_catalog_api.sql; the Markdown occurrence API writes no placements
-- and cannot substitute for complete native package membership/readback. No DDL was run by author.
-- Source: b5c06f65, modules/Consignatio/casebible/tools/casevault_placement_load.sql:29-36;
-- same revision ai_context_package_catalog_api.sql: verified/go_context_package/
-- sha256_exact_version_readback insert and independent readback; ai_workproduct_source_occurrence_api.sql.
-- Live baseline: original ovh-files 100.91.190.107, casebible, 2026-10-11T16:32:22.899520+00:00.
-- Receipt: D:/WSL/trial-ovh/analysis/native-catalog-constraints-original-20261011.json
-- Receipt SHA256: cefabde2bebd76bcf7d05d789ed9ccb1cc9dff1458052cd47e4c2982f7243549.
-- Byline: Codex / GPT-6 / 2026-10-11; parent coordinates Case Vault owner and application.
-- Rollback requires quiescing native registration and no remaining incompatible verified rows.
-- A rollback operation is the existing context-<64 hex> operation, never an invented identifier.
-- With no added verified rows, operation can be omitted. Capture stdout/stderr as private receipt;
-- notices before a transaction failure do not prove commitment. No DELETE or new table is used.
\set ON_ERROR_STOP on
\if :{?migration_mode}
\else
  \echo 'Required: -v migration_mode=apply or rollback'
  \quit 2
\endif
\if :{?operation}
\else
  \set operation ''
\endif

BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SET LOCAL search_path=pg_catalog;
SELECT set_config('propria.native_placement_mode', :'migration_mode', true),
       set_config('propria.native_placement_operation', :'operation', true);
LOCK TABLE raw_duck.casevault_placement IN ACCESS EXCLUSIVE MODE;

-- Change one guarded constraint atomically; preserve its original comment for exact rollback.
-- Inputs: captured baseline and selected mode/operation. Outputs: notices. Effects: scoped DDL/DML only.
DO $migration$
DECLARE
  baseline_status constant text := $status$CHECK ((status = ANY (ARRAY['ok'::text, 'mismatch'::text, 'missing'::text, 'unverified'::text])))$status$;
  baseline_hash constant text := $hash$CHECK (((status <> 'ok'::text) OR ((src_sha1 IS NOT NULL) AND (src_sha1 = dst_sha1) AND (src_size = dst_size) AND (proof_level = ANY (ARRAY['b2_stored_sha1'::text, 'vps_computed_sha1'::text]))))) NOT VALID$hash$;
  tag constant text := 'ai-context-placement-status-v1';
  mode text := current_setting('propria.native_placement_mode');
  op text := current_setting('propria.native_placement_operation');
  current_status text; current_hash text; current_valid boolean; hash_valid boolean;
  status_oid oid; hash_oid oid; original_comment text; marker jsonb; expanded_definition text;
  before_rows jsonb; after_rows jsonb; changed bigint;
BEGIN
  IF current_database()<>'casebible' OR mode NOT IN ('apply','rollback') THEN
    RAISE EXCEPTION 'exact casebible database and explicit apply/rollback mode required';
  END IF;
  SELECT oid,pg_get_constraintdef(oid),convalidated,obj_description(oid,'pg_constraint')
    INTO STRICT status_oid,current_status,current_valid,original_comment FROM pg_constraint
    WHERE conrelid='raw_duck.casevault_placement'::regclass AND conname='casevault_placement_status_check' AND contype='c';
  SELECT oid,pg_get_constraintdef(oid),convalidated INTO STRICT hash_oid,current_hash,hash_valid FROM pg_constraint
    WHERE conrelid='raw_duck.casevault_placement'::regclass AND conname='casevault_placement_ok_needs_hash' AND contype='c';
  IF current_hash IS DISTINCT FROM baseline_hash OR hash_valid IS DISTINCT FROM false THEN
    RAISE EXCEPTION 'original SHA1 constraint or NOT VALID flag drifted; no changes permitted';
  END IF;

  IF current_status=baseline_status AND current_valid THEN
    IF mode='rollback' THEN
      RAISE NOTICE 'Original status CHECK is already restored; no rows changed';
      RETURN;
    END IF;
    -- All legacy statuses pass their original predicate. NULL method/proof/size can never
    -- satisfy the new branch: explicit non-null checks prevent PostgreSQL CHECK NULL escape.
    ALTER TABLE raw_duck.casevault_placement DROP CONSTRAINT casevault_placement_status_check;
    ALTER TABLE raw_duck.casevault_placement ADD CONSTRAINT casevault_placement_status_check CHECK (
      status IN ('ok','mismatch','missing','unverified') OR
      (status='verified' AND method IS NOT NULL AND method='go_context_package'
       AND proof_level IS NOT NULL AND proof_level='sha256_exact_version_readback'
       AND src_size IS NOT NULL AND dst_size IS NOT NULL AND src_size=dst_size));
    SELECT pg_get_constraintdef(oid) INTO STRICT expanded_definition FROM pg_constraint
      WHERE conrelid='raw_duck.casevault_placement'::regclass AND conname='casevault_placement_status_check';
    -- Metadata is attached only to this replaced CHECK, avoiding a parallel migration table.
    EXECUTE format('COMMENT ON CONSTRAINT casevault_placement_status_check ON raw_duck.casevault_placement IS %L',
      jsonb_build_object('migration',tag,'original_definition',baseline_status,'original_validated',true,
        'original_comment',original_comment,'expanded_definition',expanded_definition)::text);
    RAISE NOTICE 'Native verified profile admitted; original SHA1 CHECK retained unchanged';
  ELSE
    -- Reapply/rollback accepts only this exact migration's stored definition, never arbitrary drift.
    BEGIN marker:=original_comment::jsonb;
    EXCEPTION WHEN invalid_text_representation THEN RAISE EXCEPTION 'unrecognized status CHECK change'; END;
    IF marker->>'migration' IS DISTINCT FROM tag OR marker->>'original_definition' IS DISTINCT FROM baseline_status
      OR marker->'original_validated' IS DISTINCT FROM 'true'::jsonb
      OR marker->>'expanded_definition' IS DISTINCT FROM current_status OR NOT current_valid THEN
      RAISE EXCEPTION 'status CHECK differs from exact captured/migrated definitions';
    END IF;
    IF mode='apply' THEN
      RAISE NOTICE 'Exact native status compatibility already applied; no rows changed';
    ELSE
      IF op<>'' AND op !~ '^context-[0-9a-f]{64}$' THEN RAISE EXCEPTION 'exact existing native operation required'; END IF;
      IF op<>'' THEN
        IF EXISTS (SELECT 1 FROM raw_duck.casevault_placement WHERE plan_id='ai-context/'||op AND status='verified'
          AND (method IS DISTINCT FROM 'go_context_package' OR proof_level IS DISTINCT FROM 'sha256_exact_version_readback')) THEN
          RAISE EXCEPTION 'operation contains placements outside this native derived profile';
        END IF;
        SELECT coalesce(jsonb_agg(to_jsonb(p) ORDER BY dst_key),'[]'::jsonb) INTO before_rows
          FROM raw_duck.casevault_placement p WHERE plan_id='ai-context/'||op AND status='verified';
        IF jsonb_array_length(before_rows)>65 THEN RAISE EXCEPTION 'operation exceeds native package member ceiling'; END IF;
        UPDATE raw_duck.casevault_placement SET status='unverified'
          WHERE plan_id='ai-context/'||op AND status='verified'
            AND method='go_context_package' AND proof_level='sha256_exact_version_readback';
        GET DIAGNOSTICS changed=ROW_COUNT;
        SELECT coalesce(jsonb_agg(to_jsonb(p) ORDER BY dst_key),'[]'::jsonb) INTO after_rows
          FROM raw_duck.casevault_placement p WHERE plan_id='ai-context/'||op
            AND dst_key IN (SELECT value->>'dst_key' FROM jsonb_array_elements(before_rows));
        RAISE NOTICE 'retired_operation=%, rows=%, before_sha256=%, after_sha256=%',op,changed,
          encode(sha256(convert_to(before_rows::text,'UTF8')),'hex'),encode(sha256(convert_to(after_rows::text,'UTF8')),'hex');
      END IF;
      IF EXISTS (SELECT 1 FROM raw_duck.casevault_placement WHERE status NOT IN ('ok','mismatch','missing','unverified')) THEN
        RAISE EXCEPTION 'incompatible added rows remain; entire rollback aborted, including scoped retirement';
      END IF;
      ALTER TABLE raw_duck.casevault_placement DROP CONSTRAINT casevault_placement_status_check;
      EXECUTE 'ALTER TABLE raw_duck.casevault_placement ADD CONSTRAINT casevault_placement_status_check '||baseline_status;
      EXECUTE format('COMMENT ON CONSTRAINT casevault_placement_status_check ON raw_duck.casevault_placement IS %L',
        marker->>'original_comment');
      RAISE NOTICE 'Original validated status CHECK restored; package receipts/members/sources and all proof fields preserved';
    END IF;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE oid=hash_oid AND pg_get_constraintdef(oid)=baseline_hash
    AND convalidated=false) THEN RAISE EXCEPTION 'original SHA1 check changed unexpectedly'; END IF;
END
$migration$;
COMMIT;

-- Read back committed schema state; inputs none, outputs exact definitions/flags, effects read-only.
SELECT conname,pg_get_constraintdef(oid) AS definition,convalidated
  FROM pg_constraint WHERE conrelid='raw_duck.casevault_placement'::regclass
    AND conname IN ('casevault_placement_status_check','casevault_placement_ok_needs_hash') ORDER BY conname;
SELECT plan_id,status,method,proof_level,count(*) AS committed_rows
  FROM raw_duck.casevault_placement WHERE plan_id='ai-context/'||:'operation'
  GROUP BY plan_id,status,method,proof_level ORDER BY status;
