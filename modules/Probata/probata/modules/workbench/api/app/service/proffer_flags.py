"""Durable, reversible preview annotations backed by the existing flag spine."""

from __future__ import annotations

import json
from datetime import datetime
from typing import Any

from app.service import flags as flags_service
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor
from app.types.proffer_flags import (
    ProfferPotentialPromotionFlag,
    ProfferPotentialPromotionFlagRequest,
)

_NOTES_VERSION = "proffer-potential-promotion/v1"


def _normalize(row: dict[str, Any]) -> ProfferPotentialPromotionFlag | None:
    try:
        metadata = json.loads(str(row.get("notes") or ""))
    except (TypeError, ValueError, json.JSONDecodeError):
        return None  # Ordinary run flags may have free-text notes.
    if not isinstance(metadata, dict) or metadata.get("contract") != _NOTES_VERSION:
        return None
    try:
        if row.get("target_kind") != "run" or row.get("target_id") != metadata["preview_handle"]:
            raise ValueError("governed preview handle does not match the flag target")
        if metadata["classification"] != "potential_promotion":
            raise ValueError("governed classification is invalid")
        return ProfferPotentialPromotionFlag(
            flag_id=str(row.get("flag_id") or row["id"]),
            preview_handle=metadata["preview_handle"],
            matter_mode=metadata["matter_mode"],
            scope=metadata["scope"],
            target_id=metadata["target_id"],
            attempt_id=metadata["attempt_id"],
            reason=str(row["claim"]),
            actor_subject_uid=metadata["actor_subject_uid"],
            actor_username=metadata["actor_username"],
            flagged_at=datetime.fromisoformat(str(row["created_at"])),
            status=str(row.get("status") or "open"),
        )
    except (KeyError, TypeError, ValueError) as error:
        raise ProfferError("Potential-promotion flag store returned an invalid governed row", 502) from error


def create_potential_promotion_flag(
    preview_handle: str,
    mode: MatterMode,
    request: ProfferPotentialPromotionFlagRequest,
    actor: ProfferDecisionActor,
) -> ProfferPotentialPromotionFlag:
    if request.scope not in ("record", "chunk"):
        raise ProfferError("Preview content target is invalid or unsupported", 422)
    try:
        row = flags_service.create_proffer_potential_promotion_flag(
            {
                "preview_handle": preview_handle,
                "matter_mode": mode,
                "scope": request.scope,
                "target_id": request.target_id,
                "attempt_id": request.attempt_id,
                "actor_subject_uid": actor.subject_uid,
                "actor_username": actor.username,
                "claim": request.reason,
            }
        )
    except flags_service.SpineError as error:
        if error.status_code == 409:
            raise ProfferError("Preview attempt changed or target is no longer present", 409) from error
        if error.status_code == 422:
            raise ProfferError("Preview content target is invalid or unsupported", 422) from error
        raise ProfferError("Potential-promotion flag store is unavailable", 502) from error
    normalized = _normalize(row)
    if normalized is None:
        raise ProfferError("Potential-promotion flag store omitted its Proffer contract", 502)
    return normalized


def list_potential_promotion_flags(preview_handle: str, mode: MatterMode) -> list[ProfferPotentialPromotionFlag]:
    result: list[ProfferPotentialPromotionFlag] = []
    for row in flags_service.list_proffer_potential_promotion_flags(preview_handle):
        normalized = _normalize(row)
        if normalized and normalized.preview_handle == preview_handle and normalized.matter_mode == mode:
            result.append(normalized)
    return sorted(result, key=lambda item: (item.flagged_at, item.flag_id))
