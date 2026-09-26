"""Questions the judge decides + both-parent factor analysis.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from fastapi import FastAPI
from fastapi.testclient import TestClient
from legal_workspace.api import main as main_mod
from legal_workspace.api.factor_routes import router as factor_router
from legal_workspace.domain.factors import (
    FACTOR_TITLES,
    FactorLetter,
    empty_factor_matrix,
    structural_both_parent_prompt,
)
from legal_workspace.domain.issue import default_custody_modification_issue, issue_tree
from legal_workspace.services import workspace as workspace_mod


def _walk(issue):
    yield issue
    for child in issue.children:
        yield from _walk(child)


def test_default_issue_has_children() -> None:
    issue = default_custody_modification_issue()
    assert issue.title.startswith("Modification of custody")
    assert "Vodvarka" in issue.governing_authority
    assert len(issue.children) == 3
    titles = [child.title for child in issue.children]
    assert titles[0] == "Proper cause / change of circumstances"
    assert "Vodvarka" in issue.children[0].governing_authority
    assert titles[1] == "Established custodial environment / burden"
    assert "Pierron" in issue.children[1].governing_authority
    assert "Shade" in issue.children[1].governing_authority
    assert titles[2] == "Best interests MCL 722.23 (a)–(l)"


def test_issue_tree_helper_returns_seed() -> None:
    seeded = issue_tree()
    assert len(seeded.children) == 3
    live = default_custody_modification_issue()
    assert issue_tree(live).issue_id == live.issue_id


def test_seed_issue_has_no_invented_citations() -> None:
    for node in _walk(default_custody_modification_issue()):
        for element in node.elements:
            assert element.supporting == []
            assert element.contradicting == []


def test_best_interests_elements_use_factor_titles() -> None:
    best = next(
        child
        for child in default_custody_modification_issue().children
        if child.title.startswith("Best interests")
    )
    labels = [element.label for element in best.elements]
    assert len(labels) == 12
    for letter, title in FACTOR_TITLES.items():
        assert f"({letter.value}) {title}" in labels


def test_both_parent_prompt_mentions_both_sides_and_not_a_finding() -> None:
    entry = empty_factor_matrix()[0]
    prompt = structural_both_parent_prompt(entry)
    lowered = prompt.lower()
    assert "petitioner" in lowered
    assert "respondent" in lowered
    assert "both" in lowered
    assert "not a finding" in lowered
    assert "court_safe=false" in lowered
    assert "do not diagnose" in lowered
    assert "do not invent facts" in lowered
    assert entry.letter is FactorLetter.A


def test_empty_factor_matrix_has_no_invented_citations() -> None:
    for entry in empty_factor_matrix():
        assert entry.petitioner.citations == []
        assert entry.respondent.citations == []
        assert entry.contradictions == []
        assert entry.proposed_finding is None
        assert entry.both_parent_analysis is None


def test_factor_routes_read_live_workspace(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    app = FastAPI()
    app.include_router(factor_router)
    client = TestClient(app)

    issues = client.get("/v1/issues")
    assert issues.status_code == 200
    body = issues.json()
    assert len(body["children"]) == 3
    for node in [body, *body["children"]]:
        for element in node["elements"]:
            assert element["supporting"] == []
            assert element["contradicting"] == []

    analysis = client.get("/v1/factors/j/analysis")
    assert analysis.status_code == 200
    payload = analysis.json()
    assert payload["letter"] == "j"
    assert payload["petitioner"]["citations"] == []
    assert payload["respondent"]["citations"] == []
    assert payload["both_parent_analysis"] == ""

    missing = client.get("/v1/factors/z/analysis")
    assert missing.status_code == 422

    note = client.post("/v1/factors/g/notes", json={"side": "petitioner", "text": "No diagnosis — only documented conduct."})
    assert note.status_code == 200, note.text
    assert any("No diagnosis" in item for item in note.json()["petitioner"]["for_parent"])

    child = client.post(
        "/v1/issues",
        json={"title": "Parenting-time midweek", "governing_authority": "MCL 722.27a"},
    )
    assert child.status_code == 200, child.text
    tree = client.get("/v1/issues").json()
    assert any(row["title"] == "Parenting-time midweek" for row in tree["children"])
