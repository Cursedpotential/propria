"""Expose combined content search through Workbench's existing authenticated boundary.

Inputs are browser query choices; outputs preserve cited hits and source errors.
Only upstream reads occur. This route does not grant promotion or evidence access.
Byline: Codex · 2026-10-06.
"""
from fastapi import APIRouter, HTTPException
from starlette.concurrency import run_in_threadpool

from app.repo.spine_client import SpineError
from app.repo.intake_discovery import DiscoveryError
from app.service import retrieval as service
from app.types.retrieval import GraphResolveRequest, SearchRequest

router = APIRouter(prefix="/api/retrieval", tags=["retrieval"])


@router.post("/relationships")
async def relationships_endpoint(body: GraphResolveRequest):
    """Resolve exact source identity to completed graph snapshots without choosing a version.

    Input is a source/document pair and optional version; output is upstream
    graph references and ambiguity. Read-only; use before graph neighborhood reads.
    """
    try:
        return await service.resolve_graph(body)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.post("/search")
async def search_endpoint(body: SearchRequest):
    """Return combined search hits with original citations and explicit backend failures.

    Input is a bounded SearchRequest; output is Platform retrieval JSON. Performs
    read-only upstream work; use instead of separate filename/message searches.
    """
    try:
        return await run_in_threadpool(service.search, body)
    except SpineError as error:
        raise HTTPException(error.status_code, error.detail) from None


@router.get("/capabilities")
async def capabilities_endpoint():
    """Return Platform's search capabilities for the authenticated Workbench user.

    No inputs; output is upstream capability JSON. One read occurs, without
    probes that write data or create indexes. Use when constructing search modes.
    """
    try:
        return await run_in_threadpool(service.capabilities)
    except SpineError as error:
        raise HTTPException(error.status_code, error.detail) from None
