"""Workbench BFF routes for folder batch imports.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, HTTPException, Path

from app.runtime.operating_mode import OperatingMode
from app.service.proffer_batch import batch_status, start_batch
from app.service.proffer_errors import ProfferError
from app.types.proffer_batch import (
    ProfferBatchStartRequest,
    ProfferBatchStartResponse,
    ProfferBatchStatus,
)

router = APIRouter(prefix="/api/proffer", tags=["proffer"])

BatchID = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{32,128}$")]


@router.post("/start-batch", response_model=ProfferBatchStartResponse, status_code=201)
async def start_batch_endpoint(
    body: ProfferBatchStartRequest, mode: OperatingMode
):
    try:
        return await start_batch(body, mode=mode)
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None


@router.get("/batches/{batch_id}", response_model=ProfferBatchStatus)
async def batch_status_endpoint(batch_id: BatchID, mode: OperatingMode):
    try:
        return await batch_status(batch_id, mode=mode)
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
