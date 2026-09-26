"""Operations HTTP: Activity log and Inbound notices. Same events file. English labels.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter

from legal_workspace.services.persist import default_store_dir, read_jsonl
from legal_workspace.services.workspace import WORKSPACE

router = APIRouter()


def _events(limit: int = 200) -> list[dict[str, Any]]:
    rows = read_jsonl(WORKSPACE.events_path, limit=limit)
    extra = default_store_dir() / "automation_events.jsonl"
    if extra != WORKSPACE.events_path:
        rows.extend(read_jsonl(extra, limit=limit))
    rows.sort(key=lambda item: str(item.get("at") or ""), reverse=True)
    return rows[:limit]


@router.get("/v1/audit")
def list_audit(limit: int = 200) -> list[dict[str, Any]]:
    return _events(limit)


@router.get("/v1/triggers")
def list_triggers(limit: int = 100) -> list[dict[str, Any]]:
    return _events(limit)
