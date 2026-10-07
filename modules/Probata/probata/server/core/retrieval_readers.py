"""Read configured Intake and Proffer context without indexing, case writes or evidence-store access.

Inputs are bounded requests and server-only endpoint/collection/vector configuration; outputs retain supplied citations.
Only bounded HTTP reads and remote query embeddings occur. Pick these ports for direct or Temporal-owned composition.
Every CONTEXT_RETRIEVAL_* URL/name is an explicit server allowlist, never a request field. Missing configuration
fails its leg. Query embeddings must match the published named vectors; no provider or default-vector fallback exists.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import json
import math
import os
import re
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.parse import urlsplit
from uuid import UUID

import httpx

from server.core.retrieval_adapters import intake_hits, proffer_hits
from server.core.retrieval_composition import RetrievalLeg
from server.core.retrieval_contracts import ReadHit, RetrievalRequest

MAX_UPSTREAM_BYTES = 2 << 20
MODES = frozenset({"keyword", "hybrid", "vector"})
HYBRID_ALPHA = 0.7  # Existing Intake filesystem-search contract.


def configured_origin(value: str) -> str:
    """Validate a server-configured HTTP origin; return it without credentials/paths, with no network effects."""
    parsed = urlsplit(value)
    if (parsed.scheme not in {"http", "https"} or not parsed.hostname or parsed.username or parsed.password
            or parsed.query or parsed.fragment or parsed.path not in {"", "/"}):
        raise RuntimeError("context upstream origin is not configured")
    return value.rstrip("/")


def configured_name(value: str, *, collection: bool = False) -> str:
    """Validate a server-selected GraphQL name; return it without I/O or caller-selected storage access."""
    pattern = r"[A-Z][A-Za-z0-9_]*" if collection else r"[A-Za-z_][A-Za-z0-9_]*"
    if re.fullmatch(pattern, value) is None:
        raise RuntimeError("context collection/vector is not configured")
    return value


def bearer_headers(path: str, *, fallback: str = "") -> dict[str, str]:
    """Read a configured mounted credential for one request; return auth headers without logging or caching secrets."""
    token = fallback
    if path:
        with Path(path).open("rb") as stream:
            raw = stream.read(4097)
        if not raw or len(raw) > 4096:
            raise RuntimeError("context credential unavailable")
        token = raw.decode("utf-8").strip()
    if token and re.fullmatch(r"[A-Za-z0-9\-._~+/]+=*", token) is None:
        raise RuntimeError("context credential unavailable")
    return {"Authorization": f"Bearer {token}"} if token else {}


@dataclass(frozen=True)
class ReaderConfig:
    """Hold server-only allowlists; inputs are environment values, outputs are reader settings, with no I/O."""

    intake_url: str = ""
    intake_collection: str = ""
    intake_vector: str = ""
    weaviate_url: str = ""
    proffer_collection: str = ""
    proffer_vector: str = ""
    proffer_matter_id: str = ""
    embed_url: str = ""
    embed_model: str = ""
    embed_dimensions: str = ""
    embed_key_file: str = ""
    weaviate_key_file: str = ""
    intake_key_file: str = ""

    @classmethod
    def from_environment(cls) -> ReaderConfig:
        """Read explicit server allowlists without provider calls; return settings, never accept callback URLs in requests."""
        values = {name: os.getenv(f"CONTEXT_RETRIEVAL_{name.upper()}", "") for name in cls.__dataclass_fields__}
        values["proffer_matter_id"] = os.getenv("PROFFER_MATTER_ID", "")
        return cls(**values)

    def matter(self) -> str:
        """Validate the configured Proffer UUID; return exact canonical scope without choosing a caller's case."""
        try:
            value = UUID(self.proffer_matter_id)
        except ValueError as exc:
            raise RuntimeError("Proffer matter is not configured") from exc
        if not value.int or str(value).startswith(("deadbeef-", "cafebabe-")):
            raise RuntimeError("Proffer matter is not configured")
        return str(value)

    def embedding_settings(self) -> tuple[str, str, int]:
        """Validate explicit remote embedding configuration; return endpoint/model/dimension without provider discovery."""
        parsed = urlsplit(self.embed_url)
        if (parsed.scheme not in {"http", "https"} or not parsed.hostname or parsed.username or parsed.password
                or parsed.query or parsed.fragment or parsed.path.rstrip("/") != "/v1"):
            raise RuntimeError("query embedding endpoint is not configured")
        try:
            dimensions = int(self.embed_dimensions)
        except ValueError as exc:
            raise RuntimeError("query embedding dimensions are not configured") from exc
        if not self.embed_model.strip() or not 1 <= dimensions <= 65536:
            raise RuntimeError("query embedding model/dimensions are not configured")
        return self.embed_url.rstrip("/") + "/embeddings", self.embed_model, dimensions


