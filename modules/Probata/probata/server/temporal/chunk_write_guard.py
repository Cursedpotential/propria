"""Fence chunk mutations before importing any database, embedding, or vector-store client.

Byline: Codex · GPT-5 · 2026-10-05

The neutral pair is configuration, not approval. A fresh authenticated engine header
must approve every write; this boundary never mints identities or a Dev workspace.
"""

from __future__ import annotations

from temporalio.exceptions import ApplicationError

from server.case_management.authoritative_case_scope import (
    configured_case_scope,  # noqa: F401 -- existing read-only CLI compatibility surface
    require_authoritative_live_case_scope,
)


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


def require_live_chunk_write(operating_mode: str, matter_id: str, court_case_id: str) -> None:
    """Admit only explicit Live chunk writes bound to the configured approved case.

    Inputs: durable mode and matter/court scope. Outputs: None on admission, otherwise
    non-retryable ApplicationError. Effects: one authenticated read-only engine header
    verification, only after local admission. No DB, embedder or store imports.
    Pick this before mutation bodies; read-only activities need no write authorization.
    """
    try:
        if canonical_operating_mode(operating_mode) != "LIVE":
            raise ValueError("DEV canonical writes are unavailable until a disposable workspace exists")
        require_authoritative_live_case_scope("LIVE", matter_id, court_case_id)
    except ValueError as error:
        raise ApplicationError(str(error), type="ChunkWriteDenied", non_retryable=True) from error
