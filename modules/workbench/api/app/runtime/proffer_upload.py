"""Upload and staged-acquisition routes for Proffer sources.

Split out of app/runtime/proffer.py on 2026-09-22 (that module was over the
300-line cap). Byline: Claude Code · Fable 5.1 · 2026-09-22.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, HTTPException, Path, Query, Request

from app.service.proffer import ProfferError, complete_upload_response, open_upload_stream
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferUploadResponse

router = APIRouter(tags=["proffer"])


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


@router.post("/upload", response_model=ProfferUploadResponse, status_code=201)
async def upload_endpoint(request: Request, mode: Annotated[MatterMode, Query()]):
    try:
        client, response = await open_upload_stream(
            request.stream(),
            mode=mode,
            content_type=request.headers.get("content-type"),
            content_length=request.headers.get("content-length"),
        )
        return await complete_upload_response(client, response, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/staged/{staged_id}/acquisition", response_model=ProfferUploadResponse, status_code=201)
async def staged_acquisition_endpoint(
    staged_id: Annotated[str, Path(pattern=r"^[a-f0-9]{64}$")],
    mode: Annotated[MatterMode, Query()],
):
    from app.service.proffer_staged import acquire_staged

    try:
        return await acquire_staged(staged_id, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None
