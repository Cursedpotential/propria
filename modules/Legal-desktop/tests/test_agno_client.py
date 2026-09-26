"""Agno client is read-only and fail-closed.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from uuid import uuid4

import httpx
from fastapi.testclient import TestClient

from legal_workspace.api.main import app
from legal_workspace.services.agno_client import (
    HEALTH_TIMEOUT,
    list_matters,
    normalize_sha256,
    probe_health,
    read_status,
    verify_package_hashes,
    verify_sha256,
)


def _client(handler) -> httpx.Client:
    return httpx.Client(transport=httpx.MockTransport(handler), timeout=HEALTH_TIMEOUT)


def test_probe_health_ok_uses_service_name_not_ip() -> None:
    seen: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        seen.append(str(request.url))
        assert request.method == "GET"
        assert request.url.path == "/health"
        return httpx.Response(200, json={"status": "ok"})

    result = probe_health(client=_client(handler))
    assert result.reachable is True
    assert result.status == "ok"
    assert result.evidence_platform == "evidence-platform"
    assert seen and "evidence-platform" in seen[0]
    assert "100." not in seen[0]


def test_probe_health_fail_closed_on_http_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(503, text="down")

    result = probe_health(client=_client(handler))
    assert result.reachable is False
    assert result.status is None
    assert result.evidence_platform == "evidence-platform"


def test_probe_health_fail_closed_on_timeout() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("slow")

    result = probe_health(client=_client(handler))
    assert result.reachable is False
    assert result.status is None


def test_list_matters_projects_identity_only() -> None:
    matter_id = str(uuid4())

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        assert request.url.path == "/v1/matters"
        assert "100." not in str(request.url)
        return httpx.Response(
            200,
            json={
                "data": [
                    {
                        "id": matter_id,
                        "title": "Genesee custody",
                        "evidence_items": [{"hash": "sha256:do-not-clone"}],
                        "source_content": "raw transcript stays on Agno",
                    }
                ],
                "total": 1,
                "limit": 50,
                "offset": 0,
            },
        )

    result = list_matters(client=_client(handler))
    assert result.ok is True
    assert len(result.matters) == 1
    assert result.matters[0].matter_id == matter_id
    assert result.matters[0].display_name == "Genesee custody"
    dumped = result.matters[0].__dict__
    assert "evidence_items" not in dumped
    assert "source_content" not in dumped


def test_list_matters_fail_closed_on_http_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(503, text="down")

    result = list_matters(client=_client(handler))
    assert result.ok is False
    assert result.matters == ()
    assert result.reason == "HTTPStatusError"


def test_list_matters_fail_closed_on_malformed_payload() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"unexpected": True})

    result = list_matters(client=_client(handler))
    assert result.ok is False
    assert result.matters == ()
    assert result.reason == "unexpected-shape"


def test_read_status_fail_closed_when_agno_down() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("no evidence-platform")

    snap = read_status(client=_client(handler))
    assert snap.reachable is False
    assert snap.matters_visible is False
    assert snap.evidence_platform == "evidence-platform"


def test_read_status_reports_visible_matters() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/health":
            return httpx.Response(200, json={"status": "ok"})
        if request.url.path == "/v1/matters":
            return httpx.Response(
                200,
                json={
                    "data": [{"id": str(uuid4()), "title": "Primary"}],
                    "total": 1,
                    "limit": 50,
                    "offset": 0,
                },
            )
        return httpx.Response(404)

    snap = read_status(client=_client(handler))
    assert snap.reachable is True
    assert snap.matters_visible is True
    assert snap.evidence_platform == "evidence-platform"


def test_agno_status_endpoint_fail_closed_without_agno() -> None:
    response = TestClient(app).get("/v1/agno/status")
    assert response.status_code == 200
    body = response.json()
    assert body == {
        "reachable": False,
        "evidence_platform": "evidence-platform",
        "matters_visible": False,
    }
    assert "100." not in body["evidence_platform"]


def test_normalize_sha256_strips_prefix_and_rejects_mock() -> None:
    digest = "ce3b9a1f2e2b0d3b1b3ae682fd11d7c4f35d88fc780c9bba171f5749515f68aa"
    assert normalize_sha256(f"sha256:{digest}") == digest
    assert normalize_sha256("sha256:span") is None
    assert normalize_sha256("not-hex") is None


def test_verify_sha256_intact_uses_service_name() -> None:
    digest = "ab" * 32
    seen: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        seen.append(str(request.url))
        assert request.method == "POST"
        assert request.url.path == f"/v1/verify/{digest}"
        return httpx.Response(
            200,
            json={
                "sha256_match": True,
                "verdict": "intact",
                "computed": digest,
                "recorded": digest,
            },
        )

    result = verify_sha256(f"sha256:{digest}", client=_client(handler))
    assert result.ok is True
    assert result.reachable is True
    assert result.verdict == "intact"
    assert seen and "evidence-platform" in seen[0]
    assert "100." not in seen[0]


def test_verify_sha256_fail_closed_on_timeout() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("slow")

    result = verify_sha256("ab" * 32, client=_client(handler))
    assert result.ok is False
    assert result.reachable is False
    assert result.reason == "TimeoutException"


def test_verify_sha256_404_is_not_ok() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(404, text="no custody record")

    result = verify_sha256("ab" * 32, client=_client(handler))
    assert result.ok is False
    assert result.reachable is True
    assert result.reason == "no-custody-record"


def test_verify_package_skips_network_for_non_hex() -> None:
    called = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        called["n"] += 1
        return httpx.Response(500)

    results = verify_package_hashes(["sha256:span", "sha256:mock-skip"], client=_client(handler))
    assert called["n"] == 0
    assert all(row.reason == "not-a-sha256-hex" for row in results)
