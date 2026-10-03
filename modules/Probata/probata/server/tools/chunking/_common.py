"""Shared plumbing for the chunking tools (underscore prefix: not a tool module).

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

import time
from pathlib import Path
from typing import Any

FORMAT_MESSAGE_LINES = "message_lines"
CHONKIE_VERSION = "chonkie-1.7.0"
_MAX_INPUT_BYTES = 64 * 1024 * 1024


def read_lines(payload: dict[str, Any]) -> list[str]:
    """The rendered message lines: ``payload['lines']``, or the UTF-8 file ``payload['path']`` split on newlines."""
    if "lines" in payload:
        lines = payload["lines"]
        if not isinstance(lines, list) or not all(isinstance(line, str) for line in lines):
            raise ValueError("lines must be a list of strings, one per message")
        return list(lines)
    if "path" not in payload:
        raise ValueError(
            "payload must include 'lines' (one string per message) or 'path' (a file with one message per line)"
        )
    path = Path(str(payload["path"]))
    if not path.is_file():
        raise FileNotFoundError(path)
    if path.stat().st_size > _MAX_INPUT_BYTES:
        raise ValueError(f"{path.name} is larger than the {_MAX_INPUT_BYTES:,}-byte chunking input cap")
    return path.read_bytes().decode("utf-8", errors="replace").splitlines()


def run_chunker(payload: dict[str, Any], name: str) -> dict[str, Any]:
    """Cut the lines into overlapping message spans with the context_chunks chunker called ``name``."""
    from server.context_chunks.chunker import chunk_spans, chunker_version
    from server.context_chunks.config import DEFAULT_OVERLAP, MAX_CHUNK_CHARS

    lines = read_lines(payload)
    overlap = int(payload.get("overlap", DEFAULT_OVERLAP))
    max_chars = int(payload.get("max_chars", MAX_CHUNK_CHARS))
    started = time.perf_counter()
    spans = chunk_spans(lines, name, overlap, max_chars=max_chars)
    return {
        "spans": [[first, last] for first, last in spans],
        "stats": {
            "method": name,
            "chunker_version": chunker_version(name, overlap),
            "message_count": len(lines),
            "chunk_count": len(spans),
            "overlap": overlap,
            "max_chars": max_chars,
            "elapsed_s": round(time.perf_counter() - started, 3),
        },
    }
