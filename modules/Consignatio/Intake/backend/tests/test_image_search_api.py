"""Image search lanes, the kind filter and POST /images/search. Byline: Claude Code · Sonnet 5.5 ·
2026-10-03

Weaviate is an httpx MockTransport that records the GraphQL it receives; no network, synthetic rows
only.
"""

import json

import httpx
import pytest
from fastapi.testclient import TestClient

from casebible_index import image_search_api
from casebible_index.image_embedders import ImageEmbedders
from casebible_index.image_search import ImageSearchError, WeaviateImageSearcher
from casebible_index.image_search_api import ImageSearchRequest

ROW = {
    "source_id": "consignatio-vault-v1",
    "source_path": "v/Screenshot_a.png",
    "filename": "Screenshot_a.png",
    "content_sha256": "aa",
    "original_time": "2026-01-02T03:04:05Z",
    "original_time_source": "filename",
    "original_time_confidence": "medium",
    "original_time_conflict": False,
    "device": "",
    "software": "",
    "gps": "",
    "is_screenshot": True,
    "ocr_text": "rent receipt paid on the 3rd",
    "provider": "b2",
    "bucket": "salem-data",
    "vault_key": "v/Screenshot_a.png",
    "image_kind": "screenshot",
    "source_kind": "image",
    "page": 0,
    "identity": "aa",
}


def weaviate(queries: list[str]) -> httpx.AsyncBaseTransport:
    def handler(request: httpx.Request) -> httpx.Response:
        query = json.loads(request.content)["query"]
        queries.append(query)
        rows = [
            {
                **ROW,
                "_additional": {
                    "id": "11111111-1111-1111-1111-111111111111",
                    "distance": 0.25,
                    "score": "1.5",
                },
            }
        ]
        return httpx.Response(200, json={"data": {"Get": {"IntakeImageV1": rows}}})

    return httpx.MockTransport(handler)


def embedders(single=True, jina=True) -> ImageEmbedders:
    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content)
        if "jina" in str(request.url):
            return httpx.Response(200, json={"data": [{"embeddings": [[0.1] * 128, [0.2] * 128]}]})
        return (
            httpx.Response(200, json={"data": [{"embedding": [0.3] * 2048, "index": 0}]})
            if body
            else httpx.Response(400)
        )

    return ImageEmbedders(
        httpx.AsyncClient(transport=httpx.MockTransport(handler)),
        "nim",
        "k" if single else "",
        "j" if jina else None,
    )


@pytest.mark.asyncio
async def test_each_enabled_lane_is_queried_and_the_kind_filter_reaches_every_query():
    queries: list[str] = []
    searcher = WeaviateImageSearcher("http://w.test", "IntakeImageV1")
    hits = await searcher.search(
        "rent receipt",
        limit=5,
        mode="hybrid",
        embedders=embedders(),
        transport=weaviate(queries),
        kind="screenshot",
        lanes=("image_maxsim", "image_single", "image_clip"),
    )
    joined = "\n".join(queries)
    assert (
        'targetVectors: ["image_maxsim"]' in joined and 'targetVectors: ["image_single"]' in joined
    )
    assert 'nearText: {concepts: ["rent receipt"], targetVectors: ["image_clip"]}' in joined
    assert 'bm25: {query: "rent receipt", properties: ["ocr_text"]}' in joined
    assert all(
        'path: ["image_kind"], operator: Equal, valueText: "screenshot"' in q for q in queries
    )
    assert "image_colqwen" not in joined
    by = {h.matched_by: h for h in hits}
    assert {"ocr_literal", "maxsim", "single", "clip"} == set(by) or "ocr_literal" in by
    top = hits[0]
    assert (top.provider, top.bucket, top.vault_key, top.image_kind) == (
        "b2",
        "salem-data",
        "v/Screenshot_a.png",
        "screenshot",
    )


@pytest.mark.asyncio
async def test_a_maxsim_only_index_never_calls_the_single_vector_embedder():
    queries: list[str] = []
    searcher = WeaviateImageSearcher("http://w.test", "IntakeImageV1")
    await searcher.search(
        "rent",
        limit=3,
        mode="hybrid",
        embedders=embedders(single=False),
        transport=weaviate(queries),
        lanes=("image_maxsim",),
    )
    assert any('targetVectors: ["image_maxsim"]' in q for q in queries)
    assert not any("image_single" in q for q in queries)


@pytest.mark.asyncio
async def test_unknown_kind_is_refused():
    searcher = WeaviateImageSearcher("http://w.test", "IntakeImageV1")
    with pytest.raises(ImageSearchError):
        await searcher.search(
            "x", limit=3, mode="keyword", embedders=None, transport=weaviate([]), kind="video"
        )


def test_images_search_route_reports_lanes_and_503s_without_configuration(monkeypatch, tmp_path):
    from casebible_index.api import create_api

    monkeypatch.setenv("CASEBIBLE_OUTPUT_DIR", str(tmp_path))
    monkeypatch.setenv("INTAKE_SOURCE_MODE", "catalog")
    monkeypatch.setenv("INTAKE_VAULT_BUCKET", "salem-data")
    for name in ("INTAKE_IMAGES_WEAVIATE_URL", "INTAKE_WEAVIATE_URL"):
        monkeypatch.delenv(name, raising=False)
    client = TestClient(create_api())
    assert client.post("/images/search", json={"query": "rent"}).status_code == 503

    async def fake(request: ImageSearchRequest):
        return image_search_api.ImageSearchResponse(
            query=request.query,
            collection="IntakeImageV1",
            kind=request.kind,
            lanes=["image_maxsim"],
            hits=[],
        )

    monkeypatch.setattr("casebible_index.api.search_images", fake)
    answer = client.post("/images/search", json={"query": "rent", "kind": "screenshot"})
    assert answer.status_code == 200 and answer.json()["kind"] == "screenshot"
    assert client.post("/images/search", json={"query": "rent", "kind": "video"}).status_code == 422
