-- Byline: Claude Code · Sonnet 5.5 · 2026-10-02
-- The Workbench's Extractions view (desktop and mobile /m) reads what the extractors found, grouped by extractor, for a
-- conversation: working.extraction_run (the extractor tag), working.candidate_entity and working.candidate_event, and
-- context.normalized_generation (the conversation's current generation). workbench_reader stays SELECT-only.
-- Additive and idempotent. working.claim_candidate is NOT granted: it is the AI-chat claims lane (keyed by chat
-- conversation and message) and has no link to message conversations.
--
-- Dry run (changes nothing):  psql -U ai -d platform -f workbench_extractions_20261002.sql
-- Apply:                      psql -U ai -d platform -v finish=COMMIT -f workbench_extractions_20261002.sql

\if :{?finish}
\else
\set finish ROLLBACK
\endif

BEGIN;
GRANT SELECT ON working.extraction_run, working.candidate_entity, working.candidate_event, context.normalized_generation TO workbench_reader;

SELECT 'workbench_reader reads working.extraction_run' AS check, has_table_privilege('workbench_reader', 'working.extraction_run', 'SELECT') AS ok
UNION ALL SELECT 'workbench_reader reads working.candidate_entity', has_table_privilege('workbench_reader', 'working.candidate_entity', 'SELECT')
UNION ALL SELECT 'workbench_reader reads working.candidate_event', has_table_privilege('workbench_reader', 'working.candidate_event', 'SELECT')
UNION ALL SELECT 'workbench_reader reads context.normalized_generation', has_table_privilege('workbench_reader', 'context.normalized_generation', 'SELECT')
UNION ALL SELECT 'workbench_reader still cannot write working.candidate_entity', NOT has_table_privilege('workbench_reader', 'working.candidate_entity', 'INSERT')
UNION ALL SELECT 'workbench_reader still cannot write working.extraction_run', NOT has_table_privilege('workbench_reader', 'working.extraction_run', 'UPDATE');

:finish;
