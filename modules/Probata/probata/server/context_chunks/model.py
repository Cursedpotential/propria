"""Plain data shapes shared by the chunk units. No I/O.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime

CORPUS_FIRST_PARTY = "first_party"
CORPUS_THIRD_PARTY = "acquired_third_party"


@dataclass(frozen=True)
class ThreadRef:
    """One conversation: a first-party context thread, or an acquired third-party conversation."""

    corpus: str
    thread_id: str


@dataclass
class Message:
    id: str
    at: datetime | None
    sender: str
    body: str
    source_version_id: str | None = None
    sender_entity_id: str | None = None
    participant_entity_ids: list[str] = field(default_factory=list)
    participant_names: list[str] = field(default_factory=list)


@dataclass
class Thread:
    ref: ThreadRef
    matter_id: str | None
    messages: list[Message]


@dataclass
class CallFile:
    """One call-log file = one source version's call records."""

    source_version_id: str
    matter_id: str | None
    call_ids: list[str]
    lines: list[str]
    first_at: datetime | None
    last_at: datetime | None
    participant_entity_ids: list[str] = field(default_factory=list)
    participant_names: list[str] = field(default_factory=list)


@dataclass
class ThreadPlan:
    """The chunk Activity's output for one thread: spans over the thread's message order, no payloads.

    ``spans`` are inclusive [first, last] indexes into the thread's messages in thread order, already widened so
    that every chunk reaches ``overlap`` messages into the next. ``digest`` binds the plan to the exact ordered
    message ids it was cut from, so the publish unit refuses a plan whose thread has since changed.
    """

    corpus: str
    thread_id: str
    message_count: int
    digest: str
    chunker: str
    chunker_version: str
    overlap: int
    spans: list[list[int]]

    def to_dict(self) -> dict:
        return {
            "corpus": self.corpus,
            "thread_id": self.thread_id,
            "message_count": self.message_count,
            "digest": self.digest,
            "chunker": self.chunker,
            "chunker_version": self.chunker_version,
            "overlap": self.overlap,
            "spans": [list(s) for s in self.spans],
        }

    @staticmethod
    def from_dict(d: dict) -> ThreadPlan:
        return ThreadPlan(
            corpus=str(d["corpus"]),
            thread_id=str(d["thread_id"]),
            message_count=int(d["message_count"]),
            digest=str(d["digest"]),
            chunker=str(d["chunker"]),
            chunker_version=str(d["chunker_version"]),
            overlap=int(d["overlap"]),
            spans=[[int(s[0]), int(s[1])] for s in d["spans"]],
        )
