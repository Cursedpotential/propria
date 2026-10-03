"""Postgres reads for the chunk units: committed threads, their messages in thread order, and call-log files.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Read-only. Postgres is the record of truth; this module never writes. A message body lives in
``context.normalized_record_identity.normalized_payload -> content -> body``, keyed by the message id (the committed
``working.message.id`` IS the normalized record id; ``third_party_message`` carries it as ``normalized_record_id``).

First-party corpus: a thread's messages are ``working.first_party_context_thread_message`` of the thread's current
version (the highest ``version_ordinal`` that is not rejected or superseded), in ``thread_ordinal`` order.
Acquired third-party corpus: ``working.third_party_message`` grouped by ``conversation_id`` and ordered by
``occurred_at`` (``working.third_party_context_thread`` holds no rows yet; the conversation is the unit until it does).
Names: a sender's entity name where known, else the registry's confirmed identifier for the number or name, else the
raw string; ``self`` is the thread's owner.
"""

from __future__ import annotations

import re
from datetime import datetime
from typing import Any, Protocol

from sqlalchemy import text
from sqlalchemy.engine import Connection

from server.context_chunks.model import (
    CORPUS_FIRST_PARTY,
    CORPUS_THIRD_PARTY,
    CallFile,
    Message,
    Thread,
    ThreadRef,
)
from server.context_chunks.render import one_line


class Source(Protocol):
    def threads_for_source_version(self, source_version_id: str) -> list[ThreadRef]: ...
    def all_threads(self) -> list[ThreadRef]: ...
    def load_thread(self, ref: ThreadRef) -> Thread: ...
    def call_source_versions(self, source_version_id: str | None = None) -> list[str]: ...
    def load_call_file(self, source_version_id: str) -> CallFile: ...
    def thread_message_counts(self) -> dict[ThreadRef, int]: ...
    def thread_message_ids(self, ref: ThreadRef) -> list[str]: ...


class Resolver:
    """Registry identifiers -> (entity id, display name): confirmed names and phone numbers (last ten digits)."""

    def __init__(self, rows: list[dict[str, Any]]):
        self._names: dict[str, tuple[str, str]] = {}
        self._phones: dict[str, tuple[str, str]] = {}
        for row in rows:
            ident = str(row["identifier"] or "").strip()
            hit = (str(row["entity_id"]), str(row["display_name"]))
            if row["kind"] == "phone":
                self._phones[re.sub(r"\D", "", ident)[-10:]] = hit
            else:
                self._names[ident.lower()] = hit

    def resolve(self, raw: str | None, owner: tuple[str | None, str | None] = (None, None)) -> tuple[str | None, str]:
        value = (raw or "").strip()
        if not value:
            return None, "Unknown"
        if value.lower() == "self":
            return owner[0], owner[1] or "Device owner"
        digits = re.sub(r"\D", "", value)
        if len(digits) >= 10 and (hit := self._phones.get(digits[-10:])):
            return hit
        if hit := self._names.get(value.lower()):
            return hit
        return None, value


def _dedupe(values: list[str]) -> list[str]:
    seen: dict[str, None] = {}
    for v in values:
        if v and v not in seen:
            seen[v] = None
    return list(seen)


