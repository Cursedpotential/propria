"""Recover a run's operating mode only from an explicit durable receipt.

Byline: Claude Code · Fable 5.1 · 2026-09-20.
Byline: Codex · GPT-6.1-Sol · 2026-10-05.

The same case belongs to DEV and LIVE, so a matter ID cannot prove a run's
operating mode. Older receipts without an explicit flag remain unavailable.
"""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from uuid import UUID

from app.service import matter_mode
from app.types.matter_mode import MatterMode

_MODES: tuple[MatterMode, ...] = ("DEV", "LIVE")


def mode_for_operation(matter_id: UUID | None, operating_mode: str | None) -> MatterMode | None:
    """Accept an explicit canonical receipt only for the configured actual case.

    Inputs: durable matter UUID and canonical receipt flag. Output: mode or None.
    Effects: none. Pick over matter inference because both policies share a case.
    """
    if matter_id is None or operating_mode not in _MODES:
        return None
    try:
        return operating_mode if matter_mode.configured_matter_id(operating_mode) == matter_id else None
    except matter_mode.MatterModeError:
        return None


def rebind(preview_handle: str, matter_id: UUID | None, operating_mode: str | None) -> MatterMode | None:
    """Cache a handle only when durable scope and explicit policy are proven.

    Inputs: opaque handle, durable matter and flag. Output: mode or None.
    Effects: updates process cache only with proof. Pick for catalog recovery;
    require also enforces the requested policy on an individual operation.
    """
    proven = mode_for_operation(matter_id, operating_mode)
    if proven is not None:
        matter_mode.bind_preview_mode(preview_handle, proven)
    return proven


async def require(
    preview_handle: str,
    mode: MatterMode,
    fetch_binding: Callable[[str], Awaitable[tuple[UUID | None, str | None]]],
) -> None:
    """Require a matching explicit policy, recovering a lost cache binding once.

    Inputs: handle, requested mode and durable binding reader. Output: None or
    MatterModeError. Effects: one GET/cache update when unbound, never persistence.
    Pick over rebind when mismatch must reject rather than filter catalog rows.
    """
    try:
        matter_mode.require_preview_mode(preview_handle, mode)
        return
    except matter_mode.MatterModeError as error:
        if "cannot be verified" not in error.detail:
            raise
    matter_id, operating_mode = await fetch_binding(preview_handle)
    rebind(preview_handle, matter_id, operating_mode)
    matter_mode.require_preview_mode(preview_handle, mode)
