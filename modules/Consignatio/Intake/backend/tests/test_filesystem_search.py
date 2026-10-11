import json

import httpx
import pytest

from casebible_index.filesystem_search import (
    FilesystemSearchError,
    FilesystemSearchRequest,
    WeaviateFilesystemSearcher,
    WeaviateSearchConfig,
)


def config(**changes):
    return WeaviateSearchConfig(
        **{
            "url": "https://search.example",
            "collection": "IntakeTestChunks",
            "target_vector": "text_nim",
            "dimensions": 3,
            **changes,
        }
    )


@pytest.mark.asyncio
@pytest.mark.parametrize("mode", ["keyword", "hybrid", "vector"])
async def test_search_modes_have_distinct_weaviate_operators_and_truthful_metrics(mode):
    def handler(request):
        query = json.loads(request.content)["query"]
        assert request.url.path == "/v1/graphql"
        assert "limit: 20" in query
        assert 'where: {path: ["active"], operator: Equal, valueBoolean: true}' in query
        assert all(
            field in query
            for field in (
                "source_id source_path vault_key resolution document_id chunk_id filename text",
            )
        )
        assert ("bm25:" in query) == (mode == "keyword")
        assert ("hybrid:" in query) == (mode == "hybrid")
        assert ("nearVector:" in query) == (mode == "vector")
        assert ('targetVectors: ["text_nim"]' in query) == (mode in {"hybrid", "vector"})
        assert ("_additional {id distance}" in query) == (mode == "vector")
        assert ("_additional {id score}" in query) == (mode != "vector")
        additional = (
            {"id": "object-1", "distance": "0.14"}
            if mode == "vector"
            else {"id": "object-1", "score": "0.8"}
        )
        return httpx.Response(
            200,
            json={
                "data": {
                    "Get": {
                        "IntakeTestChunks": [
                            {
                                "source_id": "r2-raw",
                                "source_path": "V:\\raw\\note.txt",
                                "document_id": "doc-1",
                                "chunk_id": "chunk-1",
                                "filename": "note.txt",
                                "text": "Matching excerpt",
                                "_additional": additional,
                            }
                        ]
                    }
                }
            },
        )

    request_vector = [1, 0, 0] if mode == "vector" else None
    result = await WeaviateFilesystemSearcher(config()).search(
        FilesystemSearchRequest(query='a "quoted" query', mode=mode, vector=request_vector),
        vector=[1, 0, 0] if mode == "hybrid" else None,
        transport=httpx.MockTransport(handler),
    )
    assert result.coverage == "unknown"
    assert result.hits[0].source_path == "V:\\raw\\note.txt"
    if mode == "vector":
        assert result.target_vector == "text_nim"
        assert result.hits[0].score is None
        assert result.hits[0].distance == 0.14
    else:
        assert result.target_vector == ("text_nim" if mode == "hybrid" else None)
        assert result.hits[0].score == 0.8
        assert result.hits[0].distance is None


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "payload",
    [
        {"errors": [{"message": "sensitive provider response"}]},
        {"data": {"Get": {"IntakeTestChunks": None}}},
        {"data": {}},
    ],
)
async def test_failure_is_not_empty_success_or_provider_leak(payload):
    with pytest.raises(FilesystemSearchError) as error:
        await WeaviateFilesystemSearcher(config()).search(
            FilesystemSearchRequest(query="test", mode="keyword"),
            transport=httpx.MockTransport(lambda _: httpx.Response(200, json=payload)),
        )
    assert "sensitive" not in str(error.value)


@pytest.mark.asyncio
@pytest.mark.parametrize("vector", [[1], [0, 0, 0], [float("nan"), 0, 1]])
async def test_bad_vectors_fail_before_network(vector):
    with pytest.raises(ValueError):
        await WeaviateFilesystemSearcher(config()).search(
            FilesystemSearchRequest(query="test", mode="vector", vector=vector),
            vector=vector,
        )


@pytest.mark.parametrize(
    "payload",
    [
        {"query": "test", "mode": "vector"},
        {"query": "test", "mode": "keyword", "vector": [1, 0, 0]},
        {"query": "test", "mode": "hybrid", "vector": [1, 0, 0]},
    ],
)
def test_external_vector_is_required_only_for_pure_vector_mode(payload):
    with pytest.raises(ValueError):
        FilesystemSearchRequest(**payload)


@pytest.mark.parametrize(
    "changes",
    [
        {"collection": "Bad) { dangerous"},
        {"url": "https://key:secret@example.com"},
        {"target_vector": ""},
        {"url": "file:///tmp/search"},
    ],
)
def test_invalid_configuration_rejected(changes):
    with pytest.raises(ValueError):
        WeaviateFilesystemSearcher(config(**changes))
