"""The extract-and-chunk stage for one object: stream it, route it, chunk it, write Parquet. No \
CocoIndex runtime here.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02 (the body of ``pipeline.process_file`` of \
Claude Code · Opus 5 ·
> 2026-09-22, generalized from "one catalog object" to "one object spec", so an archive member \
goes through exactly
> the code a top-level object does)

What it adds to the 2026-09-22 body:

* **Routing** (routing.py): an AI chat export gets a document row with status \
``routed_ai_chat`` and no chunks (the
  AI-chat workstream owns it); a message export (SMS/MMS/call XML) is parsed into threads on \
disk and chunked as
  conversations with the shared chunking module (``INTAKE_CONVERSATION_MODE=chunk``, the \
default) or kept as a
  document row with status ``routed_message_export`` (``route``).
* **Archive members**: a ZIP is opened over ranged reads, each supported member is indexed as \
its own document
  (``member_path`` set, ``vault_key`` = the archive's key), a nested ZIP is opened from a local \
spool to a bounded
  depth, and every member that is not text (media, unsupported) is recorded in ONE \
members-inventory Parquet per
  archive instead of one tiny file each.
* **Content-addressed chunks**: every chunk row carries ``content_hash`` and \
``chunker_version`` (chunk_identity.py).

Stages are separable by flags: with ``embed`` unset the rows are written ``pending`` (the embed \
stage fills vectors
later), with ``on_rows`` unset nothing is declared to a CocoIndex target (the publish stage \
writes Weaviate).
"""

from __future__ import annotations

import asyncio
import hashlib
import tempfile
import uuid
import zipfile
from collections.abc import AsyncIterator, Awaitable, Callable
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path, PurePosixPath
from typing import Any

from .chunk_identity import content_hash, document_chunker_version
from .models import DocumentEnrichment, SourceMetadata, TextChunk
from .parquet_store import (
    SCHEMA_VERSION,
    chunk_rows_table,
    document_row_table,
    stable_document_id,
    streaming_artifact_id,
    vault_document_id,
    vault_version_id,
    write_chunk_shard,
    write_document_row,
)
from .routing import (
    KIND_ARCHIVE,
    KIND_DOCUMENT,
    STATUS_ROUTED_AI_CHAT,
    STATUS_ROUTED_MESSAGES,
    classify_key,
    inventory_status,
    maybe_ai_chat_name,
    sniff_ai_chat,
    sniff_message_export,
)
from .stream_extract import StreamOutcome, _spool, extract_stream

HEAD_BYTES = 131072
ARCHIVE_MAX_DEPTH = 3
WindowsFn = Callable[[], AsyncIterator[bytes]]


@dataclass(frozen=True)
class IndexParams:
    source_id: str
    output_dir: Path
    chunk_size: int
    chunk_overlap: int
    chunk_flush_size: int
    summary_max_chars: int
    embed_batch_size: int
    embed_model: str
    summary_model: str
    embed_dimensions: int
    embed_enabled: bool
    summary_enabled: bool
    conversation_mode: str = "chunk"
    ai_chat_mode: str = "route"
    chunker_name: str | None = None
    overlap_messages: int | None = None
    index_archive_members: bool = False
    archive_member_limit: int = 0
    spool_dir: Path | None = None


@dataclass
class ObjectSpec:
    """Everything the stage needs to know about one object, wherever it came from."""

    key: str  # the name whose extension routes it (the member path for an archive member)
    relative_path: str  # the catalog key (the archive's key for a member)
    vault_key: str
    member_path: str
    identity: str
    byte_size: int
    resolution: str
    windows: WindowsFn
    head: Callable[[int], Awaitable[bytes]]
    provider: str = ""
    bucket: str = ""
    local: bool = False
    extra: dict[str, Any] = field(default_factory=dict)


@dataclass
class Hooks:
    """The stage's seams to the outside. Each is optional; absence means that capability is off."""

    embed: Callable[[list[str]], Awaitable[tuple[list[list[float]], list[str]]]] | None = None
    on_rows: Callable[[list[dict]], None] | None = None
    summarize: Callable[..., Awaitable[DocumentEnrichment]] | None = None
    note_transformed: Callable[[str], None] | None = None
    beat: Callable[[str], None] | None = None
    open_archive: Callable[[ObjectSpec], Awaitable[Any]] | None = None
    engine: Any = None
    calls_engine: Any = None


