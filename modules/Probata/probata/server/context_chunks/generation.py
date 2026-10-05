"""Chunk source for a run BEFORE the commit: the run's normalized generation, not the committed ``working.*`` tables.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner rule 2026-10-02: everything goes to Weaviate first, so it is searchable before the preview, before the owner's
decision and before the Postgres commit. The records are read from ``context.normalized_record_identity`` (one
generation); every id a chunk carries is a normalized record id. The first-party import copies those ids unchanged
(a committed ``working.message`` id IS the normalized record id), so after the commit the same ids are the Postgres
message ids and nothing is rewritten.

The conversations are cut the way the first-party import cuts them, so a chunk never straddles two threads:
messages are grouped by (corpus, conversation key) where the key is ``smsthreads.ConversationKey`` (modules/engine/
derive/smsthreads, ported here; the Go test ``TestConversationKeyVectorsSharedWithThePythonPort`` and
tests/test_context_chunks.py assert the same vectors) and the corpus is ``first_party`` when the owner is a stated party
under the run's recorded participant resolution (modules/engine/disclosure, ported here), else ``acquired_third_party``.
A message the match-up found already held by an earlier source is left out (the earlier source's chunks hold it).

This class is a ``Source`` (source.py): the same chunk and publish units run over it, and the same units run over
``PgSource`` for the re-chunk of data already committed.
"""

from __future__ import annotations

import hashlib
import re
from datetime import UTC, datetime
from typing import Any

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
from server.context_chunks.source import Resolver, _dedupe

SELF = "self"
RESOLUTION_BASIS = "owner_participant:v1"
_IGNORED_PARTIES = {"", "null", "insert-address-token", SELF}
# Closed formats shared with engine/contextsearch.IsAIChatFormat; they belong to the AI per-record lane.
_AI_CHAT_FORMATS = frozenset({
    "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json",
    "claude_conversations_json", "gemini_activity_json", "ai_markdown_transcript",
    "ai_generic_json", "ai_conversations_json", "ai_chat_file",
})


# ------------------------------------------------------------------ ports of the Go rules
def normalize_party(value: str) -> str:
    """smsthreads.normalizeParty: digits for phone numbers (a NANP leading 1 dropped), else a file-safe lower-case form."""
    value = (value or "").strip().lower()
    if value in _IGNORED_PARTIES:
        return ""
    digits = sum(1 for c in value if c.isdigit() and c.isascii())
    other = sum(1 for c in value if not (c.isascii() and c.isdigit()) and c not in "+- ().")
    if other == 0 and digits > 0:
        number = "".join(c for c in value if c.isascii() and c.isdigit())
        return number[1:] if len(number) == 11 and number[0] == "1" else number
    return "".join(c for c in value if (c.isascii() and c.isalnum() and (c.islower() or c.isdigit())) or c in "@.-")


def conversation_key(values: list[str]) -> tuple[str, list[str]]:
    """smsthreads.ConversationKey: the sorted, joined, normalized participant set; ``self`` is dropped."""
    participants = sorted({n for v in values if (n := normalize_party(v))})
    if not participants:
        return "unknown", []
    joined = "_".join(participants)
    if len(joined) > 80:
        joined = f"group-{len(participants)}-{hashlib.sha256(joined.encode()).hexdigest()[:12]}"
    return joined, participants


class ResolutionError(ValueError):
    pass


def owner_took_part(resolution: dict[str, Any], sender: str, recipients: list[str]) -> bool:
    """disclosure.Resolution.ForMessage: the owner is a stated sender or recipient, ``self`` counting for the owner only
    when the run's perspective person is the owner. Fails closed on an identifier the resolution does not hold."""
    if resolution.get("basis") != RESOLUTION_BASIS:
        raise ResolutionError(f"participant resolution basis is not {RESOLUTION_BASIS}")
    known = {i["raw"]: i for i in resolution.get("identifiers", [])}
    took_part = False
    for raw in [s for s in [sender, *recipients] if s and s.strip()]:
        if raw.strip().lower() == SELF:
            perspective = (resolution.get("perspective_person_id") or "").strip()
            if not perspective:
                raise ResolutionError('a participant is "self" but the run names no perspective person')
            took_part = took_part or perspective.lower() == (resolution.get("owner_person_id") or "").strip().lower()
            continue
        if raw not in known:
            raise ResolutionError(f"identifier {raw!r} is not in this run's participant resolution")
        took_part = took_part or bool(known[raw].get("is_owner"))
    return took_part


def parse_parties(payload: dict[str, Any]) -> tuple[list[str], str, list[str]]:
    """(all parties, sender, recipients) of a record as FirstPartyContextStore.readMessages reads them: a participant of
    unknown role counts as a recipient unless it repeats the sender."""
    parties, sender, recipients, unknown = [], "", [], []
    for party in payload.get("participants") or []:
        identifier = (party.get("identifier") or "").strip()
        if not identifier:
            continue
        parties.append(identifier)
        role = party.get("role")
        if role == "sender":
            sender = sender or identifier
        elif role == "recipient":
            recipients.append(identifier)
        else:
            unknown.append(identifier)
    recipients += [u for u in unknown if u.lower() != sender.lower()]
    return parties, sender, recipients


