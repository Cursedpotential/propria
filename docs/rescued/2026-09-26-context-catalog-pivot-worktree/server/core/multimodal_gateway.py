"""Typed, fail-closed Portkey client for the non-evidence context index.

The client is deliberately storage-neutral: it accepts bounded values and signed
HTTP(S) object references, calls one gateway endpoint, and returns typed results.
It never reads PostgreSQL, object storage, or ambient provider credentials.

Provider selection, rate-limit failover, and model pinning are supplied as
materialized Portkey configs.  Configs containing unresolved ``$ENV`` placeholders
are rejected before a request is made so secrets cannot accidentally be sent as
literal provider credentials.

Byline: Codex · GPT-5.6-Sol · 2026-09-09
"""

from __future__ import annotations

from collections.abc import Mapping, Sequence
from dataclasses import dataclass
import json
import math
from typing import Any, Literal
from urllib.parse import urlsplit
from uuid import uuid4

import httpx


EMBEDDING_DIMENSIONS = 2048
DEFAULT_EMBED_MODEL_ALIAS = "context-vl-embed"
DEFAULT_RERANK_MODEL_ALIAS = "context-vl-rerank"
DEFAULT_EXTRACT_MODEL_ALIAS = "context-omni-extract"


class MultimodalGatewayError(RuntimeError):
    """A safe, fail-closed gateway error that never includes request content."""

    def __init__(self, message: str, *, trace_id: str, status_code: int = 502):
        self.trace_id = trace_id
        self.status_code = status_code
        super().__init__(message)


@dataclass(frozen=True, slots=True)
class GatewayRoute:
    """One materialized Portkey config and its stable request-side model alias."""

    config: str | Mapping[str, Any]
    model_alias: str

    def serialized_config(self) -> str:
        """Return compact JSON, rejecting empty or unresolved configuration."""
        if isinstance(self.config, str):
            value = self.config.strip()
            if not value:
                raise ValueError("Portkey route config must not be empty")
            try:
                parsed = json.loads(value)
            except json.JSONDecodeError as error:
                raise ValueError("Portkey route config must be valid JSON") from error
        else:
            parsed = dict(self.config)
            value = json.dumps(parsed, separators=(",", ":"), sort_keys=True)

        if not isinstance(parsed, dict) or not parsed:
            raise ValueError("Portkey route config must be a non-empty JSON object")
        if _contains_unresolved_secret(parsed):
            raise ValueError("Portkey route config contains unresolved secret placeholders")
        if not self.model_alias.strip():
            raise ValueError("Portkey route model alias must not be empty")
        return value


@dataclass(frozen=True, slots=True)
class EmbeddingRequest:
    """A single query or document embedding input.

    Queries are text-only because the selected model is trained to retrieve visual
    documents from textual queries. Documents may contain text, an image reference,
    or both. Image bytes are never accepted here; callers pass a signed URL.
    """

    input_type: Literal["query", "passage"]
    text: str | None = None
    image_url: str | None = None

    def __post_init__(self) -> None:
        text = self.text.strip() if self.text else ""
        if not text and not self.image_url:
            raise ValueError("embedding input requires text, image_url, or both")
        if self.input_type == "query" and self.image_url is not None:
            raise ValueError("query embeddings are text-only for this model")
        if self.image_url is not None:
            _validate_remote_reference(self.image_url)

    def wire_input(self) -> str | list[dict[str, list[dict[str, Any]]]]:
        """Build the OpenRouter/NVIDIA-compatible multimodal embedding input."""
        text = self.text.strip() if self.text else ""
        if self.image_url is None:
            return text

        content: list[dict[str, Any]] = []
        if text:
            content.append({"type": "text", "text": text})
        content.append({"type": "image_url", "image_url": {"url": self.image_url}})
        return [{"content": content}]


@dataclass(frozen=True, slots=True)
class EmbeddingResult:
    """A dimension-checked vector and its request trace."""

    vector: tuple[float, ...]
    model: str
    trace_id: str


@dataclass(frozen=True, slots=True)
class RerankCandidate:
    """One bounded context-index candidate."""

    candidate_id: str
    text: str | None = None
    image_url: str | None = None

    def __post_init__(self) -> None:
        if not self.candidate_id.strip():
            raise ValueError("candidate_id must not be empty")
        text = self.text.strip() if self.text else ""
        if not text and not self.image_url:
            raise ValueError("rerank candidate requires text, image_url, or both")
        if self.image_url is not None:
            _validate_remote_reference(self.image_url)

    def wire_document(self) -> str | dict[str, list[dict[str, Any]]]:
        """Return the shared OpenRouter multimodal rerank document shape."""
        text = self.text.strip() if self.text else ""
        if self.image_url is None:
            return text

        content: list[dict[str, Any]] = []
        if text:
            content.append({"type": "text", "text": text})
        content.append({"type": "image_url", "image_url": {"url": self.image_url}})
        return {"content": content}


@dataclass(frozen=True, slots=True)
class RerankResult:
    """A provider-neutral relevance result."""

    candidate_id: str
    source_index: int
    score: float


