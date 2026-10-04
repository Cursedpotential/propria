-- Coco Super Index object list (DuckDB dialect; the catalog is attached as `catalog`).
-- Byline: Claude Code · Sonnet 5.5 · 2026-10-02 (replaces catalog_source.sql as the default; that file read the
-- 2026-09-22 static snapshot and never saw a new listing).
-- Must return key, size, sha1. Every other column rides along into the index as catalog fields
-- (catalog_source.py); widen the index by editing this query, not the code.
-- View built by Consignatio/casebible/tools/superindex_source_current_20261003.sql.
--
-- `resolution`: the catalog has no column of that name. The nearest recorded value is the first occurrence's
-- `disposition` (e.g. "content_on_b2"). It is passed through unchanged, never invented.
SELECT
    key,
    size,
    sha1,
    provider,
    bucket,
    name,
    md5,
    modtime,
    listed_at,
    occurrence_count,
    occurrences,
    COALESCE(json_extract_string(occurrences, '$[0].disposition'), 'unknown') AS resolution
FROM catalog.raw_duck.superindex_source_current
