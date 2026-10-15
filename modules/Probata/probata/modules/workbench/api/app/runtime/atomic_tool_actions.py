"""Expose source-pinned read-only tool actions through the existing Proffer starter.

Inputs: authenticated actor, Idempotency-Key, one immutable source locator and
reviewed digest. Outputs: actual Temporal workflow/run IDs and bounded status.
Effects: a read-only source tool through the Go gateway. Choose this over the
MCP catalog's direct-call route whenever an operator starts an action.

Byline: Codex · GPT-6.1 · 2026-10-07.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Header, HTTPException, Request
from pydantic import BaseModel, Field

from app.service import proffer
from app.service.matter_mode import MatterModeError, configured_court_case_id, configured_matter_id
from app.service.proffer_errors import ProfferError

router = APIRouter(prefix="/api/atomic-tool-actions", tags=["atomic-tool-actions"])


class SourceAction(BaseModel):
    """Operator-selected immutable source and tool options; no host path or raw bytes."""

    tool_id: str = Field(min_length=3, max_length=128)
    source_ref: str = Field(min_length=8, max_length=2048)
    source_sha256: str = Field(pattern=r"^[0-9a-f]{64}$")
    args: dict[str, Any] = Field(default_factory=dict)


def _actor_headers(request: Request, key: str | None = None) -> dict[str, str]:
    """Forward the verified Workbench actor and optional bounded idempotency key."""
    uid = str(getattr(request.state, "subject_uid", "")).strip()
    username = str(getattr(request.state, "principal", "")).strip()
    if not uid or not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    headers = {"X-authentik-uid": uid, "X-authentik-username": username}
    if key is not None:
        if not key.strip() or len(key) > 512:
            raise HTTPException(status_code=422, detail="a bounded Idempotency-Key is required")
        headers["Idempotency-Key"] = key.strip()
    return headers


@router.post("", status_code=202)
async def start_source_action(body: SourceAction, request: Request, idempotency_key: str = Header(alias="Idempotency-Key")) -> dict[str, str]:
    """Start or join one read-only Temporal tool action with exact source pin.

    Input: selected tool, locator, digest, options, Authentik actor and key.
    Output: actual workflow_id and run_id. Effect: one governed tool activity.
    """
    headers = _actor_headers(request, idempotency_key)
    try:
        matter = configured_matter_id("LIVE")
        court_case = configured_court_case_id("LIVE")
        response = await proffer._request(
            "POST", "/reference-import/atomic-tools/actions",
            json={"operating_mode": "LIVE", "matter_id": str(matter), "court_case_id": str(court_case), **body.model_dump()},
            headers=headers,
        )
        payload = proffer._json_payload(response, "atomic tool start")
    except MatterModeError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    if not isinstance(payload, dict) or not all(isinstance(payload.get(k), str) for k in ("workflow_id", "run_id")):
        raise HTTPException(status_code=502, detail="Proffer starter returned no Temporal execution IDs")
    return {"workflow_id": payload["workflow_id"], "run_id": payload["run_id"]}


@router.get("/{workflow_id}")
async def source_action_status(workflow_id: str, request: Request) -> dict[str, Any]:
    """Read actor-bound Temporal status, result reference and audit metadata.

    Input: workflow ID and authenticated actor. Output: bounded status only.
    Effect: a read from the shared Proffer starter; no direct tool execution.
    """
    if len(workflow_id) > 160 or not workflow_id.startswith("source-pinned-tool:"):
        raise HTTPException(status_code=404, detail="unknown source tool workflow")
    try:
        response = await proffer._request("GET", f"/reference-import/atomic-tools/workflows/{workflow_id}", headers=_actor_headers(request))
        payload = proffer._json_payload(response, "atomic tool status")
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    if not isinstance(payload, dict) or payload.get("workflow_id") != workflow_id:
        raise HTTPException(status_code=502, detail="Proffer starter returned invalid atomic tool status")
    return payload
