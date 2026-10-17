"""Read existing canonical knowledge-conflict review cases for Workbench Activity.

Inputs: authenticated Workbench requests, the configured review-queue API, service credential.
Outputs: small status/list of previously persisted review references. Effects: no writes.
Choose for knowledge reconciliation alerts; do not attach them to evidence promotion
or issue approval decisions through this read-only adapter.
"""
from __future__ import annotations

import ipaddress
import os
from urllib.parse import urlsplit

import httpx
from fastapi import APIRouter, HTTPException, Request

router = APIRouter(prefix="/api/knowledge-review", tags=["knowledge-review"])


def _queue_endpoint() -> str:
    """Allow a fixed HTTPS service or a Tailscale/loopback HTTP endpoint only."""
    value = os.environ.get("KNOWLEDGE_REVIEW_QUEUE_URL", "").strip()
    if not value:
        raise HTTPException(503, "Knowledge review backend is not configured")
    try:
        url = urlsplit(value)
        if (
            url.scheme not in {"http", "https"}
            or not url.hostname
            or url.username or url.password or url.fragment or url.query
            or not url.path.startswith("/")
        ):
            raise ValueError("invalid URL")
        if url.scheme == "http":
            address = ipaddress.ip_address(url.hostname)
            if not (address.is_loopback or address in ipaddress.ip_network("100.64.0.0/10")):
                raise ValueError("plain HTTP requires a loopback or tailnet peer")
        return value
    except ValueError:
        raise HTTPException(503, "Knowledge review backend requires a fixed trusted endpoint") from None


def _relative_review_href(value: object) -> str | None:
    """Keep review links in Workbench: no external URLs or javascript navigation."""
    if not isinstance(value, str) or len(value) > 700 or not value.startswith("/"):
        return None
    if value.startswith("//") or any(char in value for char in ("\\", "#", "\r", "\n")):
        return None
    if not (value.startswith("/activity") or value.startswith("/review")):
        return None
    return value


def _compact(payload: object) -> dict:
    """Normalize backend queue metadata into bounded, provenance-bearing display items."""
    if not isinstance(payload, dict) or payload.get("schema") != "propria-knowledge-review-queue/v1":
        raise HTTPException(502, "Knowledge review backend returned an unsupported contract")
    items = payload.get("items")
    if not isinstance(items, list) or len(items) > 200:
        raise HTTPException(502, "Knowledge review backend returned an invalid queue")
    output = []
    for item in items:
        if not isinstance(item, dict):
            raise HTTPException(502, "Invalid review entry")
        ref = item.get("review_ref")
        if not isinstance(ref, str) or not ref or len(ref) > 512:
            raise HTTPException(502, "Review entry has no canonical reference")
        severity = item.get("severity")
        if severity not in {"low", "medium", "high", "critical"}:
            raise HTTPException(502, "Review entry has invalid severity")
        status = item.get("status")
        if status not in {"queued", "deferred"}:
            raise HTTPException(502, "Review queue contains a nonpending entry")
        output.append({
            "review_ref": ref,
            "severity": severity,
            "status": status,
            "kind": str(item.get("kind") or "knowledge_conflict")[:80],
            "scope": str(item.get("scope") or "propria")[:120],
            "summary": str(item.get("summary") or "Review conflicting or stale knowledge")[:240],
            "source_count": max(0, min(int(item.get("source_count") or 0), 100)),
            "review_href": _relative_review_href(item.get("review_href")),
        })
    count = payload.get("total")
    if not isinstance(count, int) or count < len(output):
        count = len(output)
    return {"schema": "propria-knowledge-review-inbox/v1", "status": "ready",
            "total": count, "items": output,
            "truncated": bool(payload.get("next_cursor"))}


@router.get("/inbox")
async def inbox(request: Request) -> dict:
    """Show actionable knowledge reviews; never infer approval from a notification."""
    if not getattr(request.state, "subject_uid", None):
        raise HTTPException(401, "Authenticated Workbench session required")
    endpoint = _queue_endpoint()
    token = os.environ.get("KNOWLEDGE_REVIEW_QUEUE_TOKEN", "").strip()
    if not token:
        raise HTTPException(503, "Knowledge review backend credential is unavailable")
    try:
        async with httpx.AsyncClient(timeout=15, follow_redirects=False, trust_env=False) as client:
            response = await client.get(endpoint,
                                        headers={"Authorization": "Bearer " + token,
                                                 "Accept": "application/json"})
        if response.status_code != 200:
            raise HTTPException(503, f"Knowledge review backend unavailable (HTTP {response.status_code})")
        if len(response.content) > 1024 * 1024:
            raise HTTPException(502, "Knowledge review response exceeds bounded size")
        return _compact(response.json())
    except (httpx.HTTPError, ValueError):
        raise HTTPException(503, "Knowledge review backend connection or response failed") from None
