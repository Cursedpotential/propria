-- Transformation t006 v1: materialize timeline_exploded (owner: both views must be
-- accessible, persistent objects). timeline_sequence stays a light view; the heavy
-- exploded UNION becomes a physical, indexed materialized view — fast to browse, and
-- REFRESHed as part of the working-layer rebuild.
-- Byline: Claude Code · Fable 5 · 2026-07-24
BEGIN;

INSERT INTO ops.transformation (name, version, description, doc_path, sql_path)
VALUES ('t006_materialize_timeline_exploded', 1,
        'Materializes analysis.timeline_exploded (events + exploded path waypoints, one time-ordered stream) as a physical indexed materialized view for fast browsing; timeline_sequence remains a plain view. Refresh with REFRESH MATERIALIZED VIEW after each working rebuild.',
        'docs/transformations/t006_materialize_timeline_exploded.md', 'db/transformations/t006_materialize_timeline_exploded_v1.sql')
ON CONFLICT (name, version) DO NOTHING;

DROP VIEW IF EXISTS analysis.timeline_exploded;

CREATE MATERIALIZED VIEW analysis.timeline_exploded AS
SELECT subject_id, start_utc AS ts_utc, start_eastern AS eastern, end_eastern,
       event_type AS kind, event_id, NULL::int AS seq,
       round(duration_seconds/60.0,1) AS duration_min,
       lat, lng, place_id, NULL::numeric AS step_distance_m
FROM analysis.latest_events
WHERE event_type IN ('visit','activity')
UNION ALL
SELECT e.subject_id, w.ts_utc,
       w.ts_utc AT TIME ZONE 'America/Detroit' AS eastern, NULL::timestamp AS end_eastern,
       CASE WHEN w.seq = min(w.seq) OVER p THEN 'path_start'
            WHEN w.seq = max(w.seq) OVER p THEN 'path_end'
            ELSE 'waypoint' END AS kind,
       e.event_id, w.seq, NULL::numeric AS duration_min,
       w.lat, w.lng, NULL::text AS place_id, w.step_distance_m
FROM working.waypoint w
JOIN analysis.latest_events e ON e.event_id = w.event_id
WHERE e.event_type = 'timeline_path'
WINDOW p AS (PARTITION BY e.event_id)
WITH DATA;

CREATE INDEX idx_tlexpl_subj_ts ON analysis.timeline_exploded (subject_id, ts_utc);
CREATE INDEX idx_tlexpl_kind ON analysis.timeline_exploded (kind);
CREATE INDEX idx_tlexpl_event ON analysis.timeline_exploded (event_id);

COMMIT;

SELECT 'timeline_exploded (materialized) rows' AS obj, count(*) FROM analysis.timeline_exploded
UNION ALL SELECT 'timeline_sequence (view) rows', count(*) FROM analysis.timeline_sequence;
-- Refresh contract: after any working rebuild -> REFRESH MATERIALIZED VIEW analysis.timeline_exploded;
