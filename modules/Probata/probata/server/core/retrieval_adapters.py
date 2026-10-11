"""Adapt existing Intake and Proffer read results without network, indexing or guessed identities.

Inputs are existing API/Weaviate objects; outputs are bounded ReadHit objects with supplied citations.
Use with injected read legs; these adapters neither query a store nor confer evidence authority.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import math
from typing import Any, Literal

from server.core.retrieval_contracts import Citation, ReadHit


def intake_hits(
    payload: dict[str, Any], *, mode: Literal["keyword", "hybrid", "vector"] = "keyword",
) -> list[ReadHit]:
    """Convert /filesystem/search results to cited hits without I/O; preserve IDs and available vault locators.

    Inputs are the existing response and actual mode; output is ReadHit, with negative distance for pure vector.
    Missing required fields fail the leg rather than inventing a source/version/hash mapping.
    """
    return [ReadHit(
        text=row["text"], score=weaviate_score(row, mode),
        citation=Citation(
            collection=payload["collection"], object_id=row["object_id"],
            source_id=row["source_id"], document_id=row["document_id"], chunk_id=row["chunk_id"],
            vector_name=payload.get("target_vector"),
            locator={key: row[key] for key in ("source_path", "vault_key", "resolution") if row.get(key)},
        ),
    ) for row in payload["hits"]]


def weaviate_score(additional: dict[str, Any], mode: Literal["keyword", "hybrid", "vector"]) -> float:
    """Adapt provider score/distance to a finite descending-order score without inventing a ranking algorithm.

    Inputs are Weaviate additional metadata and actual query mode; output is score or negative distance.
    Pure vector requires distance. Pick this before library RRF; no normalization, fallback or I/O occurs.
    """
    score = -float(additional["distance"]) if mode == "vector" else float(additional["score"])
    if not math.isfinite(score):
        raise ValueError("nonfinite Weaviate ranking metadata")
    return score


def proffer_hits(
    collection: str, rows: list[dict[str, Any]], *, vector_name: str | None = None,
    mode: Literal["keyword", "hybrid", "vector"] = "keyword",
) -> list[ReadHit]:
    """Convert Proffer chunk objects to cited hits without I/O; require real Weaviate object IDs.

    Inputs are collection, GraphQL rows, actual mode/target vector; output preserves versions/message locators.
    Pick this for ProfferChunks, not raw message-event objects or the display-only Read search response.
    """
    hits = []
    for row in rows:
        versions = list(row.get("source_version_ids") or [])
        single = row.get("source_version_id")
        if single and single not in versions:
            versions.append(single)
        locator = {key: row[key] for key in ("first_message_id", "thread_id") if row.get(key)}
        # Call-log chunks have no message/thread. This is a returned canonical version, never a synthesized URI.
        if single or versions:
            locator["source_version_id"] = single or versions[0]
        hits.append(ReadHit(
            text=row["text"], score=weaviate_score(row["_additional"], mode),
            citation=Citation(
                collection=collection, object_id=row["_additional"]["id"],
                source_version_ids=tuple(versions), first_message_id=row.get("first_message_id"),
                last_message_id=row.get("last_message_id"), thread_id=row.get("thread_id"),
                message_ids=tuple(row.get("message_ids") or ()),
                normalized_record_ids=tuple(row.get("normalized_record_ids") or ()),
                chunk_index=row.get("chunk_index"),
                chunk_id=row.get("chunk_id"), content_hash=row.get("content_hash"), vector_name=vector_name,
                record_kind=row.get("record_kind"), locator=locator,
            ),
        ))
    return hits
