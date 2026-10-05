"""server/temporal/chunk_activities.py — conversation chunks for Weaviate, as Activities on queue ``evidence-pipeline``.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
Updated: Codex · GPT-5 · 2026-10-05 — explicit operating-mode and case-scope write fence.

The Go ProfferWorkflow (modules/engine/proffer/context_chunks.go) calls these BEFORE the preview and the owner's
decision, right after the proposal of the first-party context (owner 2026-10-02: everything goes to Weaviate first),
reading the run's normalized generation (``normalized_generation_id``), not the committed working.* tables; the same
way the Go extraction flow calls ``build_timeline_generation_activity`` on this queue. One unit,
one job; references in, small results out (the ATOMICITY rules):

    chunk_context_threads_activity     CHUNK: the generation's conversations (or, with no generation, the committed threads a
                                       source version touched) -> per-thread plans (message-index
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

from server.temporal.chunk_write_guard import require_live_chunk_write

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
    """The Go ``proffer.ChunkThreadsRequest`` (JSON field names match).

    With ``normalized_generation_id`` the conversations are cut from that generation's normalized records (the run's
    participant resolution and message match-up receipts, when given, decide the corpus and the messages already held
    by an earlier source). Without it they are the committed threads a source version touched (working.*).
    """

    request_id: str = ""
    source_version_id: str = ""
    chunker: str = ""
    overlap: int = 0
    normalized_generation_id: str = ""
    participant_resolution_id: str = ""
    message_matches_id: str = ""
    # Re-chunk of committed data: the named threads ({"corpus", "thread_id"}), read from working.*.
    thread_refs: list[dict[str, str]] = field(default_factory=list)
    operating_mode: str = ""
    matter_id: str = ""
    court_case_id: str = ""


@dataclass
class PublishChunksParams:
    """The Go ``proffer.PublishChunksRequest``: one thread's plan, as the chunk Activity returned it."""

    request_id: str = ""
    plan: dict[str, Any] = field(default_factory=dict)
    normalized_generation_id: str = ""
    participant_resolution_id: str = ""
    message_matches_id: str = ""
    operating_mode: str = ""
    matter_id: str = ""
    court_case_id: str = ""


@dataclass
class PublishCallLogFilesParams:
    """The Go ``proffer.PublishCallLogFilesRequest``."""

    request_id: str = ""
    source_version_id: str = ""
    normalized_generation_id: str = ""
    participant_resolution_id: str = ""
    message_matches_id: str = ""
    operating_mode: str = ""
    matter_id: str = ""
    court_case_id: str = ""


def _source(conn: Any, params: Any) -> Any:
    """GenerationSource for a run before its commit, PgSource (committed working.*) otherwise."""
    from server.context_chunks.generation import GenerationSource
    from server.context_chunks.source import PgSource

    if params.normalized_generation_id.strip():
        return GenerationSource(
            conn,
            params.normalized_generation_id.strip(),
            params.participant_resolution_id.strip(),
            params.message_matches_id.strip(),
        )
    return PgSource(conn)


def _beat(detail: str) -> None:
    if activity.in_activity():
        activity.heartbeat(detail)


@activity.defn(name=CHUNK_CONTEXT_THREADS_ACTIVITY)
def chunk_context_threads_activity(params: ChunkThreadsParams) -> dict[str, Any]:
    from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.model import ThreadRef
    from server.context_chunks.service import plan_source_version, plan_threads

    if not params.source_version_id.strip() and not params.thread_refs:
        raise ApplicationError(
            f"{CHUNK_CONTEXT_THREADS_ACTIVITY} requires a source version id or thread references", non_retryable=True
        )
    chunker = params.chunker.strip() or DEFAULT_CHUNKER
    overlap = params.overlap if params.overlap > 0 else DEFAULT_OVERLAP
    try:
        with read_only_connection() as conn:
            if params.thread_refs:
                refs = [ThreadRef(r["corpus"], r["thread_id"]) for r in params.thread_refs]
                plans = plan_threads(_source(conn, params), refs, chunker, overlap, beat=_beat)
            else:
                plans = plan_source_version(
                    _source(conn, params), params.source_version_id, chunker, overlap, beat=_beat
                )
    except (ValueError, LookupError) as error:  # unknown chunker, unusable records, a missing receipt: not transient
        raise ApplicationError(str(error), non_retryable=True) from error
    activity.logger.info(
        "chunked %d thread(s) for source version %s (%s)", len(plans), params.source_version_id, chunker
    )
    version = plans[0].chunker_version if plans else ""
    return {"chunker": chunker, "chunker_version": version, "overlap": overlap, "threads": [p.to_dict() for p in plans]}


@activity.defn(name=PUBLISH_CONTEXT_CHUNKS_ACTIVITY)
def publish_context_chunks_activity(params: PublishChunksParams) -> dict[str, Any]:
    """Embed and replace one admitted Live thread's search chunks.

    Inputs: thread plan, generation references, explicit mode and approved matter/court.
    Outputs: publication counters. Effects: reads Postgres, calls NIM, writes Weaviate.
    Pick this for thread chunks, not the sibling call-log file publisher. Missing/Dev
    authority fails before heavy imports; existing evidence/plan checks remain in force.
    """
    require_live_chunk_write(params.operating_mode, params.matter_id, params.court_case_id)

    from server.context_chunks.config import load_config
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.embed import NimEmbedder
    from server.context_chunks.model import ThreadPlan
    from server.context_chunks.service import PlanStale, publish_thread
    from server.context_chunks.store import ChunkStore

    plan = ThreadPlan.from_dict(params.plan)
    config = load_config()
    embedder = NimEmbedder(config)
    try:
        with read_only_connection() as conn:
            result = publish_thread(
                _source(conn, params),
                ChunkStore(config.weaviate_url, config.collection),
                embedder,
                plan,
                beat=_beat,
                generation_id=params.normalized_generation_id.strip(),
                run_id=params.request_id.strip(),
                reuse=ChunkStore(config.weaviate_url, config.reuse_collection) if config.reuse_collection else None,
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
        "reused": result.reused,
    }


@activity.defn(name=PUBLISH_CALL_LOG_FILES_ACTIVITY)
def publish_call_log_files_activity(params: PublishCallLogFilesParams) -> dict[str, Any]:
    """Embed and publish admitted Live call-log files as search entries.

    Inputs: source/generation references, explicit mode and approved matter/court.
    Outputs: file/call/embed counters. Effects: reads Postgres, calls NIM, writes Weaviate.
    Pick this for file-level call logs, not conversation chunks. Missing/Dev authority
    fails before heavy imports; individual calls remain in the canonical source.
    """
    require_live_chunk_write(params.operating_mode, params.matter_id, params.court_case_id)

    from server.context_chunks.config import load_config
    from server.context_chunks.db import read_only_connection
    from server.context_chunks.embed import NimEmbedder
    from server.context_chunks.service import publish_call_files
    from server.context_chunks.store import ChunkStore

    if not params.source_version_id.strip():
        raise ApplicationError(f"{PUBLISH_CALL_LOG_FILES_ACTIVITY} requires a source version id", non_retryable=True)
    config = load_config()
    embedder = NimEmbedder(config)
    with read_only_connection() as conn:
        result = publish_call_files(
            _source(conn, params),
            ChunkStore(config.weaviate_url, config.collection),
            embedder,
            params.source_version_id,
            beat=_beat,
            generation_id=params.normalized_generation_id.strip(),
        )
    activity.logger.info("published %d call-log file(s), %d calls", result["files"], result["calls"])
    return {**result, "embed_requests": embedder.calls}
