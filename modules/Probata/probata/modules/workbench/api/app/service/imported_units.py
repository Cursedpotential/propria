"""Assemble an imported unit from registered source keys and exact PG generations.

Inputs: existing opaque import IDs. Outputs: whole-unit inventory, scoped messages
and source-linked records. Reads only; directory association is not proof of identity.
Byline: ChatGPT, 2026-10-17.
"""
from __future__ import annotations

import asyncio
import base64
import hashlib
import json
import re
from datetime import datetime, timezone
from typing import Any
from uuid import UUID

from app.repo import imported_units_pg as pg
from app.service import imported

_FACEBOOK_PART = re.compile(r"^(?P<folder>.+/messages/(?:inbox|archived_threads|message_requests|filtered_threads)/[^/]+)/message_(?P<part>[1-9][0-9]*)\.json$")


def select_members(selected: dict[str, Any], exports: list[dict[str, Any]]) -> tuple[str, list[dict[str, Any]], str]:
    """Select registered Facebook siblings in the same exact qualified folder.

    Input: recorded imports, not browser filenames. Output: structural scope and
    all matching members. Pure; never merges bytes, people or source occurrences.
    """
    match = _FACEBOOK_PART.fullmatch(selected["export_key"]) if selected["format"] == "Facebook" else None
    if not match:
        return selected["export_key"], [selected], "registered_export"
    folder = match["folder"]
    members = []
    for source in exports:
        candidate = _FACEBOOK_PART.fullmatch(source["export_key"]) if source["format"] == "Facebook" else None
        if candidate and candidate["folder"] == folder:
            members.append((int(candidate["part"]), source))
    return folder, [source for _, source in sorted(members, key=lambda pair: (pair[0], pair[1]["export_key"]))], "registered_facebook_folder"


async def resolve(source_id: str, thread_id: str | None = None) -> dict[str, Any]:
    """Resolve one selected import to registered members and current generations.

    Input: import and optional existing thread IDs. Output: a versioned read scope.
    Reads PG/registry summaries only. No filesystem scan, import or graph write.
    """
    exports, _ = await imported._exports()
    selected = next((source for source in exports if source["id"] == source_id), None)
    if selected is None:
        raise imported.ImportedError("Unknown imported source", 404)
    root, members, basis = select_members(selected, exports)
    conv = None
    if thread_id and basis != "registered_facebook_folder":
        thread_export, conv = imported.decode_id(thread_id, 2)
        if thread_export != selected["export_key"]:
            raise imported.ImportedError("Conversation does not belong to the selected source", 422)
    inventory = await asyncio.to_thread(pg.snapshot, imported.live_matter(), [x["export_key"] for x in members])
    signature = hashlib.sha256(json.dumps(sorted((r["source_version_id"], r["generation_id"]) for r in inventory)).encode()).hexdigest()
    return {"source_id": source_id, "root": root, "members": members, "inventory": inventory,
            "membership_basis": basis, "conversation": conv, "generation_signature": signature,
            "read_at": datetime.now(timezone.utc).isoformat()}


def describe(scope: dict[str, Any]) -> dict[str, Any]:
    """Produce readable membership and generation coverage without claiming validity.

    Input: resolved scope. Output: normalized-count inventory and explicit limits.
    Pure; latest generations are a viewing policy, not an owner preference or proof.
    """
    members = []
    for source in scope["members"]:
        versions = [dict(row, acquired_at=imported._iso(row["acquired_at"])) for row in scope["inventory"] if row["export_key"] == source["export_key"]]
        members.append({"source": source, "versions": versions,
                        "messages": sum(row["messages"] for row in versions),
                        "normalized_records": sum(row["normalized_records"] for row in versions)})
    targets = {}
    for row in scope["inventory"]:
        if not row["messages"]:
            continue
        native_id = imported.encode_id(row["export_key"], row["conv"])
        source = next(item for item in scope["members"] if item["export_key"] == row["export_key"])
        target = targets.setdefault(native_id, {"thread_id": native_id, "source_id": source["id"],
                                              "label": row["conv"], "generation_ids": []})
        if row["generation_id"] not in target["generation_ids"]:
            target["generation_ids"].append(row["generation_id"])
    return {key: value for key, value in scope.items() if key not in {"members", "inventory"}} | {
        "members": members, "extraction_targets": list(targets.values()),
        "generation_policy": "Latest retained source version and normalization generation per exact source key; earlier versions remain in History.",
        "coverage": "Registered imported sources only. Unimported folder members and unregistered attachments are not inventoried by this read.",
        "message_count": sum(member["messages"] for member in members),
        "normalization_verified": False,
    }


