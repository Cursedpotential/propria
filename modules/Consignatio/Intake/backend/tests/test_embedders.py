"""Embedding slots. Byline: Claude Code · Sonnet 5.5 · 2026-10-02"""

import json

import httpx
import pytest

from casebible_index.embedders import (
    BLANK_PLACEHOLDER,
    EmbedError,
    SlotEmbedder,
    VectorSlot,
    guard_input,
    load_slots,
)


def slot(**kw) -> VectorSlot:
    base = dict(
        name="text_nim",
        model="m",
        dimensions=3,
        base_url="https://nim.test/v1",
        key_secret="K",
        batch_size=2,
        max_retries=1,
    )
    base.update(kw)
    return VectorSlot(**base)


def test_guards_rewrite_data_image_and_placeholder_blanks_without_dropping():
    assert guard_input("see data:image/png;base64,xx") == "see data: image/png;base64,xx"
    assert guard_input("   ") == BLANK_PLACEHOLDER
    assert guard_input("") == BLANK_PLACEHOLDER
    assert len(guard_input("a" * 9000)) == 8000


def test_default_slot_is_the_nim_embedder_and_legal_needs_a_model():
    [only] = load_slots({})
    assert (only.name, only.model, only.dimensions, only.style) == (
        "text_nim",
        "nvidia/nemotron-3-embed-1b",
        2048,
        "nim",
    )
    with pytest.raises(ValueError, match="INTAKE_VECTOR_LEGAL_MODEL"):
        load_slots({"INTAKE_VECTOR_SLOTS": "text_nim,legal"})
    both = load_slots(
        {"INTAKE_VECTOR_SLOTS": "text_nim,legal", "INTAKE_VECTOR_LEGAL_MODEL": "voyage-law-2"}
    )
    assert (
        [s.name for s in both] == ["text_nim", "legal"]
        and both[1].style == "voyage"
        and both[1].dimensions == 1024
    )
    with pytest.raises(ValueError, match="Unknown vector slot"):
        load_slots({"INTAKE_VECTOR_SLOTS": "mystery"})
    with pytest.raises(ValueError, match="repeats"):
        load_slots({"INTAKE_VECTOR_SLOTS": "text_nim,text_nim"})


def test_request_bodies_follow_each_providers_shape():
    nim = slot().request_body(["a"])
    assert (
        nim["input_type"] == "passage"
        and nim["truncate"] == "END"
        and nim["encoding_format"] == "float"
    )
    voyage = slot(name="legal", style="voyage", document_input_type=None).request_body(["a"])
    assert voyage["input_type"] == "document" and "encoding_format" not in voyage


@pytest.mark.asyncio
async def test_batches_guarded_inputs_in_order_with_one_request_per_batch():
    seen = []

    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content)
        seen.append(body["input"])
        return httpx.Response(
            200,
            json={
                "data": [
                    {"index": i, "embedding": [1.0, float(i), 0.5]}
                    for i, _ in enumerate(body["input"])
                ]
            },
        )

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
        embedder = SlotEmbedder(slot(), client, api_key="k")
        vectors, statuses = await embedder.embed(["data:image/x", "  ", "c"])
    assert seen == [["data: image/x", BLANK_PLACEHOLDER], ["c"]]
    assert len(vectors) == 3 and statuses == ["ok"] * 3 and embedder.requests == 2


@pytest.mark.asyncio
async def test_a_refused_text_is_isolated_by_bisection_and_marked_truncated_or_failed():
    def handler(request: httpx.Request) -> httpx.Response:
        texts = json.loads(request.content)["input"]
        if any(len(t) > 300 for t in texts):
            return httpx.Response(400, json={"error": "too many tokens"})
        return httpx.Response(
            200,
            json={
                "data": [{"index": i, "embedding": [1.0, 2.0, 3.0]} for i, _ in enumerate(texts)]
            },
        )

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
        embedder = SlotEmbedder(slot(), client, api_key="k")
        vectors, statuses = await embedder.embed(["short", "x" * 2000])
    assert statuses == ["ok", "truncated"] and all(any(v) for v in vectors)


@pytest.mark.asyncio
async def test_an_outage_is_not_bisected_it_raises_for_the_activity_to_retry():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        return httpx.Response(503)

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
        embedder = SlotEmbedder(slot(batch_size=8, max_retries=1), client, api_key="k")
        with pytest.raises(EmbedError):
            await embedder.embed(["a", "b", "c", "d"])
    assert calls["n"] == 2  # one batch, two attempts, no bisection


@pytest.mark.asyncio
async def test_a_wrong_dimension_or_zero_vector_is_rejected():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"data": [{"index": 0, "embedding": [0.0, 0.0, 0.0]}]})

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
        embedder = SlotEmbedder(slot(), client, api_key="k")
        vectors, statuses = await embedder.embed(["a"])
    assert statuses == ["failed"] and vectors == [[0.0, 0.0, 0.0]]
