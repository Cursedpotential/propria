"""The page contract shared by every entity and event extractor, and the read of one window of messages.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

A page is the default extractor's reply schema with the message labels replaced by record ids, so the messages never
travel through Temporal history: the Go side re-reads the same window by ordinal and grounds every mention in it.
The Go counterpart is ``model.ExternalPage`` in ``modules/engine/extraction/model/external.go``; the JSON field names
here are that struct's.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from sqlalchemy import text
from sqlalchemy.engine import Connection

# The window an extractor reads per Activity call (Go: flow.ExternalPageMessages).
PAGE_MESSAGES = 200

# Mentions kept per entity in a page (the Go side keeps what it can ground).
MAX_MENTIONS_PER_ENTITY = 25

EVENT_TYPES = (
    "appointment",
    "court",
    "medical",
    "school",
    "custody_exchange",
    "travel",
    "incident",
    "communication",
    "financial",
    "residence",
    "work",
    "other",
)

_FULL_DATE = re.compile(r"^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2})?$")


@dataclass(frozen=True)
class Message:
    """One message of the window, as the extractors read it."""

    record_id: str
    ordinal: int
    occurred_at: datetime | None
    body: str
    sender: str = ""
    recipients: tuple[str, ...] = ()


def read_window(conn: Connection, generation_id: str, after_ordinal: int, limit: int = PAGE_MESSAGES) -> list[Message]:
    """The messages of one normalized generation after ``after_ordinal``, in ordinal order, at most ``limit``.

    The same keyset read as the Go ``MessagePage``, so the Go side re-reads exactly this window. Read-only.
    """
    rows = conn.execute(
        text(
            "SELECT id::text AS id, record_ordinal AS ordinal, occurred_at, "
            "coalesce(normalized_payload->'content'->>'body', '') AS body, "
            "coalesce(normalized_payload->'participants', '[]'::jsonb) AS participants "
            "FROM context.normalized_record_identity "
            "WHERE normalized_generation_id = CAST(:g AS uuid) AND record_type = 'message' AND record_ordinal > :after "
            "ORDER BY record_ordinal LIMIT :limit"
        ),
        {"g": generation_id, "after": after_ordinal, "limit": max(1, min(int(limit), 1000))},
    ).mappings()
    out: list[Message] = []
    for row in rows:
        sender = ""
        recipients: list[str] = []
        for participant in row["participants"] or []:
            role = str(participant.get("role", "")).lower()
            identifier = str(participant.get("identifier", ""))
            if role in ("sender", "from") and not sender:
                sender = identifier
            elif role in ("recipient", "to", "cc", "bcc"):
                recipients.append(identifier)
        out.append(
            Message(
                record_id=row["id"],
                ordinal=int(row["ordinal"]),
                occurred_at=row["occurred_at"],
                body=row["body"],
                sender=sender,
                recipients=tuple(recipients),
            )
        )
    return out


@dataclass
class _Entity:
    name: str
    aliases: list[str] = field(default_factory=list)
    mentions: list[dict[str, str]] = field(default_factory=list)


class PageBuilder:
    """Accumulates one window's entities and events and renders the page every extractor returns.

    Entities of one kind with the same case-folded name become one entity with all their mentions, so a name that
    appears in ten messages is one entry, as the default extractor reports it.
    """

    def __init__(self, extractor: str, version: str, *, model_id: str = "") -> None:
        self.extractor = extractor
        self.version = version
        self.model_id = model_id
        self._kinds: dict[str, dict[str, _Entity]] = {"people": {}, "places": {}, "organizations": {}}
        self._events: list[dict[str, Any]] = []

    def add_entity(self, kind: str, name: str, record_id: str, surface: str | None = None) -> None:
        """Add one mention of an entity; ``kind`` is people, places or organizations."""
        name = " ".join(name.split())
        if not name or kind not in self._kinds:
            return
        entity = self._kinds[kind].setdefault(name.casefold(), _Entity(name=name))
        mention = {"record_id": record_id, "text": surface if surface else name}
        if mention not in entity.mentions and len(entity.mentions) < MAX_MENTIONS_PER_ENTITY:
            entity.mentions.append(mention)

    def add_event(
        self,
        title: str,
        record_id: str,
        *,
        description: str = "",
        date: str | None = None,
        when: str | None = None,
        event_type: str = "other",
        people: list[str] | None = None,
        places: list[str] | None = None,
        organizations: list[str] | None = None,
    ) -> None:
        """Add one event tied to one message; an unknown type becomes ``other``, a date that is not a full date is dropped."""
        title = " ".join(title.split())[:200]
        if not title:
            return
        self._events.append(
            {
                "title": title,
                "description": description[:2000],
                "record_id": record_id,
                "date": date if date and _FULL_DATE.match(date) else None,
                "when": when or None,
                "type": event_type if event_type in EVENT_TYPES else "other",
                "people": people or [],
                "places": places or [],
                "organizations": organizations or [],
            }
        )

    def page(self, window: list[Message], after_ordinal: int, *, limit: int = PAGE_MESSAGES) -> dict[str, Any]:
        """The page: what was found, and where the window ended."""
        last = window[-1].ordinal if window else after_ordinal
        return {
            "extractor": self.extractor,
            "extractor_version": self.version,
            "model_id": self.model_id,
            "after_ordinal": after_ordinal,
            "last_ordinal": last,
            "messages": len(window),
            "done": len(window) < limit,
            "people": self._render("people"),
            "places": self._render("places"),
            "organizations": self._render("organizations"),
            "events": self._events,
        }

    def _render(self, kind: str) -> list[dict[str, Any]]:
        return [
            {"name": entity.name, "aliases": entity.aliases, "mentions": entity.mentions}
            for entity in self._kinds[kind].values()
        ]


def skipped_page(extractor: str, reason: str, after_ordinal: int) -> dict[str, Any]:
    """The page of an extractor that cannot run here (for example, no model key); the Go side records the reason."""
    return {
        "extractor": extractor,
        "extractor_version": "0",
        "after_ordinal": after_ordinal,
        "last_ordinal": after_ordinal,
        "messages": 0,
        "done": True,
        "skipped": True,
        "reason": reason,
        "people": [],
        "places": [],
        "organizations": [],
        "events": [],
    }
