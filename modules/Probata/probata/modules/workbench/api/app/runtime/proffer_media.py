"""Media byte-stream route for one decoded Proffer attachment.

Byline: Claude Code · Sonnet 5 · 2026-09-21.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, Header, HTTPException, Path
from fastapi.responses import StreamingResponse

from app.runtime.operating_mode import OperatingMode
from app.service.proffer import ProfferError
from app.service.proffer_media import stream_preview_media

router = APIRouter(tags=["proffer"])

PreviewHandle = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{32,128}$")]
Sha256Hex = Annotated[str, Path(pattern=r"^[0-9a-f]{64}$")]


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


@router.get("/previews/{preview_handle}/media/{sha256}")
async def preview_media_endpoint(
    preview_handle: PreviewHandle,
    sha256: Sha256Hex,
    mode: OperatingMode,
    range_header: Annotated[str | None, Header(alias="Range")] = None,
):
    try:
        result = await stream_preview_media(preview_handle, sha256, mode=mode, range_header=range_header)
    except ProfferError as error:
        raise _translate(error) from None

    headers = {
        "Accept-Ranges": "bytes",
        "Cache-Control": "private, max-age=31536000, immutable",
        "X-Content-Type-Options": "nosniff",
        "Content-Disposition": f'{result.disposition}; filename="{sha256}{result.extension}"',
        "Content-Length": str(result.content_length),
    }
    if result.byte_range is not None:
        start, end = result.byte_range
        headers["Content-Range"] = f"bytes {start}-{end}/{result.total_size}"

    return StreamingResponse(
        result.iterator,
        status_code=result.status_code,
        media_type=result.content_type,
        headers=headers,
    )
