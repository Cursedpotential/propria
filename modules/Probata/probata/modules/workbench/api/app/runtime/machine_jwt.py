"""Verify Authentik-issued access tokens presented by machine clients.

Byline: Claude Code · Opus 5.5 · 2026-09-28 (DF-30: Authentik service accounts for machines)

A machine client (the devbox first) authenticates to Authentik with the
client-credentials grant, as an Authentik service account holding an app
password, and presents the access token as ``Authorization: Bearer <JWT>``.
The Workbench accepts it only when every check passes: signature against
Authentik's published JWKS (cached), exact issuer, audience, expiry, and
membership of an allowlisted group. Anything else is ``None`` (fail closed).
The JWT path is off unless all four settings are configured.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from functools import lru_cache

import jwt
from app.config import Settings
from starlette.concurrency import run_in_threadpool

_ALGORITHMS = ["RS256", "ES256"]
_LEEWAY_SECONDS = 30
_MAX_TOKEN_LEN = 8192
_JWKS_CACHE_SECONDS = 300
_USERNAME_RE = re.compile(r"[A-Za-z0-9][A-Za-z0-9._@-]{0,149}")


@dataclass(frozen=True)
class MachinePrincipal:
    username: str
    subject: str


def _csv(value: str) -> list[str]:
    return [part.strip() for part in value.split(",") if part.strip()]


def machine_jwt_enabled(settings: Settings) -> bool:
    """True only when issuer, audience, JWKS URL and allowed groups are all set."""
    return bool(
        _csv(settings.machine_jwt_issuers)
        and settings.machine_jwt_audience.strip()
        and settings.machine_jwt_jwks_url.strip()
        and _csv(settings.machine_jwt_allowed_groups)
    )


@lru_cache(maxsize=4)
def _jwks_client(url: str) -> jwt.PyJWKClient:
    return jwt.PyJWKClient(url, cache_jwk_set=True, lifespan=_JWKS_CACHE_SECONDS, timeout=5)


def _verify(token: str, settings: Settings) -> MachinePrincipal | None:
    if not machine_jwt_enabled(settings) or not token or len(token) > _MAX_TOKEN_LEN:
        return None
    try:
        signing_key = _jwks_client(settings.machine_jwt_jwks_url.strip()).get_signing_key_from_jwt(token)
        claims = jwt.decode(
            token,
            signing_key.key,
            algorithms=_ALGORITHMS,
            audience=settings.machine_jwt_audience.strip(),
            issuer=_csv(settings.machine_jwt_issuers),
            leeway=_LEEWAY_SECONDS,
            options={"require": ["exp", "iat", "iss", "aud", "sub"]},
        )
    except (jwt.PyJWTError, ValueError, TypeError):
        return None
    groups = claims.get("groups")
    if not isinstance(groups, list) or not set(_csv(settings.machine_jwt_allowed_groups)) & {
        group for group in groups if isinstance(group, str)
    }:
        return None
    username = claims.get("preferred_username")
    subject = claims.get("sub")
    if not isinstance(username, str) or not _USERNAME_RE.fullmatch(username):
        return None
    if not isinstance(subject, str) or not subject or len(subject) > 256:
        return None
    return MachinePrincipal(username=username, subject=subject)


async def verify_machine_token(token: str, settings: Settings) -> MachinePrincipal | None:
    """Verify off the event loop: a JWKS refresh is a blocking HTTP call."""
    return await run_in_threadpool(_verify, token, settings)
