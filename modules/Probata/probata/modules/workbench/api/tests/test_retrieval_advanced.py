"""Verify Super Index searching is truly pre-ranked and does not broaden caller authority.

Inputs: synthetic retrieval requests and recorded catalog paths. Outputs: pass/fail
assertions. Side effects: none outside test fixtures. Choose for the Workbench
search extension rather than exercising the user's Case Vault.
"""
from fastapi import FastAPI
from fastapi.testclient import TestClient
import pytest

from app.runtime.retrieval import router
from app.service import retrieval as service


@pytest.fixture
def client():
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


def test_superindex_exact_filters_are_forwarded_before_ranking(client, monkeypatch):
    observed = []
    response = {"request_id": "scope1", "status": "success", "items": [], "legs": []}
    monkeypatch.setattr(service, "spine_json", lambda *args, **kwargs: observed.append((args, kwargs)) or response)
    result = client.post("/api/retrieval/search", json={
        "request_id": "scope1", "query": "example synthetic source",
        "mode": "hybrid", "legs": ["casebible"], "limit": 20,
        "path_prefix": "source-packages/examples/",
        "source_id": "synthetic-collection-1",
    })
    assert result.status_code == 200 and result.json()["status"] == "success"
    args, kwargs = observed[0]
    assert args == ("POST", "/v1/context/retrieve")
    assert kwargs["json"]["legs"] == ["casebible"]
    assert kwargs["json"]["scope"] == {
        "path_prefix": "source-packages/examples/",
        "source_id": "synthetic-collection-1",
    }
    assert "path_prefix" not in kwargs["json"]
    assert "source_id" not in kwargs["json"]


@pytest.mark.parametrize("bad_filter", [
    {"legs": ["intake", "casebible"], "path_prefix": "folder/"},
    {"legs": ["proffer"], "path_prefix": "folder/"},
    {"legs": ["casebible"], "path_prefix": "../folder"},
    {"legs": ["casebible"], "path_prefix": "/absolute"},
    {"legs": ["casebible"], "path_prefix": "b2://other/folder"},
    {"legs": ["casebible"], "path_prefix": "folder/*"},
    {"legs": ["casebible"], "path_prefix": "a\\b"},
    {"legs": ["casebible"], "source_id": "  "},
    {"legs": ["evidence"]},
    {"scope": {"matter_id": "another-case"}},
])
def test_rejects_unsupported_or_untrusted_scope(client, monkeypatch, bad_filter):
    monkeypatch.setattr(service, "spine_json", lambda *args, **kwargs: pytest.fail("invalid request reached Platform"))
    result = client.post("/api/retrieval/search", json={"request_id": "bad1", "query": "synthetic", **bad_filter})
    assert result.status_code == 422


def test_legacy_two_source_search_still_has_empty_scope(client, monkeypatch):
    observed = []
    monkeypatch.setattr(service, "spine_json", lambda *args, **kwargs: observed.append(kwargs) or {
        "status": "success", "items": [], "legs": [],
    })
    result = client.post("/api/retrieval/search", json={"request_id": "default1", "query": "synthetic"})
    assert result.status_code == 200
    assert observed[0]["json"]["legs"] == ["intake", "proffer"]
    assert observed[0]["json"]["scope"] == {}


def test_superindex_hit_can_reopen_catalog_source(client, monkeypatch):
    monkeypatch.setenv("SOURCE_ROOTS_JSON", '[{"id":"fixture","label":"Fixture","url":"b2://fixture/corpus/"}]')
    source = {"legs": ["casebible"], "citation": {"locator": {"vault_key": "corpus/folder/a.txt"}}}
    result = service._attach_file_locations({"items": [source]})
    assert result["items"][0]["location"] == {
        "name": "a.txt", "href": "/sources?root=fixture&file=folder%2Fa.txt",
    }
