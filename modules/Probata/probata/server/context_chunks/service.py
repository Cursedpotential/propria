"""The chunk units: chunk a thread (plan of spans), embed+publish a thread's chunks, publish call-log files.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Three units, each one job (the ATOMICITY rules), callable directly or as Temporal Activities
(server/temporal/chunk_activities.py):

    plan_thread / plan_source_version   CHUNK. Reads Postgres, runs the chunker, returns spans. No writes, no embed.
    publish_thread                      EMBED + PUBLISH one thread's chunks and replace that thread's old chunks.
    publish_call_files                  EMBED + PUBLISH one entry per call-log file.

References, not payloads: the plan carries message INDEXES and a digest, never message text; the publish unit re-reads
Postgres and refuses a plan whose thread has changed since it was cut. A thread that grew is re-chunked whole: the new
chunks are written first, then every chunk of that thread from another chunking (another digest) is deleted.
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any, Protocol

from server.context_chunks import ids
from server.context_chunks.chunker import chunk_spans, chunker_version
from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP
from server.context_chunks.model import CallFile, Thread, ThreadPlan, ThreadRef
from server.context_chunks.render import render_line
from server.context_chunks.source import Source
from server.context_chunks.store import RECORD_KIND_CALL_FILE, RECORD_KIND_CHUNK

ORIGIN_SYSTEM = "probata"
CALL_LOG_CORPUS = "call_log"
Heartbeat = Callable[[str], None]


class PlanStale(RuntimeError):
    """The thread's messages changed between chunking and publishing; chunk it again."""


class Embedder(Protocol):
    @property
    def model(self) -> str: ...

    def embed(self, texts: list[str]) -> list[list[float]]: ...


class Store(Protocol):
    def ensure_collection(self) -> list[str]: ...
    def upsert(self, objects: list[dict]) -> int: ...
    def delete_other_generations(self, thread_id: str, digest: str) -> int: ...
    def has_generation(self, thread_id: str, digest: str) -> bool: ...


@dataclass
class PublishResult:
    corpus: str
    thread_id: str
    chunks_written: int = 0
    stale_deleted: int = 0
    embed_texts: int = 0
    skipped_existing: bool = False


def thread_lines(thread: Thread) -> list[str]:
    return [render_line(m.at, m.sender, m.body) for m in thread.messages]


# ------------------------------------------------------------------ CHUNK
def plan_thread(
    thread: Thread,
    chunker: str = DEFAULT_CHUNKER,
    overlap: int = DEFAULT_OVERLAP,
    *,
    engine: Any = None,
    beat: Heartbeat | None = None,
) -> ThreadPlan:
    version = chunker_version(chunker, overlap)
    message_ids = [m.id for m in thread.messages]
    spans = chunk_spans(thread_lines(thread), chunker, overlap, chunker=engine, beat=beat)
    return ThreadPlan(
        corpus=thread.ref.corpus,
        thread_id=thread.ref.thread_id,
        message_count=len(message_ids),
        digest=ids.thread_digest(thread.ref.thread_id, message_ids, version),
        chunker=chunker,
        chunker_version=version,
        overlap=overlap,
        spans=[[a, b] for a, b in spans],
    )


def plan_source_version(
    source: Source,
    source_version_id: str,
    chunker: str = DEFAULT_CHUNKER,
    overlap: int = DEFAULT_OVERLAP,
    *,
    beat: Heartbeat | None = None,
) -> list[ThreadPlan]:
    """The plans of every thread the source version touched (first-party threads and third-party conversations)."""
    plans = []
    for ref in source.threads_for_source_version(source_version_id):
        if beat:
            beat(f"chunk {ref.corpus} {ref.thread_id}")
        plans.append(plan_thread(source.load_thread(ref), chunker, overlap, beat=beat))
    return plans


