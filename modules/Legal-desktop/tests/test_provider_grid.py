"""Provider trust grid and Confidential Mode routing.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

import json

import httpx
from fastapi import FastAPI
from fastapi.testclient import TestClient
from legal_workspace.api.privilege_routes import router
from legal_workspace.domain.provider_grid import (
    ProviderGrid,
    ProviderRole,
    confidential_blocked_reason,
    eligible_confidential_models,
    provider_grid,
)
from legal_workspace.services.gateway import invoke_chat


def _app() -> FastAPI:
    app = FastAPI()
    app.include_router(router)
    return app


def test_grid_matches_cat6_and_is_not_a_legal_conclusion() -> None:
    grid = provider_grid()
    ids = [row.id for row in grid.rows]
    assert ids == [
        "ollama-cloud",
        "openrouter-zdr",
        "nim-hosted",
        "venice-private",
        "claude-consumer",
        "chatgpt-consumer",
    ]
    assert grid.pacer is False
    assert grid.local_ollama_as_trust_posture is False
    assert grid.court_safe is False
    assert grid.legal_conclusion is False
    assert "not a privilege" in grid.disclaimer.lower()
    assert "local ollama" in grid.disclaimer.lower()
    assert not any(row.id == "ollama" for row in grid.rows)
    assert not any("local" in row.id for row in grid.rows)
    assert not any("pacer" in row.id for row in grid.rows)

    by_id = {row.id: row for row in grid.rows}
    assert by_id["ollama-cloud"].role is ProviderRole.PRIMARY
    assert by_id["ollama-cloud"].confidential_eligible is True
    assert by_id["ollama-cloud"].train is False
    assert by_id["openrouter-zdr"].role is ProviderRole.SECONDARY
    assert by_id["openrouter-zdr"].confidential_eligible is True
    assert by_id["nim-hosted"].role is ProviderRole.EMBED
    assert by_id["nim-hosted"].confidential_eligible is False
    assert "not long privileged" in by_id["nim-hosted"].use.lower()
    assert by_id["venice-private"].role is ProviderRole.OPTIONAL
    assert by_id["venice-private"].confidential_eligible is True
    assert "third-party" in by_id["venice-private"].use.lower()
    assert by_id["claude-consumer"].role is ProviderRole.BLOCK
    assert by_id["claude-consumer"].confidential_eligible is False
    assert by_id["chatgpt-consumer"].role is ProviderRole.BLOCK
    assert by_id["chatgpt-consumer"].confidential_eligible is False


def test_forced_grid_flags_stay_false() -> None:
    forced = ProviderGrid(
        rows=[],
        eligible_confidential_models=["claude-consumer"],
        pacer=True,
        local_ollama_as_trust_posture=True,
        court_safe=True,
        legal_conclusion=True,
    )
    assert forced.pacer is False
    assert forced.local_ollama_as_trust_posture is False
    assert forced.court_safe is False
    assert forced.legal_conclusion is False


def test_eligible_order_is_verified_only() -> None:
    eligible = eligible_confidential_models()
    assert eligible[0] == "ollama-cloud"
    assert eligible[1] == "openrouter-zdr"
    assert "venice-private" in eligible
    assert "nim-hosted" not in eligible
    assert "claude-consumer" not in eligible
    assert "chatgpt-consumer" not in eligible
    assert confidential_blocked_reason("ollama-cloud") is None
    assert confidential_blocked_reason("claude-consumer") is not None
    assert "no silent fallback" in confidential_blocked_reason("claude-consumer").lower()
    assert confidential_blocked_reason("nim-hosted") is not None


def test_http_providers_grid() -> None:
    client = TestClient(_app())
    response = client.get("/v1/providers")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["pacer"] is False
    assert body["local_ollama_as_trust_posture"] is False
    assert body["court_safe"] is False
    assert body["legal_conclusion"] is False
    assert body["eligible_confidential_models"][0] == "ollama-cloud"
    assert any(row["id"] == "ollama-cloud" for row in body["rows"])
    assert not any(row["id"] == "ollama" for row in body["rows"])


class _InvokeOff:
    invoke_models = False


class _InvokeOn:
    invoke_models = True


def test_http_invoke_disabled_fail_closed(monkeypatch) -> None:
    monkeypatch.setattr(
        "legal_workspace.api.privilege_routes.get_settings",
        lambda: _InvokeOff(),
    )
    client = TestClient(_app())
    response = client.post(
        "/v1/gateway:invoke",
        json={"prompt": "ping", "model": "ollama-cloud", "confidential": True},
    )
    assert response.status_code == 409
    assert response.json()["detail"] == "invoke-disabled"


def test_http_confidential_blocks_consumer_claude(monkeypatch) -> None:
    monkeypatch.setattr(
        "legal_workspace.api.privilege_routes.get_settings",
        lambda: _InvokeOn(),
    )
    client = TestClient(_app())
    response = client.post(
        "/v1/gateway:invoke",
        json={"prompt": "secret", "model": "claude-consumer", "confidential": True},
    )
    assert response.status_code == 409
    assert response.json()["detail"] == "confidential_blocked"


def test_http_ineligible_model_gate_does_not_invoke(monkeypatch) -> None:
    monkeypatch.setattr(
        "legal_workspace.api.privilege_routes.get_settings",
        lambda: _InvokeOn(),
    )
    called: list[str] = []

    def boom(*_args, **_kwargs):
        called.append("invoked")
        raise AssertionError("ineligible model must not reach invoke_chat body")

    monkeypatch.setattr("legal_workspace.api.privilege_routes.invoke_chat", boom)
    client = TestClient(_app())
    response = client.post(
        "/v1/gateway:invoke",
        json={"prompt": "secret", "model": "chatgpt-consumer", "confidential": True},
    )
    assert response.status_code == 409
    assert response.json()["detail"] == "confidential_blocked"
    assert called == []


def test_http_confidential_allows_ollama_cloud(monkeypatch) -> None:
    monkeypatch.setattr(
        "legal_workspace.api.privilege_routes.get_settings",
        lambda: _InvokeOn(),
    )

    def handler(request: httpx.Request) -> httpx.Response:
        payload = json.loads(request.content)
        assert payload["model"] == "ollama-cloud"
        return httpx.Response(
            200, json={"choices": [{"message": {"content": "hypothesis only"}}]}
        )

    transport = httpx.MockTransport(handler)
    real_client = httpx.Client(transport=transport, timeout=2.0)

    def fake_invoke(prompt: str, model: str, confidential: bool = False):
        return invoke_chat(prompt, model, client=real_client, confidential=confidential)

    monkeypatch.setattr("legal_workspace.api.privilege_routes.invoke_chat", fake_invoke)
    client = TestClient(_app())
    response = client.post(
        "/v1/gateway:invoke",
        json={"prompt": "ping", "model": "ollama-cloud", "confidential": True},
    )
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["ok"] is True
    assert body["model"] == "ollama-cloud"
    assert body["confidential"] is True
    assert body["court_safe"] is False
    assert body["fallback"] is None
    assert body["text"] == "hypothesis only"
