"""Authenticated pass-through to the engine's folder-batch routes.

Byline: Claude Code · Opus 5 · 2026-09-22.

The engine owns fan-out, the durable batch workflow and the folder-locator
authority (`modules/engine/runtimeapi/proffer_batch.go`). This module adds
nothing but the Workbench's scope check and its TEST/Live echo, so a batch
start is the same governed action as a single start.
"""

from __future__ import annotations

from app.service.matter_mode import configured_court_case_id, configured_matter_id, require_scope
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
    """Start one Live folder batch and verify its durable explicit policy.

    Inputs: case scope, source folder and canonical flag. Output: batch handle.
    Effects: one engine POST then durable status GET; Dev dispatch is denied.
    Pick over single start for folder fan-out owned by the engine.
    """
    if mode == "DEV":
        raise ProfferError("Development writes require an isolated data workspace; no canonical write was dispatched", 409)
    if request.matter_mode != mode:
        raise ProfferError("matter_mode in the start-batch body must match the mode query", 409)
    _mode_call(require_scope, mode, request.matter_id, request.court_case_id)
    body = request.model_dump(mode="json", exclude={"matter_mode"})
    body["operating_mode"] = mode
    body["source_context_ref"] = body.get("source_context_ref") or ""
    response = await _request("POST", "/reference-import/start-batch", json=body)
    payload = _mode_payload(
        _json_payload(response, "start-batch response"), "start-batch response", mode
    )
    result = _validated(ProfferBatchStartResponse, payload, "start-batch response")
    if result.batch_id != request.batch_id:
        raise ProfferError("the engine returned a different batch id", 502)
    await batch_status(result.batch_id, mode=mode)
    return result


async def batch_status(batch_id: str, *, mode: MatterMode) -> ProfferBatchStatus:
    """Read a batch only with explicit durable policy and fixed-case scope proof.

    Inputs: batch ID and requested policy. Output: items/counts or a closed error.
    Effects: one engine GET. Pick over operation detail for folder progress.
    """
    response = await _request("GET", f"/reference-import/batches/{batch_id}")
    raw = _json_payload(response, "batch status")
    if not isinstance(raw, dict) or (
        raw.get("operating_mode") != mode
        or raw.get("matter_id") != str(_mode_call(configured_matter_id, mode))
        or raw.get("court_case_id") != str(_mode_call(configured_court_case_id, mode))
    ):
        raise ProfferError("Batch has no matching explicit durable operating policy and case scope", 409)
    # Durable proof fields belong to the engine boundary, not the projection's
    # item/count response model; validate before discarding them.
    payload = _mode_payload(
        {key: value for key, value in raw.items() if key not in {"operating_mode", "matter_id", "court_case_id"}},
        "batch status", mode,
    )
    result = _validated(ProfferBatchStatus, payload, "batch status")
    if result.batch_id != batch_id:
        raise ProfferError("the engine returned a different batch id", 502)
    return result
