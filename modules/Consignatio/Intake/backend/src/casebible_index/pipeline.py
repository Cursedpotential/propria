"""The Coco super index pipeline: catalog or filesystem in, Parquet + Weaviate out.

> Byline: Claude Code · Opus 5 · 2026-09-22 (rewritten for streaming and bucket reads;
> original Claude Code · Opus 5 · 2026-09-18)

What changed on 2026-09-22, and why:

* **Catalog objects are read from the bucket**, not from a mount under
  ``CASEBIBLE_SOURCE_DIR``. The previous build listed the catalog correctly and then
  tried to open the object as a local path, which no deployment ever had.
* **No caps.** The 8 MiB reject, the 1,000,000-extracted-character raise and the
  512-chunk raise are gone (owner 2026-09-19/09-20: stream everything). Objects are read
  in windows, chunks are flushed to Parquet shards as they are produced, and peak memory
  is a window plus one shard regardless of object size.
* **Hits carry the vault key**, so a result can be opened from B2.
* **Weaviate is on by default** when a collection is configured.
"""

from __future__ import annotations

import hashlib
import os
import sys
from collections.abc import AsyncIterator
from contextlib import AsyncExitStack, asynccontextmanager
from dataclasses import replace
from pathlib import Path, PurePosixPath
from types import MappingProxyType

import cocoindex as coco
import httpx
from cocoindex.connectors import localfs
from cocoindex.ops.text import RecursiveSplitter
from cocoindex.resources.file import PatternFilePathMatcher

from .catalog_source import iter_catalog_objects, safe_relative_key
from .config import SUPPORTED_EXTENSIONS, Settings
from .filesystem_search import WeaviateSearchConfig
from .indexer import Hooks, IndexParams, MemberInfo, ObjectSpec, ZipHandle, index_object
from .models import DocumentEnrichment, TextChunk
from .nim import NimClient, NimError
from .object_store import ObjectStore, ReadCounters, configured_credentials
from .run_status import RunStatus
from .secrets import get_secret
from .source_runtime import source_lock
from .stream_extract import ARCHIVE_EXTENSIONS
from .vault_source import OBJECT_STORE, OBJECT_STORES, LocalStreamFile, VaultFile
from .weaviate_target import WEAVIATE_WRITER, WeaviateObjectWriter, declare_chunk
from .weaviate_target import ObjectSpec as ObjectSpec_W

NIM_CLIENT = coco.ContextKey[NimClient]("casebible_nim_client")
RUN_STATUS = coco.ContextKey[RunStatus]("intake_filesystem_run_status")
CATALOG_DSN = coco.ContextKey[str]("intake_catalog_dsn")
_splitter = RecursiveSplitter()
_EXCLUDED_SEGMENTS = frozenset({".git", ".review_hold", "to_be_deleted", "__pycache__"})
INDEXABLE_EXTENSIONS = frozenset(SUPPORTED_EXTENSIONS) | ARCHIVE_EXTENSIONS


def _catalog_key_is_indexable(key: str) -> bool:
    """Objects that enter the extract stage. Media and unsupported objects are not dropped: \
discovery records
    every one of them in the inventory (discovery.py), so the index represents the whole catalog."""
    path = safe_relative_key(key)
    if path is None or _EXCLUDED_SEGMENTS.intersection(path.parts):
        return False
    return path.suffix.casefold() in INDEXABLE_EXTENSIONS


def catalog_stable_key(provider: str, bucket: str, key: str, primary: bool) -> str:
    """CocoIndex component key: the plain key in the primary vault bucket (so existing memo \
state survives), and
    ``provider:bucket:key`` anywhere else, because two buckets may hold the same key."""
    return key if primary else f"{provider}:{bucket}:{key}"


def _fallback_enrichment(filename: str, notes: tuple[str, ...]) -> DocumentEnrichment:
    return DocumentEnrichment(
        title=filename,
        document_type="unknown",
        short_summary="No usable text was available for summarization.",
        confidence=0.0,
        review_notes=list(notes),
    )


