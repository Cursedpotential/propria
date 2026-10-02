#!/usr/bin/env python3
"""Stream B2 objects once and record SHA-256, SHA-1 and MD5 of their bytes. Runs ON ovh-files.

Byline: Claude Code · Opus 5.5 · 2026-10-01.
Owner 2026-09-30 22:21 EDT: files must be "verifiably what they say they are, ready for court evaluation";
owner 2026-10-01 07:01 EDT "go" on the hash pass for the B2 objects that carry no SHA-1 (multipart uploads)
or that match their sources by size only. Log: modules/Consignatio/docs/LOG.md, 2026-10-01 entry.

Reads only: `rclone cat` of each key, hashed in one pass. Nothing is written to B2. No local copy is kept.
Resumable: a SQLite ledger records every finished key; a rerun skips them. Failures are recorded, not retried
silently; rerun with --retry-failed.

Usage (detached, credentials only from the systemd EnvironmentFile):
  systemd-run --unit casebible-hash-pass-20261001 -p EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env \
    /usr/bin/python3 hash_pass_20261001.py run --list keys.tsv --work /data/consignatio/court-ready-20261001/hash-pass
  python3 hash_pass_20261001.py status --work ...
  python3 hash_pass_20261001.py export --work ... > results.tsv
Input keys.tsv: object_key<TAB>size, header line first.
"""
from __future__ import annotations

import argparse
import concurrent.futures as cf
import hashlib
import sqlite3
import subprocess
import sys
import threading
import time
from pathlib import Path

CONF = "/opt/casebible/rclone.conf"        # holds no b2 section; the b2 remote comes from the environment
BUCKET = "b2:salem-data/"
CHUNK = 8 * 1024 * 1024

_lock = threading.Lock()


def ledger(work: Path) -> sqlite3.Connection:
    db = sqlite3.connect(work / "ledger.sqlite", check_same_thread=False)
    db.execute("""create table if not exists done (
        object_key text primary key, size_expected integer, size_read integer,
        sha256 text, sha1 text, md5 text, status text, err text, seconds real, ts text)""")
    db.commit()
    return db


def hash_one(key: str, size: int) -> tuple:
    start = time.time()
    h256, h1, h5 = hashlib.sha256(), hashlib.sha1(), hashlib.md5()
    n = 0
    proc = subprocess.Popen(["rclone", "cat", "--config", CONF, "--low-level-retries", "10", BUCKET + key],
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    try:
        while True:
            buf = proc.stdout.read(CHUNK)
            if not buf:
                break
            n += len(buf)
            h256.update(buf); h1.update(buf); h5.update(buf)
    finally:
        rc = proc.wait()
    err = proc.stderr.read().decode(errors="replace")[-500:]
    if rc != 0:
        return (key, size, n, None, None, None, "failed", f"rclone exit {rc}: {err}", time.time() - start)
    status = "ok" if n == size else "size_mismatch"
    return (key, size, n, h256.hexdigest(), h1.hexdigest(), h5.hexdigest(), status, err or None, time.time() - start)


def run(args) -> int:
    work = Path(args.work); work.mkdir(parents=True, exist_ok=True)
    db = ledger(work)
    finished = {r[0] for r in db.execute(
        "select object_key from done where status <> 'failed'" if args.retry_failed else "select object_key from done")}
    todo = []
    with open(args.list, encoding="utf-8") as f:
        next(f)
        for line in f:
            key, size = line.rstrip("\n").split("\t")
            if key not in finished:
                todo.append((key, int(size)))
    todo.sort(key=lambda kv: -kv[1])                       # largest first, so the long tail is small files
    print(f"{time.strftime('%FT%TZ', time.gmtime())} todo={len(todo)} bytes={sum(s for _, s in todo)}", flush=True)
    with cf.ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = {pool.submit(hash_one, k, s): k for k, s in todo}
        for i, fut in enumerate(cf.as_completed(futures), 1):
            row = fut.result()
            with _lock:
                db.execute("insert or replace into done values (?,?,?,?,?,?,?,?,?,datetime('now'))", row)
                db.commit()
            print(f"{time.strftime('%FT%TZ', time.gmtime())} {i}/{len(todo)} {row[6]} {row[2]} {row[0]}", flush=True)
    return 0


def status(args) -> int:
    db = ledger(Path(args.work))
    for st, n, b in db.execute("select status, count(*), coalesce(sum(size_read),0) from done group by 1"):
        print(f"{st}\t{n}\t{b}")
    return 0


def export(args) -> int:
    db = ledger(Path(args.work))
    print("object_key\tsize_expected\tsize_read\tsha256\tsha1\tmd5\tstatus\terr")
    for row in db.execute("select object_key,size_expected,size_read,sha256,sha1,md5,status,coalesce(err,'') from done"):
        print("\t".join("" if v is None else str(v).replace("\t", " ").replace("\n", " ") for v in row))
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    r = sub.add_parser("run"); r.add_argument("--list", required=True); r.add_argument("--work", required=True)
    r.add_argument("--workers", type=int, default=6); r.add_argument("--retry-failed", action="store_true")
    for name in ("status", "export"):
        s = sub.add_parser(name); s.add_argument("--work", required=True)
    args = ap.parse_args()
    return {"run": run, "status": status, "export": export}[args.cmd](args)


if __name__ == "__main__":
    sys.exit(main())
