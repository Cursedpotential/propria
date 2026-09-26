"""Open questions persist. Currency flags stale cited drafts.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.research import ResearchCreate, ResearchStatus
from legal_workspace.domain.work_product import WorkProductState, WorkProductVersion
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_structural_question_and_roundtrip(tmp_path) -> None:
    first = Workspace(tmp_path)
    seeded = first.load().research_questions
    assert seeded
    assert all(item.jurisdiction == "US-MI" for item in seeded)
    added = first.add_research_question(
        ResearchCreate(
            question="Does Shade still govern the established-custodial-environment inquiry here?",
            plan="Read Shade against the last order once that order is imported. Do not invent facts.",
            authority_ids=["Shade v Wright"],
            uncertainty="Need the last order before applying the test.",
        )
    )
    reloaded = Workspace(tmp_path)
    assert any(item.question_id == added.question_id for item in reloaded.load().research_questions)
    done = reloaded.set_research_status(added.question_id, ResearchStatus.ANSWERED)
    assert done.status is ResearchStatus.ANSWERED
    assert Workspace(tmp_path).load().research_questions


def test_currency_flag_marks_cited_work_product(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    state = workspace.load()
    state.work_product = WorkProductVersion(
        work_product_id=state.matter.matter_id,
        cited_authority_ids=["MCL 722.23"],
        content_hash="sha256:tmp",
    )
    workspace._write(state, action="test-set-product")
    flag = workspace.flag_authority_currency("MCL 722.23", "Owner recorded a statute amendment check.")
    assert flag.identifier == "MCL 722.23"
    assert workspace.load().work_product.state is WorkProductState.REVALIDATION_REQUIRED
    try:
        workspace.flag_authority_currency("Not A Real Cite", "no")
        raise AssertionError("unknown authority must fail")
    except ValueError as exc:
        assert "unknown" in str(exc)


def test_http_research_and_home_count(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    home = client.get("/v1/matter")
    assert home.status_code == 200
    assert home.json()["open_research_count"] >= 1
    assert any(row["path"] == "/open-questions" and row["label"] == "Open questions" for row in home.json()["next_surfaces"])
    created = client.post(
        "/v1/research",
        json={
            "question": "Is the Genesee FOC parenting-time guideline current?",
            "plan": "Check the FOCB memo date against the packet copy. Do not invent a revision.",
            "jurisdiction": "US-MI",
        },
    )
    assert created.status_code == 200, created.text
    listed = client.get("/v1/research")
    assert listed.status_code == 200
    assert len(listed.json()) >= 2
    flagged = client.post(
        "/v1/currency-flags",
        json={"identifier": "MCL 722.23", "note": "Spot-check against legislature site."},
    )
    assert flagged.status_code == 200
