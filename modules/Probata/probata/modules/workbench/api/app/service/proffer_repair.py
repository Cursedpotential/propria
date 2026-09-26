"""Operator decisions for one Proffer preview: approve/reject, and the repair decision with its idempotency binding.

Split out of app/service/proffer.py on 2026-09-22 (that module was over the
300-line cap). The engine call goes through ``app.service.proffer`` so tests that
patch ``proffer._request`` keep covering it. Byline: Claude Code · Fable 5.1 · 2026-09-22.
"""

from __future__ import annotations

import hashlib
import json

from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import (
    ProfferDecisionActor,
    ProfferDecisionRequest,
    ProfferDecisionResponse,
    ProfferRepairDecisionRequest,
    ProfferRepairDecisionResponse,
)


async def decide(
    preview_handle: str,
    request: ProfferDecisionRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ProfferDecisionResponse:
    await proffer._require_mode(preview_handle, mode)
    response = await proffer._request(
        "POST",
        f"/reference-import/previews/{preview_handle}/decision",
        json=request.model_dump(mode="json"),
        headers={
            "X-authentik-uid": actor.subject_uid,
            "X-authentik-username": actor.username,
        },
    )
    result = proffer._validated(
        ProfferDecisionResponse,
        proffer._mode_payload(proffer._json_payload(response, "decision response"), "decision response", mode),
        "decision response",
    )
    if result.preview_handle != preview_handle:
        raise ProfferError("Proffer decision response correlation failed", 502)
    return result


def repair_idempotency_key(
    preview_handle: str,
    request: ProfferRepairDecisionRequest,
    actor: ProfferDecisionActor,
) -> str:
    """Bind repair retries to handle, immutable subject, and canonical choice."""
    canonical = json.dumps(
        request.model_dump(mode="json"),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    )
    digest = hashlib.sha256(f"{preview_handle}\x00{actor.subject_uid}\x00{canonical}".encode()).hexdigest()
    return f"proffer-repair:{digest}"


async def decide_repair(
    preview_handle: str,
    request: ProfferRepairDecisionRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> ProfferRepairDecisionResponse:
    await proffer._require_mode(preview_handle, mode)
    response = await proffer._request(
        "POST",
        f"/reference-import/previews/{preview_handle}/repair-decision",
        json=request.model_dump(mode="json"),
        headers={
            "X-authentik-uid": actor.subject_uid,
            "X-authentik-username": actor.username,
            "Idempotency-Key": repair_idempotency_key(preview_handle, request, actor),
        },
    )
    result = proffer._validated(
        ProfferRepairDecisionResponse,
        proffer._mode_payload(
            proffer._json_payload(response, "repair decision response"), "repair decision response", mode
        ),
        "repair decision response",
    )
    if result.preview_handle != preview_handle:
        raise ProfferError("Proffer repair decision response correlation failed", 502)
    return result
