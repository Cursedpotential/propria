"""Verify neutral deployment scope against the engine's bounded identity receipt.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Byline amendment: Codex · GPT-6.1-Sol · 2026-10-06 — bounded private scope transport.
"""

import asyncio
import json
from typing import Any
from urllib.parse import urlsplit

import httpx

from app.config import settings
from app.service.matter_mode import (
    MatterModeError,
    configured_court_case_id,
    configured_matter_id,
)
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode

_TIMEOUT_SECONDS = 5.0
_MAX_RESPONSE_BYTES = 65536
_CONFIG_ERROR = "Proffer case verification is not configured or is invalid"
_UPSTREAM_ERROR = "Proffer authoritative case verification is unavailable"
_HEADER_ERROR = "Proffer authoritative case header is invalid"


def _scope_url() -> str:
    """Validate the private base URL before reading credentials or issuing a GET.

    Inputs: configured starter URL. Output: scope endpoint or safe 503 error.
    Effects: none. Pick only for bounded approval, not general Proffer requests.
    """
    raw = settings.proffer_starter_url
    try:
        parts = urlsplit(raw)
        valid = (
            raw and all(32 < ord(char) < 127 for char in raw)
            and parts.scheme in {"http", "https"} and parts.hostname
            and parts.username is None and parts.password is None
            and not parts.query and not parts.fragment
            and "?" not in raw and "#" not in raw and "\\" not in raw
            and "%" not in parts.netloc
            and (parts.port is None or 0 < parts.port <= 65535)
        )
    except ValueError:
        valid = False
    if not valid:
        raise ProfferError(_CONFIG_ERROR, 503)
    return raw.rstrip("/") + "/case-identity/scope"


def _unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    """Decode one unambiguous JSON object without permitting duplicate fields.

    Input: decoder key/value pairs. Output: dictionary or ValueError.
    Effects: none. Pick for authoritative scope JSON, not permissive UI payloads.
    """
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate field")
        result[key] = value
    return result


async def _read_scope(mode: MatterMode) -> dict[str, Any]:
    """Fetch a fresh bounded scope receipt without proxies, redirects or retries.

    Input: canonical policy after local config checks. Output: JSON scope or safe
    502/503 error. Effects: mounted-token read and one streamed authenticated GET,
    limited to 64 KiB and five seconds total. Pick over generic _request only for
    approval; full Case content and upload transport retain their own contracts.
    """
    from app.service.proffer import _service_authorization_headers

    url = _scope_url()
    headers = {**_service_authorization_headers(), "Accept": "application/json", "Accept-Encoding": "identity"}
    raw = bytearray()
    try:
        async with (
            asyncio.timeout(_TIMEOUT_SECONDS),
            httpx.AsyncClient(timeout=_TIMEOUT_SECONDS, trust_env=False, follow_redirects=False) as client,
            client.stream("GET", url, params={"mode": mode}, headers=headers) as response,
        ):
            if response.status_code != 200:
                raise ProfferError(_UPSTREAM_ERROR, 503)
            if response.headers.get("content-encoding", "identity").lower() != "identity":
                raise ProfferError(_HEADER_ERROR, 502)
            length = response.headers.get("content-length")
            if length is not None and (not length.isdecimal() or int(length) > _MAX_RESPONSE_BYTES):
                raise ProfferError(_HEADER_ERROR, 502)
            async for chunk in response.aiter_raw():
                if len(raw) + len(chunk) > _MAX_RESPONSE_BYTES:
                    raise ProfferError(_HEADER_ERROR, 502)
                raw.extend(chunk)
    except (TimeoutError, httpx.HTTPError, OSError, ValueError):
        raise ProfferError(_UPSTREAM_ERROR, 503) from None
    try:
        header = json.loads(raw, object_pairs_hook=_unique_object)
    except (UnicodeDecodeError, ValueError, RecursionError):
        raise ProfferError(_HEADER_ERROR, 502) from None
    if not isinstance(header, dict):
        raise ProfferError(_HEADER_ERROR, 502)
    return header


async def verify_case_scope(mode: MatterMode, header: dict[str, Any] | None = None) -> None:
    """Compare the configured pair with the single approved engine case.

    Inputs: canonical mode and optional already-fetched header. Outputs: None or
    ProfferError (503 for bad config, 502 for a wrong/unverifiable case).
    Side effects: one bounded authenticated GET when header is absent; no writes,
    catalog reads or caching. Pick for admission, not preview-mode recovery.
    """
    try:
        matter_id = str(configured_matter_id(mode))
        court_case_id = str(configured_court_case_id(mode))
    except MatterModeError as error:
        raise ProfferError(error.detail, error.status_code) from None
    if header is None:
        header = await _read_scope(mode)
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
