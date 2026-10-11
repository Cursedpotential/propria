"""Canonical operating modes with input-only aliases for rollout clients.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from typing import Annotated, Literal

from pydantic import BeforeValidator

CanonicalMatterMode = Literal["DEV", "LIVE"]


def _normalize_mode(value: object) -> object:
    """Translate legacy wire inputs into flags, never alternate identities."""
    if value == "TEST":
        return "DEV"
    if value == "REAL":
        return "LIVE"
    return value


MatterMode = Annotated[CanonicalMatterMode, BeforeValidator(_normalize_mode)]
