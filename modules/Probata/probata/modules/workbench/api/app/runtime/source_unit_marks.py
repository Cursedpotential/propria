"""Routes for hand-marked source units and the Takeout parts proposal.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Request
from starlette.concurrency import run_in_threadpool

from app.repo.source_unit_marks import UnitMarkStoreError
from app.service.source_unit_marks import list_marks, propose_unit, record_mark
from app.types.source_unit_marks import (
    SourceUnitMark,
    SourceUnitMarkList,
    SourceUnitMarkRequest,
    SourceUnitProposal,
    SourceUnitProposalRequest,
)

router = APIRouter(prefix="/api/sources", tags=["sources"])


def _principal(request: Request) -> str:
    username = str(getattr(request.state, "principal", "")).strip()
    if not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    return username


@router.get("/unit-marks", response_model=SourceUnitMarkList)
async def list_unit_marks_endpoint():
    try:
        return await run_in_threadpool(list_marks)
    except UnitMarkStoreError as error:
        raise HTTPException(error.status, error.message) from None


@router.post("/unit-marks", response_model=SourceUnitMark, status_code=201)
async def record_unit_mark_endpoint(body: SourceUnitMarkRequest, request: Request):
    try:
        return await run_in_threadpool(record_mark, body, _principal(request))
    except UnitMarkStoreError as error:
        raise HTTPException(error.status, error.message) from None


@router.post("/unit-proposal", response_model=SourceUnitProposal)
async def unit_proposal_endpoint(body: SourceUnitProposalRequest):
    return await run_in_threadpool(propose_unit, body)
