-- Admit the derive execution path in context.handler_compatibility.
--
-- Byline: Claude Code · Opus 5 · 2026-09-20
--
-- ⚠ NOT APPLIED. This script is written, reviewed and committed; it has NOT
--   been run against the live database. Apply it (or rebuild from the
--   snapshot, which now carries the same constraint) BEFORE routing any
--   source through derive_structured_text_activity: until then
--   RecommendHandler will fail on the INSERT into context.handler_compatibility
--   with
--     new row for relation "handler_compatibility" violates check constraint
--     "handler_compatibility_execution_path_check"
--   and no smsbackuprestore_xml source can start.
--
-- Verified read-only on 2026-09-20 against platform on ovh-files, as
-- platform_runtime:
--   handler_compatibility_execution_path_check
--     CHECK ((execution_path = ANY (ARRAY['decoder'::text, 'duckdb'::text])))
--
-- Why a third value rather than reusing 'duckdb': the derive route produces
-- no parser bundle and no raw generation. Labelling it 'duckdb' would make
-- context.guard_raw_generation_transition()'s duckdb branch reachable for a
-- run that has nothing to seal, and would make the durable registry lie about
-- which implementation ran.
--
-- Idempotent: re-running it is a no-op once the constraint already admits
-- 'derive'. Existing rows are untouched — the new value only widens what is
-- accepted, so no row can become invalid.

BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'context.handler_compatibility'::regclass
          AND conname = 'handler_compatibility_execution_path_check'
          AND pg_get_constraintdef(oid) LIKE '%''derive''%'
    ) THEN
        RAISE NOTICE 'handler_compatibility_execution_path_check already admits derive; nothing to do';
        RETURN;
    END IF;

    ALTER TABLE context.handler_compatibility
        DROP CONSTRAINT IF EXISTS handler_compatibility_execution_path_check;

    ALTER TABLE context.handler_compatibility
        ADD CONSTRAINT handler_compatibility_execution_path_check
        CHECK (execution_path = ANY (ARRAY['decoder'::text, 'duckdb'::text, 'derive'::text]));
END
$$;

-- Proof the change took, printed before the transaction ends.
SELECT conname, pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE conrelid = 'context.handler_compatibility'::regclass
  AND conname = 'handler_compatibility_execution_path_check';

COMMIT;
