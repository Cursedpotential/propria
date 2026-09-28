"""Cancel one Proffer run: the request and the engine's acknowledgement.

Byline: Claude Code · Opus 5.5 · 2026-09-28 (D05-C06: public cancel through Temporal).
"""

from __future__ import annotations

from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, StringConstraints

from app.types.matter_mode import MatterMode
from app.types.proffer import OpaquePreviewHandle

CancelReason = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=4000)]


class ProfferCancelRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    reason: CancelReason


class ProfferCancelResponse(BaseModel):
    model_config = ConfigDict(extra="ignore")

    preview_handle: OpaquePreviewHandle
    status: Literal["cancel_requested"]
    matter_mode: MatterMode
