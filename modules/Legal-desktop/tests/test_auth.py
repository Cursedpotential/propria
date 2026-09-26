"""Positive and negative tests for the advocatio API trust boundary.

Byline: Codex · GPT-5 · 2026-09-12
"""

from __future__ import annotations

import hashlib
import hmac
import json
import time
from datetime import UTC, datetime, timedelta
from types import SimpleNamespace
from uuid import uuid4

import jwt
import pytest
from cryptography.hazmat.primitives.asymmetric import rsa
from fastapi.testclient import TestClient
from legal_workspace.api import auth as auth_mod
from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services.workspace import Workspace
from test_review_release import _drafted

ISSUER = "https://auth.example.test/application/o/advocatio/"
AUDIENCE = "advocatio"
BFF_SECRET = "test-only-bff-signing-secret-at-least-32-bytes"


class _StaticJwksClient:
    def __init__(self, public_key: rsa.RSAPublicKey) -> None:
        self._public_key = public_key

    def get_signing_key_from_jwt(self, _token: str) -> SimpleNamespace:
        return SimpleNamespace(key=self._public_key)


@pytest.fixture
def oidc_keys() -> tuple[rsa.RSAPrivateKey, rsa.RSAPublicKey]:
    private = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    return private, private.public_key()


@pytest.fixture
def secure_auth(monkeypatch: pytest.MonkeyPatch, oidc_keys) -> rsa.RSAPrivateKey:
    private, public = oidc_keys
    monkeypatch.setenv("LEGAL_WORKSPACE_BYPASS_AUTH", "false")
    monkeypatch.setenv("LEGAL_TAILNET_OWNER_ACCESS", "true")
    monkeypatch.setenv("AUTHENTIK_ISSUER", ISSUER)
    monkeypatch.setenv("AUTHENTIK_AUDIENCE", AUDIENCE)
    monkeypatch.setenv("AUTHENTIK_JWKS_URL", f"{ISSUER}jwks/")
    monkeypatch.setenv("AUTHENTIK_ALLOWED_GROUPS", "advocatio-users")
    monkeypatch.setenv("LEGAL_BFF_SIGNING_SECRET", BFF_SECRET)

    monkeypatch.setattr(auth_mod, "_jwks_client", lambda _url: _StaticJwksClient(public))
    return private


def _token(
    private_key: rsa.RSAPrivateKey,
    *,
    audience: str = AUDIENCE,
    expires_delta: timedelta = timedelta(minutes=5),
    groups: list[str] | None = None,
) -> str:
    now = datetime.now(UTC)
    return jwt.encode(
        {
            "iss": ISSUER,
            "aud": audience,
            "sub": "owner-subject",
            "iat": now,
            "exp": now + expires_delta,
            "preferred_username": "owner",
            "email": "owner@example.test",
            "groups": groups if groups is not None else ["advocatio-users"],
        },
        private_key,
        algorithm="RS256",
        headers={"kid": "test-key"},
    )


def _signed_bff_headers(method: str, target: str, body: bytes = b"") -> dict[str, str]:
    timestamp = str(int(time.time()))
    nonce = "unit-test-nonce"
    message = "\n".join(
        (timestamp, nonce, method.upper(), target, hashlib.sha256(body).hexdigest())
    ).encode()
    signature = hmac.new(BFF_SECRET.encode(), message, hashlib.sha256).hexdigest()
    return {
        "x-legal-bff-timestamp": timestamp,
        "x-legal-bff-nonce": nonce,
        "x-legal-bff-signature": signature,
    }


def test_missing_auth_is_denied_when_not_on_tailnet(secure_auth) -> None:
    response = TestClient(app).get("/v1/auth/whoami")
    assert response.status_code == 401
    assert response.json() == {"detail": "Valid Authentik identity required"}


def test_valid_authentik_token_projects_minimal_identity(secure_auth) -> None:
    token = _token(secure_auth)
    response = TestClient(app).get(
        "/v1/auth/whoami",
        headers={"authorization": f"Bearer {token}"},
    )
    assert response.status_code == 200
    assert response.json() == {
        "subject": "owner-subject",
        "username": "owner",
        "email": "owner@example.test",
        "groups": ["advocatio-users"],
        "source": "authentik",
    }


@pytest.mark.parametrize(
    ("audience", "expires_delta", "groups"),
    [
        ("wrong-app", timedelta(minutes=5), ["advocatio-users"]),
        (AUDIENCE, timedelta(minutes=-5), ["advocatio-users"]),
        (AUDIENCE, timedelta(minutes=5), ["unrelated-users"]),
    ],
)
def test_wrong_audience_expired_or_unauthorized_group_is_denied(
    secure_auth,
    audience: str,
    expires_delta: timedelta,
    groups: list[str],
) -> None:
    token = _token(
        secure_auth,
        audience=audience,
        expires_delta=expires_delta,
        groups=groups,
    )
    response = TestClient(app).get(
        "/v1/auth/whoami",
        headers={"authorization": f"Bearer {token}"},
    )
    assert response.status_code == 401


