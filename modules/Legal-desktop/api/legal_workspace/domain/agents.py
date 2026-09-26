"""Narrow legal-team roles. Runs are traces, not court work.

> _Byline: Grok · grok-4.6 · 2026-08-18_
No live model call is required to record a run. Agents cannot approve,
file, serve, or establish facts. Output is never court-safe.
"""

from __future__ import annotations

import re
from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class AgentRole(str, Enum):
    INTAKE = "intake"
    RESEARCH = "research"
    ELEMENT_MAPPER = "element_mapper"
    DRAFTER = "drafter"
    CITATION = "citation"
    REDTEAM = "redteam"
    DISCOVERY = "discovery"
    FILING_CHECKER = "filing_checker"


class AgentRunStatus(str, Enum):
    RECORDED = "recorded"
    BLOCKED = "blocked"
    NEEDS_OWNER_REVIEW = "needs_owner_review"


FORBIDDEN_INTENTS = frozenset(
    {
        "approve",
        "release",
        "file",
        "serve",
        "email",
        "transmit",
        "establish_fact",
        "diagnose",
        "sign",
        "notarize",
    }
)


class AgentRunCreate(BaseModel):
    role: AgentRole | None = None
    intent: str
    prompt: str
    requested_model: str = "unevaluated-manual"
    target_type: str = "matter"
    target_id: str = "primary"


class AgentRun(BaseModel):
    run_id: UUID = Field(default_factory=uuid4)
    role: AgentRole
    intent: str
    prompt: str
    requested_model: str
    effective_model: str
    target_type: str
    target_id: str
    output: str
    status: AgentRunStatus
    cost_usd: float = 0.0
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    court_safe: bool = False
    exportable: bool = False
    disclosure: str = "private_strategy"
    epistemic_class: str = "agent_hypothesis"


def route_role(intent: str, explicit: AgentRole | None = None) -> AgentRole:
    if explicit is not None:
        return explicit
    from legal_workspace.services.routing import load_routing

    text = intent.lower()
    table = load_routing()
    for role_name, keywords in table.role_keywords.items():
        if any(word in text for word in keywords):
            try:
                return AgentRole(role_name)
            except ValueError:
                continue
    return AgentRole.INTAKE


def forbidden_intent(intent: str) -> str | None:
    lowered = intent.lower()
    for token in FORBIDDEN_INTENTS:
        pattern = token.replace("_", "[_ ]")
        if re.search(rf"\b{pattern}\b", lowered):
            return token
    return None
