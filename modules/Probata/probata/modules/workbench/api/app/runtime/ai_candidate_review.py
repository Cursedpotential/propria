"""Bridge staged AI candidates to the existing owner review API.

Inputs: the workflow's verified Stage source pin and an explicit item decision.
Outputs: actual candidates or committed decision receipts. Effects: bounded Go
requests only; no chat normalization, extra approval or graph-success inference.
Choose for native context-v1 exports, not SMS preview overlays.
"""
from typing import Literal
from uuid import UUID

from fastapi import HTTPException, Request
from pydantic import BaseModel, Field, ValidationError

from app.runtime.analysis import _actor_headers
from app.service import proffer
from app.service.proffer_errors import ProfferError


class ReviewSource(BaseModel):
    """Keep the exact verified native source returned by candidate staging.

    Input: Stage result. Output: original version/hash pin for Go verification.
    Effects: none; never replace empty native object IDs with generated IDs.
    """
    source_version_id: UUID
    source_ref: str = Field(pattern=r"^b2://", min_length=8, max_length=2048)
    version_id: str | None = Field(default=None, min_length=1, max_length=512)
    source_object_id: Literal[""] = ""
    source_sha256: str = Field(pattern=r"^[a-f0-9]{64}$")
    prepared_ref: str = Field(pattern=r"^b2://", min_length=8, max_length=4096)


class CandidateDecision(BaseModel):
    """Submit one inspected candidate's existing decision and content digest.

    Inputs: displayed item identity, exact digest and owner choice. Output: Go
    decision body. Effects: none until submitted; no batch or implicit approval.
    """
    candidate_id: UUID
    expected_content_sha256: str = Field(pattern=r"^[a-f0-9]{64}$")
    decision: Literal["approved", "rejected", "needs_info"]


def source_pin(status: dict) -> dict:
    """Resolve review pins only from the workflow's verified atomic Stage result.

    Input: actual context workflow status. Output: native source pin. Effects:
    none; missing staging remains unavailable rather than an invented empty list.
    """
    if not status.get("review_source"):
        raise HTTPException(409, "Candidate staging has not supplied its verified source yet")
    try:
        return ReviewSource.model_validate(status["review_source"]).model_dump(mode="json")
    except ValidationError:
        raise HTTPException(502, "Candidate staging returned an invalid source pin") from None


async def candidates(status: dict, request: Request, after_id: str) -> dict:
    """Read one bounded candidate page from the existing exact-source review API.

    Inputs: verified workflow status and page cursor. Output: real pending and
    decided candidates. Effects: Go GET only; choose for Read's review panel.
    """
    pin = source_pin(status)
    response = await proffer._request("GET", "/reference-import/ai-candidates",
        params={**{k: v for k, v in pin.items() if v is not None},
                "preview_handle": "", "matter_mode": "LIVE", "after_id": after_id},
        headers=_actor_headers(request))
    payload = proffer._json_payload(response, "AI candidate review")
    if (not isinstance(payload, dict) or payload.get("source_version_id") != pin["source_version_id"]
            or payload.get("matter_mode") != "LIVE" or not isinstance(payload.get("candidates"), list)
            or len(payload["candidates"]) > 200 or not isinstance(payload.get("next_cursor"), str)):
        raise ProfferError("AI candidate review returned a mismatched source or page", 502)
    for row in payload["candidates"]:
        if not isinstance(row, dict) or not isinstance(row.get("candidate"), dict):
            raise ProfferError("AI candidate review returned an invalid item", 502)
        try:
            CandidateDecision(candidate_id=row.get("candidate_id"),
                              expected_content_sha256=row.get("content_sha256"), decision="needs_info")
        except ValidationError:
            raise ProfferError("AI candidate review returned invalid item provenance", 502) from None
        original = row["candidate"]
        if any(original.get(k) != v for k, v in pin.items()):
            raise ProfferError("AI candidate escaped its exact original source pin", 502)
    return {**payload, "source": pin}


async def decide(status: dict, body: CandidateDecision, request: Request, key: str) -> dict:
    """Record one explicit semantic choice through the existing Go decision route.

    Inputs: verified Stage pin, displayed candidate digest, choice and click key.
    Output: committed receipt plus separate upstream graph status when supplied.
    Effects: one Go decision; approval is never reported as graph completion.
    """
    pin = source_pin(status)
    submitted = body.model_dump(mode="json")
    response = await proffer._request("POST", "/reference-import/ai-candidates/decision",
        json={**pin, **submitted, "preview_handle": "", "matter_mode": "LIVE"},
        headers=_actor_headers(request, key))
    payload = proffer._json_payload(response, "AI candidate decision")
    if (not isinstance(payload, dict) or payload.get("candidate_id") != submitted["candidate_id"]
            or payload.get("decision") != submitted["decision"] or payload.get("matter_mode") != "LIVE"
            or not isinstance(payload.get("decision_id"), str) or not payload["decision_id"]
            or not isinstance(payload.get("request_digest"), str) or not payload["request_digest"]):
        raise ProfferError("AI candidate decision returned an invalid receipt", 502)
    return payload
