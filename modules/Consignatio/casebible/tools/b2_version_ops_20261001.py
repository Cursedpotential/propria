#!/usr/bin/env python3
"""Server-side B2 operations by exact file version: copy a version to a key, and hide a key. Runs ON ovh-files.

Byline: Claude Code · Opus 5.5 · 2026-10-01.
Owner 2026-10-01 07:01 EDT "go" on (1) restoring the 1,450 files whose bytes exist only as an old B2 version and
(2) moving the known all-zero vault objects into _quarantine. Log: docs/URGENT-TODO.md, 2026-10-01 entry.

Why the native API: rclone cannot copy one specific old version by its file ID. b2_copy_file can, server-side, with no
download. Nothing is ever deleted: `restore` only adds a new visible version; `quarantine` copies the object to its
quarantine key, verifies the copy (SHA-1 and size equal to the source version), and only then hides the original key,
which leaves the original bytes in place as a noncurrent version (reversible by deleting the hide marker).

Immutable rule: a destination key that already holds a visible object is never overwritten; the row is refused.
Credentials: RCLONE_CONFIG_B2_ACCOUNT / RCLONE_CONFIG_B2_KEY from the systemd EnvironmentFile, never printed.
Resumable through a SQLite ledger.

Usage:
  systemd-run --unit ... -p EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env \
    /usr/bin/python3 b2_version_ops_20261001.py restore    --list restore.tsv --work <dir> [--dry-run]
  ... quarantine --list zero.tsv --work <dir> [--dry-run]
Lists are tab CSV with columns file_id, src_key, size, sha1, dest_key.
"""
from __future__ import annotations

import argparse
import base64
import csv
import json
import os
import sqlite3
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

BUCKET = "salem-data"


class B2:
    def __init__(self) -> None:
        cred = f"{os.environ['RCLONE_CONFIG_B2_ACCOUNT']}:{os.environ['RCLONE_CONFIG_B2_KEY']}"
        req = urllib.request.Request("https://api.backblazeb2.com/b2api/v3/b2_authorize_account",
                                     headers={"Authorization": "Basic " + base64.b64encode(cred.encode()).decode()})
        auth = json.load(urllib.request.urlopen(req, timeout=60))
        self.api = auth["apiInfo"]["storageApi"]["apiUrl"]
        self.token = auth["authorizationToken"]
        self.account = auth["accountId"]
        self.bucket_id = self._bucket_id()

    def call(self, name: str, body: dict) -> dict:
        for attempt in range(6):
            req = urllib.request.Request(f"{self.api}/b2api/v3/{name}", data=json.dumps(body).encode(),
                                         headers={"Authorization": self.token, "Content-Type": "application/json"})
            try:
                return json.load(urllib.request.urlopen(req, timeout=300))
            except urllib.error.HTTPError as e:
                detail = e.read().decode(errors="replace")[:300]
                if e.code in (429, 500, 503) and attempt < 5:
                    time.sleep(2 ** attempt); continue
                raise RuntimeError(f"{name} HTTP {e.code}: {detail}") from None
        raise RuntimeError(f"{name}: retries exhausted")

    def _bucket_id(self) -> str:
        r = self.call("b2_list_buckets", {"accountId": self.account, "bucketName": BUCKET})
        return r["buckets"][0]["bucketId"]

    def visible(self, key: str) -> dict | None:
        """The current visible version at exactly `key`, or None."""
        r = self.call("b2_list_file_names", {"bucketId": self.bucket_id, "startFileName": key, "maxFileCount": 1,
                                             "prefix": key})
        for f in r.get("files", []):
            if f["fileName"] == key and f.get("action") == "upload":
                return f
        return None

    def info(self, file_id: str) -> dict:
        return self.call("b2_get_file_info", {"fileId": file_id})

    def copy(self, file_id: str, dest_key: str) -> dict:
        return self.call("b2_copy_file", {"sourceFileId": file_id, "fileName": dest_key, "metadataDirective": "COPY"})

    def hide(self, key: str) -> dict:
        return self.call("b2_hide_file", {"bucketId": self.bucket_id, "fileName": key})


