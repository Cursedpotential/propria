"""Shared error type for the Workbench Proffer boundary.

Byline amendment: Claude Code · Opus 5.5 · 2026-09-26 — optional `payload`: the upstream
JSON error object, so a route can relay more than `detail` (the repair run refusal's checks).
"""

from typing import Any

import httpx


class ProfferError(Exception):
    def __init__(self, detail: str, status_code: int = 502, *, payload: dict[str, Any] | None = None):
        self.detail = detail
        self.status_code = status_code
        self.payload = payload
        super().__init__(detail)


def upstream_payload(response: httpx.Response) -> dict[str, Any] | None:
    """The upstream JSON object, or None when the body is not one."""
    try:
        payload: Any = response.json()
    except ValueError:  # JSONDecodeError and UnicodeDecodeError are both ValueErrors
        return None
    return payload if isinstance(payload, dict) else None
