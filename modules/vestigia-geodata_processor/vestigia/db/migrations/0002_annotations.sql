-- Migration 0002: owner annotation layer (labels, flags, place names)
-- Byline: Claude Code · Fable 5 · 2026-07-24
-- Owner-writable, audited; never touches raw. Views join these to show labels.
-- UP
BEGIN;

-- Owner names/categories for places (until + alongside geocoding)
CREATE TABLE ref.place_label (
    place_label_id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    place_id     TEXT,                      -- Google place id, when known
    lat_r4       NUMERIC(9,4), lng_r4 NUMERIC(9,4),   -- fallback key when no place_id
    label        TEXT NOT NULL,             -- "Her mom's house"
    business_name TEXT,
    category     TEXT,                      -- residence / bar / work / school / ...
    concerning   BOOLEAN NOT NULL DEFAULT false,
    severity     SMALLINT CHECK (severity BETWEEN 1 AND 5),
    note         TEXT,
    created_by   TEXT NOT NULL DEFAULT 'owner',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (place_id IS NOT NULL OR (lat_r4 IS NOT NULL AND lng_r4 IS NOT NULL))
);
CREATE INDEX idx_placelabel_place ON ref.place_label (place_id);
CREATE INDEX idx_placelabel_r4 ON ref.place_label (lat_r4, lng_r4);

-- Generic flags/notes on anything (event, waypoint, day, place, route)
CREATE TABLE ref.annotation (
    annotation_id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    target_kind  TEXT NOT NULL CHECK (target_kind IN ('event','waypoint','day','place','route','other')),
    target_ref   TEXT NOT NULL,             -- serial_display / place_id / YYYY-MM-DD / free key
    category     TEXT NOT NULL,             -- anomaly_confirmed / concerning / question / exhibit_candidate / dismissed / custom
    label        TEXT,
    note         TEXT,
    created_by   TEXT NOT NULL DEFAULT 'owner',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_annotation_target ON ref.annotation (target_kind, target_ref);
CREATE INDEX idx_annotation_cat ON ref.annotation (category);

COMMIT;
-- DOWN: BEGIN; DROP TABLE ref.annotation; DROP TABLE ref.place_label; COMMIT;
