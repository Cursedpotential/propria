"""Read cursor-paged call projections with the original log fallback.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

The facade owns shared cache, configuration and upstream injection seams.
Inputs/outputs remain the existing Imported read contract; no writes occur here.
"""

from __future__ import annotations

import asyncio
from typing import Any

from app.service import imported


def _call_party(plist: list[dict[str, Any]], people: imported._People) -> dict[str, Any]:
    """Read cursor-paged call projections with the original log fallback.
    """
    ident = next((p.get("identifier") for p in plist if p.get("identifier") not in (None, "self")), None)
    if not ident:
        return {"id": None, "label": "Unknown", "mine": False, "person": None}
    return imported._participant(ident, people, None)


async def calls(*, cursor: str | None, limit: int) -> dict[str, Any]:
    """Read cursor-paged call projections with the original log fallback.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    matter = imported.live_matter()
    people = await asyncio.to_thread(imported._people)
    ts, row_id = imported._cursor_decode(cursor)
    use_log = await asyncio.to_thread(imported.pg.call_log_has_rows)
    items: list[dict[str, Any]] = []
    imported.summary = None
    if use_log:
        rows = await asyncio.to_thread(imported.pg.calls_from_log, ts=ts, row_id=row_id, limit=limit)
        for row in rows[:limit]:
            number = row["from_e164"] or row["from_raw"] if row["direction"] == "inbound" else row["to_e164"] or row["to_raw"]
            items.append({
                "id": row["id"], "at": imported._iso(row["occurred_at"]), "kind": row["call_type"],
                "direction": "incoming" if row["direction"] == "inbound" else "outgoing",
                "missed": row["call_type"] in ("missed", "rejected", "blocked_incoming"),
                "duration_s": row["duration_s"], "with": imported._call_party([{"identifier": number}], people), "device": None,
            })
        source = "working.call_log"
    else:
        rows = await asyncio.to_thread(imported.pg.calls_normalized, matter, ts=ts, row_id=row_id, limit=limit)
        for row in rows[:limit]:
            content = row["content"] or {}
            desc = imported.describe_export(row["export_key"], people)
            items.append({
                "id": row["id"], "at": imported._iso(row["occurred_at"]),
                "kind": "missed" if content.get("missed") else content.get("disposition") or "call",
                "direction": content.get("direction"), "missed": bool(content.get("missed")),
                "duration_s": content.get("duration_seconds"),
                "with": imported._call_party(row["participants"] or [], people), "device": desc["device"],
            })
        source = "context.normalized_record_identity (record_type call)"
    more = len(rows) > limit
    next_cursor = imported._cursor_encode(rows[limit - 1]["occurred_at"], rows[limit - 1]["id"]) if more else None
    if not cursor:
        imported.summary = await asyncio.to_thread(imported.pg.calls_summary, matter)
        imported.summary = {**imported.summary, "first_at": imported._iso(imported.summary["first_at"]), "last_at": imported._iso(imported.summary["last_at"])}
    return {"items": items, "next_cursor": next_cursor, "summary": imported.summary, "read_from": source}
