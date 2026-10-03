-- Byline: Claude Code · Sonnet · 2026-10-02
-- The mobile Imported view takes each source file's status from PostgreSQL, not only from the Proffer run
-- list: a file is Done when ANY of its source versions has a successful publish_generation_activity receipt.
-- That needs workbench_reader (SELECT-only) to read the two activity tables. Additive and idempotent.
--
-- Dry run (changes nothing):  psql -U ai -d platform -f workbench_status_20261002.sql
-- Apply:                      psql -U ai -d platform -v finish=COMMIT -f workbench_status_20261002.sql

\if :{?finish}
\else
\set finish ROLLBACK
\endif

BEGIN;
GRANT SELECT ON context.activity_execution, context.activity_receipt TO workbench_reader;

SELECT 'workbench_reader reads context.activity_execution' AS check, has_table_privilege('workbench_reader', 'context.activity_execution', 'SELECT') AS ok
UNION ALL SELECT 'workbench_reader reads context.activity_receipt', has_table_privilege('workbench_reader', 'context.activity_receipt', 'SELECT')
UNION ALL SELECT 'workbench_reader still cannot write context.activity_receipt', NOT has_table_privilege('workbench_reader', 'context.activity_receipt', 'INSERT')
UNION ALL SELECT 'publish receipts exist to read', EXISTS (
    SELECT 1 FROM context.activity_execution ae JOIN context.activity_receipt ar ON ar.activity_execution_id = ae.id
    WHERE ae.activity_name = 'publish_generation_activity' AND ar.status = 'success');

:finish;
