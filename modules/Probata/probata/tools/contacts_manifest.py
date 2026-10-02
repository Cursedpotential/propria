#!/usr/bin/env python3
"""Find the contact exports in the Case Bible catalog, fetch them, and write the import manifest.

Byline: Claude Code · Sonnet · 2026-10-02

Run this ON THE VPS (ovh-files; it needs `docker`, `rclone` and the B2 remote). It only reads: the catalog
table raw_duck.b2_objects (columns key, size, sha1, listed_at) is queried, and each object is copied from B2
into a temporary folder on the VPS. Nothing is written to B2, the catalog or the registry.

What it selects (case-insensitive, on the catalog key):
  * every .vcf
  * .csv / .json whose file name contains "contact"
  * Facebook / Instagram / Meta contact lists: imported_contacts and synced_contacts (.json)
Duplicates are dropped by sha1, keeping the most recently listed key for each content.

Output (OUT_DIR, default /tmp/contacts-manifest-YYYYMMDD):
  files/<sha1>.<ext>     the fetched objects
  manifest.jsonl         {"key", "sha1", "path", "listed_at", "size"} per file, ready for tools/import_contacts.py

    python3 contacts_manifest.py --dry-run     # lists what it would fetch, counts by type, total size; fetches nothing
    python3 contacts_manifest.py               # fetches and writes manifest.jsonl

Settings (environment, all optional):
  CB_PSQL        the full psql command that reaches the catalog (default: docker exec into the casebible-pg
                 container, `psql -U $CB_USER -d $CB_DB`; CB_USER defaults to postgres, CB_DB to casebible)
  RCLONE_REMOTE  where the catalog keys live (default b2:salem-data; the key is appended after a slash)
  RCLONE_CONFIG  path to rclone.conf when it is not in rclone's default place
  KEY_STRIP      a prefix to remove from catalog keys before fetching (default b2://salem-data/)
Check the first lines of --dry-run output before fetching: if the catalog keys are not what the patterns
expect, adjust the SQL below.
"""

from __future__ import annotations

import argparse
import csv
import io
import json
import os
import shlex
import subprocess
import sys
import time
from pathlib import Path

SQL = r"""
COPY (
  SELECT DISTINCT ON (sha1) key, size, sha1, to_char(listed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS listed_at
  FROM raw_duck.b2_objects
  WHERE sha1 IS NOT NULL AND sha1 <> '' AND (
        key ~* '\.vcf$'
     OR key ~* '(^|/)[^/]*contact[^/]*\.(csv|json)$'
     OR key ~* '(facebook|instagram|meta)[^[:space:]]*/[^[:space:]]*(imported_contacts|synced_contacts)[^/]*\.json$'
     OR key ~* '/(imported_contacts|synced_contacts)[^/]*\.json$'
  )
  ORDER BY sha1, listed_at DESC
) TO STDOUT WITH (FORMAT csv)
"""


def kind_of(key: str) -> str:
    lowered = key.lower()
    if lowered.endswith(".vcf"):
        return "vcard"
    if "imported_contacts" in lowered or "synced_contacts" in lowered:
        return "facebook/instagram"
    return "csv/json contacts"


def catalog_rows() -> list[dict]:
    command = os.environ.get("CB_PSQL")
    if not command:
        container = subprocess.run(["docker", "ps", "-q", "--filter", "name=casebible-pg"], capture_output=True, text=True, check=True).stdout.split()
        if not container:
            raise SystemExit("no casebible-pg container is running here; set CB_PSQL")
        command = f"docker exec -i {container[0]} psql -U {os.environ.get('CB_USER', 'postgres')} -d {os.environ.get('CB_DB', 'casebible')}"
    result = subprocess.run(shlex.split(command) + ["-X", "-q", "-v", "ON_ERROR_STOP=1", "-f", "-"], input=SQL, capture_output=True, text=True)
    if result.returncode != 0:
        raise SystemExit(f"catalog query failed:\n{result.stderr[:1500]}")
    rows = []
    for key, size, sha1, listed_at in csv.reader(io.StringIO(result.stdout)):
        rows.append({"key": key, "size": int(size or 0), "sha1": sha1.lower(), "listed_at": listed_at})
    return rows


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--dry-run", action="store_true", help="list and count only; fetch nothing")
    parser.add_argument("--out", default=os.environ.get("OUT_DIR") or f"/tmp/contacts-manifest-{time.strftime('%Y%m%d')}")
    args = parser.parse_args()

    rows = catalog_rows()
    counts: dict[str, list[int]] = {}
    for row in rows:
        entry = counts.setdefault(kind_of(row["key"]), [0, 0])
        entry[0] += 1
        entry[1] += row["size"]
    print(f"{len(rows)} unique contact files by sha1")
    for name, (count, size) in sorted(counts.items()):
        print(f"  {name}: {count} files, {size / 1024 / 1024:.1f} MiB")
    print(f"  total {sum(v[1] for v in counts.values()) / 1024 / 1024:.1f} MiB")
    for row in rows[:10]:
        print(f"  e.g. {row['key']}  ({row['size']} bytes, listed {row['listed_at']})")
    if args.dry_run:
        print("dry run: nothing fetched, no manifest written")
        return 0

    out = Path(args.out)
    (out / "files").mkdir(parents=True, exist_ok=True)
    remote = os.environ.get("RCLONE_REMOTE", "b2:salem-data").rstrip("/")
    strip = os.environ.get("KEY_STRIP", "b2://salem-data/")
    env = dict(os.environ)
    manifest = []
    failed = 0
    for row in rows:
        suffix = Path(row["key"]).suffix.lower() or ".bin"
        relative = f"files/{row['sha1']}{suffix}"
        key = row["key"][len(strip):] if row["key"].startswith(strip) else row["key"]
        target = out / relative
        if not target.is_file() or target.stat().st_size != row["size"]:
            result = subprocess.run(["rclone", "copyto", f"{remote}/{key}", str(target), "--retries", "3", "--low-level-retries", "5"],
                                    capture_output=True, text=True, env=env)
            if result.returncode != 0:
                failed += 1
                print(f"fetch failed for {row['key']}: {result.stderr.strip()[:300]}", file=sys.stderr)
                continue
        manifest.append({"key": row["key"], "sha1": row["sha1"], "path": relative, "listed_at": row["listed_at"], "size": row["size"]})
    (out / "manifest.jsonl").write_text("".join(json.dumps(item) + "\n" for item in manifest), encoding="utf-8")
    print(f"fetched {len(manifest)} files into {out} ({failed} failed); manifest: {out / 'manifest.jsonl'}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
