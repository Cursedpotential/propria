#!/usr/bin/env python3
# Byline: Claude Code · Sonnet · 2026-10-02
"""Scrambled-object survey, step 3a: turn heads.ndjson into a SQL file that creates and fills
raw_duck.scramble_head_probe_20261002.

(The catalog PostgreSQL runs pg_duckdb, which refuses COPY ... FROM, so the rows go in as INSERTs.)
Usage: heads_to_sql.py heads.ndjson > heads_load.sql ; psql -f heads_load.sql ; psql -f scrambled_survey_20261002_classify.sql
"""
import json
import sys

DDL = """BEGIN;
DROP TABLE IF EXISTS raw_duck.scramble_head_probe_20261002;
CREATE TABLE raw_duck.scramble_head_probe_20261002 (
	sha1 text PRIMARY KEY,
	size bigint NOT NULL,
	name text NOT NULL,
	ext text,
	probe_class text NOT NULL,  -- marker_ok | text | no_marker_high | no_marker_low | unreadable
	entropy numeric,
	printable numeric,
	magic text,
	key_used text,              -- the catalog key whose head was read (relative to consignatio/vault/v1/)
	tries int,
	copies int,
	in_nxplel boolean,
	in_multi boolean
);
"""


def lit(value):
    if value is None:
        return "NULL"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return repr(value)
    text = str(value)
    assert "$h$" not in text
    return "$h$" + text + "$h$"


def main(path):
    print(DDL)
    batch = []
    for line in open(path, encoding="utf-8"):
        if not line.strip():
            continue
        r = json.loads(line)
        batch.append("(" + ", ".join(lit(r.get(k)) for k in (
            "sha1", "size", "name", "ext", "class", "entropy", "printable", "magic", "key_used", "tries", "copies", "in_nxplel", "in_multi")) + ")")
        if len(batch) == 500:
            print("INSERT INTO raw_duck.scramble_head_probe_20261002 VALUES\n" + ",\n".join(batch) + "\nON CONFLICT (sha1) DO NOTHING;")
            batch = []
    if batch:
        print("INSERT INTO raw_duck.scramble_head_probe_20261002 VALUES\n" + ",\n".join(batch) + "\nON CONFLICT (sha1) DO NOTHING;")
    print("COMMIT;")


if __name__ == "__main__":
    main(sys.argv[1])
