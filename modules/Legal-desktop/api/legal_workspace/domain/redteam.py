"""Adversarial / red-team reviews. Scratch evaluations, never court facts.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Lenses come from the build guide (adversarial reviewer) and the
custody-packet habit of mapping both sides, missing proof, and
adverse authority. AI output here is a hypothesis, not a finding.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class RedTeamLens(str, Enum):
    OPPOSING_COUNSEL = "opposing_counsel"
    NEUTRAL_JUDGE = "neutral_judge"
    FOC_OR_REFEREE = "foc_or_referee"
    ADVERSE_AUTHORITY = "adverse_authority"
    MISSING_PROOF = "missing_proof"
    CREDIBILITY = "credibility"
    PROVISIONAL_LOCAL = "provisional_local"


class RedTeamFinding(BaseModel):
    finding_id: UUID = Field(default_factory=uuid4)
    severity: str
    claim: str
    why: str
    not_a_court_finding: bool = True


class RedTeamRun(BaseModel):
    run_id: UUID = Field(default_factory=uuid4)
    target_type: str
    target_id: str
    lens: RedTeamLens
    prompt_or_notes: str
    findings: list[RedTeamFinding] = Field(default_factory=list)
    model: str | None = None
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    disclosure: str = "private_strategy"
    court_safe: bool = False
    exportable: bool = False


class RedTeamCreate(BaseModel):
    target_type: str
    target_id: str
    lens: RedTeamLens
    prompt_or_notes: str
    findings: list[RedTeamFinding] = Field(default_factory=list)
    model: str | None = None
