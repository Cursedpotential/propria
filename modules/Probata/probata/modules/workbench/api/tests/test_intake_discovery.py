"""Pre-ingest boundary regression checks. Byline: Codex · 2026-09-20."""
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.repo import intake_discovery as repo
from app.runtime.intake_discovery import router
from app.service import intake_discovery as service


@pytest.fixture
def client():
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


def test_catalog_parameters_and_pagination_are_query_bound(monkeypatch):
    calls = []
    def query(sql, parameters):
        calls.append((sql, parameters))
        return [{"rel": "folder/a", "name": "a"}, {"rel": "folder/b", "name": "b"}]
    monkeypatch.setattr(repo, "_query", query)
    result = repo.catalog_page(parent="folder", query="a%_", limit=1)
    assert result["has_more"] and result["items"][0]["source_ref"] is None
    assert "a%_" not in calls[0][0]
    assert calls[0][1][2] == "a\\%\\_%"
    with pytest.raises(repo.DiscoveryError, match="Cursor"):
        repo.catalog_page(parent="other", query="a%_", cursor=result["next_cursor"])


@pytest.mark.parametrize("parent", ["../other", "/absolute", "a/../b", "a\\b"])
def test_parent_reference_rejects_traversal(parent):
    with pytest.raises(repo.DiscoveryError):
        repo.normalize_parent(parent)


def test_missing_catalog_is_not_empty_success(client, monkeypatch):
    monkeypatch.delenv("INTAKE_DISCOVERY_PG_PASSWORD_FILE", raising=False)
    response = client.get("/api/intake/discovery/tree")
    assert response.status_code == 503


def test_unsupported_filters_do_not_query_unscoped_backend(client):
    for params in ({"q": "term", "mode": "hybrid", "parent": "private"},
                   {"q": "term", "atomic_unit": "archive"},
                   {"q": "term", "file_type": "pdf"}):
        assert client.get("/api/intake/discovery/search", params=params).status_code == 422


@pytest.mark.asyncio
async def test_index_uses_intake_filesystem_endpoint_preserves_coverage(monkeypatch):
    calls = []
    async def upstream(path, payload):
        calls.append((path, payload))
        return {"collection": "IntakeCorpus", "hits": [{"object_id": "one", "source_path": "a.zip/chat.json", "filename": "chat.json", "source_id": "s", "document_id": "d", "chunk_id": "c", "text": "excerpt", "score": 1}]}
    monkeypatch.setattr(service, "_upstream", upstream)
    result = await service.search_index("term", "contents", 10)
    assert calls == [("/filesystem/search", {"query": "term", "mode": "keyword", "limit": 10})]
    assert result["coverage"] == "unknown" and not result["complete"]
    assert result["items"][0]["source_ref"] is None


def test_atomic_members_are_historical_not_ingest_refs(monkeypatch):
    monkeypatch.setattr(repo, "_query", lambda sql, params: [{"unit_id": 7, "key": "historical/a"}])
    result = repo.atomic_members(7)
    assert result["items"][0]["key"] == "historical/a"
    assert result["source_links_verified"] is False


def test_substring_search_is_bounded_to_catalog_subtree_and_escaped(monkeypatch):
    calls = []
    monkeypatch.setattr(repo, "_query", lambda sql, params: calls.append((sql, params)) or [])
    result = repo.catalog_page(parent="source", query="Te%rm", mode="filename_substring")
    assert "ILIKE" in calls[0][0] and "Te%rm" not in calls[0][0]
    assert calls[0][1][:5] == ("source", "source/%", "", "%Te\\%rm%", "%Te\\%rm%")
    assert result["query_scope"]["mode"] == "filename_substring"


@pytest.mark.parametrize("cursor", ["!bad", "____", "eyJub3BlIjoxfQ=="])
def test_invalid_cursor_is_client_error(client, cursor):
    assert client.get("/api/intake/discovery/tree", params={"cursor": cursor}).status_code == 422


def test_configured_but_inaccessible_catalog_is_not_ready(monkeypatch):
    monkeypatch.setattr(service, "configured", lambda: True)
    monkeypatch.setattr(service, "index_url", lambda: "")
    def fail(*args):
        raise repo.DiscoveryError("unavailable")
    monkeypatch.setattr(service, "_query", fail)
    result = service.capabilities()
    assert result["catalog_configured"] and not result["tree"]
