"""App settings persist on disk — stand-in for legal_core.app_settings.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_settings_persist_phase_and_reject_light_theme(tmp_path) -> None:
    store = Workspace(tmp_path)
    state = store.update_settings(case_phase="Hearing")
    assert state.case_phase == "Hearing"
    assert state.theme == "dark"
    assert (tmp_path / "legal.sqlite").is_file()
    reloaded = Workspace(tmp_path).load()
    assert reloaded.case_phase == "Hearing"
    try:
        store.update_settings(theme="light")
        raise AssertionError("light theme must be rejected")
    except ValueError as exc:
        assert "dark-only" in str(exc)


def test_http_settings_round_trip(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    listed = client.get("/v1/settings")
    assert listed.status_code == 200
    assert listed.json()["theme"] == "dark"
    saved = client.put("/v1/settings", json={"case_phase": "Trial"})
    assert saved.status_code == 200, saved.text
    assert saved.json()["case_phase"] == "Trial"
    assert Workspace(tmp_path).load().case_phase == "Trial"
    bad = client.put("/v1/settings", json={"theme": "light"})
    assert bad.status_code == 400
