# Byline: Codex · GPT-6.1-sol · 2026-10-11
"""Forward optional discovery through the authenticated engine boundary.

Inputs: admitted case, server dataset pins and explicit owner feedback. Outputs:
real persisted runs, context and receipts. Effects: fixed-route engine requests;
choose over source-processing or publication APIs for discovery review only.
"""

from uuid import UUID

from app.service import proffer
from app.service.matter_mode import MatterModeError, require_scope
from app.service.proffer_errors import ProfferError
from app.types.discovery import FeedbackRequest, RunRequest, Scope
from app.types.proffer import ProfferDecisionActor


def _params(scope: Scope) -> dict[str, str]:
    """Validate requested IDs against configured case; return query pins, no remote effects."""
    try:
        require_scope(scope.mode, scope.matter_id, scope.court_case_id)
    except MatterModeError as error:
        raise ProfferError(error.detail, error.status_code) from None
    return scope.model_dump(mode="json")


def _headers(actor: ProfferDecisionActor, key: str | None = None) -> dict[str, str]:
    """Forward only the server-authenticated actor and optional retry key; no effects."""
    headers = {"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username}
    if key is not None:
        headers["Idempotency-Key"] = key
    return headers


def _checked_run(payload: object, scope: Scope, run_id: UUID | None = None) -> dict:
    """Reject crossed case, mode or run responses before display; output real run, no effects."""
    if not isinstance(payload, dict) or not isinstance(payload.get("request"), dict):
        raise ProfferError("Discovery returned an invalid run", 502)
    request = payload["request"]
    if request.get("scope") != scope.model_dump(mode="json") or (
        run_id is not None and request.get("run_id") != str(run_id)
    ):
        raise ProfferError("Discovery returned a different case, mode or run", 502)
    return payload


async def datasets(scope: Scope, actor: ProfferDecisionActor) -> dict:
    """Read server-allowlisted datasets for this actual case and policy.

    Inputs: admitted scope and authenticated actor. Output: bounded dataset list.
    Effects: one fixed GET; use before choosing inputs, never accept SQL or URLs.
    """
    reply = await proffer._request("GET", "/discovery/datasets", params=_params(scope), headers=_headers(actor))
    payload = proffer._json_payload(reply, "discovery datasets")
    rows = payload.get("items") if isinstance(payload, dict) else None
    if (
        not isinstance(rows, list)
        or len(rows) > 200
        or any(not isinstance(row, dict) or row.get("scope") != scope.model_dump(mode="json") for row in rows)
    ):
        raise ProfferError("Discovery returned mismatched datasets", 502)
    return payload


async def start(body: RunRequest, actor: ProfferDecisionActor, key: str) -> dict:
    """Start one bounded discovery run using verified server selection pins.

    Inputs: run configuration, session actor and retry key. Output: real durable run.
    Effects: one engine POST; no ingestion, source rewrite or owner publication.
    """
    _params(body.scope)
    response = await proffer._request(
        "POST", "/discovery/runs", json=body.model_dump(mode="json"), headers=_headers(actor, key)
    )
    return _checked_run(proffer._json_payload(response, "discovery start"), body.scope, body.run_id)


async def run(run_id: UUID, scope: Scope, actor: ProfferDecisionActor) -> dict:
    """Read one durable run with case/mode correlation.

    Inputs: actual run ID and admitted scope/actor. Output: persisted status.
    Effects: fixed GET; pick to resume review after browser reload.
    """
    response = await proffer._request(
        "GET", f"/discovery/runs/{run_id}", params=_params(scope), headers=_headers(actor)
    )
    return _checked_run(proffer._json_payload(response, "discovery run"), scope, run_id)


async def candidates(run_id: UUID, scope: Scope, actor: ProfferDecisionActor, cursor: str) -> dict:
    """Read one bounded grouped candidate page after validating the run scope.

    Inputs: run, scope, actor and opaque cursor. Output: real candidate queue.
    Effects: two engine GETs; choose for grouped review, never inference on load.
    """
    await run(run_id, scope, actor)
    response = await proffer._request(
        "GET",
        f"/discovery/runs/{run_id}/candidates",
        params={**_params(scope), "cursor": cursor},
        headers=_headers(actor),
    )
    payload = proffer._json_payload(response, "discovery candidates")
    if not isinstance(payload, dict) or not isinstance(payload.get("items"), list) or len(payload["items"]) > 100:
        raise ProfferError("Discovery returned an invalid candidate page", 502)
    return payload


async def context(run_id: UUID, candidate_id: str, scope: Scope, actor: ProfferDecisionActor) -> dict:
    """Read chronological source-linked context for one candidate of a validated run.

    Inputs: run/candidate identity and case/actor. Output: supplied context with locators.
    Effects: two fixed GETs; pick for source verification rather than browser URLs.
    """
    await run(run_id, scope, actor)
    response = await proffer._request(
        "GET", f"/discovery/runs/{run_id}/context/{candidate_id}", params=_params(scope), headers=_headers(actor)
    )
    payload = proffer._json_payload(response, "discovery context")
    if (
        not isinstance(payload, dict)
        or not isinstance(payload.get("candidate"), dict)
        or payload["candidate"].get("candidate_id") != str(candidate_id)
        or not isinstance(payload.get("records"), list)
        or len(payload["records"]) > 400
    ):
        raise ProfferError("Discovery returned mismatched candidate context", 502)
    return payload


async def feedback(run_id: UUID, body: FeedbackRequest, scope: Scope, actor: ProfferDecisionActor, key: str) -> dict:
    """Persist one explicit revision-checked owner correction and verify its receipt.

    Inputs: actual run, reviewed candidate, session actor and retry key. Output: saved
    feedback plus run. Effects: scope GET then feedback POST; no graph publication.
    """
    await run(run_id, scope, actor)
    response = await proffer._request(
        "POST",
        f"/discovery/runs/{run_id}/feedback",
        params=_params(scope),
        json=body.model_dump(mode="json"),
        headers=_headers(actor, key),
    )
    payload = proffer._json_payload(response, "discovery feedback")
    saved = payload.get("feedback") if isinstance(payload, dict) else None
    if (
        not isinstance(saved, dict)
        or saved.get("feedback_id") != str(body.feedback_id)
        or saved.get("candidate_id") != str(body.candidate_id)
        or saved.get("kind") != body.kind
        or saved.get("actor") != actor.model_dump()
    ):
        raise ProfferError("Discovery feedback acknowledgement did not match your submission", 502)
    _checked_run(payload.get("run"), scope, run_id)
    return payload
