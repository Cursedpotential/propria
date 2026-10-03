#!/usr/bin/env python3
"""Record in the Case Bible catalog that the zero-filled local files were moved to quarantine.

Byline: Claude Code · Sonnet · 2026-10-02

Owner (Matt) 2026-10-02 20:19 EDT: 'a "moved to quarantine" status in the catalog for the
D: and F: zero-filled files - yes'.

What it writes (PG db casebible on ovh-files, schema raw_duck), following the 2026-09-13
pattern in docs/receipts/corruption-hunt/quarantine/pg_catalog_quarantine.COMMIT.sql
(audit table raw_duck.integrity_hold + integrity_status/integrity_reason columns on the
working table; rows are kept, never deleted):

  raw_duck.source_occurrences   ADD COLUMN IF NOT EXISTS integrity_status, integrity_reason;
                                matched rows get integrity_status='moved_to_quarantine',
                                integrity_reason='all_zero_payload', and metadata gains
                                {"quarantined_to", "quarantined_at", "ledger"}.
                                The row identity (source, scope, path, source_id) and the
                                disposition / md5 / b2_key are NOT changed.
  raw_duck.integrity_hold       one append-only audit row per matched occurrence
                                (catalog 'raw_duck.source_occurrences', bucket = source).

Ledgers read (all under docs/receipts/corruption-hunt/):
  D:  dbackup-quarantine-20261002.tsv            (source, dest, size, verified_zero, result)
  F:  dbackup/*-F-local-quarantine-APPLY.jsonl    (decision == 'moved'; 09-13, src/dst per file)

Matching: source + scope '' + path relative to the drive root that the catalog source covers
(local/D-Backup = D:\\Backup, local/F-case = F:\\case, local/F-Disk-Drill = F:\\Disk Drill),
backslashes turned to '/', exact case. Case-insensitive near-misses and unmatched ledger
rows are reported, never guessed.

Usage:  quarantine_local_zero_catalog_20261002.py --rollback   (validate; always ROLLBACK)
        quarantine_local_zero_catalog_20261002.py --commit     (apply; COMMIT, then read back)
"""
import csv, io, json, subprocess, sys
from pathlib import Path

REC = Path(__file__).resolve().parents[2] / "docs" / "receipts" / "corruption-hunt"
D_LEDGER = REC / "dbackup-quarantine-20261002.tsv"
F_LEDGERS = sorted((REC / "dbackup").glob("*-F-local-quarantine-APPLY.jsonl"))
D_AT, F_AT = "2026-10-02", "2026-09-13"
BS = chr(92)

F_ROOTS = [("F:" + BS + "case" + BS, "local/F-case"), ("F:" + BS + "Disk Drill" + BS, "local/F-Disk-Drill")]


def load():
    rows, seen = [], set()
    d_root = "D:" + BS + "Backup" + BS
    with open(D_LEDGER, encoding="utf-8", newline="") as fh:
        for r in csv.DictReader(fh, delimiter="\t"):
            assert r["result"] == "moved" and r["post_check"] == "ok" and r["verified_zero"] == "yes", r
            assert r["source"].startswith(d_root)
            rel = r["source"][len(d_root):].replace(BS, "/")
            key = ("local/D-Backup", rel)
            assert key not in seen
            seen.add(key)
            rows.append((key[0], rel, int(r["size"]), r["dest"], D_AT, D_LEDGER.name))
    for lf in F_LEDGERS:
        for line in lf.read_text(encoding="utf-8").splitlines():
            j = json.loads(line)
            if j["decision"] != "moved":
                continue
            for root, src in F_ROOTS:
                if j["src"].startswith(root):
                    rel = j["src"][len(root):].replace(BS, "/")
                    key = (src, rel)
                    assert key not in seen, key
                    seen.add(key)
                    rows.append((src, rel, int(j["size"]), j["dst"], F_AT, "dbackup/" + lf.name))
                    break
            else:
                raise SystemExit("unmapped F source " + j["src"])
    return rows


SQL_HEAD = """
begin;
create temp table led(source text, path text, size bigint, quarantined_to text, quarantined_at text, ledger text);
copy led from stdin with (format csv);
"""

