"""Search over the Case Bible chunk collection (CaseBibleChunks20261002). One unit, one job: read.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Hybrid (BM25 + vector on one named vector) or keyword, over active chunks only, every hit carrying
the locator that
opens it (``vault_key``, ``member_path``, ``catalog_path``) and what it covers. A query embedding is
supplied by the
caller; this unit never embeds. Dict filters in GraphQL syntax only; no provider body or chunk text
in an error.
"""

from __future__ import annotations

import json
import math
import re
from typing import Literal

import httpx
from pydantic import BaseModel, Field

HIT_FIELDS = (
    "text record_kind thread_id vault_key member_path catalog_path filename "
    "document_id content_hash chunk_index "
    "message_count start_at end_at participant_names chunker_version resolution origin_system"
)


class ChunkSearchRequest(BaseModel):
    query: str = Field(min_length=1, max_length=4096)
    limit: int = Field(default=20, ge=1, le=100)
    mode: Literal["hybrid", "keyword"] = "hybrid"
    slot: str = Field(default="text_nim", pattern=r"^[A-Za-z_][A-Za-z0-9_]*$")
    record_kind: Literal["conversation_chunk", "document_chunk"] | None = None
    path_prefix: str | None = Field(default=None, max_length=1024)


class ChunkHit(BaseModel):
    object_id: str
    score: float
    text: str
    record_kind: str = ""
    thread_id: str = ""
    vault_key: str = ""
    member_path: str = ""
    catalog_path: str = ""
    filename: str = ""
    document_id: str = ""
    content_hash: str = ""
    chunk_index: int = 0
    message_count: int = 0
    start_at: str | None = None
    end_at: str | None = None
    participant_names: list[str] = Field(default_factory=list)
    chunker_version: str = ""
    resolution: str = ""


class ChunkSearchResponse(BaseModel):
    query: str
    collection: str
    slot: str | None
    hits: list[ChunkHit]


class ChunkSearchError(RuntimeError):
    pass


def _where(request: ChunkSearchRequest) -> str:
    operands = ['{path: ["active"], operator: Equal, valueBoolean: true}']
    if request.record_kind:
        operands.append(
            f'{{path: ["record_kind"], operator: Equal, '
            f"valueText: {json.dumps(request.record_kind)}}}"
        )
    if request.path_prefix:
        operands.append(
            f'{{path: ["vault_key"], operator: Like, '
            f"valueText: {json.dumps(request.path_prefix + '*')}}}"
        )
    return (
        operands[0]
        if len(operands) == 1
        else "{operator: And, operands: [" + ", ".join(operands) + "]}"
    )


async def search_chunks(
    base_url: str,
    collection: str,
    request: ChunkSearchRequest,
    *,
    vector: list[float] | None = None,
    dimensions: int | None = None,
    api_key: str = "",
    transport: httpx.AsyncBaseTransport | None = None,
) -> ChunkSearchResponse:
    if not re.fullmatch(r"[A-Z][A-Za-z0-9_]*", collection):
        raise ValueError("Explicit valid chunk collection name required")
    query_text = json.dumps(request.query, ensure_ascii=True)
    if request.mode == "hybrid":
        if vector is None or (dimensions is not None and len(vector) != dimensions):
            raise ValueError("Query embedding dimension mismatch")
        if not all(math.isfinite(v) for v in vector) or not any(vector):
            raise ValueError("Query embedding must be finite and nonzero")
        operator = (
            f"hybrid: {{query: {query_text}, alpha: 0.7, "
            f"vector: {json.dumps(vector, allow_nan=False)}, "
            f"targetVectors: [{json.dumps(request.slot)}]}}"
        )
    else:
        operator = f"bm25: {{query: {query_text}}}"
    graphql = (
        f"{{ Get {{ {collection}(limit: {request.limit}, {operator}, where: {_where(request)}) "
        f"{{ {HIT_FIELDS} _additional {{id score}} }} }} }}"
    )
    headers = {"Authorization": f"Bearer {api_key}"} if api_key else {}
    try:
        async with httpx.AsyncClient(
            timeout=30.0, headers=headers, transport=transport, follow_redirects=False
        ) as client:
            response = await client.post(
                base_url.rstrip("/") + "/v1/graphql", json={"query": graphql}
            )
            response.raise_for_status()
            body = response.json()
        if body.get("errors"):
            raise ChunkSearchError("Weaviate rejected the chunk query")
        rows = body["data"]["Get"][collection]
        hits = [
            ChunkHit(
                object_id=row["_additional"]["id"],
                score=float(row["_additional"]["score"]),
                **{k: row[k] for k in row if k != "_additional" and row[k] is not None},
            )
            for row in rows
        ]
    except (httpx.HTTPError, KeyError, TypeError, ValueError) as exc:
        raise ChunkSearchError("Chunk search service unavailable or incompatible") from exc
    return ChunkSearchResponse(
        query=request.query,
        collection=collection,
        slot=request.slot if request.mode == "hybrid" else None,
        hits=hits,
    )
