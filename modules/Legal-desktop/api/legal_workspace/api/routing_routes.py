"""HTTP for the owner-editable page catalog.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Query

from legal_workspace.services.routing import (
    RoutingTable,
    path_for_query,
    public_routing,
    save_routing,
)

router = APIRouter()


@router.get("/v1/routing")
def get_routing() -> dict:
    return public_routing()


@router.get("/v1/routing/resolve")
def resolve_page(q: str = Query(..., min_length=1)) -> dict:
    path = path_for_query(q)
    if path is None:
        raise HTTPException(status_code=404, detail="unknown page")
    return {"query": q.strip(), "path": path}


@router.put("/v1/routing")
def put_routing(body: RoutingTable) -> dict:
    if body.version < 1:
        raise HTTPException(status_code=400, detail="routing version must be >= 1")
    path = save_routing(body)
    return {"ok": True, "path": str(path), "table": public_routing()}
