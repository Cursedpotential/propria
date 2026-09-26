-- Transformation t002 v1: wave-1 analysis views (ADR-0012)
-- Byline: Claude Code · Fable 5 · 2026-07-24
-- Views are disposable/regenerable; they read the latest build only.
BEGIN;

INSERT INTO ops.transformation (name, version, description, doc_path, sql_path)
VALUES ('t002_wave1_views', 1,
        'Wave-1 analysis views: per-place visit analytics (counts, durations, first/last visit, overnight counts), overnight event listing, daily/monthly coverage for gap analysis, impossible-speed anomalies (audited parameter), and intra-path dwell/unrecorded-stop candidates.',
        'docs/transformations/t002_wave1_views.md',
        'db/transformations/t002_wave1_views_v1.sql')
ON CONFLICT (name, version) DO NOTHING;

CREATE SCHEMA IF NOT EXISTS analysis;

CREATE OR REPLACE VIEW analysis.latest_events AS
SELECT * FROM working.event WHERE build_id = (SELECT max(build_id) FROM working.build);

-- Per-place visit analytics (recovered spec: vw_place_analytics, per subject per chunk-dedup)
CREATE OR REPLACE VIEW analysis.place_analytics AS
SELECT subject_id, place_id, lat_r4, lng_r4,
       count(DISTINCT record_id)                              AS visit_count,
       round(avg(EXTRACT(epoch FROM end_utc - start_utc))/60) AS avg_minutes,
       round(min(EXTRACT(epoch FROM end_utc - start_utc))/60) AS shortest_minutes,
       round(max(EXTRACT(epoch FROM end_utc - start_utc))/60) AS longest_minutes,
       min(start_eastern)::date                               AS first_visit,
       max(start_eastern)::date                               AS last_visit,
       count(DISTINCT record_id) FILTER (WHERE overnight_simple) AS overnight_visits,
       round(100.0 * count(DISTINCT record_id) FILTER (WHERE overnight_simple)
             / count(DISTINCT record_id), 1)                  AS overnight_pct
FROM analysis.latest_events
WHERE event_type = 'visit'
GROUP BY subject_id, place_id, lat_r4, lng_r4;

CREATE OR REPLACE VIEW analysis.overnight_events AS
SELECT subject_id, serial_display, raw_type, start_eastern, end_eastern,
       round(EXTRACT(epoch FROM end_utc - start_utc)/3600, 1) AS hours,
       place_id, lat, lng, probability
FROM analysis.latest_events
WHERE overnight_simple
ORDER BY start_utc;

-- Coverage & gap hunting (per subject; dedup across corroborating chunks via record hash join)
CREATE OR REPLACE VIEW analysis.daily_coverage AS
SELECT subject_id, start_eastern::date AS day,
       count(*) AS events,
       count(*) FILTER (WHERE event_type='visit')    AS visits,
       count(*) FILTER (WHERE event_type='activity') AS activities
FROM analysis.latest_events
GROUP BY subject_id, day;

CREATE OR REPLACE VIEW analysis.monthly_coverage AS
SELECT subject_id, to_char(start_eastern, 'YYYY-MM') AS month, count(*) AS events,
       count(DISTINCT start_eastern::date) AS days_with_data
FROM analysis.latest_events
GROUP BY subject_id, month ORDER BY month;

-- Impossible-speed anomalies from waypoint stream (parameter impossible_speed_mph)
CREATE OR REPLACE VIEW analysis.speed_anomalies AS
SELECT w.event_id, e.subject_id, e.serial_display, w.seq, w.ts_utc,
       round(w.speed_mps * 2.23694, 1) AS mph, w.gap_prev_s
FROM working.waypoint w
JOIN analysis.latest_events e USING (event_id)
WHERE w.speed_mps * 2.23694 > (SELECT (value::text)::numeric FROM ref.parameter WHERE param_name='impossible_speed_mph');

-- Intra-path unrecorded-stop candidates (owner's waypoint-gap idea, first cut)
CREATE OR REPLACE VIEW analysis.dwell_stops AS
SELECT w.event_id, e.subject_id, e.serial_display, w.seq, w.ts_utc AT TIME ZONE 'America/Detroit' AS eastern,
       w.lat, w.lng, w.gap_prev_s/60 AS gap_minutes
FROM working.waypoint w
JOIN analysis.latest_events e USING (event_id)
WHERE w.dwell_candidate
ORDER BY w.gap_prev_s DESC;

COMMIT;

SELECT 'place_analytics' v, count(*) FROM analysis.place_analytics
UNION ALL SELECT 'overnight_events', count(*) FROM analysis.overnight_events
UNION ALL SELECT 'daily_coverage', count(*) FROM analysis.daily_coverage
UNION ALL SELECT 'speed_anomalies', count(*) FROM analysis.speed_anomalies
UNION ALL SELECT 'dwell_stops', count(*) FROM analysis.dwell_stops;
