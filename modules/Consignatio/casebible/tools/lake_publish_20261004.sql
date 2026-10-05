-- lake_publish_20261004.sql - catalog record of the 2026-10-04 lake publish
-- (B2 bucket salem-data, prefix consignatio/_system/lake/2026-10-04/ plus the pointer consignatio/_system/lake/LATEST).
--
-- Byline: Codex, 2026-10-04. Generation ledger derived from the September publisher.
-- Historical source: Claude Code, Opus 5.5, 2026-09-27; owner B2 publication decision 2026-09-27.
-- Inputs: validated manifest rows; outputs: publication ledger; effects: transactionally admits verified artifacts.
-- Choose for this authorized generation, not inventory loading or evidence promotion.
-- The export manifest lands in the catalog itself (the catalog is the source of truth); the same rows are
-- published as manifest.csv beside the Parquet files.
--
-- Applied by lake_publish_20261004.sh finalize, only after upload, rclone check and the readback
-- (sha256 of the bytes downloaded from B2 = sha256 at export; Parquet rows read back = PG rows).
-- The script appends one INSERT per published object in a single transaction; the primary key refuses a second load.

CREATE TABLE IF NOT EXISTS raw_duck.lake_publish_20261004 (
    table_name    text        NOT NULL,             -- raw_duck table exported, or the file name of a non-table object
    rows          bigint      NOT NULL,             -- PG rows = Parquet rows read back from B2; CSV data rows; tables in schema.json; 1 for LATEST
    parquet_bytes bigint      NOT NULL,             -- byte size of the published object (non-Parquet objects: their file size)
    b2_key        text        PRIMARY KEY,          -- object key inside bucket salem-data
    sha256        text        NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    published_at  timestamptz NOT NULL,             -- when the upload passed rclone check
    status        text        NOT NULL,             -- current | lineage | bridge | ... (casebible/tools/lake_publish_20261004.tables.txt)
    object_kind   text        NOT NULL CHECK (object_kind IN ('parquet', 'csv', 'json', 'text')),
    recorded_at   timestamptz NOT NULL DEFAULT now(),
    script        text        NOT NULL DEFAULT 'casebible/tools/lake_publish_20261004.sh'
);

COMMENT ON TABLE raw_duck.lake_publish_20261004 IS
  'Lake publish 2026-10-04: every object written to b2://salem-data/consignatio/_system/lake/ (Parquet export of the current raw_duck catalog tables, corrupt_missing.csv, schema.json, manifest.csv, LATEST), with rows, bytes and sha256 verified by readback. Script casebible/tools/lake_publish_20261004.sh; log docs/LOG.md 2026-10-04.';
