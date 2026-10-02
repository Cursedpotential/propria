-- Migration 0007: persist per-waypoint leg distance + path total distance (owner-requested)
-- t001 computed inter-point haversine transiently (for speed) but never stored it. This
-- persists it: a pre-route straight-line approximation, before full route reconstruction.
-- UP
BEGIN;

ALTER TABLE working.waypoint
    ADD COLUMN IF NOT EXISTS step_distance_m NUMERIC(12,2);  -- haversine from previous point

ALTER TABLE working.event
    ADD COLUMN IF NOT EXISTS path_distance_m_haversine NUMERIC(14,2),  -- sum of leg haversines
    ADD COLUMN IF NOT EXISTS duration_seconds INTEGER;                 -- materialized end-start

-- leg distance: haversine to previous waypoint in the same event+device track, seq-ordered
WITH legs AS (
    SELECT waypoint_id,
           ST_DistanceSphere(geom, lag(geom) OVER (PARTITION BY event_id, device_index ORDER BY seq)) AS d
    FROM working.waypoint
    WHERE build_id = (SELECT max(build_id) FROM working.build)
)
UPDATE working.waypoint w
SET step_distance_m = round(legs.d::numeric, 2)
FROM legs WHERE w.waypoint_id = legs.waypoint_id;

-- path total = sum of leg distances per path event
WITH totals AS (
    SELECT event_id, round(sum(step_distance_m)::numeric, 2) AS total
    FROM working.waypoint
    WHERE build_id = (SELECT max(build_id) FROM working.build)
    GROUP BY event_id
)
UPDATE working.event e
SET path_distance_m_haversine = totals.total
FROM totals WHERE e.event_id = totals.event_id;

-- materialized duration (seconds) for every event that has both bookends
UPDATE working.event
SET duration_seconds = EXTRACT(epoch FROM end_utc - start_utc)::int
WHERE start_utc IS NOT NULL AND end_utc IS NOT NULL
  AND build_id = (SELECT max(build_id) FROM working.build);

CREATE INDEX IF NOT EXISTS idx_wwp_step ON working.waypoint (step_distance_m);

COMMIT;

SELECT 'waypoints with leg distance' AS what, count(*) FROM working.waypoint WHERE step_distance_m IS NOT NULL
UNION ALL SELECT 'paths with total distance', count(*) FROM working.event WHERE path_distance_m_haversine IS NOT NULL
UNION ALL SELECT 'events with duration_seconds', count(*) FROM working.event WHERE duration_seconds IS NOT NULL
UNION ALL SELECT 'events MISSING duration', count(*) FROM working.event WHERE duration_seconds IS NULL;
-- DOWN: ALTER TABLE working.waypoint DROP COLUMN step_distance_m; ALTER TABLE working.event DROP COLUMN path_distance_m_haversine, DROP COLUMN duration_seconds;
