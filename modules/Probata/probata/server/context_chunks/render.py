"""Chunk text: one line per message, ``[YYYY-MM-DD HH:MM] Sender: body``, plus the NIM input guards.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Times are UTC. A message body is collapsed to one line (every run of whitespace, newlines included, becomes one
space), so one line is always exactly one message and a chunker's character offset maps back to a message index.
"""

from __future__ import annotations

from datetime import UTC, datetime

from server.context_chunks.config import MAX_INPUT_CHARS

BLANK_PLACEHOLDER = "(empty)"
NO_TEXT = "(attachment)"


def one_line(text: str | None) -> str:
    return " ".join((text or "").split())


def render_line(at: datetime | None, sender: str, body: str | None) -> str:
    stamp = "unknown time" if at is None else (at.astimezone(UTC) if at.tzinfo else at).strftime("%Y-%m-%d %H:%M")
    who = one_line(sender) or "Unknown"
    return f"[{stamp}] {who}: {one_line(body) or NO_TEXT}"


def nim_input(text: str) -> str:
    """The NIM input guards. The whole request fails when any input holds the lowercase text ``data:image/`` or is
    blank (owner notes 2026-09-26), so rewrite rather than drop: the batch keeps its length and order."""
    text = text.replace("data:image/", "data: image/")
    if len(text) > MAX_INPUT_CHARS:
        text = text[:MAX_INPUT_CHARS]
    return text if text.strip() else BLANK_PLACEHOLDER