# ------------------------------------------------------------------ EMBED + PUBLISH
def _iso(at: datetime | None) -> str | None:
    if at is None:
        return None
    return (at if at.tzinfo else at.replace(tzinfo=UTC)).astimezone(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def build_chunk_objects(
    thread: Thread, plan: ThreadPlan, *, embed_model: str, now: datetime | None = None
) -> list[dict]:
    """The chunk objects without vectors: id, properties (message ids etc.), text."""
    lines = thread_lines(thread)
    stamp = _iso(now or datetime.now(UTC))
    objects = []
    for index, (first, last) in enumerate(plan.spans):
        members = thread.messages[first : last + 1]
        first_id, last_id = members[0].id, members[-1].id
        times = [m.at for m in members if m.at is not None]
        entity_ids = [e for m in members for e in m.participant_entity_ids]
        names = [n for m in members for n in m.participant_names]
        properties: dict[str, Any] = {
            "text": "\n".join(lines[first : last + 1]),
            "record_kind": RECORD_KIND_CHUNK,
            "corpus": plan.corpus,
            "thread_id": plan.thread_id,
            "thread_digest": plan.digest,
            "first_message_id": first_id,
            "last_message_id": last_id,
            "message_ids": [m.id for m in members],
            "participant_entity_ids": _uniq(entity_ids),
            "participant_names": _uniq(names),
            "source_version_ids": _uniq([m.source_version_id or "" for m in members]),
            "chunker": plan.chunker,
            "chunker_version": plan.chunker_version,
            "overlap": plan.overlap,
            "chunk_index": index,
            "message_count": len(members),
            "embed_model": embed_model,
            "origin_system": ORIGIN_SYSTEM,
            "object_id_construction": ids.CHUNK_ID_CONSTRUCTION,
            "indexed_at": stamp,
        }
        if thread.matter_id:
            properties["matter_id"] = thread.matter_id
        if times:
            properties["start_at"], properties["end_at"] = _iso(min(times)), _iso(max(times))
        objects.append(
            {
                "id": ids.chunk_object_id(plan.thread_id, first_id, last_id, plan.chunker_version),
                "properties": properties,
            }
        )
    return objects


def _uniq(values: list[str]) -> list[str]:
    return list(dict.fromkeys(v for v in values if v))


def publish_thread(
    source: Source,
    store: Store,
    embedder: Embedder,
    plan: ThreadPlan,
    *,
    beat: Heartbeat | None = None,
    resume: bool = True,
    batch: int = 32,
) -> PublishResult:
    """Embed and publish one thread's chunks, then replace the thread's chunks of any other chunking.

    Refuses (PlanStale) when the thread's ordered message ids no longer match the plan's digest. With ``resume`` a
    thread whose current chunking is already in the collection is left alone (a restarted run skips finished threads).
    """
    result = PublishResult(corpus=plan.corpus, thread_id=plan.thread_id)
    if resume and store.has_generation(plan.thread_id, plan.digest):
        stale = store.delete_other_generations(plan.thread_id, plan.digest)
        return PublishResult(plan.corpus, plan.thread_id, stale_deleted=stale, skipped_existing=True)
    thread = source.load_thread(ThreadRef(plan.corpus, plan.thread_id))
    if ids.thread_digest(plan.thread_id, [m.id for m in thread.messages], plan.chunker_version) != plan.digest:
        raise PlanStale(f"thread {plan.thread_id} changed after it was chunked; chunk it again")
    objects = build_chunk_objects(thread, plan, embed_model=embedder.model)
    store.ensure_collection()
    for start in range(0, len(objects), batch):
        part = objects[start : start + batch]
        vectors = embedder.embed([o["properties"]["text"] for o in part])
        if len(vectors) != len(part):
            raise RuntimeError(f"embedder returned {len(vectors)} vectors for {len(part)} chunks")
        for o, v in zip(part, vectors, strict=True):
            o["vector"] = v
        written = store.upsert(part)
        if written != len(part):
            raise RuntimeError(f"store wrote {written} of {len(part)} chunks")
        result.chunks_written += written
        result.embed_texts += len(part)
        if beat:
            beat(f"{plan.thread_id} {result.chunks_written}/{len(objects)}")
    result.stale_deleted = store.delete_other_generations(plan.thread_id, plan.digest)
    return result


# ------------------------------------------------------------------ CALL-LOG FILES
def build_call_file_object(call_file: CallFile, *, embed_model: str, now: datetime | None = None) -> dict:
    properties: dict[str, Any] = {
        "text": "\n".join(call_file.lines),
        "record_kind": RECORD_KIND_CALL_FILE,
        "corpus": CALL_LOG_CORPUS,
        "source_version_id": call_file.source_version_id,
        "source_version_ids": [call_file.source_version_id],
        "call_log_ids": call_file.call_ids,
        "participant_entity_ids": call_file.participant_entity_ids,
        "participant_names": call_file.participant_names,
        "message_count": len(call_file.call_ids),
        "chunker": "whole_file",
        "chunker_version": "call-file-v1",
        "embed_model": embed_model,
        "origin_system": ORIGIN_SYSTEM,
        "object_id_construction": ids.CALL_FILE_ID_CONSTRUCTION,
        "indexed_at": _iso(now or datetime.now(UTC)),
    }
    if call_file.matter_id:
        properties["matter_id"] = call_file.matter_id
    if call_file.first_at:
        properties["start_at"] = _iso(call_file.first_at)
    if call_file.last_at:
        properties["end_at"] = _iso(call_file.last_at)
    return {"id": ids.call_file_object_id(call_file.source_version_id), "properties": properties}


def publish_call_files(
    source: Source,
    store: Store,
    embedder: Embedder,
    source_version_id: str | None = None,
    *,
    beat: Heartbeat | None = None,
) -> dict:
    """One Weaviate object per call-log file (source version); the individual calls stay in Postgres.

    The vector embeds the start of the file (NIM input cut, config.MAX_INPUT_CHARS); full-text search covers all of it.
    """
    versions = source.call_source_versions(source_version_id)
    files = calls = 0
    if versions:
        store.ensure_collection()
    for sv in versions:
        call_file = source.load_call_file(sv)
        if not call_file.call_ids:
            continue
        obj = build_call_file_object(call_file, embed_model=embedder.model)
        (obj["vector"],) = embedder.embed([obj["properties"]["text"]])
        if store.upsert([obj]) != 1:
            raise RuntimeError(f"store did not write the call-log file {sv}")
        files += 1
        calls += len(call_file.call_ids)
        if beat:
            beat(f"call file {sv}")
    return {"files": files, "calls": calls}
