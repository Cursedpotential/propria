"""Evidence and authority citation contracts.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from datetime import datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field


class EpistemicClass(str, Enum):
    ESTABLISHED_FACT = "established_fact"
    ALLEGATION = "allegation"
    INFERENCE = "inference"
    DISPUTED_ASSERTION = "disputed_assertion"
    LEGAL_THEORY = "legal_theory"
    LEGAL_CONCLUSION = "legal_conclusion"


class EvidenceCitation(BaseModel):
    statement: str
    package_id: UUID
    assertion_id: UUID
    assertion_version: int
    span_locator: str
    epistemic_class: EpistemicClass = EpistemicClass.ESTABLISHED_FACT


class AuthorityLevel(str, Enum):
    STATUTE = "statute"
    COURT_RULE = "court_rule"
    PUBLISHED_OPINION = "published_opinion"
    UNPUBLISHED_OPINION = "unpublished_opinion"
    SCAO_FORM = "scao_form"
    SECONDARY = "secondary"


class AuthorityCitation(BaseModel):
    proposition: str
    identifier: str
    pinpoint: str | None = None
    jurisdiction: str = "US-MI"
    court: str | None = None
    authority_level: AuthorityLevel
    snapshot_hash: str | None = None
    source_url: str | None = None
    retrieved_at: datetime | None = None
    currentness_checked_at: datetime | None = None
    is_citator_verified: bool = Field(
        default=False,
        description="False unless a human-reviewed subsequent-history check ran. CourtListener is not a citator.",
    )
