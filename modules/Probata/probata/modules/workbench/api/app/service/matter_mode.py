"""Single-case identity and fail-closed operating-mode binding for Workbench.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.

The registry is a cache of explicit durable operating-mode receipts. Recovery
uses the engine's operation field, never case identity; missing historical
mode remains unbound. DEV and LIVE always use one configured actual case.
"""

from __future__ import annotations

from threading import RLock
from uuid import UUID

from app.config import settings
from app.types.matter_mode import MatterMode


class MatterModeError(Exception):
    def __init__(self, detail: str, status_code: int):
        self.detail = detail
        self.status_code = status_code
        super().__init__(detail)


_preview_modes: dict[str, MatterMode] = {}
_preview_modes_lock = RLock()


def _configured_uuid(raw: str, *, identity: str) -> UUID:
    label = f"canonical {identity} identity"
    if not raw.strip():
        raise MatterModeError(f"{label} is not configured", 503)
    try:
        value = UUID(raw.strip())
    except ValueError:
        raise MatterModeError(f"{label} configuration is invalid", 503) from None
    if value.int == 0 or str(value).startswith(("deadbeef-", "cafebabe-")):
        raise MatterModeError(f"{label} cannot be a placeholder", 503)
    return value


def configured_matter_id(mode: MatterMode) -> UUID:
    """Resolve one neutral matter ID independent of canonical operating policy.

    Input: DEV/LIVE. Output: UUID or configuration error. Effects: none.
    Pick over title/sole-matter discovery; REAL env is input-only rollout fallback.
    """
    if mode not in {"DEV", "LIVE"}:
        raise MatterModeError("Unknown operating mode", 422)
    raw = settings.proffer_matter_id or settings.proffer_real_matter_id
    return _configured_uuid(raw, identity="matter")


def configured_court_case_id(mode: MatterMode) -> UUID:
    """Resolve one neutral court-case ID independent of operating policy.

    Input: DEV/LIVE. Output: UUID or configuration error. Effects: none.
    Pick with configured_matter_id, never old TEST identities.
    """
    if mode not in {"DEV", "LIVE"}:
        raise MatterModeError("Unknown operating mode", 422)
    raw = settings.proffer_court_case_id or settings.proffer_real_court_case_id
    return _configured_uuid(raw, identity="court-case")


def require_matter(mode: MatterMode, matter_id: UUID) -> UUID:
    configured = configured_matter_id(mode)
    if matter_id != configured:
        raise MatterModeError("matter_id does not match the canonical case", 409)
    return configured


def require_scope(mode: MatterMode, matter_id: UUID, court_case_id: UUID) -> tuple[UUID, UUID]:
    configured_matter = require_matter(mode, matter_id)
    configured_court_case = configured_court_case_id(mode)
    if court_case_id != configured_court_case:
        raise MatterModeError("court_case_id does not match the canonical case", 409)
    return configured_matter, configured_court_case


def bind_preview_mode(preview_handle: str, mode: MatterMode) -> None:
    with _preview_modes_lock:
        existing = _preview_modes.get(preview_handle)
        if existing is not None and existing != mode:
            raise MatterModeError("preview handle is already bound to a different matter mode", 502)
        _preview_modes[preview_handle] = mode


def require_preview_mode(preview_handle: str, mode: MatterMode) -> None:
    with _preview_modes_lock:
        existing = _preview_modes.get(preview_handle)
    if existing is None:
        raise MatterModeError(
            "preview operating mode cannot be verified; old runs need an explicit durable mode receipt",
            409,
        )
    if existing != mode:
        raise MatterModeError("preview handle belongs to a different matter mode", 409)


def _clear_preview_modes_for_tests() -> None:
    with _preview_modes_lock:
        _preview_modes.clear()
