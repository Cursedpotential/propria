"""Verify a configured Live case through the bounded authoritative starter scope read.

Byline: Codex · GPT-5 · 2026-10-05
Updated: Codex · gpt-6.1-sol · 2026-10-06 — bounded scope read and court-parent correlation.

This import-light boundary owns no registry or cache. Every write needs a fresh
read-only approval; read and dry-run callers need no write authorization.
Credential contract: modules/workbench/api/app/service/proffer.py.
"""

from __future__ import annotations

import json
import os
import re
import stat
from http.client import HTTPException as HTTPClientError
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener
from uuid import UUID

MATTER_ENV = "PROFFER_MATTER_ID"
COURT_CASE_ENV = "PROFFER_COURT_CASE_ID"
_TIMEOUT_SECONDS = 5.0
_MAX_RESPONSE_BYTES = 65536
_MIN_TOKEN_BYTES = 32
_MAX_TOKEN_BYTES = 4096
_MAX_TOKEN_FILE_BYTES = _MAX_TOKEN_BYTES + 2
_SERVICE_TOKEN = re.compile(r"[A-Za-z0-9\-._~+/]+={0,}")
_CONFIG_ERROR = "Proffer case verification is not configured or is invalid"
_AUTH_ERROR = "Proffer service authentication is unavailable or invalid"
_UPSTREAM_ERROR = "Proffer authoritative case verification is unavailable"
_HEADER_ERROR = "Proffer authoritative case header is invalid or does not match"


class CaseScopeVerificationError(ValueError):
    """Expose only a safe diagnostic and HTTP status for failed write approval."""

    def __init__(self, message: str, http_status: int = 503) -> None:
        super().__init__(message)
        self.http_status = http_status


def case_id(value: str, name: str) -> str:
    """Return a non-nil, non-retired UUID without I/O or identity selection.

    Inputs: identifier and diagnostic field name. Outputs: canonical UUID or
    ValueError. Effects: none; use for local scope comparisons, never approval.
    """
    try:
        parsed = UUID(value.strip())
    except (AttributeError, TypeError, ValueError):
        raise ValueError(f"{name} must be a configured non-nil, non-retired UUID") from None
    normalized = str(parsed)
    if parsed.int == 0 or normalized.startswith(("deadbeef-", "cafebabe-")):
        raise ValueError(f"{name} must be a configured non-nil, non-retired UUID")
    return normalized


def configured_case_scope(*, allow_empty: bool = False) -> tuple[str, str]:
    """Read the neutral identity pair without claiming it is approved.

    Inputs: allow_empty for an entirely unconfigured read-only starter. Outputs:
    canonical pair or ValueError. Effects: environment reads only; writes must
    additionally call require_authoritative_live_case_scope.
    """
    matter = os.environ.get(MATTER_ENV, "").strip()
    court = os.environ.get(COURT_CASE_ENV, "").strip()
    if allow_empty and not matter and not court:
        return "", ""
    return case_id(matter, MATTER_ENV), case_id(court, COURT_CASE_ENV)


def _header_url() -> str:
    """Validate a private starter base URL before any credential or network I/O."""
    raw = os.environ.get("PROFFER_STARTER_URL", "")
    try:
        parts = urlsplit(raw)
        valid = (
            raw
            and all(32 < ord(char) < 127 for char in raw)
            and parts.scheme in ("http", "https")
            and parts.hostname
            and parts.username is None
            and parts.password is None
            and not parts.query
            and not parts.fragment
            and "?" not in raw
            and "#" not in raw
            and "\\" not in raw
            and "%" not in parts.netloc
            and (parts.port is None or 0 < parts.port <= 65535)
        )
    except ValueError:
        valid = False
    if not valid:
        raise CaseScopeVerificationError(_CONFIG_ERROR)
    return raw.rstrip("/") + "/case-identity/scope?mode=LIVE"


