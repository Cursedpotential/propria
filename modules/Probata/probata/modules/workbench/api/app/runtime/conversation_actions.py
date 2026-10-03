"""Workbench BFF routes for Extract and Send to Surreal on the Imported view's conversations.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

    GET  /api/extractors                                        every selectable extractor (the picker's list)
    POST /api/imported/threads/extract                          start extraction for chosen conversations and extractors
    POST /api/imported/threads/send-to-surreal                  start sending chosen conversations to surreal-case
    GET  /api/imported/workflows/{workflow_id}                  progress of an extraction or a send
    GET  /api/imported/threads/{thread_id}/extractions          entities and events found, grouped by extractor (read-only)

The two starts need an Idempotency-Key header (a retried click joins the running workflow) and an authenticated subject.
Always the live case; there is no matter or mode parameter. The reads use the ``workbench_reader`` login.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, Header, HTTPException, Path, Request

from app.service import conversation_actions as service
from app.service.proffer_errors import ProfferError
from app.types.conversation_actions import (
    ExtractConversationsRequest,
    IdempotencyKey,
    SendConversationsRequest,
    WorkflowStarted,
    WorkflowStatus,
)
from app.types.proffer import ProfferDecisionActor

router = APIRouter(prefix="/api", tags=["conversation-actions"])

Opaque = Annotated[str, Path(min_length=1, max_length=2048, pattern=r"^[A-Za-z0-9_-]+$")]
WorkflowPath = Annotated[str, Path(min_length=8, max_length=512, pattern=r"^[A-Za-z0-9:_.-]+$")]
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


@router.get("/extractors")
async def extractors_endpoint():
    """List every selectable extractor and name the default."""
    try:
        return await service.extractors()
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/imported/threads/extract", response_model=WorkflowStarted, status_code=202)
async def extract_endpoint(body: ExtractConversationsRequest, request: Request, key: RequiredKey):
    """Start entity and event extraction over the chosen conversations with the chosen extractors."""
    actor = _actor(request)
    try:
        return await service.start_extraction(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/imported/threads/send-to-surreal", response_model=WorkflowStarted, status_code=202)
async def send_endpoint(body: SendConversationsRequest, request: Request, key: RequiredKey):
    """Start sending whole conversations (and the extractions they have) to surreal-case."""
    actor = _actor(request)
    try:
        return await service.start_send(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/imported/workflows/{workflow_id}", response_model=WorkflowStatus)
async def workflow_endpoint(workflow_id: WorkflowPath):
    """Progress of an extraction or a send workflow."""
    try:
        return await service.workflow_status(workflow_id)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/imported/threads/{thread_id}/extractions")
async def extractions_endpoint(thread_id: Opaque):
    """Entities and events the extractors found in one conversation, grouped by extractor. Read-only."""
    try:
        return await service.extractions(thread_id)
    except ProfferError as error:
        raise _translate(error) from None
