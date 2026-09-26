"""Authenticated pass-through to the engine's folder-batch routes.

Byline: Claude Code · Opus 5 · 2026-09-22.

The engine owns fan-out, the durable batch workflow and the folder-locator
authority (`modules/engine/runtimeapi/proffer_batch.go`). This module adds
nothing but the Workbench's scope check and its TEST/Live echo, so a batch
start is the same governed action as a single start.
"""

from __future__ import annotations

from app.service.matter_mode import require_scope
from app.service.proffer import _json_payload, _mode_call, _mode_payload, _request, _validated
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer_batch import (
    ProfferBatchStartRequest,
    ProfferBatchStartResponse,
    ProfferBatchStatus,
)


async def start_batch(
    request: ProfferBatchStartRequest, *, mode: MatterMode
) -> ProfferBatchStartResponse:
    """Start one durable batch over one folder."""
    if request.matter_mode != mode:
        raise ProfferError("matter_mode in the start-batch body must match the mode query", 409)
    _mode_call(require_scope, mode, request.matter_id, request.court_case_id)
    body = request.model_dump(mode="json", exclude={"matter_mode"})
    body["source_context_ref"] = body.get("source_context_ref") or ""
    response = await _request("POST", "/reference-import/start-batch", json=body)
    payload = _mode_payload(
        _json_payload(response, "start-batch response"), "start-batch response", mode
    )
    result = _validated(ProfferBatchStartResponse, payload, "start-batch response")
    if result.batch_id != request.batch_id:
        raise ProfferError("the engine returned a different batch id", 502)
    return result


async def batch_status(batch_id: str, *, mode: MatterMode) -> ProfferBatchStatus:
    """Per-item status and counts for one batch."""
    response = await _request("GET", f"/reference-import/batches/{batch_id}")
    payload = _mode_payload(_json_payload(response, "batch status"), "batch status", mode)
    result = _validated(ProfferBatchStatus, payload, "batch status")
    if result.batch_id != batch_id:
        raise ProfferError("the engine returned a different batch id", 502)
    return result
