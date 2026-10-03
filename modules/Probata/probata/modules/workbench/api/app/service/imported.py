"""The mobile Imported view: sources, threads, messages, calls and message search. Read-only.

Byline: Claude Code · Sonnet · 2026-10-02

Step 5 of the six steps (Preview). What it reads, and from where:

- Sources, threads, messages and calls: the platform database, `context.*` (the source versions the
  Proffer runs created and their normalized records) joined to `working.*` (the first-party /
  third-party projection and `call_log` once it has rows). Through `app.repo.imported_pg`.
- Run state (running, awaiting review, parked, failed): the Proffer starter's operation list, the
  same read the desktop Review queue uses. A source with an approved decision is committed whatever
  the operation list says; if the list is unreachable the view says so instead of guessing.
- Message search: Weaviate ProfferChunks20261002 (conversation chunks and call-log files; the messages themselves
  stay in Postgres), queried with a dict `where` filter.

An export is the original file (for example one SMS Backup & Restore XML); the Proffer run split it
into one derived file per conversation, and each derived file is its own source version. The view
lists the export as the source and the conversations inside it as threads.
"""

from __future__ import annotations

import asyncio
import base64
import json
import re
import time
from datetime import datetime, timedelta, timezone
from typing import Any

import httpx

from app.config import settings
from app.repo import imported_pg as pg
from app.service import proffer
from app.service.proffer_errors import ProfferError

ImportedError = pg.ImportedError

_STATUS_ORDER = ["failed", "parked", "awaiting_review", "running", "not_finished", "committed", "skipped"]
_LIFECYCLE_STATUS = {
    "completed": "committed",
    "running": "running",
    "awaiting_repair_decision": "parked",
    "awaiting_preview_decision": "awaiting_review",
    "failed": "failed",
    "cancelled": "failed",
    "unavailable": "failed",
}

_cache: dict[str, tuple[float, Any]] = {}


def _cached(key: str, ttl: float, build):
    now = time.monotonic()
    hit = _cache.get(key)
    if hit and now - hit[0] < ttl:
        return hit[1]
    value = build()
    _cache[key] = (now, value)
    return value


def live_matter() -> str:
    value = settings.proffer_real_matter_id.strip()
    if not value:
        raise ImportedError("The live case identity is not configured")
    return value


# --------------------------------------------------------------------------- opaque ids

def encode_id(*parts: str) -> str:
    return base64.urlsafe_b64encode("\n".join(parts).encode()).decode().rstrip("=")


def decode_id(value: str, count: int) -> list[str]:
    try:
        raw = base64.urlsafe_b64decode(value + "=" * (-len(value) % 4)).decode()
        parts = raw.split("\n")
    except (ValueError, UnicodeError):
        raise ImportedError("Unknown reference", 404) from None
    if len(parts) != count or not all(parts) or len(raw) > 2048:
        raise ImportedError("Unknown reference", 404)
    return parts


def _cursor_encode(ts: Any, row_id: str) -> str:
    return encode_id(ts.isoformat() if hasattr(ts, "isoformat") else str(ts), row_id)


def _cursor_decode(cursor: str | None) -> tuple[str | None, str | None]:
    if not cursor:
        return None, None
    ts, row_id = decode_id(cursor, 2)
    try:
        datetime.fromisoformat(ts)
    except ValueError:
        raise ImportedError("Unknown cursor", 422) from None
    if not re.fullmatch(r"[0-9a-fA-F-]{36}", row_id):
        raise ImportedError("Unknown cursor", 422)
    return ts, row_id


# --------------------------------------------------------------------------- who is who

