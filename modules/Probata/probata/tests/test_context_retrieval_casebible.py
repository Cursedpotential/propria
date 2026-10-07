"""Synthetic tests for the opt-in CaseBible context reader; no live index, corpus or provider is queried."""

from __future__ import annotations

import asyncio
import json

import httpx
import pytest

from server.api.context_retrieval_routes import ContextQuery
from server.core.retrieval_composition import collect_legs
from server.core.retrieval_casebible import casebible_hits, casebible_where
from server.core.retrieval_contracts import RetrievalRequest, RetrievalScope
from server.core.retrieval_readers import ContextReaders, ReaderConfig


COLLECTION = "SyntheticCaseBibleChunks"
OBJECT_ID = "7e57e57e-0000-4000-8000-000000000001"


def _chunk(**overrides: object) -> dict[str, object]:
    row: dict[str, object] = {
        "text": "synthetic chunk text",
        "source_id": "source-41",
        "version_id": "version-9",
        "document_id": "document-5",
        "content_hash": "a" * 64,
        "sha1": "b" * 40,
        "chunk_index": 2,
        "chunk_ordinal": 2,
        "char_start": 128,
        "char_end": 148,
        "record_kind": "document_chunk",
        "thread_id": "",
        "vault_key": "vault/source/document.pdf",
        "member_path": "source/document.pdf",
        "catalog_path": "source/document.pdf",
        "filename": "document.pdf",
        "chunker_version": "recursive-v3",
        "resolution": "resolved",
        "origin_system": "superindex",
        "active": True,
        "_additional": {"id": OBJECT_ID, "score": 4.25},
    }
    row.update(overrides)
    return row


def _request(mode: str, *, scope: RetrievalScope | None = None) -> RetrievalRequest:
    return RetrievalRequest(
        request_id="casebible-test",
        query="synthetic query",
        mode=mode,
        scope=scope or RetrievalScope(),
        legs=("casebible",),
        per_leg_limit=5,
    )


def test_adapter_preserves_native_identity_hash_version_and_locators() -> None:
    hit = casebible_hits(COLLECTION, [_chunk()], mode="keyword", vector_name=None)[0]

    assert hit.citation.collection == COLLECTION
    assert hit.citation.object_id == OBJECT_ID
    assert hit.citation.source_id == "source-41"
    assert hit.citation.document_id == "document-5"
    assert hit.citation.chunk_index == 2
    assert hit.citation.content_hash == "a" * 64
    assert hit.citation.locator == {
        "version_id": "version-9",
        "sha1": "b" * 40,
        "vault_key": "vault/source/document.pdf",
        "member_path": "source/document.pdf",
        "catalog_path": "source/document.pdf",
        "filename": "document.pdf",
        "chunk_ordinal": "2",
        "char_start": "128",
        "char_end": "148",
        "chunker_version": "recursive-v3",
        "resolution": "resolved",
        "origin_system": "superindex",
    }


@pytest.mark.parametrize("field", ["source_id", "version_id", "document_id", "content_hash"])
def test_adapter_rejects_missing_source_identity(field: str) -> None:
    with pytest.raises(ValueError, match=field):
        casebible_hits(COLLECTION, [_chunk(**{field: ""})], mode="keyword", vector_name=None)


def test_adapter_rejects_inactive_rows_and_scores_vector_by_distance() -> None:
    with pytest.raises(ValueError, match="inactive"):
        casebible_hits(COLLECTION, [_chunk(active=False)], mode="keyword", vector_name=None)

    row = _chunk(_additional={"id": OBJECT_ID, "distance": 0.25})
    hit = casebible_hits(COLLECTION, [row], mode="vector", vector_name="text_nim")[0]
    assert hit.score == -0.25
    assert hit.citation.vector_name == "text_nim"


def test_native_where_clause_prefilters_active_and_exact_scope() -> None:
    where = casebible_where(
        RetrievalScope(
            source_id='source"-41',
            thread_id="thread-3",
            path_prefix="vault/source/",
        )
    )

    assert '{path: ["active"], operator: Equal, valueBoolean: true}' in where
    assert 'path: ["source_id"], operator: Equal, valueText: "source\\"-41"' in where
    assert 'path: ["version_id"]' not in where
    assert 'path: ["thread_id"], operator: Equal, valueText: "thread-3"' in where
    assert 'path: ["vault_key"], operator: Like, valueText: "vault/source/*"' in where


