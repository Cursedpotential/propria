-- Transformation t001 v1: P1-explode — raw records → working.event + working.waypoint
-- Byline: Claude Code · Fable 5 · 2026-07-24
-- Judge-readable doc: docs/transformations/t001_p1_explode.md (what/why/worked example)
-- Contract (ADR-0005): every raw record in scope → exactly one working.event OR one
-- working.exception row; reconciliation counts recorded on working.build.
-- Scope: modern semanticSegments records. Verbatim raw is never modified.

BEGIN;

INSERT INTO ops.transformation (name, version, description, doc_path, sql_path)
VALUES ('t001_p1_explode', 1,
        'Explodes verbatim Google Timeline records into typed events and sequence-ordered waypoints. Extracts times (using Google''s own attested timezone offsets), coordinates (full precision, plus PostGIS point and rounded cache keys), place references, probabilities, and distances. Computes Eastern display time, gap-to-previous, and the simple overnight flag from the audited overnight_window parameter. Rare trip/memory records fold into activity with their original name preserved in raw_type (ADR-0005).',
        'docs/transformations/t001_p1_explode.md',
        'db/transformations/t001_p1_explode_v1.sql')
ON CONFLICT (name, version) DO NOTHING;

INSERT INTO working.build (transform_versions)
VALUES ('{"t001_p1_explode": 1}'::jsonb);

