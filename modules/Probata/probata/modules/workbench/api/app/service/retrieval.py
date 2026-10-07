"""Forward Workbench search to the existing authenticated Platform retrieval service.

Inputs are validated browser choices; outputs retain the Platform's cited results
and per-source failures. Effects are read requests only. Use this for combined
content search instead of catalog-only discovery. Byline: Codex · 2026-10-06.
"""
from app.repo.spine_client import spine_json
from app.types.retrieval import SearchRequest


def search(payload: SearchRequest) -> dict:
    """Search approved source families while keeping provider and case authority server-owned.

    Input is SearchRequest; output is the full cited retrieval response including
    partial failures. Sends one read-only POST; creates no run or evidence item.
    """
    return spine_json("POST", "/v1/context/retrieve", json={
        **payload.model_dump(mode="json"), "scope": {},
        "per_leg_limit": min(100, max(50, payload.limit)), "leg_timeout_seconds": 25.0,
    })


def capabilities() -> dict:
    """Read supported modes and source availability from Platform without creating index work.

    Takes no input and returns the upstream capability response. Use before
    offering search controls; query-time failures remain independently visible.
    """
    return spine_json("GET", "/v1/context/retrieval-capabilities")
