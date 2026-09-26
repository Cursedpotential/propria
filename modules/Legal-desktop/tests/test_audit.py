"""Activity log and Inbound notices read the same event file. English names.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services import workspace as workspace_mod


def test_audit_and_triggers_are_lists(tmp_path) -> None:
    main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_path)
    client = TestClient(app)
    audit = client.get("/v1/audit")
    triggers = client.get("/v1/triggers")
    assert audit.status_code == 200
    assert triggers.status_code == 200
    assert isinstance(audit.json(), list)
    assert isinstance(triggers.json(), list)
    home = client.get("/v1/matter")
    assert home.status_code == 200
    assert "audit_count" in home.json()
    assert all(
        {"path", "label", "help"} <= set(item)
        for item in home.json()["next_surfaces"]
    )
    assert all("cmd" not in item for item in home.json()["next_surfaces"])
    labels = [item["label"] for item in home.json()["next_surfaces"]]
    assert "Case search" in labels
    assert "Motion writer" in labels
