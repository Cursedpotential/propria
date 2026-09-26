"""Wire types for the decoded-source viewer (SBV output read before any ingest run).

Byline: Claude Code · Fable 5.1 · 2026-09-21.
"""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict


class DecodedThread(BaseModel):
    model_config = ConfigDict(extra="forbid")
    file: str  # opaque id of one thread file; pass it back unchanged
    thread: str
    chunk: int
    participants: list[str]
    records: int
    first_occurred_at: str | None = None
    last_occurred_at: str | None = None


class DecodedManifest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    source_ref: str
    decoded_at: str | None = None
    decoder: str | None = None
    records: int
    rejected: int
    media_objects: int
    media_bytes: int
    threads: list[DecodedThread]


class DecodedAttachment(BaseModel):
    model_config = ConfigDict(extra="forbid")
    ordinal: int
    name: str | None = None
    media_type: str | None = None
    sha256: str | None = None
    byte_length: int | None = None


class DecodedMessage(BaseModel):
    model_config = ConfigDict(extra="forbid")
    ordinal: int
    kind: str
    status: str
    occurred_at: str | None = None
    sender: str | None = None
    participants: list[str]
    body: str
    attachments: list[DecodedAttachment]
    missing_attachments: int  # named in the backup with no bytes, or failed to decode


class DecodedThreadPage(BaseModel):
    model_config = ConfigDict(extra="forbid")
    file: str
    offset: int
    total_records: int
    messages: list[DecodedMessage]
    next_offset: int | None = None
