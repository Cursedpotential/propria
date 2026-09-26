"""Versioned event envelope shared with the Evidence Platform.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import UUID

from pydantic import BaseModel, Field


KNOWN_EVENT_TYPES = frozenset(
    {
        "evidence.release.approved.v1",
        "evidence.assertion.superseded.v1",
        "evidence.release.revoked.v1",
        "evidence.source.quarantined.v1",
        "matter.updated.v1",
        "court_case.updated.v1",
        "legal.evidence_request.created.v1",
        "legal.work_product.review_requested.v1",
        "legal.work_product.released.v1",
        "legal.filing_package.ready.v1",
        "legal.deadline.changed.v1",
    }
)


class EventEnvelope(BaseModel):
    event_id: UUID
    event_type: str
    schema_version: str = "1.0"
    occurred_at: datetime
    matter_id: UUID
    court_case_id: UUID | None = None
    aggregate_id: UUID
    aggregate_version: int
    trace_id: str
    payload_hash: str
    payload: dict[str, Any] = Field(default_factory=dict)

    def fail_closed_if_unknown(self) -> None:
        if self.event_type not in KNOWN_EVENT_TYPES:
            raise ValueError(f"unknown event schema: {self.event_type}")
