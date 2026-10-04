"""Chunk search and the cycle trail endpoint. Byline: Claude Code · Sonnet 5.5 · 2026-10-02"""

import json

import httpx
import pytest

from casebible_index.chunk_search import ChunkSearchRequest, search_chunks


@pytest.mark.asyncio
async def test_hybrid_search_targets_the_named_vector_and_active_chunks_only():
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["query"] = json.loads(request.content)["query"]
        row = {
            "text": "t",
            "vault_key": "v/k",
            "member_path": "",
            "chunk_index": 2,
            "participant_names": ["a"],
            "start_at": None,
            "_additional": {"id": "00000000-0000-0000-0000-000000000001", "score": "0.9"},
        }
        return httpx.Response(200, json={"data": {"Get": {"CaseBibleChunks20261002": [row]}}})

    result = await search_chunks(
        "http://w.test",
        "CaseBibleChunks20261002",
        ChunkSearchRequest(
            query="school pickup",
            slot="text_nim",
            record_kind="conversation_chunk",
            path_prefix="v/",
        ),
        vector=[1.0, 0.0, 0.5],
        dimensions=3,
        transport=httpx.MockTransport(handler),
    )
    q = seen["query"]
    assert 'targetVectors: ["text_nim"]' in q and "valueBoolean: true" in q
    assert '"conversation_chunk"' in q and '"v/*"' in q
    assert (
        result.hits[0].vault_key == "v/k"
        and result.hits[0].score == 0.9
        and result.hits[0].start_at is None
    )


@pytest.mark.asyncio
async def test_a_wrong_query_dimension_is_refused_before_any_request():
    def handler(request):
        raise AssertionError("no request may be made")

    with pytest.raises(ValueError, match="dimension"):
        await search_chunks(
            "http://w.test",
            "CaseBibleChunks20261002",
            ChunkSearchRequest(query="x"),
            vector=[1.0],
            dimensions=3,
            transport=httpx.MockTransport(handler),
        )