WITH b AS (
    SELECT max(build_id) AS build_id FROM working.build
),
scope AS (
    SELECT r.record_id, r.chunk_id, r.google_type, r.raw_json,
           c.export_id, e.account_id, a.subject_id
    FROM raw.record r
    JOIN raw.chunk c USING (chunk_id)
    JOIN raw.export e USING (export_id)
    JOIN raw.account a USING (account_id)
    WHERE r.google_type LIKE 'semanticSegments.%'
),
parsed AS (
    SELECT s.*,
        CASE s.google_type
            WHEN 'semanticSegments.visit'          THEN 'visit'
            WHEN 'semanticSegments.activity'       THEN 'activity'
            WHEN 'semanticSegments.timelinePath'   THEN 'timeline_path'
            WHEN 'semanticSegments.timelineMemory' THEN 'activity'   -- folded; raw_type keeps truth
        END AS event_type,
        replace(s.google_type, 'semanticSegments.', '') AS raw_type,
        (s.raw_json->>'startTime')::timestamptz AS start_utc,
        (s.raw_json->>'endTime')::timestamptz   AS end_utc,
        (s.raw_json->>'startTimeTimezoneUtcOffsetMinutes')::int AS start_off_min,
        CASE
          WHEN s.google_type = 'semanticSegments.visit'
            THEN s.raw_json #>> '{visit,topCandidate,placeLocation,latLng}'
          WHEN s.google_type = 'semanticSegments.activity'
            THEN s.raw_json #>> '{activity,start,latLng}'
        END AS latlng_str,
        s.raw_json #>> '{visit,topCandidate,placeId}'          AS place_id,
        COALESCE((s.raw_json #>> '{visit,probability}')::numeric,
                 (s.raw_json #>> '{activity,topCandidate,probability}')::numeric) AS probability,
        (s.raw_json #>> '{activity,distanceMeters}')::numeric  AS dist_google,
        s.raw_json #>> '{activity,end,latLng}'                 AS end_latlng_str
    FROM scope s
),
coords AS (
    SELECT p.*,
        NULLIF(split_part(replace(p.latlng_str, '°', ''), ',', 1), '')::numeric AS lat,
        NULLIF(trim(split_part(replace(p.latlng_str, '°', ''), ',', 2)), '')::numeric AS lng,
        NULLIF(split_part(replace(p.end_latlng_str, '°', ''), ',', 1), '')::numeric AS end_lat,
        NULLIF(trim(split_part(replace(p.end_latlng_str, '°', ''), ',', 2)), '')::numeric AS end_lng
    FROM parsed p
),
ow AS (
    SELECT (value->>'start')::time AS ow_start, (value->>'end')::time AS ow_end
    FROM ref.parameter WHERE param_name = 'overnight_window'
)
INSERT INTO working.event (build_id, record_id, subject_id, account_id, export_id,
        event_type, raw_type, serial_display, start_utc, end_utc, tz_name,
        start_local, end_local, start_eastern, end_eastern,
        lat, lng, geom, lat_r3, lng_r3, lat_r4, lng_r4, geohash8, geohash9,
        place_id, probability, distance_m_google, distance_m_haversine,
        distance_discrepancy, overnight_simple)
    SELECT
        (SELECT build_id FROM b), c.record_id, c.subject_id, c.account_id, c.export_id,
        c.event_type, c.raw_type,
        to_char(c.start_utc AT TIME ZONE 'America/Detroit', 'YYMMDDHH24MI')
            || lower(substr(c.raw_type, 1, 1)),
        c.start_utc, c.end_utc,
        'utc_offset_' || COALESCE(c.start_off_min::text, '?'),
        c.start_utc + make_interval(mins => COALESCE(c.start_off_min, 0)),
        c.end_utc   + make_interval(mins => COALESCE(c.start_off_min, 0)),
        c.start_utc AT TIME ZONE 'America/Detroit',
        c.end_utc   AT TIME ZONE 'America/Detroit',
        c.lat, c.lng,
        CASE WHEN c.lat IS NOT NULL THEN ST_SetSRID(ST_MakePoint(c.lng, c.lat), 4326) END,
        round(c.lat, 3), round(c.lng, 3), round(c.lat, 4), round(c.lng, 4),
        CASE WHEN c.lat IS NOT NULL THEN ST_GeoHash(ST_SetSRID(ST_MakePoint(c.lng, c.lat), 4326), 8) END,
        CASE WHEN c.lat IS NOT NULL THEN ST_GeoHash(ST_SetSRID(ST_MakePoint(c.lng, c.lat), 4326), 9) END,
        c.place_id, c.probability, c.dist_google,
        CASE WHEN c.end_lat IS NOT NULL AND c.lat IS NOT NULL THEN
            round(ST_DistanceSphere(ST_MakePoint(c.lng, c.lat), ST_MakePoint(c.end_lng, c.end_lat))::numeric, 1)
        END,
        false,
        COALESCE(
            -- v1 approximation: touches the overnight window or crosses local midnight;
            -- evaluated in Google's attested local time. v2 = home-base-aware (ADR-0008).
            ((c.start_utc + make_interval(mins => COALESCE(c.start_off_min,0)))::time >= (SELECT ow_start FROM ow)
              OR (c.end_utc + make_interval(mins => COALESCE(c.start_off_min,0)))::time < (SELECT ow_end FROM ow)
              OR ((c.end_utc + make_interval(mins => COALESCE(c.start_off_min,0)))::date
                   > (c.start_utc + make_interval(mins => COALESCE(c.start_off_min,0)))::date)),
            false)
    FROM coords c;

INSERT INTO working.exception (build_id, record_id, reason)
SELECT (SELECT max(build_id) FROM working.build), s.record_id,
       'out_of_scope_type: ' || COALESCE(s.google_type,'null')
FROM raw.record s
WHERE s.google_type IS NULL OR s.google_type NOT LIKE 'semanticSegments.%';

UPDATE working.build SET
    finished_at = now(),
    raw_count = (SELECT count(*) FROM raw.record),
    derived_count = (SELECT count(*) FROM working.event  WHERE build_id = working.build.build_id),
    exception_count = (SELECT count(*) FROM working.exception WHERE build_id = working.build.build_id)
WHERE build_id = (SELECT max(build_id) FROM working.build);

-- waypoints: sequence-ordered explode of timelinePath arrays (NEVER timestamp-ordered)
INSERT INTO working.waypoint (build_id, event_id, seq, ts_utc, lat, lng, geom)
SELECT e.build_id, e.event_id, wp.ord - 1,
       (wp.p->>'time')::timestamptz,
       split_part(replace(wp.p->>'point', '°', ''), ',', 1)::numeric,
       trim(split_part(replace(wp.p->>'point', '°', ''), ',', 2))::numeric,
       ST_SetSRID(ST_MakePoint(
           trim(split_part(replace(wp.p->>'point', '°', ''), ',', 2))::numeric,
           split_part(replace(wp.p->>'point', '°', ''), ',', 1)::numeric), 4326)
FROM working.event e
JOIN raw.record r ON r.record_id = e.record_id
CROSS JOIN LATERAL jsonb_array_elements(r.raw_json->'timelinePath') WITH ORDINALITY AS wp(p, ord)
WHERE e.raw_type = 'timelinePath'
  AND e.build_id = (SELECT max(build_id) FROM working.build);

-- waypoint speed + gap + dwell candidates (dwell_threshold_min parameter)
WITH w AS (
    SELECT waypoint_id,
           EXTRACT(epoch FROM ts_utc - lag(ts_utc) OVER (PARTITION BY event_id ORDER BY seq))::int AS gap_s,
           ST_DistanceSphere(geom, lag(geom) OVER (PARTITION BY event_id ORDER BY seq)) AS step_m
    FROM working.waypoint
    WHERE build_id = (SELECT max(build_id) FROM working.build)
), t AS (SELECT (value::text)::numeric * 60 AS dwell_s FROM ref.parameter WHERE param_name='dwell_threshold_min')
UPDATE working.waypoint wp SET
    gap_prev_s = w.gap_s,
    speed_mps = CASE WHEN w.gap_s > 0 THEN round((w.step_m / w.gap_s)::numeric, 2) END,
    dwell_candidate = COALESCE(w.gap_s >= (SELECT dwell_s FROM t) AND w.step_m < 100, false)
FROM w WHERE wp.waypoint_id = w.waypoint_id;

-- event gap-to-previous per subject
WITH g AS (
    SELECT event_id,
           EXTRACT(epoch FROM start_utc - lag(end_utc) OVER (PARTITION BY subject_id, export_id ORDER BY start_utc))::int AS gap_s
    FROM working.event
    WHERE build_id = (SELECT max(build_id) FROM working.build)
)
UPDATE working.event ev SET gap_prev_s = g.gap_s FROM g WHERE ev.event_id = g.event_id;

COMMIT;

SELECT build_id, raw_count, derived_count, exception_count,
       raw_count - derived_count - exception_count AS unaccounted
FROM working.build ORDER BY build_id DESC LIMIT 1;