def _service_authorization_headers() -> dict[str, str]:
    """Read the mounted private bearer using the existing Workbench token contract."""
    path = Path(os.environ.get("PROFFER_SERVICE_TOKEN_FILE", ""))
    if not path.is_absolute():
        raise CaseScopeVerificationError(_AUTH_ERROR)
    try:
        file_info = path.lstat()
        if not stat.S_ISREG(file_info.st_mode) or not (_MIN_TOKEN_BYTES <= file_info.st_size <= _MAX_TOKEN_FILE_BYTES):
            raise CaseScopeVerificationError(_AUTH_ERROR)
        with path.open("rb") as secret_file:
            raw = secret_file.read(_MAX_TOKEN_FILE_BYTES + 1)
    except (OSError, ValueError):
        raise CaseScopeVerificationError(_AUTH_ERROR) from None
    raw = raw.rstrip(b"\r\n")
    if not _MIN_TOKEN_BYTES <= len(raw) <= _MAX_TOKEN_BYTES or b"\x00" in raw:
        raise CaseScopeVerificationError(_AUTH_ERROR)
    try:
        token = raw.decode("utf-8")
    except UnicodeDecodeError:
        raise CaseScopeVerificationError(_AUTH_ERROR) from None
    if _SERVICE_TOKEN.fullmatch(token) is None:
        raise CaseScopeVerificationError(_AUTH_ERROR)
    return {"Authorization": f"Bearer {token}"}


class _NoRedirect(HTTPRedirectHandler):
    """Reject redirects so the service bearer never reaches a redirect target."""

    def redirect_request(self, req: Any, fp: Any, code: int, msg: str, headers: Any, newurl: str) -> None:
        """Decline every redirect without issuing another request."""
        return None


def _unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    """Reject ambiguous duplicate JSON fields in the authoritative response."""
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate field")
        result[key] = value
    return result


def _read_authoritative_header(url: str) -> dict[str, Any]:
    """Read one bounded authenticated JSON header without redirects or proxy leakage.

    Inputs: validated header URL. Outputs: JSON object or safe verification error.
    Effects: one GET, credential-file read; no logging, cache, writes or retries.
    """
    request = Request(url, headers={**_service_authorization_headers(), "Accept": "application/json"}, method="GET")
    # Environment proxies are not part of this private-service credential boundary.
    opener = build_opener(ProxyHandler({}), _NoRedirect())
    try:
        with opener.open(request, timeout=_TIMEOUT_SECONDS) as response:
            if response.status != 200:
                raise CaseScopeVerificationError(_UPSTREAM_ERROR)
            raw = response.read(_MAX_RESPONSE_BYTES + 1)
    except HTTPError as error:
        error.close()
        raise CaseScopeVerificationError(_UPSTREAM_ERROR) from None
    except (URLError, OSError, ValueError, HTTPClientError):
        raise CaseScopeVerificationError(_UPSTREAM_ERROR) from None
    if len(raw) > _MAX_RESPONSE_BYTES:
        raise CaseScopeVerificationError(_HEADER_ERROR, 502)
    try:
        header = json.loads(raw, object_pairs_hook=_unique_object)
    except (UnicodeDecodeError, ValueError, RecursionError):
        raise CaseScopeVerificationError(_HEADER_ERROR, 502) from None
    if not isinstance(header, dict):
        raise CaseScopeVerificationError(_HEADER_ERROR, 502)
    return header


def require_authoritative_live_case_scope(operating_mode: str, matter_id: str, court_case_id: str) -> tuple[str, str]:
    """Authorize an explicit Live write with a fresh matching starter header.

    Inputs: explicit canonical mode and requested matter/court pair. Outputs:
    canonical approved pair, or safe CaseScopeVerificationError. Effects: one
    authenticated scope GET only after local mode/config/scope checks; no DB or
    cache. The returned court must belong to the same approved matter.
    Pick this before mutation-body imports, not for reads or historical defaulting.
    """
    try:
        if operating_mode != "LIVE":
            raise ValueError("an explicit LIVE operating_mode is required; DEV canonical writes are unavailable")
        configured = configured_case_scope()
        requested = case_id(matter_id, "matter_id"), case_id(court_case_id, "court_case_id")
        if requested != configured:
            raise ValueError("mutation scope does not match the configured case")
    except ValueError as error:
        raise CaseScopeVerificationError(str(error)) from None
    header = _read_authoritative_header(_header_url())
    try:
        returned = case_id(header["matter"]["id"], "matter_id"), case_id(header["court_case"]["id"], "court_case_id")
        court_matter_id = case_id(header["court_case"]["matter_id"], "court_case.matter_id")
        if header.get("mode") != "LIVE" or returned != configured or court_matter_id != configured[0]:
            raise ValueError("foreign header")
    except (KeyError, TypeError, ValueError):
        raise CaseScopeVerificationError(_HEADER_ERROR, 502) from None
    return configured
