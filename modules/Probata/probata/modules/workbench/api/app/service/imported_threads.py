"""Read conversation threads and cursor-paged message projections.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

The facade owns shared cache, configuration and upstream injection seams.
Inputs/outputs remain the existing Imported read contract; no writes occur here.
"""

from __future__ import annotations

import asyncio
import re
from datetime import timedelta
from typing import Any

from app.service import imported


def _thread_title(conv: str, participants: list[dict[str, Any]], desc: dict[str, Any]) -> str:
    """Read conversation threads and cursor-paged message projections.
    """
    others = [p["label"] for p in participants if not p["mine"]]
    if desc["format"] == "Facebook":
        return others[0] if others else "Facebook conversation"
    if others:
        return ", ".join(others[:3]) + ("..." if len(others) > 3 else "")
    return imported._pretty_phone(conv) if re.fullmatch(r"\d{10}", conv) else conv


async def source_threads(source_id: str, *, limit: int, offset: int) -> dict[str, Any]:
    """Read conversation threads and cursor-paged message projections.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    matter = imported.live_matter()
    source = await imported._export_for(source_id)
    people = await asyncio.to_thread(imported._people)
    rows = await asyncio.to_thread(imported.pg.threads, matter, source["export_key"], limit=limit, offset=offset)
    more = len(rows) > limit
    rows = rows[:limit]
    parts = await asyncio.to_thread(imported.pg.thread_participants, matter, source["export_key"], [r["conv"] for r in rows])
    last = {row["conv"]: row["body"] for row in await asyncio.to_thread(imported.pg.thread_last_messages, matter, source["export_key"], [r["conv"] for r in rows])}
    by_conv: dict[str, list[dict[str, Any]]] = {}
    for part in parts:
        by_conv.setdefault(part["conv"], []).append(imported._participant(part["identifier"], people, source["owner"]))
    items = []
    for row in rows:
        seen: dict[str, dict[str, Any]] = {}
        for participant in by_conv.get(row["conv"], []):
            seen.setdefault(participant["label"], participant)
        plist = list(seen.values())
        kind = "first_party" if row["first_party"] and not row["third_party"] else (
            "third_party" if row["third_party"] and not row["first_party"] else (
                "mixed" if row["first_party"] and row["third_party"] else "unclassified"))
        items.append({
            "id": imported.encode_id(source["export_key"], row["conv"]),
            "title": imported._thread_title(row["conv"], plist, source),
            "participants": plist,
            "files": row["files"], "records": row["records"], "messages": row["msgs"], "calls": row["calls"],
            "first_at": imported._iso(row["first_at"]), "last_at": imported._iso(row["last_at"]), "last_message": last.get(row["conv"]) or "",
            "party": kind, "first_party_messages": row["first_party"], "third_party_messages": row["third_party"],
        })
    return {"source": source, "items": items, "next_offset": offset + limit if more else None}


def _message(row: dict[str, Any], people: imported._People, owner: str | None) -> dict[str, Any]:
    """Read conversation threads and cursor-paged message projections.
    """
    plist = row["participants"] or []
    sender = next((p.get("identifier") for p in plist if p.get("role") == "sender"), None)
    if sender is None and plist:
        sender = plist[0].get("identifier")
    who = imported._participant(sender, people, owner) if sender else {"id": None, "label": "Unknown", "mine": False, "person": None}
    projection = row["projection_kind"]
    return {
        "id": row["id"], "at": imported._iso(row["occurred_at"]), "body": row["body"] or "",
        "sender": who, "outgoing": bool(who["mine"]),
        "party": "first_party" if projection == "first_party" else ("third_party" if projection else None),
        "attachments": int(row["attachment_count"] or 0) if row["has_attachments"] else 0,
        "certainty": row["certainty"],
    }


async def thread_messages(
    thread_id: str, *, cursor: str | None, direction: str, limit: int, around: str | None
) -> dict[str, Any]:
    """Read conversation threads and cursor-paged message projections.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    matter = imported.live_matter()
    export_key, conv = imported.decode_id(thread_id, 2)
    exports, _ = await imported._exports()
    source = next((item for item in exports if item["export_key"] == export_key), None)
    if source is None:
        raise imported.ImportedError("Unknown thread", 404)
    people = await asyncio.to_thread(imported._people)
    ts, row_id = imported._cursor_decode(cursor)
    if around and not cursor:
        if not re.fullmatch(r"[0-9a-fA-F-]{36}", around):
            raise imported.ImportedError("Unknown message", 422)
        found = await asyncio.to_thread(imported.pg.message_time, matter, around)
        if found is None:
            raise imported.ImportedError("Unknown message", 404)
        ts = (found["occurred_at"] + timedelta(hours=12)).isoformat()
        row_id = "ffffffff-ffff-ffff-ffff-ffffffffffff"
        direction, limit = "before", max(limit, 60)
    rows = await asyncio.to_thread(
        imported.pg.messages, matter, export_key, conv, ts=ts, row_id=row_id, direction=direction, limit=limit)
    more = len(rows) > limit
    rows = rows[:limit]
    if direction == "before":
        rows.reverse()  # newest page, shown oldest-first like a chat
    items = [imported._message(row, people, source["owner"]) for row in rows]
    first = rows[0] if rows else None
    last = rows[-1] if rows else None
    if direction == "before":
        older_cursor = imported._cursor_encode(first["occurred_at"], first["id"]) if first and more else None
        # A window opened around a search hit ends before the thread's newest message.
        newer_cursor = imported._cursor_encode(last["occurred_at"], last["id"]) if last and around else None
    else:
        older_cursor = None
        newer_cursor = imported._cursor_encode(last["occurred_at"], last["id"]) if last and more else None
    return {
        "thread_id": thread_id,
        "source": {k: source[k] for k in ("id", "file_name", "format", "device", "owner", "owner_name")},
        "conversation": conv, "items": items,
        "older_cursor": older_cursor, "newer_cursor": newer_cursor, "focus": around,
    }
