"""chunking.chonkie-token — Chonkie token chunker as a registry tool.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

One chunker, one job (see ``chunking/__init__.py``). Runs ``server.context_chunks.chunker`` engine ``token_1000``.
The library is imported inside the call, so registry discovery is safe in an image without it.
"""

from __future__ import annotations

from typing import Any

from server.tools.registry import register

from ._common import CHONKIE_VERSION, FORMAT_MESSAGE_LINES, run_chunker


@register(
    id="chunking.chonkie-token",
    capability="chunk.message_spans",
    provenance="chonkie (chonkie-ai) via server/context_chunks/chunker.py engine token_1000",
    tool_version=CHONKIE_VERSION,
    formats=(FORMAT_MESSAGE_LINES,),
    quality={FORMAT_MESSAGE_LINES: "fallback"},
)
def chunk_chonkie_token(payload: dict[str, Any]) -> dict[str, Any]:
    """Chonkie TokenChunker over characters, fixed 1000-character windows (no overlap of its own).

    Formats: message_lines, one rendered message per line, given as payload['lines'] (list of strings) or
    payload['path'] (a UTF-8 file, one message per line). Optional payload['overlap'] (default 2 messages) and
    payload['max_chars'] (default 7500, the embedder's input limit).
    Side effects: none; returns {spans: [[first, last], ...], stats} as inclusive message indexes, and writes nothing.
    Cuts at a fixed size and ignores sentence and line edges, so a message can be split mid-line before the message-boundary snap.
    Pick it as the plain baseline; it is the cheapest and the least meaningful split.
    The default for chat threads is the Neural distilbert chunker, which only the temporal-worker can run (it needs a
    transformer model); this model-free tool is the selectable alternative.
    """
    return run_chunker(payload, "token_1000")
