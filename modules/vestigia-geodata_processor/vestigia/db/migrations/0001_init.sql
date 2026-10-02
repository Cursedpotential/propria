-- Migration 0001: TraceIQ core schema
-- Byline: Claude Code · Fable 5 · 2026-07-24
-- Implements ADRs 0001-0014 (docs/adr/). Five namespaces:
--   raw     = evidence, verbatim, append-only (ADR-0003/0005/0006)
--   working = materialized Tier-1 canon, rebuildable (ADR-0007)
--   geo     = N-provider geocoding + provider config (ADR-0011/0014)
--   ref     = case-knowledge reference tables + parameters (ADR-0008/0012)
--   ops     = jobs, transformations, snapshots, backups (ADR-0004/0010)
-- UP

BEGIN;

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS raw;
CREATE SCHEMA IF NOT EXISTS working;
CREATE SCHEMA IF NOT EXISTS geo;
CREATE SCHEMA IF NOT EXISTS ref;
CREATE SCHEMA IF NOT EXISTS ops;

-- ============ raw: evidence layer (append-only; nothing here is ever edited) ============

CREATE TABLE raw.subject (
    subject_id   SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_name TEXT NOT NULL UNIQUE,
    role         TEXT NOT NULL CHECK (role IN ('respondent','owner','other')),
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE raw.account (
    account_id     SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_id     SMALLINT NOT NULL REFERENCES raw.subject ON DELETE RESTRICT,
    google_account TEXT NOT NULL,
    label          TEXT,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (subject_id, google_account)
);
CREATE INDEX idx_account_subject ON raw.account (subject_id);

CREATE TABLE raw.export (
    export_id          SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id         SMALLINT NOT NULL REFERENCES raw.account ON DELETE RESTRICT,
    label              TEXT NOT NULL,
    format_generation  TEXT NOT NULL CHECK (format_generation IN ('legacy_timelineObjects','modern_semanticSegments','other_geo')),
    produced_at        DATE,            -- when Google generated it
    acquired_at        DATE,            -- when owner obtained it
    acquisition_method TEXT,            -- ADR-0003 §8: built to survive admissibility challenge
    acquisition_basis  TEXT,            -- account-access basis / authorization context
    acquired_by        TEXT,
    span_estimate      DATERANGE,
    ingest_priority    SMALLINT NOT NULL DEFAULT 100,
    notes              TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_export_account ON raw.export (account_id);

CREATE TABLE raw.chunk (
    chunk_id      INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    export_id     SMALLINT NOT NULL REFERENCES raw.export ON DELETE RESTRICT,
    file_name     TEXT NOT NULL,
    file_sha256   CHAR(64) NOT NULL,
    byte_size     BIGINT NOT NULL,
    archive_path  TEXT NOT NULL,        -- content-addressed archive location (ADR-0003 §4)
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    parse_status  TEXT NOT NULL DEFAULT 'registered'
                  CHECK (parse_status IN ('registered','parsed','unparsed','failed')),
    UNIQUE (export_id, file_sha256)
);
CREATE INDEX idx_chunk_export ON raw.chunk (export_id);
CREATE INDEX idx_chunk_sha ON raw.chunk (file_sha256);

-- One row per source record, verbatim (ADR-0005). record_sha256 = SHA-256 of the exact
-- byte slice (ADR-0006); same hash in 2+ chunks = corroborating attestations.
CREATE TABLE raw.record (
    record_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    chunk_id      INTEGER NOT NULL REFERENCES raw.chunk ON DELETE RESTRICT,
    record_index  INTEGER NOT NULL,     -- position within chunk
    record_sha256 CHAR(64) NOT NULL,
    byte_start    BIGINT,
    byte_end      BIGINT,
    google_type   TEXT,                 -- exactly what the JSON declares (may be null)
    raw_json      JSONB NOT NULL,       -- verbatim
    ingested_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chunk_id, record_index)
);
CREATE INDEX idx_record_chunk ON raw.record (chunk_id);
CREATE INDEX idx_record_sha ON raw.record (record_sha256);
CREATE INDEX idx_record_gtype ON raw.record (google_type);

CREATE TABLE raw.data_loss_event (
    loss_id        SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id     SMALLINT NOT NULL REFERENCES raw.account ON DELETE RESTRICT,
    occurred_on    DATE,
    cause          TEXT NOT NULL,       -- e.g. 'google_maps_local_migration_deletion'
    affected_range DATERANGE,
    description    TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ geo: provider config + N-provider geocoding (ADR-0011/0014) ============

CREATE TABLE geo.provider (
    provider_id    SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    enabled        BOOLEAN NOT NULL DEFAULT true,
    functions      TEXT[] NOT NULL,     -- {geocode_reverse, place_details, road_snap, gps_source, ...}
    base_url       TEXT,
    call_templates JSONB NOT NULL DEFAULT '{}'::jsonb,  -- which API calls we make, options/fields
    rate_config    JSONB NOT NULL DEFAULT '{}'::jsonb,
    cost_config    JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE geo.provider_credential (
    credential_id SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider_id   SMALLINT NOT NULL REFERENCES geo.provider ON DELETE RESTRICT,
    key_name      TEXT NOT NULL,
    key_value     TEXT NOT NULL,        -- private tailnet-only DB (ADR-0014 §3)
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','rotated','dead','unknown')),
    added_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at    TIMESTAMPTZ,
    UNIQUE (provider_id, key_name)
);

CREATE TABLE geo.credential_audit (
    audit_id      INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    credential_id SMALLINT NOT NULL REFERENCES geo.provider_credential ON DELETE RESTRICT,
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    changed_by    TEXT NOT NULL,
    change        TEXT NOT NULL
);

-- deduplicated coordinate registry (cache keys per ADR-0009)
CREATE TABLE geo.location_key (
    location_key_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lat_r4          NUMERIC(9,4) NOT NULL,
    lng_r4          NUMERIC(9,4) NOT NULL,
    lat_r3          NUMERIC(8,3) NOT NULL,
    lng_r3          NUMERIC(8,3) NOT NULL,
    geohash8        CHAR(8) NOT NULL,
    geohash9        CHAR(9) NOT NULL,
    geom            geometry(Point,4326) NOT NULL,
    first_seen      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lat_r4, lng_r4)
);
CREATE INDEX idx_lockey_gh8 ON geo.location_key (geohash8);
CREATE INDEX idx_lockey_geom ON geo.location_key USING gist (geom);

CREATE TABLE geo.api_call (
    api_call_id  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider_id  SMALLINT NOT NULL REFERENCES geo.provider ON DELETE RESTRICT,
    job_id       BIGINT,                -- FK added after ops.job exists
    template     TEXT,
    called_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    request      JSONB NOT NULL,
    response     JSONB,                 -- full response persists: API responses are evidence (ADR-0003 §5)
    http_status  SMALLINT,
    cache_level  TEXT CHECK (cache_level IN ('file','db','live')),
    cost_usd     NUMERIC(8,5) DEFAULT 0
);
CREATE INDEX idx_apicall_provider_time ON geo.api_call (provider_id, called_at);

CREATE TABLE geo.geocode_request (
    request_id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    location_key_id BIGINT REFERENCES geo.location_key ON DELETE RESTRICT,
    place_id        TEXT,               -- google place_id lookups have no location_key
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','resolved','failed')),
    CHECK (location_key_id IS NOT NULL OR place_id IS NOT NULL)
);
CREATE INDEX idx_georeq_lockey ON geo.geocode_request (location_key_id);
CREATE INDEX idx_georeq_status ON geo.geocode_request (status);

-- per-provider result tables: separate, never merged (ADR-0011)
CREATE TABLE geo.result_radar (
    result_id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id   BIGINT NOT NULL REFERENCES geo.geocode_request ON DELETE RESTRICT,
    api_call_id  BIGINT REFERENCES geo.api_call,
    address      TEXT, city TEXT, state TEXT, postal_code TEXT,
    place_label  TEXT, place_category TEXT,
    confidence   TEXT,
    raw_response JSONB NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_radar_request ON geo.result_radar (request_id);

CREATE TABLE geo.result_google (
    result_id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id   BIGINT NOT NULL REFERENCES geo.geocode_request ON DELETE RESTRICT,
    api_call_id  BIGINT REFERENCES geo.api_call,
    place_id     TEXT,
    name         TEXT, address TEXT, types TEXT[],
    business_status TEXT,
    raw_response JSONB NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_google_request ON geo.result_google (request_id);
CREATE INDEX idx_google_placeid ON geo.result_google (place_id);

CREATE TABLE geo.result_geodata (
    result_id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id   BIGINT NOT NULL REFERENCES geo.geocode_request ON DELETE RESTRICT,
    api_call_id  BIGINT REFERENCES geo.api_call,
    address      TEXT, city TEXT, state TEXT,
    raw_response JSONB NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_geodata_request ON geo.result_geodata (request_id);

CREATE TABLE geo.resolution (
    resolution_id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id       BIGINT NOT NULL UNIQUE REFERENCES geo.geocode_request ON DELETE RESTRICT,
    chosen_provider  TEXT,
    resolved_address TEXT, resolved_city TEXT, resolved_label TEXT,
    disagreement     BOOLEAN NOT NULL DEFAULT false,
    manual_override  BOOLEAN NOT NULL DEFAULT false,
    resolved_by      TEXT,
    resolved_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes            TEXT
);
CREATE INDEX idx_resolution_disagreement ON geo.resolution (disagreement) WHERE disagreement;

-- road-snapping: separate storage, raw GPS untouched (ADR-0011 §3)
CREATE TABLE geo.snap_result (
    snap_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider_id SMALLINT NOT NULL REFERENCES geo.provider ON DELETE RESTRICT,
    event_ref   BIGINT NOT NULL,        -- working.event id (soft ref: working rebuilds)
    api_call_id BIGINT REFERENCES geo.api_call,
    snapped_geom geometry(LineString,4326),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- legacy 50K Radar cache imports here verbatim as evidence (ADR-0011 §1)
CREATE TABLE geo.legacy_radar_cache (
    legacy_id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_file  TEXT NOT NULL,
    row_json     JSONB NOT NULL,
    imported_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ ref: case knowledge + tunable parameters (ADR-0008/0012) ============

CREATE TABLE ref.parameter (
    param_name   TEXT PRIMARY KEY,
    value        JSONB NOT NULL,
    description  TEXT NOT NULL,         -- judge-readable meaning
    plausible_range TEXT,               -- for sensitivity checks
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ref.parameter_audit (
    audit_id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    param_name TEXT NOT NULL REFERENCES ref.parameter ON DELETE RESTRICT,
    old_value  JSONB,
    new_value  JSONB NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    changed_by TEXT NOT NULL
);

CREATE TABLE ref.home_base (
    homebase_id  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_id   SMALLINT NOT NULL REFERENCES raw.subject ON DELETE RESTRICT,
    label        TEXT NOT NULL,
    base_type    TEXT NOT NULL CHECK (base_type IN ('home','work','secondary','other')),
    geom         geometry(Point,4326) NOT NULL,
    radius_m     INTEGER NOT NULL DEFAULT 150,
    confidence   NUMERIC(4,3),
    source       TEXT NOT NULL DEFAULT 'derived' CHECK (source IN ('derived','owner_confirmed')),
    visit_count  INTEGER,
    typical_arrival  TIME, typical_departure TIME,
    days_of_week SMALLINT[],
    active_range DATERANGE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_homebase_subject ON ref.home_base (subject_id);
CREATE INDEX idx_homebase_geom ON ref.home_base USING gist (geom);

CREATE TABLE ref.expected_schedule (
    schedule_id  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_id   SMALLINT NOT NULL REFERENCES raw.subject ON DELETE RESTRICT,
    source_doc   TEXT NOT NULL,         -- custody order / statement / text ref
    claimed_location TEXT NOT NULL,
    geom         geometry(Point,4326),
    geofence_radius_m INTEGER NOT NULL DEFAULT 200,
    window_start TIMESTAMPTZ NOT NULL,
    window_end   TIMESTAMPTZ NOT NULL,
    time_tolerance_min INTEGER NOT NULL DEFAULT 30,
    recurring    TEXT,                  -- rrule-ish, null = one-off
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sched_subject_window ON ref.expected_schedule (subject_id, window_start);

CREATE TABLE ref.problematic_location (
    problem_id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label        TEXT NOT NULL,
    geom         geometry(Point,4326) NOT NULL,
    alert_radius_m INTEGER NOT NULL DEFAULT 200,
    severity     SMALLINT NOT NULL DEFAULT 1 CHECK (severity BETWEEN 1 AND 5),
    legal_restriction TEXT,
    active       BOOLEAN NOT NULL DEFAULT true,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_problem_geom ON ref.problematic_location USING gist (geom);

CREATE TABLE ref.claim (
    claim_id     INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_id   SMALLINT NOT NULL REFERENCES raw.subject ON DELETE RESTRICT,
    claim_text   TEXT NOT NULL,
    source_doc   TEXT NOT NULL,
    claimed_location TEXT,
    geom         geometry(Point,4326),
    claim_window TSTZRANGE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ ops: jobs, transformations, snapshots, backups (ADR-0004/0010) ============

CREATE TABLE ops.job (
    job_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_type   TEXT NOT NULL,
    params     JSONB NOT NULL DEFAULT '{}'::jsonb,
    status     TEXT NOT NULL DEFAULT 'queued'
               CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
    queued_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ, finished_at TIMESTAMPTZ,
    log        TEXT, error TEXT
);
CREATE INDEX idx_job_status ON ops.job (status);

ALTER TABLE geo.api_call
    ADD CONSTRAINT fk_apicall_job FOREIGN KEY (job_id) REFERENCES ops.job (job_id);

CREATE TABLE ops.transformation (
    name        TEXT NOT NULL,
    version     INTEGER NOT NULL,
    description TEXT NOT NULL,          -- judge-readable summary; full doc + example in repo
    doc_path    TEXT NOT NULL,          -- docs/transformations/<name>.md
    sql_path    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (name, version)
);

CREATE TABLE ops.snapshot (
    snapshot_id  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label        TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    file_path    TEXT NOT NULL,
    file_sha256  CHAR(64) NOT NULL,
    parameter_values JSONB NOT NULL,    -- exhibits self-document (ADR-0008 §3)
    definition_versions JSONB NOT NULL
);

CREATE TABLE ops.backup (
    backup_id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label       TEXT NOT NULL,
    dumped_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    file_path   TEXT NOT NULL,
    file_sha256 CHAR(64) NOT NULL,
    verified    BOOLEAN NOT NULL DEFAULT false
);

-- ============ working: materialized Tier-1 canon (ADR-0007; rebuildable, never hand-edited) ============

CREATE TABLE working.build (
    build_id    INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    job_id      BIGINT REFERENCES ops.job,
    transform_versions JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_count   BIGINT, derived_count BIGINT, exception_count BIGINT   -- reconciliation (ADR-0005)
);

CREATE TABLE working.event (
    event_id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    build_id    INTEGER NOT NULL REFERENCES working.build ON DELETE CASCADE,
    record_id   BIGINT NOT NULL REFERENCES raw.record ON DELETE RESTRICT,
    subject_id  SMALLINT NOT NULL,
    account_id  SMALLINT NOT NULL,
    export_id   SMALLINT NOT NULL,
    event_type  TEXT NOT NULL CHECK (event_type IN ('visit','activity','timeline_path','point')),
    raw_type    TEXT NOT NULL,          -- breadcrumb: what Google called it (trip stays visible)
    serial_display TEXT,                -- human-readable sortable label (ADR-0006 §3)
    start_utc   TIMESTAMPTZ, end_utc TIMESTAMPTZ,
    tz_name     TEXT, start_local TIMESTAMP, end_local TIMESTAMP,
    start_eastern TIMESTAMP, end_eastern TIMESTAMP,
    path_time_source TEXT CHECK (path_time_source IN ('points_first_last','adjacent_events','container_fallback')),
    lat NUMERIC(9,6), lng NUMERIC(9,6),
    geom geometry(Point,4326),
    lat_r3 NUMERIC(8,3), lng_r3 NUMERIC(8,3),
    lat_r4 NUMERIC(9,4), lng_r4 NUMERIC(9,4),
    geohash8 CHAR(8), geohash9 CHAR(9),
    place_id TEXT,
    probability NUMERIC(5,4),
    distance_m_google NUMERIC(12,1),
    distance_m_haversine NUMERIC(12,1),
    distance_discrepancy BOOLEAN NOT NULL DEFAULT false,
    gap_prev_s  INTEGER,
    overnight_simple BOOLEAN NOT NULL DEFAULT false,
    overnight_type TEXT,
    multi_device_split BOOLEAN NOT NULL DEFAULT false,
    device_index SMALLINT NOT NULL DEFAULT 0,
    split_from_event BIGINT,
    orphan_adopted BOOLEAN NOT NULL DEFAULT false,
    parent_event_id BIGINT,
    anomaly_flags TEXT[] NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_wevent_record ON working.event (record_id);
CREATE INDEX idx_wevent_subject_time ON working.event (subject_id, start_utc);
CREATE INDEX idx_wevent_type_time ON working.event (event_type, start_utc);
CREATE INDEX idx_wevent_geom ON working.event USING gist (geom);
CREATE INDEX idx_wevent_r4 ON working.event (lat_r4, lng_r4);
CREATE INDEX idx_wevent_overnight ON working.event (overnight_simple) WHERE overnight_simple;
CREATE INDEX idx_wevent_placeid ON working.event (place_id);

CREATE TABLE working.waypoint (
    waypoint_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    build_id    INTEGER NOT NULL REFERENCES working.build ON DELETE CASCADE,
    event_id    BIGINT NOT NULL REFERENCES working.event ON DELETE CASCADE,
    seq         INTEGER NOT NULL,       -- ordering key; NEVER order by timestamp (ADR-0005)
    ts_utc      TIMESTAMPTZ,
    lat NUMERIC(9,6) NOT NULL, lng NUMERIC(9,6) NOT NULL,
    geom geometry(Point,4326) NOT NULL,
    device_index SMALLINT NOT NULL DEFAULT 0,
    speed_mps   NUMERIC(7,2),
    gap_prev_s  INTEGER,
    dwell_candidate BOOLEAN NOT NULL DEFAULT false,   -- intra-path unrecorded-stop signal (T11 v2)
    UNIQUE (event_id, device_index, seq)
);
CREATE INDEX idx_wwp_event ON working.waypoint (event_id);
CREATE INDEX idx_wwp_geom ON working.waypoint USING gist (geom);
CREATE INDEX idx_wwp_dwell ON working.waypoint (dwell_candidate) WHERE dwell_candidate;

CREATE TABLE working.exception (
    exception_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    build_id    INTEGER NOT NULL REFERENCES working.build ON DELETE CASCADE,
    record_id   BIGINT NOT NULL REFERENCES raw.record ON DELETE RESTRICT,
    reason      TEXT NOT NULL,
    detail      JSONB
);
CREATE INDEX idx_wexc_build ON working.exception (build_id);

-- ============ seed: parameters (recovered historical defaults) + providers ============

INSERT INTO ref.parameter (param_name, value, description, plausible_range) VALUES
 ('overnight_window',      '{"start":"22:00","end":"07:00"}', 'Local-time window an event must span to flag overnight_simple', '21:00-23:00 start, 06:00-08:00 end'),
 ('multi_device_min_m',    '100',  'Min meters between same-timestamp waypoints to declare physically-impossible dual device', '50-200 (urban vs rural)'),
 ('impossible_speed_mph',  '120',  'Speed above which travel is flagged impossible', '100-200'),
 ('dwell_threshold_min',   '15',   'Min minutes stationary inside a moving path to flag an unrecorded-stop candidate', '10-30'),
 ('cluster_radius_m',      '200',  'Radius for home-base/frequent-place candidate clustering', '100-500'),
 ('path_anchor_tolerance_min', '5','Max minutes to bookend a path against adjacent visit/activity times', '3-10');

INSERT INTO geo.provider (name, enabled, functions, notes) VALUES
 ('radar',   true,  '{geocode_reverse}',              'Primary reverse geocoder (ADR-0011); legacy 50K cache imports to geo.legacy_radar_cache'),
 ('google',  true,  '{place_details}',                'Fallback: place_id lookups + business names ONLY, never primary reverse geocode'),
 ('geodata', true,  '{geocode_reverse}',              'Third source: international/backup coverage'),
 ('osrm',    false, '{road_snap}',                    'Road-snapping candidate: free, self-hostable (enable after Phase C eval)'),
 ('valhalla',false, '{road_snap}',                    'Road-snapping candidate: free, self-hostable'),
 ('mapbox',  false, '{road_snap}',                    'Road-snapping candidate: generous free tier');

COMMIT;

-- DOWN
-- BEGIN; DROP SCHEMA working CASCADE; DROP SCHEMA ops CASCADE; DROP SCHEMA ref CASCADE;
-- DROP SCHEMA geo CASCADE; DROP SCHEMA raw CASCADE; COMMIT;
