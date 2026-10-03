"""Remove the per-message and per-call objects from ProfferMsgEvents20261002 once their chunks are verified.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner-approved removal (2026-10-02): the per-message objects are search copies, rebuildable from Postgres. This is the
library; it runs as Temporal Activities (server/temporal/chunk_backfill_activities.py) under the Go
``proffer_conversation_chunks_removal_workflow`` (modules/engine/proffer/context_chunks_backfill.go), so every run is
traceable in Temporal. ``python -m server.context_chunks.start remove`` starts that workflow.

Verification, three checks, all must pass before anything is deleted:
  1. every Postgres thread's messages are covered by some chunk (coverage is by message id, so a chunk cut before the
     commit counts: its normalized record ids ARE the Postgres message ids);
  2. every per-message object (record_kind message) names a message id that some chunk covers;
  3. every per-call object (record_kind call) names a call id that some call-log file entry covers.
Only objects of origin_system probata are touched. ``only_covered`` deletes just the covered objects, one by one, when
some objects are uncovered (for example a message that never reached a Postgres thread); without it a single uncovered
object stops the removal.
"""

from __future__ import annotations

from typing import Any

from server.context_chunks.model import ThreadRef
from server.context_chunks.source import Source
from server.context_chunks.store import RECORD_KIND_CALL_FILE, RECORD_KIND_CHUNK, ChunkStore

OLD_COLLECTION = "ProfferMsgEvents20261002"
KINDS = ("message", "call")


class NotVerified(RuntimeError):
    """The chunks do not cover everything the old objects stand for; nothing was deleted."""


def _where_kind(kind: str) -> dict:
    return {
        "operator": "And",
        "operands": [
            {"path": ["origin_system"], "operator": "Equal", "valueText": "probata"},
            {"path": ["record_kind"], "operator": "Equal", "valueText": kind},
        ],
    }


def chunk_coverage(chunks: ChunkStore) -> tuple[set[str], set[str]]:
    """(message ids covered by any chunk, call ids covered by any call-log file) from the chunk collection.

    Coverage is by message id, not by thread id: a chunk cut before the commit (from a run's normalized generation)
    belongs to a generation-scoped thread, and the same ids are the Postgres message ids after the commit.
    """
    messages: set[str] = set()
    calls: set[str] = set()
    for o in chunks.iter_objects(["record_kind", "message_ids", "call_log_ids", "normalized_record_ids"]):
        if o["record_kind"] == RECORD_KIND_CHUNK:
            messages.update(o["message_ids"] or [])
            messages.update(o["normalized_record_ids"] or [])
        elif o["record_kind"] == RECORD_KIND_CALL_FILE:
            calls.update(o["call_log_ids"] or [])
            calls.update(o["normalized_record_ids"] or [])
    return messages, calls


def verify(source: Source, chunks: ChunkStore, old: ChunkStore) -> dict[str, Any]:
    covered_messages, calls_covered = chunk_coverage(chunks)
    report: dict[str, Any] = {"chunk_message_ids": len(covered_messages), "chunk_call_ids": len(calls_covered)}
    # 1. Postgres threads vs chunk coverage
    refs: list[ThreadRef] = source.all_threads()
    gaps, missing = [], 0
    for ref in refs:
        short = [m for m in source.thread_message_ids(ref) if m not in covered_messages]
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


def public(report: dict[str, Any]) -> dict[str, Any]:
    """A verification report without the id lists (small enough for a Temporal result)."""
    return {k: v for k, v in report.items() if not k.startswith("_")}


def remove(
    source: Source, chunks: ChunkStore, old: ChunkStore, *, dry_run: bool = True, only_covered: bool = False
) -> dict[str, Any]:
    """Verify, then (unless ``dry_run``) delete. Returns the verification report plus what was or would be deleted.

    Raises NotVerified, deleting nothing, when something is uncovered and ``only_covered`` is not set.
    """
    report = verify(source, chunks, old)
    uncovered = report.pop("_uncovered_ids")
    out: dict[str, Any] = {
        **report,
        "dry_run": dry_run,
        "only_covered": only_covered,
        "deleted": {},
        "kept_uncovered": {},
    }
    if not report["verified"] and not only_covered:
        raise NotVerified(
            "not verified: "
            + ", ".join(f"{k}={report[k]}" for k in report if k.startswith(("uncovered", "old_")) and report[k])
        )
    for kind in KINDS:
        keep = set(uncovered[kind])
        out["kept_uncovered"][kind] = len(keep)
        if dry_run:
            out["deleted"][kind] = old.count(_where_kind(kind)) - len(keep)
        elif not keep:
            out["deleted"][kind] = old.delete_matching(_where_kind(kind))
        else:  # some uncovered: delete only the covered ones, by id
            targets = [
                o["id"]
                for o in old.iter_objects(["record_kind", "origin_system"])
                if o["record_kind"] == kind and o["origin_system"] == "probata" and o["id"] not in keep
            ]
            for object_id in targets:
                old.delete_object(object_id)
            out["deleted"][kind] = len(targets)
    return out
