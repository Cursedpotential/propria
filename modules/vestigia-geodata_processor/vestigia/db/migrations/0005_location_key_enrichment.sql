-- Migration 0005: enrich geo.location_key into the full location registry (owner spec)
-- Multi-grain rounding (r3-r7), time-seen stats, enrichment tracking flag.
-- Providers stay in their own tables (geo.result_*); this registry JOINS to them.
-- UP
BEGIN;

ALTER TABLE geo.location_key
    ADD COLUMN lat_r5 NUMERIC(10,5), ADD COLUMN lng_r5 NUMERIC(10,5),
    ADD COLUMN lat_r6 NUMERIC(11,6), ADD COLUMN lng_r6 NUMERIC(11,6),
    ADD COLUMN lat_r7 NUMERIC(12,7), ADD COLUMN lng_r7 NUMERIC(12,7),
    ADD COLUMN geohash7 CHAR(7), ADD COLUMN geohash10 CHAR(10),
    ADD COLUMN exact_coord_count INTEGER NOT NULL DEFAULT 0,  -- distinct full-precision coords under this r4 key
    ADD COLUMN event_touches    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN waypoint_touches INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN visit_touches    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN total_dwell_seconds BIGINT NOT NULL DEFAULT 0,  -- summed visit durations here
    ADD COLUMN last_seen TIMESTAMPTZ,
    ADD COLUMN enrichment_status TEXT NOT NULL DEFAULT 'unenriched'
        CHECK (enrichment_status IN ('unenriched','cached','provider_looked_up','failed')),
    ADD COLUMN cached_provider_rows INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN last_enriched_at TIMESTAMPTZ;

-- backfill finer grains + geohashes from the stored geometry
UPDATE geo.location_key SET
    lat_r5 = round(ST_Y(geom)::numeric, 5), lng_r5 = round(ST_X(geom)::numeric, 5),
    lat_r6 = round(ST_Y(geom)::numeric, 6), lng_r6 = round(ST_X(geom)::numeric, 6),
    lat_r7 = round(ST_Y(geom)::numeric, 7), lng_r7 = round(ST_X(geom)::numeric, 7),
    geohash7  = ST_GeoHash(geom, 7), geohash10 = ST_GeoHash(geom, 10);

-- exact-coordinate count per key (from geo.exact_coordinate, migration 0003)
UPDATE geo.location_key lk SET exact_coord_count = s.n
FROM (SELECT location_key_id, count(*) n FROM geo.exact_coordinate GROUP BY 1) s
WHERE lk.location_key_id = s.location_key_id;

-- touch counts + dwell + first/last from the working layer (latest build)
WITH ev AS (
    SELECT lat_r4, lng_r4,
           count(*) AS ev_n,
           count(*) FILTER (WHERE event_type='visit') AS visit_n,
           (sum(EXTRACT(epoch FROM end_utc - start_utc)) FILTER (WHERE event_type='visit'))::bigint AS dwell,
           min(start_utc) AS f, max(COALESCE(end_utc,start_utc)) AS l
    FROM working.event WHERE lat_r4 IS NOT NULL
      AND build_id = (SELECT max(build_id) FROM working.build)
    GROUP BY lat_r4, lng_r4
), wp AS (
    SELECT round(lat,4) AS lat_r4, round(lng,4) AS lng_r4, count(*) AS wp_n,
           min(ts_utc) AS f, max(ts_utc) AS l
    FROM working.waypoint WHERE build_id = (SELECT max(build_id) FROM working.build)
    GROUP BY 1,2
)
UPDATE geo.location_key lk SET
    event_touches    = COALESCE(ev.ev_n,0),
    visit_touches    = COALESCE(ev.visit_n,0),
    waypoint_touches = COALESCE(wp.wp_n,0),
    total_dwell_seconds = COALESCE(ev.dwell,0),
    first_seen = COALESCE(LEAST(ev.f, wp.f), lk.first_seen),
    last_seen  = GREATEST(ev.l, wp.l)
FROM (SELECT lat_r4, lng_r4 FROM geo.location_key) k
LEFT JOIN ev USING (lat_r4, lng_r4)
LEFT JOIN wp USING (lat_r4, lng_r4)
WHERE lk.lat_r4 = k.lat_r4 AND lk.lng_r4 = k.lng_r4;

-- enrichment status from existing cache links (migration 0003)
UPDATE geo.location_key lk SET
    cached_provider_rows = s.n,
    enrichment_status = 'cached'
FROM (SELECT location_key_id, count(*) n FROM geo.legacy_cache_link GROUP BY 1) s
WHERE lk.location_key_id = s.location_key_id;

CREATE INDEX idx_lockey_enrich ON geo.location_key (enrichment_status);
CREATE INDEX idx_lockey_gh7 ON geo.location_key (geohash7);

-- convenience: registry joined to whatever provider data exists
CREATE OR REPLACE VIEW geo.location_registry AS
SELECT lk.location_key_id, lk.lat_r4, lk.lng_r4, lk.lat_r6, lk.lng_r6,
       lk.geohash8, lk.exact_coord_count,
       lk.event_touches, lk.waypoint_touches, lk.visit_touches,
       round(lk.total_dwell_seconds/3600.0, 1) AS dwell_hours,
       lk.first_seen, lk.last_seen, lk.enrichment_status, lk.cached_provider_rows,
       lpd.legacy_provider_row
FROM geo.location_key lk
LEFT JOIN geo.location_provider_data lpd USING (location_key_id);

COMMIT;

SELECT enrichment_status, count(*),
       sum(visit_touches) AS visits, round(sum(total_dwell_seconds)/3600.0) AS dwell_hrs
FROM geo.location_key GROUP BY 1 ORDER BY 2 DESC;
-- DOWN: DROP VIEW geo.location_registry; ALTER TABLE geo.location_key DROP COLUMN ... (the added cols).
