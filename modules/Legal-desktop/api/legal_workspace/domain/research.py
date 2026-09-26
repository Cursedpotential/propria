"""Legal research questions for this Matter. Not a citator.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Questions, plans, adverse notes, and uncertainty live here.
Holdings are not invented. Currency flags are owner-recorded.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class ResearchStatus(str, Enum):
    OPEN = "open"
    UNRESOLVED = "unresolved"
    ANSWERED = "answered"
    DROPPED = "dropped"


class ResearchCreate(BaseModel):
    question: str
    plan: str
    adverse_notes: str = ""
    uncertainty: str = ""
    authority_ids: list[str] = Field(default_factory=list)
    jurisdiction: str = "US-MI"


class ResearchQuestion(BaseModel):
    question_id: UUID = Field(default_factory=uuid4)
    question: str
    plan: str
    adverse_notes: str = ""
    uncertainty: str = ""
    authority_ids: list[str] = Field(default_factory=list)
    jurisdiction: str = "US-MI"
    status: ResearchStatus = ResearchStatus.OPEN
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    updated_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    disclosure: str = "work_product_claimed"
    court_safe: bool = False
    exportable: bool = False


def structural_research_questions() -> list[ResearchQuestion]:
    """Only questions that stand without invented holdings or dates."""
    return [
        ResearchQuestion(
            question="What is the date of the last custody / parenting-time order for the Vodvarka lookback?",
            plan="Pull the last final order from the clerk or an approved LegalSourcePackage. Do not invent the date.",
            authority_ids=["Vodvarka v Grasmeyer, 259 Mich App 499 (2003)"],
            uncertainty="Packet GUARDRAILS: docket and last-order date are provisional until clerk-confirmed.",
        )
    ]


class CurrencyFlag(BaseModel):
    flag_id: UUID = Field(default_factory=uuid4)
    identifier: str
    note: str
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    marks_revalidation: bool = True
