"""Hand-marked source units: the Takeout confirm gate and the file store.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.config import settings
from app.runtime.source_unit_marks import router
from app.service import source_unit_marks as service


@pytest.fixture
def client(monkeypatch, tmp_path):
    monkeypatch.setattr(settings, "source_unit_marks_path", str(tmp_path / "unit-marks.json"))
    app = FastAPI()

    @app.middleware("http")
    async def principal(request, call_next):
        request.state.principal = "msalem"
        return await call_next(request)

    app.include_router(router)
    return TestClient(app)


def test_takeout_needs_an_explicit_confirm(client):
    body = {"unit_root": "vault/v1/takeouts", "unit_type": "takeout"}
    assert client.post("/api/sources/unit-marks", json=body).status_code == 409
    assert client.post("/api/sources/unit-marks", json={**body, "confirm": True}).status_code == 201


def test_a_plain_unit_is_recorded_and_read_back(client):
    created = client.post(
        "/api/sources/unit-marks",
        json={"unit_root": "vault/v1/notes/", "unit_type": "obsidian_vault", "label": "Notes"},
    )
    assert created.status_code == 201
    assert created.json()["unit_root"] == "vault/v1/notes"
    assert created.json()["origin"] == "hand_marked"
    listed = client.get("/api/sources/unit-marks").json()
    assert [item["unit_root"] for item in listed["items"]] == ["vault/v1/notes"]
    assert listed["catalog_units_are_read_only"] is True
    assert "unit-marks.json" in listed["storage"]


def test_remarking_one_root_replaces_rather_than_duplicates(client):
    for kind in ("other", "git_repo"):
        client.post("/api/sources/unit-marks", json={"unit_root": "vault/v1/code", "unit_type": kind})
    items = client.get("/api/sources/unit-marks").json()["items"]
    assert len(items) == 1 and items[0]["unit_type"] == "git_repo"


def test_takeout_parts_are_proposed_with_the_gaps_named(client):
    response = client.post(
        "/api/sources/unit-proposal",
        json={
            "unit_root": "vault/v1/takeouts/",
            "names": [
                "takeout-20220301T120000Z-7fa2-001.zip",
                "takeout-20220301T120000Z-7fa2-003.zip",
                "unrelated.txt",
            ],
        },
    )
    body = response.json()
    assert body["looks_like"] == "takeout" and body["requires_confirmation"] is True
    assert body["basis"] == "observed_listing"
    assert body["takeout_sets"][0]["parts_present"] == [1, 3]
    assert body["takeout_sets"][0]["parts_missing"] == [2]


def test_a_folder_with_no_pattern_proposes_nothing_to_confirm(client):
    body = client.post(
        "/api/sources/unit-proposal", json={"unit_root": "vault/v1/loose", "names": ["a.xml", "b.xml"]}
    ).json()
    assert body["looks_like"] is None and body["requires_confirmation"] is False
    assert body["takeout_sets"] == []


def test_marker_directories_name_their_kind():
    assert service._kind_from_names([".obsidian"]) == "obsidian_vault"
    assert service._kind_from_names([".git"]) == "git_repo"
    assert service._kind_from_names(["notes.md"]) is None
