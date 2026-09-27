"""Exact preview content reads: one page of records and chunks, and one target.

Split out of app/service/proffer.py on 2026-09-27 (DF-18: that module was over the
300-line cap). The engine call goes through ``app.service.proffer`` so tests that
patch ``proffer._request`` keep covering it. Byline: Claude Code · Opus 5.5 · 2026-09-27.
"""

from __future__ import annotations

from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferContentResponse


async def preview_content(
    preview_handle: str,
    *,
    mode: MatterMode,
    record_cursor: str | None,
    chunk_cursor: str | None,
    limit: int,
) -> ProfferContentResponse:
    """Read exact retained-package, generic-record, and pre-publication chunk data."""
    await proffer._require_mode(preview_handle, mode)
    params: dict[str, str | int] = {"limit": limit}
    if record_cursor:
        params["record_cursor"] = record_cursor
    if chunk_cursor:
        params["chunk_cursor"] = chunk_cursor
    response = await proffer._request("GET", f"/reference-import/previews/{preview_handle}/content", params=params)
    result = proffer._validated(
        ProfferContentResponse,
        proffer._mode_payload(proffer._json_payload(response, "preview content page"), "preview content page", mode),
        "preview content page",
    )
    if result.preview_handle != preview_handle or result.matter_mode != mode:
        raise ProfferError("Proffer preview content correlation failed", 502)
    return result


async def preview_content_target(
    preview_handle: str, *, mode: MatterMode, scope: str, target_id: str
) -> tuple[str, bool]:
    """Resolve one current-attempt record or chunk independently of page cursors."""
    await proffer._require_mode(preview_handle, mode)
    if scope not in ("record", "chunk") or not target_id or len(target_id) > 512:
        raise ProfferError("Preview content target is invalid or unsupported", 422)
    response = await proffer._request(
        "GET",
        f"/reference-import/previews/{preview_handle}/content-target",
        params={"scope": scope, "target_id": target_id},
    )
    payload = proffer._json_payload(response, "exact preview content target")
    if (
        not isinstance(payload, dict)
        or payload.get("preview_handle") != preview_handle
        or not isinstance(payload.get("attempt_id"), str)
        or not payload["attempt_id"]
        or type(payload.get("found")) is not bool
    ):
        raise ProfferError("Proffer starter returned an invalid exact preview content target", 502)
    return payload["attempt_id"], payload["found"]