def test_signed_bff_request_is_accepted_without_browser_bearer(secure_auth) -> None:
    response = TestClient(app).get(
        "/v1/auth/whoami",
        headers=_signed_bff_headers("GET", "/v1/auth/whoami"),
    )
    assert response.status_code == 200
    assert response.json()["source"] == "signed-bff"


def test_signed_bff_request_verifies_percent_encoded_colon_route(secure_auth) -> None:
    # The web bridge signs encodeURIComponent() segments, so ":" arrives as "%3A".
    target = "/v1/privilege%3Ascan"
    body = b'{"text": "synthetic"}'
    headers = _signed_bff_headers("POST", target, body)
    headers["content-type"] = "application/json"
    response = TestClient(app).post(target, headers=headers, content=body)
    assert response.status_code != 401


def test_mcp_gateway_token_lane(secure_auth, monkeypatch: pytest.MonkeyPatch) -> None:
    token = "gateway-" + "k" * 40
    monkeypatch.setenv("LEGAL_MCP_GATEWAY_TOKEN", token)
    accepted = TestClient(app).get("/v1/auth/whoami", headers={"authorization": f"Bearer {token}"})
    assert accepted.status_code == 200
    assert accepted.json()["source"] == "mcp-gateway"
    denied = TestClient(app).get("/v1/auth/whoami", headers={"authorization": "Bearer " + "x" * 48})
    assert denied.status_code == 401


def test_review_action_requires_human_even_when_service_is_authenticated(
    secure_auth, monkeypatch: pytest.MonkeyPatch, tmp_path
) -> None:
    monkeypatch.setattr(main_mod, "WORKSPACE", Workspace(tmp_path))
    body = {
        "section_id": str(uuid4()),
        "verdict": "approve",
        "rationale": "synthetic action-auth probe",
        "reviewer": "owner",
    }
    human = TestClient(app).post(
        "/v1/reviews",
        headers={"authorization": f"Bearer {_token(secure_auth)}"},
        json=body,
    )
    assert human.status_code == 404

    service_token = "gateway-" + "k" * 40
    monkeypatch.setenv("LEGAL_MCP_GATEWAY_TOKEN", service_token)
    service = TestClient(app).post(
        "/v1/reviews",
        headers={"authorization": f"Bearer {service_token}"},
        json=body,
    )
    assert service.status_code == 403
    assert "authenticated human" in service.json()["detail"]


def test_review_action_rejects_bff_and_human_without_review_group(
    secure_auth, monkeypatch: pytest.MonkeyPatch, tmp_path
) -> None:
    monkeypatch.setattr(main_mod, "WORKSPACE", Workspace(tmp_path))
    body = json.dumps(
        {
            "section_id": str(uuid4()),
            "verdict": "approve",
            "rationale": "synthetic action-auth probe",
            "reviewer": "owner",
        }
    ).encode()
    before = {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()}
    bff = TestClient(app).post(
        "/v1/reviews",
        headers={
            "content-type": "application/json",
            **_signed_bff_headers("POST", "/v1/reviews", body),
        },
        content=body,
    )
    assert bff.status_code == 403

    monkeypatch.setenv("AUTHENTIK_ALLOWED_GROUPS", "advocatio-users,readers")
    reader = TestClient(app).post(
        "/v1/reviews",
        headers={
            "authorization": f"Bearer {_token(secure_auth, groups=['readers'])}",
            "content-type": "application/json",
        },
        content=body,
    )
    assert reader.status_code == 403
    assert {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()} == before


def test_valid_human_review_uses_token_subject_not_forged_body(
    secure_auth, monkeypatch: pytest.MonkeyPatch, tmp_path
) -> None:
    workspace = Workspace(tmp_path)
    section = _drafted(workspace)
    monkeypatch.setattr(main_mod, "WORKSPACE", workspace)
    response = TestClient(app).post(
        "/v1/reviews",
        headers={"authorization": f"Bearer {_token(secure_auth)}"},
        json={
            "section_id": str(section.section_id),
            "verdict": "approve",
            "rationale": "synthetic reviewed citation",
            "reviewer": "forged-body-value",
        },
    )
    assert response.status_code == 200, response.text
    assert response.json()["reviewer"] == "authentik:owner-subject"
    assert Workspace(tmp_path).load().reviews[-1].reviewer == "authentik:owner-subject"


