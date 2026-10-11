"""Tailnet Workbench read-only raw-message preview routes."""

import asyncio
from typing import Annotated
from fastapi import APIRouter, HTTPException, Path, Request
from fastapi.responses import JSONResponse
from app.service import source_message_preview as service
from app.service.proffer_errors import ProfferError
from app.service.source_inspection import SourceInspectionError
from app.types.source_message_preview import SourceMessagePreviewRequest, SourceMessagePreviewResponse

router = APIRouter(prefix="/source-preview", tags=["proffer"])


async def _connected(request: Request, operation):
    """Cancel a bounded upstream call when the browser disconnects.

    Inputs: HTTP request and preview coroutine. Output: typed JSON page/status.
    Effects: task cancellation only; pick for either preview route, no writes.
    """
    task = asyncio.create_task(operation)
    try:
        while not task.done():
            if await request.is_disconnected():
                raise HTTPException(499, "Preview request cancelled")
            await asyncio.wait({task}, timeout=0.25)
        result, status = await task
        return JSONResponse(result.model_dump(mode="json"), status_code=status,
                            headers={"Cache-Control": "no-store"})
    except (ProfferError, SourceInspectionError) as error:
        raise HTTPException(error.status_code, error.detail) from None
    finally:
        if not task.done():
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)


@router.post("", response_model=SourceMessagePreviewResponse)
async def start_message_preview(body: SourceMessagePreviewRequest, request: Request):
    """Open one bounded SMS message preview without starting processing.

    Inputs: allowlisted source listing pins and cursor. Output: ready page or
    pending handle. Effects: HEAD and Go read-preview operation only; pick when
    selecting an original SMS file before extraction, indexing or promotion.
    """
    return await _connected(request, service.start_page(body))


@router.get("/{page_id}", response_model=SourceMessagePreviewResponse)
async def poll_message_preview(request: Request, page_id: Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{1,128}$")]):
    """Read a pending bounded message preview by its opaque handle.

    Input: page ID, never a URL. Output: ready/pending page or expiry. Effects:
    one fixed-route Go read, cancelled on disconnect; pick after a 202 response.
    """
    return await _connected(request, service.poll_page(page_id))