@dataclass(frozen=True, slots=True)
class ExtractionRequest:
    """One on-demand extraction instruction against a signed media reference."""

    instruction: str
    source_url: str
    media_type: Literal["image", "video"] = "image"
    max_output_tokens: int = 2048

    def __post_init__(self) -> None:
        if not self.instruction.strip():
            raise ValueError("extraction instruction must not be empty")
        _validate_remote_reference(self.source_url)
        if self.max_output_tokens < 1 or self.max_output_tokens > 8192:
            raise ValueError("max_output_tokens must be between 1 and 8192")

    def wire_content(self) -> list[dict[str, Any]]:
        """Build OpenAI-compatible multimodal message content."""
        content: list[dict[str, Any]] = [{"type": "text", "text": self.instruction.strip()}]
        if self.media_type == "image":
            content.append({"type": "image_url", "image_url": {"url": self.source_url}})
        else:
            content.append({"type": "video_url", "video_url": {"url": self.source_url}})
        return content


@dataclass(frozen=True, slots=True)
class ExtractionResult:
    """Extracted text and its request trace."""

    text: str
    model: str
    trace_id: str


class ContextMultimodalGateway:
    """Execute atomic context embedding, reranking, and extraction calls."""

    def __init__(
        self,
        *,
        base_url: str,
        portkey_api_key: str,
        embed_route: GatewayRoute,
        rerank_route: GatewayRoute,
        extract_route: GatewayRoute,
        client: httpx.AsyncClient | None = None,
        timeout: float = 180.0,
    ) -> None:
        self._base_url = _validate_gateway_base_url(base_url)
        if not portkey_api_key.strip():
            raise ValueError("Portkey authentication is required")
        self._api_key = portkey_api_key
        self._embed_route = embed_route
        self._rerank_route = rerank_route
        self._extract_route = extract_route
        self._owned_client = client is None
        self._client = client or httpx.AsyncClient(timeout=httpx.Timeout(timeout, connect=15.0))

    async def __aenter__(self) -> ContextMultimodalGateway:
        return self

    async def __aexit__(self, *_exc: object) -> None:
        await self.aclose()

    async def aclose(self) -> None:
        """Close only the client allocated by this gateway instance."""
        if self._owned_client:
            await self._client.aclose()

    async def embed(self, request: EmbeddingRequest) -> EmbeddingResult:
        """Create exactly one fixed-width embedding or raise."""
        trace_id = str(uuid4())
        payload: dict[str, Any] = {
            "model": self._embed_route.model_alias,
            "input": request.wire_input(),
            "input_type": request.input_type,
            "dimensions": EMBEDDING_DIMENSIONS,
            "encoding_format": "float",
        }
        response = await self._post("embeddings", payload, self._embed_route, trace_id)
        body = _json_object(response, trace_id)
        rows = body.get("data")
        if not isinstance(rows, list) or len(rows) != 1 or not isinstance(rows[0], dict):
            raise MultimodalGatewayError("Portkey returned an invalid embedding result", trace_id=trace_id)
        vector = _validated_vector(rows[0].get("embedding"), trace_id)
        return EmbeddingResult(vector=vector, model=_response_model(body), trace_id=trace_id)

    async def rerank(
        self,
        query: str,
        candidates: Sequence[RerankCandidate],
        *,
        top_n: int | None = None,
    ) -> tuple[RerankResult, ...]:
        """Rerank candidates and normalize OpenRouter/Portkey or NVIDIA results."""
        clean_query = query.strip()
        if not clean_query:
            raise ValueError("rerank query must not be empty")
        if not candidates:
            raise ValueError("rerank requires at least one candidate")
        if top_n is None:
            top_n = len(candidates)
        if top_n < 1 or top_n > len(candidates):
            raise ValueError("top_n must be between 1 and the candidate count")

        trace_id = str(uuid4())
        payload: dict[str, Any] = {
            "model": self._rerank_route.model_alias,
            "query": clean_query,
            "documents": [candidate.wire_document() for candidate in candidates],
            "top_n": top_n,
            "return_documents": False,
        }
        response = await self._post("rerank", payload, self._rerank_route, trace_id)
        body = _json_object(response, trace_id)
        normalized = _normalize_rankings(body, candidates, trace_id)
        if len(normalized) < top_n:
            raise MultimodalGatewayError("Portkey returned too few rerank results", trace_id=trace_id)
        return tuple(sorted(normalized, key=lambda item: item.score, reverse=True)[:top_n])

    async def extract(self, request: ExtractionRequest) -> ExtractionResult:
        """Extract text from one remote media reference or raise."""
        trace_id = str(uuid4())
        payload: dict[str, Any] = {
            "model": self._extract_route.model_alias,
            "messages": [{"role": "user", "content": request.wire_content()}],
            "max_tokens": request.max_output_tokens,
            "stream": False,
        }
        response = await self._post("chat/completions", payload, self._extract_route, trace_id)
        body = _json_object(response, trace_id)
        text = _completion_text(body)
        if not text:
            raise MultimodalGatewayError("Portkey returned no extracted text", trace_id=trace_id)
        return ExtractionResult(text=text, model=_response_model(body), trace_id=trace_id)

    async def _post(
        self,
        path: str,
        payload: Mapping[str, Any],
        route: GatewayRoute,
        trace_id: str,
    ) -> httpx.Response:
        headers = {
            "Accept": "application/json",
            "Content-Type": "application/json",
            "x-portkey-api-key": self._api_key,
            "x-portkey-config": route.serialized_config(),
            "x-portkey-trace-id": trace_id,
            "x-portkey-metadata": json.dumps(
                {"surface": "context-index", "route": path}, separators=(",", ":")
            ),
        }
        try:
            response = await self._client.post(f"{self._base_url}/{path}", headers=headers, json=payload)
        except httpx.HTTPError as error:
            raise MultimodalGatewayError("Portkey is unreachable", trace_id=trace_id) from error
        if not response.is_success:
            # Provider responses can echo prompts or signed object URLs.  Never put
            # a body excerpt in exceptions, logs, or Temporal failure details.
            raise MultimodalGatewayError(
                f"Portkey request failed with status {response.status_code}",
                trace_id=trace_id,
                status_code=response.status_code,
            )
        return response


