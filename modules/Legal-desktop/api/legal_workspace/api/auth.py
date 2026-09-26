"""Authentication adapters for the private advocatio API.

The public human UI is protected by Authentik at Traefik. The Next.js BFF
forwards Authentik's signed identity JWT; this module validates its signature,
issuer, audience and lifetime before any legal-domain route runs. Direct
requests observed on the tailnet retain device-level access. Neither a tailnet
address nor the BFF request signature establishes a human review actor.

Byline: Codex · GPT-5 · 2026-09-12
Auth ordering amendment: Codex · GPT-6 · 2026-09-23
"""

from __future__ import annotations

import hashlib
import hmac
import ipaddress
import re
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from functools import lru_cache
from typing import Any

import jwt
from fastapi import Request, Response
from fastapi.responses import JSONResponse
from jwt import InvalidTokenError, PyJWKClient
from starlette.middleware.base import BaseHTTPMiddleware

from legal_workspace.config import Settings, get_settings

_TAILNET = ipaddress.ip_network("100.64.0.0/10")
_BFF_MAX_AGE_SECONDS = 30


@dataclass(frozen=True)
class AuthenticatedPrincipal:
    """Minimal identity made available to request handlers and audit adapters."""

    subject: str
    username: str | None
    email: str | None
    groups: tuple[str, ...]
    source: str


class AuthenticationUnavailable(RuntimeError):
    """Raised when production authentication has not been configured."""


class AuthenticationDenied(ValueError):
    """Raised when supplied credentials do not satisfy the auth contract."""


class PrincipalAuthorizationDenied(PermissionError):
    """Raised when an authenticated identity is ineligible for an action."""


def require_human_review_actor(
    principal: AuthenticatedPrincipal,
    *,
    settings: Settings | None = None,
) -> str:
    """Return a stable audit actor only for an eligible verified human.

    Gateway, BFF, network/device, bypass, and agent identities authenticate a
    transport or service. They do not prove that a human adopted legal work.
    """

    current = settings or get_settings()
    eligible_groups = {
        item.strip()
        for item in current.authentik_review_groups.split(",")
        if item.strip()
    }
    if principal.source == "tailnet":
        raise PrincipalAuthorizationDenied(
            "Tailnet human attestation required for review; contact the ingress owner"
        )
    if principal.source != "authentik":
        raise PrincipalAuthorizationDenied(
            "an eligible authenticated human must record the review verdict"
        )
    if not eligible_groups or not eligible_groups.intersection(principal.groups):
        raise PrincipalAuthorizationDenied(
            "authenticated human is not in an eligible review group"
        )
    subject = principal.subject.strip()
    if not subject:
        raise PrincipalAuthorizationDenied("authenticated human has no stable subject")
    return f"authentik:{subject}"


class AuthentikTokenVerifier:
    """Verify Authentik JWTs against the provider's pinned issuer contract."""

    def __init__(self, settings: Settings, *, jwks_client: PyJWKClient | None = None) -> None:
        if not settings.authentik_issuer:
            raise AuthenticationUnavailable("AUTHENTIK_ISSUER is required")
        if not settings.authentik_audience:
            raise AuthenticationUnavailable("AUTHENTIK_AUDIENCE is required")
        if not settings.authentik_jwks_url:
            raise AuthenticationUnavailable("AUTHENTIK_JWKS_URL is required")
        self._settings = settings
        self._jwks = jwks_client or _jwks_client(settings.authentik_jwks_url)

    def verify(self, token: str) -> AuthenticatedPrincipal:
        try:
            header = jwt.get_unverified_header(token)
            if header.get("alg") != "RS256":
                raise AuthenticationDenied("only Authentik RS256 tokens are accepted")
            signing_key = self._jwks.get_signing_key_from_jwt(token)
            claims = jwt.decode(
                token,
                signing_key.key,
                algorithms=["RS256"],
                audience=self._settings.authentik_audience,
                issuer=self._settings.authentik_issuer,
                leeway=self._settings.auth_clock_skew_seconds,
                options={
                    "require": ["aud", "exp", "iat", "iss", "sub"],
                    "verify_signature": True,
                    "verify_aud": True,
                    "verify_exp": True,
                    "verify_iat": True,
                    "verify_iss": True,
                },
            )
        except AuthenticationDenied:
            raise
        except (InvalidTokenError, ValueError, TypeError) as exc:
            raise AuthenticationDenied("invalid Authentik identity token") from exc

        groups = _claim_values(claims.get("groups"))
        allowed = {
            item.strip()
            for item in self._settings.authentik_allowed_groups.split(",")
            if item.strip()
        }
        if allowed and not allowed.intersection(groups):
            raise AuthenticationDenied("user is not in an allowed Authentik group")

        subject = str(claims.get("sub") or "").strip()
        if not subject:
            raise AuthenticationDenied("Authentik token has no subject")
        return AuthenticatedPrincipal(
            subject=subject,
            username=_optional_claim(claims, "preferred_username"),
            email=_optional_claim(claims, "email"),
            groups=tuple(groups),
            source="authentik",
        )


@lru_cache(maxsize=8)
def _jwks_client(url: str) -> PyJWKClient:
    """Reuse Authentik's bounded JWKS cache across requests."""

    return PyJWKClient(url, cache_keys=True, lifespan=300)


def _optional_claim(claims: dict[str, Any], name: str) -> str | None:
    value = claims.get(name)
    return str(value) if value is not None else None


def _claim_values(value: object) -> list[str]:
    if isinstance(value, str):
        return [item for item in value.split("|") if item]
    if isinstance(value, list):
        return [str(item) for item in value if str(item)]
    return []


