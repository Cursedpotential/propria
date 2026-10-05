"""The mobile Imported view: sources, threads, messages, calls and message search. Read-only.

Byline: Claude Code · Sonnet · 2026-10-02
Updated: Codex · GPT-6.1-Sol · 2026-10-05 (cohesive domain split).

Step 5 of the six steps (Preview). What it reads, and from where:

- Sources, threads, messages and calls: the platform database, `context.*` (the source versions the
  Proffer runs created and their normalized records) joined to `working.*` (the first-party /
  third-party projection and `call_log` once it has rows). Through `app.repo.imported_pg`.
- Run state (running, awaiting review, parked, failed): the Proffer starter's operation list, the
  same read the desktop Review queue uses. A source with an approved decision is committed whatever
  the operation list says; if the list is unreachable the view says so instead of guessing.
- Message search: Weaviate ProfferChunks20261002 (conversation chunks and call-log files; the messages themselves
  stay in Postgres), queried with a dict `where` filter.

An export is the original file (for example one SMS Backup & Restore XML); the Proffer run split it
into one derived file per conversation, and each derived file is its own source version. The view
lists the export as the source and the conversations inside it as threads.
"""

from __future__ import annotations

import asyncio
import base64
import re
import time
from datetime import datetime
from typing import Any, Callable, TypeVar

import httpx as httpx

from app.config import settings as settings
from app.repo import imported_pg as pg
from app.service import proffer
from app.service.matter_mode import MatterModeError, configured_matter_id
from app.service.proffer_errors import ProfferError

from app.service.imported_people import (
    _People as _People,
    _digits as _digits,
    looks_like_phone_value as looks_like_phone_value,
    _pretty_phone as _pretty_phone,
    describe_export as describe_export,
    _participant as _participant,
)
from app.service.imported_sources import (
    _version_status as _version_status,
    _is_media as _is_media,
    _file_status as _file_status,
    file_rows as file_rows,
    _exports as _exports,
    sources as sources,
    _export_for as _export_for,
    summary as summary,
)
from app.service.imported_threads import (
    _thread_title as _thread_title,
    source_threads as source_threads,
    _message as _message,
    thread_messages as thread_messages,
)
from app.service.imported_calls import (
    _call_party as _call_party,
    calls as calls,
)
from app.service.imported_search import (
    _snippet as _snippet,
    _names_label as _names_label,
    search as search,
    review_queue as review_queue,
)
from app.service.imported_numbers import (
    number_records as number_records,
    _activity as _activity,
    identity as identity,
    number_status as number_status,
    unknown_numbers as unknown_numbers,
)

ImportedError = pg.ImportedError

_STATUS_ORDER = ["failed", "parked", "awaiting_review", "running", "not_finished", "committed", "skipped"]
_LIFECYCLE_STATUS = {
    "completed": "committed",
    "running": "running",
    "awaiting_repair_decision": "parked",
    "awaiting_preview_decision": "awaiting_review",
    "failed": "failed",
    "cancelled": "failed",
    "unavailable": "failed",
}

_cache: dict[str, tuple[float, Any]] = {}


_CachedValue = TypeVar("_CachedValue")

_lifecycle_task: asyncio.Task | None = None
_LIFECYCLE_FRESH_S = 45.0

_PATH_DEVICE = re.compile(r"/(sms-backup-restore(?:-calls)?)/(\d{10})(?:/|$)")


def _cached(key: str, ttl: float, build: Callable[[], _CachedValue]) -> _CachedValue:
    """Return a TTL-cached value, building and caching only on a miss."""
    now = time.monotonic()
    hit = _cache.get(key)
    if hit and now - hit[0] < ttl:
        return hit[1]
    value = build()
    _cache[key] = (now, value)
    return value


def live_matter() -> str:
    """Resolve the single canonical case used by Live Imported views."""
    try:
        return str(configured_matter_id("LIVE"))
    except MatterModeError as error:
        raise ImportedError(error.detail, error.status_code) from None


