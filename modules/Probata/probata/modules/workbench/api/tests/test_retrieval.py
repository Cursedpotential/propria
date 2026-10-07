"""Check the combined-search boundary with synthetic results and no external writes.

Byline: Codex · 2026-10-06.
"""
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.repo.spine_client import SpineError
from app.runtime.retrieval import router
from app.service import retrieval as service


@pytest.fixture
def client():
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


def test_search_retains_partial_status_and_all_citations(client, monkeypatch):
    seen = []
    reply = {"status": "partial", "items": [{"citation": {
        "source_version_ids": ["synthetic-a", "synthetic-b"]}}],
        "legs": [{"name": "intake", "status": "failed", "error": "timeout"},
                 {"name": "proffer", "status": "success", "count": 1}]}
    monkeypatch.setattr(service, "spine_json", lambda *args, **kwargs: seen.append((args, kwargs)) or reply)
    result = client.post("/api/retrieval/search", json={"request_id": "test-1", "query": "synthetic"})
    assert result.status_code == 200 and result.json() == reply
    args, kwargs = seen[0]
    assert args == ("POST", "/v1/context/retrieve")
    assert kwargs["json"]["legs"] == ["intake", "proffer"]
    assert kwargs["json"]["scope"] == {}


@pytest.mark.parametrize("extra", [
    {"scope": {"matter_id": "another-case"}}, {"url": "http://untrusted/"},
    {"legs": ["evidence"]}, {"legs": ["intake", "intake"]},
    {"query": "   "}, {"limit": 1000},
])
def test_search_rejects_client_authority_and_invalid_bounds(client, monkeypatch, extra):
    monkeypatch.setattr(service, "spine_json", lambda *args, **kwargs: pytest.fail("rejected request reached Platform"))
    assert client.post("/api/retrieval/search", json={
        "request_id": "test-2", "query": "synthetic", **extra}).status_code == 422


def test_upstream_failure_is_not_empty_search_success(client, monkeypatch):
    def fail(*args, **kwargs):
        raise SpineError("retrieval unavailable", 503)
    monkeypatch.setattr(service, "spine_json", fail)
    response = client.post("/api/retrieval/search", json={"request_id": "test-3", "query": "synthetic"})
    assert response.status_code == 503
    assert response.json() == {"detail": "retrieval unavailable"}


def test_navigation_uses_canonical_matter_scoped_versions_without_trusting_locator_urls(monkeypatch):
    """Resolve original source links from canonical rows, never from an index-supplied URL."""
    version = "11111111-1111-4111-8111-111111111111"
    seen = []
    monkeypatch.setattr(service.imported, "live_matter", lambda: "configured-matter")
    monkeypatch.setattr(service.imported.pg, "versions_to_threads", lambda matter, ids:
                        seen.append((matter, ids)) or [{"id": version, "export_key": "b2://synthetic/original.txt", "conv": "thread"}])
    citation = {"source_version_ids": [version], "locator": {"href": "https://untrusted.example/"}}
    result = service._attach_read_locations({"items": [{"legs": ["proffer"], "citation": citation}]})
    item = result["items"][0]
    assert item["citation"] == citation
    assert seen == [("configured-matter", [version])]
    source = item["navigation"]["sources"][0]
    assert source["href"].startswith("/read?thread=")
    assert source["source_uri"] == "b2://synthetic/original.txt"


def test_navigation_failure_keeps_results_with_explicit_unavailable_state(monkeypatch):
    """A canonical-reader outage does not turn a successful search into missing results."""
    monkeypatch.setattr(service.imported, "live_matter", lambda: "configured-matter")
    def fail(*args):
        raise service.imported.ImportedError("reader unavailable")
    monkeypatch.setattr(service.imported.pg, "versions_to_threads", fail)
    result = service._attach_read_locations({"items": [{"legs": ["proffer"], "text": "synthetic",
        "citation": {"source_version_ids": ["11111111-1111-4111-8111-111111111111"]}}]})
    assert result["items"][0]["text"] == "synthetic"
    assert result["items"][0]["navigation"] == {"status": "unavailable", "sources": []}


def test_relationships_preserves_snapshot_ambiguity(client, monkeypatch):
    """The browser receives every verified snapshot; the bridge never picks a latest one."""
    seen = []
    result = {"source_id": "synthetic-source", "document_id": "synthetic-document",
              "ambiguous": True, "overflow": False, "matches": [{"record_id": "occurrence:a"}, {"record_id": "occurrence:b"}]}
    async def read(path, payload):
        seen.append((path, payload))
        return result
    monkeypatch.setattr(service, "_upstream", read)
    response = client.post("/api/retrieval/relationships", json={"source_id": "synthetic-source", "document_id": "synthetic-document"})
    assert response.status_code == 200 and response.json() == result
    assert seen == [("/filesystem/graph/resolve", {"source_id": "synthetic-source", "document_id": "synthetic-document"})]


def test_relationships_rejects_caller_query_and_endpoint(client, monkeypatch):
    """Only source coordinates cross the graph boundary; raw queries and endpoints cannot."""
    response = client.post("/api/retrieval/relationships", json={"source_id": "synthetic-source",
        "document_id": "synthetic-document", "query": "SELECT * FROM memory", "url": "http://untrusted/"})
    assert response.status_code == 422


def test_intake_location_link_stays_inside_one_configured_root(monkeypatch):
    """Recorded keys open the allowlisted file browser without claiming content verification."""
    monkeypatch.setenv("SOURCE_ROOTS_JSON", '[{"id":"fixture","label":"Synthetic","url":"b2://fixture/corpus/"}]')
    item = {"legs": ["intake"], "citation": {"locator": {"vault_key": "corpus/folder/file.txt"}}}
    output = service._attach_file_locations({"items": [item]})["items"][0]
    assert output["location"] == {"name": "file.txt", "href": "/sources?root=fixture&file=folder%2Ffile.txt"}


@pytest.mark.parametrize("key", ["other/file.txt", "corpus/../secret.txt", "b2://foreign/corpus/file.txt",
    "https://[", "b2://fixture/corpus/file.txt?versionId=retained", "corpus/file.txt#other"])
def test_intake_location_does_not_guess_or_drop_version_identity(monkeypatch, key):
    """Unknown roots, traversal and exact versions never become an unversioned browse link."""
    monkeypatch.setenv("SOURCE_ROOTS_JSON", '[{"id":"fixture","label":"Synthetic","url":"b2://fixture/corpus/"}]')
    output = service._attach_file_locations({"items": [{"legs": ["intake"], "citation": {"locator": {"vault_key": key}}}]})
    assert "location" not in output["items"][0]


def test_intake_location_rejects_ambiguous_bucket_mapping(monkeypatch):
    """A key alone cannot silently choose between buckets with the same prefix."""
    monkeypatch.setenv("SOURCE_ROOTS_JSON", '[{"id":"a","label":"A","url":"b2://a/corpus/"},{"id":"b","label":"B","url":"b2://b/corpus/"}]')
    output = service._attach_file_locations({"items": [{"legs": ["intake"], "citation": {"locator": {"vault_key": "corpus/file.txt"}}}]})
    assert "location" not in output["items"][0]