def ledger(work: Path) -> sqlite3.Connection:
    db = sqlite3.connect(work / "ledger.sqlite")
    db.execute("""create table if not exists op (mode text, file_id text, src_key text, dest_key text, size integer,
        sha1 text, status text, new_file_id text, err text, ts text, primary key (mode, file_id))""")
    db.commit()
    return db


def same(src: dict, dst: dict) -> bool:
    s1 = src.get("contentSha1") or (src.get("fileInfo") or {}).get("large_file_sha1")
    d1 = dst.get("contentSha1") or (dst.get("fileInfo") or {}).get("large_file_sha1")
    return src.get("contentLength") == dst.get("contentLength") and s1 == d1 and s1 not in (None, "none")


def run(mode: str, args) -> int:
    work = Path(args.work); work.mkdir(parents=True, exist_ok=True)
    db = ledger(work)
    done = {r[0] for r in db.execute("select file_id from op where mode = ? and status in ('ok','skipped_same')", (mode,))}
    with open(args.list, encoding="utf-8", newline="") as f:
        rows = [r for r in csv.DictReader(f, delimiter="\t") if r["file_id"] not in done]
    b2 = B2()
    print(f"{time.strftime('%FT%TZ', time.gmtime())} {mode} todo={len(rows)} dry_run={args.dry_run}", flush=True)
    counts: dict[str, int] = {}
    for i, r in enumerate(rows, 1):
        fid, src, dest = r["file_id"], r["src_key"], r["dest_key"]
        status, new_id, err = "", None, None
        try:
            src_info = b2.info(fid)
            if int(src_info["contentLength"]) != int(r["size"]):
                raise RuntimeError(f"source version size {src_info['contentLength']} != catalog {r['size']}")
            existing = b2.visible(dest)
            if existing and same(src_info, existing):
                status, new_id = "skipped_same", existing["fileId"]
            elif existing:
                status, err = "refused_dest_occupied", f"visible {existing['fileId']} at destination"
            elif args.dry_run:
                status = "would_copy"
            else:
                new = b2.copy(fid, dest)
                new_id = new["fileId"]
                if not same(src_info, new):
                    raise RuntimeError(f"copy mismatch: new {new_id}")
                status = "ok"
            if mode == "quarantine" and status in ("ok", "skipped_same") and not args.dry_run:
                cur = b2.visible(src)
                if cur and cur["fileId"] == fid:          # hide only if the zero-filled version is still the visible one
                    b2.hide(src)
                    status = status + "+hidden" if status != "ok" else "ok"
                elif cur:
                    status, err = "copied_not_hidden", f"visible version at source changed to {cur['fileId']}"
        except Exception as e:  # recorded per row; the run continues
            status, err = "failed", str(e)[:400]
        db.execute("insert or replace into op values (?,?,?,?,?,?,?,?,?,datetime('now'))",
                   (mode, fid, src, dest, int(r["size"]), r["sha1"], status, new_id, err))
        db.commit()
        counts[status] = counts.get(status, 0) + 1
        if i % 50 == 0 or status not in ("ok", "would_copy", "skipped_same"):
            print(f"{time.strftime('%FT%TZ', time.gmtime())} {i}/{len(rows)} {status} {dest} {err or ''}", flush=True)
    print(f"{time.strftime('%FT%TZ', time.gmtime())} done {counts}", flush=True)
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("mode", choices=["restore", "quarantine"])
    ap.add_argument("--list", required=True); ap.add_argument("--work", required=True)
    ap.add_argument("--dry-run", action="store_true")
    a = ap.parse_args()
    return run(a.mode, a)


if __name__ == "__main__":
    sys.exit(main())
