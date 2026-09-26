-- Migration 0008: import provenance + geocode linkage for claims/expected schedules
-- ADR-0016 adopts the legacy expected_schedule/schedule_comparison capability: express
-- "subject claimed to be at X, device shows Y". ref.expected_schedule and ref.claim
-- already exist (0001) and hold the claim side. What was missing:
--   (a) provenance for claims that arrive from ANOTHER TOOL or by hand (owner: "much of
--       that data will come from a different tool or manually"), so every claim row can be
--       traced to the file/person that produced it — chain of custody applies to claims
--       just as it does to evidence;
--   (b) geocode linkage, because a claim arrives as ADDRESS TEXT and needs a resolved
--       coordinate before any distance comparison is meaningful.
-- The comparison itself is DERIVED and therefore a transformation (t007), not a table.
-- UP
BEGIN;

-- (a) Import provenance. One row per import run (file, paste, or manual entry session).
CREATE TABLE ref.import_batch (
    import_batch_id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind         TEXT NOT NULL CHECK (kind IN ('expected_schedule','claim','known_place')),
    source_kind  TEXT NOT NULL CHECK (source_kind IN ('file','manual','external_tool')),
    source_name  TEXT NOT NULL,               -- filename, tool name, or "ui-manual-entry"
    source_sha256 CHAR(64),                   -- hash of the imported file when there is one
    row_count    INTEGER NOT NULL DEFAULT 0,
    imported_by  TEXT NOT NULL DEFAULT 'owner',
    imported_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes        TEXT
);
COMMENT ON TABLE ref.import_batch IS
  'Provenance for owner-supplied claim data (ADR-0016). Claims are assertions to be tested '
  'against evidence, never evidence themselves — but their origin must still be traceable.';

ALTER TABLE ref.expected_schedule
    ADD COLUMN IF NOT EXISTS import_batch_id INTEGER REFERENCES ref.import_batch(import_batch_id),
    -- Resolved coordinate for claimed_location. NULL until geocoded; the comparison in
    -- t007 skips unresolved rows rather than silently treating them as matching.
    ADD COLUMN IF NOT EXISTS location_key_id BIGINT REFERENCES geo.location_key(location_key_id),
    ADD COLUMN IF NOT EXISTS geocode_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (geocode_status IN ('pending','resolved','failed','manual')),
    ADD COLUMN IF NOT EXISTS claimed_address TEXT,   -- structured/normalized form of claimed_location
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE ref.claim
    ADD COLUMN IF NOT EXISTS import_batch_id INTEGER REFERENCES ref.import_batch(import_batch_id),
    ADD COLUMN IF NOT EXISTS location_key_id BIGINT REFERENCES geo.location_key(location_key_id),
    ADD COLUMN IF NOT EXISTS geocode_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (geocode_status IN ('pending','resolved','failed','manual')),
    ADD COLUMN IF NOT EXISTS claimed_address TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_expsched_window ON ref.expected_schedule (window_start, window_end);
CREATE INDEX IF NOT EXISTS idx_expsched_geocode ON ref.expected_schedule (geocode_status)
    WHERE geocode_status <> 'resolved';
CREATE INDEX IF NOT EXISTS idx_expsched_geom ON ref.expected_schedule USING gist (geom);
CREATE INDEX IF NOT EXISTS idx_claim_window ON ref.claim USING gist (claim_window);
CREATE INDEX IF NOT EXISTS idx_claim_geom ON ref.claim USING gist (geom);

COMMIT;

-- DOWN (manual, destructive — drops owner-entered claim provenance):
--   ALTER TABLE ref.claim DROP COLUMN import_batch_id, DROP COLUMN location_key_id,
--     DROP COLUMN geocode_status, DROP COLUMN claimed_address, DROP COLUMN updated_at;
--   ALTER TABLE ref.expected_schedule DROP COLUMN import_batch_id, DROP COLUMN location_key_id,
--     DROP COLUMN geocode_status, DROP COLUMN claimed_address, DROP COLUMN updated_at;
--   DROP TABLE ref.import_batch;