def _fallback_enrichment(filename: str, notes: tuple[str, ...]) -> DocumentEnrichment:
    return DocumentEnrichment(
        title=filename,
        document_type="unknown",
        short_summary="No usable text was available for summarization.",
        confidence=0.0,
        review_notes=list(notes),
    )


def _ids(spec: ObjectSpec, params: IndexParams) -> tuple[str, str]:
    if spec.local:
        document_id = stable_document_id(params.source_id, spec.relative_path)
        version = vault_version_id(
            document_id,
            spec.identity,
            embed_model=params.embed_model,
            summary_model=params.summary_model,
        )
        return document_id, version
    document_id = vault_document_id(params.source_id, spec.identity)
    return document_id, vault_version_id(
        document_id,
        spec.identity,
        embed_model=params.embed_model,
        summary_model=params.summary_model,
    )


async def _conversation_chunks(
    spec: ObjectSpec, params: IndexParams, hooks: Hooks, outcome: StreamOutcome
) -> AsyncIterator[TextChunk]:
    """Message export -> conversation chunks. Disk-backed, one thread in memory at a time."""
    from .conversation_chunks import chunk_spool
    from .message_sources import MessageSpool, parse_message_xml

    spool_root = params.spool_dir or Path(tempfile.gettempdir())
    xml_path = await _spool(spec.windows(), ".xml")
    messages = MessageSpool(spool_root, f"messages-{uuid.uuid4().hex}")
    notes: list[str] = []

    def fill() -> None:
        with open(xml_path, "rb") as handle:
            for thread, message in parse_message_xml(handle, notes):
                messages.add(thread, message)
        messages.finish()

    try:
        await asyncio.to_thread(fill)
        outcome.method = "conversation_chunks"
        outcome.media_type = "application/xml"
        outcome.notes = outcome.notes + tuple(notes) + (f"{messages.count} message records read.",)
        if messages.count == 0:
            outcome.status = "skipped_no_text"
            return
        generator = chunk_spool(
            messages,
            spec.identity,
            chunker_name=params.chunker_name,
            overlap=params.overlap_messages,
            engine=hooks.engine,
            calls_engine=hooks.calls_engine,
            beat=hooks.beat,
        )
        while True:
            chunk = await asyncio.to_thread(next, generator, None)
            if chunk is None:
                break
            yield TextChunk(
                chunk.ordinal,
                chunk.char_start,
                chunk.char_end,
                chunk.text,
                meta={
                    "chunk_kind": "conversation",
                    "content_hash": chunk.content_hash,
                    "chunker": chunk.chunker,
                    "chunker_version": chunk.chunker_version,
                    "overlap": chunk.overlap,
                    "thread_id": chunk.thread_id,
                    "first_message_index": chunk.first_index,
                    "last_message_index": chunk.last_index,
                    "message_count": chunk.message_count,
                    "start_at": chunk.start_at,
                    "end_at": chunk.end_at,
                    "participants": list(chunk.participants),
                },
            )
    finally:
        await asyncio.to_thread(messages.close)
        await asyncio.to_thread(xml_path.unlink, True)


def _note_locator(
    output_dir: Path, document_id: str, spec: ObjectSpec, observed_at: datetime
) -> None:
    """Record where a document is NOW (one tiny immutable file per distinct locator)."""
    import pyarrow as pa

    from .parquet_store import _write_artifact_once

    locator = spec.vault_key + "|" + spec.member_path
    locator_hash = hashlib.sha256(locator.encode()).hexdigest()[:16]
    stem = f"{document_id}--{locator_hash}"
    table = pa.Table.from_pylist(
        [{"document_id": document_id, "vault_key": spec.vault_key, "relative_path": \
spec.relative_path,
          "member_path": spec.member_path, "observed_at": observed_at}],
        schema=pa.schema([("document_id", pa.string()), ("vault_key", pa.string()), \
("relative_path", pa.string()),
                          ("member_path", pa.string()), ("observed_at", pa.timestamp("us", \
tz="UTC"))]),
    )
    _write_artifact_once(output_dir / "datasets" / "locators" / f"{stem}.parquet", table)