class _People:
    """Phone numbers and names from the registry, the one identity store."""

    def __init__(self, rows: list[dict[str, Any]]):
        self.by_phone: dict[str, dict[str, str]] = {}
        self.by_person: dict[str, dict[str, str]] = {}
        self.candidates: dict[str, list[str]] = {}
        # Every person still marked unconfirmed (verification_state 'proposed'): a placeholder for a number
        # nobody named, or a person only a contact export named. Keyed by entity.
        self.unconfirmed: dict[str, dict[str, Any]] = {}
        self.names: list[tuple[str, dict[str, str]]] = []
        for row in rows:
            unconfirmed = row.get("verification_state") == "proposed"
            who = {
                "person": row["person"], "name": row["display_name"], "role": row["role_in_case"] or "",
                "entity_id": row.get("entity_id"), "unconfirmed": unconfirmed,
            }
            if unconfirmed:
                entry = self.unconfirmed.setdefault(
                    who["entity_id"], {"entity_id": who["entity_id"], "name": who["name"], "numbers": [], "emails": []})
                if row["kind"] == "phone" and _digits(row["identifier"] or "") not in entry["numbers"]:
                    entry["numbers"].append(_digits(row["identifier"] or ""))
                elif row["kind"] == "email" and row["identifier"] not in entry["emails"]:
                    entry["emails"].append(row["identifier"])
            else:
                self.by_person.setdefault(who["person"], who)
            if row["kind"] == "name" and row.get("alias_status") == "candidate" and row.get("entity_id"):
                # Names a contact export gave this number, kept unconfirmed for the owner to pick from.
                names = self.candidates.setdefault(row["entity_id"], [])
                if row.get("alias_text_raw") and row["alias_text_raw"] not in names:
                    names.append(row["alias_text_raw"])
            ident = (row["identifier"] or "").strip().lower()
            if row["kind"] == "phone":
                self.by_phone[_digits(ident)] = who
            elif row["kind"] == "name" and len(ident) >= 5 and ident not in {"owner"}:
                self.names.append((ident, who))

    def phone(self, value: str | None) -> dict[str, str] | None:
        return self.by_phone.get(_digits(value or "")) if value else None

    def in_path(self, path: str) -> dict[str, str] | None:
        lowered = path.lower()
        for ident, who in self.names:
            if re.sub(r"[^a-z]", "", ident) and re.sub(r"[^a-z]", "", ident) in lowered:
                return who
        return None


def _digits(value: str) -> str:
    digits = re.sub(r"\D", "", value)
    return digits[-10:] if len(digits) >= 10 else digits


def _people() -> _People:
    return _cached("people", 10, lambda: _People(pg.people()))


def looks_like_phone_value(value: str) -> bool:
    digits = re.sub(r"\D", "", value)
    return bool(re.fullmatch(r"\+?[\d\s().-]{10,16}", value)) and (len(digits) == 10 or (len(digits) == 11 and digits[0] == "1"))


def _pretty_phone(value: str) -> str:
    digits = _digits(value)
    if len(digits) == 10:
        return f"({digits[:3]}) {digits[3:6]}-{digits[6:]}"
    return value


_PATH_DEVICE = re.compile(r"/(sms-backup-restore(?:-calls)?)/(\d{10})(?:/|$)")


def describe_export(export_key: str, people: _People) -> dict[str, Any]:
    """Format, device phone and owner, read from the casevault key of an export."""
    lowered = export_key.lower()
    if "sms-backup-restore-calls" in lowered:
        label = "Calls"
    elif "sms-backup-restore" in lowered:
        label = "SMS"
    elif "facebook" in lowered:
        label = "Facebook"
    else:
        label = "Other"
    device = _PATH_DEVICE.search(export_key)
    device_phone = device.group(2) if device else None
    owner = people.phone(device_phone) if device_phone else people.in_path(export_key.split("/messaging/")[-1])
    return {
        "format": label,
        "device": _pretty_phone(device_phone) if device_phone else None,
        "owner": owner["person"] if owner else None,
        "owner_name": owner["name"] if owner else None,
        "file_name": export_key.rsplit("/", 1)[-1],
        "casevault_key": export_key.split("casevault/", 1)[-1],
    }


def _participant(identifier: str, people: _People, owner: str | None) -> dict[str, Any]:
    """One participant: a name when the registry knows the number, else the number itself.

    `self` in an export is the phone's owner. `mine` marks the user's own side of a conversation
    (the registry person whose role is `user`), so a chat reads the same whichever phone it came from.
    """
    if identifier == "self":
        who = people.by_person.get(owner or "")
        label = who["name"] if who else "This phone"
    else:
        who = people.phone(identifier)
        looks_like_phone = looks_like_phone_value(identifier)
        label = who["name"] if who else (_pretty_phone(identifier) if looks_like_phone else identifier)
    number = _digits(identifier) if identifier != "self" and looks_like_phone_value(identifier) else None
    return {
        "id": identifier, "label": label, "mine": bool(who and who["role"] == "user"),
        "person": who["person"] if who else None,
        # The registry person this number belongs to, whether it is still a placeholder, and the
        # 10-digit number when nobody carries it yet (the "Who is this?" control starts from these).
        "entity_id": who.get("entity_id") if who else None,
        "unconfirmed": bool(who and who.get("unconfirmed")),
        "number": number if (number and (not who or who.get("unconfirmed"))) else None,
    }


