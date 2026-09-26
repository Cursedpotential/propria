"""Your review decisions. Agents cannot approve or release.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Approval records a human verdict against a content hash. It does not
rewrite draft history. A later edit of the same section invalidates
the decision because the hash no longer matches.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class ReviewVerdict(str, Enum):
    APPROVE = "approve"
    REJECT = "reject"
    REQUEST_CHANGES = "request_changes"


class ReviewCreate(BaseModel):
    section_id: UUID
    verdict: ReviewVerdict
    rationale: str
    # Backward-compatible input only. Authorization and the persisted actor are
    # derived from the authenticated principal, never from this caller value.
    reviewer: str | None = Field(default=None, exclude=True)
    matter_id: UUID | None = None
    package_id: UUID | None = None
    manifest_hash: str | None = None
    package_schema_version: str | None = None


class ReviewDecision(BaseModel):
    review_id: UUID = Field(default_factory=uuid4)
    section_id: UUID
    work_product_id: UUID | None = None
    verdict: ReviewVerdict
    rationale: str
    reviewer: str = "owner"
    content_hash: str
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    disclosure: str = "work_product_claimed"
    court_safe: bool = False
    exportable: bool = False
