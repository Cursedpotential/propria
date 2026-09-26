"""Pluggable sources: CourtListener identity search, not a citator.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from typing import Any

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic import ValidationError

from legal_workspace.api.source_routes import source_router
from legal_workspace.domain.sources import (
    SearchResult,
    courtlistener_default_config,
    redact_source_config,
)
from legal_workspace.services.sources import (
    SEARCH_TIMEOUT,
    load_source_configs,
    search_source,
)


@pytest.fixture(autouse=True)
def _clear_court_listener_token(monkeypatch) -> None:
    monkeypatch.delenv("COURT_LISTENER_TOKEN", raising=False)


def _client(handler) -> httpx.Client:
    return httpx.Client(transport=httpx.MockTransport(handler), timeout=SEARCH_TIMEOUT)


def _app_client() -> TestClient:
    app = FastAPI()
    app.include_router(source_router)
    return TestClient(app)


def _sample_row() -> dict[str, Any]:
    return {
        "cluster_id": 4242,
        "caseName": "Alpha v Beta",
        "citation": ["1 Test 2"],
        "court": "Mich. Ct. App.",
        "dateFiled": "2003-02-11",
        "absolute_url": "/opinion/4242/alpha-v-beta/",
        "snippet": "opinion body must not become evidence",
        "text": "FULL OPINION TEXT MUST BE DROPPED",
    }


def test_default_config_is_courtlistener_free_and_not_pacer() -> None:
    configs = load_source_configs()
    assert len(configs) == 1
    cfg = configs[0]
    assert cfg is not None
    assert cfg.id == "courtlistener"
    assert cfg.display_name == "CourtListener / RECAP"
    assert cfg.may_cost_money is False
    assert cfg.daily_budget is None
    dumped = courtlistener_default_config().model_dump(by_alias=True, exclude_none=True)
    assert dumped["endpoints"]["search"]["url"] == (
        "https://www.courtlistener.com/api/rest/v4/search/"
    )
    assert dumped["endpoints"]["search"]["query"] == {"q": "{{query}}", "type": "o"}
    assert dumped["map"]["list"] == "$.results"
    assert dumped["map"]["Case"]["name"] == "$.caseName"
    blob = " ".join(f"{item.id} {item.display_name}" for item in configs).lower()
    assert "pacer" not in blob
    assert all("pacer" not in item.id.lower() for item in configs)


def test_search_maps_identity_fields_and_drops_opinion_text() -> None:
    seen: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        seen.append(request)
        assert request.method == "GET"
        assert request.url.host == "www.courtlistener.com"
        assert request.url.path == "/api/rest/v4/search/"
        assert request.url.params["q"] == "Alpha"
        assert request.url.params["type"] == "o"
        assert "100." not in str(request.url)
        assert "Authorization" not in request.headers
        return httpx.Response(200, json={"count": 1, "results": [_sample_row()]})

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    assert result.ok is True
    assert result.reason is None
    assert result.is_citator_verified is False
    assert len(result.hits) == 1
    hit = result.hits[0]
    assert hit.id == "4242"
    assert hit.name == "Alpha v Beta"
    assert hit.citation == "1 Test 2"
    assert hit.court == "Mich. Ct. App."
    assert hit.date == "2003-02-11"
    assert hit.url == "https://www.courtlistener.com/opinion/4242/alpha-v-beta/"
    dumped = hit.model_dump()
    assert set(dumped) == {"id", "name", "citation", "court", "date", "url"}
    assert "snippet" not in dumped
    assert "text" not in dumped
    assert dumped.get("is_citator_verified") is not True
    envelope = result.model_dump()
    assert envelope.get("is_citator_verified") is not True
    assert envelope["is_citator_verified"] is False
    assert seen


def test_search_fail_closed_on_timeout() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("slow")

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    assert result.ok is False
    assert result.hits == ()
    assert result.reason == "timeout"
    assert result.is_citator_verified is False


def test_search_fail_closed_on_503() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(503, text="down")

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    assert result.ok is False
    assert result.hits == ()
    assert result.reason == "upstream-error"
    assert result.is_citator_verified is False


def test_search_401_is_auth_required() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(401, text="no")

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    assert result.ok is False
    assert result.hits == ()
    assert result.reason == "auth-required"
    assert result.is_citator_verified is False


def test_search_403_is_auth_required() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(403, text="no")

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    assert result.ok is False
    assert result.hits == ()
    assert result.reason == "auth-required"


def test_token_sent_when_env_set(monkeypatch) -> None:
    monkeypatch.setenv("COURT_LISTENER_TOKEN", "test-token")

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == "Token test-token"
        return httpx.Response(200, json={"results": []})

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    assert result.ok is True
    assert result.hits == ()


def test_unknown_source_does_not_call_network() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise AssertionError(f"unexpected request {request.url}")

    result = search_source("pacer", "anything", client=_client(handler))
    assert result.ok is False
    assert result.hits == ()
    assert result.reason == "unknown-source"


def test_results_never_include_citator_verified_true() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"results": [_sample_row()]})

    result = search_source("courtlistener", "Alpha", client=_client(handler))
    dumped = result.model_dump()
    assert dumped["is_citator_verified"] is False
    assert "is_citator_verified" not in result.hits[0].model_dump()
    with pytest.raises(ValidationError):
        SearchResult(ok=True, hits=(), is_citator_verified=True)  # type: ignore[arg-type]


def test_list_route_redacts_and_excludes_pacer() -> None:
    response = _app_client().get("/v1/sources")
    assert response.status_code == 200
    body = response.json()
    assert len(body) == 1
    row = body[0]
    assert row["id"] == "courtlistener"
    assert row["may_cost_money"] is False
    assert "pacer" not in row["id"].lower()
    assert "pacer" not in row["display_name"].lower()
    assert "token" not in row
    auth = row["auth"]
    assert "token" not in {key.lower() for key in auth}
    redacted = redact_source_config(courtlistener_default_config())
    assert redacted["id"] == "courtlistener"


def test_search_route_never_returns_citator_verified(monkeypatch) -> None:
    def fake(source_id: str, query: str, client: httpx.Client | None = None) -> SearchResult:
        assert source_id == "courtlistener"
        assert query == "Alpha"
        return search_source(
            source_id,
            query,
            client=_client(
                lambda request: httpx.Response(200, json={"results": [_sample_row()]})
            ),
        )

    monkeypatch.setattr(
        "legal_workspace.api.source_routes.search_source",
        fake,
    )
    response = _app_client().get("/v1/sources/courtlistener/search", params={"q": "Alpha"})
    assert response.status_code == 200
    body = response.json()
    assert body["ok"] is True
    assert body.get("is_citator_verified") is not True
    assert body["is_citator_verified"] is False
    assert body["hits"][0]["citation"] == "1 Test 2"
    assert "snippet" not in body["hits"][0]
    assert "is_citator_verified" not in body["hits"][0]
