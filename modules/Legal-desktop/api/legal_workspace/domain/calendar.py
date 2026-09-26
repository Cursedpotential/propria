"""Court calendar and historic docket timeline.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Owner-entered hearings, deadlines, filings, and orders. Not Agno's
authored factual timeline. No invented dates. Seeded list is empty.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class EventKind(str, Enum):
    HEARING = "hearing"
    DEADLINE = "deadline"
    FILING = "filing"
    ORDER = "order"
    FOC = "foc"
    CONFERENCE = "conference"
    SERVICE = "service"
    OTHER = "other"


class EventStatus(str, Enum):
    SCHEDULED = "scheduled"
    OCCURRED = "occurred"
    VACATED = "vacated"
    CONTINUED = "continued"
    UNKNOWN = "unknown"


class DocketEventCreate(BaseModel):
    occurs_at: datetime
    title: str
    kind: EventKind
    detail: str = ""
    location: str = ""
    source: str = "owner"
    confirmed: bool = False
    status: EventStatus | None = None


class DocketEvent(BaseModel):
    event_id: UUID = Field(default_factory=uuid4)
    occurs_at: datetime
    title: str
    kind: EventKind
    detail: str = ""
    location: str = ""
    source: str = "owner"
    confirmed: bool = False
    status: EventStatus = EventStatus.SCHEDULED
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    court_safe: bool = False
    disclosure: str = "legal_core"


def default_status_for(occurs_at: datetime, now: datetime | None = None) -> EventStatus:
    moment = now or datetime.now(UTC)
    if occurs_at.tzinfo is None:
        occurs_at = occurs_at.replace(tzinfo=UTC)
    if occurs_at <= moment:
        return EventStatus.OCCURRED
    return EventStatus.SCHEDULED
