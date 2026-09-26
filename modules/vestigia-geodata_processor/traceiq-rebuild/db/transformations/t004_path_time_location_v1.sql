-- Transformation t004 v1: correct timeline_path start/end TIMES and LOCATIONS
-- Byline: Claude Code · Fable 5 · 2026-07-24
-- Google's path CONTAINER times are ~2h-rounded and useless (a 5-min trip shows as 2h).
-- Correct them from the actual waypoints (tier 1), tagging path_time_source so the method
-- is transparent (ADR-0005). Raw container times stay verbatim in raw.record. Also set the
-- path's representative location from its first waypoint, and recompute duration + eastern.
-- Tier 2 (bookend to adjacent visit/activity ±5min for path-less paths) = v2 follow-up.
BEGIN;

INSERT INTO ops.transformation (name, version, description, doc_path, sql_path)
VALUES ('t004_path_time_location', 1,
        'Corrects timeline_path event times from actual first/last waypoint (Google container times are ~2h-rounded and wrong); sets path location from first waypoint; recomputes duration and Eastern display; tags path_time_source (points_first_last / container_fallback). Raw is untouched.',
        'docs/transformations/t004_path_time_location.md', 'db/transformations/t004_path_time_location_v1.sql')
ON CONFLICT (name, version) DO NOTHING;

-- TIER 1: paths WITH waypoints -> use first/last waypoint time + first waypoint location
WITH wp AS (
    SELECT event_id,
           min(ts_utc) AS first_ts, max(ts_utc) AS last_ts,
           (array_agg(lat ORDER BY seq))[1] AS first_lat,
           (array_agg(lng ORDER BY seq))[1] AS first_lng
    FROM working.waypoint
    WHERE build_id = (SELECT max(build_id) FROM working.build) AND ts_utc IS NOT NULL
    GROUP BY event_id
)
UPDATE working.event e SET
    start_utc = wp.first_ts,
    end_utc   = wp.last_ts,
    start_eastern = wp.first_ts AT TIME ZONE 'America/Detroit',
    end_eastern   = wp.last_ts  AT TIME ZONE 'America/Detroit',
    lat = wp.first_lat, lng = wp.first_lng,
    geom = CASE WHEN wp.first_lat IS NOT NULL THEN ST_SetSRID(ST_MakePoint(wp.first_lng, wp.first_lat),4326) END,
    lat_r4 = round(wp.first_lat,4), lng_r4 = round(wp.first_lng,4),
    path_time_source = 'points_first_last'
FROM wp
WHERE e.event_id = wp.event_id AND e.event_type = 'timeline_path';

-- TIER 3: paths WITHOUT usable waypoints -> keep container times, flag low-confidence
UPDATE working.event e
SET path_time_source = 'container_fallback'
WHERE e.event_type = 'timeline_path' AND e.path_time_source IS NULL;

-- recompute duration for paths now that times are corrected
UPDATE working.event
SET duration_seconds = EXTRACT(epoch FROM end_utc - start_utc)::int
WHERE event_type = 'timeline_path' AND start_utc IS NOT NULL AND end_utc IS NOT NULL;

COMMIT;

SELECT path_time_source, count(*),
       round(avg(duration_seconds)/60.0,1) AS avg_min,
       count(*) FILTER (WHERE lat IS NOT NULL) AS has_location
FROM analysis.latest_events WHERE event_type='timeline_path' GROUP BY 1;
