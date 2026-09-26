"""Consignatio catalog HTTP surface (read-only). Parent mounts `router`.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
GET only. Discovers objects through the catalog, reports lifecycle from the
catalog's promotion record, and hands back the configured B2 retrieval link.
Never writes the catalog, B2, or the workspace. Desk annotations live in the
workspace store keyed by the catalog's object id, never in evidence objects.
"""

from __future__ import annotations

from typing import Any, Literal

from fastapi import APIRouter, HTTPException, Query

from legal_workspace.services.consignatio_catalog import (
    CatalogObject,
    CatalogPage,
    CatalogStatus,
    CatalogUnavailable,
    catalog_status,
    get_object,
    list_objects,
    list_promotions,
    signed_artifact_link,
)

router = APIRouter(prefix="/v1/evidence-catalog")


@router.get("/status")
def status() -> CatalogStatus:
    return catalog_status()


@router.get("/objects")
def objects(
    prefix: str = Query(default=""),
    q: str = Query(default=""),
    after: str = Query(default=""),
    limit: int = Query(default=50, ge=1, le=200),
) -> CatalogPage:
    try:
        return list_objects(prefix=prefix, q=q, after=after, limit=limit)
    except CatalogUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc


@router.get("/objects/{object_id}")
def object_detail(object_id: str) -> CatalogObject:
    try:
        found = get_object(object_id)
    except CatalogUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    if found is None:
        raise HTTPException(status_code=404, detail="object id not in the catalog")
    return found


@router.get("/objects/{object_id}/link")
def object_link(object_id: str) -> dict[str, Any]:
    try:
        link = signed_artifact_link(object_id)
    except CatalogUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    if link is None:
        raise HTTPException(status_code=404, detail="object id not in the catalog")
    return link


@router.get("/promotions")
def promotions(state: Literal["pending", "promoted"] = "promoted") -> list[dict[str, Any]]:
    try:
        return list_promotions(state)
    except CatalogUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
