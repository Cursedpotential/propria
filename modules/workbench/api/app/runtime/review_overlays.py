"""Authenticated BFF routes for the Review metadata screen and context review.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

    GET  /api/proffer/previews/{handle}/metadata?mode=&subject_sha256=
    POST /api/proffer/previews/{handle}/metadata/corrections?mode=
    GET  /api/proffer/previews/{handle}/messages/{message_id}/context-review?mode=&horizon=
    POST /api/proffer/previews/{handle}/messages/{message_id}/context-review?mode=
    POST /api/proffer/previews/{handle}/messages/{message_id}/foreshadowing?mode=

Every write carries the authenticated Authentik subject; the browser never
names an actor. The context-review read defaults to the as-lived horizon and an
as-lived answer has no foreshadowing member at all.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, HTTPException, Path, Query, Request
from fastapi.responses import JSONResponse

from app.service.context_review import read_context_review, write_context_review, write_foreshadowing
from app.service.proffer import ProfferError
from app.service.source_metadata import correct_metadata, metadata_screen
from app.types.context_review import ContextReviewReceipt, ContextReviewRequest, ForeshadowingRequest, Horizon
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor
from app.types.source_metadata import MetadataCorrectionReceipt, MetadataCorrectionRequest, MetadataScreenResponse

router = APIRouter(prefix="/api/proffer", tags=["proffer"])

PreviewHandle = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{32,128}$")]
MessageID = Annotated[
    str, Path(pattern=r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
]
Digest = Annotated[str | None, Query(pattern=r"^[0-9a-f]{64}$")]


def _actor(request: Request) -> ProfferDecisionActor:
    subject_uid = str(getattr(request.state, "subject_uid", "")).strip()
    username = str(getattr(request.state, "principal", "")).strip()
    if not subject_uid or not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    return ProfferDecisionActor(subject_uid=subject_uid, username=username)


def _http(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


@router.get("/previews/{preview_handle}/metadata", response_model=MetadataScreenResponse)
async def metadata_screen_endpoint(
    preview_handle: PreviewHandle,
    mode: Annotated[MatterMode, Query()],
    subject_sha256: Digest = None,
):
    """Every metadata fact the platform holds for one file of the run, with its sidecars."""
    try:
        return await metadata_screen(preview_handle, mode=mode, subject_sha256=subject_sha256)
    except ProfferError as error:
        raise _http(error) from None


@router.post(
    "/previews/{preview_handle}/metadata/corrections",
    response_model=MetadataCorrectionReceipt,
    status_code=201,
)
async def metadata_correction_endpoint(
    preview_handle: PreviewHandle,
    body: MetadataCorrectionRequest,
    request: Request,
    mode: Annotated[MatterMode, Query()],
):
    """Append one attributed correction overlay; the observed value is never written."""
    try:
        return await correct_metadata(preview_handle, body, _actor(request), mode=mode)
    except ProfferError as error:
        raise _http(error) from None


@router.get("/previews/{preview_handle}/messages/{message_id}/context-review")
async def context_review_endpoint(
    preview_handle: PreviewHandle,
    message_id: MessageID,
    mode: Annotated[MatterMode, Query()],
    horizon: Annotated[Horizon, Query()] = "as_lived",
) -> JSONResponse:
    """Review history of one message. `foreshadowing` is present only for horizon=hindsight."""
    try:
        view = await read_context_review(preview_handle, message_id, mode=mode, horizon=horizon)
    except ProfferError as error:
        raise _http(error) from None
    content = view.model_dump(mode="json")
    if horizon != "hindsight":
        content.pop("foreshadowing", None)
    return JSONResponse(content=content, headers={"Cache-Control": "no-store"})


@router.post(
    "/previews/{preview_handle}/messages/{message_id}/context-review",
    response_model=ContextReviewReceipt,
    status_code=201,
)
async def context_review_write_endpoint(
    preview_handle: PreviewHandle,
    message_id: MessageID,
    body: ContextReviewRequest,
    request: Request,
    mode: Annotated[MatterMode, Query()],
):
    """Append one review revision: to, about, about the child, relevant."""
    try:
        return await write_context_review(preview_handle, message_id, body, _actor(request), mode=mode)
    except ProfferError as error:
        raise _http(error) from None


@router.post(
    "/previews/{preview_handle}/messages/{message_id}/foreshadowing",
    response_model=ContextReviewReceipt,
    status_code=201,
)
async def foreshadowing_write_endpoint(
    preview_handle: PreviewHandle,
    message_id: MessageID,
    body: ForeshadowingRequest,
    request: Request,
    mode: Annotated[MatterMode, Query()],
):
    """Set or clear the hindsight-only foreshadowing flag."""
    try:
        return await write_foreshadowing(preview_handle, message_id, body, _actor(request), mode=mode)
    except ProfferError as error:
        raise _http(error) from None
