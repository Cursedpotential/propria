# Byline: Claude Code · Sonnet (agent) · 2026-07-23 (C4: Knowledge browser + Graphiti pane)
# Byline: Codex · GPT-5 · 2026-08-15 (case/group-scoped browser contract)
# Byline: Codex · GPT-5 · 2026-08-16 (canonical knowledge item detail)
# Byline: Codex · GPT-5 · 2026-08-18 (native evidence horizon proxy)
# Byline: Claude Code · Opus 5.5 · 2026-09-27 (DF-24: Graphiti routes removed; Graphiti is retired, D-070)
"""Knowledge projection search and canonical source inspection.

Thin FastAPI wrappers over app/service/knowledge.py; every SpineError
becomes an HTTPException with the upstream's
own error message preserved verbatim — mirrors app/runtime/inspect.py's
error-translation pattern.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated, Literal

from fastapi import APIRouter, HTTPException, Query

from app.repo.spine_client import SpineError
from app.service import knowledge as knowledge_service

router = APIRouter(prefix="/api", tags=["knowledge"])


# ---------------------------------------------------------------------------
# Knowledge search (Weaviate projection) and browse (canonical PostgreSQL)
# ---------------------------------------------------------------------------


@router.get("/knowledge/search")
async def search_knowledge_endpoint(
    q: Annotated[str, Query(min_length=1, max_length=2000)],
    case_id: Annotated[str, Query(min_length=1, max_length=200)] = "primary",
    lane: Literal["platform", "legal", "personal_history", "context", "evidence"] | None = None,
    limit: Annotated[int | None, Query(ge=1, le=100)] = None,
    horizon: datetime | None = None,
):
    try:
        return knowledge_service.search(
            q,
            case_id=case_id,
            lane=lane,
            limit=limit,
            horizon=horizon.isoformat() if horizon is not None else None,
        )
    except ValueError as e:
        raise HTTPException(status_code=422, detail=str(e)) from None
    except SpineError as e:
        raise HTTPException(status_code=e.status_code, detail=e.detail) from None


@router.get("/knowledge/contents")
async def list_knowledge_contents_endpoint(
    case_id: Annotated[str, Query(min_length=1, max_length=200)],
    lane: Literal["platform", "legal", "personal_history", "context", "evidence"],
    limit: Annotated[int | None, Query(ge=1, le=100)] = None,
    offset: Annotated[int | None, Query(ge=0)] = None,
):
    try:
        return knowledge_service.list_contents(case_id=case_id, lane=lane, limit=limit, offset=offset)
    except ValueError as e:
        raise HTTPException(status_code=422, detail=str(e)) from None
    except SpineError as e:
        raise HTTPException(status_code=e.status_code, detail=e.detail) from None


@router.get("/knowledge/contents/{artifact_id}")
async def get_knowledge_content_endpoint(
    artifact_id: str,
    case_id: Annotated[str, Query(min_length=1, max_length=200)],
):
    try:
        return knowledge_service.get_content(artifact_id, case_id=case_id)
    except ValueError as e:
        raise HTTPException(status_code=422, detail=str(e)) from None
    except SpineError as e:
        raise HTTPException(status_code=e.status_code, detail=e.detail) from None
