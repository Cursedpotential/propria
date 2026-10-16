"""Typed boundary models for the Proffer starter.

Byline: Codex · GPT-5 · 2026-08-28.
"""

from __future__ import annotations

import json
from datetime import datetime
from typing import Annotated, Any, Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator, model_validator

from app.types.matter_mode import MatterMode
from app.types.proffer_content import ProfferContentResponse  # noqa: F401
from app.types.proffer_handler import (
    ProfferHandlerCandidate,
    ProfferHandlerSelectionDecisionRequest,  # noqa: F401
    ProfferHandlerSelectionDecisionResponse,  # noqa: F401
)
from app.types.proffer_messages import (
    ProfferPreviewAttachment,  # noqa: F401
    ProfferPreviewEvent,  # noqa: F401
    ProfferPreviewMessage,  # noqa: F401
    ProfferPreviewMessagesResponse,  # noqa: F401
    ProfferPreviewParticipant,  # noqa: F401
)
from app.types.proffer_sources import (
    ProfferSourceBrowserResponse,  # noqa: F401
    ProfferSourceObject,  # noqa: F401
    ProfferSourcePrefix,  # noqa: F401
    ProfferSourceRoot,  # noqa: F401
    SourceFileKind,  # noqa: F401
    SourceLocation,  # noqa: F401
)
from app.types.source_roots import validate_authorized_source_ref

NonBlank = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1)]
OpaquePreviewHandle = Annotated[
    str,
    StringConstraints(strip_whitespace=True, pattern=r"^[A-Za-z0-9_-]{32,128}$"),
]
Sha256Digest = Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]
BoundedReason = Annotated[str, StringConstraints(max_length=4000)]
OpaqueCursor = Annotated[str, StringConstraints(min_length=1, max_length=512)]
BoundedActorIdentity = Annotated[
    str,
    StringConstraints(strip_whitespace=True, min_length=1, max_length=512, pattern=r"^[^\x00\r\n]+$"),
]
BoundedToolID = Annotated[str, StringConstraints(strip_whitespace=True, max_length=256)]
ProfferOperationLifecycle = Literal[
    "running",
    "awaiting_repair_decision",
    "awaiting_preview_decision",
    "completed",
    "failed",
    "rerun_required",
    "cancelled",  # D05-C06, Claude Code · Opus 5.5 · 2026-09-28
    "unavailable",
]
ProfferOperationWait = Literal["repair_decision", "preview_decision"]


class ProfferStartRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    request_id: NonBlank
    matter_id: UUID
    court_case_id: UUID
    source_ref: NonBlank
    declared_format: NonBlank
    parser_options_ref: NonBlank
    source_context_ref: UUID | None = None
    # First-party context import (D04): the case owner and whose device or
    # export the source is, as registry.person ids. Passed through unchanged;
    # the engine fails a message run loudly when either is absent.
    # Byline: Claude Code · Opus 5.5 · 2026-10-01
    owner_person_id: UUID | None = None
    perspective_person_id: UUID | None = None
    matter_mode: MatterMode = "LIVE"

    @field_validator("source_ref")
    @classmethod
    def source_must_be_authorized(cls, value: str) -> str:
        return validate_authorized_source_ref(value)


class ProfferStartResponse(BaseModel):
    # Go may add non-security response metadata without breaking this typed BFF.
    model_config = ConfigDict(extra="ignore")

    preview_handle: OpaquePreviewHandle
    matter_mode: MatterMode


class ProfferUploadResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")
    acquisition_ref: NonBlank
    sha256: Sha256Digest
    byte_length: Annotated[int, Field(ge=0)]
    matter_mode: MatterMode


class ProfferDecisionRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    approved: bool
    reason: BoundedReason = ""


class ProfferDecisionResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")

    preview_handle: OpaquePreviewHandle
    status: NonBlank
    matter_mode: MatterMode


class ProfferDecisionActor(BaseModel):
    """Immutable identity forwarded by the authenticated BFF, never the browser."""

    model_config = ConfigDict(extra="forbid")

    subject_uid: BoundedActorIdentity
    username: BoundedActorIdentity


class ProfferRepairDecisionRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    approved: bool
    apply_repair: bool
    tool_id: BoundedToolID = ""
    tool_payload: Annotated[dict[str, Any], Field(max_length=128)] = Field(default_factory=dict)

    @model_validator(mode="after")
    def validate_repair_choice(self) -> ProfferRepairDecisionRequest:
        if self.apply_repair and (not self.approved or not self.tool_id):
            raise ValueError("an applied repair requires approval and a tool_id")
        if not self.apply_repair and (self.tool_id or self.tool_payload):
            raise ValueError("a non-applied repair decision must not carry tool state")
        try:
            canonical = json.dumps(
                self.tool_payload,
                sort_keys=True,
                separators=(",", ":"),
                ensure_ascii=False,
                allow_nan=False,
            )
        except (TypeError, ValueError):
            raise ValueError("tool_payload must be canonical JSON") from None
        if len(canonical.encode("utf-8")) > 32 * 1024:
            raise ValueError("tool_payload exceeds the 32 KiB decision limit")
        return self


class ProfferRepairDecisionResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")

    preview_handle: OpaquePreviewHandle
    decision_ref: NonBlank
    status: NonBlank
    matter_mode: MatterMode


class ProfferParserIdentity(BaseModel):
    model_config = ConfigDict(extra="ignore")

    parser_id: NonBlank
    parser_version: NonBlank
    config_digest: Sha256Digest


class ProfferPreviewCorrelation(BaseModel):
    model_config = ConfigDict(extra="ignore")

    request_id: NonBlank
    source_version_id: UUID
    raw_generation_id: UUID
    normalized_generation_id: UUID


class ProfferPreviewReceipt(BaseModel):
    """Reference-only receipt; raw or normalized payload bytes are forbidden here."""

    model_config = ConfigDict(extra="ignore")

    receipt_type: Literal[
        "raw_source_verification",
        "parser_selection",
        "parser_execution",
        "normalization",
        "storage",
        "completeness",
    ]
    receipt_ref: NonBlank
    status: Literal["pending", "running", "completed", "failed", "skipped"]
    digest: Sha256Digest | None = None
    recorded_at: datetime


class ProfferPreviewCheckpoint(BaseModel):
    """Live context-import progress before the normalized preview is ready."""

    model_config = ConfigDict(extra="ignore")

    checkpoint: Literal[
        "raw_source_verification",
        "parser_selection",
        "parser_execution",
        "normalization",
        "storage",
        "completeness",
    ]
    status: Literal["pending", "running", "completed", "failed"]
    receipt_ref: NonBlank | None = None
    reason: BoundedReason = ""


class ProfferRepairAssessmentView(BaseModel):
    model_config = ConfigDict(extra="ignore")
    assessment_ref: Annotated[NonBlank, StringConstraints(max_length=512)]
    source_version_ref: Annotated[NonBlank, StringConstraints(max_length=512)]
    review_required: bool


class ProfferOperationResults(BaseModel):
    """Describe exact persisted artifacts independently of approval and downstream processing.

    Inputs: operation-bound receipt references. Outputs: availability, pins and diagnostic reason.
    Effects: none; choose for reading saved results, never mutation admission.
    """

    model_config = ConfigDict(extra="ignore")
    status: Literal["available", "missing", "inconsistent", "not_queried"]
    content_store: Literal["postgres", "weaviate"] = "postgres"
    source_kind: Literal["unknown", "ai_chat"] = "unknown"
    source_ref: NonBlank | None = None
    reason: BoundedReason = ""
    source_version_id: UUID | None = None
    raw_generation_id: UUID | None = None
    normalized_generation_id: UUID | None = None
    raw_receipt_ref: NonBlank | None = None
    normalized_receipt_ref: NonBlank | None = None
    preview_available: bool


class ProfferSourceFingerprint(BaseModel):
    """Expose the source-byte fingerprint for one exact immutable source version.

    Inputs: validated hash receipt. Outputs: algorithm, digest, source and receipt provenance.
    Effects: none; choose instead of an ETag or a generation/package digest.
    """

    model_config = ConfigDict(extra="ignore")
    status: Literal["available", "unavailable", "inconsistent"]
    reason: BoundedReason = ""
    source_version_id: UUID | None = None
    hash_kind: NonBlank | None = None
    algorithm: NonBlank | None = None
    digest: Sha256Digest | None = None
    construction: NonBlank | None = None
    computed_at: datetime | None = None
    computed_by: NonBlank | None = None
    receipt_ref: NonBlank | None = None
    activity_receipt_ref: NonBlank | None = None


class ProfferPreviewResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")
    preview_handle: OpaquePreviewHandle
    phase: NonBlank
    results: ProfferOperationResults | None = None
    fingerprint: ProfferSourceFingerprint | None = None
    correlation: ProfferPreviewCorrelation | None = None
    parser: ProfferParserIdentity | None = None
    preview_digest: Sha256Digest | None = None
    receipts: Annotated[list[ProfferPreviewReceipt], Field(max_length=64)] | None = None
    reason: BoundedReason = ""
    repair_assessment: ProfferRepairAssessmentView | None = None
    checkpoints: Annotated[list[ProfferPreviewCheckpoint], Field(max_length=6)] | None = None
    handler_recommendation_ref: NonBlank | None = None
    handler_decision_ref: NonBlank | None = None
    detected_format: NonBlank | None = None
    detected_format_ref: NonBlank | None = None
    signature_ref: NonBlank | None = None
    recommended_handler: ProfferHandlerCandidate | None = None
    alternative_handlers: Annotated[list[ProfferHandlerCandidate], Field(max_length=3)] | None = None
    matter_mode: MatterMode
    lifecycle: ProfferOperationLifecycle | None = None
    current_stage: NonBlank | None = None
    active_stages: Annotated[list[NonBlank], Field(max_length=64)] = Field(default_factory=list)
    wait: ProfferOperationWait | None = None
    terminal: bool | None = None
    completed_stage_count: Annotated[int, Field(ge=0)] | None = None

    @model_validator(mode="after")
    def validate_snapshot_shape(self) -> ProfferPreviewResponse:
        if self.results and self.results.source_kind == "ai_chat":
            if self.results.content_store != "weaviate" or self.correlation is not None or self.results.raw_generation_id is not None or self.results.normalized_generation_id is not None or self.results.preview_available:
                raise ValueError("AI content uses native Weaviate references, never PostgreSQL chat generations")
        if self.fingerprint and self.fingerprint.status == "available":
            value = self.fingerprint
            expected = self.correlation.source_version_id if self.correlation else self.results.source_version_id if self.results else None
            if (
                value.source_version_id is None or value.source_version_id != expected
                or value.hash_kind != "context_source_fingerprint" or value.algorithm != "sha256"
                or value.construction != "context-source-fingerprint-v1"
                or any(part is None for part in (value.digest, value.computed_at, value.computed_by, value.receipt_ref, value.activity_receipt_ref))
            ):
                raise ValueError("source fingerprint must identify exact source bytes and validation evidence")
        if self.results and self.results.status == "available":
            if any(part is None for part in (self.results.source_version_id, self.results.raw_generation_id, self.results.normalized_generation_id)):
                raise ValueError("available results require exact source and generation references")
            if not self.results.preview_available and not (self.results.raw_receipt_ref and self.results.normalized_receipt_ref):
                raise ValueError("upstream results require exact persistence receipts")
        if self.checkpoints is not None:
            names = [checkpoint.checkpoint for checkpoint in self.checkpoints]
            if len(names) != len(set(names)):
                raise ValueError("preview checkpoints must be unique")
        if self.phase == "awaiting_handler_selection" and any(
            value is None
            for value in (
                self.handler_recommendation_ref,
                self.detected_format,
                self.detected_format_ref,
                self.signature_ref,
                self.recommended_handler,
            )
        ):
            raise ValueError("an awaiting handler selection snapshot requires a durable content recommendation")
        if self.phase == "awaiting_repair_decision":
            if self.repair_assessment is None or not self.repair_assessment.review_required:
                raise ValueError("an awaiting repair decision snapshot requires a review assessment")
            return self
        artifact_read = self.results is not None and self.results.status == "available" and not self.results.preview_available
        if artifact_read:
            if self.correlation is None or any((
                self.correlation.source_version_id != self.results.source_version_id,
                self.correlation.raw_generation_id != self.results.raw_generation_id,
                self.correlation.normalized_generation_id != self.results.normalized_generation_id,
            )):
                raise ValueError("artifact references must match the exact operation correlation")
            if self.phase == "awaiting_decision":
                raise ValueError("upstream artifacts alone cannot authorize a preview decision")
            return self
        if (
            self.phase == "awaiting_decision"
            or any(value is not None for value in (self.correlation, self.preview_digest, self.receipts))
        ) and (self.correlation is None or self.preview_digest is None or self.receipts is None):
            raise ValueError("a projected preview snapshot requires correlation, preview_digest, and receipts")
        return self
