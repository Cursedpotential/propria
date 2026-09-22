"""Wire types for the batched "is this source decoded?" check.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field


class DecodedExistsRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    source_refs: list[str] = Field(min_length=1, max_length=200)


class DecodedExistsItem(BaseModel):
    model_config = ConfigDict(extra="forbid")
    source_ref: str
    decoded: bool
    reason: str = ""  # empty when the answer is a plain yes or no


class DecodedExistsResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")
    items: list[DecodedExistsItem]