class PgSource:
    def __init__(self, conn: Connection):
        self._c = conn
        self._resolver: Resolver | None = None

    # ------------------------------------------------------------- identity
    def resolver(self) -> Resolver:
        if self._resolver is None:
            rows = (
                self._c.execute(
                    text(
                        "SELECT entity_id::text AS entity_id, display_name::text AS display_name, identifier::text AS identifier, "
                        "kind FROM registry.vw_case_identifier WHERE status = 'confirmed'"
                    )
                )
                .mappings()
                .all()
            )
            self._resolver = Resolver([dict(r) for r in rows])
        return self._resolver

    # ------------------------------------------------------------- threads
    def threads_for_source_version(self, source_version_id: str) -> list[ThreadRef]:
        first = (
            self._c.execute(
                text(
                    "SELECT DISTINCT s.context_thread_id::text FROM working.first_party_context_thread_source s "
                    "WHERE s.source_version_id = CAST(:sv AS uuid) ORDER BY 1"
                ),
                {"sv": source_version_id},
            )
            .scalars()
            .all()
        )
        third = (
            self._c.execute(
                text(
                    "SELECT DISTINCT t.conversation_id::text FROM working.third_party_message t "
                    "JOIN context.normalized_record_identity n ON n.id = t.normalized_record_id "
                    "WHERE n.source_version_id = CAST(:sv AS uuid) ORDER BY 1"
                ),
                {"sv": source_version_id},
            )
            .scalars()
            .all()
        )
        return [ThreadRef(CORPUS_FIRST_PARTY, t) for t in first] + [ThreadRef(CORPUS_THIRD_PARTY, t) for t in third]

    def all_threads(self) -> list[ThreadRef]:
        first = (
            self._c.execute(
                text(
                    "SELECT context_thread_id::text FROM working.first_party_context_thread ORDER BY created_at, context_thread_id"
                )
            )
            .scalars()
            .all()
        )
        third = (
            self._c.execute(text("SELECT DISTINCT conversation_id::text FROM working.third_party_message ORDER BY 1"))
            .scalars()
            .all()
        )
        return [ThreadRef(CORPUS_FIRST_PARTY, t) for t in first] + [ThreadRef(CORPUS_THIRD_PARTY, t) for t in third]

    def thread_message_counts(self) -> dict[ThreadRef, int]:
        counts: dict[ThreadRef, int] = {}
        for tid, n in self._c.execute(text(_CURRENT_VERSION_COUNTS)).all():
            counts[ThreadRef(CORPUS_FIRST_PARTY, tid)] = int(n)
        for cid, n in self._c.execute(
            text("SELECT conversation_id::text, count(*) FROM working.third_party_message GROUP BY 1")
        ).all():
            counts[ThreadRef(CORPUS_THIRD_PARTY, cid)] = int(n)
        return counts

    def thread_message_ids(self, ref: ThreadRef) -> list[str]:
        """The thread's message ids in thread order, without bodies (cheap; used to verify chunk coverage)."""
        if ref.corpus == CORPUS_FIRST_PARTY:
            sql = (
                "WITH "
                + _CURRENT_VERSION.format(scope="AND v.context_thread_id = CAST(:t AS uuid)")
                + " SELECT tm.message_id::text FROM cur JOIN working.first_party_context_thread_message tm "
                "ON tm.thread_version_id = cur.id ORDER BY tm.thread_ordinal"
            )
        else:
            sql = (
                "SELECT t.id::text FROM working.third_party_message t WHERE t.conversation_id = CAST(:t AS uuid) "
                "ORDER BY t.occurred_at NULLS LAST, t.id"
            )
        return list(self._c.execute(text(sql), {"t": ref.thread_id}).scalars().all())

    def load_thread(self, ref: ThreadRef) -> Thread:
        if ref.corpus == CORPUS_FIRST_PARTY:
            return self._load_first_party(ref)
        if ref.corpus == CORPUS_THIRD_PARTY:
            return self._load_third_party(ref)
        raise ValueError(f"unknown corpus {ref.corpus!r}")

    def _load_first_party(self, ref: ThreadRef) -> Thread:
        owner = self._c.execute(
            text(
                "SELECT t.owner_person_id::text, e.display_name::text FROM working.first_party_context_thread t "
                "LEFT JOIN registry.entity e ON e.id = t.owner_person_id WHERE t.context_thread_id = CAST(:t AS uuid)"
            ),
            {"t": ref.thread_id},
        ).one_or_none()
        if owner is None:
            raise LookupError(f"first-party thread {ref.thread_id} is not in Postgres")
        rows = self._c.execute(text(_FIRST_PARTY_MESSAGES), {"t": ref.thread_id}).mappings().all()
        parts = self._c.execute(text(_FIRST_PARTY_PARTICIPANTS), {"t": ref.thread_id}).mappings().all()
        return self._assemble(ref, rows, parts, (owner[0], owner[1]))

    def _load_third_party(self, ref: ThreadRef) -> Thread:
        rows = self._c.execute(text(_THIRD_PARTY_MESSAGES), {"c": ref.thread_id}).mappings().all()
        parts = self._c.execute(text(_THIRD_PARTY_PARTICIPANTS), {"c": ref.thread_id}).mappings().all()
        return self._assemble(ref, rows, parts, (None, None))

    def _assemble(self, ref: ThreadRef, rows, parts, owner: tuple[str | None, str | None]) -> Thread:
        resolver = self.resolver()
        by_message: dict[str, tuple[list[str], list[str]]] = {}
        for p in parts:
            ids, names = by_message.setdefault(p["message_id"], ([], []))
            if p["entity_id"]:
                ids.append(p["entity_id"])
            entity, name = resolver.resolve(p["participant_e164"] or p["participant_raw"], owner)
            if entity:
                ids.append(entity)
            names.append(name)
        messages: list[Message] = []
        matter = None
        for r in rows:
            entity, name = resolver.resolve(r["sender_raw"], owner)
            sender_entity = r["sender_entity_id"] or entity
            if r["sender_entity_name"]:
                name = r["sender_entity_name"]
            ids, names = by_message.get(r["id"], ([], []))
            at: datetime | None = r["at"]
            messages.append(
                Message(
                    id=r["id"],
                    at=at,
                    sender=(r["sender_raw"] or "").strip(),
                    sender_name=name,
                    body=r["body"] or "",
                    source_version_id=r["svid"],
                    sender_entity_id=sender_entity,
                    participant_entity_ids=_dedupe(([sender_entity] if sender_entity else []) + ids),
                    participant_names=_dedupe([name] + names),
                )
            )
            matter = matter or r["matter_id"]
        return Thread(ref=ref, matter_id=matter, messages=messages)

    # ------------------------------------------------------------- call-log files
    def call_source_versions(self, source_version_id: str | None = None) -> list[str]:
        sql = (
            "SELECT DISTINCT n.source_version_id::text FROM working.call_log c "
            "JOIN context.normalized_record_identity n ON n.id = c.id WHERE n.source_version_id IS NOT NULL"
        )
        params: dict[str, Any] = {}
        if source_version_id:
            sql += " AND n.source_version_id = CAST(:sv AS uuid)"
            params["sv"] = source_version_id
        return list(self._c.execute(text(sql + " ORDER BY 1"), params).scalars().all())

    def load_call_file(self, source_version_id: str) -> CallFile:
        rows = self._c.execute(text(_CALLS), {"sv": source_version_id}).mappings().all()
        resolver = self.resolver()
        owner = self._owner_for_matter()
        lines, ids, entities, names = [], [], [], []
        matter = None
        for r in rows:
            outbound = r["direction"] == "outbound" or r["call_type"] in ("outgoing", "blocked_outgoing")
            raw, ent, known = (
                (r["to_raw"], r["to_entity_id"], r["to_name"])
                if outbound
                else (r["from_raw"], r["from_entity_id"], r["from_name"])
            )
            entity, name = resolver.resolve(raw, owner)
            entity = ent or entity
            if known:
                name = known
            stamp = "unknown time" if r["started_at"] is None else r["started_at"].strftime("%Y-%m-%d %H:%M")
            ids.append(r["id"])
            entities.append(entity or "")
            names.append(name)
            lines.append(
                one_line(
                    f"[{stamp}] {r['call_type']} call {'to' if outbound else 'from'} {name}, {r['duration_s'] or 0}s"
                )
            )
            matter = matter or r["matter_id"]
        times = [r["started_at"] for r in rows if r["started_at"] is not None]
        return CallFile(
            source_version_id=source_version_id,
            matter_id=matter,
            call_ids=ids,
            lines=lines,
            first_at=min(times) if times else None,
            last_at=max(times) if times else None,
            participant_entity_ids=_dedupe(entities),
            participant_names=_dedupe(names),
        )

    def _owner_for_matter(self) -> tuple[str | None, str | None]:
        row = self._c.execute(
            text(
                "SELECT t.owner_person_id::text, e.display_name::text FROM working.first_party_context_thread t "
                "LEFT JOIN registry.entity e ON e.id = t.owner_person_id ORDER BY t.created_at LIMIT 1"
            )
        ).one_or_none()
        return (row[0], row[1]) if row else (None, None)


