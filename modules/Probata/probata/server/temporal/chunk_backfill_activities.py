"""server/temporal/chunk_backfill_activities.py — the re-chunk of committed data and the removal of the per-message objects,
as Activities of two Go workflows, so every run is traceable in Temporal.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
Updated: Codex · GPT-5 · 2026-10-05 — explicit operating-mode and case-scope removal fence.

Owner order 2026-10-02: "Make sure all these things get run as Temporal activities and are traceable." The workflows are
``proffer_conversation_chunks_backfill_workflow`` and ``proffer_conversation_chunks_removal_workflow``
(modules/engine/proffer/context_chunks_backfill.go); dry-run is a workflow input. The per-thread work reuses
``chunk_context_threads_activity`` (with ``thread_refs``) and ``publish_context_chunks_activity`` from chunk_activities.py.

    list_context_threads_activity       the committed threads to re-chunk (optionally only those no chunk covers yet) and
                                        the call-log source versions. Reads Postgres and the chunk collection.
    estimate_context_chunks_activity    the dry-run: threads, messages, estimated or exact chunks, embed calls.
    verify_chunk_coverage_activity      the verification report; deletes nothing.
    remove_per_message_objects_activity verify, then delete (or count, with dry_run) the old per-message and per-call objects.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from temporalio import activity
from temporalio.exceptions import ApplicationError

from server.temporal.chunk_write_guard import require_live_chunk_write

LIST_CONTEXT_THREADS_ACTIVITY = "list_context_threads_activity"
ESTIMATE_CONTEXT_CHUNKS_ACTIVITY = "estimate_context_chunks_activity"
VERIFY_CHUNK_COVERAGE_ACTIVITY = "verify_chunk_coverage_activity"
REMOVE_PER_MESSAGE_OBJECTS_ACTIVITY = "remove_per_message_objects_activity"

CHUNK_BACKFILL_ACTIVITY_NAMES = (
    LIST_CONTEXT_THREADS_ACTIVITY,
    ESTIMATE_CONTEXT_CHUNKS_ACTIVITY,
    VERIFY_CHUNK_COVERAGE_ACTIVITY,
    REMOVE_PER_MESSAGE_OBJECTS_ACTIVITY,
)


@dataclass
class ListThreadsParams:
    """The Go ``proffer.ListContextThreadsRequest``."""

    request_id: str = ""
    skip_covered: bool = False
    limit: int = 0


@dataclass
class EstimateParams:
    """The Go ``proffer.EstimateContextChunksRequest``."""

    request_id: str = ""
    chunker: str = ""
    overlap: int = 0
    exact: bool = False
    sample: int = 6
    sample_cap: int = 5000


@dataclass
class RemovalParams:
    """The Go ``proffer.RemovalRequest`` (verify and remove share it)."""

    request_id: str = ""
    old_collection: str = ""
    dry_run: bool = True
    only_covered: bool = False
    operating_mode: str = ""
    matter_id: str = ""
    court_case_id: str = ""


def _beat(detail: str) -> None:
    if activity.in_activity():
        activity.heartbeat(detail)


@activity.defn(name=LIST_CONTEXT_THREADS_ACTIVITY)
def list_context_threads_activity(params: ListThreadsParams) -> dict[str, Any]:
    from server.context_chunks.config import load_config
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.rechunk import list_threads
    from server.context_chunks.remove_per_message import chunk_coverage
    from server.context_chunks.source import PgSource
    from server.context_chunks.store import ChunkStore

    covered = None
    if params.skip_covered:
        config = load_config()
        covered = chunk_coverage(ChunkStore(config.weaviate_url, config.collection))[0]
    with read_only_connection() as conn:
        threads, call_versions = list_threads(PgSource(conn), covered=covered, limit=params.limit)
    activity.logger.info("%d thread(s) to re-chunk, %d call-log file(s)", len(threads), len(call_versions))
    return {"threads": threads, "call_source_versions": call_versions}


@activity.defn(name=ESTIMATE_CONTEXT_CHUNKS_ACTIVITY)
def estimate_context_chunks_activity(params: EstimateParams) -> dict[str, Any]:
    from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.rechunk import dry_run
    from server.context_chunks.source import PgSource

    chunker = params.chunker.strip() or DEFAULT_CHUNKER
    overlap = params.overlap if params.overlap > 0 else DEFAULT_OVERLAP
    try:
        with read_only_connection() as conn:
            return dry_run(PgSource(conn), chunker, overlap, params.sample, params.exact, params.sample_cap, beat=_beat)
    except ValueError as error:
        raise ApplicationError(str(error), non_retryable=True) from error


def _stores(old_collection: str):
    from server.context_chunks.config import load_config
    from server.context_chunks.remove_per_message import OLD_COLLECTION
    from server.context_chunks.store import ChunkStore

    config = load_config()
    return (
        ChunkStore(config.weaviate_url, config.collection),
        ChunkStore(config.weaviate_url, old_collection.strip() or OLD_COLLECTION),
    )


@activity.defn(name=VERIFY_CHUNK_COVERAGE_ACTIVITY)
def verify_chunk_coverage_activity(params: RemovalParams) -> dict[str, Any]:
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.remove_per_message import public, verify
    from server.context_chunks.source import PgSource

    chunks, old = _stores(params.old_collection)
    with read_only_connection() as conn:
        return public(verify(PgSource(conn), chunks, old))


@activity.defn(name=REMOVE_PER_MESSAGE_OBJECTS_ACTIVITY)
def remove_per_message_objects_activity(params: RemovalParams) -> dict[str, Any]:
    """Verify coverage and count, or explicitly admit Live removal of old search objects.

    Inputs: old collection, dry-run/coverage flags, and mode plus approved matter/court.
    Outputs: coverage/removal report. Effects: reads Postgres/Weaviate; non-dry-run may
    delete vector objects after the existing coverage gate. Pick verify_chunk_coverage
    for a report only. Dry-run needs no write authority; mutation fails before imports.
    """
    if params.dry_run is not True:
        require_live_chunk_write(params.operating_mode, params.matter_id, params.court_case_id)

    from server.context_chunks.db import read_only_connection
    from server.context_chunks.remove_per_message import NotVerified, remove
    from server.context_chunks.source import PgSource

    chunks, old = _stores(params.old_collection)
    try:
        with read_only_connection() as conn:
            return remove(PgSource(conn), chunks, old, dry_run=params.dry_run, only_covered=params.only_covered)
    except NotVerified as error:
        raise ApplicationError(str(error), non_retryable=True) from error