# --------------------------------------------------------------------------- run state

_lifecycle_task: asyncio.Task | None = None
_LIFECYCLE_FRESH_S = 45.0


async def _refresh_lifecycles() -> None:
    """Read every Proffer operation of the live case (the starter takes seconds per page)."""
    found: dict[str, str] = {}
    cursor: str | None = None
    try:
        for _ in range(12):
            params: dict[str, str | int] = {"limit": 100}
            if cursor:
                params["cursor"] = cursor
            response = await proffer._request("GET", "/reference-import/operations", params=params)
            payload = response.json()
            for item in payload.get("items") or []:
                ref = item.get("source_version_ref")
                if ref and item.get("matter_id") == live_matter():
                    found[ref] = item.get("lifecycle") or ""
            cursor = payload.get("next_cursor")
            if not cursor:
                break
    except (ProfferError, ValueError, ImportedError):
        previous = _cache.get("lifecycles")
        _cache["lifecycles"] = (time.monotonic(), previous[1] if previous else None)
        return
    _cache["lifecycles"] = (time.monotonic(), found)


def warm() -> None:
    """Start the run-state read in the background; callers never wait on it."""
    global _lifecycle_task
    if _lifecycle_task is None or _lifecycle_task.done():
        _lifecycle_task = asyncio.create_task(_refresh_lifecycles())


async def _lifecycles() -> dict[str, str] | None:
    """source_version_id -> Proffer lifecycle, or None until the first read has finished.

    The starter's operation list takes about ten seconds in all, so a phone never waits for it: the
    last answer is served at once and refreshed in the background. A source with an approved
    decision is committed whatever this says (that is a PostgreSQL fact).
    """
    hit = _cache.get("lifecycles")
    if hit is None or time.monotonic() - hit[0] > _LIFECYCLE_FRESH_S:
        warm()
    return hit[1] if hit else None


def _version_status(row: dict[str, Any], lifecycles: dict[str, str] | None) -> str:
    """One source version's own outcome: the publish receipt in PostgreSQL first, the run list second."""
    if row["published"] or row["approved"]:
        return "committed"
    if row["rejected"]:
        return "failed"
    if lifecycles is None:
        return "not_finished"
    lifecycle = lifecycles.get(row["id"])
    return _LIFECYCLE_STATUS.get(lifecycle or "", "not_finished")


def _is_media(key: str) -> bool:
    return ".derived/media/" in key


def _file_status(versions: list[dict[str, Any]], lifecycles: dict[str, str] | None) -> tuple[str, dict[str, Any], int]:
    """ONE status per FILE across all its source versions and attempts.

    A file that published on any attempt is Done (committed); earlier failed attempts are counted, not shown
    as the file's status. Otherwise the latest attempt's outcome stands. Derived media that never imported
    is skipped. Returns (status, the version that stands for the file's counts, failed attempts).
    """
    ordered = sorted(versions, key=lambda v: (v["version_ordinal"], v["acquired_at"]), reverse=True)
    statuses = [_version_status(v, lifecycles) for v in ordered]
    failed_attempts = sum(1 for status in statuses if status == "failed")
    for version, status in zip(ordered, statuses):
        if status == "committed":
            return "committed", version, failed_attempts
    key = ordered[0]["source_key"]
    if _is_media(key):
        return "skipped", ordered[0], failed_attempts
    return statuses[0], ordered[0], failed_attempts


def _iso(value: Any) -> str | None:
    return value.isoformat() if value is not None else None


# --------------------------------------------------------------------------- sources

def file_rows(rows: list[dict[str, Any]], lifecycles: dict[str, str] | None) -> list[dict[str, Any]]:
    """Every source FILE (one per source key) with its single status, its counts and its export."""
    by_key: dict[str, list[dict[str, Any]]] = {}
    for row in rows:
        by_key.setdefault(row["source_key"], []).append(row)
    out = []
    for key, versions in by_key.items():
        status, stand, failed_attempts = _file_status(versions, lifecycles)
        out.append({
            "key": key, "export_key": stand["export_key"], "status": status, "failed_attempts": failed_attempts,
            "attempts": len(versions), "raw": stand["raw_n"], "normalized": stand["norm_n"], "messages": stand["msgs"], "calls": stand["calls"],
            "first_at": stand["first_at"], "last_at": stand["last_at"], "acquired_at": max(v["acquired_at"] for v in versions),
            "is_parent": key == stand["export_key"], "is_media": _is_media(key), "version_id": stand["id"],
        })
    return out