SQL_BODY = """
select 'ledger rows', source, count(*) from led group by 2 order by 2;
create temp table m as
  select o.source, o.scope, o.path, o.source_id, o.size, o.md5, o.disposition, l.quarantined_to, l.quarantined_at, l.ledger, l.size as led_size
  from led l join raw_duck.source_occurrences o on o.source = l.source and o.scope = '' and o.path = l.path;
select 'matched occurrences (exact)', source, count(*), count(distinct path) as distinct_paths, count(*) filter (where size <> led_size) as size_mismatch from m group by 2 order by 2;
select 'matched by disposition', source, disposition, count(*), count(*) filter (where md5 is not null) as md5_set from m group by 2,3 order by 2,3;
select 'ledger rows UNMATCHED (exact)', l.source, count(*) from led l where not exists (select 1 from m where m.source = l.source and m.path = l.path) group by 2 order by 2;
select 'unmatched but case-insensitive hit', l.source, count(*) from led l
  where not exists (select 1 from m where m.source = l.source and m.path = l.path)
    and exists (select 1 from raw_duck.source_occurrences o where o.source = l.source and o.scope = '' and lower(o.path) = lower(l.path)) group by 2 order by 2;
select 'unmatched, same filename+size elsewhere in same source', l.source, count(*) from led l
  where not exists (select 1 from m where m.source = l.source and m.path = l.path)
    and exists (select 1 from raw_duck.source_occurrences o where o.source = l.source and o.size = l.size
                and o.path like '%/' || regexp_replace(l.path, '^.*/', '')) group by 2 order by 2;

alter table raw_duck.source_occurrences add column if not exists integrity_status text, add column if not exists integrity_reason text;

insert into raw_duck.integrity_hold (catalog, bucket, path, size, hash_kind, original_hash, status, reason, detector)
select 'raw_duck.source_occurrences', m.source, m.path, m.size, 'md5', m.md5, 'moved_to_quarantine', 'all_zero_payload',
       'local-quarantine-move ' || m.quarantined_at || ' (catalog record 2026-10-02)'
from m
where not exists (select 1 from raw_duck.integrity_hold h where h.catalog = 'raw_duck.source_occurrences'
                  and h.bucket = m.source and h.path = m.path and h.status = 'moved_to_quarantine');

update raw_duck.source_occurrences o
set integrity_status = 'moved_to_quarantine', integrity_reason = 'all_zero_payload',
    metadata = coalesce(o.metadata, '{}'::jsonb) || jsonb_build_object(
        'quarantined_to', m.quarantined_to, 'quarantined_at', m.quarantined_at, 'ledger', m.ledger)
from m where o.source = m.source and o.scope = m.scope and o.path = m.path and o.source_id = m.source_id;

select 'AFTER flagged', source, count(*) from raw_duck.source_occurrences where integrity_status = 'moved_to_quarantine' group by 2 order by 2;
select 'AFTER flagged with metadata.quarantined_to', count(*) from raw_duck.source_occurrences where integrity_status = 'moved_to_quarantine' and metadata ? 'quarantined_to';
select 'AFTER audit rows', bucket, count(*) from raw_duck.integrity_hold where catalog = 'raw_duck.source_occurrences' and status = 'moved_to_quarantine' group by 2 order by 2;
select 'AFTER rows total unchanged', count(*) from raw_duck.source_occurrences;
select 'AFTER flagged outside ledger sources (must be 0)', count(*) from raw_duck.source_occurrences where integrity_status is not null and source not in ('local/D-Backup','local/F-case','local/F-Disk-Drill');
"""


def main():
    mode = sys.argv[1] if len(sys.argv) > 1 else ""
    if mode not in ("--rollback", "--commit"):
        raise SystemExit(__doc__)
    rows = load()
    buf = io.StringIO()
    csv.writer(buf, lineterminator="\n").writerows(rows)
    sql = SQL_HEAD + buf.getvalue() + "\\.\n" + SQL_BODY + ("COMMIT;\n" if mode == "--commit" else "ROLLBACK;\n")
    remote = 'C=$(docker ps --format "{{.Names}}" | grep ^casebible-pg); docker exec -i $C psql -U postgres -d casebible -v ON_ERROR_STOP=1'
    import os
    env = dict(os.environ, MSYS_NO_PATHCONV="1")
    ssh = ["ssh", "-i", str(Path.home() / ".ssh" / "ovh"), "root@100.91.190.107", remote]
    p = subprocess.run(ssh, input=sql.encode("utf-8"), capture_output=True, env=env)
    sys.stdout.write(p.stdout.decode("utf-8", "replace"))
    sys.stderr.write(p.stderr.decode("utf-8", "replace"))
    print("ledger rows loaded by script:", len(rows), "| mode:", mode, "| exit:", p.returncode)
    sys.exit(p.returncode)


if __name__ == "__main__":
    main()
