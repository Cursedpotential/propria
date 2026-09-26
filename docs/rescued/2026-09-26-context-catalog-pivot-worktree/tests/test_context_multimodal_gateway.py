from __future__ import annotations

import json

import httpx
import pytest

from server.core.multimodal_gateway import (
    EMBEDDING_DIMENSIONS,
    ContextMultimodalGateway,
    EmbeddingRequest,
    ExtractionRequest,
    GatewayRoute,
    MultimodalGatewayError,
    RerankCandidate,
)


def _route(alias: str) -> GatewayRoute:
    return GatewayRoute(config={"provider": "openai", "api_key": "test-provider-key"}, model_alias=alias)


def _gateway(handler: httpx.MockTransport) -> ContextMultimodalGateway:
    client = httpx.AsyncClient(transport=handler)
    return ContextMultimodalGateway(
        base_url="https://portkey.test/v1",
        portkey_api_key="test-portkey-key",
        embed_route=_route("context-vl-embed"),
        rerank_route=_route("context-vl-rerank"),
        extract_route=_route("context-omni-extract"),
        client=client,
    )


@pytest.mark.asyncio
async def test_embed_builds_multimodal_payload_and_requires_2048_dimensions() -> None:
    observed: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        observed["path"] = request.url.path
        observed["payload"] = json.loads(request.content)
        observed["headers"] = dict(request.headers)
        return httpx.Response(
            200,
            json={"model": "nvidia/llama-nemotron-embed-vl-1b-v2", "data": [{"embedding": [0.5] * 2048}]},
        )

    gateway = _gateway(httpx.MockTransport(handler))
    result = await gateway.embed(
        EmbeddingRequest(
            input_type="passage",
            text="screenshot of a conversation",
            image_url="https://objects.example/context/sha256.png?signature=redacted",
        )
    )

    assert len(result.vector) == EMBEDDING_DIMENSIONS
    assert observed["path"] == "/v1/embeddings"
    payload = observed["payload"]
    assert isinstance(payload, dict)
    assert payload["input_type"] == "passage"
    assert payload["dimensions"] == 2048
    assert payload["input"][0]["content"][1]["type"] == "image_url"
    headers = observed["headers"]
    assert isinstance(headers, dict)
    assert "test-portkey-key" not in json.dumps(payload)
    assert headers["x-portkey-trace-id"] == result.trace_id


@pytest.mark.asyncio
async def test_embed_dimension_mismatch_fails_closed() -> None:
    def handler(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"data": [{"embedding": [0.0] * 1024}]})

    gateway = _gateway(httpx.MockTransport(handler))
    with pytest.raises(MultimodalGatewayError, match="dimension mismatch"):
        await gateway.embed(EmbeddingRequest(input_type="query", text="find this"))


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("response_body", "expected"),
    [
        (
            {"results": [{"index": 1, "relevance_score": 0.9}, {"index": 0, "relevance_score": 0.4}]},
            [("b", 0.9), ("a", 0.4)],
        ),
        (
            {"rankings": [{"index": 0, "logit": 4.2}, {"index": 1, "logit": -1.0}]},
            [("a", 4.2), ("b", -1.0)],
        ),
    ],
)
async def test_rerank_normalizes_portkey_and_nvidia_response_shapes(
    response_body: dict[str, object], expected: list[tuple[str, float]]
) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/rerank"
        payload = json.loads(request.content)
        assert payload["return_documents"] is False
        return httpx.Response(200, json=response_body)

    gateway = _gateway(httpx.MockTransport(handler))
    results = await gateway.rerank(
        "relevant conversation",
        [RerankCandidate("a", text="first"), RerankCandidate("b", text="second")],
    )

    assert [(item.candidate_id, item.score) for item in results] == expected


@pytest.mark.asyncio
async def test_extract_uses_remote_reference_and_parses_text_blocks() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        payload = json.loads(request.content)
        assert request.url.path == "/v1/chat/completions"
        assert payload["messages"][0]["content"][1] == {
            "type": "image_url",
            "image_url": {"url": "https://objects.example/context/screenshot.png"},
        }
        return httpx.Response(
            200,
            json={
                "model": "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning",
                "choices": [{"message": {"content": [{"type": "text", "text": "Line one"}, {"text": "Line two"}]}}],
            },
        )

    gateway = _gateway(httpx.MockTransport(handler))
    result = await gateway.extract(
        ExtractionRequest(
            instruction="Extract visible text.",
            source_url="https://objects.example/context/screenshot.png",
        )
    )
    assert result.text == "Line one\nLine two"


@pytest.mark.asyncio
async def test_upstream_error_does_not_expose_response_body_or_source_url() -> None:
    secret_body = "signed=https://objects.example/private?credential=secret"

    def handler(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(429, text=secret_body)

    gateway = _gateway(httpx.MockTransport(handler))
    with pytest.raises(MultimodalGatewayError) as caught:
        await gateway.extract(
            ExtractionRequest(
                instruction="Extract visible text.",
                source_url="https://objects.example/private?credential=secret",
            )
        )

    assert caught.value.status_code == 429
    assert secret_body not in str(caught.value)
    assert "objects.example" not in str(caught.value)


def test_unresolved_config_placeholders_fail_before_request() -> None:
    route = GatewayRoute(
        config={"provider": "openai", "api_key": "$OPENROUTER_API_KEY"},
        model_alias="context-vl-embed",
    )
    with pytest.raises(ValueError, match="unresolved secret placeholders"):
        route.serialized_config()


@pytest.mark.parametrize(
    "value",
    [
        EmbeddingRequest,
        RerankCandidate,
        ExtractionRequest,
    ],
)
def test_remote_inputs_do_not_accept_inline_or_local_media(value: type[object]) -> None:
    with pytest.raises(ValueError, match="HTTPS"):
        if value is EmbeddingRequest:
            EmbeddingRequest(input_type="passage", image_url="data:image/png;base64,abc")
        elif value is RerankCandidate:
            RerankCandidate("candidate", image_url="file:///tmp/image.png")
        else:
            ExtractionRequest(instruction="extract", source_url="http://localhost/image.png")
