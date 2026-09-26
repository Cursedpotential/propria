"""Read-only Intake discovery endpoints. Byline: Codex · 2026-09-20."""
from typing import Literal

from fastapi import APIRouter, HTTPException, Query
from pydantic import BaseModel, ConfigDict, Field
from starlette.concurrency import run_in_threadpool

from app.repo.intake_discovery import (
    DiscoveryError,
    atomic_members,
    atomic_units,
    catalog_by_vault_key,
    catalog_page,
    units_for_member_keys,
    units_for_roots,
    units_under_prefix,
)
from app.service.intake_discovery import capabilities, neighbors, search_index

router = APIRouter(prefix="/api/intake/discovery", tags=["intake-discovery"])


class UnitLookupRequest(BaseModel):
    """One page of folder prefixes and object keys, asked about together."""

    model_config = ConfigDict(extra="forbid")
    roots: list[str] = Field(default_factory=list, max_length=200)
    keys: list[str] = Field(default_factory=list, max_length=400)


@router.post("/unit-lookup")
async def unit_lookup_endpoint(body: UnitLookupRequest):
    """Which of these folders are catalog units, and which keys are members."""
    try:
        roots = await run_in_threadpool(units_for_roots, body.roots)
        members = await run_in_threadpool(units_for_member_keys, body.keys)
        return {"units": roots["items"], "members": members["items"],
                "backend": "casebible_atomic_units", "source_links_verified": False}
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/units/under-prefix")
async def units_under_prefix_endpoint(prefix: str = Query(..., min_length=1, max_length=4096)):
    """Which recorded units the files under ONE vault folder belong to."""
    try:
        return await run_in_threadpool(units_under_prefix, prefix)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/catalog/by-vault-key")
async def catalog_by_vault_key_endpoint(vault_key: str = Query(..., min_length=1, max_length=4096)):
    """Catalog provenance for one vault object: original source, scope, path, occurrences."""
    try:
        return await run_in_threadpool(catalog_by_vault_key, vault_key)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


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
