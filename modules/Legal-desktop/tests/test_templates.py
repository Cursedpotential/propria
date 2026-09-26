"""Templates instantiate drafts without invented dates.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.templates import TemplateInstantiate, catalog
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_catalog_is_this_matter_only() -> None:
    ids = {row.template_id for row in catalog()}
    assert "motion-parenting-time-specific" in ids
    assert "objection-to-referee" in ids
    assert all(row.proceeding == "postjudgment_domestic_relations" for row in catalog())
    assert all(row.court_safe is False for row in catalog())
    notice = next(row for row in catalog() if row.template_id == "notice-of-hearing")
    assert any("blank" in item.lower() or "clerk" in item.lower() for item in notice.outline)


def test_instantiate_persists_draft(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    section = workspace.instantiate_template(
        TemplateInstantiate(template_id="motion-parenting-time-specific")
    )
    assert "MCL 722.27a(8)" in section.body
    assert "do not invent" in section.body.lower() or "clerk-confirm" in section.body.lower()
    reloaded = Workspace(tmp_path)
    assert reloaded.load().drafts[0].section_id == section.section_id


def test_http_templates(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    listed = client.get("/v1/templates")
    assert listed.status_code == 200
    assert len(listed.json()) >= 6
    created = client.post(
        "/v1/templates:instantiate",
        json={"template_id": "proposed-order"},
    )
    assert created.status_code == 200, created.text
    assert created.json()["heading"]
    missing = client.post("/v1/templates:instantiate", json={"template_id": "not-a-template"})
    assert missing.status_code == 404
    home = client.get("/v1/matter")
    assert any(row["path"] == "/templates" and row["label"] == "Starting templates" for row in home.json()["next_surfaces"])
    assert home.json()["draft_count"] == 1
