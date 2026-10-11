"""Authenticated pass-through to the Proffer starter.

Byline: Codex · GPT-5 · 2026-08-28.
Byline amendment: Claude Code · Opus 5.5 · 2026-09-26 — an upstream error keeps its JSON body
on `ProfferError.payload` (the repair run refusal relays its checks).
Byline amendment: Claude Code · Opus 5.5 · 2026-09-27 — preview content reads moved to
app/service/proffer_content.py (DF-18: this module was over the 300-line cap).
"""

from __future__ import annotations

import json
import re
import stat
from pathlib import Path
from typing import Any

import httpx
from pydantic import ValidationError

from app.config import settings
from app.service import preview_mode_recovery
from app.service.matter_mode import (
    MatterModeError,
    configured_matter_id,
)
from app.service.proffer_errors import ProfferError, upstream_payload
from app.service.proffer_sources import browse_sources  # noqa: F401
from app.types.matter_mode import MatterMode
from app.types.proffer import (
    ProfferContentResponse,
    ProfferDecisionActor,
    ProfferDecisionRequest,
    ProfferDecisionResponse,
    ProfferHandlerSelectionDecisionRequest,
    ProfferHandlerSelectionDecisionResponse,
    ProfferPreviewMessagesResponse,
    ProfferPreviewResponse,
    ProfferRepairDecisionRequest,
    ProfferRepairDecisionResponse,
    ProfferUploadResponse,
)
from app.types.proffer_search import PreviewMessageFilter

_SERVICE_TOKEN = re.compile(r"[A-Za-z0-9\-._~+/]+={0,}")
_MIN_SERVICE_TOKEN_BYTES = 32
_MAX_SERVICE_TOKEN_BYTES = 4096
_MAX_SERVICE_TOKEN_FILE_BYTES = _MAX_SERVICE_TOKEN_BYTES + 2
_SAFE_SERVICE_AUTH_ERROR = "Proffer service authentication is unavailable or invalid"


def _service_authorization_headers() -> dict[str, str]:
    """Read and validate the mounted Proffer service token for one request."""
    path = Path(settings.proffer_service_token_file)
    if not path.is_absolute():
        raise ProfferError(_SAFE_SERVICE_AUTH_ERROR, 503)
    try:
        file_info = path.lstat()
        if not stat.S_ISREG(file_info.st_mode) or not (
            _MIN_SERVICE_TOKEN_BYTES <= file_info.st_size <= _MAX_SERVICE_TOKEN_FILE_BYTES
        ):
            raise ProfferError(_SAFE_SERVICE_AUTH_ERROR, 503)
        with path.open("rb") as secret_file:
            raw = secret_file.read(_MAX_SERVICE_TOKEN_FILE_BYTES + 1)
    except OSError:
        raise ProfferError(_SAFE_SERVICE_AUTH_ERROR, 503) from None
    raw = raw.rstrip(b"\r\n")
    if not (_MIN_SERVICE_TOKEN_BYTES <= len(raw) <= _MAX_SERVICE_TOKEN_BYTES) or b"\x00" in raw:
        raise ProfferError(_SAFE_SERVICE_AUTH_ERROR, 503)
    try:
        token = raw.decode("utf-8")
    except UnicodeDecodeError:
        raise ProfferError(_SAFE_SERVICE_AUTH_ERROR, 503) from None
    if _SERVICE_TOKEN.fullmatch(token) is None:
        raise ProfferError(_SAFE_SERVICE_AUTH_ERROR, 503)
    return {"Authorization": f"Bearer {token}"}


def _detail(response: httpx.Response) -> str:
    payload = upstream_payload(response)
    return payload["detail"] if payload and isinstance(payload.get("detail"), str) else response.text[:500]


async def _request(method: str, path: str, **kwargs: Any) -> httpx.Response:
    if not settings.proffer_starter_url.strip():
        raise ProfferError("Proffer starter is not configured", 503)
    headers = {key: value for key, value in dict(kwargs.pop("headers", {})).items() if key.lower() != "authorization"}
    headers.update(_service_authorization_headers())
    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            response = await client.request(
                method, f"{settings.proffer_starter_url.rstrip('/')}{path}", headers=headers, **kwargs
            )
    except httpx.HTTPError as error:
        raise ProfferError(f"Proffer starter unreachable: {error}") from error
    if response.status_code >= 400:
        detail = _detail(response) or "Proffer starter rejected the request"
        raise ProfferError(detail, response.status_code, payload=upstream_payload(response))
    return response


def _mode_call(call, *args):
    try:
        return call(*args)
    except MatterModeError as error:
        raise ProfferError(error.detail, error.status_code) from None


def _mode_payload(payload: Any, label: str, mode: MatterMode) -> dict[str, Any]:
    """Add the BFF-only mode echo while rejecting contradictory upstream data."""
    if not isinstance(payload, dict):
        raise ProfferError(f"Proffer starter returned an invalid {label}", 502)
    upstream_mode = payload.get("matter_mode")
    if upstream_mode is not None and upstream_mode != mode:
        raise ProfferError(f"Proffer starter returned {label} for a different matter mode", 502)
    result = dict(payload)
    result["matter_mode"] = mode
    return result


