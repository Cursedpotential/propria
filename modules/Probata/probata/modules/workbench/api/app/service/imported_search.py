"""Search conversation chunks and read previews waiting for review.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

The facade owns shared cache, configuration and upstream injection seams.
Inputs/outputs remain the existing Imported read contract; no writes occur here.
"""

from __future__ import annotations

import asyncio
import json
import re
from typing import Any

from app.service import imported


def _snippet(text: str, query: str, width: int = 420) -> str:
    """The lines of a chunk around the first line that holds a query word (or its first lines), one line per message."""
    lines = [line for line in (text or "").split("\n") if line.strip()]
    words = [w.lower() for w in query.split() if len(w) > 1]
    at = next((i for i, line in enumerate(lines) if any(w in line.lower() for w in words)), 0)
    out = "\n".join(lines[max(0, at - 1):at + 3])
    return out if len(out) <= width else out[:width].rsplit(" ", 1)[0] + "..."


def _names_label(names: list[str]) -> str:
    """Search conversation chunks and read previews waiting for review.
    """
    unique = list(dict.fromkeys(n for n in names if n))
    if not unique:
        return "Unknown"
    return ", ".join(unique[:3]) + (f" +{len(unique) - 3}" if len(unique) > 3 else "")


async def search(query: str, *, limit: int, offset: int) -> dict[str, Any]:
    """Search the conversation chunks (and the call-log files) of the live case.

    One hit is a chunk: a run of consecutive messages that the Proffer chunker kept together, each line one message.
    It opens its thread at the chunk's first message. A call-log file is one entry per file and opens no thread.
    """
    text = " ".join(query.split())[:200]
    if not text:
        raise imported.ImportedError("Type something to search for", 422)
    matter = imported.live_matter()
    graph = (
        "{ Get { %s(limit: %d, offset: %d, "
        "hybrid: {query: %s, alpha: 0, properties: [\"text\"]}, "
        "where: {path: [\"matter_id\"], operator: Equal, valueText: %s}) "
        "{ text participant_names start_at first_message_id source_version_ids source_version_id record_kind "
        "message_count _additional { score } } } }"
    ) % (imported.settings.imported_weaviate_class, limit + 1, offset, json.dumps(text), json.dumps(matter))
    try:
        async with imported.httpx.AsyncClient(timeout=15.0) as client:
            response = await client.post(f"{imported.settings.imported_weaviate_url.rstrip('/')}/v1/graphql", json={"query": graph})
        payload = response.json()
        if response.status_code >= 400 or payload.get("errors"):
            raise ValueError("search rejected")
        hits = payload["data"]["Get"][imported.settings.imported_weaviate_class] or []
    except (imported.httpx.HTTPError, ValueError, KeyError, TypeError):
        raise imported.ImportedError("Message search is unavailable right now") from None
    more = len(hits) > limit
    hits = hits[:limit]

    def version_of(hit: dict[str, Any]) -> str | None:
        versions = hit.get("source_version_ids") or []
        return (versions[0] if versions else None) or hit.get("source_version_id")

    people = await asyncio.to_thread(imported._people)
    where = await asyncio.to_thread(imported.pg.versions_to_threads, matter, sorted({v for v in map(version_of, hits) if v}))
    place = {row["id"]: row for row in where}
    items = []
    for hit in hits:
        is_calls = hit.get("record_kind") == "call_log_file"
        row = place.get(version_of(hit))
        desc = imported.describe_export(row["export_key"], people) if row else None
        items.append({
            "id": hit.get("first_message_id") or version_of(hit) or "",
            "body": imported._snippet(hit.get("text") or "", text),
            "sender": ("Call log: " if is_calls else "") + imported._names_label(hit.get("participant_names") or []),
            "at": hit.get("start_at"),
            "score": float((hit.get("_additional") or {}).get("score") or 0),
            "thread_id": None if is_calls or not row else imported.encode_id(row["export_key"], row["conv"]),
            "source": desc["file_name"] if desc else None, "format": "Calls" if is_calls else desc["format"] if desc else None,
            "device": desc["device"] if desc else None,
            "kind": "call_log" if is_calls else "conversation",
            "message_count": hit.get("message_count"),
        })
    return {"query": text, "mode": "keyword", "items": items, "next_offset": offset + limit if more else None,
            "note": "Keyword match on conversation text; each result is a run of messages, and opens the thread at its first message. "
                    "Meaning-based matching needs an embedding credential the Workbench does not hold yet."}


async def review_queue() -> dict[str, Any]:
    """Previews waiting for a decision. Read-only: the decision itself is the existing Review route."""
    matter = imported.live_matter()
    people = await asyncio.to_thread(imported._people)
    rows = await asyncio.to_thread(imported.pg.review_queue, matter)
    items = []
    for row in rows:
        desc = imported.describe_export(row["export_key"], people)
        conv = row["conv"]
        items.append({
            "preview_handle": row["preview_handle"],
            "title": imported._pretty_phone(conv) if re.fullmatch(r"\d{10}", conv) else conv,
            "file_name": desc["file_name"], "format": desc["format"], "device": desc["device"], "owner": desc["owner"],
            "records": row["records"], "waiting_since": imported._iso(row["waiting_since"]),
        })
    return {"items": items, "total": len(items)}