@pytest.mark.parametrize("wildcard", ["*", "?"])
def test_casebible_prefix_wildcards_fail_request_validation_before_query(wildcard: str) -> None:
    prefix = f"vault/source{wildcard}/"
    public = ContextQuery(query="synthetic query", legs=("casebible",), scope={"path_prefix": prefix})
    with pytest.raises(ValueError, match="CaseBible path_prefix wildcards are unsupported"):
        public.internal()

    with pytest.raises(ValueError, match="wildcards are unsupported"):
        casebible_where(RetrievalScope(path_prefix=prefix))


def test_reader_uses_each_supported_mode_and_explicit_named_vector() -> None:
    queries: list[str] = []

    def handle(request: httpx.Request) -> httpx.Response:
        payload = json.loads(request.content)
        query = payload["query"]
        queries.append(query)
        row = _chunk()
        if "nearVector" in query:
            row["_additional"] = {"id": OBJECT_ID, "distance": 0.2}
        return httpx.Response(200, json={"data": {"Get": {COLLECTION: [row]}}})

    config = ReaderConfig(
        weaviate_url="https://weaviate.invalid",
        casebible_collection=COLLECTION,
        embed_url="https://nim.invalid/v1",
        embed_model="nvidia/nemotron-3-embed-1b",
        embed_dimensions="2048",
    )
    readers = ContextReaders(config, transport=httpx.MockTransport(handle))

    async def embed(_: RetrievalRequest) -> list[float]:
        return [0.25] * 2048

    readers.embedding = embed  # type: ignore[method-assign]

    async def run() -> None:
        for mode in ("keyword", "hybrid", "vector"):
            hits = await readers.casebible(_request(mode))
            assert len(hits) == 1

    asyncio.run(run())

    assert 'active"], operator: Equal, valueBoolean: true' in queries[0]
    assert "bm25:" in queries[0] and "properties:" not in queries[0]
    assert "hybrid:" in queries[1] and "alpha: 0.7" in queries[1]
    assert "properties:" not in queries[1]
    assert 'targetVectors: ["text_nim"]' in queries[1]
    assert "nearVector:" in queries[2] and 'targetVectors: ["text_nim"]' in queries[2]


def test_absent_collection_fails_leg_while_configured_empty_collection_succeeds() -> None:
    absent = ContextReaders(
        ReaderConfig(weaviate_url="https://weaviate.invalid", casebible_collection=COLLECTION),
        transport=httpx.MockTransport(
            lambda _: httpx.Response(
                200,
                json={"errors": [{"message": "collection is absent"}]},
            )
        ),
    )
    empty = ContextReaders(
        ReaderConfig(weaviate_url="https://weaviate.invalid", casebible_collection=COLLECTION),
        transport=httpx.MockTransport(
            lambda _: httpx.Response(
                200,
                json={"data": {"Get": {COLLECTION: []}}},
            )
        ),
    )

    async def run() -> None:
        failed = await collect_legs(_request("keyword"), absent.legs())
        empty_result = await collect_legs(_request("keyword"), empty.legs())
        assert failed["outcomes"][0].status == "failed"
        assert empty_result["outcomes"][0].status == "success"
        assert empty_result["outcomes"][0].count == 0

    asyncio.run(run())


def test_unconfigured_collection_is_reported_and_route_allowlists_opt_in_leg() -> None:
    capability = ContextReaders(ReaderConfig()).capabilities()["legs"]["casebible"]
    assert capability["state"] == "unconfigured"
    assert capability["configured_modes"] == []
    assert capability["collection_configured"] is False

    query = ContextQuery(query="synthetic query", legs=("casebible",))
    assert query.internal().legs == ("casebible",)


def test_collection_has_no_default_and_embedding_modes_require_native_profile(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("CONTEXT_RETRIEVAL_CASEBIBLE_COLLECTION", raising=False)
    assert ReaderConfig.from_environment().casebible_collection == ""

    compatible = ContextReaders(
        ReaderConfig(
            weaviate_url="https://weaviate.invalid",
            casebible_collection=COLLECTION,
            embed_url="https://nim.invalid/v1",
            embed_model="nvidia/nemotron-3-embed-1b",
            embed_dimensions="2048",
        )
    ).casebible_capability()
    incompatible = ContextReaders(
        ReaderConfig(
            weaviate_url="https://weaviate.invalid",
            casebible_collection=COLLECTION,
            embed_url="https://nim.invalid/v1",
            embed_model="other/model",
            embed_dimensions="2048",
        )
    ).casebible_capability()

    assert compatible["configured_modes"] == ["keyword", "hybrid", "vector"]
    assert compatible["vector_name"] == "text_nim"
    assert compatible["vector_dimensions"] == 2048
    assert incompatible["configured_modes"] == ["keyword"]
