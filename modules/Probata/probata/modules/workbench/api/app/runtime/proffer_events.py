"""Server-sent event stream for one Proffer preview.

Split out of app/runtime/proffer.py on 2026-09-22 (that module was over the
300-line cap). Byline: Claude Code · Fable 5.1 · 2026-09-22.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, Header, HTTPException, Path
from fastapi.responses import StreamingResponse

from app.runtime.operating_mode import OperatingMode
from app.service.proffer import ProfferError, open_preview_event_stream, validated_preview_events

router = APIRouter(tags=["proffer"])

PreviewHandle = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{32,128}$")]


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


@router.get("/previews/{preview_handle}/events")
async def preview_events_endpoint(
    preview_handle: PreviewHandle,
    mode: OperatingMode,
    last_event_id_header: Annotated[str | None, Header(alias="Last-Event-ID")] = None,
):
    last_event_id: int | None = None
    if last_event_id_header is not None:
        try:
            last_event_id = int(last_event_id_header)
        except ValueError:
            raise HTTPException(status_code=422, detail="Last-Event-ID must be a non-negative integer") from None
        if last_event_id < 0:
            raise HTTPException(status_code=422, detail="Last-Event-ID must be a non-negative integer")
    try:
        client, response = await open_preview_event_stream(preview_handle, mode=mode, last_event_id=last_event_id)
    except ProfferError as error:
        raise _translate(error) from None

    async def body():
        try:
            async for event in validated_preview_events(
                response,
                preview_handle=preview_handle,
                mode=mode,
                last_event_id=last_event_id,
            ):
                yield event
        finally:
            await response.aclose()
            await client.aclose()

    return StreamingResponse(
        body(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
