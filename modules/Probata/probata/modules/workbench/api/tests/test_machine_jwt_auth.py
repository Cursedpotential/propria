"""Machine-client Bearer JWTs (Authentik service accounts) at the Workbench boundary.

Byline: Claude Code · Opus 5.5 · 2026-09-28 (DF-30)
"""

from __future__ import annotations

import json
import time
from unittest.mock import patch

import jwt
import pytest
from app.config import Settings
from app.runtime import auth, machine_jwt
from cryptography.hazmat.primitives.asymmetric import rsa
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from starlette.middleware.base import BaseHTTPMiddleware

ISSUER = "https://authentik.example.ts.net/application/o/workbench-api/"
AUDIENCE = "propria-workbench-api"
JWKS_URL = "http://authentik-server:9000/application/o/workbench-api/jwks/"
SERVE_PEER = "100.72.169.40"
TRAEFIK_PEER = "10.201.8.1"
KID = "test-kid"

_KEY = rsa.generate_private_key(public_exponent=65537, key_size=2048)
_OTHER_KEY = rsa.generate_private_key(public_exponent=65537, key_size=2048)


def _jwks() -> dict:
    jwk = json.loads(jwt.algorithms.RSAAlgorithm.to_jwk(_KEY.public_key()))
    jwk.update({"kid": KID, "alg": "RS256", "use": "sig"})
    return {"keys": [jwk]}


def _token(key=_KEY, **overrides) -> str:
    now = int(time.time())
    claims = {
        "iss": ISSUER,
        "aud": AUDIENCE,
        "sub": "hashed-subject-1",
        "iat": now,
        "exp": now + 600,
        "preferred_username": "devbox",
        "groups": ["propria-machines"],
    }
    claims.update(overrides)
    claims = {k: v for k, v in claims.items() if v is not None}
    return jwt.encode(claims, key, algorithm="RS256", headers={"kid": KID})


@pytest.fixture(autouse=True)
def _published_jwks():
    machine_jwt._jwks_client.cache_clear()
    with patch.object(jwt.PyJWKClient, "fetch_data", return_value=_jwks()):
        yield
    machine_jwt._jwks_client.cache_clear()


def _client(host: str, *, enabled: bool = True, tailnet_bypass: bool = False) -> TestClient:
    configured = Settings(_env_file=None)
    values = {
        "trusted_auth_proxy_cidrs": f"{TRAEFIK_PEER}/32",
        "trusted_tailscale_serve_proxy_cidrs": f"{SERVE_PEER}/32",
        "tailnet_auth_bypass_enabled": tailnet_bypass,
        "machine_jwt_issuers": f"{ISSUER}, https://auth.example.com/application/o/workbench-api/" if enabled else "",
        "machine_jwt_audience": AUDIENCE,
        "machine_jwt_jwks_url": JWKS_URL,
        "machine_jwt_allowed_groups": "propria-machines",
    }
    for name, value in values.items():
        object.__setattr__(configured, name, value)

    app = FastAPI()

    async def isolated(request, call_next):
        with patch("app.runtime.auth.settings", configured):
            return await auth.authentication_middleware(request, call_next)

    app.add_middleware(BaseHTTPMiddleware, dispatch=isolated)

    @app.get("/health")
    def health() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/principal")
    def principal(request: Request) -> dict[str, str]:
        return {"principal": request.state.principal, "subject_uid": request.state.subject_uid}

    return TestClient(app, client=(host, 50000))


def _bearer(token: str) -> dict[str, str]:
    return {"Authorization": f"Bearer {token}"}


@pytest.mark.parametrize("peer", [SERVE_PEER, TRAEFIK_PEER])
def test_valid_token_admits_machine_principal_on_both_doors(peer: str) -> None:
    response = _client(peer).get("/principal", headers=_bearer(_token()))
    assert response.status_code == 200
    assert response.json() == {
        "principal": "authentik-sa:devbox",
        "subject_uid": "authentik-sa:hashed-subject-1",
    }