class ChunkAccumulator:
    """Turn a stream of text pieces into chunks without holding the document.

    Each piece is split with the same recursive splitter the non-streaming build used.
    A trailing overlap is carried into the next piece so a chunk boundary between two
    windows is not a hard cut.
    """

    def __init__(self, chunk_size: int, chunk_overlap: int) -> None:
        self.chunk_size = chunk_size
        self.chunk_overlap = chunk_overlap
        self._carry = ""
        self._offset = 0
        self._ordinal = 0

    def _emit(self, text: str, offset: int) -> list[TextChunk]:
        """Emit one split as chunks, hard-wrapping it if it has no separator to split on.

        A minified JSON record or a single very long line comes back from the recursive
        splitter as ONE part far larger than ``chunk_size`` — a 341,373-character chunk in
        the vault's 61 MB conversations.json — and the embedding provider rejects it with a
        400. Streaming everything means never refusing such a record: it is cut on size,
        with the same overlap. Byline: Claude Code · Opus 5 · 2026-09-22.
        """
        chunks: list[TextChunk] = []
        if len(text) <= self.chunk_size:
            chunks.append(TextChunk(self._ordinal, offset, offset + len(text), text))
            self._ordinal += 1
            return chunks
        step = max(self.chunk_size - self.chunk_overlap, 1)
        for start in range(0, len(text), step):
            part = text[start:start + self.chunk_size]
            if not part:
                break
            chunks.append(
                TextChunk(self._ordinal, offset + start, offset + start + len(part), part)
            )
            self._ordinal += 1
            if start + self.chunk_size >= len(text):
                break
        return chunks

    def feed(self, piece: str) -> list[TextChunk]:
        text = self._carry + ("\n" if self._carry else "") + piece
        parts = _splitter.split(
            text, chunk_size=self.chunk_size, chunk_overlap=self.chunk_overlap,
            language="markdown",
        )
        if not parts:
            self._carry = text
            return []
        # Keep the last split back: the next piece may continue it.
        emit, self._carry = list(parts[:-1]), parts[-1].text
        chunks: list[TextChunk] = []
        for part in emit:
            chunks.extend(self._emit(part.text, self._offset + part.start.char_offset))
        # A carry that can never be split further would grow without bound; cut it now.
        if len(self._carry) > self.chunk_size * 4:
            carried, self._carry = self._carry, ""
            chunks.extend(self._emit(carried, self._offset + len(text) - len(carried)))
            self._offset += len(text)
            return chunks
        self._offset += max(len(text) - len(self._carry), 0)
        return chunks

    def finish(self) -> list[TextChunk]:
        if not self._carry.strip():
            return []
        carried, self._carry = self._carry, ""
        return self._emit(carried, self._offset)

    @property
    def count(self) -> int:
        return self._ordinal


EMBED_FLOOR_CHARS = 256


