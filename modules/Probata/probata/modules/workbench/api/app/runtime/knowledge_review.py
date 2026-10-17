"""Expose the canonical Platform knowledge-review queue in the EXISTING Workbench Activity UI.

Inputs: authenticated Authentik Workbench identity, Platform API bearer, reviewed version pins.
Outputs: compact inbox, exact review detail, source-independent owner decision receipt.
Effects: GET reads or one guarded owner decision forwarded to canonical Platform PG API.
Does NOT update a memory, delete source material, approve case evidence or invent a review queue.
"""
from __future__ import annotations

import hmac
import json
import os
import re
from pathlib import Path
from uuid import UUID

import httpx
from fastapi import APIRouter, HTTPException, Request
from pydantic import BaseModel, ConfigDict, Field

from app.config import settings
from app.repo.platform_api_auth import PlatformAPIAuthError, platform_api_bearer_headers

router = APIRouter(prefix="/api/knowledge-review", tags=["knowledge-review"])
BASE_PATH = "/v1/knowledge-reconcile/reviews"


class DecisionRequest(BaseModel):
    """User-supplied intention; reviewer identity is ONLY taken from Authentik request state."""
    model_config = ConfigDict(extra="forbid")
    expected_case_version: int = Field(ge=1)
    request_id: str = Field(min_length=16, max_length=256)
    decision: str = Field(pattern="^(approve_plan|reject|defer)$")
    rationale: str = Field(min_length=2, max_length=2000)


def _reviewer(request: Request) -> str:
    """Authenticated Workbench subject is the only source of owner identity."""
    uid = getattr(request.state, "subject_uid", None)
    if not isinstance(uid, str) or not uid or len(uid) > 256:
        raise HTTPException(401, "authenticated owner required")
    return uid


def _platform_url(path: str) -> str:
    """Use only existing fixed Platform service URL, not a user-supplied destination."""
    base = settings.platform_api_url.rstrip("/")
    if not base.startswith(("http://", "https://")) or not path.startswith("/v1/knowledge-reconcile/"):
        raise HTTPException(503, "configured Platform review API is unavailable")
    return base + path


async def _request(method: str, path: str, *, body: dict | None = None, owner: bool = False) -> dict:
    """Call one bounded canonical Platform API operation, preserving upstream HTTP gate failures."""
    try:
        headers = {**platform_api_bearer_headers(), "accept": "application/json"}
    except PlatformAPIAuthError:
        raise HTTPException(503, "Platform service credentials unavailable") from None
    if owner:
        configured = os.getenv("KNOWLEDGE_REVIEW_DECISION_TOKEN_FILE", "")
        if not configured:
            raise HTTPException(503, "knowledge review decision capability is not configured")
        try:
            token = Path(configured).read_text(encoding="utf-8").strip()
        except OSError:
            raise HTTPException(503, "knowledge review decision capability unavailable") from None
        if len(token) < 32 or len(token) > 4096:
            raise HTTPException(503, "invalid decision capability")
        headers["x-knowledge-review-owner-token"] = token
    try:
        async with httpx.AsyncClient(timeout=18, follow_redirects=False, trust_env=False) as client:
            response = await client.request(method, _platform_url(path), headers=headers, json=body)
        if response.status_code >= 400:
            code = response.status_code
            if code in (400, 401, 403, 404, 409, 422, 503):
                raise HTTPException(code, "canonical review request was rejected or unavailable")
            raise HTTPException(502, "Platform review request failed")
        if len(response.content) > 1024 * 1024:
            raise HTTPException(502, "review result exceeded display limit")
        result = response.json()
        if not isinstance(result, dict):
            raise ValueError()
        return result
    except HTTPException:
        raise
    except (httpx.HTTPError, ValueError, TypeError):
        raise HTTPException(503, "canonical Platform review service is unreachable") from None


def _review_href(value: object) -> str | None:
    """Bound links to Workbench Activity/Review, never a third-party URL."""
    if not isinstance(value, str) or len(value) > 700 or not value.startswith("/"):
        return None
    if value.startswith("//") or any(char in value for char in ("\\", "#", "\r", "\n")):
        return None
    if not (value.startswith("/activity") or value.startswith("/review")):
        return None
    return value


def _compact(payload: object) -> dict:
    """Validate bounded, persisted PG review references before displaying HITL alerts."""
    if not isinstance(payload, dict) or payload.get("schema") != "propria-knowledge-review-queue/v1":
        raise HTTPException(502, "Platform review API returned the wrong schema")
    items = payload.get("items")
    if not isinstance(items, list) or len(items) > 100:
        raise HTTPException(502, "review queue is not bounded")
    rows = []
    for item in items:
        if not isinstance(item, dict):
            raise HTTPException(502, "invalid review row")
        ref = item.get("review_ref")
        if not isinstance(ref, str) or not ref or len(ref) > 512:
            raise HTTPException(502, "review missing canonical ID")
        if item.get("severity") not in ("low", "medium", "high", "critical"):
            raise HTTPException(502, "review has invalid severity")
        if item.get("status") not in ("queued", "deferred"):
            raise HTTPException(502, "review has invalid pending state")
        rows.append({
            "review_ref": ref, "severity": item["severity"], "status": item["status"],
            "kind": str(item.get("kind") or "knowledge_conflict")[:80],
            "scope": str(item.get("scope") or "propria")[:120],
            "summary": str(item.get("summary") or "Review a knowledge conflict")[:240],
            "source_count": max(0, min(int(item.get("source_count") or 0), 20)),
            "case_version": int(item.get("case_version") or 1),
            "review_href": _review_href(item.get("review_href")),
        })
    total = payload.get("total")
    if not isinstance(total, int) or total < len(rows):
        raise HTTPException(502, "unverifiable queue total")
    return {"schema": "propria-knowledge-review-inbox/v1", "status": "ready",
            "total": total, "items": rows, "truncated": bool(payload.get("next_cursor"))}


@router.get("/inbox")
async def inbox(request: Request) -> dict:
    """Display active memory/doc reconciliation cases in Activity; never invent an empty queue."""
    _reviewer(request)
    return _compact(await _request("GET", BASE_PATH))


@router.get("/{review_ref}")
async def detail(request: Request, review_ref: UUID) -> dict:
    """Read exact source-reference/version pins before permitting a decision."""
    _reviewer(request)
    result = await _request("GET", f"{BASE_PATH}/{review_ref}")
    if result.get("schema") != "propria-knowledge-review-detail/v1" or result.get("review_ref") != str(review_ref):
        raise HTTPException(502, "review detail identity mismatch")
    # Do not forward arbitrary source text, raw tool calls, model prompts, or auth data.
    allowed = ("schema", "review_ref", "kind", "severity", "status", "scope", "summary",
               "source_refs", "source_revisions", "case_version", "final_disposition",
               "created_at", "decided_at", "native_write_receipt")
    return {key: result.get(key) for key in allowed}


@router.post("/{review_ref}/decision")
async def decide(request: Request, review_ref: UUID, body: DecisionRequest) -> dict:
    """Forward authenticated owner's exact revision-bound verdict; no native source mutation."""
    reviewer = _reviewer(request)
    result = await _request("POST", f"{BASE_PATH}/{review_ref}/decision", owner=True, body={
        **body.model_dump(), "reviewer_ref": reviewer
    })
    if result.get("review_ref") != str(review_ref) or result.get("decision") != body.decision or result.get("source_applied") is not False:
        raise HTTPException(502, "review decision receipt mismatch")
    return result
