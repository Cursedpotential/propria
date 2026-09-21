"""Routes for viewing SBV's decoded output before an ingest run exists.

Byline: Claude Code · Fable 5.1 · 2026-09-21.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, Header, HTTPException, Path, Query
from fastapi.responses import StreamingResponse

from app.service.proffer import ProfferError
from app.service.proffer_decoded import decoded_manifest, decoded_thread_page
from app.service.proffer_media import resolve_source_media, stream_resolved_media
from app.types.proffer_decoded import DecodedManifest, DecodedThreadPage

router = APIRouter(prefix="/decoded", tags=["proffer"])

SourceRef = Annotated[str, Query(min_length=8, max_length=2048)]
Sha256Hex = Annotated[str, Path(pattern=r"^[0-9a-f]{64}$")]


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


@router.get("/manifest", response_model=DecodedManifest)
def decoded_manifest_endpoint(source_ref: SourceRef):
    try:
        return decoded_manifest(source_ref)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/thread", response_model=DecodedThreadPage)
def decoded_thread_endpoint(
    source_ref: SourceRef,
    file: Annotated[str, Query(min_length=1, max_length=1024)],
    offset: Annotated[int, Query(ge=0)] = 0,
    limit: Annotated[int, Query(ge=1, le=500)] = 200,
):
    try:
        return decoded_thread_page(source_ref, file, offset=offset, limit=limit)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/media/{sha256}")
def decoded_media_endpoint(
    sha256: Sha256Hex,
    source_ref: SourceRef,
    range_header: Annotated[str | None, Header(alias="Range")] = None,
):
    try:
        result = stream_resolved_media(resolve_source_media(source_ref, sha256), range_header)
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
        result.iterator, status_code=result.status_code, media_type=result.content_type, headers=headers
    )