async def _exports() -> tuple[list[dict[str, Any]], bool]:
    matter = live_matter()
    rows = await asyncio.to_thread(lambda: _cached(f"sv:{matter}", 15, lambda: pg.source_versions(matter)))
    lifecycles = await _lifecycles()
    people = await asyncio.to_thread(_people)
    grouped: dict[str, dict[str, Any]] = {}
    for file in file_rows(rows, lifecycles):
        group = grouped.setdefault(
            file["export_key"],
            {
                "export_key": file["export_key"], "files": 0, "raw": 0, "normalized": 0, "committed": 0,
                "messages": 0, "calls": 0, "first_at": None, "last_at": None, "imported_at": None,
                "status_counts": {}, "split": None, "_parent": None, "_children": [],
            },
        )
        if file["is_parent"]:
            group["_parent"] = file
        elif not file["is_media"]:
            group["_children"].append(file)
        if file["is_media"] and file["status"] == "skipped":
            group["status_counts"]["skipped"] = group["status_counts"].get("skipped", 0) + 1
            continue
        group["raw"] += file["raw"]
        group["normalized"] += file["normalized"]
        group["messages"] += file["messages"]
        group["calls"] += file["calls"]
        if file["status"] == "committed":
            group["committed"] += file["normalized"]
        if file["first_at"] and (group["first_at"] is None or file["first_at"] < group["first_at"]):
            group["first_at"] = file["first_at"]
        if file["last_at"] and (group["last_at"] is None or file["last_at"] > group["last_at"]):
            group["last_at"] = file["last_at"]
        if group["imported_at"] is None or file["acquired_at"] > group["imported_at"]:
            group["imported_at"] = file["acquired_at"]
    exports = []
    for group in grouped.values():
        children, parent = group.pop("_children"), group.pop("_parent")
        child_counts: dict[str, int] = {}
        for child in children:
            child_counts[child["status"]] = child_counts.get(child["status"], 0) + 1
        is_split = parent is not None and bool(children)
        if is_split:
            # The parent backup was split into one file per conversation: its own status is the children's.
            done, failed = child_counts.get("committed", 0), child_counts.get("failed", 0)
            group["split"] = {"total": len(children), "done": done, "failed": failed,
                              "in_progress": len(children) - done - failed}
            counted = children
        else:
            counted = ([parent] if parent else []) + children
        for file in counted:
            group["status_counts"][file["status"]] = group["status_counts"].get(file["status"], 0) + 1
        group["files"] = len(counted) + (1 if is_split else 0)  # the parent counts as a file
        counts = group["status_counts"]
        group["status"] = next((name for name in _STATUS_ORDER if counts.get(name)), "committed" if counts.get("skipped") else "not_finished")
        group["failed_attempts"] = sum(f["failed_attempts"] for f in counted)
        group["id"] = encode_id(group["export_key"])
        group.update(describe_export(group["export_key"], people))
        for key in ("first_at", "last_at", "imported_at"):
            group[key] = _iso(group[key])
        exports.append(group)
    exports.sort(key=lambda item: item["imported_at"] or "", reverse=True)
    return exports, lifecycles is not None


async def sources(*, limit: int, offset: int, fmt: str | None) -> dict[str, Any]:
    exports, lifecycle_ok = await _exports()
    if fmt:
        exports = [item for item in exports if item["format"].lower() == fmt.lower()]
    page = exports[offset: offset + limit]
    totals = {"files": sum(i["files"] for i in exports), "raw": sum(i["raw"] for i in exports),
              "normalized": sum(i["normalized"] for i in exports), "committed": sum(i["committed"] for i in exports)}
    return {
        "items": page, "total": len(exports), "next_offset": offset + limit if offset + limit < len(exports) else None,
        "totals": totals, "run_state_available": lifecycle_ok,
    }


async def _export_for(source_id: str) -> dict[str, Any]:
    (export_key,) = decode_id(source_id, 1)
    exports, _ = await _exports()
    for item in exports:
        if item["export_key"] == export_key:
            return item
    raise ImportedError("Unknown source", 404)


