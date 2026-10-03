"""chunking.chonkie-recursive — Chonkie recursive chunker as a registry tool.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

One chunker, one job (see ``chunking/__init__.py``). Runs ``server.context_chunks.chunker`` engine ``recursive_1000``.
The library is imported inside the call, so registry discovery is safe in an image without it.
"""

from __future__ import annotations

from typing import Any

from server.tools.registry import register

from ._common import CHONKIE_VERSION, FORMAT_MESSAGE_LINES, run_chunker


@register(
    id="chunking.chonkie-recursive",
    capability="chunk.message_spans",
    provenance="chonkie (chonkie-ai) via server/context_chunks/chunker.py engine recursive_1000",
    tool_version=CHONKIE_VERSION,
    formats=(FORMAT_MESSAGE_LINES,),
    quality={FORMAT_MESSAGE_LINES: "fallback"},
)
def chunk_chonkie_recursive(payload: dict[str, Any]) -> dict[str, Any]:
    """Chonkie RecursiveChunker, splitting on the largest separator that fits and recursing, chunks of about 1000 characters.

    Formats: message_lines, one rendered message per line, given as payload['lines'] (list of strings) or
    payload['path'] (a UTF-8 file, one message per line). Optional payload['overlap'] (default 2 messages) and
    payload['max_chars'] (default 7500, the embedder's input limit).
    Side effects: none; returns {spans: [[first, last], ...], stats} as inclusive message indexes, and writes nothing.
    Prefers paragraph and line breaks over mid-line cuts.
    Pick it when threads contain multi-line messages.
    The default for chat threads is the Neural distilbert chunker, which only the temporal-worker can run (it needs a
    transformer model); this model-free tool is the selectable alternative.
    """
    return run_chunker(payload, "recursive_1000")
