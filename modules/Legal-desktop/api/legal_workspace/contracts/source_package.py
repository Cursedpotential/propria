"""Immutable approved evidence selection released to legal work.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from datetime import datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field


class ReviewState(str, Enum):
    CANDIDATE = "candidate"
    APPROVED = "approved"
    REVOKED = "revoked"
    QUARANTINED = "quarantined"


class LegalSourcePackageItem(BaseModel):
    item_id: UUID
    assertion_id: UUID
    assertion_version: int
    span_locator: str
    custody_locator: str
    content_hash: str
    review_state: ReviewState
    permitted_use: str = "legal_drafting"


class LegalSourcePackage(BaseModel):
    package_id: UUID
    schema_version: str = "1.0"
    manifest_hash: str
    matter_id: UUID
    court_case_id: UUID | None = None
    items: list[LegalSourcePackageItem] = Field(default_factory=list)
    created_at: datetime
    originating_run_id: str | None = None
    model_egress_policy: str = "verified_provider_only"
