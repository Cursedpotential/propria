"""Adapt active CaseBible Weaviate chunks to exact context citations without changing their source identity.

Inputs are rows from the configured CaseBible chunk collection and query mode; output retains the original
Weaviate object, source/document/version IDs, source SHA-1, chunk-text hash and source locators. No I/O or
indexing occurs. Use this adapter for the native CaseBible chunk schema, not IntakeCorpus or Proffer chunks.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import json
from typing import Any, Literal

from server.core.retrieval_adapters import weaviate_score
from server.core.retrieval_contracts import ReadHit, RetrievalScope

CASEBIBLE_VECTOR_NAME = "text_nim"
CASEBIBLE_EMBED_MODEL = "nvidia/nemotron-3-embed-1b"
CASEBIBLE_EMBED_DIMENSIONS = 2048
CASEBIBLE_FIELDS = (
    "text source_id version_id document_id content_hash sha1 chunk_index chunk_ordinal "
    "char_start char_end record_kind thread_id vault_key member_path catalog_path filename "
    "chunker_version resolution origin_system active"
)
CASEBIBLE_MODES = frozenset({"keyword", "hybrid", "vector"})
CASEBIBLE_SCOPES = frozenset({"source_id", "thread_id", "path_prefix"})


def casebible_where(scope: RetrievalScope) -> str:
    """Build only native CaseBible pre-ranking filters; input is typed scope, output is Weaviate GraphQL syntax, no I/O.

    `active` is always required; source/thread/path filters use existing scalar properties. Pick this filter builder
    for CaseBible chunks so inactive versions cannot rank and locator scopes are never applied post-search.
    """
    operands = ['{path: ["active"], operator: Equal, valueBoolean: true}']
    if scope.source_id:
        operands.append(f'{{path: ["source_id"], operator: Equal, valueText: {_graphql_string(scope.source_id)}}}')
    if scope.thread_id:
        operands.append(f'{{path: ["thread_id"], operator: Equal, valueText: {_graphql_string(scope.thread_id)}}}')
    if scope.path_prefix:
        if any(wildcard in scope.path_prefix for wildcard in ("*", "?")):
            raise ValueError("CaseBible path_prefix wildcards are unsupported by Weaviate Like")
        operands.append(
            f'{{path: ["vault_key"], operator: Like, valueText: {_graphql_string(scope.path_prefix + "*")}}}'
        )
    if len(operands) == 1:
        return operands[0]
    return "{operator: And, operands: [" + ", ".join(operands) + "]}"


def _graphql_string(value: str) -> str:
    """JSON-quote one GraphQL string value; input is a validated identifier/prefix, output cannot alter query syntax."""
    return json.dumps(value, ensure_ascii=True)


def casebible_hits(
    collection: str,
    rows: list[dict[str, Any]],
    *,
    mode: Literal["keyword", "hybrid", "vector"],
    vector_name: str | None,
) -> list[ReadHit]:
    """Convert native CaseBible rows to cited hits without guessing a version or locator; perform no I/O.

    Inputs are the configured collection, bounded GraphQL rows, actual mode and named vector. Output preserves the
    Weaviate ID, source/document/version identity, SHA-1 source fingerprint, SHA-256 chunk hash and byte locators.
    Pick this over `proffer_hits` for CaseBible's `source_id`/`version_id` schema. Missing identity fails the leg.
    """
    hits = []
    for row in rows:
        if row.get("active") is not True:
            raise ValueError("CaseBible returned an inactive chunk")
        source_id = _required_text(row, "source_id")
        document_id = _required_text(row, "document_id")
        version_id = _required_text(row, "version_id")
        content_hash = _required_text(row, "content_hash")
        additional = row["_additional"]
        locator = {
            key: str(row[key])
            for key in (
                "version_id",
                "sha1",
                "vault_key",
                "member_path",
                "catalog_path",
                "filename",
                "chunk_ordinal",
                "char_start",
                "char_end",
                "chunker_version",
                "resolution",
                "origin_system",
            )
            if row.get(key) is not None and row[key] != ""
        }
        locator["version_id"] = version_id
        hits.append(
            ReadHit(
                text=row["text"],
                score=weaviate_score(additional, mode),
                citation={
                    "collection": collection,
                    "object_id": additional["id"],
                    "source_id": source_id,
                    "document_id": document_id,
                    "chunk_index": row["chunk_index"],
                    "thread_id": row.get("thread_id") or None,
                    "content_hash": content_hash,
                    "vector_name": vector_name if mode != "keyword" else None,
                    "record_kind": row.get("record_kind") or None,
                    "locator": locator,
                },
            )
        )
    return hits


def _required_text(row: dict[str, Any], field: str) -> str:
    """Require a nonblank native identity field; input is one provider row, output is its unchanged string value."""
    value = row.get(field)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"CaseBible chunk is missing {field}")
    return value