async def _embed(
    client: NimClient | None, texts: list[str], *, batch_size: int, dimensions: int
) -> tuple[list[list[float]], list[str]]:
    """Embed a batch. Returns the vectors and a per-text status.

    A chunk is bounded in CHARACTERS; the provider bounds it in TOKENS. Text with almost no
    whitespace — minified CSS or JSON inside a chat export — tokenizes far denser than prose,
    and a 2,400-character chunk of it is rejected with a 400 that failed the whole object.
    Streaming everything means the object survives: the batch is bisected, and a single text
    that still fails is halved until its vector can be produced, down to a floor. The FULL
    chunk text is still stored and still searchable lexically; only the vector covers a
    prefix, and that is recorded as ``embedding_status="truncated"`` rather than hidden.
    Byline: Claude Code · Opus 5 · 2026-09-22.
    """
    if client is None:
        return [[0.0] * dimensions for _ in texts], ["pending"] * len(texts)
    try:
        return await client.embed_documents(texts, batch_size=batch_size), ["ok"] * len(texts)
    except NimError:
        if len(texts) > 1:
            middle = len(texts) // 2
            left_vectors, left_status = await _embed(
                client, texts[:middle], batch_size=batch_size, dimensions=dimensions
            )
            right_vectors, right_status = await _embed(
                client, texts[middle:], batch_size=batch_size, dimensions=dimensions
            )
            return left_vectors + right_vectors, left_status + right_status
    text = texts[0]
    while len(text) > EMBED_FLOOR_CHARS:
        text = text[: len(text) // 2]
        try:
            vectors = await client.embed_documents([text], batch_size=1)
            return vectors, ["truncated"]
        except NimError:
            continue
    return [[0.0] * dimensions], ["failed"]


async def _open_remote_zip(file: VaultFile, closers: list) -> ZipHandle:
    """A ZIP in the bucket, listed and read by ranged GETs; the archive is never downloaded \
whole."""
    import asyncio

    from .archive_members import list_members, read_member

    store, key, size = file.store, file.key, file.catalog_object.byte_size
    client = httpx.Client(timeout=120.0, follow_redirects=False)
    closers.append(client)

    def listing(limit: int) -> list[MemberInfo]:
        return [
            MemberInfo(m.member_path, m.byte_size, m.crc32)
            for m in list_members(store, key, size, client, limit=limit)
        ]

    async def opener(member: str):
        iterator = read_member(store, key, size, client, member)
        while True:
            window = await asyncio.to_thread(next, iterator, None)
            if window is None:
                return
            yield window

    return ZipHandle(listing, opener)


@coco.fn(memo=True)
async def process_file(
    file,
    *,
    source_id: str,
    output_dir: Path,
    chunk_size: int,
    chunk_overlap: int,
    chunk_flush_size: int,
    summary_max_chars: int,
    embed_batch_size: int,
    embed_model: str,
    summary_model: str,
    embed_dimensions: int,
    embed_enabled: bool,
    summary_enabled: bool,
    weaviate_target: tuple[str, str, str] | None,
    conversation_mode: str = "chunk",
    ai_chat_mode: str = "route",
    chunker_name: str = "",
    overlap_messages: int = 0,
    index_archive_members: bool = False,
    archive_member_limit: int = 0,
    spool_dir: str = "",
) -> None:
    """CocoIndex wrapper: build the object spec and the hooks, run the extract-and-chunk stage \
(indexer.py)."""
    key = file.key
    byte_size = await file.size()
    if isinstance(file, VaultFile):
        fields = file.catalog_object.fields
        spec = ObjectSpec(
            key=key, relative_path=file.locator, vault_key=file.locator, member_path="",
            identity=file.identity,
            byte_size=byte_size, resolution=file.resolution, windows=file.windows, head=file.head,
            provider=str(fields.get("provider") or ""), bucket=str(fields.get("bucket") or ""),
        )
    else:
        metadata = await file._fetch_metadata()
        spec = ObjectSpec(
            key=key, relative_path=PurePosixPath(key).as_posix(), vault_key="", member_path="",
            identity=f"local:{byte_size}:{metadata.modified_time.isoformat()}", byte_size=byte_size,
            resolution=file.resolution, windows=file.windows, head=file.head, local=True,
        )
    client = coco.use_context(NIM_CLIENT) if embed_enabled or summary_enabled else None
    embed_client = client if embed_enabled else None
    status = coco.use_context(RUN_STATUS)
    closers: list = []

    async def embed(texts: list[str]):
        return await _embed(
            embed_client, texts, batch_size=embed_batch_size, dimensions=embed_dimensions
        )

    def on_rows(rows: list[dict]) -> None:
        if weaviate_target is None:
            return
        origin, collection, vector_name = weaviate_target
        for row in rows:
            # "failed" carries a zero vector; Weaviate rejects it and it would be a false
            # negative anyway. "truncated" is a real vector.
            if row["embedding_status"] in {"pending", "failed"}:
                continue
            declare_chunk(origin, collection, row["chunk_id"], ObjectSpec_W(
                properties={
                    "source_id": source_id, "source_path": spec.relative_path,
                    "vault_key": spec.vault_key, "resolution": spec.resolution,
                    "document_id": row["document_id"], "chunk_id": row["chunk_id"],
                    "filename": row["filename"], "text": row["text"], "embed_model": embed_model,
                },
                vectors={vector_name: row["embedding"]},
            ))

    def note_transformed(index_status: str) -> None:
        status.files_transformed += 1
        if index_status != "indexed":
            status.files_without_usable_text += 1
        if status.files_transformed % 25 == 0:
            status.save(output_dir, "running")

    async def open_archive(_: ObjectSpec) -> ZipHandle:
        return await _open_remote_zip(file, closers)

    params = IndexParams(
        source_id=source_id, output_dir=output_dir, chunk_size=chunk_size,
        chunk_overlap=chunk_overlap, chunk_flush_size=chunk_flush_size,
        summary_max_chars=summary_max_chars, embed_batch_size=embed_batch_size,
        embed_model=embed_model, summary_model=summary_model, embed_dimensions=embed_dimensions,
        embed_enabled=embed_enabled, summary_enabled=summary_enabled,
        conversation_mode=conversation_mode, ai_chat_mode=ai_chat_mode,
        chunker_name=chunker_name or None, overlap_messages=overlap_messages or None,
        index_archive_members=index_archive_members, archive_member_limit=archive_member_limit,
        spool_dir=Path(spool_dir) if spool_dir else None,
    )
    hooks = Hooks(
        embed=embed if embed_enabled else None,
        on_rows=on_rows if weaviate_target is not None else None,
        summarize=client.summarize if summary_enabled and client is not None else None,
        note_transformed=note_transformed,
        open_archive=open_archive if isinstance(file, VaultFile) else None,
    )
    try:
        await index_object(spec, params, hooks, ChunkAccumulator)
    finally:
        for closer in closers:
            closer.close()


@coco.fn
async def app_main(
    sourcedir: Path,
    output_dir: Path,
    source_id: str,
    chunk_size: int,
    chunk_overlap: int,
    chunk_flush_size: int,
    summary_max_chars: int,
    embed_batch_size: int,
    embed_model: str,
    summary_model: str,
    embed_dimensions: int,
    embed_enabled: bool,
    summary_enabled: bool,
    weaviate_target: tuple[str, str, str] | None,
    source_mode: str,
    catalog_query_file: Path,
    catalog_limit: int,
    catalog_path_prefix: str,
    conversation_mode: str = "chunk",
    ai_chat_mode: str = "route",
    chunker_name: str = "",
    overlap_messages: int = 0,
    index_archive_members: bool = False,
    archive_member_limit: int = 0,
    spool_dir: str = "",
    source_buckets: tuple[str, ...] = (),
) -> None:
    async def filesystem_items():
        patterns = [f"**/*{extension}" for extension in SUPPORTED_EXTENSIONS]
        files = localfs.walk_dir(
            sourcedir,
            recursive=True,
            path_matcher=PatternFilePathMatcher(
                included_patterns=patterns,
                excluded_patterns=[f"**/{segment}/**" for segment in sorted(_EXCLUDED_SEGMENTS)],
            ),
            live=False,
        )
        status = coco.use_context(RUN_STATUS)
        async for key, file in files.items():
            status.files_observed += 1
            yield key, LocalStreamFile(file.file_path)

    async def catalog_items():
        status = coco.use_context(RUN_STATUS)
        dsn = coco.use_context(CATALOG_DSN)
        query = catalog_query_file.read_text(encoding="utf-8")
        taken = 0
        wanted = set(source_buckets)
        async for obj in iter_catalog_objects(dsn, query):
            provider = str(obj.fields.get("provider") or "b2")
            bucket = str(obj.fields.get("bucket") or "")
            if wanted and f"{provider}:{bucket}" not in wanted:
                continue
            if catalog_path_prefix and not obj.key.startswith(catalog_path_prefix):
                continue
            if not _catalog_key_is_indexable(obj.key):
                continue
            primary = (not bucket) or (provider == "b2" and bucket == _settings.vault_bucket)
            obj = replace(obj, fields=MappingProxyType({**obj.fields, "primary": primary}))
            status.files_observed += 1
            yield catalog_stable_key(provider, bucket, obj.key, primary), VaultFile(obj)
            taken += 1
            if catalog_limit and taken >= catalog_limit:
                break

    handle = await coco.mount_each(
        process_file,
        catalog_items() if source_mode == "catalog" else filesystem_items(),
        source_id=source_id,
        output_dir=output_dir,
        chunk_size=chunk_size,
        chunk_overlap=chunk_overlap,
        chunk_flush_size=chunk_flush_size,
        summary_max_chars=summary_max_chars,
        embed_batch_size=embed_batch_size,
        embed_model=embed_model,
        summary_model=summary_model,
        embed_dimensions=embed_dimensions,
        embed_enabled=embed_enabled,
        summary_enabled=summary_enabled,
        weaviate_target=weaviate_target,
        conversation_mode=conversation_mode,
        ai_chat_mode=ai_chat_mode,
        chunker_name=chunker_name,
        overlap_messages=overlap_messages,
        index_archive_members=index_archive_members,
        archive_member_limit=archive_member_limit,
        spool_dir=spool_dir,
    )
    await handle.ready()


_settings = Settings.from_env().resolved()
_settings.validate(require_source=False)


def _weaviate_target_from_env() -> tuple[str, str, str] | None:
    """Weaviate is the default vector target when a collection is configured.

    Owner ruling (audit item I-6, 2026-09-22): Weaviate on by default, not opt-in.
    ``INTAKE_WEAVIATE_INDEX_ENABLED=0`` still turns it off explicitly.
    """
    collection = os.getenv("INTAKE_WEAVIATE_COLLECTION", "").strip()
    enabled = os.getenv("INTAKE_WEAVIATE_INDEX_ENABLED", "1" if collection else "0")
    if enabled != "1" or not collection:
        return None
    return (
        os.getenv("INTAKE_WEAVIATE_URL", "").strip(),
        collection,
        os.getenv("INTAKE_WEAVIATE_TEXT_VECTOR", "text_vector").strip(),
    )


_weaviate_target = _weaviate_target_from_env()
_EMBED_ENABLED = os.getenv("INTAKE_EMBED_MODE", "on").strip().casefold() != "deferred"
_SUMMARY_ENABLED = os.getenv("INTAKE_SUMMARY_MODE", "on").strip().casefold() == "on"


@asynccontextmanager
async def resources_lifespan(builder: coco.EnvironmentBuilder) -> AsyncIterator[None]:
    _settings.output_dir.mkdir(parents=True, exist_ok=True)
    _settings.state_dir.mkdir(parents=True, exist_ok=True)
    builder.settings.db_path = _settings.state_dir / "cocoindex.db"
    async with AsyncExitStack() as stack:
        if _settings.source_mode == "catalog":
            catalog_dsn = get_secret("INTAKE_CATALOG_DSN")
            if not catalog_dsn:
                raise ValueError("INTAKE_SOURCE_MODE=catalog needs INTAKE_CATALOG_DSN")
            builder.provide(CATALOG_DSN, catalog_dsn)
            credentials = configured_credentials(_settings.object_store_scheme)
            store_client = await stack.enter_async_context(
                httpx.AsyncClient(timeout=120.0, follow_redirects=False)
            )
            builder.provide(OBJECT_STORE, ObjectStore(
                credentials, _settings.vault_bucket, store_client, counters=READ_COUNTERS,
            ))
            #One read-only store per configured (provider, bucket). The primary vault
            #  bucket uses the
            # credentials above; any other provider resolves its own OBJECT_STORES_JSON entry.
            stores: dict[tuple[str, str], ObjectStore] = {}
            for entry in _settings.source_buckets:
                provider, _, bucket = entry.partition(":")
                scheme_credentials = (
                    credentials if provider == _settings.object_store_scheme
                    else configured_credentials(provider)
                )
                stores[(provider, bucket)] = ObjectStore(
                    scheme_credentials, bucket, store_client, counters=READ_COUNTERS,
                )
            builder.provide(OBJECT_STORES, stores)
        api_key = get_secret("NVIDIA_API_KEY")
        if not api_key and (_EMBED_ENABLED or _SUMMARY_ENABLED):
            raise ValueError("NVIDIA_API_KEY is not configured; set INTAKE_EMBED_MODE=deferred")
        client = await stack.enter_async_context(NimClient(
            api_key=api_key or "unset",
            base_url=_settings.nim_base_url,
            embed_model=_settings.embed_model,
            summary_model=_settings.summary_model,
            dimensions=_settings.embed_dimensions,
            timeout_seconds=_settings.timeout_seconds,
            max_retries=_settings.max_retries,
            max_concurrency=_settings.max_concurrency,
        ))
        builder.provide(NIM_CLIENT, client)
        if _weaviate_target is not None:
            origin, collection, target_vector = _weaviate_target
            target_config = WeaviateSearchConfig(
                url=origin, collection=collection, target_vector=target_vector,
                dimensions=_settings.embed_dimensions,
                api_key=get_secret("INTAKE_WEAVIATE_API_KEY") or "",
            )
            # Validate at startup, loudly, not on the first query (audit note 2026-09-22).
            target_config.validate()
            declared = os.getenv("INTAKE_WEAVIATE_EMBED_MODEL", _settings.embed_model)
            if declared != _settings.embed_model:
                raise ValueError("Filesystem Weaviate embedding model must match NIM")
            headers = {"Authorization": f"Bearer {target_config.api_key}"} \
                if target_config.api_key else {}
            http_client = await stack.enter_async_context(httpx.AsyncClient(
                headers=headers, timeout=20.0, follow_redirects=False,
            ))
            writer = WeaviateObjectWriter(target_config, http_client)
            await writer.verify_schema()
            builder.provide(WEAVIATE_WRITER, writer)
        yield


READ_COUNTERS = ReadCounters()


@coco.lifespan
async def coco_lifespan(builder: coco.EnvironmentBuilder) -> AsyncIterator[None]:
    with source_lock(_settings.lock_dir, _settings.source_id):
        status = RunStatus(_settings.source_id, str(_settings.source_dir))
        builder.provide(RUN_STATUS, status)

        async def observe_failure(exc: BaseException, context: coco.ExceptionContext) -> None:
            status.failure_events += 1
            # "Index run did not finish cleanly" with no reason cost this build several
            # blind rounds. The failure type and a bounded message go to stderr; provider
            # errors carry no corpus text (Claude Code · Opus 5 · 2026-09-22).
            print(
                f"[superindex] failure {status.failure_events}: "
                f"{type(exc).__name__}: {str(exc)[:400]}",
                file=sys.stderr, flush=True,
            )
            if status.failure_events == 1 or status.failure_events % 25 == 0:
                status.save(_settings.output_dir, "running_with_errors")

        builder.set_exception_handler(observe_failure)
        status.save(_settings.output_dir, "running")
        state = "finished"
        try:
            async with resources_lifespan(builder):
                yield
        except BaseException:
            state = "failed_or_interrupted"
            raise
        finally:
            if state == "finished" and status.failure_events:
                state = "finished_with_errors"
            status.save(_settings.output_dir, state)


app = coco.App(
    coco.AppConfig(
        name="IntakeFilesystem_" + hashlib.sha256(_settings.source_id.encode()).hexdigest()[:16],
        max_inflight_components=_settings.max_inflight_files,
    ),
    app_main,
    sourcedir=_settings.source_dir,
    output_dir=_settings.output_dir,
    source_id=_settings.source_id,
    chunk_size=_settings.chunk_size,
    chunk_overlap=_settings.chunk_overlap,
    chunk_flush_size=_settings.chunk_flush_size,
    summary_max_chars=_settings.summary_max_chars,
    embed_batch_size=_settings.embed_batch_size,
    embed_model=_settings.embed_model,
    summary_model=_settings.summary_model,
    embed_dimensions=_settings.embed_dimensions,
    embed_enabled=_EMBED_ENABLED,
    summary_enabled=_SUMMARY_ENABLED,
    weaviate_target=_weaviate_target,
    source_mode=_settings.source_mode,
    catalog_query_file=_settings.catalog_query_file,
    catalog_limit=_settings.catalog_limit,
    catalog_path_prefix=_settings.catalog_path_prefix,
    conversation_mode=_settings.conversation_mode,
    ai_chat_mode=_settings.ai_chat_mode,
    chunker_name=_settings.chunker_name,
    overlap_messages=_settings.overlap_messages,
    index_archive_members=_settings.index_archive_members,
    archive_member_limit=_settings.archive_member_limit,
    spool_dir=str(_settings.spool_dir) if _settings.spool_dir else "",
    source_buckets=_settings.source_buckets,
)
