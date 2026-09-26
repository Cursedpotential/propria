"""Owner task list. Not legal advice and not a deadline calculator.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Seeded items are only structural, complete, and applicable to this
Genesee custody matter. No invented hearing dates.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class TodoStatus(str, Enum):
    OPEN = "open"
    WAITING = "waiting"
    DONE = "done"
    DROPPED = "dropped"


class CaseTodo(BaseModel):
    todo_id: UUID = Field(default_factory=uuid4)
    title: str
    detail: str
    status: TodoStatus = TodoStatus.OPEN
    source: str
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    court_safe: bool = False


class TodoCreate(BaseModel):
    title: str
    detail: str = ""
    source: str = "owner"


class TodoPatch(BaseModel):
    title: str | None = None
    detail: str | None = None
    status: TodoStatus | None = None


def structural_todos() -> list[CaseTodo]:
    """Only tasks that are complete enough to stand without invented facts."""
    return [
        CaseTodo(
            title="Confirm docket number and assigned judge/referee with the clerk",
            detail="Genesee local facts are PROVISIONAL until clerk-confirmed (custody-packet GUARDRAILS).",
            source="packet:guardrails",
        ),
        CaseTodo(
            title="Import an approved LegalSourcePackage from the Evidence Platform",
            detail="No court-facing draft until accepted facts exist. Do not invent evidence rows.",
            source="build-guide:AC-SOURCE-001",
        ),
        CaseTodo(
            title="Red-team each theory from opposing-counsel and missing-proof lenses before promoting a draft",
            detail="My private notes only. Red-team output is not a finding.",
            source="build-guide:adversarial-reviewer",
        ),
    ]
