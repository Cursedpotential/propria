"""Resolve explicit operating policy without selecting another case identity.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from typing import Annotated

from fastapi import Depends, HTTPException, Query, Request

from app.service.case_scope import verify_case_scope
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode

DEV_WRITE_DENIED = (
    "Development writes require an isolated data workspace; no canonical write was dispatched"
)


async def resolve_operating_mode(request: Request, mode: Annotated[MatterMode, Query()] = "LIVE") -> MatterMode:
    """Validate a query policy and deny Dev mutations before route dispatch.

    Inputs: HTTP method and optional mode (legacy aliases normalize at input).
    Outputs: canonical DEV/LIVE, default LIVE, or HTTP 409/422.
    Side effects: read-only authoritative case verification after validation.
    Use for operating policy, not unrelated search modes or auth.
    """
    if mode == "DEV" and request.method in {"POST", "PUT", "PATCH", "DELETE"}:
        raise HTTPException(status_code=409, detail=DEV_WRITE_DENIED)
    try:
        await verify_case_scope(mode)
    except ProfferError as error:
        raise HTTPException(status_code=error.status_code, detail=error.detail) from None
    return mode


OperatingMode = Annotated[MatterMode, Depends(resolve_operating_mode)]
