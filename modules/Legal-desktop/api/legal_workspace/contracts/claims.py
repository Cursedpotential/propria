"""Private claim records; explicit evidence relationships never establish facts."""

from __future__ import annotations

from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import AwareDatetime, BaseModel, ConfigDict, Field, model_validator

ClaimKind = Literal["assertion", "allegation", "question", "theory"]
Relationship = Literal["supports", "partial", "contradicts", "context"]
FollowupKind = Literal["locate_document", "investigate", "research", "discovery"]


class StrictModel(BaseModel):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)


class OriginReference(StrictModel):
    system: Literal["probata"] = "probata"
    kind: Literal["entity", "event"]
    record_id: str = Field(min_length=1, max_length=250, pattern=r"^[^\x00-\x1f\x7f]+$")
    record_version: str = Field(min_length=1, max_length=250, pattern=r"^[^\x00-\x1f\x7f]+$")


class OriginRecord(StrictModel):
    origin: OriginReference
    title: str = ""
    payload: dict = Field(default_factory=dict)


class FromProbataCreate(StrictModel):
    origin: OriginReference
    response: str = Field(default="", max_length=20000)


class SourceReference(StrictModel):
    package_id: UUID
    item_id: UUID
    assertion_id: UUID
    assertion_version: int = Field(ge=1)
    manifest_hash: str = Field(pattern=r"^sha256:[0-9a-fA-F]{64}$")
    content_hash: str = Field(pattern=r"^sha256:[0-9a-fA-F]{64}$")
    span_locator: str = Field(min_length=1, max_length=2000)
    custody_locator: str = Field(min_length=1, max_length=2000)


class ClaimCreate(StrictModel):
    text: str = Field(min_length=1, max_length=20000)
    kind: ClaimKind = "assertion"
    claimant: str = Field(default="", max_length=250)
    response: str = Field(default="", max_length=20000)
    origin: OriginReference | None = None


class RevisionRequest(StrictModel):
    expected_revision: int = Field(ge=1)


class ClaimPatch(RevisionRequest):
    text: str | None = Field(default=None, min_length=1, max_length=20000)
    kind: ClaimKind | None = None
    claimant: str | None = Field(default=None, max_length=250)
    response: str | None = Field(default=None, max_length=20000)

    @model_validator(mode="after")
    def reject_explicit_null(self):
        if any(getattr(self, key) is None for key in self.model_fields_set - {"expected_revision"}):
            raise ValueError("Claim fields cannot be null.")
        return self


class LinkCreate(RevisionRequest):
    relationship: Relationship
    source: SourceReference | None = None
    context_reference: str = Field(default="", max_length=2000)
    note: str = Field(default="", max_length=5000)

    @model_validator(mode="after")
    def require_reference(self):
        if self.source is None and (self.relationship != "context" or not self.context_reference):
            raise ValueError("Evidence relationships require an accepted source span.")
        return self


class EvidenceLink(StrictModel):
    link_id: UUID
    relationship: Relationship
    source: SourceReference | None = None
    context_reference: str = ""
    note: str = ""
    active: bool = True
    validation_status: Literal["accepted", "context", "unavailable"]
    created_at: datetime


class LinkPatch(RevisionRequest):
    active: bool


class GapCreate(RevisionRequest):
    description: str = Field(min_length=1, max_length=5000)


class GapPatch(RevisionRequest):
    status: Literal["open", "resolved"]


class ClaimGap(StrictModel):
    gap_id: UUID
    description: str
    status: Literal["open", "resolved"] = "open"
    created_at: datetime


class FollowupCreate(RevisionRequest):
    kind: FollowupKind
    description: str = Field(min_length=1, max_length=5000)
    gap_id: UUID | None = None


class FollowupPatch(RevisionRequest):
    status: Literal["open", "done", "cancelled"]


class InvestigationSource(StrictModel):
    kind: Literal["entity", "event"]
    record_id: UUID
    record_version: str = Field(min_length=1, max_length=300)


class InvestigationResult(StrictModel):
    summary: str = Field(default="", max_length=5000)
    sources: list[InvestigationSource] = Field(default_factory=list, max_length=30)
    tool: str = Field(default="", max_length=250)
    run_id: str = Field(default="", max_length=250)


class InvestigationRequest(StrictModel):
    mode: Literal["REAL", "TEST"]
    matter_id: UUID
    court_case_id: UUID
    legal_matter_id: UUID
    claim_id: UUID
    followup_id: UUID
    question: str = Field(min_length=1, max_length=5000)
    sources: list[InvestigationSource] = Field(default_factory=list, max_length=30)


class InvestigationResponse(InvestigationRequest):
    request_id: UUID
    status: Literal["received", "running", "completed", "failed", "cancelled"]
    created_at: AwareDatetime
    updated_at: AwareDatetime
    results: list[InvestigationResult] = Field(default_factory=list, max_length=100)


class FollowupInvestigation(StrictModel):
    state: Literal["prepared", "acknowledged"] = "prepared"
    idempotency_key: UUID
    request: InvestigationRequest
    actor_uid: str = Field(min_length=1, max_length=200)
    actor_username: str = Field(min_length=1, max_length=200)
    request_id: UUID | None = None
    remote_status: Literal["received", "running", "completed", "failed", "cancelled"] | None = None
    remote_updated_at: datetime | None = None
    last_error: str | None = None
    results: list[InvestigationResult] = Field(default_factory=list, max_length=100)


class ClaimFollowup(StrictModel):
    followup_id: UUID
    investigation: FollowupInvestigation | None = None
    kind: FollowupKind
    description: str
    gap_id: UUID | None = None
    status: Literal["open", "done", "cancelled"] = "open"
    created_at: datetime


class ClaimRecord(ClaimCreate):
    claim_id: UUID
    matter_id: UUID
    revision: int
    origin_state: Literal["none", "unchanged", "changed", "unavailable"] = "none"
    origin_record: OriginRecord | None = None
    links: list[EvidenceLink] = Field(default_factory=list)
    gaps: list[ClaimGap] = Field(default_factory=list)
    followups: list[ClaimFollowup] = Field(default_factory=list)
    evidence_status: Literal[
        "evidence_needed",
        "evidence_linked",
        "partially_supported",
        "conflicting_evidence",
        "context_only",
    ] = "evidence_needed"
    created_at: datetime
    updated_at: datetime


class SourceOptions(StrictModel):
    available: bool
    reason: str | None = None
    items: list[SourceReference] = Field(default_factory=list)


class GapReportRow(StrictModel):
    claim_id: UUID
    claim_text: str
    revision: int
    evidence_status: str
    gap_id: UUID | None = None
    description: str
    followups: list[ClaimFollowup] = Field(default_factory=list)


class ClaimRevision(StrictModel):
    revision: int
    actor: str
    action: str
    occurred_at: datetime
    record: ClaimRecord