def test_short_mcp_gateway_token_leaves_the_lane_off(secure_auth, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEGAL_MCP_GATEWAY_TOKEN", "short")
    response = TestClient(app).get("/v1/auth/whoami", headers={"authorization": "Bearer short"})
    assert response.status_code == 401


def test_invalid_bff_signature_is_denied(secure_auth) -> None:
    headers = _signed_bff_headers("GET", "/v1/auth/whoami")
    headers["x-legal-bff-signature"] = "0" * 64
    response = TestClient(app).get("/v1/auth/whoami", headers=headers)
    assert response.status_code == 401


def test_direct_tailnet_owner_access_remains_unfettered(secure_auth) -> None:
    client = TestClient(app, client=("100.100.10.20", 41200))
    response = client.get("/v1/auth/whoami")
    assert response.status_code == 200
    assert response.json()["source"] == "tailnet"


def test_tailnet_peer_with_authentik_jwt_keeps_verified_human_identity(
    secure_auth, monkeypatch: pytest.MonkeyPatch, tmp_path
) -> None:
    workspace = Workspace(tmp_path)
    section = _drafted(workspace)
    monkeypatch.setattr(main_mod, "WORKSPACE", workspace)
    client = TestClient(app, client=("100.100.10.20", 41200))
    bearer = {"authorization": f"Bearer {_token(secure_auth)}"}
    whoami = client.get("/v1/auth/whoami", headers=bearer)
    assert whoami.status_code == 200
    assert whoami.json()["source"] == "authentik"
    review = client.post(
        "/v1/reviews",
        headers=bearer,
        json={
            "section_id": str(section.section_id),
            "verdict": "approve",
            "rationale": "verified human through the BFF bearer path",
            "reviewer": "forged-owner",
        },
    )
    assert review.status_code == 200, review.text
    assert review.json()["reviewer"] == "authentik:owner-subject"


def test_tailnet_peer_cannot_downgrade_bad_jwt_or_claim_human_with_headers(
    secure_auth, monkeypatch: pytest.MonkeyPatch, tmp_path
) -> None:
    workspace = Workspace(tmp_path)
    section = _drafted(workspace)
    monkeypatch.setattr(main_mod, "WORKSPACE", workspace)
    body = {
        "section_id": str(section.section_id),
        "verdict": "approve",
        "rationale": "forged identity probe",
        "reviewer": "owner",
    }
    client = TestClient(app, client=("100.100.10.20", 41200))
    before = {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()}
    forged = {"tailscale-user-login": "owner@example.test"}
    invalid_jwt = client.post(
        "/v1/reviews", headers={**forged, "authorization": "Bearer bad-jwt"}, json=body
    )
    assert invalid_jwt.status_code == 401
    device_only = client.post("/v1/reviews", headers=forged, json=body)
    assert device_only.status_code == 403
    assert "Tailnet human attestation required" in device_only.json()["detail"]
    assert {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()} == before


def test_signed_bff_and_service_credentials_cannot_use_forged_tailnet_identity(
    secure_auth, monkeypatch: pytest.MonkeyPatch, tmp_path
) -> None:
    workspace = Workspace(tmp_path)
    section = _drafted(workspace)
    monkeypatch.setattr(main_mod, "WORKSPACE", workspace)
    body = json.dumps({
        "section_id": str(section.section_id),
        "verdict": "approve",
        "rationale": "service and BFF identity probe",
        "reviewer": "owner",
    }).encode()
    before = {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()}
    forged = {"tailscale-user-login": "owner@example.test", "content-type": "application/json"}
    bff = TestClient(app).post(
        "/v1/reviews",
        headers={**forged, **_signed_bff_headers("POST", "/v1/reviews", body)},
        content=body,
    )
    assert bff.status_code == 403
    service_token = "gateway-" + "k" * 40
    monkeypatch.setenv("LEGAL_MCP_GATEWAY_TOKEN", service_token)
    service = TestClient(app).post(
        "/v1/reviews",
        headers={**forged, "authorization": f"Bearer {service_token}"},
        content=body,
    )
    assert service.status_code == 403
    assert {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()} == before


def test_forwarded_tailnet_address_does_not_bypass_socket_check(secure_auth) -> None:
    response = TestClient(app).get(
        "/v1/auth/whoami",
        headers={"x-forwarded-for": "100.100.10.20"},
    )
    assert response.status_code == 401


def test_missing_oidc_configuration_fails_closed(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEGAL_WORKSPACE_BYPASS_AUTH", "false")
    monkeypatch.setenv("LEGAL_TAILNET_OWNER_ACCESS", "false")
    monkeypatch.delenv("AUTHENTIK_ISSUER", raising=False)
    monkeypatch.delenv("AUTHENTIK_AUDIENCE", raising=False)
    monkeypatch.delenv("AUTHENTIK_JWKS_URL", raising=False)
    monkeypatch.delenv("LEGAL_BFF_SIGNING_SECRET", raising=False)
    response = TestClient(app).get(
        "/v1/auth/whoami",
        headers={"authorization": "Bearer not-a-jwt"},
    )
    assert response.status_code == 503
