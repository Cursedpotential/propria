"""Verify configured retrieval ports and Platform routes with synthetic HTTP, without databases or provider calls.

Inputs are fabricated API/GraphQL replies; outputs prove scope, auth, source preservation and vector contract behavior.
All remote requests use MockTransport. No real matter, service, corpus, workflow or index is touched.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import asyncio
import json
from dataclasses import replace

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from server.api.context_retrieval_routes import register_context_retrieval_routes
from server.core.retrieval_composition import retrieve_context
from server.core.retrieval_contracts import RetrievalRequest, RetrievalScope
from server.core.retrieval_readers import ContextReaders, ReaderConfig

MATTER = "11111111-1111-4111-8111-111111111111"
FOREIGN_MATTER = "22222222-2222-4222-8222-222222222222"
CONFIG = ReaderConfig(
    intake_url="https://intake.invalid", intake_collection="SyntheticIntake", intake_vector="text_vector",
    weaviate_url="https://weaviate.invalid", proffer_collection="SyntheticProffer", proffer_vector="text_nim",
    proffer_matter_id=MATTER, embed_url="https://embed.invalid/v1", embed_model="synthetic-model", embed_dimensions="2",
)


def intake_payload() -> dict:
    """Return a fabricated existing Intake response; preserve fixture IDs without genuine corpus data."""
    return {"collection": "SyntheticIntake", "target_vector": "text_vector", "hits": [{
        "object_id": "intake-object", "source_id": "intake-source", "document_id": "intake-doc", "chunk_id": "chunk",
        "source_path": "synthetic/source", "vault_key": "synthetic/vault", "resolution": "catalog", "text": "same", "score": 2,
    }]}


def proffer_row(*, matter: str = MATTER) -> dict:
    """Return a fabricated call-log projection with only actual version coordinates; perform no I/O."""
    return {"matter_id": matter, "text": "same", "source_version_id": "proffer-version", "record_kind": "call_log_file",
            "_additional": {"id": "proffer-object", "score": 2, "distance": 0.1}}


def readers_fixture(monkeypatch, *, config: ReaderConfig = CONFIG, foreign: bool = False):
    """Build mock-only reader ports and captured requests; no real credentials, upstream or store are used."""
    monkeypatch.setenv("NVIDIA_API_KEY", "synthetic-key")
    calls = []

    def handle(request: httpx.Request) -> httpx.Response:
        """Respond to one synthetic read request; capture query filters without contacting a service."""
        body = json.loads(request.content)
        calls.append((str(request.url), body))
        if request.url.host == "embed.invalid":
            assert body["input_type"] == "query" and body["model"] == "synthetic-model"
            return httpx.Response(200, json={"data": [{"embedding": [0.2, 0.3]}]})
        if request.url.host == "intake.invalid":
            payload = intake_payload()
            if body["mode"] == "vector":
                assert body["vector"] == [0.2, 0.3]
                payload["hits"][0].update(score=None, distance=0.2)
            else:
                assert "vector" not in body
            return httpx.Response(200, json=payload)
        query = body["query"]
        if "SyntheticProffer(" in query:
            assert 'path: ["matter_id"]' in query and MATTER in query
            row = proffer_row(matter=FOREIGN_MATTER if foreign else MATTER)
            # Pure-vector providers need not include score.
            if "nearVector" in query:
                del row["_additional"]["score"]
            return httpx.Response(200, json={"data": {"Get": {"SyntheticProffer": [row]}}})
        raise AssertionError("unexpected upstream read")

    return ContextReaders(config, transport=httpx.MockTransport(handle)), calls


def compose(readers: ContextReaders, **kwargs):
    """Invoke the real libraries with synthetic HTTP ports; return composed results without external effects."""
    request = RetrievalRequest(request_id="synthetic", query="same", legs=("intake", "proffer"), **kwargs)
    return asyncio.run(retrieve_context(request, readers.legs()))


@pytest.mark.parametrize("mode", ["keyword", "hybrid", "vector"])
def test_modes_preserve_two_origins_and_proffer_scope(monkeypatch, mode):
    """Read equal text from two origins in every mode; assert named-vector queries and pre-ranking Proffer matter filters."""
    readers, calls = readers_fixture(monkeypatch)
    result = compose(readers, mode=mode)
    assert result.status == "success" and len(result.items) == 2
    assert {item.citation.collection for item in result.items} == {"SyntheticIntake", "SyntheticProffer"}
    intake = next(item for item in result.items if item.citation.source_id)
    assert intake.citation.locator["vault_key"] == "synthetic/vault"
    queries = [body["query"] for url, body in calls if url.endswith("/v1/graphql")]
    if mode == "vector":
        assert len(queries) == 1
        assert all("nearVector:" in query and "id distance" in query and "id score" not in query for query in queries)
        assert any('targetVectors: ["text_nim"]' in query for query in queries)
        assert any(url.endswith("/filesystem/search") and body["mode"] == "vector" for url, body in calls)
    if mode == "keyword":
        assert not any(url.endswith("/embeddings") for url, _ in calls)


def test_scope_failure_is_local_and_version_filter_precedes_limit(monkeypatch):
    """Request exact source versions; Intake fails unsupported scope while Proffer applies actual array/single-version filters."""
    readers, calls = readers_fixture(monkeypatch)
    result = compose(readers, scope=RetrievalScope(source_version_ids=("proffer-version",)))
    assert result.status == "partial"
    assert result.legs[0].error == "unsupported_scope"
    assert result.legs[1].status == "success"
    assert len(calls) == 1
    assert 'operator: ContainsAny, valueText: ["proffer-version"]' in calls[0][1]["query"]
    assert 'path: ["source_version_id"], operator: Equal' in calls[0][1]["query"]


def test_cross_case_reply_and_missing_matter_fail_only_proffer(monkeypatch):
    """Reject a foreign Proffer row and an unconfigured matter; global Intake stays readable without broader Proffer fallback."""
    readers, _ = readers_fixture(monkeypatch, foreign=True)
    result = compose(readers)
    assert result.status == "partial" and result.legs[1].error == "invalid_result"
    assert [item.citation.collection for item in result.items] == ["SyntheticIntake"]
    readers, calls = readers_fixture(monkeypatch, config=replace(CONFIG, proffer_matter_id=""))
    result = compose(readers)
    assert result.status == "partial" and result.legs[1].error == "unavailable"
    assert all(url.endswith("/filesystem/search") for url, _ in calls)


def test_collection_mismatch_and_missing_embedding_do_not_fallback(monkeypatch):
    """Reject an unallowlisted Intake response and missing vector config; neighboring keyword data remains separately accounted."""
    readers, _ = readers_fixture(monkeypatch, config=replace(CONFIG, intake_collection="WrongCollection"))
    result = compose(readers)
    assert result.status == "partial" and result.legs[0].error == "invalid_result"
    readers, calls = readers_fixture(monkeypatch, config=replace(CONFIG, embed_model=""))
    result = compose(readers, mode="vector")
    assert result.status == "failed" and not calls


def client_fixture(monkeypatch):
    """Register routes on a test-only app with mocked auth/HTTP; output client and captured reads without a local service."""
    monkeypatch.setattr("server.api.platform_auth.read_platform_api_bearer", lambda: "synthetic-owner")
    readers, calls = readers_fixture(monkeypatch)
    app = FastAPI()
    register_context_retrieval_routes(app, readers=readers)
    return TestClient(app), calls


def test_routes_require_owner_auth_and_capabilities_make_no_reads(monkeypatch):
    """Require the mounted-owner contract on every route; capabilities expose no endpoints/secrets and do not probe stores."""
    client, calls = client_fixture(monkeypatch)
    for path in ("/v1/context/retrieve", "/v1/context/retrieve-flow"):
        assert client.post(path, json={}).status_code == 401
    assert client.get("/v1/context/retrieval-capabilities").status_code == 401
    response = client.get("/v1/context/retrieval-capabilities", headers={"Authorization": "Bearer synthetic-owner"})
    assert response.status_code == 200 and not calls
    body = response.json()
    assert body["legs"]["proffer"]["scope_policy"] == "server_matter"
    assert body["graph"]["available"] is False and body["evidence"]["available"] is False
    assert "https://" not in response.text and "synthetic-key" not in response.text


def test_default_query_and_flow_reject_caller_authority(monkeypatch):
    """Exercise direct/default and compact/vector routes; reject caller matter, credentials, unknown legs and arbitrary upstreams."""
    client, calls = client_fixture(monkeypatch)
    headers = {"Authorization": "Bearer synthetic-owner"}
    response = client.post("/v1/context/retrieve", json={"query": "same"}, headers=headers)
    assert response.status_code == 200 and response.json()["status"] == "success"
    assert [leg["name"] for leg in response.json()["legs"]] == ["intake", "proffer"]
    calls.clear()
    for body in ({"query": "same", "matter_id": FOREIGN_MATTER},
                 {"query": "same", "scope": {"matter_id": FOREIGN_MATTER}},
                 {"query": "same", "legs": ["evidence"]},
                 {"query": "same", "url": "https://untrusted.invalid"},
                 {"query": "same", "credentials": "forbidden"}):
        assert client.post("/v1/context/retrieve", json=body, headers=headers).status_code == 422
    assert not calls
    flow = {"flow": "read_retrieval", "request_id": "synthetic-flow",
            "inputs": {"query": "same", "leg_names": "intake,proffer", "mode": "vector"}}
    response = client.post("/v1/context/retrieve-flow", json=flow, headers=headers)
    assert response.status_code == 200
    assert response.json()["outputs"]["mode"] == "vector"
    assert response.json()["outputs"]["request_id"] == "synthetic-flow"
    assert all("text" not in item for item in response.json()["outputs"]["items"])
    calls.clear()
    flow["inputs"]["scope_json"] = json.dumps({"matter_id": FOREIGN_MATTER})
    assert client.post("/v1/context/retrieve-flow", json=flow, headers=headers).status_code == 422
    assert not calls


def test_dependency_and_body_bounds_fail_before_upstreams(monkeypatch):
    """Simulate missing optional libraries and excessive input; return explicit errors without upstream or installation effects."""
    client, calls = client_fixture(monkeypatch)
    headers = {"Authorization": "Bearer synthetic-owner"}
    monkeypatch.setattr("server.api.context_retrieval_routes.dependencies_available", lambda: False)
    assert client.post("/v1/context/retrieve", json={"query": "same"}, headers=headers).status_code == 503
    assert client.post("/v1/context/retrieve", content="x" * 32769, headers=headers).status_code == 413
    assert not calls
