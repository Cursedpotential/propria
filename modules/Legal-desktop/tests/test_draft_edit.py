"""Draft edits persist. Released sections fork.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.auth import AuthenticatedPrincipal
from legal_workspace.api.main import app
from legal_workspace.domain.release import ReleaseCreate
from legal_workspace.domain.factors import FactorCitationLink, FactorLetter
from legal_workspace.domain.review import ReviewCreate, ReviewVerdict
from legal_workspace.domain.templates import TemplateInstantiate
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace
from conftest import seed_synthetic_approved_package
from test_first_slice import _approved_package

HUMAN = AuthenticatedPrincipal(
    subject="owner-subject",
    username="owner",
    email="owner@example.test",
    groups=("advocatio-users",),
    source="authentik",
)


def test_edit_updates_unreleased_draft(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    section = workspace.instantiate_template(
        TemplateInstantiate(template_id="motion-parenting-time-specific")
    )
    edited = workspace.update_draft(section.section_id, "Edited heading", "Edited body.")
    assert edited.section_id == section.section_id
    assert Workspace(tmp_path).load().drafts[0].body == "Edited body."


def test_edit_of_released_section_forks(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    package, citation = _approved_package(workspace.load().matter.matter_id)
    seed_synthetic_approved_package(workspace, package)
    workspace.attach_factor_citation(
        FactorCitationLink(
            letter=FactorLetter.J,
            side="petitioner",
            citation=citation,
            package_id=package.package_id,
        )
    )
    section = workspace.draft_factor_section(FactorLetter.J, "Factor (j)", "Original.")
    workspace.add_review(
        ReviewCreate(
            section_id=section.section_id,
            verdict=ReviewVerdict.APPROVE,
            rationale="test",
        ),
        principal=HUMAN,
    )
    workspace.build_release_candidate(ReleaseCreate(section_ids=[section.section_id]))
    forked = workspace.update_draft(section.section_id, "Fork", "New text.")
    assert forked.section_id != section.section_id
    reloaded = Workspace(tmp_path).load()
    assert any(item.body == "Original." for item in reloaded.drafts)
    assert any(item.body == "New text." for item in reloaded.drafts)


def test_http_put_edit_does_not_collide_with_gate(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    created = client.post(
        "/v1/templates:instantiate",
        json={"template_id": "motion-parenting-time-specific"},
    )
    assert created.status_code == 200, created.text
    section_id = created.json()["section_id"]
    gated = client.post(f"/v1/drafts/{section_id}:gate")
    assert gated.status_code != 422, gated.text
    edited = client.put(
        f"/v1/drafts/{section_id}",
        json={"heading": "HTTP edit", "body": "Persisted over PUT."},
    )
    assert edited.status_code == 200, edited.text
    assert edited.json()["body"] == "Persisted over PUT."
    assert Workspace(tmp_path).load().drafts[0].body == "Persisted over PUT."