def _direct_tailnet_client(request: Request) -> bool:
    """Use the socket peer only; forwarded headers are attacker-controlled here."""

    if request.client is None:
        return False
    try:
        return ipaddress.ip_address(request.client.host) in _TAILNET
    except ValueError:
        return False


def _bearer_token(authorization: str | None) -> str | None:
    if not authorization:
        return None
    scheme, separator, credentials = authorization.partition(" ")
    if not separator or scheme.lower() != "bearer" or not credentials.strip():
        return None
    return credentials.strip()


def _is_mcp_gateway_token(token: str, settings: Settings) -> bool:
    """Service credential ContextForge presents to the /mcp face. Off when unset."""
    secret = settings.mcp_gateway_token
    if len(secret.encode()) < 32:
        return False
    return hmac.compare_digest(token.encode(), secret.encode())


def _canonical_bff_message(request: Request, timestamp: str, nonce: str, body: bytes) -> bytes:
    # Sign the path as sent on the wire: the web bridge percent-encodes each
    # segment (":" becomes "%3A"), and request.url.path is already decoded, so
    # every colon route (/v1/documents:convert, /v1/bates:stamp) failed to verify.
    raw_path = request.scope.get("raw_path")
    target = raw_path.decode("ascii") if raw_path else request.url.path
    if request.url.query:
        target = f"{target}?{request.url.query}"
    body_hash = hashlib.sha256(body).hexdigest()
    return f"{timestamp}\n{nonce}\n{request.method.upper()}\n{target}\n{body_hash}".encode()


async def _verify_bff_signature(request: Request, settings: Settings) -> bool:
    timestamp = request.headers.get("x-legal-bff-timestamp", "")
    nonce = request.headers.get("x-legal-bff-nonce", "")
    supplied = request.headers.get("x-legal-bff-signature", "")
    secret = settings.legal_bff_signing_secret
    if not timestamp or not nonce or not supplied:
        return False
    if len(secret.encode()) < 32:
        raise AuthenticationUnavailable("LEGAL_BFF_SIGNING_SECRET must be at least 32 bytes")
    try:
        issued_at = int(timestamp)
    except ValueError:
        return False
    if abs(int(time.time()) - issued_at) > _BFF_MAX_AGE_SECONDS:
        return False
    body = await request.body()
    expected = hmac.new(
        secret.encode(),
        _canonical_bff_message(request, timestamp, nonce, body),
        hashlib.sha256,
    ).hexdigest()
    return hmac.compare_digest(supplied, expected)


class LegalWorkspaceAuthMiddleware(BaseHTTPMiddleware):
    """Apply the two-lane ingress contract at the private API boundary."""

    async def dispatch(
        self,
        request: Request,
        call_next: Callable[[Request], Awaitable[Response]],
    ) -> Response:
        if request.url.path == "/health":
            return await call_next(request)

        # Collabora callbacks carry a document-scoped capability, not an Authentik JWT.
        # Validate before all transport/test bypasses, only on the exact WOPI surface.
        wopi = re.fullmatch(r"/wopi/files/([^/]+)(?:/contents)?", request.url.path)
        if wopi:
            from legal_workspace.api.office_routes import editor, token_for
            from legal_workspace.services.office_editor import OfficeError

            try:
                session = editor().authorize(wopi.group(1), token_for(request))
            except OfficeError:
                return JSONResponse(status_code=401, content={"detail": "Invalid editor session"},
                                    headers={"Cache-Control": "no-store"})
            request.state.auth = AuthenticatedPrincipal(
                subject=session["actor"], username=None, email=None,
                groups=(), source="office-session",
            )
            return await call_next(request)

        settings = get_settings()
        if settings.bypass_auth:
            request.state.auth = AuthenticatedPrincipal(
                subject="test-bypass",
                username=None,
                email=None,
                groups=(),
                source="explicit-test-bypass",
            )
            return await call_next(request)

        try:
            # A presented credential must be checked before any network fallback.
            # In particular, an invalid Authentik JWT from a tailnet peer cannot
            # become a device principal by changing the source ordering.
            authorization = request.headers.get("authorization")
            if authorization is not None:
                token = _bearer_token(authorization)
                if token is None:
                    raise AuthenticationDenied("invalid authorization header")
                if _is_mcp_gateway_token(token, settings):
                    request.state.auth = AuthenticatedPrincipal(
                        subject="contextforge-gateway",
                        username=None,
                        email=None,
                        groups=(),
                        source="mcp-gateway",
                    )
                else:
                    request.state.auth = AuthentikTokenVerifier(settings).verify(token)
                return await call_next(request)

            bff_headers = (
                "x-legal-bff-timestamp",
                "x-legal-bff-nonce",
                "x-legal-bff-signature",
            )
            if any(name in request.headers for name in bff_headers):
                if not await _verify_bff_signature(request, settings):
                    raise AuthenticationDenied("invalid BFF signature")
                request.state.auth = AuthenticatedPrincipal(
                    subject="legal-web-bff",
                    username=None,
                    email=None,
                    groups=(),
                    source="signed-bff",
                )
                return await call_next(request)

            if settings.tailnet_owner_access and _direct_tailnet_client(request):
                request.state.auth = AuthenticatedPrincipal(
                    subject="tailnet-device",
                    username=None,
                    email=None,
                    groups=(),
                    source="tailnet",
                )
                return await call_next(request)
            raise AuthenticationDenied("authentication required")
        except AuthenticationUnavailable as exc:
            return JSONResponse(status_code=503, content={"detail": str(exc)})
        except AuthenticationDenied:
            return JSONResponse(
                status_code=401,
                content={"detail": "Valid Authentik identity required"},
                headers={"WWW-Authenticate": 'Bearer realm="advocatio"'},
            )