class ContextReaders:
    """Bind allowlisted HTTP reads to composition ports; inputs are server settings, outputs preserve original identities."""

    def __init__(self, config: ReaderConfig, *, transport: httpx.AsyncBaseTransport | None = None) -> None:
        """Retain server config and optional test transport; perform no discovery, network calls or writes."""
        self.config = config
        self.transport = transport

    async def post(self, url: str, body: dict[str, Any], headers: dict[str, str], timeout: float) -> dict[str, Any]:
        """POST a bounded read request; return bounded JSON without redirects, retries, provider-body errors or writes."""
        async with (
            httpx.AsyncClient(transport=self.transport, follow_redirects=False, timeout=timeout) as client,
            client.stream("POST", url, json=body, headers=headers) as response,
        ):
            response.raise_for_status()
            data = bytearray()
            async for chunk in response.aiter_bytes():
                data.extend(chunk)
                if len(data) > MAX_UPSTREAM_BYTES:
                    raise ValueError("context upstream response exceeds bound")
        result = json.loads(data)
        if not isinstance(result, dict):
            raise TypeError("invalid context upstream envelope")
        return result

    async def embedding(self, request: RetrievalRequest) -> list[float]:
        """Embed one query remotely using explicit matching settings; return a finite vector without fallback or indexing."""
        url, model, dimensions = self.config.embedding_settings()
        headers = bearer_headers(self.config.embed_key_file, fallback=os.getenv("NVIDIA_API_KEY", ""))
        if not headers:
            raise RuntimeError("query embedding credential unavailable")
        result = await self.post(url, {
            "model": model, "input": [request.query], "input_type": "query",
            "encoding_format": "float", "truncate": "NONE",
        }, headers, request.leg_timeout_seconds)
        rows = result["data"]
        if not isinstance(rows, list) or len(rows) != 1:
            raise ValueError("invalid query embedding result")
        vector = rows[0]["embedding"]
        if (not isinstance(vector, list) or len(vector) != dimensions
                or any(type(v) not in {int, float} or not math.isfinite(v) for v in vector) or not any(vector)):
            raise ValueError("query embedding dimension/value mismatch")
        return vector

    async def graphql(
        self, request: RetrievalRequest, collection: str, vector_name: str, where: str, fields: str,
    ) -> list[dict[str, Any]]:
        """Query one configured collection with pre-ranking scope; return original rows using explicit named-vector targets."""
        origin = configured_origin(self.config.weaviate_url)
        collection = configured_name(collection, collection=True)
        query = json.dumps(request.query)
        additional = "id score"
        if request.mode == "keyword":
            operator = f"bm25: {{query: {query}, properties: [\"text\"]}}"
        else:
            target = json.dumps(configured_name(vector_name))
            vector = json.dumps(await self.embedding(request), allow_nan=False)
            if request.mode == "vector":
                operator = f"nearVector: {{vector: {vector}, targetVectors: [{target}]}}"
                additional = "id distance"
            else:
                # Match the existing filesystem searcher's explicit alpha and text query contract.
                operator = (f"hybrid: {{query: {query}, alpha: {HYBRID_ALPHA}, properties: [\"text\"], "
                            f"vector: {vector}, targetVectors: [{target}]}}")
        result = await self.post(origin + "/v1/graphql", {"query": (
            f"{{ Get {{ {collection}(limit: {request.per_leg_limit}, {operator}, where: {where}) "
            f"{{ {fields} _additional {{ {additional} }} }} }} }}"
        )}, bearer_headers(self.config.weaviate_key_file), request.leg_timeout_seconds)
        if result.get("errors"):
            raise RuntimeError("context GraphQL read rejected")
        rows = result["data"]["Get"][collection]
        if not isinstance(rows, list) or len(rows) > request.per_leg_limit:
            raise ValueError("invalid context result list")
        return rows

    async def intake(self, request: RetrievalRequest) -> list[ReadHit]:
        """Read the allowlisted filesystem index; return exact Intake locators without unsupported source-scope broadening."""
        if request.scope.requested_fields():
            raise ValueError("Intake source scope is unsupported")
        collection = configured_name(self.config.intake_collection, collection=True)
        origin = configured_origin(self.config.intake_url)
        body: dict[str, Any] = {"query": request.query, "mode": request.mode, "limit": request.per_leg_limit}
        if request.mode == "vector":
            body["vector"] = await self.embedding(request)
        result = await self.post(origin + "/filesystem/search", body, bearer_headers(self.config.intake_key_file),
                                 request.leg_timeout_seconds)
        if result.get("collection") != collection:
            raise ValueError("Intake collection differs from server allowlist")
        if request.mode != "keyword" and result.get("target_vector") != configured_name(self.config.intake_vector):
            raise ValueError("Intake vector differs from server allowlist")
        return intake_hits(result, mode=request.mode)

    async def proffer(self, request: RetrievalRequest) -> list[ReadHit]:
        """Read configured-matter Proffer chunks; return versions/message locators without cross-case or evidence reads."""
        if request.scope.requested_fields() - {"source_version_ids"}:
            raise ValueError("Proffer scope is unsupported")
        matter = self.config.matter()
        where = '{path: ["matter_id"], operator: Equal, valueText: ' + json.dumps(matter) + "}"
        versions = request.scope.source_version_ids
        if versions:
            version_filters = [
                '{path: ["source_version_ids"], operator: ContainsAny, valueText: ' + json.dumps(versions) + "}",
                *['{path: ["source_version_id"], operator: Equal, valueText: ' + json.dumps(v) + "}" for v in versions],
            ]
            where = "{operator: And, operands: [" + where + ",{operator: Or, operands: [" + ",".join(version_filters) + "]}]}"
        rows = await self.graphql(request, self.config.proffer_collection, self.config.proffer_vector, where,
                                  "matter_id text thread_id first_message_id last_message_id message_ids "
                                  "normalized_record_ids chunk_index content_hash source_version_ids source_version_id record_kind")
        if any(row.get("matter_id") != matter for row in rows):
            raise ValueError("Proffer returned an out-of-scope object")
        return proffer_hits(self.config.proffer_collection, rows, mode=request.mode,
                            vector_name=self.config.proffer_vector if request.mode != "keyword" else None)

    def legs(self) -> Mapping[str, RetrievalLeg]:
        """Return the Intake/Proffer port allowlist without I/O; evidence and unbound graph readers remain excluded."""
        return {"intake": RetrievalLeg(self.intake, MODES),
                "proffer": RetrievalLeg(self.proffer, MODES, frozenset({"source_version_ids"}))}

    def capabilities(self) -> dict[str, Any]:
        """Describe configured support without network health claims or credential/endpoint disclosure; never infer coverage."""
        result = {}
        for name in ("intake", "proffer"):
            modes = []
            for mode in sorted(MODES):
                try:
                    configured_name(getattr(self.config, name + "_collection"), collection=True)
                    if name == "proffer":
                        self.config.matter()
                    configured_origin(self.config.intake_url if name == "intake" else self.config.weaviate_url)
                    if mode != "keyword":
                        configured_name(getattr(self.config, name + "_vector"))
                        if name == "proffer" or mode == "vector":
                            self.config.embedding_settings()
                    modes.append(mode)
                except (ValueError, RuntimeError):
                    continue
            result[name] = {"configured_modes": modes, "supported_scopes": sorted(self.legs()[name].supported_scopes),
                            "scope_policy": "server_matter" if name == "proffer" else "global_index",
                            "coverage": "unknown", "live_verified": False}
        return {"legs": result, "graph": {"available": False, "reason": "reader_not_bound"},
                "evidence": {"available": False, "reason": "approved_capability_adapter_required"}}