# --------------------------------------------------------------------------- threads

def _thread_title(conv: str, participants: list[dict[str, Any]], desc: dict[str, Any]) -> str:
    others = [p["label"] for p in participants if not p["mine"]]
    if desc["format"] == "Facebook":
        return others[0] if others else "Facebook conversation"
    if others:
        return ", ".join(others[:3]) + ("..." if len(others) > 3 else "")
    return _pretty_phone(conv) if re.fullmatch(r"\d{10}", conv) else conv


async def source_threads(source_id: str, *, limit: int, offset: int) -> dict[str, Any]:
    matter = live_matter()
    source = await _export_for(source_id)
    people = await asyncio.to_thread(_people)
    rows = await asyncio.to_thread(pg.threads, matter, source["export_key"], limit=limit, offset=offset)
    more = len(rows) > limit
    rows = rows[:limit]
    parts = await asyncio.to_thread(pg.thread_participants, matter, source["export_key"], [r["conv"] for r in rows])
    last = {row["conv"]: row["body"] for row in await asyncio.to_thread(pg.thread_last_messages, matter, source["export_key"], [r["conv"] for r in rows])}
    by_conv: dict[str, list[dict[str, Any]]] = {}
    for part in parts:
        by_conv.setdefault(part["conv"], []).append(_participant(part["identifier"], people, source["owner"]))
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
            "id": encode_id(source["export_key"], row["conv"]),
            "title": _thread_title(row["conv"], plist, source),
            "participants": plist,
            "files": row["files"], "records": row["records"], "messages": row["msgs"], "calls": row["calls"],
            "first_at": _iso(row["first_at"]), "last_at": _iso(row["last_at"]), "last_message": last.get(row["conv"]) or "",
            "party": kind, "first_party_messages": row["first_party"], "third_party_messages": row["third_party"],
        })
    return {"source": source, "items": items, "next_offset": offset + limit if more else None}


# --------------------------------------------------------------------------- messages

def _message(row: dict[str, Any], people: _People, owner: str | None) -> dict[str, Any]:
    plist = row["participants"] or []
    sender = next((p.get("identifier") for p in plist if p.get("role") == "sender"), None)
    if sender is None and plist:
        sender = plist[0].get("identifier")
    who = _participant(sender, people, owner) if sender else {"id": None, "label": "Unknown", "mine": False, "person": None}
    projection = row["projection_kind"]
    return {
        "id": row["id"], "at": _iso(row["occurred_at"]), "body": row["body"] or "",
        "sender": who, "outgoing": bool(who["mine"]),
        "party": "first_party" if projection == "first_party" else ("third_party" if projection else None),
        "attachments": int(row["attachment_count"] or 0) if row["has_attachments"] else 0,
        "certainty": row["certainty"],
    }


async def thread_messages(
    thread_id: str, *, cursor: str | None, direction: str, limit: int, around: str | None
) -> dict[str, Any]:
    matter = live_matter()
    export_key, conv = decode_id(thread_id, 2)
    exports, _ = await _exports()
    source = next((item for item in exports if item["export_key"] == export_key), None)
    if source is None:
        raise ImportedError("Unknown thread", 404)
    people = await asyncio.to_thread(_people)
    ts, row_id = _cursor_decode(cursor)
    if around and not cursor:
        if not re.fullmatch(r"[0-9a-fA-F-]{36}", around):
            raise ImportedError("Unknown message", 422)
        found = await asyncio.to_thread(pg.message_time, matter, around)
        if found is None:
            raise ImportedError("Unknown message", 404)
        ts = (found["occurred_at"] + timedelta(hours=12)).isoformat()
        row_id = "ffffffff-ffff-ffff-ffff-ffffffffffff"
        direction, limit = "before", max(limit, 60)
    rows = await asyncio.to_thread(
        pg.messages, matter, export_key, conv, ts=ts, row_id=row_id, direction=direction, limit=limit)
    more = len(rows) > limit
    rows = rows[:limit]
    if direction == "before":
        rows.reverse()  # newest page, shown oldest-first like a chat
    items = [_message(row, people, source["owner"]) for row in rows]
    first = rows[0] if rows else None
    last = rows[-1] if rows else None
    if direction == "before":
        older_cursor = _cursor_encode(first["occurred_at"], first["id"]) if first and more else None
        # A window opened around a search hit ends before the thread's newest message.
        newer_cursor = _cursor_encode(last["occurred_at"], last["id"]) if last and around else None
    else:
        older_cursor = None
        newer_cursor = _cursor_encode(last["occurred_at"], last["id"]) if last and more else None
    return {
        "thread_id": thread_id,
        "source": {k: source[k] for k in ("id", "file_name", "format", "device", "owner", "owner_name")},
        "conversation": conv, "items": items,
        "older_cursor": older_cursor, "newer_cursor": newer_cursor, "focus": around,
    }


