-- Entity and event extraction: grants for the engine role.
--
-- Byline: Claude Code · Opus 5.5 · 2026-09-25
--
-- ⚠ NOT APPLIED. Written for review; the parent session applies it.
--
-- What it enables: "Extract entities" (propose) -> owner corrections ->
-- "Run workflow" (commit) on the Proffer starter and worker, which both
-- connect as platform_runtime (PLATFORM_DATABASE_URL_FILE; user verified
-- read-only 2026-09-25). No table is created: staging uses the existing
-- working.extraction_run / candidate_entity / candidate_event, commits write
-- the existing registry.entity / entity_alias, working.entity_mention /
-- entity_resolution, timeline.event_candidate (+ its typed
-- source_available_from anchor in context.relative_time_anchor, which
-- platform_runtime can already SELECT/INSERT) and timeline.timeline_member.
--
-- Live state before this script (read-only as platform_runtime, 2026-09-25):
--   * platform_runtime: no privilege on any table above; USAGE on registry
--     and working, NOT on timeline or ai.
--   * every table above holds 0 rows.
--   * ai schema: 198 functions, 0 SECURITY DEFINER, all PUBLIC EXECUTE;
--     23 tables, all with explicit ACLs. USAGE on it is name lookup only
--     (needed for the ::ai.source_ref[] provenance casts); it opens no table.
--
-- Least privilege: INSERT everywhere, UPDATE only on the columns that are
-- the documented lifecycle of a row —
--   * extraction_run: status/finished_at/error/stats (finishing a run);
--   * candidate_*: review_state + the promotion trail (supersede, commit);
--   * entity_resolution: sys_period (closing a superseded resolution).
-- No DELETE anywhere. working.entity_mention stays append-only by trigger.
--
-- Idempotent: GRANT is a no-op when already held. The final SELECT reads the
-- result back. After applying, prove the write statements parse for the
-- engine role without writing anything:
--   modules/engine: go run ./cmd/entity-dryrun sql-verify | psql -U platform_runtime -d platform
--   (every statement must print "prepared" with no ERROR line).

BEGIN;

GRANT USAGE ON SCHEMA ai TO platform_runtime;
GRANT USAGE ON SCHEMA timeline TO platform_runtime;

-- Staging (proposals and owner corrections).
GRANT SELECT, INSERT ON TABLE working.extraction_run TO platform_runtime;
GRANT UPDATE (status, finished_at, error, stats) ON TABLE working.extraction_run TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.candidate_entity TO platform_runtime;
GRANT UPDATE (review_state, promoted_to_table, promoted_to_id, promoted_at) ON TABLE working.candidate_entity TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.candidate_event TO platform_runtime;
GRANT UPDATE (review_state, promoted_to_table, promoted_to_id, promoted_at) ON TABLE working.candidate_event TO platform_runtime;

-- Committed entities.
GRANT SELECT, INSERT ON TABLE registry.entity TO platform_runtime;
GRANT SELECT, INSERT ON TABLE registry.entity_alias TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.entity_mention TO platform_runtime;
GRANT SELECT, INSERT ON TABLE working.entity_resolution TO platform_runtime;
GRANT UPDATE (sys_period) ON TABLE working.entity_resolution TO platform_runtime;

-- Committed events (the timeline_writer lane, granted to the engine role).
GRANT SELECT, INSERT ON TABLE timeline.event_candidate TO platform_runtime;
GRANT SELECT, INSERT ON TABLE timeline.event_candidate_relative_time_anchor TO platform_runtime;
GRANT SELECT, INSERT ON TABLE timeline.timeline_collection TO platform_runtime;
GRANT SELECT, INSERT ON TABLE timeline.timeline_member TO platform_runtime;

-- Read-back: every privilege this script grants, as held now.
SELECT object, privilege, has
FROM (
  VALUES
    ('schema ai', 'USAGE', has_schema_privilege('platform_runtime', 'ai', 'USAGE')),
    ('schema timeline', 'USAGE', has_schema_privilege('platform_runtime', 'timeline', 'USAGE')),
    ('working.extraction_run', 'INSERT', has_table_privilege('platform_runtime', 'working.extraction_run', 'INSERT')),
    ('working.extraction_run.status', 'UPDATE', has_column_privilege('platform_runtime', 'working.extraction_run', 'status', 'UPDATE')),
    ('working.candidate_entity', 'INSERT', has_table_privilege('platform_runtime', 'working.candidate_entity', 'INSERT')),
    ('working.candidate_entity.review_state', 'UPDATE', has_column_privilege('platform_runtime', 'working.candidate_entity', 'review_state', 'UPDATE')),
    ('working.candidate_event', 'INSERT', has_table_privilege('platform_runtime', 'working.candidate_event', 'INSERT')),
    ('working.candidate_event.promoted_to_id', 'UPDATE', has_column_privilege('platform_runtime', 'working.candidate_event', 'promoted_to_id', 'UPDATE')),
    ('registry.entity', 'INSERT', has_table_privilege('platform_runtime', 'registry.entity', 'INSERT')),
    ('registry.entity', 'UPDATE (must be false)', has_table_privilege('platform_runtime', 'registry.entity', 'UPDATE')),
    ('registry.entity_alias', 'INSERT', has_table_privilege('platform_runtime', 'registry.entity_alias', 'INSERT')),
    ('working.entity_mention', 'INSERT', has_table_privilege('platform_runtime', 'working.entity_mention', 'INSERT')),
    ('working.entity_resolution', 'INSERT', has_table_privilege('platform_runtime', 'working.entity_resolution', 'INSERT')),
    ('working.entity_resolution.sys_period', 'UPDATE', has_column_privilege('platform_runtime', 'working.entity_resolution', 'sys_period', 'UPDATE')),
    ('timeline.event_candidate', 'INSERT', has_table_privilege('platform_runtime', 'timeline.event_candidate', 'INSERT')),
    ('timeline.event_candidate_relative_time_anchor', 'INSERT', has_table_privilege('platform_runtime', 'timeline.event_candidate_relative_time_anchor', 'INSERT')),
    ('timeline.timeline_collection', 'INSERT', has_table_privilege('platform_runtime', 'timeline.timeline_collection', 'INSERT')),
    ('timeline.timeline_member', 'INSERT', has_table_privilege('platform_runtime', 'timeline.timeline_member', 'INSERT')),
    ('context.relative_time_anchor', 'INSERT (pre-existing)', has_table_privilege('platform_runtime', 'context.relative_time_anchor', 'INSERT'))
) AS granted(object, privilege, has);

COMMIT;