async def index_object(
    spec: ObjectSpec,
    params: IndexParams,
    hooks: Hooks,
    accumulator_factory: Callable[[int, int], Any],
) -> dict[str, Any]:
    """Index one object. Returns a small result (status, counts) for the caller's receipt."""
    from .stream_extract import ARCHIVE_EXTENSIONS

    filename = PurePosixPath(spec.member_path or spec.key).name
    extension = PurePosixPath(spec.member_path or spec.key).suffix.casefold()
    document_id, version_id = _ids(spec, params)
    now = datetime.now(UTC)
    outcome = StreamOutcome()
    chunker_version_doc = document_chunker_version(params.chunk_size, params.chunk_overlap)

    # ----- routing: decided from a bounded head, before any full read -----
    routed: str | None = None
    conversation = False
    if (
        extension == ".json"
        and params.ai_chat_mode == "route"
        and maybe_ai_chat_name(spec.key)
        and sniff_ai_chat(await spec.head(HEAD_BYTES))
    ):
        routed = STATUS_ROUTED_AI_CHAT
    elif extension == ".xml" and sniff_message_export(await spec.head(HEAD_BYTES)):
        if params.conversation_mode == "chunk":
            conversation = True
        else:
            routed = STATUS_ROUTED_MESSAGES

    artifact = streaming_artifact_id(
        version_id,
        chunk_size=params.chunk_size,
        chunk_overlap=params.chunk_overlap,
        embed_model=params.embed_model,
        summary_model=params.summary_model,
    )
    #Already derived (same content, same derivation): never stream the object again. A
    #  file that was only MOVED
    #lands here with a new key, so a move costs no bucket read and no embedding (owner
    #  2026-09-22 10:48); the new
    # key is recorded as a locator for the publish stage to patch into Weaviate.
    existing = params.output_dir / "datasets" / "documents" / \
