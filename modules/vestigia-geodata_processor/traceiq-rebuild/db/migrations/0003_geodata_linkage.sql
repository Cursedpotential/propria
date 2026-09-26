-- Migration 0003: rounded↔full geodata linkage + provider links + unique Google IDs
-- Byline: Claude Code · Fable 5 · 2026-07-24 (owner-specified structure)
-- UP
BEGIN;

-- Full-precision coordinates, each linked to its rounded location_key (owner ask #1)
CREATE TABLE geo.exact_coordinate (
    exact_id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lat             NUMERIC(9,6) NOT NULL,
    lng             NUMERIC(9,6) NOT NULL,
    location_key_id BIGINT NOT NULL REFERENCES geo.location_key ON DELETE RESTRICT,
    event_touches    INTEGER NOT NULL DEFAULT 0,
    waypoint_touches INTEGER NOT NULL DEFAULT 0,
    first_seen TIMESTAMPTZ, last_seen TIMESTAMPTZ,
    UNIQUE (lat, lng)
);
CREATE INDEX idx_exact_lockey ON geo.exact_coordinate (location_key_id);

-- Legacy provider-cache rows matched to rounded keys (owner ask #2)
CREATE TABLE geo.legacy_cache_link (
    legacy_id       BIGINT NOT NULL REFERENCES geo.legacy_radar_cache ON DELETE RESTRICT,
    location_key_id BIGINT NOT NULL REFERENCES geo.location_key ON DELETE RESTRICT,
    match_method    TEXT NOT NULL,
    PRIMARY KEY (legacy_id, location_key_id)
);

COMMIT;

-- Populate exact coordinates from every source of coordinates
INSERT INTO geo.exact_coordinate (lat, lng, location_key_id, event_touches, waypoint_touches, first_seen, last_seen)
SELECT u.lat, u.lng, lk.location_key_id,
       sum(u.ev), sum(u.wp), min(u.t0), max(u.t1)
FROM (
    SELECT lat, lng, 1 AS ev, 0 AS wp, start_utc AS t0, end_utc AS t1 FROM working.event WHERE lat IS NOT NULL
    UNION ALL
    SELECT lat, lng, 0, 1, ts_utc, ts_utc FROM working.waypoint
) u
JOIN geo.location_key lk ON lk.lat_r4 = round(u.lat,4) AND lk.lng_r4 = round(u.lng,4)
GROUP BY u.lat, u.lng, lk.location_key_id
ON CONFLICT (lat, lng) DO NOTHING;

-- Link legacy cache rows to rounded keys (CSV: request_lat/lng; JSON: 'lat,lng' key)
INSERT INTO geo.legacy_cache_link (legacy_id, location_key_id, match_method)
SELECT l.legacy_id, lk.location_key_id, 'r4:' || split_part(l.source_file, E'\\', -1)
FROM geo.legacy_radar_cache l
JOIN geo.location_key lk ON
    lk.lat_r4 = round(COALESCE(
        NULLIF(l.row_json->>'request_lat','')::numeric,
        NULLIF(split_part(l.row_json->>'key', ',', 1),'')::numeric), 4)
    AND lk.lng_r4 = round(COALESCE(
        NULLIF(l.row_json->>'request_lng','')::numeric,
        NULLIF(trim(split_part(l.row_json->>'key', ',', 2)),'')::numeric), 4)
ON CONFLICT DO NOTHING;

-- One-stop provider-data view per rounded location (extends as live providers fill)
CREATE OR REPLACE VIEW geo.location_provider_data AS
SELECT lk.location_key_id, lk.lat_r4, lk.lng_r4, lk.geohash8,
       l.source_file, l.row_json AS legacy_provider_row
FROM geo.location_key lk
LEFT JOIN geo.legacy_cache_link lcl USING (location_key_id)
LEFT JOIN geo.legacy_radar_cache l USING (legacy_id);

-- Unique Google place IDs (owner ask #3)
CREATE OR REPLACE VIEW analysis.unique_google_place_ids AS
SELECT place_id,
       count(*) AS visits,
       count(*) FILTER (WHERE overnight_simple) AS overnights,
       min(start_eastern)::date AS first_seen, max(start_eastern)::date AS last_seen,
       min(lat_r4) AS lat_r4, min(lng_r4) AS lng_r4
FROM analysis.latest_events
WHERE place_id IS NOT NULL
GROUP BY place_id;

SELECT (SELECT count(*) FROM geo.exact_coordinate)      AS exact_coords,
       (SELECT count(*) FROM geo.legacy_cache_link)     AS legacy_links,
       (SELECT count(DISTINCT location_key_id) FROM geo.legacy_cache_link) AS keys_with_cache,
       (SELECT count(*) FROM geo.location_key)          AS location_keys,
       (SELECT count(*) FROM analysis.unique_google_place_ids) AS unique_google_ids;
