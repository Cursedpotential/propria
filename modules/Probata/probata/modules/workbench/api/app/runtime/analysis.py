"""Connect Read to real Temporal graph queries and version-pinned findings.

Inputs: existing personal-tailnet identity, recorded pins and perspective.
Outputs: actual workflow status and cited content. Effects: query starts/reads,
never source ingestion, approval, graph edits or evidence promotion.
"""
from fastapi import APIRouter, Header, HTTPException, Query, Request
from starlette.concurrency import run_in_threadpool

from app.repo.analysis_artifacts import read_query_artifact
from app.runtime.proffer import _decision_actor
from app.service import proffer
from app.service.matter_mode import MatterModeError, configured_court_case_id, configured_matter_id
from app.service.proffer_errors import ProfferError
from app.types.analysis import AnalysisContent, AnalysisQuery, AnalysisResult, ProjectionSnapshot

router = APIRouter(prefix="/api/analysis", tags=["analysis"])


def _actor_headers(request: Request, key: str | None = None) -> dict[str, str]:
    """Reuse the personal tailnet actor for Go correlation, without a new login gate.

    Inputs: existing middleware identity and optional click key. Output: original
    Go headers. Effects: none; use for query start/status correlation only.
    """
    actor = _decision_actor(request)
    headers = {"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username}
    if key is not None:
        if not key.strip() or len(key) > 512:
            raise HTTPException(422, "A bounded Idempotency-Key is required")
        headers["Idempotency-Key"] = key.strip()
    return headers


def _case() -> dict[str, str]:
    """Resolve the existing personal case without adding a role or approval step."""
    return {"matter_id": str(configured_matter_id("LIVE")), "court_case_id": str(configured_court_case_id("LIVE"))}


def _fail(error):
    """Preserve concrete upstream failures without returning private object errors."""
    raise HTTPException(status_code=error.status_code, detail=error.detail) from None


@router.get("/projections")
async def analysis_projections(request: Request, cursor: str = Query(default="", max_length=2048)):
    """List real completed checkpoints for the existing case with stable pagination.

    Inputs: opaque returned cursor. Output: recorded projection/source pins.
    Effects: one bounded Go read; no graph writes or new approval requirements.
    """
    try:
        case = _case()
        response = await proffer._request("GET", "/reference-import/analysis/projections",
            params={**case, "limit": 50, **({"cursor": cursor} if cursor else {})}, headers=_actor_headers(request))
        payload = proffer._json_payload(response, "analysis projection catalog")
        if not isinstance(payload, dict) or not isinstance(payload.get("items"), list) or len(payload["items"]) > 50 or not isinstance(payload.get("has_more"), bool):
            raise ProfferError("Analysis returned an invalid projection catalog", 502)
        items = [proffer._validated(ProjectionSnapshot, item, "analysis checkpoint") for item in payload["items"]]
        next_cursor = payload.get("next_cursor", "")
        if not isinstance(next_cursor, str) or len(next_cursor) > 2048 or payload["has_more"] != bool(next_cursor) or any(any(getattr(item, k) != v for k, v in case.items()) for item in items):
            raise ProfferError("Analysis checkpoint catalog escaped its case or pagination bounds", 502)
        return {"items": [item.model_dump(mode="json") for item in items], "has_more": payload["has_more"], "next_cursor": next_cursor}
    except (ProfferError, MatterModeError) as error:
        _fail(error)


@router.post("/queries", status_code=202)
async def start_analysis_query(body: AnalysisQuery, request: Request, idempotency_key: str = Header(alias="Idempotency-Key")):
    """Start one bounded query using recorded projection pins and the chosen cutoff.

    Input: pins, perspective and click key. Output: actual workflow/run IDs.
    Effect: shared Go workflow start; use for analysis reads, not source browsing.
    """
    try:
        scope = {**body.model_dump(mode="json", exclude_none=True), **_case()}
        response = await proffer._request("POST", "/reference-import/analysis/queries",
            json={"operating_mode": "LIVE", "scope": scope}, headers=_actor_headers(request, idempotency_key))
        payload = proffer._json_payload(response, "analysis query start")
    except (ProfferError, MatterModeError) as error:
        _fail(error)
    if not isinstance(payload, dict) or not all(isinstance(payload.get(k), str) and payload[k] for k in ("workflow_id", "run_id")):
        raise HTTPException(502, "Analysis starter returned no workflow execution")
    return {"workflow_id": payload["workflow_id"], "run_id": payload["run_id"]}


async def _status(workflow_id: str, request: Request) -> dict:
    """Read bounded Go status using the existing tailnet identity and service token."""
    if len(workflow_id) > 160 or not workflow_id.startswith("approved-context-query:") or any(c in workflow_id for c in "/?#\r\n"):
        raise HTTPException(404, "Unknown analysis query")
    response = await proffer._request("GET", f"/reference-import/analysis/workflows/{workflow_id}", headers=_actor_headers(request))
    payload = proffer._json_payload(response, "analysis query status")
    if not isinstance(payload, dict) or payload.get("workflow_id") != workflow_id or not isinstance(payload.get("outcome"), str):
        raise ProfferError("Analysis returned mismatched workflow status", 502)
    if payload.get("result") is not None:
        result = proffer._validated(AnalysisResult, payload["result"], "analysis query result")
        if any(getattr(result, k) != v for k, v in _case().items()):
            raise ProfferError("Analysis returned a different case", 502)
        payload["result"] = result.model_dump(mode="json")
    return payload


@router.get("/workflows/{workflow_id}")
async def analysis_query_status(workflow_id: str, request: Request):
    """Return actual query progress and exact-version result references.

    Inputs: returned workflow ID and existing tailnet identity. Output: Go status.
    Effects: status read only; use for polling, never report queued as completed.
    """
    try:
        return await _status(workflow_id, request)
    except (ProfferError, MatterModeError) as error:
        _fail(error)


@router.get("/workflows/{workflow_id}/content", response_model=AnalysisContent)
async def analysis_query_content(workflow_id: str, request: Request):
    """Read verified findings with complete source citations from a finished query.

    Input: workflow ID. Output: bounded cited claims. Effects: status and exact
    B2-version read only; use after completion instead of exposing storage keys.
    """
    try:
        status = await _status(workflow_id, request)
        if status["outcome"] != "completed" or not status.get("result"):
            raise HTTPException(409, "Analysis query has not completed")
        result = AnalysisResult.model_validate(status["result"])
        payload = await run_in_threadpool(read_query_artifact, result.artifact)
        content = proffer._validated(AnalysisContent, payload, "analysis findings")
        keys = ("matter_id", "court_case_id", "access_policy_id", "approved_revision_id", "approval_digest", "projection_generation_id", "projection_hash", "perspective", "has_more", "next_cursor")
        if content.actor_subject_uid != request.state.subject_uid or any(getattr(content, k) != getattr(result, k) for k in keys) or len(content.claims) != result.claim_count or len(content.claims) > content.limit or content.has_more != bool(content.next_cursor):
            raise ProfferError("Analysis findings differ from the pinned workflow result", 502)
        for claim in content.claims:
            if any(getattr(claim, k) != getattr(content, k) for k in ("matter_id", "court_case_id", "approved_revision_id", "approval_digest")) or claim.control_generation_id != content.projection_generation_id:
                raise ProfferError("Analysis finding escaped its selected projection", 502)
            if content.perspective == "as_lived" and (claim.source_available_from is None or claim.source_available_from > content.horizon):
                raise ProfferError("Analysis finding falls after the selected as-lived cutoff", 502)
        return content
    except (ProfferError, MatterModeError) as error:
        _fail(error)