def _contains_unresolved_secret(value: object) -> bool:
    if isinstance(value, str):
        return value.startswith("$")
    if isinstance(value, Mapping):
        return any(_contains_unresolved_secret(item) for item in value.values())
    if isinstance(value, Sequence) and not isinstance(value, (str, bytes, bytearray)):
        return any(_contains_unresolved_secret(item) for item in value)
    return False


def _validate_gateway_base_url(value: str) -> str:
    base_url = value.strip().rstrip("/")
    parsed = urlsplit(base_url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc or parsed.username or parsed.password:
        raise ValueError("Portkey base_url must be an HTTP(S) URL without user information")
    return base_url


def _validate_remote_reference(value: str) -> None:
    parsed = urlsplit(value)
    if parsed.scheme != "https" or not parsed.netloc or parsed.username or parsed.password:
        raise ValueError("media references must be HTTPS URLs without user information")


def _json_object(response: httpx.Response, trace_id: str) -> dict[str, Any]:
    try:
        body = response.json()
    except (json.JSONDecodeError, ValueError) as error:
        raise MultimodalGatewayError("Portkey returned invalid JSON", trace_id=trace_id) from error
    if not isinstance(body, dict):
        raise MultimodalGatewayError("Portkey returned an invalid JSON object", trace_id=trace_id)
    return body


def _response_model(body: Mapping[str, Any]) -> str:
    model = body.get("model")
    return model if isinstance(model, str) and model.strip() else "unknown"


def _validated_vector(value: object, trace_id: str) -> tuple[float, ...]:
    if not isinstance(value, list) or len(value) != EMBEDDING_DIMENSIONS:
        raise MultimodalGatewayError(
            f"embedding dimension mismatch; expected {EMBEDDING_DIMENSIONS}", trace_id=trace_id
        )
    vector: list[float] = []
    for item in value:
        if isinstance(item, bool) or not isinstance(item, (int, float)) or not math.isfinite(float(item)):
            raise MultimodalGatewayError("embedding contains a non-finite value", trace_id=trace_id)
        vector.append(float(item))
    return tuple(vector)


def _normalize_rankings(
    body: Mapping[str, Any],
    candidates: Sequence[RerankCandidate],
    trace_id: str,
) -> list[RerankResult]:
    raw_results = body.get("results")
    index_key = "index"
    score_key = "relevance_score"
    if not isinstance(raw_results, list):
        raw_results = body.get("rankings")
        score_key = "logit"
    if not isinstance(raw_results, list):
        raise MultimodalGatewayError("Portkey returned an invalid rerank result", trace_id=trace_id)

    seen: set[int] = set()
    normalized: list[RerankResult] = []
    for item in raw_results:
        if not isinstance(item, dict):
            raise MultimodalGatewayError("Portkey returned an invalid rerank row", trace_id=trace_id)
        index = item.get(index_key)
        score = item.get(score_key)
        if (
            isinstance(index, bool)
            or not isinstance(index, int)
            or index < 0
            or index >= len(candidates)
            or index in seen
            or isinstance(score, bool)
            or not isinstance(score, (int, float))
            or not math.isfinite(float(score))
        ):
            raise MultimodalGatewayError("Portkey returned an invalid rerank row", trace_id=trace_id)
        seen.add(index)
        normalized.append(
            RerankResult(candidate_id=candidates[index].candidate_id, source_index=index, score=float(score))
        )
    return normalized


def _completion_text(body: Mapping[str, Any]) -> str:
    choices = body.get("choices")
    if not isinstance(choices, list) or not choices or not isinstance(choices[0], dict):
        return ""
    message = choices[0].get("message")
    if not isinstance(message, dict):
        return ""
    content = message.get("content")
    if isinstance(content, str):
        return content.strip()
    if not isinstance(content, list):
        return ""
    parts: list[str] = []
    for item in content:
        if not isinstance(item, dict):
            continue
        text = item.get("text")
        if isinstance(text, str) and text.strip():
            parts.append(text.strip())
    return "\n".join(parts)