def encode_id(*parts: str) -> str:
    """Encode reference parts as an opaque URL-safe ID.

    Input: reference parts. Output: unpadded base64 ID; no I/O.
    Pick for Imported source, conversation and cursor references.
    """
    return base64.urlsafe_b64encode("\n".join(parts).encode()).decode().rstrip("=")


def decode_id(value: str, count: int) -> list[str]:
    """Decode and validate an opaque Imported reference.

    Inputs: ID and expected part count. Output: parts, or a clean 404; no I/O.
    Pick for source and conversation references, not raw external locators.
    """
    try:
        raw = base64.urlsafe_b64decode(value + "=" * (-len(value) % 4)).decode()
        parts = raw.split("\n")
    except (ValueError, UnicodeError):
        raise ImportedError("Unknown reference", 404) from None
    if len(parts) != count or not all(parts) or len(raw) > 2048:
        raise ImportedError("Unknown reference", 404)
    return parts


def _cursor_encode(ts: Any, row_id: str) -> str:
    """Encode the timestamp and row ID of a keyset page boundary."""
    return encode_id(ts.isoformat() if hasattr(ts, "isoformat") else str(ts), row_id)


def _cursor_decode(cursor: str | None) -> tuple[str | None, str | None]:
    """Validate a keyset timestamp/UUID cursor, preserving its 404/422 errors."""
    if not cursor:
        return None, None
    ts, row_id = decode_id(cursor, 2)
    try:
        datetime.fromisoformat(ts)
    except ValueError:
        raise ImportedError("Unknown cursor", 422) from None
    if not re.fullmatch(r"[0-9a-fA-F-]{36}", row_id):
        raise ImportedError("Unknown cursor", 422)
    return ts, row_id


def _people() -> _People:
    """Read registry identities through the shared ten-second cache."""
    return _cached("people", 10, lambda: _People(pg.people()))


async def _refresh_lifecycles() -> None:
    """Read every Proffer operation of the live case (the starter takes seconds per page)."""
    found: dict[str, str] = {}
    cursor: str | None = None
    try:
        for _ in range(12):
            params: dict[str, str | int] = {"limit": 100}
            if cursor:
                params["cursor"] = cursor
            response = await proffer._request("GET", "/reference-import/operations", params=params)
            payload = response.json()
            for item in payload.get("items") or []:
                ref = item.get("source_version_ref")
                if ref and item.get("matter_id") == live_matter():
                    found[ref] = item.get("lifecycle") or ""
            cursor = payload.get("next_cursor")
            if not cursor:
                break
    except (ProfferError, ValueError, ImportedError):
        previous = _cache.get("lifecycles")
        _cache["lifecycles"] = (time.monotonic(), previous[1] if previous else None)
        return
    _cache["lifecycles"] = (time.monotonic(), found)


def warm() -> None:
    """Start the run-state read in the background; callers never wait on it."""
    global _lifecycle_task
    if _lifecycle_task is None or _lifecycle_task.done():
        _lifecycle_task = asyncio.create_task(_refresh_lifecycles())


async def _lifecycles() -> dict[str, str] | None:
    """source_version_id -> Proffer lifecycle, or None until the first read has finished.

    The starter's operation list takes about ten seconds in all, so a phone never waits for it: the
    last answer is served at once and refreshed in the background. A source with an approved
    decision is committed whatever this says (that is a PostgreSQL fact).
    """
    hit = _cache.get("lifecycles")
    if hit is None or time.monotonic() - hit[0] > _LIFECYCLE_FRESH_S:
        warm()
    return hit[1] if hit else None


def _iso(value: Any) -> str | None:
    """Serialize a present datetime, preserving absent values as None."""
    return value.isoformat() if value is not None else None


def invalidate() -> None:
    """Forget cached registry and activity reads after an identity change."""
    for key in [k for k in _cache if k in ("people", "entity-activity", "unlinked") or k.startswith("activity:") or k.startswith("sv:")]:
        _cache.pop(key, None)
