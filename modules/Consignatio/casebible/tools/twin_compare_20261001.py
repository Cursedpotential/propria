#!/usr/bin/env python3
"""Compare two same-name, same-size copies byte by byte and say how they differ. Runs ON ovh-files.

Byline: Claude Code · Opus 5.5 · 2026-10-01.
Owner 2026-09-30 22:21 EDT: "no funny business with any of them"; 2026-10-01 07:01 EDT "go" on comparing the
~2,700 same-size media twins (raw_duck.verification_20260930 flag altered_twin). Log: docs/LOG.md 2026-10-01.

Reads only: both objects are streamed side by side with `rclone cat`; nothing is written to B2 and no copy is kept.
Per pair it records how many bytes differ, where the first and last difference sit, and whether either side holds
all-zero blocks the other does not. Classes:
  zeroed_blocks      one side has all-zero 1 MiB blocks where the other has data: partial corruption (zero fill)
  metadata_region    every difference lies in the first or last 256 KiB: a header/metadata change (EXIF, XMP, moov/udta)
  sparse             under 1% of bytes differ, spread through the file: scattered damage or an in-place edit
  different_content  the bytes differ throughout: different files that share a name and a size
  identical          no difference (should not happen; would mean a catalog hash error)
Resumable through a SQLite ledger. Input pairs.tsv (tab CSV): group_id, size, key_a, key_b.
"""
from __future__ import annotations

import argparse
import concurrent.futures as cf
import csv
import sqlite3
import subprocess
import sys
import threading
import time
from pathlib import Path

CONF = "/opt/casebible/rclone.conf"
BUCKET = "b2:salem-data/"
CHUNK = 1024 * 1024
EDGE = 256 * 1024
_lock = threading.Lock()


def open_stream(key: str) -> subprocess.Popen:
    return subprocess.Popen(["rclone", "cat", "--config", CONF, "--low-level-retries", "10", BUCKET + key],
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def read_full(stream, n: int) -> bytes:
    parts, got = [], 0
    while got < n:
        b = stream.read(n - got)
        if not b:
            break
        parts.append(b); got += len(b)
    return b"".join(parts)


def compare(group_id: str, size: int, key_a: str, key_b: str) -> tuple:
    start = time.time()
    pa, pb = open_stream(key_a), open_stream(key_b)
    off = diff_bytes = diff_chunks = zero_a = zero_b = 0
    first = last = None
    try:
        while True:
            a, b = read_full(pa.stdout, CHUNK), read_full(pb.stdout, CHUNK)
            if not a and not b:
                break
            if a != b:
                n = min(len(a), len(b))
                x = (int.from_bytes(a[:n], "big") ^ int.from_bytes(b[:n], "big")).to_bytes(n, "big")
                d = n - x.count(0) + abs(len(a) - len(b))
                if d:
                    diff_chunks += 1; diff_bytes += d
                    lead = n - len(x.lstrip(b"\0")); trail = n - len(x.rstrip(b"\0"))
                    if first is None:
                        first = off + lead
                    last = off + n - 1 - trail
                    if a.count(0) == len(a) and b.count(0) != len(b):
                        zero_a += 1
                    if b.count(0) == len(b) and a.count(0) != len(a):
                        zero_b += 1
            off += max(len(a), len(b))
    finally:
        ra, rb = pa.wait(), pb.wait()
    if ra or rb:
        err = (pa.stderr.read() + pb.stderr.read()).decode(errors="replace")[-400:]
        return (group_id, size, key_a, key_b, off, None, None, None, None, None, None, "failed", err, time.time() - start)
    if diff_bytes == 0:
        cls = "identical"
    elif zero_a or zero_b:
        cls = "zeroed_blocks"
    elif first is not None and (last < EDGE or first >= size - EDGE or (first < EDGE and last >= size - EDGE and diff_bytes <= 2 * EDGE)):
        cls = "metadata_region"
    elif diff_bytes < size * 0.01:
        cls = "sparse"
    else:
        cls = "different_content"
    return (group_id, size, key_a, key_b, off, diff_bytes, diff_chunks, first, last, zero_a, zero_b, cls, None,
            time.time() - start)


def ledger(work: Path) -> sqlite3.Connection:
    db = sqlite3.connect(work / "ledger.sqlite", check_same_thread=False)
    db.execute("""create table if not exists pair (
        group_id text, size integer, key_a text, key_b text, bytes_read integer, diff_bytes integer, diff_chunks integer,
        first_diff integer, last_diff integer, zero_blocks_a integer, zero_blocks_b integer, class text, err text,
        seconds real, ts text, primary key (key_a, key_b))""")
    db.commit()
    return db


def run(args) -> int:
    work = Path(args.work); work.mkdir(parents=True, exist_ok=True)
    db = ledger(work)
    done = {(a, b) for a, b in db.execute("select key_a, key_b from pair where class <> 'failed'")}
    with open(args.pairs, encoding="utf-8", newline="") as f:
        rows = [r for r in csv.DictReader(f, delimiter="\t") if (r["key_a"], r["key_b"]) not in done]
    print(f"{time.strftime('%FT%TZ', time.gmtime())} todo={len(rows)}", flush=True)
    with cf.ThreadPoolExecutor(max_workers=args.workers) as pool:
        futs = [pool.submit(compare, r["group_id"], int(r["size"]), r["key_a"], r["key_b"]) for r in rows]
        for i, fut in enumerate(cf.as_completed(futs), 1):
            row = fut.result()
            with _lock:
                db.execute("insert or replace into pair values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now'))", row)
                db.commit()
            print(f"{time.strftime('%FT%TZ', time.gmtime())} {i}/{len(rows)} {row[11]} {row[0]}", flush=True)
    return 0


def export(args) -> int:
    db = ledger(Path(args.work))
    w = csv.writer(sys.stdout, delimiter="\t", lineterminator="\n")
    w.writerow(["group_id", "size", "key_a", "key_b", "bytes_read", "diff_bytes", "diff_chunks", "first_diff",
                "last_diff", "zero_blocks_a", "zero_blocks_b", "class", "err"])
    for row in db.execute("select group_id,size,key_a,key_b,bytes_read,diff_bytes,diff_chunks,first_diff,last_diff,"
                          "zero_blocks_a,zero_blocks_b,class,coalesce(err,'') from pair"):
        w.writerow(["" if v is None else v for v in row])
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    r = sub.add_parser("run"); r.add_argument("--pairs", required=True); r.add_argument("--work", required=True)
    r.add_argument("--workers", type=int, default=4)
    e = sub.add_parser("export"); e.add_argument("--work", required=True)
    a = ap.parse_args()
    return {"run": run, "export": export}[a.cmd](a)


if __name__ == "__main__":
    sys.exit(main())
