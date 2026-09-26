"""Read-only Evidence Platform client. Agno is truth; never clone evidence.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any

import httpx

from legal_workspace.config import get_settings

_HEX64 = re.compile(r"^[0-9a-f]{64}$")

HEALTH_TIMEOUT = 2.0
CONNECT_TIMEOUT = 0.4
_MATTER_ID_KEYS = ("id", "matter_id")
_MATTER_NAME_KEYS = ("title", "display_name")


@dataclass(frozen=True)
class HealthProbe:
    reachable: bool
    evidence_platform: str
    status: str | None = None


@dataclass(frozen=True)
class MatterProjection:
    """Identity only. Evidence bytes, spans, and hashes stay on Agno."""

    matter_id: str
    display_name: str


@dataclass(frozen=True)
class MattersList:
    ok: bool
    matters: tuple[MatterProjection, ...]
    reason: str | None = None


@dataclass(frozen=True)
class AgnoStatus:
    reachable: bool
    evidence_platform: str
    matters_visible: bool


@dataclass(frozen=True)
class HashVerify:
    """Identity-only custody verdict. Never clones bytes or chain payloads."""

    digest: str
    ok: bool
    reachable: bool
    verdict: str | None = None
    reason: str | None = None


def normalize_sha256(value: str) -> str | None:
    """Strip an optional sha256: prefix. Only 64 hex chars are Agno-callable."""
    raw = value.strip()
    if raw.lower().startswith("sha256:"):
        raw = raw.split(":", 1)[1]
    digest = raw.strip().lower()
    return digest if _HEX64.fullmatch(digest) else None


def _service_name() -> str:
    return get_settings().evidence_platform_service


def _base_url() -> str:
    return get_settings().evidence_platform_base_url.rstrip("/")


def _client(existing: httpx.Client | None) -> tuple[httpx.Client, bool]:
    if existing is not None:
        return existing, False

    settings = get_settings()
    headers = {}

    # Add Context Forge gateway token for authentication when making requests to Agno
    if settings.contextforge_gateway_token:
        headers["Authorization"] = f"Bearer {settings.contextforge_gateway_token}"

    client = httpx.Client(
        timeout=httpx.Timeout(HEALTH_TIMEOUT, connect=CONNECT_TIMEOUT),
        headers=headers
    )
    return client, True


def _project_matter(row: object) -> MatterProjection | None:
    if not isinstance(row, dict):
        return None
    matter_id = next((row[key] for key in _MATTER_ID_KEYS if row.get(key) is not None), None)
    name = next((row[key] for key in _MATTER_NAME_KEYS if row.get(key) is not None), None)
    if matter_id is None or name is None:
        return None
    return MatterProjection(matter_id=str(matter_id), display_name=str(name))


def _extract_matters(body: object) -> tuple[MatterProjection, ...] | None:
    if isinstance(body, list):
        rows: list[Any] = body
    elif isinstance(body, dict):
        data = body.get("data")
        if not isinstance(data, list):
            data = body.get("matters")
        if not isinstance(data, list):
            return None
        rows = data
    else:
        return None
    projected: list[MatterProjection] = []
    for row in rows:
        item = _project_matter(row)
        if item is None:
            return None
        projected.append(item)
    return tuple(projected)


def probe_health(client: httpx.Client | None = None) -> HealthProbe:
    """GET {evidence_platform_base_url}/health. 2s timeout. Fail closed."""
    url = f"{_base_url()}/health"
    http, closer = _client(client)
    try:
        response = http.get(url, timeout=HEALTH_TIMEOUT)
        response.raise_for_status()
        body = response.json() if response.content else {}
        status = body.get("status") if isinstance(body, dict) else None
        return HealthProbe(reachable=True, evidence_platform=_service_name(), status=status)
    except Exception:
        return HealthProbe(reachable=False, evidence_platform=_service_name(), status=None)
    finally:
        if closer:
            http.close()


def list_matters(client: httpx.Client | None = None) -> MattersList:
    """GET /v1/matters. Fail closed. Identity projection only — do not clone evidence."""
    url = f"{_base_url()}/v1/matters"
    http, closer = _client(client)
    try:
        response = http.get(url, timeout=HEALTH_TIMEOUT)
        response.raise_for_status()
        projected = _extract_matters(response.json())
        if projected is None:
            return MattersList(ok=False, matters=(), reason="unexpected-shape")
        return MattersList(ok=True, matters=projected)
    except Exception as exc:
        return MattersList(ok=False, matters=(), reason=exc.__class__.__name__)
    finally:
        if closer:
            http.close()


def read_status(client: httpx.Client | None = None) -> AgnoStatus:
    http, closer = _client(client)
    try:
        health = probe_health(client=http)
        matters = list_matters(client=http)
        return AgnoStatus(
            reachable=health.reachable,
            evidence_platform=_service_name(),
            matters_visible=matters.ok,
        )
    finally:
        if closer:
            http.close()


def verify_sha256(digest: str, client: httpx.Client | None = None) -> HashVerify:
    """POST Agno /v1/verify/{sha256}. Fail closed. Do not store evidence bytes."""
    normalized = normalize_sha256(digest)
    if normalized is None:
        return HashVerify(
            digest=digest,
            ok=False,
            reachable=False,
            reason="not-a-sha256-hex",
        )
    url = f"{_base_url()}/v1/verify/{normalized}"
    http, closer = _client(client)
    try:
        response = http.post(url, timeout=HEALTH_TIMEOUT)
        if response.status_code == 404:
            return HashVerify(
                digest=normalized,
                ok=False,
                reachable=True,
                reason="no-custody-record",
            )
        response.raise_for_status()
        body = response.json() if response.content else {}
        verdict = body.get("verdict") if isinstance(body, dict) else None
        match = body.get("sha256_match") if isinstance(body, dict) else None
        ok = verdict in {"intact", "hash-only-ok"} or match is True
        return HashVerify(
            digest=normalized,
            ok=bool(ok),
            reachable=True,
            verdict=str(verdict) if verdict is not None else None,
            reason=None if ok else (str(verdict) if verdict else "verify-rejected"),
        )
    except Exception as exc:
        return HashVerify(
            digest=normalized,
            ok=False,
            reachable=False,
            reason=exc.__class__.__name__,
        )
    finally:
        if closer:
            http.close()


def verify_package_hashes(
    hashes: list[str],
    client: httpx.Client | None = None,
) -> tuple[HashVerify, ...]:
    http, closer = _client(client)
    try:
        return tuple(verify_sha256(item, client=http) for item in hashes)
    finally:
        if closer:
            http.close()
