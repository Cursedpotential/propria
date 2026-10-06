"""Authenticated adapter for durable Proffer source-context receipts.

Byline: Codex · GPT-5.6-Sol · 2026-08-30.
Byline: Claude Code · Opus 5.5 · 2026-09-25 (read-back for the Review Actions panel).
Byline amendment: Codex · GPT-6.1-Sol · 2026-10-05 — explicit canonical policy on authored context.
"""

from __future__ import annotations

import hashlib
import json

from app.service.matter_mode import MatterModeError, require_scope
from app.service.proffer import ProfferError, _json_payload, _mode_payload, _request, _require_mode, _validated
from app.types.source_context import ProfferRunSourceContext, SourceContextCreateRequest, SourceContextReceipt
from app.types.proffer import MatterMode, ProfferDecisionActor


async def run_source_context(preview_handle: str, *, mode: MatterMode) -> ProfferRunSourceContext:
    """Read one run's registration facts and newest operator context revision.

    Mode ownership is proven from durable state first (it survives a BFF restart);
    the engine answer is validated fail-closed and echoed with the active mode.
    """
    await _require_mode(preview_handle, mode)
    response = await _request("GET", f"/reference-import/previews/{preview_handle}/source-context")
    result = _validated(
        ProfferRunSourceContext,
        _mode_payload(_json_payload(response, "run source context"), "run source context", mode),
        "run source context",
    )
    if result.preview_handle != preview_handle:
        raise ProfferError("Proffer run source context correlation failed", 502)
    return result


async def create_source_context(
    request: SourceContextCreateRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> SourceContextReceipt:
    """Author one Live context receipt under the exact configured case scope.

    Inputs: source assertions, authenticated actor and canonical mode. Output: receipt.
    Effects: authenticated idempotent POST; Dev denies before hashing/dispatch.
    Pick over run_source_context for a new append-only operator context revision.
    """
    if mode == "DEV":
        raise ProfferError("Development writes require an isolated data workspace; no source context was dispatched", 409)
    if request.matter_mode != mode:
        raise ProfferError("matter_mode in the source-context body must match the mode query", 409)
    try:
        require_scope(mode, request.matter_id, request.court_case_id)
    except MatterModeError as error:
        raise ProfferError(error.detail, error.status_code) from None
    canonical = json.dumps(
        request.model_dump(mode="json"),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    )
    key = hashlib.sha256(f"{actor.subject_uid}\x00{canonical}".encode()).hexdigest()
    response = await _request(
        "POST",
        "/reference-import/source-contexts",
        json={**request.model_dump(mode="json", exclude={"matter_mode"}), "operating_mode": mode},
        headers={
            "X-authentik-uid": actor.subject_uid,
            "X-authentik-username": actor.username,
            "Idempotency-Key": f"proffer-source-context:{key}",
        },
    )
    return _validated(
        SourceContextReceipt,
        _mode_payload(_json_payload(response, "source context response"), "source context response", mode),
        "source context response",
    )
