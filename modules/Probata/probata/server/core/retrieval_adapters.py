"""Adapt existing Intake and Proffer read results without network, indexing or guessed identities.

Inputs are existing API/Weaviate objects; outputs are bounded ReadHit objects with supplied citations.
Use with injected read legs; these adapters neither query a store nor confer evidence authority.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

from typing import Any

from server.core.retrieval_contracts import Citation, ReadHit


def intake_hits(payload: dict[str, Any]) -> list[ReadHit]:
    """Convert /filesystem/search results to cited hits without I/O; preserve IDs and available vault locators.

    Input is the existing response, including collection and hits; output is a list of ReadHit.
    Missing required fields fail the leg rather than inventing a source/version/hash mapping.
    """
    return [ReadHit(
        text=row["text"], score=row["score"],
        citation=Citation(
            collection=payload["collection"], object_id=row["object_id"],
            source_id=row["source_id"], document_id=row["document_id"], chunk_id=row["chunk_id"],
            vector_name=payload.get("target_vector"),
            locator={key: row[key] for key in ("source_path", "vault_key", "resolution") if row.get(key)},
        ),
    ) for row in payload["hits"]]


def proffer_hits(collection: str, rows: list[dict[str, Any]], *, vector_name: str | None = None) -> list[ReadHit]:
    """Convert Proffer chunk objects to cited hits without I/O; require real Weaviate object IDs.

    Inputs are collection, GraphQL rows and optional actual target vector; output preserves versions/message locators.
    Pick this for ProfferChunks, not raw message-event objects or the display-only Read search response.
    """
    hits = []
    for row in rows:
        versions = list(row.get("source_version_ids") or [])
        single = row.get("source_version_id")
        if single and single not in versions:
            versions.append(single)
        hits.append(ReadHit(
            text=row["text"], score=row["_additional"]["score"],
            citation=Citation(
                collection=collection, object_id=row["_additional"]["id"],
                source_version_ids=tuple(versions), first_message_id=row.get("first_message_id"),
                chunk_id=row.get("chunk_id"), content_hash=row.get("content_hash"), vector_name=vector_name,
                locator={key: row[key] for key in ("first_message_id", "thread_id") if row.get(key)},
            ),
        ))
    return hits
