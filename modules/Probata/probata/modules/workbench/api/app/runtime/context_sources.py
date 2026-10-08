"""Submit an explicitly selected AI-chat source to the Go context workflow.

Inputs: verified Workbench actor, original source/version refs and idempotency key.
Outputs: Temporal execution IDs and actor-bound status. Side effects: delegates
to the Go engine; no Python extraction, database write, custody or evidence action.
Choose this for supported AI context ingestion, not the legacy message parser.

Byline: Codex · GPT-6 · 2026-10-08.
"""

from __future__ import annotations

from typing import Literal

from fastapi import APIRouter, Header, HTTPException, Query, Request
from pydantic import BaseModel, Field

from app.runtime.analysis import _actor_headers
from app.runtime import ai_candidate_review
from app.service import proffer
from app.service.proffer_errors import ProfferError

router = APIRouter(prefix="/api/context/sources", tags=["context-sources"])


class AIContextSource(BaseModel):
    """Identify one complete original AI-chat source and its provider version.

    Inputs: source locator, exact provider version, optional package locator and
    explicit native-format selection. Output: bounded Go starter request.
    Effects: none; choose only for an AI-chat export, never generic JSON/SMS.
    """

    source_ref: str = Field(min_length=8, max_length=2048)
    provider_version_id: str = Field(min_length=1, max_length=512)
    package_ref: str | None = Field(default=None, max_length=2048)
    source_kind: Literal["ai_chat"] = "ai_chat"
    declared_format: Literal["chatgpt", "claude", "gemini_markdown"]


@router.post("", status_code=202)
async def submit_ai_context_source(
    body: AIContextSource,
    request: Request,
    idempotency_key: str = Header(alias="Idempotency-Key"),
) -> dict[str, str]:
    """Start or join the Go context workflow for one version-pinned AI export.

    Inputs: actor, key and original source/version/package refs. Output: real
    workflow/run IDs. Effect: Go Temporal start only; no direct Python reader.
    """
    headers = _actor_headers(request, idempotency_key)
    try:
        response = await proffer._request(
            "POST", "/reference-import/context/sources",
            json={"contract_version": "context-v1", **body.model_dump(exclude_none=True)},
            headers=headers,
        )
        payload = proffer._json_payload(response, "AI context source start")
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    if not isinstance(payload, dict) or not all(isinstance(payload.get(key), str) and payload[key] for key in ("workflow_id", "run_id")):
        raise HTTPException(status_code=502, detail="Go context starter returned no Temporal execution IDs")
    return {"workflow_id": payload["workflow_id"], "run_id": payload["run_id"]}


@router.get("/workflows/{workflow_id}")
async def ai_context_source_status(workflow_id: str, request: Request) -> dict:
    """Read the actor-owned status of a previously started context workflow.

    Inputs: returned workflow ID and verified actor. Output: Go's bounded
    refs/counts/status. Effect: status read only, without returning source body.
    """
    if not workflow_id or len(workflow_id) > 160 or any(char in workflow_id for char in "/?#"):
        raise HTTPException(status_code=404, detail="unknown context workflow")
    try:
        response = await proffer._request(
            "GET", f"/reference-import/context/workflows/{workflow_id}",
            headers=_actor_headers(request),
        )
        payload = proffer._json_payload(response, "AI context source status")
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    if not isinstance(payload, dict) or payload.get("workflow_id") != workflow_id:
        raise HTTPException(status_code=502, detail="Go context starter returned invalid workflow status")
    return payload


@router.get("/workflows/{workflow_id}/candidates")
async def ai_context_candidates(workflow_id: str, request: Request,
                                after_id: str = Query(default="", max_length=128)):
    """Read staged candidates for the workflow's verified original source.

    Inputs: actual workflow ID and returned page cursor. Output: bounded review
    page. Effects: Go status/list reads only; use for the existing Read screen.
    """
    status = await ai_context_source_status(workflow_id, request)
    try:
        return await ai_candidate_review.candidates(status, request, after_id)
    except ProfferError as error:
        raise HTTPException(error.status_code, error.detail) from None


@router.post("/workflows/{workflow_id}/candidates/decision")
async def ai_context_candidate_decision(workflow_id: str, body: ai_candidate_review.CandidateDecision,
                                        request: Request, idempotency_key: str = Header(alias="Idempotency-Key")):
    """Record the owner's choice for one candidate without additional gates.

    Inputs: actual workflow, item digest, decision and retry key. Output: committed
    decision receipt. Effects: existing Go decision; graph publication is separate.
    """
    status = await ai_context_source_status(workflow_id, request)
    try:
        return await ai_candidate_review.decide(status, body, request, idempotency_key)
    except ProfferError as error:
        raise HTTPException(error.status_code, error.detail) from None
