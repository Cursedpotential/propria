"""Evidence investigation requests sent back to the Evidence Platform.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Missing or contradictory proof is requested here. It is never invented
as an established fact, and this store is not a docket — no hearing
dates are seeded. court_safe=false. Not exportable.
"""

from __future__ import annotations

import hashlib
import json
from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field

from legal_workspace.contracts.events import EventEnvelope
from legal_workspace.domain.factors import FactorLetter

EVIDENCE_REQUEST_CREATED = "legal.evidence_request.created.v1"


class InvestigationKind(str, Enum):
    MISSING_PROOF = "missing_proof"
    CONTRADICTION = "contradiction"


class InvestigationStatus(str, Enum):
    OPEN = "open"
    SENT = "sent"
    ANSWERED = "answered"
    DROPPED = "dropped"


class InvestigationCreate(BaseModel):
    needed: str
    why: str
    kind: InvestigationKind = InvestigationKind.MISSING_PROOF
    linked_issue: str = ""
    factor_letter: FactorLetter | None = None
    contradiction: str = ""
    existing_assertion_ids: list[UUID] = Field(default_factory=list)


class InvestigationRequest(BaseModel):
    request_id: UUID = Field(default_factory=uuid4)
    needed: str
    why: str
    kind: InvestigationKind = InvestigationKind.MISSING_PROOF
    linked_issue: str = ""
    factor_letter: FactorLetter | None = None
    contradiction: str = ""
    existing_assertion_ids: list[UUID] = Field(default_factory=list)
    status: InvestigationStatus = InvestigationStatus.OPEN
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    court_safe: bool = False
    exportable: bool = False
    disclosure: str = "work_product_claimed"
    event_type: str = EVIDENCE_REQUEST_CREATED


EvidenceInvestigationRequest = InvestigationRequest


def to_created_event(
    request: InvestigationRequest,
    *,
    matter_id: UUID,
    court_case_id: UUID | None,
    aggregate_version: int,
) -> EventEnvelope:
    payload = request.model_dump(mode="json")
    blob = json.dumps(payload, sort_keys=True, default=str, separators=(",", ":"))
    envelope = EventEnvelope(
        event_id=uuid4(),
        event_type=EVIDENCE_REQUEST_CREATED,
        occurred_at=request.created_at,
        matter_id=matter_id,
        court_case_id=court_case_id,
        aggregate_id=request.request_id,
        aggregate_version=aggregate_version,
        trace_id=f"investigation:{request.request_id}",
        payload_hash="sha256:" + hashlib.sha256(blob.encode("utf-8")).hexdigest(),
        payload=payload,
    )
    envelope.fail_closed_if_unknown()
    return envelope
