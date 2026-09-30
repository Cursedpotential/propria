"""Cancel one Proffer run through the engine's Temporal cancellation (D05-C06).

The engine records the actor and reason in the run's own history as a Signal,
then asks Temporal to cancel the run; this BFF never edits an engine row. The
engine call goes through ``app.service.proffer`` so tests that patch
``proffer._request`` cover it. Byline: Claude Code · Opus 5.5 · 2026-09-28.
"""

from __future__ import annotations

from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor
from app.types.proffer_cancel import ProfferCancelRequest, ProfferCancelResponse


async def cancel(
    preview_handle: str,
    request: ProfferCancelRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ProfferCancelResponse:
    await proffer._require_mode(preview_handle, mode)
    response = await proffer._request(
        "POST",
        f"/reference-import/previews/{preview_handle}/cancel",
        json=request.model_dump(mode="json"),
        headers={
            "X-authentik-uid": actor.subject_uid,
            "X-authentik-username": actor.username,
        },
    )
    result = proffer._validated(
        ProfferCancelResponse,
        proffer._mode_payload(proffer._json_payload(response, "cancel response"), "cancel response", mode),
        "cancel response",
    )
    if result.preview_handle != preview_handle:
        raise ProfferError("Proffer cancel response correlation failed", 502)
    return result
