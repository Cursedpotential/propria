"""Relay bounded raw previews to Go without parsing XML or starting ELT."""

import json
import httpx
from pydantic import ValidationError
from starlette.concurrency import run_in_threadpool
from app.config import settings
from app.service.proffer import _service_authorization_headers
from app.service.proffer_errors import ProfferError
from app.service.source_inspection import _metadata
from app.types.proffer import ProfferDecisionActor
from app.types.source_message_preview import (
    SourceMessagePreviewPending, SourceMessagePreviewReady, SourceMessagePreviewRequest,
)

MAX_RESPONSE_BYTES = 512 * 1024
BASE_PATH = "/reference-import/source-preview"


async def _relay(method: str, path: str, actor: ProfferDecisionActor, body: dict | None = None):
    """Read a capped Go response using the existing mounted service credential.

    Inputs: fixed route, middleware-validated actor and optional JSON request. Outputs: typed page and
    HTTP status. Effects: one fifteen-second upstream call, cancelled with its task.
    Pick for raw viewing only; never follows upstream URLs or starts processing.
    """
    if not settings.proffer_starter_url.strip():
        raise ProfferError("Message preview is unavailable", 503)
    try:
        async with httpx.AsyncClient(timeout=15.0, follow_redirects=False) as client:
            async with client.stream(method, settings.proffer_starter_url.rstrip("/") + path,
                                     headers={**_service_authorization_headers(),
                                              "X-authentik-uid": actor.subject_uid,
                                              "X-authentik-username": actor.username}, json=body) as response:
                data = bytearray()
                async for chunk in response.aiter_bytes(chunk_size=16384):
                    if len(data) + len(chunk) > MAX_RESPONSE_BYTES:
                        raise ProfferError("Message preview exceeded its response limit", 502)
                    data.extend(chunk)
                status = response.status_code
    except httpx.TimeoutException:
        raise ProfferError("Message preview timed out; try again", 504) from None
    except httpx.HTTPError:
        raise ProfferError("Message preview is unavailable", 503) from None
    if status not in (200, 202):
        allowed = {409, 410, 422, 503, 504}
        raise ProfferError({409: "The source or preview cursor changed; reopen the file",
                            410: "This preview expired; reopen the file",
                            422: "This file cannot be previewed as SMS XML"}.get(status, "Message preview is unavailable"),
                           status if status in allowed else 502)
    try:
        payload = json.loads(data)
        model = SourceMessagePreviewReady if status == 200 else SourceMessagePreviewPending
        result = model.model_validate(payload)
    except (ValueError, ValidationError, TypeError):
        raise ProfferError("Message preview returned an invalid page", 502) from None
    if status == 202 and result.poll_url != f"{BASE_PATH}/{result.page_id}":
        raise ProfferError("Message preview returned an invalid poll location", 502)
    return result, status


async def start_page(request: SourceMessagePreviewRequest, actor: ProfferDecisionActor):
    """HEAD-pin one allowlisted source and request one read-only Go message page.

    Input: original listing metadata, middleware actor, optional cursor and limit. Output: ready
    page or pending handle. Effects: HEAD and atomic preview only, no case gate,
    acquisition, extraction, indexing or promotion. Pick before full processing.
    """
    _, head = await run_in_threadpool(_metadata, request)
    version = head.get("VersionId")
    version = version if isinstance(version, str) and version.lower() != "null" else None
    body = {"source_ref": request.source_ref, "expected_etag": str(head["ETag"]),
            "limit": request.limit}
    if version:
        body["version_id"] = version
    if request.cursor:
        body["cursor"] = request.cursor
    result, status = await _relay("POST", BASE_PATH, actor, body)
    if status == 200 and (result.source.source_ref != request.source_ref
                         or result.source.etag.strip('"') != str(head["ETag"]).strip('"')
                         or result.source.byte_length != int(head["ContentLength"])
                         or (version and result.source.version_id != version)):
        raise ProfferError("Message preview source correlation failed", 502)
    return result, status


async def poll_page(page_id: str, actor: ProfferDecisionActor):
    """Poll a validated opaque handle at the fixed Go preview route.

    Input: route-validated page ID and middleware actor. Output: capped ready/pending response.
    Effects: read only; use for a pending page, never a supplied remote URL.
    """
    result, status = await _relay("GET", f"{BASE_PATH}/{page_id}", actor)
    if result.page_id != page_id:
        raise ProfferError("Message preview page correlation failed", 502)
    return result, status