def decode_cursor(cursor: str | None, scope_key: str) -> tuple[str | None, str | None]:
    """Read a cursor tied to the exact selected generation set and search scope.

    Input: opaque cursor and expected fingerprint. Output: nullable time and UUID.
    Pure validation; a changed scope requires reopening, not mixing old/new pages.
    """
    if not cursor:
        return None, None
    try:
        value = json.loads(base64.urlsafe_b64decode(cursor + "=" * (-len(cursor) % 4)))
        if not isinstance(value, dict) or set(value) != {"scope", "time", "id"}:
            raise ValueError("shape")
        row_id = str(UUID(value["id"]))
        timestamp = value["time"]
        if timestamp is not None:
            parsed = datetime.fromisoformat(timestamp)
            if parsed.tzinfo is None:
                raise ValueError("timezone")
        if value["scope"] != scope_key:
            raise imported.ImportedError("The unit or search changed. Refresh to read the current version set.", 409)
        return timestamp, row_id
    except imported.ImportedError:
        raise
    except (ValueError, TypeError, KeyError, UnicodeError, AttributeError):
        raise imported.ImportedError("Invalid unit cursor", 422) from None


async def messages(source_id: str, *, thread_id: str | None, member_id: str | None,
                   q: str, cursor: str | None, limit: int) -> dict[str, Any]:
    """Search and read all selected unit members in one ordered PG stream.

    Input: registered source, optional member/thread, literal query and page cursor.
    Output: actual normalized records with source versions and complete stored fields.
    Read-only. Historical generations remain available but are not silently combined.
    """
    scope = await resolve(source_id, thread_id)
    members = scope["members"]
    if member_id:
        members = [member for member in members if member["id"] == member_id]
        if not members:
            raise imported.ImportedError("Selected member does not belong to this unit", 422)
    keys = {member["export_key"] for member in members}
    generations = [row["generation_id"] for row in scope["inventory"]
                   if row["generation_id"] and row["export_key"] in keys
                   and (scope["conversation"] is None or row["conv"] == scope["conversation"])]
    scope_key = hashlib.sha256(json.dumps([scope["root"], scope["generation_signature"], sorted(keys), scope["conversation"], q]).encode()).hexdigest()
    after_time, after_id = decode_cursor(cursor, scope_key)
    rows = await asyncio.to_thread(pg.message_page, imported.live_matter(), generations,
                                  q=q, after_time=after_time, after_id=after_id, limit=limit) if generations else []
    more, rows = len(rows) > limit, rows[:limit]
    people = await asyncio.to_thread(imported._people)
    sources = {member["export_key"]: member for member in members}
    items = []
    for row in rows:
        if row["export_key"] not in sources or row["generation_id"] not in generations:
            raise imported.ImportedError("The record read crossed the selected unit or generation scope", 502)
        source = sources[row["export_key"]]
        item = imported._message(row, people, source["owner"])
        if row["projection_kind"] == "mixed":
            item["party"] = None
        item["source"] = {"id": source["id"], "file_name": source["file_name"], "source_key": row["source_key"],
                          "export_key": row["export_key"], "thread_id": imported.encode_id(row["export_key"], row["conv"])}
        item["projection_kind"] = row["projection_kind"]
        items.append(item)
    next_cursor = None
    if more and rows:
        next_cursor = base64.urlsafe_b64encode(json.dumps({"scope": scope_key, "time": imported._iso(rows[-1]["occurred_at"]), "id": rows[-1]["id"]}).encode()).decode().rstrip("=")
    return {"items": items, "next_cursor": next_cursor, "generation_signature": scope["generation_signature"],
            "read_at": scope["read_at"], "query": q, "search_scope": "all_selected_pg_records"}


async def get_record(source_id: str, record_id: str) -> dict[str, Any]:
    """Read one current or historical PG record only within the resolved source unit.

    Input: import ID and record UUID. Output: full normalized row and current/history
    indication. SELECT only; returns 404 for cross-unit or unavailable record IDs.
    """
    scope = await resolve(source_id)
    row = await asyncio.to_thread(pg.record, imported.live_matter(), [x["export_key"] for x in scope["members"]], record_id)
    if row is None:
        raise imported.ImportedError("Record is not available in the selected unit", 404)
    row["occurred_at"] = imported._iso(row["occurred_at"])
    row["current_generation"] = row["generation_id"] in {r["generation_id"] for r in scope["inventory"]}
    row["read_at"] = scope["read_at"]
    return row