# --------------------------------------------------------------------------- calls

def _call_party(plist: list[dict[str, Any]], people: _People) -> dict[str, Any]:
    ident = next((p.get("identifier") for p in plist if p.get("identifier") not in (None, "self")), None)
    if not ident:
        return {"id": None, "label": "Unknown", "mine": False, "person": None}
    return _participant(ident, people, None)


async def calls(*, cursor: str | None, limit: int) -> dict[str, Any]:
    matter = live_matter()
    people = await asyncio.to_thread(_people)
    ts, row_id = _cursor_decode(cursor)
    use_log = await asyncio.to_thread(pg.call_log_has_rows)
    items: list[dict[str, Any]] = []
    summary = None
    if use_log:
        rows = await asyncio.to_thread(pg.calls_from_log, ts=ts, row_id=row_id, limit=limit)
        for row in rows[:limit]:
            number = row["from_e164"] or row["from_raw"] if row["direction"] == "inbound" else row["to_e164"] or row["to_raw"]
            items.append({
                "id": row["id"], "at": _iso(row["occurred_at"]), "kind": row["call_type"],
                "direction": "incoming" if row["direction"] == "inbound" else "outgoing",
                "missed": row["call_type"] in ("missed", "rejected", "blocked_incoming"),
                "duration_s": row["duration_s"], "with": _call_party([{"identifier": number}], people), "device": None,
            })
        source = "working.call_log"
    else:
        rows = await asyncio.to_thread(pg.calls_normalized, matter, ts=ts, row_id=row_id, limit=limit)
        for row in rows[:limit]:
            content = row["content"] or {}
            desc = describe_export(row["export_key"], people)
            items.append({
                "id": row["id"], "at": _iso(row["occurred_at"]),
                "kind": "missed" if content.get("missed") else content.get("disposition") or "call",
                "direction": content.get("direction"), "missed": bool(content.get("missed")),
                "duration_s": content.get("duration_seconds"),
                "with": _call_party(row["participants"] or [], people), "device": desc["device"],
            })
        source = "context.normalized_record_identity (record_type call)"
    more = len(rows) > limit
    next_cursor = _cursor_encode(rows[limit - 1]["occurred_at"], rows[limit - 1]["id"]) if more else None
    if not cursor:
        summary = await asyncio.to_thread(pg.calls_summary, matter)
        summary = {**summary, "first_at": _iso(summary["first_at"]), "last_at": _iso(summary["last_at"])}
    return {"items": items, "next_cursor": next_cursor, "summary": summary, "read_from": source}


# --------------------------------------------------------------------------- search

def _snippet(text: str, query: str, width: int = 420) -> str:
    """The lines of a chunk around the first line that holds a query word (or its first lines), one line per message."""
    lines = [line for line in (text or "").split("\n") if line.strip()]
    words = [w.lower() for w in query.split() if len(w) > 1]
    at = next((i for i, line in enumerate(lines) if any(w in line.lower() for w in words)), 0)
    out = "\n".join(lines[max(0, at - 1):at + 3])
    return out if len(out) <= width else out[:width].rsplit(" ", 1)[0] + "..."


