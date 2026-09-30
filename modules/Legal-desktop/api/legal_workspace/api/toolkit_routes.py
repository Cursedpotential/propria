"""Family Law Toolkit records, read-only. Parent mounts `router`.

> _Byline: Claude Code · Opus 5.5 · 2026-09-27_
GET only. Opens toolkit-owned records (legal sources, cheat sheets, case documents)
through the shared legal-record contract, so ids and versions match the toolkit.
Never writes the toolkit store. Sibling: evidence_catalog_routes.py.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Query

from legal_workspace.services.family_court_toolkit import (
    ToolkitListing,
    ToolkitRecord,
    ToolkitStatus,
    ToolkitUnavailable,
    get_record,
    list_records,
    toolkit_status,
)

router = APIRouter(prefix="/v1/toolkit")


@router.get("/status")
def status() -> ToolkitStatus:
    return toolkit_status()


@router.get("/records")
def records(table: str = Query(default="source")) -> ToolkitListing:
    try:
        return list_records(table)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except ToolkitUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc


@router.get("/records/{ref}")
def record_detail(ref: str) -> ToolkitRecord:
    try:
        found = get_record(ref)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except ToolkitUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    if found is None:
        raise HTTPException(status_code=404, detail="record not in the toolkit store")
    return found
