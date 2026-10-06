"""Retired staged-R2 acquisition admission; historical records stay unchanged.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode


async def acquire_staged(staged_id: str, *, mode: MatterMode):
    """Refuse a new acquisition from retired R2 staging before any lookup or read.

    Inputs: historical staged ID and selected policy. Output: HTTP-facing 409 error.
    Effects: none; no staging lookup, object read or identity rewrite. Pick the
    canonical /api/proffer/upload stream with original bytes for a fresh receipt.
    """
    raise ProfferError(
        "R2 staged acquisition is retired. Historical records remain readable; "
        "re-upload the original file through the current upload flow (/api/proffer/upload).",
        409,
    )
