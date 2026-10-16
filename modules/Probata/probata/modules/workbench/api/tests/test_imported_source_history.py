"""Verify read-only, exact export-history grouping without fabricating derivation.

Inputs: synthetic source keys, versions and API requests. Outputs: pass/fail
assertions. Effects: in-memory fixtures only, no source or database writes.
"""
from datetime import datetime, timezone

from fastapi import FastAPI
from fastapi.testclient import TestClient
import pytest

from app.runtime.imported import router
from app.service import imported


@pytest.fixture
def history_client(monkeypatch):
    app = FastAPI()
    app.include_router(router)
    exported = "b2://fixture/casevault/message.xml"
    now = datetime(2026, 10, 5, tzinfo=timezone.utc)
    rows = [
        {
            "id": "v2", "source_key": "b2://fixture/casevault/message.xml.derived/a.json",
            "export_key": exported, "version_ordinal": 2, "acquired_at": now,
            "published": False, "approved": True, "rejected": False,
            "raw_n": 4, "norm_n": 3, "msgs": 3, "calls": 0,
        },
        {
            "id": "v1", "source_key": exported,
            "export_key": exported, "version_ordinal": 1, "acquired_at": now,
            "published": True, "approved": True, "rejected": False,
            "raw_n": 20, "norm_n": 18, "msgs": 18, "calls": 0,
        },
        {
            "id": "v-other", "source_key": "b2://fixture/unrelated/other.xml",
            "export_key": "b2://fixture/unrelated/other.xml",
            "version_ordinal": 1, "acquired_at": now,
            "published": True, "approved": True, "rejected": False,
            "raw_n": 9, "norm_n": 9, "msgs": 9, "calls": 0,
        },
    ]
    async def selected(source_id):
        if source_id != "source-1":
            raise imported.ImportedError("Unknown source", 404)
        return {"export_key": exported}
    monkeypatch.setattr(imported, "_export_for", selected)
    monkeypatch.setattr(imported, "live_matter", lambda: "real-case")
    monkeypatch.setattr(imported, "_cached", lambda key, seconds, read: read())
    monkeypatch.setattr(imported.pg, "source_versions", lambda matter: rows if matter == "real-case" else [])
    return TestClient(app)


def test_history_returns_only_exact_export_group_and_version_lineage(history_client):
    response = history_client.get("/api/imported/sources/source-1/history?limit=1")
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 2 and data["next_offset"] == 1
    assert data["relation_basis"] == "exact_recorded_export_key"
    assert data["items"][0]["source_version_id"] == "v2"
    assert data["items"][0]["relation"] == "same_recorded_export"
    assert "duplicate" not in data["items"][0]

    second = history_client.get("/api/imported/sources/source-1/history?offset=1&limit=1")
    assert second.status_code == 200
    assert second.json()["items"][0]["relation"] == "export_root"
    assert second.json()["next_offset"] is None


def test_source_history_404s_and_bounded_pagination(history_client):
    assert history_client.get("/api/imported/sources/unavailable/history").status_code == 404
    assert history_client.get("/api/imported/sources/source-1/history?limit=1000").status_code == 422
    assert history_client.get("/api/imported/sources/source-1/history?offset=-1").status_code == 422



def test_imported_facets_filter_all_recorded_sources_before_paginating(history_client, monkeypatch):
    rows = [
        {"id": "a", "file_name": "takeout-a.xml", "casevault_key": "A/takeout-a.xml",
         "format": "SMS", "status": "failed", "owner": "Person A", "owner_name": "Person A",
         "files": 1, "raw": 8, "normalized": 4, "committed": 0},
        {"id": "b", "file_name": "takeout-b.xml", "casevault_key": "B/takeout-b.xml",
         "format": "SMS", "status": "failed", "owner": "Person B", "owner_name": "Person B",
         "files": 2, "raw": 5, "normalized": 3, "committed": 0},
        {"id": "c", "file_name": "unrelated.xml", "casevault_key": "C/unrelated.xml",
         "format": "SMS", "status": "committed", "owner": None, "owner_name": None,
         "files": 1, "raw": 3, "normalized": 3, "committed": 3},
        {"id": "d", "file_name": "takeout-social.json", "casevault_key": "D/takeout-social.json",
         "format": "Facebook", "status": "failed", "owner": None, "owner_name": None,
         "files": 1, "raw": 7, "normalized": 1, "committed": 0},
    ]

    async def fake_exports():
        return rows, True

    monkeypatch.setattr(imported, "_exports", fake_exports)
    first = history_client.get(
        "/api/imported/sources?format=SMS&status=failed&owner=recorded&q=takeout&limit=1"
    )
    assert first.status_code == 200
    assert first.json()["total"] == 2
    assert [x["id"] for x in first.json()["items"]] == ["a"]
    assert first.json()["next_offset"] == 1
    assert first.json()["totals"]["raw"] == 13
    second = history_client.get(
        "/api/imported/sources?format=SMS&status=failed&owner=recorded&q=takeout&limit=1&offset=1"
    )
    assert second.json()["items"][0]["id"] == "b"
    assert second.json()["next_offset"] is None
    unowned = history_client.get("/api/imported/sources?owner=unrecorded&limit=10")
    assert [x["id"] for x in unowned.json()["items"]] == ["c", "d"]

    for query in ("owner=other", "status=made_up", "q=" + "x" * 201):
        assert history_client.get("/api/imported/sources?" + query).status_code == 422
