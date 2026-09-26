-- Transformation t005 v1: chronological timeline views (owner-requested)
-- Byline: Claude Code · Fable 5 · 2026-07-24
-- (1) timeline_sequence: every event in time order with its preceding/following neighbor.
-- (2) timeline_exploded: one chronological stream where visit/activity show start+stop and
--     each timeline_path is BLOWN UP into path_start · every waypoint · path_end.
BEGIN;

INSERT INTO ops.transformation (name, version, description, doc_path, sql_path)
VALUES ('t005_timeline_views', 1,
        'Chronological timeline views: timeline_sequence (events in order w/ prev/next neighbor) and timeline_exploded (visit/activity as start+stop rows, timeline_path exploded into path_start/waypoint/path_end rows) — one time-ordered stream per subject.',
        'docs/transformations/t005_timeline_views.md', 'db/transformations/t005_timeline_views_v1.sql')
ON CONFLICT (name, version) DO NOTHING;

-- refresh latest_events so it exposes the new columns (duration_seconds, path/step distance)
CREATE OR REPLACE VIEW analysis.latest_events AS
SELECT * FROM working.event WHERE build_id = (SELECT max(build_id) FROM working.build);

-- (1) chronological sequence with neighbor context
CREATE OR REPLACE VIEW analysis.timeline_sequence AS
SELECT subject_id, event_id, event_type, raw_type,
       start_utc, end_utc, start_eastern, end_eastern,
       duration_seconds, round(duration_seconds/60.0,1) AS duration_min,
       lat, lng, place_id, overnight_simple, path_time_source,
       distance_m_haversine, path_distance_m_haversine,
       lag(event_type)   OVER w AS prev_type,
       lag(end_eastern)  OVER w AS prev_end,
       lag(place_id)     OVER w AS prev_place,
       lead(event_type)  OVER w AS next_type,
       lead(start_eastern) OVER w AS next_start,
       lead(place_id)    OVER w AS next_place
FROM analysis.latest_events
WINDOW w AS (PARTITION BY subject_id ORDER BY start_utc, event_id);

-- (2) exploded stream: events as single rows, paths blown up into their waypoints
CREATE OR REPLACE VIEW analysis.timeline_exploded AS
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
WINDOW p AS (PARTITION BY e.event_id);

COMMIT;

SELECT kind, count(*) FROM analysis.timeline_exploded GROUP BY 1 ORDER BY 2 DESC;
