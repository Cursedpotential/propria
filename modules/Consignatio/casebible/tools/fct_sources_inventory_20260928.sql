-- fct_sources_inventory_20260928.sql - catalog load of every copy of the Family Court Toolkit's primary legal sources
-- (R2 casebible-sorted/fct-sources/primary, the desktop plugin copy, the ovh-files OpenCode-home copy, the
-- surreal-case reference rows, and a probe of the proposed B2 prefix), then the reconciliation queries the plan quotes.
--
-- Byline: Claude Code · Opus 5.5 · 2026-09-28 (agent sources-b2-sync for the "portal cut over" session)
-- Owner 2026-09-28 04:01 EDT: "Everything needs to be migrated to B2. R2 is being retired."
-- Run by fct_sources_inventory_20260928.sh on ovh-files, which copies the TSVs to /tmp/fct-sources-20260928/ in the
-- PG container first. Re-running replaces the rows for 2026-09-28 (the table holds one listing per location).
-- Plan and receipt: modules/Consignatio/docs/receipts/2026-09-28-fct-sources-r2-to-b2-plan.md

CREATE TABLE IF NOT EXISTS raw_duck.fct_sources_inventory_20260928 (
    location  text        NOT NULL CHECK (location IN ('r2', 'desktop_local', 'vps_local', 'store_reference', 'b2_probe')),
    path      text        NOT NULL,             -- relative to .../sources/primary/
    size      bigint,                           -- NULL for store rows (the store keeps the text, not the file size)
    md5       text,                             -- R2 ETag or computed; NULL where the location has none
    sha256    text,                             -- computed (local copies) or the store's own sha256; NULL for R2
    listed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (location, path)
);
COMMENT ON TABLE raw_duck.fct_sources_inventory_20260928 IS
  'Family Court Toolkit primary sources: one row per copy (R2 fct-sources/primary, desktop plugin, ovh-files OpenCode-home plugin, surreal-case reference rows, proposed B2 prefix). Script casebible/tools/fct_sources_inventory_20260928.{sh,sql}; plan docs/receipts/2026-09-28-fct-sources-r2-to-b2-plan.md.';

BEGIN;
DELETE FROM raw_duck.fct_sources_inventory_20260928;
CREATE TEMP TABLE stage (path text, size bigint, md5 text, sha256 text);

\copy stage FROM '/tmp/fct-sources-20260928/r2.tsv' WITH (FORMAT text)
INSERT INTO raw_duck.fct_sources_inventory_20260928 (location, path, size, md5, sha256)
SELECT 'r2', path, size, nullif(md5, ''), nullif(sha256, '') FROM stage;
TRUNCATE stage;

\copy stage FROM '/tmp/fct-sources-20260928/desktop_local.tsv' WITH (FORMAT text)
INSERT INTO raw_duck.fct_sources_inventory_20260928 (location, path, size, md5, sha256)
SELECT 'desktop_local', path, size, md5, sha256 FROM stage;
TRUNCATE stage;

\copy stage FROM '/tmp/fct-sources-20260928/vps_local.tsv' WITH (FORMAT text)
INSERT INTO raw_duck.fct_sources_inventory_20260928 (location, path, size, md5, sha256)
SELECT 'vps_local', path, size, md5, sha256 FROM stage;
TRUNCATE stage;

\copy stage FROM '/tmp/fct-sources-20260928/store_reference.tsv' WITH (FORMAT text)
INSERT INTO raw_duck.fct_sources_inventory_20260928 (location, path, size, md5, sha256)
SELECT 'store_reference', path, NULL, NULL, sha256 FROM stage;
TRUNCATE stage;

CREATE TEMP TABLE stage2 (path text, size bigint);
\copy stage2 FROM '/tmp/fct-sources-20260928/b2_probe.tsv' WITH (FORMAT text)
INSERT INTO raw_duck.fct_sources_inventory_20260928 (location, path, size)
SELECT 'b2_probe', path, size FROM stage2;
COMMIT;

-- Reconciliation (quoted in the plan)
\echo '== rows per location'
SELECT location, count(*) AS objects, sum(size) AS bytes FROM raw_duck.fct_sources_inventory_20260928 GROUP BY 1 ORDER BY 1;

\echo '== R2 vs desktop vs ovh-files copy, by path (md5 + size)'
WITH r AS (SELECT * FROM raw_duck.fct_sources_inventory_20260928 WHERE location = 'r2'),
     d AS (SELECT * FROM raw_duck.fct_sources_inventory_20260928 WHERE location = 'desktop_local'),
     v AS (SELECT * FROM raw_duck.fct_sources_inventory_20260928 WHERE location = 'vps_local')
SELECT count(*) FILTER (WHERE r.path IS NOT NULL AND d.path IS NOT NULL AND v.path IS NOT NULL
                          AND r.md5 = d.md5 AND d.md5 = v.md5 AND r.size = d.size AND d.size = v.size) AS identical_in_all_three,
       count(*) FILTER (WHERE d.sha256 IS DISTINCT FROM v.sha256 AND d.path IS NOT NULL AND v.path IS NOT NULL) AS desktop_vs_vps_sha256_differs,
       count(*) FILTER (WHERE r.md5 IS DISTINCT FROM d.md5 AND r.path IS NOT NULL AND d.path IS NOT NULL) AS r2_vs_desktop_md5_differs,
       count(*) FILTER (WHERE r.path IS NULL) AS missing_on_r2,
       count(*) FILTER (WHERE d.path IS NULL) AS missing_on_desktop,
       count(*) FILTER (WHERE v.path IS NULL) AS missing_on_vps
FROM r FULL JOIN d USING (path) FULL JOIN v USING (path);

\echo '== by extension (R2)'
SELECT coalesce(nullif(substring(path from '\.([^./]+)$'), ''), '(none)') AS ext, count(*), sum(size) AS bytes
FROM raw_duck.fct_sources_inventory_20260928 WHERE location = 'r2' GROUP BY 1 ORDER BY 1;

\echo '== text files whose store reference row carries a different sha256 than the desktop copy'
SELECT d.path, d.sha256 AS desktop_sha256, s.sha256 AS store_sha256
FROM raw_duck.fct_sources_inventory_20260928 d
LEFT JOIN raw_duck.fct_sources_inventory_20260928 s ON s.location = 'store_reference' AND s.path = d.path
WHERE d.location = 'desktop_local' AND d.path ~ '\.(md|html)$' AND s.sha256 IS DISTINCT FROM d.sha256;

\echo '== PDFs and files with no store row at all'
SELECT d.path, d.size FROM raw_duck.fct_sources_inventory_20260928 d
WHERE d.location = 'desktop_local'
  AND NOT EXISTS (SELECT 1 FROM raw_duck.fct_sources_inventory_20260928 s WHERE s.location = 'store_reference' AND s.path = d.path)
ORDER BY 1;

\echo '== contents already stored on B2 (b2_content, md5 + size) - bytes-once check'
SELECT count(DISTINCT (d.md5, d.size)) AS already_on_b2, sum(d.size) AS bytes, string_agg(DISTINCT c.origin, ',') AS origins
FROM raw_duck.fct_sources_inventory_20260928 d JOIN raw_duck.b2_content c ON c.md5 = d.md5 AND c.size = d.size
WHERE d.location = 'desktop_local';