_CURRENT_VERSION = """
cur AS (
  SELECT DISTINCT ON (v.context_thread_id) v.id, v.context_thread_id
  FROM working.first_party_context_thread_version v
  WHERE v.review_state IN ('proposed', 'approved') {scope}
  ORDER BY v.context_thread_id, v.version_ordinal DESC
)"""

_CURRENT_VERSION_COUNTS = (
    "WITH " + _CURRENT_VERSION.format(scope="") + " SELECT cur.context_thread_id::text, count(*) FROM cur "
    "JOIN working.first_party_context_thread_message tm ON tm.thread_version_id = cur.id GROUP BY 1"
)

_FIRST_PARTY_MESSAGES = (
    "WITH "
    + _CURRENT_VERSION.format(scope="AND v.context_thread_id = CAST(:t AS uuid)")
    + """
SELECT m.id::text AS id, COALESCE(tm.occurred_at, m.ts_utc) AS at, m.sender_raw,
       m.sender_entity_id::text AS sender_entity_id, se.display_name::text AS sender_entity_name,
       n.normalized_payload -> 'content' ->> 'body' AS body,
       n.source_version_id::text AS svid, sv.matter_id::text AS matter_id
FROM cur
JOIN working.first_party_context_thread_message tm ON tm.thread_version_id = cur.id
JOIN working.message m ON m.id = tm.message_id
JOIN context.normalized_record_identity n ON n.id = m.id
LEFT JOIN context.source_version sv ON sv.id = n.source_version_id
LEFT JOIN registry.entity se ON se.id = m.sender_entity_id
ORDER BY tm.thread_ordinal"""
)

