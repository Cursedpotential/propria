"""Verify neutral deployment scope against the engine's bounded identity receipt.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from typing import Any

from app.service.matter_mode import MatterModeError, configured_court_case_id, configured_matter_id
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode


async def verify_case_scope(mode: MatterMode, header: dict[str, Any] | None = None) -> None:
    """Compare the configured pair with the single approved engine case.

    Inputs: canonical mode and optional already-fetched header. Outputs: None or
    ProfferError (503 for bad config, 502 for a wrong/unverifiable case).
    Side effects: one authenticated GET when header is absent; no writes/catalog
    reads. Pick this for scope admission, not preview operating-mode recovery.
    """
    from app.service import proffer

    try:
        matter_id = str(configured_matter_id(mode))
        court_case_id = str(configured_court_case_id(mode))
    except MatterModeError as error:
        raise ProfferError(error.detail, error.status_code) from None
    if header is None:
        response = await proffer._request("GET", "/case-identity/scope", params={"mode": mode})
        header = proffer._json_payload(response, "canonical case header")
    if not isinstance(header, dict):
        raise ProfferError("The authoritative case header is unavailable", 502)
    matter = header.get("matter")
    court_case = header.get("court_case")
    if (
        header.get("mode") != mode
        or not isinstance(matter, dict)
        or not isinstance(court_case, dict)
        or matter.get("id") != matter_id
        or court_case.get("id") != court_case_id
        or court_case.get("matter_id") != matter_id
    ):
        raise ProfferError("Configured case scope does not match the authoritative case header", 502)