def _require_mode_configuration(mode: MatterMode) -> None:
    _mode_call(configured_matter_id, mode)


from app.service.proffer_start import operation_binding as _operation_binding
from app.service.proffer_start import start as start


async def _require_mode(preview_handle: str, mode: MatterMode) -> None:
    """Mode check that survives a BFF restart (see preview_mode_recovery)."""
    try:
        await preview_mode_recovery.require(preview_handle, mode, _operation_binding)
    except MatterModeError as error:
        raise ProfferError(error.detail, error.status_code) from None


async def require_preview_mode(preview_handle: str, *, mode: MatterMode) -> None:
    """Bind a write to the handle's configured matter mode before admission."""
    await _require_mode(preview_handle, mode)


def _validated(model, payload: Any, label: str):
    try:
        return model.model_validate(payload)
    except (ValidationError, ValueError, TypeError) as error:
        raise ProfferError(f"Proffer starter returned an invalid {label}", 502) from error


def _json_payload(response: httpx.Response, label: str) -> Any:
    """Normalize malformed upstream JSON to the BFF's fail-closed 502 contract."""
    try:
        return response.json()
    except (ValueError, json.JSONDecodeError, UnicodeDecodeError, TypeError) as error:
        raise ProfferError(f"Proffer starter returned malformed JSON for {label}", 502) from error


async def decide(
    preview_handle: str,
    request: ProfferDecisionRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ProfferDecisionResponse:
    from app.service.proffer_repair import decide as implementation

    return await implementation(preview_handle, request, actor, mode=mode)


async def decide_handler_selection(
    preview_handle: str,
    request: ProfferHandlerSelectionDecisionRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ProfferHandlerSelectionDecisionResponse:
    from app.service.proffer_handler_selection import decide_handler_selection as implementation

    return await implementation(preview_handle, request, actor, mode=mode)


def _repair_idempotency_key(
    preview_handle: str,
    request: ProfferRepairDecisionRequest,
    actor: ProfferDecisionActor,
) -> str:
    from app.service.proffer_repair import repair_idempotency_key

    return repair_idempotency_key(preview_handle, request, actor)


async def decide_repair(
    preview_handle: str,
    request: ProfferRepairDecisionRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ProfferRepairDecisionResponse:
    from app.service.proffer_repair import decide_repair as implementation

    return await implementation(preview_handle, request, actor, mode=mode)


async def preview(preview_handle: str, *, mode: MatterMode) -> ProfferPreviewResponse:
    await _require_mode(preview_handle, mode)
    response = await _request("GET", f"/reference-import/previews/{preview_handle}")
    result = _validated(
        ProfferPreviewResponse,
        _mode_payload(_json_payload(response, "preview snapshot"), "preview snapshot", mode),
        "preview snapshot",
    )
    if result.preview_handle != preview_handle:
        raise ProfferError("Proffer preview snapshot correlation failed", 502)
    return result


async def preview_messages(
    preview_handle: str,
    *,
    mode: MatterMode,
    cursor: str | None,
    limit: int,
    search: PreviewMessageFilter | None = None,
) -> ProfferPreviewMessagesResponse:
    from app.service.proffer_message_page import preview_messages as implementation

    return await implementation(preview_handle, mode=mode, cursor=cursor, limit=limit, search=search)


async def preview_content(
    preview_handle: str, *, mode: MatterMode, record_cursor: str | None, chunk_cursor: str | None, limit: int
) -> ProfferContentResponse:
    from app.service.proffer_content import preview_content as implementation

    return await implementation(
        preview_handle, mode=mode, record_cursor=record_cursor, chunk_cursor=chunk_cursor, limit=limit
    )


async def preview_content_target(
    preview_handle: str, *, mode: MatterMode, scope: str, target_id: str
) -> tuple[str, bool]:
    from app.service.proffer_content import preview_content_target as implementation

    return await implementation(preview_handle, mode=mode, scope=scope, target_id=target_id)


async def open_preview_event_stream(preview_handle: str, *, mode: MatterMode, last_event_id: int | None):
    from app.service.proffer_streams import open_preview_event_stream as implementation

    return await implementation(preview_handle, mode=mode, last_event_id=last_event_id)


async def validated_preview_events(response, *, preview_handle: str, mode: MatterMode, last_event_id: int | None):
    from app.service.proffer_streams import validated_preview_events as implementation

    async for event in implementation(response, preview_handle=preview_handle, mode=mode, last_event_id=last_event_id):
        yield event


async def open_upload_stream(body, *, mode: MatterMode, content_type: str | None, content_length: str | None):
    from app.service.proffer_streams import open_upload_stream as implementation

    _require_mode_configuration(mode)
    return await implementation(body, content_type=content_type, content_length=content_length)


async def complete_upload_response(client, response, *, mode: MatterMode) -> ProfferUploadResponse:
    from app.service.proffer_streams import complete_upload_response as implementation

    return await implementation(client, response, mode=mode)
