"""server/temporal/chunk_activities.py — conversation chunks for Weaviate, as Activities on queue ``evidence-pipeline``.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The Go ProfferWorkflow (modules/engine/proffer/context_chunks.go) calls these after the first-party threads are
committed, the same way the Go extraction flow calls ``build_timeline_generation_activity`` on this queue. One unit,
one job; references in, small results out (the ATOMICITY rules):

    chunk_context_threads_activity     CHUNK: the threads a source version touched -> per-thread plans (message-index
                                       spans + digest). Reads Postgres; no embed, no writes. Safe to retry.
    publish_context_chunks_activity    EMBED + PUBLISH one thread's chunks into ProfferChunks20261002 and replace that
                                       thread's chunks of any other chunking. Idempotent: ids are deterministic.
    publish_call_log_files_activity    one Weaviate entry per call-log file (the source version's calls); the
                                       individual calls stay in Postgres.

Import rule (as in timeline_activities.py): temporalio + stdlib at module level; everything heavy (SQLAlchemy engine,
Chonkie, torch) is imported inside the bodies so importing this module is side-effect free.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from temporalio import activity
from temporalio.exceptions import ApplicationError

CHUNK_CONTEXT_THREADS_ACTIVITY = "chunk_context_threads_activity"
PUBLISH_CONTEXT_CHUNKS_ACTIVITY = "publish_context_chunks_activity"
PUBLISH_CALL_LOG_FILES_ACTIVITY = "publish_call_log_files_activity"

CHUNK_ACTIVITY_NAMES = (
    CHUNK_CONTEXT_THREADS_ACTIVITY,
    PUBLISH_CONTEXT_CHUNKS_ACTIVITY,
    PUBLISH_CALL_LOG_FILES_ACTIVITY,
)


@dataclass
class ChunkThreadsParams:
    """The Go ``proffer.ChunkThreadsRequest`` (JSON field names match)."""

    request_id: str = ""
    source_version_id: str = ""
    chunker: str = ""
    overlap: int = 0


@dataclass
class PublishChunksParams:
    """The Go ``proffer.PublishChunksRequest``: one thread's plan, as the chunk Activity returned it."""

    request_id: str = ""
    plan: dict[str, Any] = field(default_factory=dict)


@dataclass
class PublishCallLogFilesParams:
    """The Go ``proffer.PublishCallLogFilesRequest``."""

    request_id: str = ""
    source_version_id: str = ""


def _beat(detail: str) -> None:
    if activity.in_activity():
        activity.heartbeat(detail)


@activity.defn(name=CHUNK_CONTEXT_THREADS_ACTIVITY)
def chunk_context_threads_activity(params: ChunkThreadsParams) -> dict[str, Any]:
    from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.service import plan_source_version
    from server.context_chunks.source import PgSource

    if not params.source_version_id.strip():
        raise ApplicationError(f"{CHUNK_CONTEXT_THREADS_ACTIVITY} requires a source version id", non_retryable=True)
    chunker = params.chunker.strip() or DEFAULT_CHUNKER
    overlap = params.overlap if params.overlap > 0 else DEFAULT_OVERLAP
    try:
        with read_only_connection() as conn:
            plans = plan_source_version(PgSource(conn), params.source_version_id, chunker, overlap, beat=_beat)
    except ValueError as error:  # an unknown chunker name is a request error, not a transient one
        raise ApplicationError(str(error), non_retryable=True) from error
    activity.logger.info(
        "chunked %d thread(s) for source version %s (%s)", len(plans), params.source_version_id, chunker
    )
    version = plans[0].chunker_version if plans else ""
    return {"chunker": chunker, "chunker_version": version, "overlap": overlap, "threads": [p.to_dict() for p in plans]}


@activity.defn(name=PUBLISH_CONTEXT_CHUNKS_ACTIVITY)
def publish_context_chunks_activity(params: PublishChunksParams) -> dict[str, Any]:
    from server.context_chunks.config import load_config
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.embed import NimEmbedder
    from server.context_chunks.model import ThreadPlan
    from server.context_chunks.service import PlanStale, publish_thread
    from server.context_chunks.source import PgSource
    from server.context_chunks.store import ChunkStore

    plan = ThreadPlan.from_dict(params.plan)
    config = load_config()
    embedder = NimEmbedder(config)
    try:
        with read_only_connection() as conn:
            result = publish_thread(
                PgSource(conn), ChunkStore(config.weaviate_url, config.collection), embedder, plan, beat=_beat
            )
    except PlanStale as error:
        raise ApplicationError(str(error), non_retryable=True) from error
    activity.logger.info(
        "published %d chunk(s) of thread %s (stale deleted %d)",
        result.chunks_written,
        plan.thread_id,
        result.stale_deleted,
    )
    return {
        "corpus": result.corpus,
        "thread_id": result.thread_id,
        "chunks_written": result.chunks_written,
        "stale_deleted": result.stale_deleted,
        "embed_texts": result.embed_texts,
        "embed_requests": embedder.calls,
        "skipped_existing": result.skipped_existing,
    }


@activity.defn(name=PUBLISH_CALL_LOG_FILES_ACTIVITY)
def publish_call_log_files_activity(params: PublishCallLogFilesParams) -> dict[str, Any]:
    from server.context_chunks.config import load_config
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.embed import NimEmbedder
    from server.context_chunks.service import publish_call_files
    from server.context_chunks.source import PgSource
    from server.context_chunks.store import ChunkStore

    if not params.source_version_id.strip():
        raise ApplicationError(f"{PUBLISH_CALL_LOG_FILES_ACTIVITY} requires a source version id", non_retryable=True)
    config = load_config()
    embedder = NimEmbedder(config)
    with read_only_connection() as conn:
        result = publish_call_files(
            PgSource(conn),
            ChunkStore(config.weaviate_url, config.collection),
            embedder,
            params.source_version_id,
            beat=_beat,
        )
    activity.logger.info("published %d call-log file(s), %d calls", result["files"], result["calls"])
    return {**result, "embed_requests": embedder.calls}
