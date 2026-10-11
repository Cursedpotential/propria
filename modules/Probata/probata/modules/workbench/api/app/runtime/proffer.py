"""Workbench BFF routes for the Proffer starter.

Byline: Codex · GPT-5 · 2026-08-28.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, HTTPException, Path, Query, Request

from app.repo.object_store_client import DEFAULT_SOURCE_ROOT_ID
from app.runtime.operating_mode import OperatingMode
from app.runtime.proffer_cancel import router as _cancel_router
from app.runtime.proffer_decoded import router as _decoded_router
from app.runtime.proffer_events import router as _events_router
from app.runtime.proffer_media import router as _media_router
from app.runtime.proffer_search_params import MessageSearchFilter
from app.runtime.proffer_upload import router as _upload_router
from app.service.proffer import (
    ProfferError,
    browse_sources,
    decide,
    decide_handler_selection,
    decide_repair,
    preview,
    preview_content,
    preview_messages,
    require_preview_mode,
    start,
)
from app.service.proffer_flags import create_potential_promotion_flag, list_potential_promotion_flags
from app.service.proffer_operations import list_operations, operation
from app.service.proffer_operator import operator_snapshot
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
    ProfferSourceBrowserResponse,
    ProfferStartRequest,
    ProfferStartResponse,
)
from app.types.proffer_flags import (
    ProfferPotentialPromotionFlag,
    ProfferPotentialPromotionFlagList,
    ProfferPotentialPromotionFlagRequest,
)
from app.types.proffer_operations import (
    ProfferOperationDetail,
    ProfferOperationLifecycle,
    ProfferOperationListResponse,
)
from app.types.proffer_operator import ProfferOperatorSnapshot

router = APIRouter(prefix="/api/proffer", tags=["proffer"])
router.include_router(_media_router)  # GET .../media/{sha256}: see app/runtime/proffer_media.py
router.include_router(_decoded_router)  # GET /decoded/*: SBV output before an ingest run exists
router.include_router(_events_router)  # GET .../events: see app/runtime/proffer_events.py
router.include_router(_cancel_router)  # POST .../cancel: see app/runtime/proffer_cancel.py
router.include_router(_upload_router)  # POST /upload, /staged/{id}/acquisition: see app/runtime/proffer_upload.py


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


def _decision_actor(request: Request) -> ProfferDecisionActor:
    subject_uid = str(getattr(request.state, "subject_uid", "")).strip()
    username = str(getattr(request.state, "principal", "")).strip()
    if not subject_uid or not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    try:
        return ProfferDecisionActor(subject_uid=subject_uid, username=username)
    except ValueError:
        raise HTTPException(status_code=401, detail="authenticated subject identity is invalid") from None


@router.get("/sources", response_model=ProfferSourceBrowserResponse)
def sources_endpoint(
    mode: Annotated[MatterMode, Query()] = "LIVE",
    root_id: Annotated[str, Query(min_length=1, max_length=64)] = DEFAULT_SOURCE_ROOT_ID,
    prefix: Annotated[str, Query(max_length=1024)] = "",
    continuation_token: Annotated[str | None, Query(max_length=4096)] = None,
    filter: Annotated[str, Query(max_length=256)] = "",
    filter_scope: Annotated[str, Query(pattern=r"^(root|folder)$")] = "root",
    file_type: Annotated[list[str] | None, Query()] = None,
    page_size: Annotated[int, Query(ge=1, le=500)] = 100,
):
    """Browse one configured object-store root without admitting a case.

    Inputs: root ID, relative prefix, optional filename filter and page cursor.
    Output: bounded files/folders and available configured roots. Effects: one
    provider listing; use for navigation, not processing authorization.
    """
    try:
        return browse_sources(
            mode=mode,
            root_id=root_id,
            prefix=prefix,
            continuation_token=continuation_token,
            filter_text=filter,
            filter_scope=filter_scope,
            file_types=tuple(file_type or ()),
            page_size=page_size,
        )
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/start", response_model=ProfferStartResponse, status_code=201)
async def start_endpoint(body: ProfferStartRequest, mode: OperatingMode):
    try:
        return await start(body, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


PreviewHandle = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{32,128}$")]


@router.post("/previews/{preview_handle}/decision", response_model=ProfferDecisionResponse)
async def decision_endpoint(
    preview_handle: PreviewHandle,
    body: ProfferDecisionRequest,
    request: Request,
    mode: OperatingMode,
):
    if not body.approved and not body.reason.strip():
        raise HTTPException(status_code=422, detail="a rejection decision requires a non-empty reason")
    actor = _decision_actor(request)
    try:
        return await decide(preview_handle, body, actor, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post(
    "/previews/{preview_handle}/handler-selection",
    response_model=ProfferHandlerSelectionDecisionResponse,
)
async def handler_selection_endpoint(
    preview_handle: PreviewHandle,
    body: ProfferHandlerSelectionDecisionRequest,
    request: Request,
    mode: OperatingMode,
):
    actor = _decision_actor(request)
    try:
        return await decide_handler_selection(preview_handle, body, actor, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post(
    "/previews/{preview_handle}/repair-decision",
    response_model=ProfferRepairDecisionResponse,
)
async def repair_decision_endpoint(
    preview_handle: PreviewHandle,
    body: ProfferRepairDecisionRequest,
    request: Request,
    mode: OperatingMode,
):
    actor = _decision_actor(request)
    try:
        return await decide_repair(preview_handle, body, actor, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/previews/{preview_handle}", response_model=ProfferPreviewResponse)
async def preview_endpoint(preview_handle: PreviewHandle, mode: OperatingMode):
    try:
        return await preview(preview_handle, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/operations", response_model=ProfferOperationListResponse)
async def operations_endpoint(
    status: Annotated[ProfferOperationLifecycle | None, Query()] = None,
    cursor: Annotated[str | None, Query(max_length=512)] = None,
    limit: Annotated[int, Query(ge=1, le=100)] = 50,
):
    try:
        return await list_operations(status=status, cursor=cursor, limit=limit)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/operations/{preview_handle}", response_model=ProfferOperationDetail)
async def operation_endpoint(preview_handle: PreviewHandle):
    try:
        return await operation(preview_handle)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/previews/{preview_handle}/operator", response_model=ProfferOperatorSnapshot)
async def operator_snapshot_endpoint(preview_handle: PreviewHandle, mode: OperatingMode):
    try:
        return await operator_snapshot(preview_handle, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/previews/{preview_handle}/messages", response_model=ProfferPreviewMessagesResponse)
async def preview_messages_endpoint(
    preview_handle: PreviewHandle,
    mode: OperatingMode,
    cursor: Annotated[str | None, Query(max_length=512)] = None,
    limit: Annotated[int, Query(ge=1, le=250)] = 100,
    search: MessageSearchFilter = None,
):
    try:
        return await preview_messages(preview_handle, mode=mode, cursor=cursor, limit=limit, search=search)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/previews/{preview_handle}/content", response_model=ProfferContentResponse)
async def preview_content_endpoint(
    preview_handle: PreviewHandle,
    mode: OperatingMode,
    record_cursor: Annotated[str | None, Query(max_length=512)] = None,
    chunk_cursor: Annotated[str | None, Query(max_length=512)] = None,
    limit: Annotated[int, Query(ge=1, le=250)] = 100,
):
    try:
        return await preview_content(
            preview_handle,
            mode=mode,
            record_cursor=record_cursor,
            chunk_cursor=chunk_cursor,
            limit=limit,
        )
    except ProfferError as error:
        raise _translate(error) from None


@router.get(
    "/previews/{preview_handle}/potential-promotion-flags",
    response_model=ProfferPotentialPromotionFlagList,
)
async def potential_promotion_flags_endpoint(
    preview_handle: PreviewHandle,
    mode: OperatingMode,
):
    try:
        await preview_content(
            preview_handle,
            mode=mode,
            record_cursor=None,
            chunk_cursor=None,
            limit=1,
        )
        return ProfferPotentialPromotionFlagList(flags=list_potential_promotion_flags(preview_handle, mode))
    except ProfferError as error:
        raise _translate(error) from None


@router.post(
    "/previews/{preview_handle}/potential-promotion-flags",
    response_model=ProfferPotentialPromotionFlag,
    status_code=201,
)
async def create_potential_promotion_flag_endpoint(
    preview_handle: PreviewHandle,
    body: ProfferPotentialPromotionFlagRequest,
    request: Request,
    mode: OperatingMode,
):
    actor = _decision_actor(request)
    try:
        await require_preview_mode(preview_handle, mode=mode)
        if body.scope not in ("record", "chunk"):
            raise HTTPException(status_code=422, detail="Preview content target is invalid or unsupported")
        return create_potential_promotion_flag(preview_handle, mode, body, actor)
    except ProfferError as error:
        raise _translate(error) from None
