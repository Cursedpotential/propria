-- Transformation t003 v1: remaining Tier-1 analysis views (recovered-spec parity lane)
-- Byline: Claude Code · Fable 5 · 2026-07-24
BEGIN;

INSERT INTO ops.transformation (name, version, description, doc_path, sql_path)
VALUES ('t003_analysis_suite', 1,
        'Tier-1 completions of the recovered view inventory: route patterns (frequent start-end pairs), bouncy trips (travel far out of proportion to displacement), hourly/day-of-week patterns, location frequency across all event types, and multi-device timestamp-collision signals.',
        'docs/transformations/t003_analysis_suite.md',
        'db/transformations/t003_analysis_suite_v1.sql')
ON CONFLICT (name, version) DO NOTHING;

-- Frequent routes: start-r4 -> end-r4 pairs on activities
CREATE OR REPLACE VIEW analysis.route_patterns AS
SELECT subject_id, lat_r4 AS start_lat_r4, lng_r4 AS start_lng_r4,
       round((raw_dist.end_lat)::numeric, 4)  AS end_lat_r4,
       round((raw_dist.end_lng)::numeric, 4)  AS end_lng_r4,
       count(*) AS trips,
       round(avg(distance_m_google)) AS avg_google_m,
       min(start_eastern)::date AS first_trip, max(start_eastern)::date AS last_trip
FROM analysis.latest_events e
CROSS JOIN LATERAL (
    SELECT split_part(replace(r.raw_json #>> '{activity,end,latLng}', '°', ''), ',', 1)::numeric AS end_lat,
           trim(split_part(replace(r.raw_json #>> '{activity,end,latLng}', '°', ''), ',', 2))::numeric AS end_lng
    FROM raw.record r WHERE r.record_id = e.record_id
) raw_dist
WHERE e.event_type = 'activity' AND e.lat IS NOT NULL AND raw_dist.end_lat IS NOT NULL
GROUP BY 1, 2, 3, 4, 5
HAVING count(*) >= 3;

-- Bouncy trips: distance traveled >> straight-line displacement (boomerang/zigzag)
CREATE OR REPLACE VIEW analysis.bouncy_trips AS
SELECT subject_id, serial_display, start_eastern, end_eastern,
       distance_m_google, distance_m_haversine,
       round(distance_m_google / NULLIF(distance_m_haversine, 0), 1) AS bounce_ratio
FROM analysis.latest_events
WHERE event_type = 'activity'
  AND distance_m_google > 1000
  AND distance_m_google > 3 * COALESCE(distance_m_haversine, 0)
ORDER BY bounce_ratio DESC NULLS LAST;

-- Hour-of-day / day-of-week rhythm
CREATE OR REPLACE VIEW analysis.hourly_patterns AS
SELECT subject_id, event_type,
       extract(dow from start_eastern)::int AS dow,
       extract(hour from start_eastern)::int AS hour,
       count(*) AS events
FROM analysis.latest_events
WHERE start_eastern IS NOT NULL
GROUP BY 1, 2, 3, 4;

-- Location frequency across ALL typed events (not just visits)
CREATE OR REPLACE VIEW analysis.location_frequency AS
SELECT subject_id, lat_r4, lng_r4,
       count(*) AS touches,
       count(*) FILTER (WHERE event_type = 'visit') AS visits,
       count(*) FILTER (WHERE overnight_simple) AS overnight_touches,
       min(start_eastern)::date AS first_seen, max(start_eastern)::date AS last_seen
FROM analysis.latest_events
WHERE lat IS NOT NULL
GROUP BY 1, 2, 3;

-- Multi-device signal: same-event duplicate-timestamp waypoint pairs with real separation
CREATE OR REPLACE VIEW analysis.multi_device_signals AS
SELECT e.subject_id, e.serial_display, w1.ts_utc,
       round(ST_DistanceSphere(w1.geom, w2.geom)::numeric) AS separation_m
FROM working.waypoint w1
JOIN working.waypoint w2
  ON w1.event_id = w2.event_id AND w1.ts_utc = w2.ts_utc AND w1.waypoint_id < w2.waypoint_id
JOIN analysis.latest_events e ON e.event_id = w1.event_id
WHERE ST_DistanceSphere(w1.geom, w2.geom) >
      (SELECT (value::text)::numeric FROM ref.parameter WHERE param_name = 'multi_device_min_m')
ORDER BY separation_m DESC;

COMMIT;

SELECT 'route_patterns' v, count(*) FROM analysis.route_patterns
UNION ALL SELECT 'bouncy_trips', count(*) FROM analysis.bouncy_trips
UNION ALL SELECT 'hourly_patterns', count(*) FROM analysis.hourly_patterns
UNION ALL SELECT 'location_frequency', count(*) FROM analysis.location_frequency
UNION ALL SELECT 'multi_device_signals', count(*) FROM analysis.multi_device_signals;
