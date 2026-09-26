"""Context review overlays for one Review message, and the hindsight-only flag.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

Mode ownership is proven from durable state before every call. Reads default to
the as-lived horizon; an as-lived answer is stripped of any foreshadowing member
even if the engine returned one (defence in depth: AGENTS.md "WHY THIS EXISTS",
a leaked hindsight fact silently spoils the ignorant walk). Writes carry the
Authentik actor and an idempotency key derived from the exact submission.
"""

from __future__ import annotations

import hashlib
import json

from app.service.proffer import ProfferError, _json_payload, _mode_payload, _request, _require_mode, _validated
from app.types.context_review import (
    ContextReviewReceipt,
    ContextReviewRequest,
    ContextReviewView,
    ForeshadowingRequest,
    Horizon,
)
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor


def _base(preview_handle: str, message_id: str) -> str:
    return f"/reference-import/previews/{preview_handle}/messages/{message_id}"


async def read_context_review(
    preview_handle: str, message_id: str, *, mode: MatterMode, horizon: Horizon = "as_lived"
) -> ContextReviewView:
    """The message's review history; the foreshadowing history only when horizon is hindsight."""
    await _require_mode(preview_handle, mode)
    response = await _request("GET", f"{_base(preview_handle, message_id)}/context-review?horizon={horizon}")
    payload = _mode_payload(_json_payload(response, "context review"), "context review", mode)
    if horizon != "hindsight":
        payload.pop("foreshadowing", None)
    view = _validated(ContextReviewView, payload, "context review")
    if view.preview_handle != preview_handle or view.message_id.lower() != message_id.lower():
        raise ProfferError("Proffer context review correlation failed", 502)
    if view.horizon != horizon:
        raise ProfferError("Proffer context review answered for a different horizon", 502)
    return view


async def _write(
    path: str, body: dict, actor: ProfferDecisionActor, *, kind: str, mode: MatterMode
) -> ContextReviewReceipt:
    canonical = json.dumps(body, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    key = hashlib.sha256(f"{path}\x00{actor.subject_uid}\x00{canonical}".encode()).hexdigest()
    response = await _request(
        "POST",
        path,
        json=body,
        headers={
            "X-authentik-uid": actor.subject_uid,
            "X-authentik-username": actor.username,
            "Idempotency-Key": f"{kind}:{key}",
        },
    )
    return _validated(
        ContextReviewReceipt,
        _mode_payload(_json_payload(response, f"{kind} receipt"), f"{kind} receipt", mode),
        f"{kind} receipt",
    )


async def write_context_review(
    preview_handle: str,
    message_id: str,
    request: ContextReviewRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ContextReviewReceipt:
    """Append one review revision (to / about / about the child / relevant)."""
    await _require_mode(preview_handle, mode)
    receipt = await _write(
        f"{_base(preview_handle, message_id)}/context-review",
        request.model_dump(mode="json"),
        actor,
        kind="context-review",
        mode=mode,
    )
    if receipt.horizon != "as_lived":
        raise ProfferError("Proffer recorded a context review on the wrong horizon", 502)
    return receipt


async def write_foreshadowing(
    preview_handle: str,
    message_id: str,
    request: ForeshadowingRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ContextReviewReceipt:
    """Set or clear the hindsight-only foreshadowing flag."""
    await _require_mode(preview_handle, mode)
    receipt = await _write(
        f"{_base(preview_handle, message_id)}/foreshadowing",
        request.model_dump(mode="json"),
        actor,
        kind="foreshadowing",
        mode=mode,
    )
    if receipt.horizon != "hindsight":
        raise ProfferError("Proffer recorded the foreshadowing flag on the wrong horizon", 502)
    return receipt