def _when(value: Any) -> datetime | None:
    if value is None:
        return None
    return value if value.tzinfo else value.replace(tzinfo=UTC)


# ------------------------------------------------------------------ the source
class GenerationSource:
    """Threads and the call-log file of ONE normalized generation, read-only, before anything is committed."""

    def __init__(self, conn: Connection, generation_id: str, resolution_id: str = "", matches_id: str = ""):
        """Bind a normalized generation only after proving it belongs to the human chunk lane.

        Inputs: database and generation/optional resolution/match refs; output: a read-only source or a bounded error.
        Effects: source/raw provenance read. Choose before planning or publishing, including publisher resume checks.
        """
        self._c, self.generation_id = conn, generation_id
        self._resolution_id, self._matches_id = resolution_id, matches_id
        self._threads: dict[ThreadRef, Thread] | None = None
        self._calls: CallFile | None = None
        self._resolver: Resolver | None = None
        self._assert_human_generation()

    # -- receipts ------------------------------------------------------
    def resolution(self) -> dict[str, Any] | None:
        if not self._resolution_id:
            return None
        row = self._c.execute(
            text(
                "SELECT receipt.result_ref -> 'resolution' FROM context.activity_receipt receipt "
                "JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id "
                "WHERE execution.activity_name = 'resolve_context_participants_activity' AND receipt.status = 'success' "
                "AND receipt.result_ref ->> 'ref_kind' = 'participant_resolution' AND receipt.result_ref ->> 'ref_id' = :r "
                "ORDER BY receipt.attempt DESC LIMIT 1"
            ),
            {"r": self._resolution_id},
        ).one_or_none()
        if row is None:
            raise LookupError(f"no successful receipt holds participant resolution {self._resolution_id}")
        return dict(row[0])

    def matched(self) -> set[str]:
        if not self._matches_id:
            return set()
        row = self._c.execute(
            text(
                "SELECT receipt.result_ref -> 'matches' FROM context.activity_receipt receipt "
                "JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id "
                "WHERE receipt.id = CAST(:r AS uuid) AND receipt.status = 'success' "
                "AND execution.activity_name = 'match_message_occurrences_activity'"
            ),
            {"r": self._matches_id},
        ).one_or_none()
        if row is None:
            raise LookupError(f"no successful message match receipt {self._matches_id}")
        return {str(m["record_id"]).lower() for m in (row[0] or [])}

    def _resolver_(self) -> Resolver:
        if self._resolver is None:
            rows = (
                self._c.execute(
                    text(
                        "SELECT entity_id::text AS entity_id, display_name::text AS display_name, "
                        "identifier::text AS identifier, kind FROM registry.vw_case_identifier WHERE status = 'confirmed'"
                    )
                )
                .mappings()
                .all()
            )
            self._resolver = Resolver([dict(r) for r in rows])
        return self._resolver

    def _person(self, person_id: str) -> tuple[str | None, str | None]:
        if not person_id:
            return None, None
        row = self._c.execute(
            text("SELECT id::text, display_name::text FROM registry.entity WHERE id = CAST(:p AS uuid)"),
            {"p": person_id},
        ).one_or_none()
        return (row[0], row[1]) if row else (person_id, None)

    # -- loading ---------------------------------------------------------
    def _assert_human_generation(self) -> None:
        """Reject verified AI provenance before reading records or resolving any human identities.

        Inputs: this source's generation id and database connection; output: none or a bounded lookup/value error.
        Effects: one read-only provenance query. Choose for every direct chunk/publish entry, independent of request labels.
        """
        row = self._c.execute(
            text(
                "SELECT sv.declared_format, coalesce(raw.format_id, '') "
                "FROM context.normalized_generation generation "
                "JOIN context.raw_generation raw ON raw.id = generation.raw_generation_id "
                "JOIN context.source_version sv ON sv.id = generation.source_version_id "
                "WHERE generation.id = CAST(:g AS uuid)"
            ),
            {"g": self.generation_id},
        ).one_or_none()
        if row is None:
            raise LookupError("the normalized generation has no retained source/raw provenance")
        if any(str(value or "").strip().lower() in _AI_CHAT_FORMATS for value in row):
            raise ValueError("AI chat generations use the AI per-record search lane, not human conversation chunks")

    def _records(self) -> list[dict[str, Any]]:
        rows = (
            self._c.execute(
                text(
                    "SELECT n.id::text AS id, n.record_type, n.record_ordinal, n.occurred_at, "
                    "n.source_version_id::text AS svid, n.normalized_payload AS payload, sv.matter_id::text AS matter_id "
                    "FROM context.normalized_record_identity n LEFT JOIN context.source_version sv ON sv.id = n.source_version_id "
                    "WHERE n.normalized_generation_id = CAST(:g AS uuid) ORDER BY n.record_ordinal"
                ),
                {"g": self.generation_id},
            )
            .mappings()
            .all()
        )
        return [dict(r) for r in rows]

    def _load(self) -> None:
        if self._threads is not None:
            return
        records = self._records()
        resolution = self.resolution()
        resolver = self._resolver_()
        perspective = self._person((resolution or {}).get("perspective_person_id", "") or "")
        owner = perspective if perspective[1] else self._person((resolution or {}).get("owner_person_id", "") or "")
        matched = self.matched()
        grouped: dict[ThreadRef, list[Message]] = {}
        matter: dict[ThreadRef, str | None] = {}
        call_rows: list[dict[str, Any]] = []
        for r in records:
            if r["record_type"] == "call":
                call_rows.append(r)
                continue
            if r["record_type"] != "message" or r["id"].lower() in matched:
                continue
            if resolution is None:
                raise LookupError("the generation holds messages but the run passes no participant resolution")
            payload = r["payload"]
            parties, sender, recipients = parse_parties(payload)
            key, _ = conversation_key(parties)
            if key == "unknown":
                key, _ = conversation_key([sender, *recipients])
            if key == "unknown":
                raise ValueError(f"message record {r['id']} names no party other than the device owner")
            corpus = CORPUS_FIRST_PARTY if owner_took_part(resolution, sender, recipients) else CORPUS_THIRD_PARTY
            ref = ThreadRef(corpus, f"gen-{self.generation_id}/{corpus}/{key}")
            entity, name = resolver.resolve(sender, owner)
            ids, names = [entity] if entity else [], [name]
            for p in parties:
                e, n = resolver.resolve(p, owner)
                ids += [e] if e else []
                names.append(n)
            grouped.setdefault(ref, []).append(
                Message(
                    id=r["id"],
                    at=_when(r["occurred_at"]),
                    sender=sender,
                    sender_name=name,
                    body=(payload.get("content") or {}).get("body") or (payload.get("content") or {}).get("text") or "",
                    source_version_id=r["svid"],
                    sender_entity_id=entity,
                    participant_entity_ids=_dedupe(ids),
                    participant_names=_dedupe(names),
                )
            )
            matter.setdefault(ref, r["matter_id"])
        self._threads = {
            ref: Thread(ref=ref, matter_id=matter.get(ref), messages=msgs) for ref, msgs in grouped.items()
        }
        self._calls = self._build_calls(call_rows, resolver, owner) if call_rows else None

    def _build_calls(
        self, rows: list[dict[str, Any]], resolver: Resolver, owner: tuple[str | None, str | None]
    ) -> CallFile:
        lines, ids, entities, names, times = [], [], [], [], []
        for r in rows:
            payload = r["payload"]
            content = payload.get("content") or {}
            _, sender, recipients = parse_parties(payload)
            others = [p for p in [*recipients, sender] if p and p.lower() != SELF]
            entity, name = resolver.resolve(others[0] if others else "", owner)
            at = _when(r["occurred_at"])
            kind = "missed" if content.get("missed") in (True, "true") else (content.get("disposition") or "call")
            outbound = content.get("direction") == "outgoing"
            stamp = "unknown time" if at is None else at.strftime("%Y-%m-%d %H:%M")
            lines.append(
                one_line(
                    f"[{stamp}] {kind} call {'to' if outbound else 'from'} {name}, {content.get('duration_seconds') or 0}s"
                )
            )
            ids.append(r["id"])
            entities.append(entity or "")
            names.append(name)
            if at is not None:
                times.append(at)
        return CallFile(
            source_version_id=rows[0]["svid"],
            matter_id=rows[0]["matter_id"],
            call_ids=ids,
            lines=lines,
            first_at=min(times) if times else None,
            last_at=max(times) if times else None,
            participant_entity_ids=_dedupe(entities),
            participant_names=_dedupe(names),
        )

    # -- the Source protocol ---------------------------------------------
    def threads_for_source_version(self, source_version_id: str) -> list[ThreadRef]:
        self._load()
        assert self._threads is not None
        return sorted(self._threads, key=lambda r: (r.corpus, r.thread_id))

    def all_threads(self) -> list[ThreadRef]:
        return self.threads_for_source_version("")

    def load_thread(self, ref: ThreadRef) -> Thread:
        self._load()
        assert self._threads is not None
        if ref not in self._threads:
            raise LookupError(f"thread {ref.thread_id} is not in generation {self.generation_id}")
        return self._threads[ref]

    def thread_message_ids(self, ref: ThreadRef) -> list[str]:
        return [m.id for m in self.load_thread(ref).messages]

    def thread_message_counts(self) -> dict[ThreadRef, int]:
        self._load()
        assert self._threads is not None
        return {r: len(t.messages) for r, t in self._threads.items()}

    def call_source_versions(self, source_version_id: str | None = None) -> list[str]:
        self._load()
        return [self._calls.source_version_id] if self._calls else []

    def load_call_file(self, source_version_id: str) -> CallFile:
        self._load()
        if not self._calls or self._calls.source_version_id != source_version_id:
            raise LookupError(
                f"no call records for source version {source_version_id} in generation {self.generation_id}"
            )
        return self._calls


GENERATION_THREAD_PREFIX = "gen-"
_UUID = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")


def is_uuid(value: str) -> bool:
    return bool(_UUID.match((value or "").strip().lower()))
