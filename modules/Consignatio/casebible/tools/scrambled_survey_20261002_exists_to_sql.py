#!/usr/bin/env python3
# Byline: Claude Code · Sonnet · 2026-10-02
"""Scrambled-object survey, step 4b: turn exists.tsv into SQL that fills the existence columns of
raw_duck.scrambled_objects_20261002 and prints the dry-run totals.

exists_in_b2 is true only when B2 holds a visible object at that exact key whose size AND SHA-1 equal the
catalog's (so the key still holds the scrambled bytes the survey saw, not something replaced since).
Usage: exists_to_sql.py exists.tsv > exists_load.sql ; psql -f exists_load.sql
"""
import sys

print("""BEGIN;
ALTER TABLE raw_duck.scrambled_objects_20261002
	ADD COLUMN IF NOT EXISTS b2_file_id text, ADD COLUMN IF NOT EXISTS b2_sha1 text, ADD COLUMN IF NOT EXISTS b2_size bigint;
CREATE TEMP TABLE b2_exists (full_key text PRIMARY KEY, found boolean, b2_size bigint, b2_file_id text, b2_sha1 text);""")
batch = []


def flush():
    global batch
    if batch:
        print("INSERT INTO b2_exists VALUES\n" + ",\n".join(batch) + "\nON CONFLICT DO NOTHING;")
        batch = []


for line in open(sys.argv[1], encoding="utf-8"):
    key, found, size, fid, sha = (line.rstrip("\n").split("\t") + ["", "", "", "", ""])[:5]
    assert "$k$" not in key
    batch.append(f"($k${key}$k$, {'true' if found == '1' else 'false'}, {size if size not in ('', '-1') else 'NULL'}, "
                 f"{('$k$' + fid + '$k$') if fid else 'NULL'}, {('$k$' + sha + '$k$') if sha else 'NULL'})")
    if len(batch) == 500:
        flush()
flush()
print("""UPDATE raw_duck.scrambled_objects_20261002 s
SET b2_file_id = e.b2_file_id, b2_sha1 = e.b2_sha1, b2_size = e.b2_size,
	exists_in_b2 = (e.found AND e.b2_size = s.size AND lower(coalesce(e.b2_sha1, '')) = lower(s.sha1))
FROM b2_exists e WHERE e.full_key = s.full_key;
UPDATE raw_duck.scrambled_objects_20261002 SET exists_in_b2 = false WHERE exists_in_b2 IS NULL;
COMMIT;

SELECT status, exists_in_b2, count(*) AS keys, count(DISTINCT sha1) AS hashes, sum(size) AS bytes,
	count(*) FILTER (WHERE twin_key IS NOT NULL) AS with_twin, count(*) FILTER (WHERE in_nxplel) AS in_nxplel
FROM raw_duck.scrambled_objects_20261002 GROUP BY 1, 2 ORDER BY 1, 2;""")
