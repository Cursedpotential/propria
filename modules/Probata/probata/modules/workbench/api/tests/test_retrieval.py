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