def test_second_configured_issuer_is_accepted() -> None:
    token = _token(iss="https://auth.example.com/application/o/workbench-api/")
    assert _client(SERVE_PEER).get("/principal", headers=_bearer(token)).status_code == 200


@pytest.mark.parametrize(
    "token",
    [
        pytest.param(_token(exp=int(time.time()) - 120, iat=int(time.time()) - 900), id="expired"),
        pytest.param(_token(aud="some-other-client"), id="wrong-audience"),
        pytest.param(_token(iss="https://evil.example/application/o/workbench-api/"), id="wrong-issuer"),
        pytest.param(_token(key=_OTHER_KEY), id="bad-signature"),
        pytest.param(_token(groups=["authentik Admins"]), id="wrong-group"),
        pytest.param(_token(groups=None), id="no-groups-claim"),
        pytest.param(_token(exp=None), id="no-expiry"),
        pytest.param(_token(preferred_username="bad user\n"), id="bad-username"),
        pytest.param("not-a-jwt", id="garbage"),
    ],
)
def test_invalid_token_is_refused(token: str) -> None:
    response = _client(SERVE_PEER).get("/principal", headers=_bearer(token))
    assert response.status_code == 401
    assert response.json() == {"detail": "Invalid machine token"}


def test_hs256_token_signed_with_public_key_material_is_refused() -> None:
    forged = jwt.encode(
        {
            "iss": ISSUER,
            "aud": AUDIENCE,
            "sub": "x",
            "iat": int(time.time()),
            "exp": int(time.time()) + 60,
            "preferred_username": "devbox",
            "groups": ["propria-machines"],
        },
        "shared-secret",
        algorithm="HS256",
        headers={"kid": KID},
    )
    assert _client(SERVE_PEER).get("/principal", headers=_bearer(forged)).status_code == 401


def test_invalid_token_does_not_fall_back_to_authentik_headers() -> None:
    headers = {**_bearer(_token(aud="other")), "X-authentik-uid": "u", "X-authentik-username": "owner"}
    assert _client(TRAEFIK_PEER).get("/principal", headers=headers).status_code == 401


def test_missing_token_on_serve_door_is_refused() -> None:
    response = _client(SERVE_PEER).get("/principal")
    assert response.status_code == 403


def test_spoofed_authentik_headers_without_jwt_on_serve_door_are_refused() -> None:
    headers = {"X-authentik-uid": "u", "X-authentik-username": "devbox"}
    response = _client(SERVE_PEER).get("/principal", headers=headers)
    assert response.status_code == 403
    assert response.json() == {"detail": "Untrusted proxy"}


def test_valid_token_from_untrusted_peer_is_refused() -> None:
    response = _client("172.30.0.9").get("/principal", headers=_bearer(_token()))
    assert response.status_code == 403
    assert response.json() == {"detail": "Untrusted proxy"}


def test_disabled_configuration_ignores_bearer() -> None:
    response = _client(SERVE_PEER, enabled=False).get("/principal", headers=_bearer(_token()))
    assert response.status_code == 403


def test_existing_authentik_header_path_still_admits_browser_on_traefik_door() -> None:
    headers = {"X-authentik-uid": "uid-1", "X-authentik-username": "msalem85"}
    response = _client(TRAEFIK_PEER).get("/principal", headers=headers)
    assert response.status_code == 200
    assert response.json() == {"principal": "msalem85", "subject_uid": "uid-1"}


def test_existing_tailscale_login_path_still_admits_owner_on_serve_door() -> None:
    response = _client(SERVE_PEER, tailnet_bypass=True).get(
        "/principal", headers={"Tailscale-User-Login": "owner@example.com"}
    )
    assert response.status_code == 200
    assert response.json()["principal"] == "owner@example.com"


def test_non_bearer_authorization_scheme_is_ignored() -> None:
    response = _client(SERVE_PEER).get("/principal", headers={"Authorization": "Basic Zm9vOmJhcg=="})
    assert response.status_code == 403


def test_health_stays_public() -> None:
    assert _client("203.0.113.5").get("/health").status_code == 200
