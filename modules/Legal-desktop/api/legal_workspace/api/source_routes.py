"""Research-source HTTP. Parent mounts `source_router` on the app.

> _Byline: Grok · grok-4.6 · 2026-08-18_
List redacts auth secrets. Search never returns citator verification.
Does not write the workspace or the evidence package.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Query

from legal_workspace.domain.sources import SearchResult, redact_source_config
from legal_workspace.services.sources import load_source_configs, search_source

source_router = APIRouter(prefix="")


@source_router.get("/v1/sources")
def list_sources() -> list[dict[str, Any]]:
    return [redact_source_config(item) for item in load_source_configs()]


@source_router.get("/v1/sources/{source_id}/search")
def search_sources(
    source_id: str,
    q: str = Query(default=""),
) -> SearchResult:
    return search_source(source_id, q)
