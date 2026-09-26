"""Entity and event extraction: request bodies and engine response shapes.

Byline: Claude Code · Opus 5.5 · 2026-09-25

The engine (Proffer starter, ``/reference-import/entities/*``) owns proposals,
validation and the commit workflow; these models only bound what crosses the
BFF. Proposal and event objects are passed through as validated dicts so a new
engine field reaches the panel without a BFF release.
"""

from __future__ import annotations

from typing import Annotated, Any, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints

from app.types.matter_mode import MatterMode

PreviewHandle = Annotated[str, StringConstraints(strip_whitespace=True, pattern=r"^[A-Za-z0-9_-]{32,128}$")]
Digest = Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]
RecordId = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=64)]
IdempotencyKey = Annotated[str, StringConstraints(strip_whitespace=True, min_length=8, max_length=200)]


class EntityExtractRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    preview_handle: PreviewHandle
    use_model: bool = True


class EntityRunRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    preview_handle: PreviewHandle


class EntityCommitRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    preview_handle: PreviewHandle
    digest: Digest


class EntityCorrectionRequest(BaseModel):
    """One owner edit. ``entity``/``event`` carry the engine's correction
    request (op, candidate_ids and op-specific fields); the engine validates
    the op and answers 422 on a bad one, 409 when the proposal changed."""

    model_config = ConfigDict(extra="forbid")

    preview_handle: PreviewHandle
    target: Literal["entity", "event"]
    entity: dict[str, Any] | None = None
    event: dict[str, Any] | None = None


class EventFromRecordRequest(BaseModel):
    """The owner's "event worth recalling" on one record of the run."""

    model_config = ConfigDict(extra="forbid")

    preview_handle: PreviewHandle
    record_id: RecordId
    title: Annotated[str, StringConstraints(max_length=200)] | None = None
    description: Annotated[str, StringConstraints(max_length=2000)] | None = None
    event_type: Annotated[str, StringConstraints(max_length=40)] | None = None


class ExtractionFlag(BaseModel):
    model_config = ConfigDict(extra="ignore")

    code: str
    detail: str = ""


class WorkflowStep(BaseModel):
    model_config = ConfigDict(extra="ignore")

    step: str
    status: Literal["pending", "running", "completed", "skipped", "failed"]
    detail: str = ""
    counts: dict[str, int] = Field(default_factory=dict)
    ref: str = ""
    flags: list[ExtractionFlag] = Field(default_factory=list)


class WorkflowProgress(BaseModel):
    model_config = ConfigDict(extra="ignore")

    workflow_id: str
    outcome: str
    steps: list[WorkflowStep] = Field(default_factory=list)
    matter_mode: MatterMode


class ExtractionStarted(BaseModel):
    model_config = ConfigDict(extra="ignore")

    workflow_id: str
    run_id: str
    extraction_id: str
    normalized_generation_id: str
    use_model: bool
    matter_mode: MatterMode


class ValidationCheck(BaseModel):
    model_config = ConfigDict(extra="ignore")

    rule: str
    status: Literal["pass", "fail"]
    reason: str = ""


class ValidationReport(BaseModel):
    model_config = ConfigDict(extra="ignore")

    ok: bool
    checks: list[ValidationCheck]
    digest: str
    counts: dict[str, int] = Field(default_factory=dict)
    matter_mode: MatterMode


class CommitStarted(BaseModel):
    model_config = ConfigDict(extra="ignore")

    workflow_id: str
    run_id: str
    commit_id: str
    counts: dict[str, int] = Field(default_factory=dict)
    matter_mode: MatterMode


class ProposalsResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")

    preview_handle: str
    normalized_generation_id: str
    matter_mode: MatterMode
    entities: Annotated[list[dict[str, Any]], Field(max_length=5000)]
    events: Annotated[list[dict[str, Any]], Field(max_length=5000)]
    extractions: list[dict[str, Any]] = Field(default_factory=list)
    entity_types: list[str] = Field(default_factory=list)
    event_types: list[str] = Field(default_factory=list)
    alias_kinds: list[str] = Field(default_factory=list)


class CorrectionApplied(BaseModel):
    model_config = ConfigDict(extra="ignore")

    ok: bool
    normalized_generation_id: str
    matter_mode: MatterMode


class MarkedEvent(BaseModel):
    model_config = ConfigDict(extra="ignore")

    event: dict[str, Any]
    source_available_from: str | None = None
    matter_mode: MatterMode


class RecordAddressee(BaseModel):
    model_config = ConfigDict(extra="ignore")

    role: str = ""
    identifier: str = ""
    display_name: str | None = None


class RecordView(BaseModel):
    model_config = ConfigDict(extra="ignore")

    record_id: str
    ordinal: int
    occurred_at: str | None = None
    source_available_from: str | None = None
    body: str = ""
    participants: list[RecordAddressee] = Field(default_factory=list)
    matter_mode: MatterMode


class RegistryEntityView(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: str
    display_name: str
    registry_type: str
    normalized_name: str = ""
    aliases: list[dict[str, Any]] | None = None


class RegistrySearchResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")

    entities: list[RegistryEntityView]