def _names_label(names: list[str]) -> str:
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
        raise ImportedError("Type something to search for", 422)
    matter = live_matter()
    graph = (
        "{ Get { %s(limit: %d, offset: %d, "
        "hybrid: {query: %s, alpha: 0, properties: [\"text\"]}, "
        "where: {path: [\"matter_id\"], operator: Equal, valueText: %s}) "
        "{ text participant_names start_at first_message_id source_version_ids source_version_id record_kind "
        "message_count _additional { score } } } }"
    ) % (settings.imported_weaviate_class, limit + 1, offset, json.dumps(text), json.dumps(matter))
    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            response = await client.post(f"{settings.imported_weaviate_url.rstrip('/')}/v1/graphql", json={"query": graph})
        payload = response.json()
        if response.status_code >= 400 or payload.get("errors"):
            raise ValueError("search rejected")
        hits = payload["data"]["Get"][settings.imported_weaviate_class] or []
    except (httpx.HTTPError, ValueError, KeyError, TypeError):
        raise ImportedError("Message search is unavailable right now") from None
    more = len(hits) > limit
    hits = hits[:limit]

    def version_of(hit: dict[str, Any]) -> str | None:
        versions = hit.get("source_version_ids") or []
        return (versions[0] if versions else None) or hit.get("source_version_id")

    people = await asyncio.to_thread(_people)
    where = await asyncio.to_thread(pg.versions_to_threads, matter, sorted({v for v in map(version_of, hits) if v}))
    place = {row["id"]: row for row in where}
    items = []
    for hit in hits:
        is_calls = hit.get("record_kind") == "call_log_file"
        row = place.get(version_of(hit))
        desc = describe_export(row["export_key"], people) if row else None
        items.append({
            "id": hit.get("first_message_id") or version_of(hit) or "",
            "body": _snippet(hit.get("text") or "", text),
            "sender": ("Call log: " if is_calls else "") + _names_label(hit.get("participant_names") or []),
            "at": hit.get("start_at"),
            "score": float((hit.get("_additional") or {}).get("score") or 0),
            "thread_id": None if is_calls or not row else encode_id(row["export_key"], row["conv"]),
            "source": desc["file_name"] if desc else None, "format": "Calls" if is_calls else desc["format"] if desc else None,
            "device": desc["device"] if desc else None,
            "kind": "call_log" if is_calls else "conversation",
            "message_count": hit.get("message_count"),
        })
    return {"query": text, "mode": "keyword", "items": items, "next_offset": offset + limit if more else None,
            "note": "Keyword match on conversation text; each result is a run of messages, and opens the thread at its first message. "
                    "Meaning-based matching needs an embedding credential the Workbench does not hold yet."}


# --------------------------------------------------------------------------- review queue

async def review_queue() -> dict[str, Any]:
    """Previews waiting for a decision. Read-only: the decision itself is the existing Review route."""
    matter = live_matter()
    people = await asyncio.to_thread(_people)
    rows = await asyncio.to_thread(pg.review_queue, matter)
    items = []
    for row in rows:
        desc = describe_export(row["export_key"], people)
        conv = row["conv"]
        items.append({
            "preview_handle": row["preview_handle"],
            "title": _pretty_phone(conv) if re.fullmatch(r"\d{10}", conv) else conv,
            "file_name": desc["file_name"], "format": desc["format"], "device": desc["device"], "owner": desc["owner"],
            "records": row["records"], "waiting_since": _iso(row["waiting_since"]),
        })
    return {"items": items, "total": len(items)}


# --------------------------------------------------------------------------- the source of one number

async def number_records(number: str, *, cursor: str | None, limit: int) -> dict[str, Any]:
    """Every message and call carrying this number, newest first, each with the file and conversation it came from.

    This is what the owner reads to decide whether a pending number is who it looks like: the records
    themselves, their source file (casevault key), and a link to the conversation at that message.
    """
    digits = re.sub(r"[^0-9]", "", number)
    if len(digits) == 11 and digits.startswith("1"):
        digits = digits[1:]
    if not re.fullmatch(r"[2-9][0-9]{9}", digits):
        raise ImportedError("A 10-digit phone number is required", 422)
    matter = live_matter()
    ts, row_id = _cursor_decode(cursor)
    people = await asyncio.to_thread(_people)
    rows = await asyncio.to_thread(pg.number_records, matter, digits, ts=ts, row_id=row_id, limit=limit)
    more = len(rows) > limit
    rows = rows[:limit]
    counts = await asyncio.to_thread(pg.number_record_counts, matter, digits) if not cursor else None
    items = []
    for row in rows:
        desc = describe_export(row["export_key"], people)
        source = {
            "file_name": row["source_key"].rsplit("/", 1)[-1], "casevault_key": desc["casevault_key"],
            "export_file": desc["file_name"], "format": desc["format"], "device": desc["device"], "owner": desc["owner"],
            "conversation": row["conv"], "thread_id": encode_id(row["export_key"], row["conv"]),
        }
        if row["record_type"] == "call":
            content = row["content"] or {}
            items.append({
                "type": "call", "id": row["id"], "at": _iso(row["occurred_at"]), "source": source,
                "call": {
                    "kind": "missed" if content.get("missed") else content.get("disposition") or "call",
                    "direction": content.get("direction"), "missed": bool(content.get("missed")),
                    "duration_s": content.get("duration_seconds"), "with": _call_party(row["participants"] or [], people),
                },
            })
        else:
            items.append({"type": "message", "id": row["id"], "at": _iso(row["occurred_at"]), "source": source,
                          "message": _message(row, people, desc["owner"])})
    return {
        "number": digits, "items": items, "counts": counts,
        "next_cursor": _cursor_encode(rows[-1]["occurred_at"], rows[-1]["id"]) if more and rows else None,
    }


