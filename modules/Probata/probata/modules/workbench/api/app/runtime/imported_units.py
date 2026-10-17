"""Expose read-only unit inventory, within-unit search and actual normalized records.

Input scope is an existing Imported ID, not a caller-selected case or table.
No parsing, persistence, embedding, promotion or schema mutation occurs here.
Byline: ChatGPT, 2026-10-17.
"""
from typing import Annotated
from uuid import UUID
from fastapi import APIRouter, HTTPException, Path, Query
from app.service import imported, imported_units as service

router = APIRouter(tags=["imported-units"])
SourceID = Annotated[str, Path(min_length=1, max_length=2048, pattern=r"^[A-Za-z0-9_-]+$")]
ScopedID = Annotated[str | None, Query(max_length=2048, pattern=r"^[A-Za-z0-9_-]+$")]


@router.get("/sources/{source_id}/unit")
async def unit(source_id: SourceID, thread_id: ScopedID = None):
    """Read complete registered unit membership and selected PG generation counts.

    Inputs: import ID and optional thread. Output: scoped inventory with limitations.
    Read-only; opening a unit never designates it an atomic preservation unit.
    """
    try:
        return service.describe(await service.resolve(source_id, thread_id))
    except imported.ImportedError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/sources/{source_id}/unit/messages")
async def messages(source_id: SourceID, thread_id: ScopedID = None, member_id: ScopedID = None,
                   q: Annotated[str, Query(max_length=200)] = "",
                   cursor: Annotated[str | None, Query(max_length=1024)] = None,
                   limit: Annotated[int, Query(ge=1, le=100)] = 50):
    """Search before paging across all members in the resolved normalized unit.

    Inputs: source/member/thread, literal text and keyset cursor. Output: stored PG
    messages and provenance. SELECT only, distinct from global index search.
    """
    try:
        return await service.messages(source_id, thread_id=thread_id, member_id=member_id,
                                      q=q.strip(), cursor=cursor, limit=limit)
    except imported.ImportedError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/sources/{source_id}/unit/records/{record_id}")
async def record(source_id: SourceID, record_id: UUID):
    """Inspect an exact current or historical normalized record in the same unit.

    Inputs: source ID and UUID. Output: stored fields and version coordinates.
    SELECT only; rejects records not associated with this unit's registered sources.
    """
    try:
        return await service.get_record(source_id, str(record_id))
    except imported.ImportedError as error:
        raise HTTPException(error.status, error.message) from None
