"""Typed lifecycle projections for durable Proffer operations.

Byline amendment: Codex · GPT-6.1-Sol · 2026-10-05 — empty old flags remain unknown.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated, Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, field_validator

from app.types.matter_mode import CanonicalMatterMode
from app.types.proffer import (
    BoundedReason,
    NonBlank,
    OpaqueCursor,
    OpaquePreviewHandle,
    ProfferOperationLifecycle,
    ProfferOperationWait,
)


class ProfferOperationSummary(BaseModel):
    """Reference-only lifecycle projection for one durable Proffer execution."""

    model_config = ConfigDict(extra="ignore")

    preview_handle: OpaquePreviewHandle
    request_id: NonBlank
    source_ref: NonBlank
    service: Literal["proffer"]
    created_at: datetime
    lifecycle: ProfferOperationLifecycle
    current_stage: NonBlank | None = None
    active_stages: Annotated[list[NonBlank], Field(max_length=64)] = Field(default_factory=list)
    wait: ProfferOperationWait | None = None
    terminal: bool
    reason: BoundedReason = ""
    source_version_ref: NonBlank | None = None
    matter_id: UUID | None = None
    operating_mode: CanonicalMatterMode | None = None  # absent old receipts remain unbound
    completed_stage_count: Annotated[int, Field(ge=0)]

    @field_validator("operating_mode", mode="before")
    @classmethod
    def empty_historical_mode(cls, value):
        """Normalize only an explicit empty old flag to unknown, never Live.

        Input: raw receipt flag. Output: None for empty, otherwise unchanged.
        Effects: none. Pick for catalog readability, never mutation admission.
        """
        return None if value == "" else value


class ProfferOperationStage(BaseModel):
    model_config = ConfigDict(extra="ignore")

    stage: NonBlank
    status: NonBlank
    ref: NonBlank | None = None
    receipt_ref: NonBlank | None = None
    reason: BoundedReason = ""
    attempt: Annotated[int, Field(ge=0)] | None = None
    started_at: datetime | None = None
    completed_at: datetime | None = None


class ProfferOperationDetail(ProfferOperationSummary):
    stages: Annotated[list[ProfferOperationStage], Field(max_length=128)]


class ProfferOperationListResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")

    items: Annotated[list[ProfferOperationSummary], Field(max_length=100)]
    next_cursor: OpaqueCursor | None = None
