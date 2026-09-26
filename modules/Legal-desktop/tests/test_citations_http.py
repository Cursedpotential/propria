"""POST /v1/citations:parse is structure-only. Never a citator.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from fastapi.testclient import TestClient

from legal_workspace.api.main import app

client = TestClient(app)


def test_citation_parse_never_marks_citator_verified() -> None:
    response = client.post(
        "/v1/citations:parse",
        json={"text": "See Vodvarka v Grasmeyer, 259 Mich App 499 (2003)."},
    )
    assert response.status_code in {200, 503}
    if response.status_code == 503:
        assert "Shepardize" in response.json()["detail"]
        return
    body = response.json()
    assert body["is_citator_verified"] is False
    assert body["court_safe"] is False
    assert all(item["is_citator_verified"] is False for item in body["citations"])


def test_routing_surfaces_use_handoff_english_labels() -> None:
    body = client.get("/v1/routing").json()
    labels = {item["label"] for item in body["surfaces"]}
    groups = {item["group"] for item in body["surfaces"]}
    assert groups == {"Assistant", "Research", "Contracts", "Drafting", "Operations"}
    assert "Case search" in labels
    assert "Motion writer" in labels
    assert "Ask the assistant" in labels
    assert "Cases and statutes" not in labels
    assert "Write a paper" not in labels
    assert "Your case" not in labels
    assert all("cmd" not in item for item in body["surfaces"])
