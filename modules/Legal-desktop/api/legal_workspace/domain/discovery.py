"""Discovery requests tied to issues and missing proof.

> _Byline: Grok · grok-4.6 · 2026-08-18_
AI may draft text. The owner controls scope, target, and service.
No request is marked served until the owner records an actual date.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class DiscoveryKind(str, Enum):
    INTERROGATORY = "interrogatory"
    RFP = "rfp"
    RFA = "rfa"
    SUBPOENA = "subpoena"


class DiscoveryStatus(str, Enum):
    DRAFT = "draft"
    READY = "ready"
    SERVED = "served"
    ANSWERED = "answered"
    DEFICIENT = "deficient"
    DROPPED = "dropped"


class DiscoveryCreate(BaseModel):
    kind: DiscoveryKind
    text: str
    purpose: str
    linked_issue: str
    missing_proof: str = ""
    target: str = "opposing_party"
    authority: str = ""


class DiscoveryRequest(BaseModel):
    request_id: UUID = Field(default_factory=uuid4)
    kind: DiscoveryKind
    text: str
    purpose: str
    linked_issue: str
    missing_proof: str = ""
    target: str = "opposing_party"
    authority: str = ""
    status: DiscoveryStatus = DiscoveryStatus.DRAFT
    served_on: datetime | None = None
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    court_safe: bool = False
    exportable: bool = False
    disclosure: str = "work_product_claimed"


class DiscoveryStatusUpdate(BaseModel):
    status: DiscoveryStatus
    served_on: datetime | None = None


def structural_discovery() -> list[DiscoveryRequest]:
    """Only requests that stand without invented facts or service dates."""
    return [
        DiscoveryRequest(
            kind=DiscoveryKind.RFP,
            text="Produce a complete copy of the last final custody / parenting-time order entered in this case, including any attachments and subsequent stipulated amendments.",
            purpose="Establish the Vodvarka lookback start. Date is unknown until the order is in hand.",
            linked_issue="Proper cause or change of circumstances (if a final order exists)",
            missing_proof="Last order date and text are not clerk-confirmed.",
            authority="MCL 722.27(1)(c); Vodvarka v Grasmeyer, 259 Mich App 499 (2003)",
        ),
        DiscoveryRequest(
            kind=DiscoveryKind.INTERROGATORY,
            text="Identify the parenting-time schedule as actually exercised in the last 90 days: dates, exchange locations, who transported the child, and any missed exchanges, without characterizing motive.",
            purpose="Support a request for parenting time in specific terms. Conduct only.",
            linked_issue="Best-interest factors (a)–(l) considered individually",
            missing_proof="No accepted package spans yet describe the actual schedule.",
            authority="MCL 722.27a(8); MCR 2.309",
        ),
    ]
