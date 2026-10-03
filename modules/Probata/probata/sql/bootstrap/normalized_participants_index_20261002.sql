-- Byline: Claude Code · Sonnet · 2026-10-02
-- OPTIONAL speed-up for the "see where this number appears" read (GET /api/imported/number-records). That read
-- finds the records carrying one phone number by JSON containment on the participants array, which is one
-- scan of the matter's ~100k normalized records without this index. Run it only if that read is slow.
-- No grants are needed: workbench_reader already reads every table the route uses.
--
-- Run as the owner of context.normalized_record_identity (platform_dba / platform_migrator).
-- Dry run (default) only shows whether the index exists and the query plan; nothing is created:
--     psql -U platform_dba -d platform -f normalized_participants_index_20261002.sql
-- Create it (CREATE INDEX CONCURRENTLY does not block writes; it cannot run inside a transaction):
--     psql -U platform_dba -d platform -v apply=1 -f normalized_participants_index_20261002.sql

SELECT 'index exists' AS check, EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'context' AND indexname = 'normalized_record_participants_gin') AS ok;

\if :{?apply}
CREATE INDEX CONCURRENTLY IF NOT EXISTS normalized_record_participants_gin
    ON context.normalized_record_identity USING gin ((normalized_payload->'participants') jsonb_path_ops);
SELECT 'index created' AS check, EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'context' AND indexname = 'normalized_record_participants_gin') AS ok;
\else
EXPLAIN (COSTS OFF)
SELECT n.id FROM context.normalized_record_identity n
WHERE n.normalized_payload->'participants' @> '[{"identifier": "+18105550142"}]'::jsonb
ORDER BY n.occurred_at DESC LIMIT 25;
\endif