f"{document_id}--{version_id}--{artifact[:16]}.parquet"
    if await asyncio.to_thread(existing.exists):
        await asyncio.to_thread(_note_locator, params.output_dir, document_id, spec, now)
        return {"document_id": document_id, "status": "already_indexed", "chunks": 0,
                "members_indexed": 0, "members_inventoried": 0, "conversation": False}
    accumulator = accumulator_factory(params.chunk_size, params.chunk_overlap)
    summary_text: list[str] = []
    summary_chars = 0
    pending: list[TextChunk] = []
    total_chars = 0
    shard_count = 0
    chunk_count = 0

    def chunk_row(chunk: TextChunk, embedding: list[float], status: str) -> dict:
        text_hash = hashlib.sha256(chunk.text.encode("utf-8")).hexdigest()
        meta = dict(chunk.meta or {})
        row = {
            "document_id": document_id,
            "version_id": version_id,
            "artifact_id": artifact,
            "chunk_id": str(uuid.uuid5(uuid.NAMESPACE_URL, \
f"{version_id}:{chunk.ordinal}:{text_hash}")),
            "source_id": params.source_id,
            "relative_path": spec.relative_path,
            "vault_key": spec.vault_key,
            "resolution": spec.resolution,
            "member_path": spec.member_path,
            "filename": filename,
            "document_type": "unknown",
            "document_date": None,
            "title": filename,
            "short_summary": "",
            "chunk_ordinal": chunk.ordinal,
            "char_start": chunk.start,
            "char_end": chunk.end,
            "text": chunk.text,
            "text_sha256": text_hash,
            "token_estimate": max(1, len(chunk.text) // 4),
            "embedding": embedding,
            "embedding_status": status,
            "embedding_model": params.embed_model if params.embed_enabled else "",
            "schema_version": SCHEMA_VERSION,
            "indexed_at": now,
            "chunk_kind": "document",
            "content_hash": content_hash(chunk.text),
            "chunker": "recursive_markdown",
            "chunker_version": chunker_version_doc,
            "provider": spec.provider,
            "bucket": spec.bucket,
            "sha1": spec.identity.removeprefix("sha1:") if spec.identity.startswith("sha1:") \
else "",
            "extension": extension,
        }
        row.update(meta)
        return row

    async def flush(final: bool = False) -> None:
        nonlocal pending, shard_count
        while pending and (final or len(pending) >= params.chunk_flush_size):
            batch, pending = pending[: params.chunk_flush_size], pending[params.chunk_flush_size :]
            if hooks.embed is not None and params.embed_enabled:
                vectors, statuses = await hooks.embed([c.text for c in batch])
            else:
                vectors = [[0.0] * params.embed_dimensions for _ in batch]
                statuses = ["pending"] * len(batch)
            rows = [
                chunk_row(c, list(v), s)
                for c, v, s in zip(batch, vectors, statuses, strict=True)
            ]
            write_chunk_shard(
                params.output_dir,
                document_id=document_id,
                version_id=version_id,
                artifact=artifact,
                part=shard_count,
                table=chunk_rows_table(rows, params.embed_dimensions),
            )
            if hooks.on_rows is not None:
                hooks.on_rows(rows)
            shard_count += 1
            rows.clear()

    async def feed(chunks: list[TextChunk]) -> None:
        nonlocal chunk_count
        for chunk in chunks:
            pending.append(chunk)
            chunk_count += 1

    # ----- archive: members first, then the archive's own container row -----
    members_indexed = 0
    members_inventoried = 0
    if extension in ARCHIVE_EXTENSIONS and routed is None:
        outcome.status = "container"
        outcome.method = "archive_container"
        outcome.media_type = "application/zip" if extension == ".zip" else \
"application/octet-stream"
        if params.index_archive_members and extension == ".zip" and hooks.open_archive is not None:
            members_indexed, members_inventoried, notes = await _index_zip(
                spec, params, hooks, accumulator_factory, depth=0, document_id=document_id
            )
            outcome.notes = (
                f"{members_indexed} member(s) indexed as documents, "
                f"{members_inventoried} recorded in the "
                "members inventory.",
                *notes,
            )
        else:
            outcome.notes = ("Archive listed as a container; members are indexed separately.",)
    elif routed is not None:
        outcome.status = routed
        outcome.method = "routed"
        outcome.media_type = "application/json" if extension == ".json" else "application/xml"
        outcome.notes = (
            "AI chat export: owned by the AI-chat workstream (Proffer topic chunking); \
represented here by locator "
            "only."
            if routed == STATUS_ROUTED_AI_CHAT
            else "Message export kept as a document row; conversation chunking is off \
(INTAKE_CONVERSATION_MODE=route).",
        )
    elif conversation:
        async for chunk in _conversation_chunks(spec, params, hooks, outcome):
            total_chars += len(chunk.text)
            if summary_chars < params.summary_max_chars:
                summary_text.append(chunk.text[: params.summary_max_chars - summary_chars])
                summary_chars += len(summary_text[-1])
            await feed([chunk])
            await flush()
    else:
        async for piece in extract_stream(
            spec.key if not spec.member_path else spec.member_path,
            spec.windows(),
            outcome,
        ):
            total_chars += len(piece)
            if summary_chars < params.summary_max_chars:
                summary_text.append(piece[: params.summary_max_chars - summary_chars])
                summary_chars += len(summary_text[-1])
            await feed(accumulator.feed(piece))
            await flush()
        await feed(accumulator.finish())
    await flush(final=True)

    enrichment = _fallback_enrichment(filename, outcome.notes)
    coverage, coverage_ratio = ("none", 0.0)
    if outcome.status == "indexed" and params.summary_enabled and hooks.summarize is not None \
and summary_text:
        joined = "".join(summary_text)
        coverage = "full" if summary_chars >= total_chars else "head"
        coverage_ratio = min(1.0, summary_chars / total_chars) if total_chars else 0.0
        try:
            enrichment = await hooks.summarize(
                filename=filename,
                relative_path=spec.relative_path,
                text=joined,
                source_created_at=None,
                source_modified_at=None,
                coverage=coverage,
            )
        except Exception:  # noqa: BLE001 - a failed summary must not lose the extraction
            enrichment = _fallback_enrichment(filename, ("Summary pass failed for this object.",))

    source = SourceMetadata(
        relative_path=spec.relative_path,
        filename=filename,
        extension=extension,
        byte_size=spec.byte_size,
        created_at=None,
        modified_at=None,
        modified_ns=0,
        content_sha256=spec.identity,
    )
    write_document_row(
        params.output_dir,
        document_id=document_id,
        version_id=version_id,
        artifact=artifact,
        table=document_row_table(
            {
                "document_id": document_id,
                "version_id": version_id,
                "artifact_id": artifact,
                "source_id": params.source_id,
                "relative_path": source.relative_path,
                "vault_key": spec.vault_key,
                "resolution": spec.resolution,
                "member_path": spec.member_path,
                "filename": filename,
                "extension": extension,
                "media_type": outcome.media_type,
                "byte_size": spec.byte_size,
                "content_sha256": spec.identity,
                "source_created_at": None,
                "source_modified_at": None,
                "indexed_at": now,
                "title": enrichment.title or filename,
                "document_type": enrichment.document_type,
                "document_date": enrichment.document_date,
                "date_basis": enrichment.date_basis,
                "short_summary": enrichment.short_summary,
                "detailed_summary": enrichment.detailed_summary,
                "people": enrichment.people,
                "organizations": enrichment.organizations,
                "locations": enrichment.locations,
                "dates_mentioned": enrichment.dates_mentioned,
                "topics": enrichment.topics,
                "keywords": enrichment.keywords,
                "case_relevance": enrichment.case_relevance,
                "language": enrichment.language,
                "confidence": enrichment.confidence,
                "review_notes": enrichment.review_notes,
                "review_state": "unreviewed",
                "record_role": "machine_proposal",
                "index_status": outcome.status,
                "extraction_method": outcome.method,
                "extraction_notes": list(outcome.notes),
                "page_count": outcome.page_count,
                "text_char_count": total_chars,
                "chunk_count": chunk_count,
                "summary_coverage": coverage,
                "summary_coverage_ratio": coverage_ratio,
                "summary_model": params.summary_model if params.summary_enabled else "",
                "embedding_model": params.embed_model if params.embed_enabled else "",
                "embedding_dimensions": params.embed_dimensions,
                "schema_version": SCHEMA_VERSION,
            }
        ),
    )
    if hooks.note_transformed is not None:
        hooks.note_transformed(outcome.status)
    return {
        "document_id": document_id,
        "status": outcome.status,
        "chunks": chunk_count,
        "members_indexed": members_indexed,
        "members_inventoried": members_inventoried,
        "conversation": conversation,
    }


#-----------------------------------------------------------------------------------------------
#  archive members


@dataclass(frozen=True)
class MemberInfo:
    path: str
    size: int
    crc32: int


class ZipHandle:
    """A ZIP that is listed and read member by member. Remote (ranged reads) or a local spool, \
same interface."""

    def __init__(
        self,
        listing: Callable[[int], list[MemberInfo]],
        opener: Callable[[str], AsyncIterator[bytes]],
    ) -> None:
        self.list = listing
        self.open = opener


def local_zip_handle(path: Path) -> ZipHandle:
    def listing(limit: int) -> list[MemberInfo]:
        out: list[MemberInfo] = []
        with zipfile.ZipFile(path) as archive:
            for info in archive.infolist():
                if info.is_dir():
                    continue
                out.append(MemberInfo(info.filename, info.file_size, info.CRC))
                if limit and len(out) >= limit:
                    break
        return out

    async def opener(member: str) -> AsyncIterator[bytes]:
        archive = await asyncio.to_thread(zipfile.ZipFile, path)
        try:
            handle = await asyncio.to_thread(archive.open, member)
            try:
                while True:
                    window = await asyncio.to_thread(handle.read, 1024 * 1024)
                    if not window:
                        return
                    yield window
            finally:
                await asyncio.to_thread(handle.close)
        finally:
            await asyncio.to_thread(archive.close)

    return ZipHandle(listing, opener)


async def _index_zip(
    spec: ObjectSpec,
    params: IndexParams,
    hooks: Hooks,
    accumulator_factory: Callable[[int, int], Any],
    *,
    depth: int,
    document_id: str,
    handle: ZipHandle | None = None,
    path_prefix: str = "",
) -> tuple[int, int, list[str]]:
    """Index a ZIP's members. Returns (members indexed as documents, members inventoried, notes)."""
    import pyarrow as pa

    from .parquet_store import _write_artifact_once

    notes: list[str] = []
    zip_handle = (
        handle
        if handle is not None
        else (await hooks.open_archive(spec) if hooks.open_archive else None)
    )
    if zip_handle is None:
        return 0, 0, ["Archive could not be opened."]
    try:
        members = await asyncio.to_thread(zip_handle.list, params.archive_member_limit)
    except (zipfile.BadZipFile, OSError, ValueError) as error:
        return 0, 0, [
            f"Archive listing failed ({type(error).__name__}); "
            "the container row is kept."
        ]
    if params.archive_member_limit and len(members) >= params.archive_member_limit:
        notes.append(
            "Member listing stopped at "
            f"INTAKE_ARCHIVE_MEMBER_LIMIT={params.archive_member_limit}."
        )
    indexed = 0
    inventory_rows: list[dict] = []
    for member in members:
        member_path = f"{path_prefix}{member.path}"
        kind = classify_key(member.path)
        if kind == KIND_DOCUMENT:
            identity = f"{spec.identity}!{member_path}#{member.crc32:08x}"

            def make_windows(
                name: str = member.path, opener: Callable = zip_handle.open
            ) -> WindowsFn:
                return lambda: opener(name)

            async def member_head(
                n: int, name: str = member.path, opener: Callable = zip_handle.open
            ) -> bytes:
                data = b""
                stream = opener(name)
                try:
                    async for window in stream:
                        data += window
                        if len(data) >= n:
                            break
                finally:
                    await stream.aclose()
                return data[:n]

            member_spec = ObjectSpec(
                key=member.path,
                relative_path=spec.relative_path,
                vault_key=spec.vault_key,
                member_path=member_path,
                identity=identity,
                byte_size=member.size,
                resolution=spec.resolution,
                windows=make_windows(),
                head=member_head,
                provider=spec.provider,
                bucket=spec.bucket,
            )
            await index_object(member_spec, params, hooks, accumulator_factory)
            indexed += 1
        elif kind == KIND_ARCHIVE and depth < ARCHIVE_MAX_DEPTH and \
member.path.casefold().endswith(".zip"):
            spool = await _spool(zip_handle.open(member.path), ".zip")
            try:
                nested = local_zip_handle(spool)
                nested_spec = ObjectSpec(
                    key=member.path,
                    relative_path=spec.relative_path,
                    vault_key=spec.vault_key,
                    member_path=member_path,
                    identity=f"{spec.identity}!{member_path}#{member.crc32:08x}",
                    byte_size=member.size,
                    resolution=spec.resolution,
                    windows=lambda name=member.path: zip_handle.open(name),
                    head=lambda n: asyncio.sleep(0, b""),
                    provider=spec.provider,
                    bucket=spec.bucket,
                )
                sub_indexed, sub_inventoried, sub_notes = await _index_zip(
                    nested_spec, params, hooks, accumulator_factory, depth=depth + 1,
                    document_id=document_id, handle=nested, path_prefix=f"{member_path}!/",
                )
                indexed += sub_indexed
                notes.extend(sub_notes)
                inventory_rows.append(
                    {"member_path": member_path, "byte_size": member.size, "crc32": member.crc32,
                     "status": "nested_archive", "members_indexed": sub_indexed, \
"members_inventoried": sub_inventoried}
                )
            finally:
                await asyncio.to_thread(spool.unlink, True)
        else:
            status = inventory_status(kind) if kind != KIND_ARCHIVE else "container_too_deep"
            inventory_rows.append(
                {"member_path": member_path, "byte_size": member.size, "crc32": member.crc32, \
"status": status,
                 "members_indexed": 0, "members_inventoried": 0}
            )
    if inventory_rows:
        table = pa.Table.from_pylist(
            [
                {
                    "document_id": document_id,
                    "archive_key": spec.vault_key or spec.relative_path,
                    "depth": depth,
                    **row,
                }
                for row in inventory_rows
            ]
        )
        stem = f"{document_id}--d{depth}--{hashlib.sha256(path_prefix.encode()).hexdigest()[:12]}"
        await asyncio.to_thread(
            _write_artifact_once, params.output_dir / "datasets" / "members" / \
f"{stem}.parquet", table
        )
    return indexed, len(inventory_rows), notes
