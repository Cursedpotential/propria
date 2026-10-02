#!/usr/bin/env python3
"""Load one rclone listing of a whole bucket into the catalog table raw_duck.bucket_objects. Runs ON ovh-files.

Byline: Claude Code · Opus 5.5 · 2026-10-02.
Owner 2026-10-02 19:04 EDT: "the B2 objects [table] needs to match whatever it's listing ... can we do the entire
bucket? ... can different buckets be part of the same lakehouse or catalog?" -> one table for every bucket of every
provider, one row per object per listing generation; `raw_duck.bucket_objects_current` is the newest generation of
each (provider, bucket). Replaces the per-folder B2 tables (b2_objects = intake 09-14, vault_objects_20260916_rN,
casevault_objects) for new work. Log: modules/Consignatio/docs/LOG.md, 2026-10-02 entries.

Input is `rclone lsjson -R --files-only --hash --fast-list --no-mimetype <remote>:<bucket>` (read-only list calls).
B2 reports SHA-1 (absent for large multipart uploads); R2 reports MD5 (absent when the ETag is multipart).
listed_at = the listing file's mtime (UTC), so a reload of the same file is a no-op.

Usage: bucket_objects_load.py <provider b2|r2> <bucket> <listing.json>
"""
from __future__ import annotations

import datetime as dt
import json
import os
import subprocess
import sys
import tempfile

SQL = r"""
\set ON_ERROR_STOP 1
begin;
create table if not exists raw_duck.bucket_objects (
  provider text not null,          -- b2 | r2
  bucket text not null,
  key text not null,               -- full object key inside the bucket
  size bigint not null,
  sha1 text,                       -- provider-stored SHA-1 (B2); null when the provider has none
  md5 text,                        -- provider-stored MD5 (R2 ETag); null when multipart
  modtime timestamptz,
  listed_at timestamptz not null,  -- listing generation
  primary key (provider, bucket, listed_at, key)
);
comment on table raw_duck.bucket_objects is
  'Every object of every bucket (B2 and R2), one row per object per listing generation. Loaded by casebible/tools/bucket_objects_load.py from rclone lsjson of the WHOLE bucket. Use bucket_objects_current for the newest listing of each bucket. Supersedes b2_objects (intake only, 2026-09-14), vault_objects_20260916_rN and casevault_objects for new work.';
create temp table bo_in (key text, size bigint, sha1 text, md5 text, modtime text);
\copy bo_in from '@INFILE@' with (format text)
insert into raw_duck.bucket_objects (provider, bucket, key, size, sha1, md5, modtime, listed_at)
select :'provider', :'bucket', key, size, nullif(sha1, ''), nullif(md5, ''), nullif(modtime, '')::timestamptz,
       :'listed_at'::timestamptz
from bo_in
on conflict do nothing;
create or replace view raw_duck.bucket_objects_current as
select o.* from raw_duck.bucket_objects o
join (select provider, bucket, max(listed_at) listed_at from raw_duck.bucket_objects group by 1, 2) g
  using (provider, bucket, listed_at);
comment on view raw_duck.bucket_objects_current is
  'Newest listing generation of each (provider, bucket) in raw_duck.bucket_objects.';
commit;
select provider, bucket, listed_at, count(*) objects, sum(size) bytes, count(sha1) with_sha1, count(md5) with_md5
from raw_duck.bucket_objects where provider = :'provider' and bucket = :'bucket' and listed_at = :'listed_at'::timestamptz
group by 1, 2, 3;
"""


def pg_text(v: str) -> str:
    return v.replace("\\", "\\\\").replace("\t", "\\t").replace("\n", "\\n").replace("\r", "\\r")


def main() -> int:
    if len(sys.argv) != 4 or sys.argv[1] not in ("b2", "r2"):
        print(__doc__, file=sys.stderr)
        return 2
    provider, bucket, listing = sys.argv[1], sys.argv[2], sys.argv[3]
    listed_at = dt.datetime.fromtimestamp(os.path.getmtime(listing), dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    n = 0
    with open(listing, encoding="utf-8") as f, tempfile.NamedTemporaryFile("w", encoding="utf-8", delete=False,
                                                                              suffix=".tsv", newline="\n") as out:
        for line in f:  # rclone lsjson writes one object per line inside a JSON array
            line = line.strip().rstrip(",")
            if not line.startswith("{"):
                continue
            o = json.loads(line)
            h = o.get("Hashes") or {}
            out.write("\t".join([pg_text(o["Path"]), str(o["Size"]), h.get("sha1", ""), h.get("md5", ""),
                                 o.get("ModTime", "")]) + "\n")
            n += 1
        tsv = out.name
    os.chmod(tsv, 0o644)  # docker cp keeps the mode; psql runs as the postgres user
    pg = subprocess.run("docker ps --format '{{.Names}}' | grep ^casebible-pg", shell=True, capture_output=True,
                        text=True, check=True).stdout.split()[0]
    infile = f"/tmp/bucket_objects_in_{provider}_{bucket}_{os.getpid()}.tsv"  # parallel loads must not share a file
    subprocess.run(["docker", "cp", tsv, f"{pg}:{infile}"], check=True)
    os.unlink(tsv)
    r = subprocess.run(["docker", "exec", "-i", pg, "psql", "-U", "postgres", "-d", "casebible", "-At",
                        "-v", f"provider={provider}", "-v", f"bucket={bucket}", "-v", f"listed_at={listed_at}"],
                       input=SQL.replace("@INFILE@", infile), text=True, capture_output=True)
    subprocess.run(["docker", "exec", "-u", "root", pg, "rm", "-f", infile])
    sys.stdout.write(f"parsed {n} objects from {listing} (listed_at {listed_at})\n{r.stdout}")
    sys.stderr.write(r.stderr)
    return r.returncode


if __name__ == "__main__":
    sys.exit(main())
