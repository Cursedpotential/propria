-- Migration 0006: pre-ingest JSON validation + correction audit (owner-requested)
-- Every source file is validated BEFORE ingest; malformed files may be repair-attempted,
-- but ALL corrections are logged and traced (chain of custody). The original bytes are
-- never modified — repairs produce a separate file with its own hash. Repaired files are
-- flagged and require human review before their data is treated as court evidence.
-- UP
BEGIN;

CREATE TABLE raw.ingest_validation (
    validation_id  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_path    TEXT NOT NULL,
    original_sha256 CHAR(64) NOT NULL,      -- hash of the ORIGINAL bytes (evidence)
    byte_size      BIGINT NOT NULL,
    status         TEXT NOT NULL CHECK (status IN ('valid','repaired','unparseable')),
    valid_as_is    BOOLEAN NOT NULL,
    was_repaired   BOOLEAN NOT NULL DEFAULT false,
    parse_error    TEXT,                    -- stdlib JSONDecodeError detail if it failed
    repair_notes   TEXT,                    -- what the repair changed / summary of corrections
    repaired_path  TEXT,                    -- separate repaired file (original untouched)
    repaired_sha256 CHAR(64),
    record_count   INTEGER,                 -- records found after successful parse
    section_counts JSONB,                   -- per top-level section counts
    requires_human_review BOOLEAN NOT NULL DEFAULT false,  -- true for any repaired file
    reviewed_by    TEXT,
    reviewed_at    TIMESTAMPTZ,
    chunk_id       INTEGER REFERENCES raw.chunk ON DELETE SET NULL,  -- set once ingested
    validated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ingval_sha ON raw.ingest_validation (original_sha256);
CREATE INDEX idx_ingval_status ON raw.ingest_validation (status);
CREATE INDEX idx_ingval_review ON raw.ingest_validation (requires_human_review) WHERE requires_human_review;

COMMIT;
-- DOWN: DROP TABLE raw.ingest_validation;
