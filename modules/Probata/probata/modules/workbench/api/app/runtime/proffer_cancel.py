"""POST /api/proffer/previews/{handle}/cancel: the operator cancels one run.

Byline: Claude Code · Opus 5.5 · 2026-09-28 (D05-C06).
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, HTTPException, Path, Request

from app.runtime.operating_mode import OperatingMode
from app.service.proffer_cancel import cancel
from app.service.proffer_errors import ProfferError
from app.types.proffer import ProfferDecisionActor
from app.types.proffer_cancel import ProfferCancelRequest, ProfferCancelResponse

router = APIRouter(tags=["proffer"])

PreviewHandle = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{32,128}$")]


def _actor(request: Request) -> ProfferDecisionActor:
    subject_uid = str(getattr(request.state, "subject_uid", "")).strip()
    username = str(getattr(request.state, "principal", "")).strip()
    if not subject_uid or not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    try:
        return ProfferDecisionActor(subject_uid=subject_uid, username=username)
    except ValueError:
        raise HTTPException(status_code=401, detail="authenticated subject identity is invalid") from None


@router.post("/previews/{preview_handle}/cancel", response_model=ProfferCancelResponse, status_code=202)
async def cancel_endpoint(
    preview_handle: PreviewHandle,
    body: ProfferCancelRequest,
    request: Request,
    mode: OperatingMode,
):
    actor = _actor(request)
    try:
        return await cancel(preview_handle, body, actor, mode=mode)
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
