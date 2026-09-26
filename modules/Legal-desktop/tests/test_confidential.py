"""Confidential Mode persists on disk, not only in the browser.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.agents import AgentRunCreate, AgentRunStatus
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.gateway import GatewayResult
from legal_workspace.services.workspace import Workspace


def test_confidential_toggle_persists(tmp_path) -> None:
    first = Workspace(tmp_path)
    assert first.load().confidential_mode is False
    assert first.set_confidential(True) is True
    reloaded = Workspace(tmp_path)
    assert reloaded.load().confidential_mode is True
    assert reloaded.set_confidential(False) is False
    assert Workspace(tmp_path).load().confidential_mode is False


def test_http_confidential_persists(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    assert client.get("/v1/confidential").json()["on"] is False
    put = client.put("/v1/confidential", json={"on": True})
    assert put.status_code == 200
    assert put.json()["on"] is True
    assert put.json()["court_safe"] is False
    assert Workspace(tmp_path).load().confidential_mode is True
    assert client.get("/v1/confidential").json()["on"] is True


def test_persisted_confidential_blocks_ineligible_model_without_invoker(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    workspace.set_confidential(True)
    called: list[str] = []

    def invoker(prompt: str, model: str) -> GatewayResult:
        called.append(model)
        return GatewayResult(ok=True, text="LEAK", model=model)

    run = workspace.add_agent_run(
        AgentRunCreate(
            intent="research Vodvarka lookback",
            prompt="outline only",
            requested_model="claude-consumer",
        ),
        invoker=invoker,
    )
    assert run.status is AgentRunStatus.BLOCKED
    assert "confidential_blocked" in run.output
    assert "LEAK" not in run.output
    assert called == []


def test_http_persisted_confidential_returns_409_no_fallback(tmp_path, monkeypatch) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    store.set_confidential(True)

    class _InvokeOn:
        invoke_models = True

    monkeypatch.setattr(
        "legal_workspace.api.privilege_routes.get_settings",
        lambda: _InvokeOn(),
    )
    called: list[str] = []

    def boom(*_args, **_kwargs):
        called.append("invoked")
        raise AssertionError("ineligible model must not reach invoke_chat")

    monkeypatch.setattr("legal_workspace.api.privilege_routes.invoke_chat", boom)
    client = TestClient(app)
    response = client.post(
        "/v1/gateway:invoke",
        json={"prompt": "secret", "model": "chatgpt-consumer", "confidential": False},
    )
    assert response.status_code == 409
    assert response.json()["detail"] == "confidential_blocked"
    assert called == []
