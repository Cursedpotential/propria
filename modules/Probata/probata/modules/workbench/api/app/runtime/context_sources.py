"""Submit an explicitly selected AI-chat source to the Go context workflow.

Inputs: verified Workbench actor, original source/version refs and idempotency key.
Outputs: Temporal execution IDs and actor-bound status. Side effects: delegates
to the Go engine; no Python extraction, database write, custody or evidence action.
Choose this for supported AI context ingestion, not the legacy message parser.

Byline: Codex · GPT-6 · 2026-10-08.
"""

from __future__ import annotations

from typing import Literal
from uuid import UUID

import httpx

from fastapi import APIRouter, Header, HTTPException, Query, Request
from pydantic import BaseModel, Field, model_validator

from app.runtime.analysis import _actor_headers
from app.runtime import ai_candidate_review
from app.service import proffer
from app.service.proffer_errors import ProfferError

router = APIRouter(prefix="/api/context", tags=["context-sources"])


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
    historical_source_version_id: str | None = Field(default=None, max_length=36)
    source_sha256: str | None = Field(default=None, pattern=r"^[0-9a-f]{64}$")
    matter_id: str | None = Field(default=None, min_length=1, max_length=160)
    court_case_id: str | None = Field(default=None, min_length=1, max_length=160)
    max_source_bytes: int | None = Field(default=None, ge=1, le=32 * 1024 * 1024)
    max_records: int | None = Field(default=None, ge=1, le=1024)
    max_text_bytes: int | None = Field(default=None, ge=1, le=2 * 1024 * 1024)
    max_chunks: int | None = Field(default=None, ge=1, le=256)
    max_model_calls: int | None = Field(default=None, ge=1, le=512)

    @model_validator(mode="after")
    def validate_recovery_identity(self) -> "AIContextSource":
        """Validate paired recovery and case coordinates before forwarding a native import.

        Inputs: optional historical UUID/hash and case identifiers. Output: unchanged
        request. Effects: none; use to reject incomplete lineage without reading chat text.
        """
        if (self.historical_source_version_id is None) != (self.source_sha256 is None):
            raise ValueError("historical_source_version_id and source_sha256 must be supplied together")
        if self.historical_source_version_id is not None:
            if str(UUID(self.historical_source_version_id)) != self.historical_source_version_id:
                raise ValueError("historical_source_version_id must be a canonical UUID")
        if (self.matter_id is None) != (self.court_case_id is None):
            raise ValueError("matter_id and court_case_id must be supplied together")
        return self


class HumanContextSource(BaseModel):
    """Relay native SMS source coordinates and attribution to the existing Go starter.

    Inputs: original source/version, optional registry references and provider pins.
    Outputs: bounded request. Effects: none; no AI normalization or legacy handler prerequisite.
    """

    source_ref: str = Field(min_length=8, max_length=2048)
    provider_version_id: str = Field(default="", max_length=512)
    package_ref: str | None = Field(default=None, max_length=2048)
    source_kind: Literal["sms"]
    declared_format: Literal["smsbackuprestore_xml"]
    matter_id: str = Field(default="", max_length=160)
    court_case_id: str = Field(default="", max_length=160)
    owner_person_ref: str | None = Field(default=None, max_length=160)
    perspective_person_ref: str | None = Field(default=None, max_length=160)
    perspective: str | None = Field(default=None, max_length=160)
    byte_length: int = Field(default=0, ge=0)
    etag: str | None = Field(default=None, max_length=1024)


@router.post("/sources", status_code=202)
async def submit_ai_context_source(
    body: AIContextSource | HumanContextSource,
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


@router.get("/sources/workflows/{workflow_id}")
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


@router.get("/sources/workflows/{workflow_id}/candidates")
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


@router.post("/sources/workflows/{workflow_id}/candidates/decision")
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


@router.get("/previews/{handle}")
async def human_context_snapshot(handle: str, request: Request) -> dict:
    """Relay actual native source metadata through the existing protected Go connection.

    Inputs: real human handle and existing actor. Outputs: stage refs/counts.
    Effects: read-only Go GET; no legacy preview or normalized table lookup.
    """
    return await _human_context_read(handle, request)


@router.get("/previews/{handle}/content")
async def human_context_content(handle: str, request: Request, offset: int = 0, limit: int = 25) -> dict:
    """Relay one bounded native SBV page without collecting the complete source.

    Inputs: real human handle, offset and limit 1..50. Outputs: native envelopes and continuation.
    Effects: read-only Go GET; attachment payload remains in its original source.
    """
    if offset < 0 or not 1 <= limit <= 50:
        raise HTTPException(status_code=422, detail="native page bounds invalid")
    return await _human_context_read(handle, request, offset=offset, limit=limit)


async def _human_context_read(handle: str, request: Request, *, offset: int | None = None, limit: int = 25) -> dict:
    """Validate a native UUID handle and relay its snapshot or bounded page without writes."""
    from uuid import UUID
    if not handle.startswith("human-"):
        raise HTTPException(status_code=404, detail="unknown native source")
    try:
        identifier = str(UUID(handle[6:]))
    except ValueError:
        raise HTTPException(status_code=404, detail="unknown native source") from None
    path = f"/reference-import/context/previews/human-{identifier}"
    if offset is not None:
        path += f"/content?offset={offset}&limit={limit}"
    try:
        response = await _native_bounded_get(path, _actor_headers(request))
        payload = proffer._json_payload(response, "native human context read")
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    if not isinstance(payload, dict):
        raise HTTPException(status_code=502, detail="Go native reader returned invalid data")
    return payload


async def _native_bounded_get(path: str, headers: dict[str, str]) -> httpx.Response:
    """Stream a native reader response under a four MiB limit using existing service credentials.

    Inputs: validated reader path and current actor headers. Outputs: bounded HTTP response.
    Effects: one read-only request; choose for native pages without changing legacy request behavior.
    """
    if not proffer.settings.proffer_starter_url.strip():
        raise ProfferError("Proffer starter is not configured", 503)
    headers = {key: value for key, value in headers.items() if key.lower() != "authorization"}
    headers.update(proffer._service_authorization_headers())
    limit = 4 * 1024 * 1024
    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            async with client.stream("GET", f"{proffer.settings.proffer_starter_url.rstrip('/')}{path}", headers=headers) as upstream:
                payload = bytearray()
                async for chunk in upstream.aiter_bytes(chunk_size=65536):
                    if len(payload) + len(chunk) > limit:
                        raise ProfferError("native page exceeds relay byte budget", 502)
                    payload.extend(chunk)
                response = httpx.Response(upstream.status_code, headers=upstream.headers, content=bytes(payload))
    except httpx.HTTPError as error:
        raise ProfferError("Native reader unreachable", 502) from error
    if response.status_code >= 400:
        raise ProfferError(proffer._detail(response) or "Native reader rejected request", response.status_code)
    return response
