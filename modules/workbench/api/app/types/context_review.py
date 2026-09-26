"""Context review overlays for one Review message, and the hindsight-only flag.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

Who a message is to and about, whether it is about the child, and whether it is
relevant are attributed, append-only revisions. Foreshadowing ("significant in
hindsight") is a separate hindsight-only flag: an as-lived read never carries
it, and the review body cannot set it.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator

from app.types.matter_mode import MatterMode

Horizon = Literal["as_lived", "hindsight"]
Reason = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=4000)]
PartyLabel = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=200)]
RevisionRef = Annotated[str, StringConstraints(max_length=64)]


class ReviewParty(BaseModel):
    model_config = ConfigDict(extra="forbid")

    label: PartyLabel
    entity_id: str | None = None


class ReviewAssertions(BaseModel):
    model_config = ConfigDict(extra="ignore")

    addressed_to: list[ReviewParty] = Field(default_factory=list)
    about: list[ReviewParty] = Field(default_factory=list)
    about_child: Literal["yes", "no", "unsure"] | None = None
    relevant: bool | None = None


class ReviewRevision(BaseModel):
    model_config = ConfigDict(extra="ignore")

    review_ref: str
    revision: Annotated[int, Field(ge=1)]
    assertions: ReviewAssertions
    change_reason: str
    actor_username: str
    receipt_ref: str
    recorded_at: datetime


class ForeshadowingRevision(BaseModel):
    model_config = ConfigDict(extra="ignore")

    flag_ref: str
    revision: Annotated[int, Field(ge=1)]
    foreshadowing: bool
    note: str = ""
    horizon: Literal["hindsight"]
    knowledge_time: datetime
    change_reason: str
    actor_username: str
    receipt_ref: str


class ContextReviewView(BaseModel):
    """One message's review history, newest first. `foreshadowing` exists only in hindsight."""

    model_config = ConfigDict(extra="ignore")

    preview_handle: str
    message_id: str
    horizon: Horizon
    reviews: list[ReviewRevision]
    foreshadowing: list[ForeshadowingRevision] | None = None
    matter_mode: MatterMode


class ContextReviewRequest(BaseModel):
    """A new review revision. It cannot carry the foreshadowing flag (extra=forbid)."""

    model_config = ConfigDict(extra="forbid")

    supersedes_ref: RevisionRef = ""
    addressed_to: Annotated[list[ReviewParty], Field(max_length=32)] = Field(default_factory=list)
    about: Annotated[list[ReviewParty], Field(max_length=32)] = Field(default_factory=list)
    about_child: Literal["yes", "no", "unsure"] | None = None
    relevant: bool | None = None
    change_reason: Reason

    @field_validator("addressed_to", "about")
    @classmethod
    def names_are_unique(cls, parties: list[ReviewParty]) -> list[ReviewParty]:
        seen: set[str] = set()
        for party in parties:
            key = party.label.casefold()
            if key in seen:
                raise ValueError(f"{party.label!r} is listed twice")
            seen.add(key)
        return parties


class ForeshadowingRequest(BaseModel):
    """Set or clear the hindsight-only foreshadowing flag."""

    model_config = ConfigDict(extra="forbid")

    supersedes_ref: RevisionRef = ""
    foreshadowing: bool
    note: Annotated[str, StringConstraints(max_length=4000)] = ""
    change_reason: Reason


class ContextReviewReceipt(BaseModel):
    model_config = ConfigDict(extra="ignore")

    ref: str
    receipt_ref: str
    content_digest: str
    revision: Annotated[int, Field(ge=1)]
    recorded_at: datetime
    horizon: Horizon
    matter_mode: MatterMode
