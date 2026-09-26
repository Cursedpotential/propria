"""Exhibit candidates from imported approved package items.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Candidates come only from LegalSourcePackage approved items.
This layer stores owner annotations, never evidence bytes.
Bates numbers are owner-entered. None are seeded.
"""

from __future__ import annotations

import re
from datetime import UTC, datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field

from legal_workspace.contracts.source_package import ReviewState


class ExhibitReadiness(str, Enum):
    CANDIDATE = "candidate"
    TECHNICALLY_READY = "technically_ready"
    REVIEWED = "reviewed"
    APPROVED_FOR_USE = "approved_for_use"


class ExhibitAnnotationCreate(BaseModel):
    item_id: UUID
    exhibit_label: str = ""
    bates_number: str = ""
    foundation: str = ""
    custody_note: str = ""
    redaction_note: str = ""
    relevance: str = ""
    issues: list[str] = Field(default_factory=list)
    objection_notes: str = ""
    readiness: ExhibitReadiness = ExhibitReadiness.CANDIDATE


class ExhibitAnnotation(ExhibitAnnotationCreate):
    annotated_at: datetime = Field(default_factory=lambda: datetime.now(UTC))


class ExhibitCandidate(BaseModel):
    item_id: UUID
    package_id: UUID
    assertion_id: UUID
    assertion_version: int
    span_locator: str
    custody_locator: str
    content_hash: str
    review_state: ReviewState
    exhibit_label: str = ""
    bates_number: str = ""
    foundation: str = ""
    custody_note: str = ""
    redaction_note: str = ""
    relevance: str = ""
    issues: list[str] = Field(default_factory=list)
    objection_notes: str = ""
    readiness: ExhibitReadiness = ExhibitReadiness.CANDIDATE
    bytes_present: bool = False


_PREFIX_RE = re.compile(r"[^A-Z0-9]+")


def bates_prefix(display_name: str) -> str:
    slug = _PREFIX_RE.sub("-", display_name.upper()).strip("-")
    return (slug[:16] or "LW").rstrip("-")


def next_bates_number(prefix: str, existing: list[str]) -> str:
    """Owner-triggered sequence. Never auto-seeded onto blank exhibits."""
    used: set[int] = set()
    head = f"{prefix}-"
    for value in existing:
        if not value.startswith(head):
            continue
        tail = value[len(head) :]
        if tail.isdigit():
            used.add(int(tail))
    n = 1
    while n in used:
        n += 1
    return f"{prefix}-{n:06d}"
