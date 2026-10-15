"""Verify native AI search, lineage, named vectors and independent failures using synthetic HTTP only."""
import asyncio
import json
from dataclasses import replace

import httpx
import pytest

from server.api.context_retrieval_routes import ContextQuery
from server.core.retrieval_ai import AI_EMBED_MODEL, ai_hits
from server.core.retrieval_contracts import RetrievalRequest, RetrievalScope
from server.core.retrieval_readers import ContextReaders, ReaderConfig


def row():
    """Return a synthetic native chat chunk with external span provenance; no source data or I/O."""
    provenance = {"source_ref": "b2://fixture/export.json", "provider_version_id": "provider-v1",
                  "source_version_id": "native-v1", "source_sha256": "a" * 64,
                  "historical_source_version_id": "historical-v1", "prepared_ref": "file:///fixture/prepared.json",
                  "decoded_ref": "file:///fixture/decoded.json", "original_ref": "file:///fixture/original.json",
                  "conversation_index": 0, "conversation_id": "native-conversation", "chunk_index": 2,
                  "workflow_id": "context-source:fixture", "segments": [{"record_id": "native-message", "native_locator": "/0/mapping/message"}]}
    return {"body": "A synthetic quoted passage", "conversation_id": "native-conversation", "conversation_title": "Synthetic chat",
            "source_version_id": "native-v1", "source_format": "chatgpt", "chunk_index": 2,
            "record_kind": "ai_conversation_chunk", "embed_model": AI_EMBED_MODEL,
            "provenance": [json.dumps(provenance)], "_additional": {"id": "object-one", "score": 2, "distance": 0.1}}


@pytest.mark.parametrize("mode", ["keyword", "hybrid", "vector"])
def test_ai_queries_its_native_collection_and_keeps_source_coordinates(monkeypatch, mode):
    """Prove native fields and matching named vector without querying a normalized chat store."""
    monkeypatch.setenv("NVIDIA_API_KEY", "synthetic-key")
    calls = []

    def handle(request):
        """Serve only fabricated embedding/search responses and capture bounded requests."""
        body = json.loads(request.content)
        calls.append((str(request.url), body))
        if str(request.url).endswith("/embeddings"):
            return httpx.Response(200, json={"data": [{"embedding": [1.0] * 2048}]})
        return httpx.Response(200, json={"data": {"Get": {"SyntheticAi": [row()]}}})

    config = ReaderConfig(weaviate_url="https://weaviate.invalid", ai_collection="SyntheticAi", ai_vector="text_nim",
                          embed_url="https://embed.invalid/v1", embed_model=AI_EMBED_MODEL, embed_dimensions="2048")
    reader = ContextReaders(config, transport=httpx.MockTransport(handle))
    request = RetrievalRequest(request_id="fixture", query="quoted", mode=mode, legs=("ai",),
                               scope=RetrievalScope(source_version_ids=("native-v1",)))
    hits = asyncio.run(reader.ai(request))
    assert len(hits) == 1 and hits[0].text == row()["body"]
    citation = hits[0].citation
    assert citation.source_version_ids == ("native-v1",)
    assert citation.source_sha256 == "a" * 64
    assert citation.locator["historical_source_version_id"] == "historical-v1"
    assert citation.locator["workflow_id"] == "context-source:fixture"
    assert citation.locator["prepared_ref"] == "file:///fixture/prepared.json"
    assert not citation.normalized_record_ids and citation.thread_id is None
    query = calls[-1][1]["query"]
    assert "SyntheticAi(" in query and "ai_conversation_chunk" in query and "source_version_id" in query
    if mode != "vector":
        assert 'properties: ["search_text"]' in query
    if mode != "keyword":
        assert 'targetVectors: ["text_nim"]' in query
    else:
        assert len(calls) == 1
    assert reader.capabilities()["legs"]["ai"]["configured_modes"] == ["hybrid", "keyword", "vector"]
    broken = ContextReaders(replace(config, embed_model="different-model"), transport=httpx.MockTransport(handle))
    if mode != "keyword":
        with pytest.raises(RuntimeError):
            asyncio.run(broken.ai(request))


@pytest.mark.parametrize("change", [
    {"source_version_id": "other"}, {"record_kind": "message"}, {"provenance": []},
    {"conversation_id": "different"}, {"chunk_index": 999}, {"body": ""},
])
def test_ai_rejects_conflicting_or_missing_native_coordinates(change):
    """Fail malformed native citations instead of substituting a PG human thread."""
    with pytest.raises(ValueError):
        ai_hits("SyntheticAi", [{**row(), **change}], mode="keyword", vector_name=None)


def test_public_query_allows_three_independent_readers():
    """Admit the explicit AI leg alongside existing sources without changing caller authority."""
    assert ContextQuery(query="synthetic", legs=("intake", "proffer", "ai")).internal().legs == ("intake", "proffer", "ai")
