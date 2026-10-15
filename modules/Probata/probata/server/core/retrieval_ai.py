"""Adapt native AI chat chunks to search citations without a PostgreSQL conversation lookup.

Inputs: existing Weaviate rows and native provenance. Outputs: bounded cited hits.
Effects: none; choose for the native AI collection, whose chat bodies live in Weaviate.
"""
from __future__ import annotations

import hashlib
import json
from typing import Any, Literal

from server.core.retrieval_adapters import weaviate_score
from server.core.retrieval_contracts import ReadHit, RetrievalScope

AI_FIELDS = "body conversation_id conversation_title source_format source_version_id chunk_index record_kind embed_model provenance"
AI_SCOPES = frozenset({"source_version_ids"})
AI_EMBED_MODEL = "nvidia/nemotron-3-embed-1b"
AI_EMBED_DIMENSIONS = 2048
AI_VECTOR = "text_nim"


def ai_where(scope: RetrievalScope) -> str:
    """Filter native chunks and requested exact source versions before ranking.

    Inputs: typed source scope. Output: GraphQL expression. Effects: none;
    choose for AI chat search without assuming a PG case or message identity.
    """
    operands = ['{path: ["record_kind"], operator: Equal, valueText: "ai_conversation_chunk"}']
    if scope.source_version_ids:
        versions = [('{path: ["source_version_id"], operator: Equal, valueText: '
                     + json.dumps(value) + "}") for value in scope.source_version_ids]
        operands.append("{operator: Or, operands: [" + ",".join(versions) + "]}")
    return "{operator: And, operands: [" + ",".join(operands) + "]}" if len(operands) > 1 else operands[0]


def ai_hits(collection: str, rows: list[dict[str, Any]], *,
            mode: Literal["keyword", "hybrid", "vector"], vector_name: str | None) -> list[ReadHit]:
    """Preserve native source, provider version, external span bundle and actual workflow references.

    Inputs: bounded rows and actual query mode. Output: cited excerpts. Effects:
    none; missing or conflicting identity fails this leg instead of inventing a human thread.
    """
    hits = []
    for row in rows:
        if row.get("record_kind") != "ai_conversation_chunk":
            raise ValueError("AI search returned a different record kind")
        values = row.get("provenance")
        if not isinstance(values, list) or len(values) != 1 or not isinstance(values[0], str):
            raise ValueError("AI chunk has no unique native provenance")
        provenance = json.loads(values[0])
        if not isinstance(provenance, dict):
            raise ValueError("invalid AI provenance")
        source = _text(provenance, "source_ref")
        provider = _text(provenance, "provider_version_id")
        prepared = _text(provenance, "prepared_ref")
        if provenance.get("chunk_index") != row.get("chunk_index") or str(provenance.get("conversation_id") or "") != str(row.get("conversation_id") or ""):
            raise ValueError("AI chunk locator differs from native provenance")
        version = row.get("source_version_id")
        if version and provenance.get("source_version_id") != version:
            raise ValueError("AI chunk source version differs from provenance")
        if mode != "keyword" and row.get("embed_model") != AI_EMBED_MODEL:
            raise ValueError("AI chunk embedding model differs from query")
        locator = {"source_ref": source, "provider_version_id": provider,
                   "prepared_ref": prepared, "conversation_index": str(provenance["conversation_index"]),
                   "chunk_index": str(row["chunk_index"]), "source_kind": "ai_chat",
                   "source_format": _text(row, "source_format")}
        for name in ("conversation_id", "historical_source_version_id", "decoded_ref", "original_ref", "workflow_id"):
            value = provenance.get(name)
            if isinstance(value, str) and value:
                locator[name] = value
        title = row.get("conversation_title")
        if isinstance(title, str) and title.strip():
            locator["conversation_title"] = title[:2048]
        body = _text(row, "body")
        hits.append(ReadHit(text=body, score=weaviate_score(row["_additional"], mode), citation={
            "collection": collection, "object_id": row["_additional"]["id"],
            "source_version_ids": [version] if version else [],
            "chunk_index": row["chunk_index"], "source_sha256": provenance.get("source_sha256"),
            "content_hash": hashlib.sha256(body.encode("utf-8")).hexdigest(),
            "record_kind": "ai_conversation_chunk", "vector_name": vector_name,
            "locator": locator,
        }))
    return hits


def _text(value: dict[str, Any], field: str) -> str:
    """Require a supplied nonblank string; inputs are metadata, output is unchanged, with no I/O."""
    result = value.get(field)
    if not isinstance(result, str) or not result.strip():
        raise ValueError(f"AI chunk is missing {field}")
    return result
