"""Read-only Intake discovery endpoints. Byline: Codex · 2026-09-20."""
from typing import Literal

from fastapi import APIRouter, HTTPException, Query
from starlette.concurrency import run_in_threadpool

from app.repo.intake_discovery import DiscoveryError, atomic_members, atomic_units, catalog_page
from app.service.intake_discovery import capabilities, neighbors, search_index

router = APIRouter(prefix="/api/intake/discovery", tags=["intake-discovery"])


@router.get("/units")
async def units_endpoint(unit_type: str | None = Query(None, max_length=100), after_id: int = Query(-1, ge=-1), limit: int = Query(100, ge=1, le=200)):
    try:
        return await run_in_threadpool(atomic_units, unit_type=unit_type, after_id=after_id, limit=limit)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/units/{unit_id}/members")
async def members_endpoint(unit_id: int, limit: int = Query(100, ge=1, le=200), cursor: str | None = Query(None, max_length=8192)):
    try:
        if unit_id < 0:
            raise DiscoveryError("Invalid atomic unit", 422)
        return await run_in_threadpool(atomic_members, unit_id, limit=limit, cursor=cursor)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/capabilities")
async def capabilities_endpoint():
    try:
        return await run_in_threadpool(capabilities)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/tree")
async def tree_endpoint(parent: str = Query("", max_length=4096), limit: int = Query(100, ge=1, le=200), cursor: str | None = Query(None, max_length=8192)):
    try:
        return await run_in_threadpool(catalog_page, parent=parent, limit=limit, cursor=cursor)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/search")
async def search_endpoint(q: str = Query(..., min_length=1, max_length=4096), parent: str = Query("", max_length=4096), mode: Literal["filename_prefix", "filename_substring", "contents", "hybrid"] = "filename_substring", limit: int = Query(100, ge=1, le=100), cursor: str | None = Query(None, max_length=8192), atomic_unit: str | None = None, file_type: str | None = None):
    try:
        if not q.strip():
            raise DiscoveryError("Enter a search term", 422)
        if atomic_unit is not None or file_type is not None:
            raise DiscoveryError("Atomic-unit and file-type filters are not supported by the connected index", 422)
        if mode in {"filename_prefix", "filename_substring"}:
            return await run_in_threadpool(catalog_page, parent=parent, query=q, limit=limit, cursor=cursor, mode=mode)
        if parent or cursor:
            raise DiscoveryError("Contents and hybrid index searches cannot apply folder filters or cursors", 422)
        return await search_index(q, mode, limit)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/neighbors/{table}/{key}")
async def neighbors_endpoint(table: str, key: str, limit: int = Query(50, ge=1, le=100)):
    try:
        return await neighbors(table, key, limit)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None
