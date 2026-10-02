-- Migration 0009: per-provider RAW source tables (owner-directed)
-- "Yes each provider and all of the raw ... SQLite databases are not raw data nor is a bunch
-- of derivative tables raw data. I've provided you with the API [responses], I provided you
-- with [the] canonical data file. That's what you need to run with."
--
-- RAW, for this project, is exactly three things:
--   1. raw_api_responses/*.json  — the actual provider API calls, full payloads (Radar)
--   2. place_id_db_REPAIRED.json — canonical Google place-details cache
--   3. radar_geocoding_master_good.csv — canonical Radar cache
-- (k-timeline.json is already ingested as raw.chunk/raw.record by migration 0001.)
--
-- NOT raw and deliberately excluded: legacy SQLite caches, geocoding_cache*.json kv caches,
-- derived/enriched CSVs, and the many duplicate copies of the above scattered in the corpus.
-- geo.legacy_radar_cache collapsed four shapes into one JSONB column, which is why nothing
-- there could be diffed or trusted; these tables keep each provider's raw form readable.
--
-- Source files are read-only; loaders never modify them (ADR-0003/0005).
-- UP
BEGIN;

CREATE SCHEMA IF NOT EXISTS src;
COMMENT ON SCHEMA src IS
  'Verbatim raw provider sources: API responses + canonical caches. One table per provider '
  'shape, nothing normalized or merged here — normalization happens downstream in geo.* '
  '(ADR-0011). SQLite/derivative artifacts are out of scope by owner ruling.';

-- Provenance for each loaded artifact (hash = chain of custody).
CREATE TABLE src.source_file (
    source_id     INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_sha256 CHAR(64) NOT NULL,
    kind          TEXT NOT NULL CHECK (kind IN
                    ('google_place_cache','radar_csv','radar_api_batch')),
    canonical_path TEXT NOT NULL,
    byte_size     BIGINT NOT NULL,
    row_count     INTEGER NOT NULL DEFAULT 0,
    file_mtime    TIMESTAMPTZ,
    loaded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes         TEXT,
    UNIQUE (source_sha256, kind)
);

-- GOOGLE: place-details cache, dict keyed by place_id.
-- Covers all 643 distinct place_ids in the corpus; 28 entries are empty shells (no fields),
-- which is the only real gap and is fillable with ~28 free Place Details calls.
CREATE TABLE src.google_place_cache (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_sha256 CHAR(64) NOT NULL,
    place_id      TEXT NOT NULL,
    name          TEXT,
    address       TEXT,
    url           TEXT,
    types         JSONB,                    -- Google place category lives here
    geometry      JSONB,
    is_empty      BOOLEAN NOT NULL,         -- entry present but carries no fields
    raw_value     JSONB NOT NULL,
    UNIQUE (source_sha256, place_id)
);
CREATE INDEX idx_src_gpc_place ON src.google_place_cache (place_id);
CREATE INDEX idx_src_gpc_empty ON src.google_place_cache (is_empty) WHERE is_empty;

-- RADAR: canonical flattened cache (37 columns). Held as TEXT — the source uses empty
-- strings where a typed column needs NULL, and coercing at load time would rewrite evidence.
-- Note this file is pre-filtered to is_good_match=true with accuracy capped at 100m, so it
-- is NOT a complete record of what was geocoded; the API responses are the fuller source.
CREATE TABLE src.radar_csv (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_sha256 CHAR(64) NOT NULL,
    row_index     INTEGER NOT NULL,
    request_lat TEXT, request_lng TEXT, response_lat TEXT, response_lng TEXT,
    geocode_accuracy_meters TEXT, geocode_variance_flag TEXT,
    is_good_match TEXT, is_questionable TEXT, is_bad_match TEXT, needs_re_enrichment TEXT,
    label TEXT, label_type TEXT, layer TEXT, top_type TEXT, types TEXT,
    street_number TEXT, street TEXT, city TEXT, state TEXT, state_code TEXT,
    postal_code TEXT, formatted_address TEXT, place_label TEXT, address_label TEXT,
    distance_from_request TEXT,
    timezone_id TEXT, timezone_name TEXT, timezone_code TEXT,
    google_place_id TEXT, google_place_id_found TEXT,
    problematic_poi TEXT, problematic_notes TEXT, manually_verified TEXT, custom_label TEXT,
    batch_file TEXT, batch_timestamp TEXT, api_metadata TEXT,
    UNIQUE (source_sha256, row_index)
);
CREATE INDEX idx_src_radarcsv_coord ON src.radar_csv (request_lat, request_lng);
COMMENT ON TABLE src.radar_csv IS
  'DERIVATIVE, not a source of truth (owner ruling): a debugged/QA-checked flattening of '
  'Radar responses. Its role here is a VERIFICATION ORACLE — we reproduce this flattening '
  'ourselves from src.radar_api_batch.response_data and diff against these rows to prove '
  'our own derivation is accurate. Never treat it as the origin of a fact.';

-- RADAR: raw API calls — the only source retaining the provider's full response payload.
-- 896 batch files / 8,958 responses / 5,688 distinct request coordinates, all HTTP 200.
CREATE TABLE src.radar_api_batch (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_sha256   CHAR(64) NOT NULL,        -- hash of the batch file
    batch_file      TEXT NOT NULL,
    batch_number    INTEGER,
    batch_timestamp TEXT,
    response_index  INTEGER NOT NULL,
    called_at       TIMESTAMPTZ,
    api_type        TEXT,
    req_lat         NUMERIC(12,7),
    req_lng         NUMERIC(12,7),
    response_status SMALLINT,
    request_params  JSONB,
    response_data   JSONB,                    -- FULL provider response, verbatim
    UNIQUE (source_sha256, response_index)
);
CREATE INDEX idx_src_radarapi_coord  ON src.radar_api_batch (req_lat, req_lng);
CREATE INDEX idx_src_radarapi_status ON src.radar_api_batch (response_status);

COMMIT;

-- DOWN:  DROP SCHEMA src CASCADE;
