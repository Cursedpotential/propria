"""Read number provenance, registry identity status and unnamed activity.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

The facade owns shared cache, configuration and upstream injection seams.
Inputs/outputs remain the existing Imported read contract; no writes occur here.
"""

from __future__ import annotations

import asyncio
import re
from typing import Any

from app.service import imported


async def number_records(number: str, *, cursor: str | None, limit: int) -> dict[str, Any]:
    """Every message and call carrying this number, newest first, each with the file and conversation it came from.

    This is what the owner reads to decide whether a pending number is who it looks like: the records
    themselves, their source file (casevault key), and a link to the conversation at that message.
    """
    digits = re.sub(r"[^0-9]", "", number)
    if len(digits) == 11 and digits.startswith("1"):
        digits = digits[1:]
    if not re.fullmatch(r"[2-9][0-9]{9}", digits):
        raise imported.ImportedError("A 10-digit phone number is required", 422)
    matter = imported.live_matter()
    ts, row_id = imported._cursor_decode(cursor)
    people = await asyncio.to_thread(imported._people)
    rows = await asyncio.to_thread(imported.pg.number_records, matter, digits, ts=ts, row_id=row_id, limit=limit)
    more = len(rows) > limit
    rows = rows[:limit]
    counts = await asyncio.to_thread(imported.pg.number_record_counts, matter, digits) if not cursor else None
    items = []
    for row in rows:
        desc = imported.describe_export(row["export_key"], people)
        source = {
            "file_name": row["source_key"].rsplit("/", 1)[-1], "casevault_key": desc["casevault_key"],
            "export_file": desc["file_name"], "format": desc["format"], "device": desc["device"], "owner": desc["owner"],
            "conversation": row["conv"], "thread_id": imported.encode_id(row["export_key"], row["conv"]),
        }
        if row["record_type"] == "call":
            content = row["content"] or {}
            items.append({
                "type": "call", "id": row["id"], "at": imported._iso(row["occurred_at"]), "source": source,
                "call": {
                    "kind": "missed" if content.get("missed") else content.get("disposition") or "call",
                    "direction": content.get("direction"), "missed": bool(content.get("missed")),
                    "duration_s": content.get("duration_seconds"), "with": imported._call_party(row["participants"] or [], people),
                },
            })
        else:
            items.append({"type": "message", "id": row["id"], "at": imported._iso(row["occurred_at"]), "source": source,
                          "message": imported._message(row, people, desc["owner"])})
    return {
        "number": digits, "items": items, "counts": counts,
        "next_cursor": imported._cursor_encode(rows[-1]["occurred_at"], rows[-1]["id"]) if more and rows else None,
    }


async def _activity() -> dict[str, dict[str, Any]]:
    """number -> {messages, calls, last_at} over the live case's imported records."""
    matter = imported.live_matter()
    rows = await asyncio.to_thread(lambda: imported._cached(f"activity:{matter}", 60, lambda: imported.pg.numbers_activity(matter)))
    out: dict[str, dict[str, Any]] = {}
    for row in rows:
        entry = out.setdefault(row["number"], {"messages": 0, "calls": 0, "last_at": None})
        entry["calls" if row["record_type"] == "call" else "messages"] += row["n"]
        if row["last_at"] and (entry["last_at"] is None or row["last_at"] > entry["last_at"]):
            entry["last_at"] = row["last_at"]
    return out


async def identity() -> dict[str, Any]:
    """Named people (never placeholders) a number can be merged into, from the registry."""
    people = await asyncio.to_thread(imported._people)
    seen: dict[str, dict[str, Any]] = {}
    for who in people.by_phone.values():
        if not who["unconfirmed"]:
            seen.setdefault(who["entity_id"], {"entity_id": who["entity_id"], "name": who["name"], "short": who["person"], "role": who["role"]})
    for who in people.by_person.values():
        if who.get("entity_id") and not who["unconfirmed"]:
            seen.setdefault(who["entity_id"], {"entity_id": who["entity_id"], "name": who["name"], "short": who["person"], "role": who["role"]})
    return {"people": sorted(seen.values(), key=lambda p: (p["role"] != "user", p["name"].lower()))}


async def number_status(values: list[str]) -> dict[str, Any]:
    """For the desktop thread and calls views: is each number a named person, a placeholder, or nobody yet?"""
    people = await asyncio.to_thread(imported._people)
    out: dict[str, Any] = {}
    for value in values[:50]:
        if not imported.looks_like_phone_value(value):
            continue
        number = imported._digits(value)
        who = people.by_phone.get(number)
        if who is None:
            out[value] = {"state": "unknown", "number": number, "entity_id": None, "label": imported._pretty_phone(number)}
        else:
            out[value] = {"state": "unconfirmed" if who["unconfirmed"] else "known", "number": number,
                          "entity_id": who["entity_id"], "label": who["name"]}
    return {"items": out}


async def unknown_numbers(*, kind: str, limit: int, offset: int, q: str | None) -> dict[str, Any]:
    """ONE list of who still needs naming, most frequent first.

    - kind "no_person": phone numbers on imported rows that no registry person carries yet (the contacts
      import and the back-fill work from these); total is how many rows carry the number.
    - kind "unconfirmed": people still marked unconfirmed (a placeholder for a number nobody named, or
      a person only a contact export named), ranked by the calls and messages linked to them.
    """
    people = await asyncio.to_thread(imported._people)
    items: list[dict[str, Any]] = []
    if kind in ("all", "unconfirmed"):
        rows = await asyncio.to_thread(lambda: imported._cached("entity-activity", 30, imported.pg.entity_activity))
        activity = {row["entity_id"]: row for row in rows}
        for entity_id, person in people.unconfirmed.items():
            seen = activity.get(entity_id, {"msgs": 0, "calls": 0, "last_at": None})
            first = person["numbers"][0] if person["numbers"] else None
            items.append({
                "kind": "unconfirmed", "entity_id": entity_id, "number": first,
                "label": imported._pretty_phone(first) if first else (person["emails"][0] if person["emails"] else person["name"]),
                "name": person["name"], "named": not person["name"].startswith("Unknown "),
                "messages": int(seen["msgs"]), "calls": int(seen["calls"]), "total": int(seen["msgs"]) + int(seen["calls"]),
                "last_at": imported._iso(seen["last_at"]), "candidates": people.candidates.get(entity_id, []),
            })
    if kind in ("all", "no_person"):
        unlinked = await asyncio.to_thread(lambda: imported._cached("unlinked", 30, imported.pg.working_unlinked_numbers))
        for row in unlinked:
            number = row["number"]
            if number in people.by_phone or not re.fullmatch(r"[2-9][0-9]{9}", number):
                continue
            items.append({
                "kind": "no_person", "entity_id": None, "number": number, "label": imported._pretty_phone(number), "name": None, "named": False,
                "messages": 0, "calls": 0, "total": int(row["n"]), "last_at": None, "candidates": [],
            })
    if q:
        needle = re.sub(r"[^0-9]", "", q)
        if needle:
            items = [item for item in items if item["number"] and needle in item["number"]]
        else:
            lowered = q.strip().lower()
            items = [item for item in items if lowered in (item["name"] or "").lower()]
    items.sort(key=lambda item: (-item["total"], item["number"] or item["label"]))
    return {"items": items[offset: offset + limit], "total": len(items),
            "next_offset": offset + limit if offset + limit < len(items) else None}
