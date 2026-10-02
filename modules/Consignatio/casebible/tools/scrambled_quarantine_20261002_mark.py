#!/usr/bin/env python3
# Byline: Claude Code · Sonnet · 2026-10-02
"""Scrambled-object quarantine, step after the apply: mark the catalog from the apply ledger.

b2_version_ops_20261001.py keeps a SQLite ledger (table op: mode, file_id, src_key, dest_key, size, sha1, status,
new_file_id, err, ts). Rows whose status is 'ok' or 'ok+hidden' or 'skipped_same+hidden' were copied, verified
(size and SHA-1) and hidden at the source; this script turns exactly those rows into SQL that

  * sets raw_duck.scrambled_objects_20261002.status = 'quarantined' and records the new B2 file id and time;
  * sets raw_duck.vault_objects.in_quarantine = true for the same keys (the key itself is unchanged: the
    object's new place is quarantine_key in scrambled_objects_20261002).

Rows with any other status (failed, refused_dest_occupied, copied_not_hidden) are listed on stderr and left
untouched. Nothing is deleted. Usage:
  mark.py ledger_apply/ledger.sqlite [--status would_copy]   > mark.sql     (--status only for a rehearsal on the dry-run ledger)
  psql -U postgres -d casebible -v ON_ERROR_STOP=1 -f mark.sql
"""
import sqlite3
import sys

path = sys.argv[1]
statuses = ("ok", "ok+hidden", "skipped_same+hidden")
if "--status" in sys.argv:
    statuses = (sys.argv[sys.argv.index("--status") + 1],)

db = sqlite3.connect(path)
rows = db.execute("select src_key, dest_key, new_file_id, status from op where mode='quarantine'").fetchall()
done = [r for r in rows if r[3] in statuses]
other = [r for r in rows if r[3] not in statuses]
for src, dest, fid, status in other:
    print(f"NOT MARKED {status}: {src}", file=sys.stderr)

PREFIX = "consignatio/vault/v1/"
print("BEGIN;")
print("ALTER TABLE raw_duck.scrambled_objects_20261002 ADD COLUMN IF NOT EXISTS quarantined_at timestamptz, ADD COLUMN IF NOT EXISTS quarantine_file_id text;")
print("CREATE TEMP TABLE applied (full_key text PRIMARY KEY, quarantine_key text, new_file_id text);")
batch = []


def flush():
    global batch
    if batch:
        print("INSERT INTO applied VALUES\n" + ",\n".join(batch) + "\nON CONFLICT DO NOTHING;")
        batch = []


for src, dest, fid, status in done:
    assert "$k$" not in src + dest
    batch.append(f"($k${src}$k$, $k${dest}$k$, {('$k$' + fid + '$k$') if fid else 'NULL'})")
    if len(batch) == 500:
        flush()
flush()
print("""UPDATE raw_duck.scrambled_objects_20261002 s
SET status = 'quarantined', quarantined_at = now(), quarantine_file_id = a.new_file_id
FROM applied a WHERE a.full_key = s.full_key AND a.quarantine_key = s.quarantine_key;
UPDATE raw_duck.vault_objects v SET in_quarantine = true
FROM applied a WHERE a.full_key = '""" + PREFIX + """' || v.key;
COMMIT;
SELECT status, count(*) AS keys, sum(size) AS bytes FROM raw_duck.scrambled_objects_20261002 WHERE exists_in_b2 GROUP BY 1 ORDER BY 1;""")
print(f"-- {len(done)} ledger rows marked, {len(other)} left", file=sys.stderr)