_FIRST_PARTY_PARTICIPANTS = (
    "WITH "
    + _CURRENT_VERSION.format(scope="AND v.context_thread_id = CAST(:t AS uuid)")
    + """
SELECT p.message_id::text AS message_id, p.entity_id::text AS entity_id, p.participant_raw, p.participant_e164
FROM cur
JOIN working.first_party_context_thread_message tm ON tm.thread_version_id = cur.id
JOIN working.message_participant p ON p.message_id = tm.message_id"""
)

_THIRD_PARTY_MESSAGES = """
SELECT t.id::text AS id, t.occurred_at AS at, t.sender_raw, t.sender_entity_id::text AS sender_entity_id,
       se.display_name::text AS sender_entity_name,
       n.normalized_payload -> 'content' ->> 'body' AS body,
       n.source_version_id::text AS svid, sv.matter_id::text AS matter_id
FROM working.third_party_message t
JOIN context.normalized_record_identity n ON n.id = t.normalized_record_id
LEFT JOIN context.source_version sv ON sv.id = n.source_version_id
LEFT JOIN registry.entity se ON se.id = t.sender_entity_id
WHERE t.conversation_id = CAST(:c AS uuid)
ORDER BY t.occurred_at NULLS LAST, t.id"""

_THIRD_PARTY_PARTICIPANTS = """
SELECT p.message_id::text AS message_id, p.entity_id::text AS entity_id, p.participant_raw, p.participant_e164
FROM working.third_party_message_participant p
JOIN working.third_party_message t ON t.id = p.message_id
WHERE t.conversation_id = CAST(:c AS uuid)"""

_CALLS = """
SELECT c.id::text AS id, c.started_at, c.call_type, c.direction, c.duration_s, c.from_raw, c.to_raw,
       c.from_entity_id::text AS from_entity_id, c.to_entity_id::text AS to_entity_id,
       fe.display_name::text AS from_name, te.display_name::text AS to_name,
       sv.matter_id::text AS matter_id
FROM working.call_log c
JOIN context.normalized_record_identity n ON n.id = c.id
LEFT JOIN context.source_version sv ON sv.id = n.source_version_id
LEFT JOIN registry.entity fe ON fe.id = c.from_entity_id
LEFT JOIN registry.entity te ON te.id = c.to_entity_id
WHERE n.source_version_id = CAST(:sv AS uuid)
ORDER BY c.started_at NULLS LAST, c.id"""
