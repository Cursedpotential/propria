"""Message exports as sequences of source-stated messages, grouped into threads on disk. One unit,
one job: parse.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Parses an smsbackuprestore-style XML (``<sms>``, ``<mms>``, ``<call>``) in a single streaming pass
and spills every
record into a SQLite file on the spool disk, keyed by thread. A 1.3 GB backup therefore never sits
in memory: the
parser holds one record, the spool holds the rest on disk, and the chunker later reads ONE thread at
a time in order.

Source-stated, not resolved (owner 2026-10-02: "embed the source-stated sender, not resolved
names"):

* received message / MMS: the sender is the ``address`` attribute exactly as the file states it;
* sent message / MMS: the file states no sender name for the device owner, so the sender is ``device
-> <address>``;
* the thread is the ``address`` as stated; ``contact_name`` is NOT used (it is the backup app's own
resolution);
* a call is one line in the file's single ``calls`` thread: ``<number>: call <direction>
<seconds>s``.

The parse stops, with a note, at damage in a truncated file and keeps every record read before it
(this corpus has
truncated backups).
"""

from __future__ import annotations

import sqlite3
import uuid
import xml.etree.ElementTree as ET
from collections.abc import Iterator
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

CALLS_THREAD = "[calls]"
_TEXT_TYPES = ("text/plain",)
_SMS_TYPE_SENT = {"2", "4", "5", "6"}  # sent, outbox, failed, queued
_MMS_BOX_SENT = {"2", "3", "4"}
_CALL_DIRECTION = {
    "1": "incoming",
    "2": "outgoing",
    "3": "missed",
    "4": "voicemail",
    "5": "rejected",
    "6": "blocked",
}


@dataclass(frozen=True)
class SourceMessage:
    at: datetime | None
    sender: str
    body: str
    kind: str  # sms | mms | call


def _when(raw: str | None) -> datetime | None:
    if not raw:
        return None
    try:
        value = int(raw)
    except ValueError:
        return None
    if value <= 0:
        return None
    seconds = value / 1000 if value > 10**11 else value
    try:
        return datetime.fromtimestamp(seconds, tz=UTC)
    except (OverflowError, OSError, ValueError):
        return None


def _record(element: ET.Element) -> tuple[str, SourceMessage] | None:
    tag = element.tag.rsplit("}", 1)[-1].casefold()
    attrs = element.attrib
    address = (attrs.get("address") or attrs.get("number") or "").strip()
    if tag == "sms":
        sent = (attrs.get("type") or "") in _SMS_TYPE_SENT
        sender = f"device -> {address}" if sent else address
        return address, SourceMessage(
            _when(attrs.get("date")), sender or "Unknown", attrs.get("body") or "", "sms"
        )
    if tag == "mms":
        sent = (attrs.get("msg_box") or "") in _MMS_BOX_SENT
        texts = [
            (part.attrib.get("text") or "")
            for part in element.iter()
            if part.tag.rsplit("}", 1)[-1].casefold() == "part"
            and (part.attrib.get("ct") or "").casefold() in _TEXT_TYPES
        ]
        sender = f"device -> {address}" if sent else address
        return address, SourceMessage(
            _when(attrs.get("date")), sender or "Unknown", " ".join(t for t in texts if t), "mms"
        )
    if tag == "call":
        direction = _CALL_DIRECTION.get(attrs.get("type") or "", "call")
        duration = (attrs.get("duration") or "0").strip() or "0"
        body = f"call {direction} {duration}s"
        return CALLS_THREAD, SourceMessage(
            _when(attrs.get("date")), address or "Unknown", body, "call"
        )
    return None


def parse_message_xml(handle, notes: list[str]) -> Iterator[tuple[str, SourceMessage]]:
    """Yield ``(thread, message)`` per record. Memory is one record; ``clear()`` keeps the tree from
    growing."""
    root = None
    parser = ET.iterparse(handle, events=("start", "end"))
    while True:
        try:
            event, element = next(parser)
        except StopIteration:
            return
        except ET.ParseError as error:
            notes.append(
                f"XML ends early or is malformed ({error.msg}); records up to that point were kept."
            )
            return
        if event == "start":
            if root is None:
                root = element
            continue
        tag = element.tag.rsplit("}", 1)[-1].casefold()
        if tag not in {"sms", "mms", "call"}:
            continue
        parsed = _record(element)
        element.clear()
        if root is not None:
            root.clear()
        if parsed is not None:
            yield parsed


class MessageSpool:
    """Read disk-spooled messages by thread and retain the spool in quarantine on close."""

    def __init__(self, directory: Path, name: str = "messages") -> None:
        directory.mkdir(parents=True, exist_ok=True)
        self.path = directory / f"{name}.sqlite"
        if self.path.exists():
            raise FileExistsError(f"Spool already exists: {self.path}")
        # Used from worker threads one call at a time (asyncio.to_thread); never concurrently.
        self._db = sqlite3.connect(self.path, check_same_thread=False)
        self._db.execute("PRAGMA journal_mode=OFF")
        self._db.execute("PRAGMA synchronous=OFF")
        self._db.execute(
            "CREATE TABLE m (thread TEXT NOT NULL, at REAL, ord INTEGER NOT NULL,"
            " sender TEXT NOT NULL,"
            " body TEXT NOT NULL, kind TEXT NOT NULL)"
        )
        self._ord = 0
        self._pending = 0
        self.count = 0

    def add(self, thread: str, message: SourceMessage) -> None:
        self._ord += 1
        self.count += 1
        at = message.at.timestamp() if message.at is not None else None
        self._db.execute(
            "INSERT INTO m VALUES (?,?,?,?,?,?)",
            (thread, at, self._ord, message.sender, message.body, message.kind),
        )
        self._pending += 1
        if self._pending >= 20_000:
            self._db.commit()
            self._pending = 0

    def finish(self) -> None:
        self._db.commit()
        self._db.execute("CREATE INDEX m_thread ON m (thread, at, ord)")
        self._db.commit()

    def threads(self) -> list[str]:
        return [row[0] for row in self._db.execute("SELECT DISTINCT thread FROM m ORDER BY thread")]

    def messages(self, thread: str) -> Iterator[SourceMessage]:
        # Undated messages sort last, in file order, so a thread's order is stable and total.
        cursor = self._db.execute(
            "SELECT at, sender, body, kind FROM m WHERE thread = ? ORDER BY at IS NULL, at, ord",
            (thread,),
        )
        for at, sender, body, kind in cursor:
            yield SourceMessage(
                datetime.fromtimestamp(at, tz=UTC) if at is not None else None, sender, body, kind
            )

    def close(self) -> None:
        """Close the SQLite connection and quarantine its derived spool file.

        Inputs: this spool instance. Output: None.
        Side effects: moves the file beneath the owning directory's to_be_deleted tree.
        Pick this when parsing ends to release resources while retaining recoverable data.
        Byline: Codex, 2026-10-04.
        """
        self._db.close()
        if self.path.exists():
            retained = self.path.parent / "to_be_deleted" / "message-spool" / uuid.uuid4().hex
            retained.mkdir(parents=True, exist_ok=False)
            self.path.rename(retained / self.path.name)
