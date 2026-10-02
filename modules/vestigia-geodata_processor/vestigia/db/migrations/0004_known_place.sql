-- Migration 0004: unified known_place table (owner-specified, 2026-07-24 morning)
-- One owner-knowledge table for places: multi-tag + concerning/watch flags.
-- Absorbs ref.place_label (empty) + ref.problematic_location (empty); ref.home_base
-- stays as DERIVED analytics output checked against these human labels.
-- UP
BEGIN;

CREATE TABLE ref.known_place (
    known_place_id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    place_id     TEXT,                              -- Google place id when known
    lat_r4       NUMERIC(9,4), lng_r4 NUMERIC(9,4), -- coordinate key fallback/addition
    label        TEXT NOT NULL,                     -- "Her mom's house"
    business_name TEXT,
    tags         TEXT[] NOT NULL DEFAULT '{}',      -- home_base, work, relative, school,
                                                    -- personal_interest, bar, concerning...
    is_concerning BOOLEAN NOT NULL DEFAULT false,
    is_watch      BOOLEAN NOT NULL DEFAULT false,   -- keep an eye on it
    severity      SMALLINT CHECK (severity BETWEEN 1 AND 5),
    alert_radius_m INTEGER,                         -- for proximity/violation views
    legal_restriction TEXT,                         -- order language if applicable
    active_range  DATERANGE,                        -- e.g. residence periods
    note          TEXT,
    created_by    TEXT NOT NULL DEFAULT 'owner',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (place_id IS NOT NULL OR (lat_r4 IS NOT NULL AND lng_r4 IS NOT NULL))
);
CREATE INDEX idx_knownplace_place ON ref.known_place (place_id);
CREATE INDEX idx_knownplace_r4 ON ref.known_place (lat_r4, lng_r4);
CREATE INDEX idx_knownplace_tags ON ref.known_place USING gin (tags);
CREATE INDEX idx_knownplace_concern ON ref.known_place (is_concerning) WHERE is_concerning;

INSERT INTO ref.known_place (place_id, lat_r4, lng_r4, label, business_name,
       tags, is_concerning, severity, note, created_by, created_at)
SELECT place_id, lat_r4, lng_r4, label, business_name,
       CASE WHEN category IS NOT NULL THEN ARRAY[category] ELSE '{}' END,
       concerning, severity, note, created_by, created_at
FROM ref.place_label;

DROP VIEW IF EXISTS analysis.place_analytics;
CREATE VIEW analysis.place_analytics AS
SELECT e.subject_id, e.place_id, e.lat_r4, e.lng_r4,
       kp.label AS owner_label, kp.business_name, kp.tags,
       COALESCE(kp.is_concerning, false) AS concerning,
       COALESCE(kp.is_watch, false) AS watch,
       count(DISTINCT e.record_id) AS visit_count,
       round(avg(EXTRACT(epoch FROM e.end_utc - e.start_utc))/60) AS avg_minutes,
       round(min(EXTRACT(epoch FROM e.end_utc - e.start_utc))/60) AS shortest_minutes,
       round(max(EXTRACT(epoch FROM e.end_utc - e.start_utc))/60) AS longest_minutes,
       min(e.start_eastern)::date AS first_visit, max(e.start_eastern)::date AS last_visit,
       count(DISTINCT e.record_id) FILTER (WHERE e.overnight_simple) AS overnight_visits,
       round(100.0 * count(DISTINCT e.record_id) FILTER (WHERE e.overnight_simple)
             / count(DISTINCT e.record_id), 1) AS overnight_pct
FROM analysis.latest_events e
LEFT JOIN ref.known_place kp
       ON (kp.place_id = e.place_id)
       OR (kp.place_id IS NULL AND kp.lat_r4 = e.lat_r4 AND kp.lng_r4 = e.lng_r4)
WHERE e.event_type = 'visit'
GROUP BY e.subject_id, e.place_id, e.lat_r4, e.lng_r4,
         kp.label, kp.business_name, kp.tags, kp.is_concerning, kp.is_watch;

-- retired-not-deleted (never-delete rule): renamed with _retired suffix
ALTER TABLE ref.place_label RENAME TO place_label_retired_0004;
ALTER TABLE ref.problematic_location RENAME TO problematic_location_retired_0004;

COMMIT;
-- DOWN: rename retired tables back, drop known_place, recreate old view.
