"""Wire types for folder batch imports (the engine's /reference-import batch routes).

Byline: Claude Code · Opus 5 · 2026-09-22.

These mirror `modules/engine/proffer/batch.go` (BatchInput, BatchStatus,
BatchCounts, BatchItem) so the Workbench passes references through without
reshaping them. The BFF adds only its own `matter_mode` echo, exactly as the
single-start path does.
"""

from __future__ import annotations

from typing import Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field

from app.types.matter_mode import MatterMode

BatchItemStatus = Literal["queued", "running", "waiting_on_gate", "done", "failed", "skipped"]


class ProfferBatchStartRequest(BaseModel):
    """One folder, one batch. The folder locator is validated by the engine."""

    model_config = ConfigDict(extra="forbid")

    batch_id: str = Field(min_length=32, max_length=128, pattern=r"^[A-Za-z0-9_-]{32,128}$")
    matter_id: UUID
    court_case_id: UUID
    folder_ref: str = Field(min_length=8, max_length=2048)
    declared_format: str = Field(min_length=1, max_length=128)
    parser_options_ref: str = Field(min_length=1, max_length=256)
    source_context_ref: str | None = None
    max_in_flight: int = Field(default=0, ge=0, le=16)
    # First-party context import (D04), passed to every item's run.
    # Byline: Claude Code · Opus 5.5 · 2026-10-01
    owner_person_id: UUID | None = None
    perspective_person_id: UUID | None = None
    matter_mode: MatterMode = "LIVE"


class ProfferBatchStartResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")
    batch_id: str
    matter_mode: MatterMode


class ProfferBatchCounts(BaseModel):
    model_config = ConfigDict(extra="forbid")
    total: int = 0
    queued: int = 0
    running: int = 0
    waiting_on_gate: int = 0
    done: int = 0
    failed: int = 0
    skipped: int = 0


class ProfferBatchItem(BaseModel):
    model_config = ConfigDict(extra="forbid")
    key: str
    source_ref: str = ""
    request_id: str = ""
    preview_handle: str = ""
    status: BatchItemStatus
    reason: str = ""


class ProfferBatchStatus(BaseModel):
    model_config = ConfigDict(extra="forbid")
    batch_id: str
    prefix: str
    terminal: bool
    listing_truncated: bool = False
    items_truncated: bool = False
    counts: ProfferBatchCounts
    items: list[ProfferBatchItem]
    matter_mode: MatterMode
