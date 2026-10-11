"""Expose optional discovery without altering source-processing workflows.

Inputs: case-scoped dataset choices and owner review requests. Outputs: real
engine runs, source context and saved receipts. Effects: fixed authenticated
engine requests; choose for discovery, never source mutation or publication.
"""

import asyncio
from typing import Annotated
from uuid import UUID

from fastapi import APIRouter, Header, HTTPException, Path, Query, Request
from fastapi.responses import JSONResponse

from app.runtime.operating_mode import OperatingMode
from app.service import discovery as service
from app.service.proffer_errors import ProfferError
from app.types.discovery import FeedbackRequest, Key, RunRequest, Scope
from app.types.proffer import ProfferDecisionActor

router = APIRouter(prefix="/api/discovery", tags=["discovery"])
RequiredKey = Annotated[Key, Header(alias="Idempotency-Key")]
CandidatePath = Annotated[str, Path(min_length=1, max_length=200, pattern=r"^[A-Za-z0-9_.:-]+$")]


def _actor(request: Request) -> ProfferDecisionActor:
    """Read identity solely from validated session state; output actor, no side effects."""
    try:
        return ProfferDecisionActor(
            subject_uid=getattr(request.state, "subject_uid", ""), username=getattr(request.state, "principal", "")
        )
    except ValueError:
        raise HTTPException(401, "Authenticated subject identity is unavailable") from None


async def _connected(request: Request, operation, status_code: int = 200):
    """Cancel obsolete browser requests and return uncached engine results.

    Inputs: HTTP request and service coroutine. Output: correlated JSON. Effects:
    bounded task cancellation only; choose for all discovery routes.
    """
    task = asyncio.create_task(operation)
    try:
        while not task.done():
            if await request.is_disconnected():
                raise HTTPException(499, "Discovery request cancelled")
            await asyncio.wait({task}, timeout=0.25)
        return JSONResponse(await task, status_code=status_code, headers={"Cache-Control": "no-store"})
    except ProfferError as error:
        raise HTTPException(error.status_code, error.detail) from None
    finally:
        if not task.done():
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)


@router.get("/datasets")
async def datasets(request: Request, mode: OperatingMode, matter_id: UUID, court_case_id: UUID):
    """List case-bound server datasets; input scope, output choices, effects fixed GET only."""
    return await _connected(
        request, service.datasets(Scope(mode=mode, matter_id=matter_id, court_case_id=court_case_id), _actor(request))
    )


@router.post("/runs")
async def start(body: RunRequest, request: Request, mode: OperatingMode, key: RequiredKey):
    """Start a run from dataset pins; input config, output durable run, effects engine POST."""
    if key != str(body.run_id):
        raise HTTPException(409, "Idempotency-Key must equal run_id")
    if body.scope.mode != mode:
        raise HTTPException(409, "Discovery body and query operating modes differ")
    return await _connected(request, service.start(body, _actor(request), key), status_code=202)


@router.get("/runs/{run_id}")
async def run(run_id: UUID, request: Request, mode: OperatingMode, matter_id: UUID, court_case_id: UUID):
    """Resume one scoped run; input ID/case, output real status, effects engine GET only."""
    return await _connected(
        request,
        service.run(run_id, Scope(mode=mode, matter_id=matter_id, court_case_id=court_case_id), _actor(request)),
    )


@router.get("/runs/{run_id}/candidates")
async def candidates(
    run_id: UUID,
    request: Request,
    mode: OperatingMode,
    matter_id: UUID,
    court_case_id: UUID,
    cursor: Annotated[str, Query(max_length=2048)] = "",
):
    """List a bounded grouped review page; input run/cursor, output queue, effects GETs only."""
    return await _connected(
        request,
        service.candidates(
            run_id, Scope(mode=mode, matter_id=matter_id, court_case_id=court_case_id), _actor(request), cursor
        ),
    )


@router.get("/runs/{run_id}/context/{candidate_id}")
async def context(
    run_id: UUID,
    candidate_id: CandidatePath,
    request: Request,
    mode: OperatingMode,
    matter_id: UUID,
    court_case_id: UUID,
):
    """Read one candidate's source context; input IDs/scope, output locators, effects GETs only."""
    return await _connected(
        request,
        service.context(
            run_id, candidate_id, Scope(mode=mode, matter_id=matter_id, court_case_id=court_case_id), _actor(request)
        ),
    )


@router.post("/runs/{run_id}/feedback")
async def feedback(
    run_id: UUID,
    body: FeedbackRequest,
    request: Request,
    mode: OperatingMode,
    matter_id: UUID,
    court_case_id: UUID,
    key: RequiredKey,
):
    """Save explicit revision-checked feedback; input choice, output receipt, effects engine POST."""
    if key != str(body.feedback_id):
        raise HTTPException(409, "Idempotency-Key must equal feedback_id")
    return await _connected(
        request,
        service.feedback(
            run_id, body, Scope(mode=mode, matter_id=matter_id, court_case_id=court_case_id), _actor(request), key
        ),
    )
