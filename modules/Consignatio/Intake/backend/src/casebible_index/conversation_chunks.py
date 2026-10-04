"""Conversation chunks for message exports, using the shared chunking module. One unit, one job:
chunk.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The chunker is Probata's ``server/context_chunks`` (Chonkie Neural distilbert, windowed, every chunk
reaching at
least ``overlap`` = 2 messages into the next). It is imported, never copied: in the container it is
added as a
build context (see deploy/superindex.compose.yml), on a checkout it is ``modules/Probata/probata``
on PYTHONPATH.
Only these names are used from it, so the public API another agent is finalizing has a small surface
to keep:

    server.context_chunks.render.render_line(at, sender, body) -> str
    server.context_chunks.chunker.chunk_spans(lines, name, overlap, *, chunker=None, beat=None) ->
    [(first, last)]
    server.context_chunks.chunker.chunker_version(name, overlap) -> str
    server.context_chunks.chunker.line_starts(lines) -> [offset]
    server.context_chunks.config.DEFAULT_CHUNKER / DEFAULT_OVERLAP

A chunk's identity is content-addressed (chunk_identity.py): the rendered text of its member
messages and the exact
chunker version. Nothing here knows how the chunk is stored or embedded.
"""

from __future__ import annotations

from collections.abc import Callable, Iterator
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from .chunk_identity import content_hash
from .message_sources import CALLS_THREAD, MessageSpool, SourceMessage

CALLS_CHUNKER = "fast_1000"


class ChunkCoreMissing(RuntimeError):
    """The shared chunking module is not importable in this environment."""


def _core() -> tuple[Any, Any, Any]:
    try:
        from server.context_chunks import chunker, config, render
    except ImportError as error:
        raise ChunkCoreMissing(
            "Probata's server/context_chunks is not importable. On a checkout put "
            "modules/Probata/probata on "
            "PYTHONPATH; in the container it is copied in by the compose build context "
            f"({error})"
        ) from error
    return chunker, config, render


def defaults() -> tuple[str, int]:
    """(chunker name, overlap) the shared module declares as its defaults."""
    _, config, _ = _core()
    return config.DEFAULT_CHUNKER, config.DEFAULT_OVERLAP


@dataclass(frozen=True)
class ConversationChunk:
    ordinal: int
    text: str
    content_hash: str
    chunker: str
    chunker_version: str
    overlap: int
    thread_id: str
    first_index: int
    last_index: int
    message_count: int
    char_start: int
    char_end: int
    start_at: datetime | None
    end_at: datetime | None
    participants: tuple[str, ...] = field(default_factory=tuple)


def chunk_thread(
    thread_id: str,
    messages: list[SourceMessage],
    *,
    chunker_name: str,
    overlap: int,
    first_ordinal: int = 0,
    engine: Any = None,
    beat: Callable[[str], None] | None = None,
    own_count: int | None = None,
    index_base: int = 0,
    char_base: int = 0,
) -> list[ConversationChunk]:
    """Cut one thread (messages already in order) into overlapping chunks.

    ``own_count`` / ``index_base`` / ``char_base`` serve segmented threads (chunk_spool): only
    chunks that START among
    the first ``own_count`` messages are returned (the rest are lookahead for the overlap), with
    message indexes and
    character offsets made global by the two bases."""
    if not messages:
        return []
    chunker, _, render = _core()
    lines = [render.render_line(m.at, m.sender, m.body) for m in messages]
    spans = chunker.chunk_spans(lines, chunker_name, overlap, chunker=engine, beat=beat)
    version = chunker.chunker_version(chunker_name, overlap)
    starts = chunker.line_starts(lines)
    out: list[ConversationChunk] = []
    for index, (first, last) in enumerate(spans):
        if own_count is not None and first >= own_count:
            break
        text = "\n".join(lines[first : last + 1])
        members = messages[first : last + 1]
        times = [m.at for m in members if m.at is not None]
        senders = tuple(dict.fromkeys(m.sender for m in members if m.sender))
        out.append(
            ConversationChunk(
                ordinal=first_ordinal + index,
                text=text,
                content_hash=content_hash(text),
                chunker=chunker_name,
                chunker_version=version,
                overlap=overlap,
                thread_id=thread_id,
                first_index=index_base + first,
                last_index=index_base + last,
                message_count=len(members),
                char_start=char_base + starts[first],
                char_end=char_base + starts[last] + len(lines[last]),
                start_at=min(times) if times else None,
                end_at=max(times) if times else None,
                participants=senders,
            )
        )
    return out


THREAD_SEGMENT_MESSAGES = 20_000


def _segments(
    messages: Iterator[SourceMessage], size: int, lookahead: int
) -> Iterator[tuple[list[SourceMessage], int]]:
    """(messages, own_count) per segment: ``size`` own messages plus ``lookahead`` messages of the
    next segment, so the
    last chunk of a segment can still reach into the next one. The final segment owns everything it
    holds."""
    buffer: list[SourceMessage] = []
    for message in messages:
        buffer.append(message)
        if len(buffer) == size + lookahead:
            yield buffer, size
            buffer = buffer[size:]
    if buffer:
        yield buffer, len(buffer)


def chunk_spool(
    spool: MessageSpool,
    file_identity: str,
    *,
    chunker_name: str | None = None,
    overlap: int | None = None,
    engine: Any = None,
    calls_engine: Any = None,
    beat: Callable[[str], None] | None = None,
    segment_messages: int = THREAD_SEGMENT_MESSAGES,
) -> Iterator[ConversationChunk]:
    """Every thread of a spooled message export. Memory is one SEGMENT of one thread
    (``segment_messages`` messages),
    not a whole thread: a contact with a million messages is chunked in segments of 20,000 (the
    constant is part of the
    identity of the result, so it does not change casually). A segment boundary is a chunk boundary,
    and the chunk
    before it still reaches ``overlap`` messages into the next segment. Ordinals run across the
    file."""
    default_name, default_overlap = defaults()
    name = chunker_name or default_name
    lap = default_overlap if overlap is None else overlap
    ordinal = 0
    for thread in spool.threads():
        is_calls = thread == CALLS_THREAD
        thread_overlap = 0 if is_calls else lap
        index_base = 0
        char_base = 0
        for segment, own in _segments(spool.messages(thread), segment_messages, thread_overlap):
            chunks = chunk_thread(
                f"{file_identity}#{thread}",
                segment,
                chunker_name=CALLS_CHUNKER if is_calls else name,
                overlap=thread_overlap,
                first_ordinal=ordinal,
                engine=calls_engine if is_calls else engine,
                beat=beat,
                own_count=own,
                index_base=index_base,
                char_base=char_base,
            )
            yield from chunks
            ordinal += len(chunks)
            if own < len(segment) or len(segment) == segment_messages + thread_overlap:
                rendered = _core()[2].render_line
                char_base += sum(len(rendered(m.at, m.sender, m.body)) + 1 for m in segment[:own])
            index_base += own
