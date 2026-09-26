"""Private case-strategy scratch store.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Not evidence. Not authority. Not court-safe. Never exportable.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class StrategyKind(str, Enum):
    THEORY = "theory"
    STRATEGY = "strategy"
    DIRECTION = "direction"
    SCRATCH_DRAFT = "scratch_draft"
    IDEA = "idea"
    CHAT_EXTRACT = "chat_extract"


class StrategyNote(BaseModel):
    note_id: UUID = Field(default_factory=uuid4)
    kind: StrategyKind
    title: str
    body: str
    source_chat: str | None = None
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    updated_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    disclosure: str = "private_strategy"
    court_safe: bool = False
    exportable: bool = False
    epistemic_class: str = "legal_theory"


class StrategyCreate(BaseModel):
    kind: StrategyKind
    title: str
    body: str
    source_chat: str | None = None


class StrategyPatch(BaseModel):
    kind: StrategyKind | None = None
    title: str | None = None
    body: str | None = None
