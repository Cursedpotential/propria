"""Agent runs persist. Forbidden intents are blocked, not executed.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.agents import (
    AgentRole,
    AgentRunCreate,
    AgentRunStatus,
    route_role,
)
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_router_picks_narrow_roles() -> None:
    assert route_role("attack this draft as opposing counsel") is AgentRole.REDTEAM
    assert route_role("research Vodvarka lookback") is AgentRole.RESEARCH
    assert route_role("draft a motion section") is AgentRole.DRAFTER


def test_file_intent_is_blocked_and_persisted(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    run = workspace.add_agent_run(
        AgentRunCreate(intent="file this motion", prompt="send it to the clerk")
    )
    assert run.status is AgentRunStatus.BLOCKED
    assert run.court_safe is False
    assert run.effective_model == "not-invoked"
    reloaded = Workspace(tmp_path)
    assert any(item.run_id == run.run_id for item in reloaded.load().agent_runs)


def test_research_intent_needs_owner_review(tmp_path) -> None:
    run = Workspace(tmp_path).add_agent_run(
        AgentRunCreate(intent="research MCL 722.27a(8)", prompt="outline only")
    )
    assert run.role is AgentRole.RESEARCH
    assert run.status is AgentRunStatus.NEEDS_OWNER_REVIEW
    assert "hypothesis" in run.output


def test_http_agent_runs(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    blocked = client.post(
        "/v1/agent-runs",
        json={"intent": "approve this draft", "prompt": "ship it"},
    )
    assert blocked.status_code == 200
    assert blocked.json()["status"] == "blocked"
    listed = client.get("/v1/agent-runs")
    assert listed.status_code == 200
    assert len(listed.json()) == 1
    home = client.get("/v1/matter")
    assert home.json()["agent_run_count"] == 1
    assert any(row["path"] == "/assistant-log" and row["label"] == "Assistant activity log" for row in home.json()["next_surfaces"])
