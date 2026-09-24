# Byline: Claude Code · Sonnet (agent) · 2026-07-22 (C3: corroboration flags)
"""Corroboration flags — requirements addendum 6. Proxies to the spine's /v1/flags.

A flag is an analysis-lane annotation ("this claim needs corroborating
evidence") — the story-to-evidence map for AI-chat material that's the
owner's account, not yet evidence. Flags are metadata; they never mutate
evidence. Thin passthrough, same posture as app/service/inspect.py.
"""

from __future__ import annotations

import hashlib
import hmac
import json
import time
from pathlib import Path

from app.repo.spine_client import SpineError, spine_json

__all__ = ["SpineError", "create_flag", "create_proffer_potential_promotion_flag", "list_flags", "update_flag"]
_PROFFER_DELEGATION_KEY_FILE = Path("/run/secrets/proffer-flag-delegation-key")


def create_flag(payload: dict) -> dict:
    """POST /v1/flags passthrough.

    `payload` shape: `{target_kind, target_id, claim, claim_date_start?,
    claim_date_end?, evidence_wanted?, notes?}` — see
    app/types/inspect.py::FlagCreateRequest.
    """
    return spine_json("POST", "/v1/flags", json=payload)


def create_proffer_potential_promotion_flag(payload: dict) -> dict:
    """Create a preview-bound annotation with atomic current-attempt validation."""
    try:
        key = _PROFFER_DELEGATION_KEY_FILE.read_bytes().strip()
    except OSError as error:
        raise SpineError("Proffer flag delegation is not configured", 503) from error
    if not 32 <= len(key) <= 4096:
        raise SpineError("Proffer flag delegation is not configured", 503)
    issued_at = str(int(time.time()))
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    signature = hmac.new(key, issued_at.encode("ascii") + b"." + canonical, hashlib.sha256).hexdigest()
    return spine_json(
        "POST",
        "/v1/flags/proffer-potential-promotion",
        json=payload,
        headers={"X-Proffer-Flag-Issued-At": issued_at, "X-Proffer-Flag-Signature": signature},
    )


def list_proffer_potential_promotion_flags(preview_handle: str) -> list[dict]:
    """One bounded server snapshot; an overflow is an error, never a partial list."""
    result = spine_json("GET", "/v1/flags/proffer-potential-promotion", params={"preview_handle": preview_handle})
    if not isinstance(result, dict) or not isinstance(result.get("flags"), list):
        raise SpineError("Proffer flag store returned an invalid list", 502)
    return result["flags"]


def list_flags(
    *, status: str | None = None, target_kind: str | None = None, target_id: str | None = None
) -> list[dict]:
    """GET /v1/flags?status=&target_kind= passthrough -> the flag list.

    Backs both the inline "flags on this record/run" views and the
    Evidence Queue page (grouped by evidence_wanted type).
    """
    params: dict[str, str] = {}
    if status:
        params["status"] = status
    if target_kind:
        params["target_kind"] = target_kind
    if target_id:
        params["target_id"] = target_id
    result = spine_json("GET", "/v1/flags", params=params)
    return result if isinstance(result, list) else result.get("flags", [])


def update_flag(flag_id: str, patch: dict) -> dict:
    """PATCH /v1/flags/{id} passthrough — status transitions
    (open -> partial -> corroborated/unobtainable) + notes + linking a
    corroborating artifact once found. `patch` carries only explicitly-set
    keys — see app/types/inspect.py::FlagUpdateRequest.
    """
    return spine_json("PATCH", f"/v1/flags/{flag_id}", json=patch)
