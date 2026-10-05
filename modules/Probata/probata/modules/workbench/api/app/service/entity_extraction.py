"""Entity and event extraction: authenticated pass-through to the Proffer starter.

Byline: Claude Code · Opus 5.5 · 2026-09-25
Byline amendment: Codex · GPT-6.1-Sol · 2026-10-05 — canonical operating-policy terminology.

Flow (owner 2026-09-25 19:15): Extract -> proposals shown -> owner corrections
-> Validate (pass/fail list) -> Run workflow (Temporal commit). Every route is
bound to a run's durable DEV/LIVE policy exactly like the other Proffer routes, and
every owner act carries the Authentik actor and an Idempotency-Key so a retried
click never writes twice. The engine decides; this module only checks the
mode, forwards, and validates what comes back.
"""

from __future__ import annotations

from typing import Any
from urllib.parse import quote

from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.entity_extraction import (
    CommitStarted,
    CorrectionApplied,
    EntityCommitRequest,
    EntityCorrectionRequest,
    EntityExtractRequest,
    EntityRunRequest,
    EventFromRecordRequest,
    ExtractionStarted,
    MarkedEvent,
    ProposalsResponse,
    RecordView,
    RegistrySearchResponse,
    ValidationReport,
    WorkflowProgress,
)
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor

_WORKFLOW_PREFIXES = {"extraction": "entity-extraction:", "commit": "extraction-commit:"}


def _actor_headers(actor: ProfferDecisionActor, idempotency_key: str | None) -> dict[str, str]:
    headers = {"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username}
    if idempotency_key is not None:
        headers["Idempotency-Key"] = idempotency_key
    return headers


def _body(request: Any, mode: MatterMode, **extra: Any) -> dict[str, Any]:
    body = request.model_dump(mode="json", exclude_none=True)
    body["matter_mode"] = mode
    body.update(extra)
    return body


def _result(model, response, label: str, mode: MatterMode):
    return proffer._validated(model, proffer._mode_payload(proffer._json_payload(response, label), label, mode), label)


def _require_workflow_of_run(kind: str, workflow_id: str, preview_handle: str) -> None:
    if not workflow_id.startswith(_WORKFLOW_PREFIXES[kind] + preview_handle + ":") or len(workflow_id) > 512:
        raise ProfferError("workflow does not belong to this run", 404)


async def extract(
    request: EntityExtractRequest, actor: ProfferDecisionActor, idempotency_key: str, *, mode: MatterMode
) -> ExtractionStarted:
    await proffer._require_mode(request.preview_handle, mode)
    response = await proffer._request(
        "POST",
        "/reference-import/entities/extract",
        json=_body(request, mode),
        headers=_actor_headers(actor, idempotency_key),
    )
    return _result(ExtractionStarted, response, "extraction start", mode)


async def workflow_progress(kind: str, workflow_id: str, preview_handle: str, *, mode: MatterMode) -> WorkflowProgress:
    await proffer._require_mode(preview_handle, mode)
    _require_workflow_of_run(kind, workflow_id, preview_handle)
    segment = "extractions" if kind == "extraction" else "commits"
    response = await proffer._request(
        "GET",
        f"/reference-import/entities/{segment}/{quote(workflow_id, safe='')}",
        params={"preview_handle": preview_handle},
    )
    return _result(WorkflowProgress, response, f"{kind} progress", mode)


async def proposals(preview_handle: str, *, mode: MatterMode) -> ProposalsResponse:
    await proffer._require_mode(preview_handle, mode)
    response = await proffer._request(
        "GET",
        "/reference-import/entities/proposals",
        params={"preview_handle": preview_handle, "matter_mode": mode},
    )
    result = _result(ProposalsResponse, response, "entity proposals", mode)
    if result.preview_handle != preview_handle:
        raise ProfferError("entity proposals correlation failed", 502)
    return result


async def correct(
    request: EntityCorrectionRequest, actor: ProfferDecisionActor, idempotency_key: str, *, mode: MatterMode
) -> CorrectionApplied:
    await proffer._require_mode(request.preview_handle, mode)
    if (request.target == "entity") != (request.entity is not None) or (request.target == "event") != (
        request.event is not None
    ):
        raise ProfferError("a correction carries exactly the body its target names", 422)
    response = await proffer._request(
        "POST",
        "/reference-import/entities/corrections",
        json=_body(request, mode),
        headers=_actor_headers(actor, idempotency_key),
    )
    return _result(CorrectionApplied, response, "entity correction", mode)


async def mark_event(
    request: EventFromRecordRequest, actor: ProfferDecisionActor, idempotency_key: str, *, mode: MatterMode
) -> MarkedEvent:
    await proffer._require_mode(request.preview_handle, mode)
    response = await proffer._request(
        "POST",
        "/reference-import/events/from-record",
        json=_body(request, mode),
        headers=_actor_headers(actor, idempotency_key),
    )
    return _result(MarkedEvent, response, "event mark", mode)


async def record(preview_handle: str, record_id: str, *, mode: MatterMode) -> RecordView:
    await proffer._require_mode(preview_handle, mode)
    response = await proffer._request(
        "GET",
        f"/reference-import/entities/records/{quote(record_id, safe='')}",
        params={"preview_handle": preview_handle},
    )
    result = _result(RecordView, response, "record", mode)
    if result.record_id != record_id:
        raise ProfferError("record correlation failed", 502)
    return result


async def search_registry(query: str, limit: int) -> RegistrySearchResponse:
    response = await proffer._request("GET", "/reference-import/entities/registry", params={"q": query, "limit": limit})
    return proffer._validated(
        RegistrySearchResponse, proffer._json_payload(response, "registry search"), "registry search"
    )


async def validate(request: EntityRunRequest, *, mode: MatterMode) -> ValidationReport:
    await proffer._require_mode(request.preview_handle, mode)
    response = await proffer._request("POST", "/reference-import/entities/validate", json=_body(request, mode))
    return _result(ValidationReport, response, "commit validation", mode)


async def commit(
    request: EntityCommitRequest, actor: ProfferDecisionActor, idempotency_key: str, *, mode: MatterMode
) -> CommitStarted:
    await proffer._require_mode(request.preview_handle, mode)
    response = await proffer._request(
        "POST",
        "/reference-import/entities/commit",
        json=_body(request, mode),
        headers=_actor_headers(actor, idempotency_key),
    )
    return _result(CommitStarted, response, "commit start", mode)
