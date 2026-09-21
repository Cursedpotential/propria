"""Recover a run's TEST/REAL binding from durable state instead of refusing it.

Byline: Claude Code · Fable 5.1 · 2026-09-20

`matter_mode` keeps preview-handle bindings in process memory, so every BFF
restart or redeploy orphaned every existing run: the Review catalog answered
"cannot prove TEST/REAL ownership ... restart the import" and no earlier run
could be opened (live 2026-09-20). The durable fact was always there: the
engine stores each run's `matter_id` (context.source_version), and the matter
decides the mode. This module re-derives the binding from that id. It never
guesses: an unknown or missing matter still fails closed.
"""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from uuid import UUID

from app.service import matter_mode
from app.types.matter_mode import MatterMode

_MODES: tuple[MatterMode, ...] = ("TEST", "REAL")


def mode_for_matter(matter_id: UUID | None) -> MatterMode | None:
    """Return the mode whose configured matter equals ``matter_id``, else None."""
    if matter_id is None:
        return None
    for candidate in _MODES:
        try:
            if matter_mode.configured_matter_id(candidate) == matter_id:
                return candidate
        except matter_mode.MatterModeError:
            continue  # that mode is not configured on this deployment
    return None


def rebind(preview_handle: str, matter_id: UUID | None) -> MatterMode | None:
    """Bind the handle to the mode its durable matter proves; None when unprovable."""
    proven = mode_for_matter(matter_id)
    if proven is not None:
        matter_mode.bind_preview_mode(preview_handle, proven)
    return proven


async def require(
    preview_handle: str,
    mode: MatterMode,
    fetch_matter_id: Callable[[str], Awaitable[UUID | None]],
) -> None:
    """`require_preview_mode`, recovering a lost in-memory binding exactly once."""
    try:
        matter_mode.require_preview_mode(preview_handle, mode)
        return
    except matter_mode.MatterModeError as error:
        if "no active TEST/REAL binding" not in error.detail:
            raise
    rebind(preview_handle, await fetch_matter_id(preview_handle))
    matter_mode.require_preview_mode(preview_handle, mode)
