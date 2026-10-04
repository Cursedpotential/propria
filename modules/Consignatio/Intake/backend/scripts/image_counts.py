"""Read-only catalog counts for planning the image stage: unique images by SHA-1,
screenshot-named, PDFs, and HEIC.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Run on the VPS (or anywhere that can reach the catalog):
``python scripts/image_counts.py``. It reads ``raw_duck.bucket_objects_current``
through ``INTAKE_CATALOG_DSN`` (the same view the discovery stage reads), prints
counts and bytes as JSON, and never prints the DSN. The answer feeds the cost
table in the runbook: tokens, requests, and Weaviate storage per slot are per
UNIQUE file, and the catalog lists every occurrence.
"""

from __future__ import annotations

import json

import duckdb

from casebible_index.catalog_source import _attach
from casebible_index.secrets import get_secret

SQL = r"""
WITH o AS (
  SELECT provider, bucket, key, size, sha1, lower(regexp_extract(key, '(\.[^./]+)$', 1)) AS ext,
         CASE WHEN coalesce(sha1,'') <> '' THEN sha1
              ELSE 'k-' || md5(
                provider||'|'||bucket||'|'||key||'|'||CAST(size AS VARCHAR)
              ) END AS identity,
         regexp_matches(lower(key), 'screen ?shot|screen_shot|screen-shot|screencap') AS shotname
  FROM catalog.raw_duck.bucket_objects_current
  WHERE NOT regexp_matches(key, '(^|/)(\.git|\.review_hold|to_be_deleted|__pycache__)(/|$)')
), img AS (
  SELECT * FROM o
  WHERE ext IN ('.png','.jpg','.jpeg','.webp','.gif','.bmp','.tif','.tiff')
)
SELECT 'images_processable' AS what, count(*) AS occurrences,
       count(DISTINCT identity) AS unique_files,
       (
         SELECT sum(s) FROM (SELECT max(size) s FROM img GROUP BY identity)
       )::BIGINT AS unique_bytes FROM img
UNION ALL SELECT 'images_screenshot_named', count(*), count(DISTINCT identity),
       (
         SELECT sum(s)
         FROM (SELECT max(size) s FROM img WHERE shotname GROUP BY identity)
       )::BIGINT FROM img WHERE shotname
UNION ALL
SELECT 'images_over_25_MiB', count(*), count(DISTINCT identity), sum(size)::BIGINT
FROM img WHERE size > 26214400
UNION ALL
SELECT 'heic_heif_not_processed', count(*), count(DISTINCT identity), sum(size)::BIGINT
FROM o WHERE ext IN ('.heic','.heif')
UNION ALL
SELECT 'pdf', count(*), count(DISTINCT identity), sum(size)::BIGINT
FROM o WHERE ext = '.pdf'
"""


def main() -> None:
    dsn = get_secret("INTAKE_CATALOG_DSN")
    if not dsn:
        raise SystemExit("INTAKE_CATALOG_DSN is not configured")
    con = duckdb.connect()
    try:
        _attach(con, dsn)
        rows = con.execute(SQL).fetchall()
    finally:
        con.close()
    summary = [
        dict(zip(("what", "occurrences", "unique_files", "unique_bytes"), row, strict=True))
        for row in rows
    ]
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
