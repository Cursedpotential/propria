"""Routing table remounts agents and chat paths without rewriting the app.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Drives load_routing / route_role / POST /v1/agent-runs / GET+PUT /v1/routing.
"""

from __future__ import annotations

import os
from pathlib import Path

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.agents import AgentRole, AgentRunCreate, route_role
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.routing import load_routing, save_routing
from legal_workspace.services.workspace import Workspace


def test_default_table_routes_real_intents() -> None:
    table = load_routing()
    assert table.framework == "next-fastapi"
    assert table.chat.run_path == "/v1/agent-runs"
    assert table.chat.confidential_path == "/v1/gateway:invoke"
    assert "research" in table.agents
    assert route_role("attack this draft as opposing counsel") is AgentRole.REDTEAM
    assert route_role("research Vodvarka lookback") is AgentRole.RESEARCH
    assert route_role("draft a motion section") is AgentRole.DRAFTER


def test_overlay_file_changes_which_agent_is_called(tmp_path, monkeypatch) -> None:
    table = load_routing()
    table.role_keywords = {
        "drafter": ["research", "vodvarka"],
        "research": ["unused-token"],
    }
    overlay = tmp_path / "routing.json"
    save_routing(table, overlay)
    monkeypatch.setenv("LEGAL_WORKSPACE_ROUTING_FILE", str(overlay))
    assert route_role("research Vodvarka lookback") is AgentRole.DRAFTER
    run = Workspace(tmp_path / "ws").add_agent_run(
        AgentRunCreate(intent="research Vodvarka lookback", prompt="outline only")
    )
    assert run.role is AgentRole.DRAFTER
    assert "Motion writer" in run.output


def test_mapping_edit_retargets_dispatch_and_keeps_forbidden_blocked(
    tmp_path, monkeypatch
) -> None:
    """Edit a page path and intent→role; file/approve/serve stay blocked."""
    table = load_routing()
    table.surfaces = [
        item if item.label != "Filing readiness checklist" else item.model_copy(update={"path": "/final-copy"})
        for item in table.surfaces
    ]
    table.role_keywords = {
        "drafter": ["research", "vodvarka", "statute"],
        "research": ["unused-research-token"],
    }
    overlay = tmp_path / "routing.json"
    save_routing(table, overlay)
    monkeypatch.setenv("LEGAL_WORKSPACE_ROUTING_FILE", str(overlay))
    store = workspace_mod.get_workspace(tmp_path / "ws")
    main_mod.WORKSPACE = store
    client = TestClient(app)

    from legal_workspace.services.routing import path_for_query

    assert path_for_query("Filing readiness checklist") == "/final-copy"
    resolved = client.get("/v1/routing/resolve", params={"q": "Filing readiness checklist"})
    assert resolved.status_code == 200
    assert resolved.json()["path"] == "/final-copy"

    remapped = client.post(
        "/v1/agent-runs",
        json={"intent": "research Vodvarka lookback", "prompt": "outline only"},
    )
    assert remapped.status_code == 200
    assert remapped.json()["role"] == "drafter"
    assert remapped.json()["status"] != "blocked"

    for intent in ("file this motion", "approve this draft", "serve the other parent"):
        blocked = client.post(
            "/v1/agent-runs",
            json={"intent": intent, "prompt": "do not execute"},
        )
        assert blocked.status_code == 200, blocked.text
        assert blocked.json()["status"] == "blocked"
        assert blocked.json()["effective_model"] == "not-invoked"


def test_http_get_and_put_routing(tmp_path, monkeypatch) -> None:
    overlay = tmp_path / "routing.json"
    monkeypatch.setenv("LEGAL_WORKSPACE_ROUTING_FILE", str(overlay))
    store = workspace_mod.get_workspace(tmp_path / "ws")
    main_mod.WORKSPACE = store
    client = TestClient(app)
    listed = client.get("/v1/routing")
    assert listed.status_code == 200
    body = listed.json()
    assert body["chat"]["run_path"] == "/v1/agent-runs"
    assert "mnemonics" not in body
    assert any(item["label"] == "Assistant activity log" and item["path"] == "/assistant-log" for item in body["surfaces"])
    assert all("cmd" not in item for item in body["surfaces"])
    body["chat"]["backend"] = "python-only"
    body["agents"]["research"]["model"] = "ollama-cloud"
    saved = client.put("/v1/routing", json=body)
    assert saved.status_code == 200, saved.text
    assert saved.json()["ok"] is True
    reloaded = load_routing()
    assert reloaded.chat.backend == "python-only"
    assert reloaded.agents["research"].model == "ollama-cloud"
    run = client.post(
        "/v1/agent-runs",
        json={"intent": "research MCL 722.27a(8)", "prompt": "outline only"},
    )
    assert run.status_code == 200
    assert run.json()["role"] == "research"
    assert run.json()["requested_model"] == "ollama-cloud"
