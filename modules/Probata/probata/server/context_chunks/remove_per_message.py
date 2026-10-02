"""Remove the per-message and per-call objects from ProfferMsgEvents20261002 once their chunks are verified.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner-approved removal (2026-10-02): the per-message objects are search copies, rebuildable from Postgres. Run it only
after the re-chunk is verified. Nothing here runs by itself.

    python -m server.context_chunks.remove_per_message --verify    # coverage report, deletes nothing (the default)
    python -m server.context_chunks.remove_per_message --dry-run   # the counts a deletion would remove
    python -m server.context_chunks.remove_per_message --execute   # verify, then delete

Verification, three checks, all must pass before --execute deletes anything:
  1. every Postgres thread's messages are all covered by that thread's chunks (union of ``message_ids``);
  2. every per-message object (record_kind message) names a message id that some chunk covers;
  3. every per-call object (record_kind call) names a call id that some call-log file entry covers.
Only objects of origin_system probata are touched. ``--only-covered`` deletes just the covered objects, one by one,
when some objects are uncovered (for example a message that never reached a Postgres thread); without it a single
uncovered object stops the run.
"""

from __future__ import annotations

import argparse
import sys

from server.context_chunks.config import load_config
from server.context_chunks.model import ThreadRef
from server.context_chunks.source import PgSource
from server.context_chunks.store import RECORD_KIND_CALL_FILE, RECORD_KIND_CHUNK, ChunkStore, StoreError

OLD_COLLECTION = "ProfferMsgEvents20261002"


def _where_kind(kind: str) -> dict:
    return {
        "operator": "And",
        "operands": [
            {"path": ["origin_system"], "operator": "Equal", "valueText": "probata"},
            {"path": ["record_kind"], "operator": "Equal", "valueText": kind},
        ],
    }


def chunk_coverage(chunks: ChunkStore) -> tuple[dict[str, set[str]], set[str]]:
    """(message ids covered per thread id, call ids covered) from the chunk collection."""
    per_thread: dict[str, set[str]] = {}
    calls: set[str] = set()
    for o in chunks.iter_objects(["record_kind", "thread_id", "message_ids", "call_log_ids"]):
        if o["record_kind"] == RECORD_KIND_CHUNK:
            per_thread.setdefault(o["thread_id"], set()).update(o["message_ids"] or [])
        elif o["record_kind"] == RECORD_KIND_CALL_FILE:
            calls.update(o["call_log_ids"] or [])
    return per_thread, calls


def verify(source: PgSource, chunks: ChunkStore, old: ChunkStore) -> dict:
    per_thread, calls_covered = chunk_coverage(chunks)
    covered_messages = set().union(*per_thread.values()) if per_thread else set()
    report: dict = {
        "chunk_threads": len(per_thread),
        "chunk_message_ids": len(covered_messages),
        "chunk_call_ids": len(calls_covered),
    }
    # 1. Postgres threads vs chunk coverage
    refs: list[ThreadRef] = source.all_threads()
    gaps, missing = [], 0
    for ref in refs:
        want = source.thread_message_ids(ref)
        have = per_thread.get(ref.thread_id, set())
        short = [m for m in want if m not in have]
        if short:
            gaps.append(ref)
            missing += len(short)
    report["pg_threads"] = len(refs)
    report["threads_with_uncovered_messages"] = len(gaps)
    report["uncovered_pg_messages"] = missing
    # 2 and 3. the old per-message and per-call objects
    uncovered: dict[str, list[str]] = {"message": [], "call": []}
    totals = {"message": 0, "call": 0}
    for o in old.iter_objects(["record_kind", "pg_row_id", "origin_system"]):
        kind = o["record_kind"]
        if kind not in totals or o["origin_system"] != "probata":
            continue
        totals[kind] += 1
        covered = covered_messages if kind == "message" else calls_covered
        if o["pg_row_id"] not in covered:
            uncovered[kind].append(o["id"])
    report["old_message_objects"] = totals["message"]
    report["old_call_objects"] = totals["call"]
    report["old_message_objects_uncovered"] = len(uncovered["message"])
    report["old_call_objects_uncovered"] = len(uncovered["call"])
    report["_uncovered_ids"] = uncovered
    report["verified"] = not gaps and not uncovered["message"] and not uncovered["call"]
    return report


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    mode = ap.add_mutually_exclusive_group()
    mode.add_argument("--verify", action="store_true")
    mode.add_argument("--dry-run", action="store_true")
    mode.add_argument("--execute", action="store_true")
    ap.add_argument(
        "--only-covered", action="store_true", help="delete the covered objects one by one when some are not"
    )
    ap.add_argument("--old-collection", default=OLD_COLLECTION)
    args = ap.parse_args(argv)

    from server.context_chunks.db import read_only_connection

    config = load_config()
    chunks = ChunkStore(config.weaviate_url, config.collection)
    old = ChunkStore(config.weaviate_url, args.old_collection)
    with read_only_connection() as conn:
        report = verify(PgSource(conn), chunks, old)
    uncovered = report.pop("_uncovered_ids")
    for key, value in report.items():
        print(f"  {key}: {value}")
    if not (args.dry_run or args.execute):
        return 0 if report["verified"] else 2
    if not report["verified"] and not args.only_covered:
        print("NOT VERIFIED: stopping. Fix the gaps, or rerun with --only-covered to delete just the covered objects.")
        return 2
    kinds = ("message", "call")
    if args.dry_run:
        for kind in kinds:
            n = old.count(_where_kind(kind))
            print(
                f"  would delete {n} {kind} objects from {args.old_collection} (of which uncovered kept: "
                f"{len(uncovered[kind])})"
            )
        return 0
    deleted = 0
    for kind in kinds:
        if not uncovered[kind]:
            n = old.delete_matching(_where_kind(kind))
            print(f"  deleted {n} {kind} objects from {args.old_collection}")
            deleted += n
            continue
        # some uncovered: delete only the covered ones, by id
        keep = set(uncovered[kind])
        targets = [
            o["id"]
            for o in old.iter_objects(["record_kind", "origin_system"])
            if o["record_kind"] == kind and o["origin_system"] == "probata" and o["id"] not in keep
        ]
        for object_id in targets:
            old.delete_object(object_id)
        n = len(targets)
        print(f"  deleted {n} covered {kind} objects, kept {len(keep)} uncovered")
        deleted += n
    print(f"REMOVED {deleted} objects from {args.old_collection}; chunks in {config.collection} untouched")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except StoreError as error:
        print(f"STORE ERROR: {error}")
        sys.exit(1)
