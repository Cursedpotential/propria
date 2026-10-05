"""Fence chunk mutations before importing any database, embedding, or vector-store client.

Byline: Codex · GPT-5 · 2026-10-05

The deployment owner provisions the neutral case pair from the authoritative engine.
This boundary validates that binding; it does not mint identities or implement a Dev workspace.
"""

from __future__ import annotations

import os
from uuid import UUID

from temporalio.exceptions import ApplicationError

MATTER_ENV = "PROFFER_MATTER_ID"
COURT_CASE_ENV = "PROFFER_COURT_CASE_ID"


def canonical_operating_mode(value: str) -> str:
    """Validate explicit mode and return DEV or LIVE without inventing a default.

    Inputs: an explicit wire mode; REAL/TEST are narrow historical aliases only.
    Outputs: canonical mode, or ValueError for missing/unknown values.
    Effects: none. Pick this over fresh-request defaulting for durable inputs.
    """
    mode = value.strip().upper() if isinstance(value, str) else ""
    mode = {"REAL": "LIVE", "TEST": "DEV"}.get(mode, mode)
    if mode not in ("DEV", "LIVE"):
        raise ValueError("an explicit DEV or LIVE operating_mode is required")
    return mode


def _case_id(value: str, name: str) -> str:
    """Validate one non-nil UUID without I/O and return its canonical text.

    Inputs: case identifier and diagnostic field name. Outputs: UUID text or ValueError.
    Effects: none. Use for scope comparison, not for selecting an operating mode.
    """
    try:
        parsed = UUID(value.strip())
    except (AttributeError, TypeError, ValueError) as error:
        raise ValueError(f"{name} must be a configured non-nil UUID") from error
    if parsed.int == 0:
        raise ValueError(f"{name} must be a configured non-nil UUID")
    return str(parsed)


def configured_case_scope(*, allow_empty: bool = False) -> tuple[str, str]:
    """Read the neutral approved pair without consulting mode-specific identity settings.

    Inputs: allow_empty permits an entirely unconfigured read-only starter.
    Outputs: canonical pair or ValueError for absent/partial/invalid configuration.
    Effects: environment reads only. Pick this over write admission for read-only inputs.
    """
    matter = os.environ.get(MATTER_ENV, "").strip()
    court = os.environ.get(COURT_CASE_ENV, "").strip()
    if allow_empty and not matter and not court:
        return "", ""
    return _case_id(matter, MATTER_ENV), _case_id(court, COURT_CASE_ENV)


def require_live_chunk_write(operating_mode: str, matter_id: str, court_case_id: str) -> None:
    """Admit only explicit Live chunk writes bound to the configured approved case.

    Inputs: durable mode and matter/court scope. Outputs: None on admission, otherwise
    non-retryable ApplicationError. Effects: environment reads only, no service calls.
    Pick this before mutation bodies; read-only activities need no write authorization.
    """
    try:
        if canonical_operating_mode(operating_mode) != "LIVE":
            raise ValueError("DEV canonical writes are unavailable until a disposable workspace exists")
        approved = configured_case_scope()
        requested = _case_id(matter_id, "matter_id"), _case_id(court_case_id, "court_case_id")
        if requested != approved:
            raise ValueError("chunk mutation scope does not match the configured approved case")
    except ValueError as error:
        raise ApplicationError(str(error), type="ChunkWriteDenied", non_retryable=True) from error
