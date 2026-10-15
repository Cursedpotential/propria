"""Authenticated pass-through to the engine's folder-batch routes.

Byline: Claude Code · Opus 5 · 2026-09-22.

The engine owns fan-out, the durable batch workflow and the folder-locator
authority (`modules/engine/runtimeapi/proffer_batch.go`). This module adds
nothing but the Workbench's scope check and its TEST/Live echo, so a batch
start is the same governed action as a single start.
"""

from __future__ import annotations

import re
from urllib.parse import unquote, urlsplit

from app.repo.object_store_client import validate_source_key
from app.service.matter_mode import (
    configured_court_case_id,
    configured_matter_id,
    require_scope,
)
from app.service.proffer import (
    _json_payload,
    _mode_call,
    _mode_payload,
    _request,
    _validated,
)
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer_batch import (
    ProfferBatchStartRequest,
    ProfferBatchStartResponse,
    ProfferBatchStatus,
)
from app.types.source_roots import configured_source_roots


def validate_unit_roots(folder_ref: str, unit_roots: list[str]) -> None:
    """Validate distinct nonoverlapping package folders under one configured source parent.

    Inputs: common folder locator and up to 32 exact unit locators. Output: none
    or a closed 422 error. Effects: configuration reads only; choose for unit mode,
    preserving ordinary recursive folder batches when unit_roots is absent.
    """
    def coordinates(value: str, *, allow_root: bool = False) -> tuple[str, str, str]:
        """Parse one bounded folder URI without rewriting its provider key.

        Input: source locator. Output: scheme, bucket and slash-terminated key.
        Effects: none; choose for membership comparison, never source transformation.
        """
        if not isinstance(value, str) or not 8 <= len(value) <= 2048 or re.search(r"%(?![0-9a-fA-F]{2})", value):
            raise ValueError("invalid folder locator")
        parsed = urlsplit(value)
        if not parsed.scheme or not parsed.netloc or parsed.query or parsed.fragment or "@" in parsed.netloc:
            raise ValueError("invalid folder locator")
        key = unquote(parsed.path.removeprefix("/"))
        if validate_source_key(key, allow_empty=allow_root) != key or "." in key.split("/"):
            raise ValueError("invalid folder locator")
        return parsed.scheme, parsed.netloc, key.rstrip("/") + "/" if key else ""

    try:
        if not 1 <= len(unit_roots) <= 32:
            raise ValueError("invalid number of units")
        parent = coordinates(folder_ref, allow_root=True)
        allowed = any(parent[0] == root.scheme and parent[1] == root.bucket and parent[2].startswith(root.key_prefix)
                      for root in configured_source_roots().values())
        units = [coordinates(value) for value in unit_roots]
        if not allowed or len(set(units)) != len(units) or any(
            unit[:2] != parent[:2] or not unit[2].startswith(parent[2]) for unit in units
        ) or any(left != right and right[2].startswith(left[2]) for left in units for right in units):
            raise ValueError("invalid package scope")
    except (ValueError, TypeError):
        raise ProfferError("Package folders must be distinct, nonoverlapping and inside the configured source folder", 422) from None


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
    if request.unit_roots is not None:
        validate_unit_roots(request.folder_ref, request.unit_roots)
    body = request.model_dump(mode="json", exclude={"matter_mode", "unit_roots"})
    if request.unit_roots is not None:
        body["unit_roots"] = request.unit_roots
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
