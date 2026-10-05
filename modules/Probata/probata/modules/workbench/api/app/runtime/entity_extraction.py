"""Workbench BFF routes for entity and event extraction.

Byline: Claude Code · Opus 5.5 · 2026-09-25

    POST /api/entities/extract                    start extraction for a run
    GET  /api/entities/extractions/{workflow_id}  its progress
    GET  /api/entities/proposals                  current entity + event proposals
    POST /api/entities/corrections                one owner edit (entity or event)
    GET  /api/entities/records/{record_id}        one record of the run (click a mention)
    GET  /api/entities/registry                   committed entities (merge target)
    POST /api/entities/validate                   pass/fail list before Run
    POST /api/entities/commit                     run the commit workflow
    GET  /api/entities/commits/{workflow_id}      its progress
    POST /api/events/from-record                  "event worth recalling" on a record

``GET /api/entities`` and ``POST /api/entities`` (governed entity search and
create, app/runtime/inspect.py) are unchanged; every route here is a subpath.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, Header, HTTPException, Path, Query, Request

from app.runtime.operating_mode import OperatingMode
from app.service import entity_extraction as service
from app.service.proffer_errors import ProfferError
from app.types.entity_extraction import (
    CommitStarted,
    CorrectionApplied,
    EntityCommitRequest,
    EntityCorrectionRequest,
    EntityExtractRequest,
    EntityRunRequest,
    EventFromRecordRequest,
    ExtractionStarted,
    IdempotencyKey,
    MarkedEvent,
    PreviewHandle,
    ProposalsResponse,
    RecordView,
    RegistrySearchResponse,
    ValidationReport,
    WorkflowProgress,
)
from app.types.proffer import ProfferDecisionActor

router = APIRouter(prefix="/api", tags=["entities"])

WorkflowId = Annotated[str, Path(min_length=8, max_length=512)]
RequiredKey = Annotated[IdempotencyKey, Header(alias="Idempotency-Key")]


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


def _actor(request: Request) -> ProfferDecisionActor:
    subject_uid = str(getattr(request.state, "subject_uid", "")).strip()
    username = str(getattr(request.state, "principal", "")).strip()
    if not subject_uid or not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    try:
        return ProfferDecisionActor(subject_uid=subject_uid, username=username)
    except ValueError:
        raise HTTPException(status_code=401, detail="authenticated subject identity is invalid") from None


@router.post("/entities/extract", response_model=ExtractionStarted, status_code=202)
async def extract_endpoint(
    body: EntityExtractRequest, request: Request, mode: OperatingMode, key: RequiredKey
):
    actor = _actor(request)
    try:
        return await service.extract(body, actor, key, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/entities/extractions/{workflow_id}", response_model=WorkflowProgress)
async def extraction_progress_endpoint(
    workflow_id: WorkflowId, preview_handle: Annotated[PreviewHandle, Query()], mode: OperatingMode
):
    try:
        return await service.workflow_progress("extraction", workflow_id, preview_handle, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/entities/proposals", response_model=ProposalsResponse)
async def proposals_endpoint(preview_handle: Annotated[PreviewHandle, Query()], mode: OperatingMode):
    try:
        return await service.proposals(preview_handle, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/entities/corrections", response_model=CorrectionApplied)
async def corrections_endpoint(
    body: EntityCorrectionRequest, request: Request, mode: OperatingMode, key: RequiredKey
):
    actor = _actor(request)
    try:
        return await service.correct(body, actor, key, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/entities/records/{record_id}", response_model=RecordView)
async def record_endpoint(
    record_id: Annotated[str, Path(min_length=1, max_length=64)],
    preview_handle: Annotated[PreviewHandle, Query()],
    mode: OperatingMode,
):
    try:
        return await service.record(preview_handle, record_id, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/entities/registry", response_model=RegistrySearchResponse)
async def registry_endpoint(
    q: Annotated[str, Query(max_length=200)] = "", limit: Annotated[int, Query(ge=1, le=100)] = 20
):
    try:
        return await service.search_registry(q, limit)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/entities/validate", response_model=ValidationReport)
async def validate_endpoint(body: EntityRunRequest, mode: OperatingMode):
    try:
        return await service.validate(body, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/entities/commit", response_model=CommitStarted, status_code=202)
async def commit_endpoint(
    body: EntityCommitRequest, request: Request, mode: OperatingMode, key: RequiredKey
):
    actor = _actor(request)
    try:
        return await service.commit(body, actor, key, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/entities/commits/{workflow_id}", response_model=WorkflowProgress)
async def commit_progress_endpoint(
    workflow_id: WorkflowId, preview_handle: Annotated[PreviewHandle, Query()], mode: OperatingMode
):
    try:
        return await service.workflow_progress("commit", workflow_id, preview_handle, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/events/from-record", response_model=MarkedEvent, status_code=201)
async def event_from_record_endpoint(
    body: EventFromRecordRequest, request: Request, mode: OperatingMode, key: RequiredKey
):
    actor = _actor(request)
    try:
        return await service.mark_event(body, actor, key, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None