# --------------------------------------------------------------------------- who is this

def invalidate() -> None:
    """Forget cached registry and activity reads after an identity change."""
    for key in [k for k in _cache if k in ("people", "entity-activity", "unlinked") or k.startswith("activity:") or k.startswith("sv:")]:
        _cache.pop(key, None)


async def _activity() -> dict[str, dict[str, Any]]:
    """number -> {messages, calls, last_at} over the live case's imported records."""
    matter = live_matter()
    rows = await asyncio.to_thread(lambda: _cached(f"activity:{matter}", 60, lambda: pg.numbers_activity(matter)))
    out: dict[str, dict[str, Any]] = {}
    for row in rows:
        entry = out.setdefault(row["number"], {"messages": 0, "calls": 0, "last_at": None})
        entry["calls" if row["record_type"] == "call" else "messages"] += row["n"]
        if row["last_at"] and (entry["last_at"] is None or row["last_at"] > entry["last_at"]):
            entry["last_at"] = row["last_at"]
    return out


async def identity() -> dict[str, Any]:
    """Named people (never placeholders) a number can be merged into, from the registry."""
    people = await asyncio.to_thread(_people)
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
    people = await asyncio.to_thread(_people)
    out: dict[str, Any] = {}
    for value in values[:50]:
        if not looks_like_phone_value(value):
            continue
        number = _digits(value)
        who = people.by_phone.get(number)
        if who is None:
            out[value] = {"state": "unknown", "number": number, "entity_id": None, "label": _pretty_phone(number)}
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
    people = await asyncio.to_thread(_people)
    items: list[dict[str, Any]] = []
    if kind in ("all", "unconfirmed"):
        rows = await asyncio.to_thread(lambda: _cached("entity-activity", 30, pg.entity_activity))
        activity = {row["entity_id"]: row for row in rows}
        for entity_id, person in people.unconfirmed.items():
            seen = activity.get(entity_id, {"msgs": 0, "calls": 0, "last_at": None})
            first = person["numbers"][0] if person["numbers"] else None
            items.append({
                "kind": "unconfirmed", "entity_id": entity_id, "number": first,
                "label": _pretty_phone(first) if first else (person["emails"][0] if person["emails"] else person["name"]),
                "name": person["name"], "named": not person["name"].startswith("Unknown "),
                "messages": int(seen["msgs"]), "calls": int(seen["calls"]), "total": int(seen["msgs"]) + int(seen["calls"]),
                "last_at": _iso(seen["last_at"]), "candidates": people.candidates.get(entity_id, []),
            })
    if kind in ("all", "no_person"):
        unlinked = await asyncio.to_thread(lambda: _cached("unlinked", 30, pg.working_unlinked_numbers))
        for row in unlinked:
            number = row["number"]
            if number in people.by_phone or not re.fullmatch(r"[2-9][0-9]{9}", number):
                continue
            items.append({
                "kind": "no_person", "entity_id": None, "number": number, "label": _pretty_phone(number), "name": None, "named": False,
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


# --------------------------------------------------------------------------- summary

async def summary() -> dict[str, Any]:
    exports, lifecycle_ok = await _exports()
    totals = {"sources": len(exports), "files": sum(i["files"] for i in exports),
              "raw": sum(i["raw"] for i in exports), "normalized": sum(i["normalized"] for i in exports),
              "committed": sum(i["committed"] for i in exports),
              "messages": sum(i["messages"] for i in exports), "calls": sum(i["calls"] for i in exports)}
    status: dict[str, int] = {}
    for item in exports:
        for name, count in item["status_counts"].items():
            status[name] = status.get(name, 0) + count
    return {"matter_id": live_matter(), "totals": totals, "files_by_status": status, "run_state_available": lifecycle_ok,
            "generated_at": datetime.now(timezone.utc).isoformat()}
