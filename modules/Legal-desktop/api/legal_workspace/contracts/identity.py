"""Matter / CourtCase local read-only projections.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from uuid import UUID

from pydantic import BaseModel, Field


class MatterRef(BaseModel):
    """Local projection of an Evidence Platform Matter identity."""

    matter_id: UUID
    display_name: str
    is_friendly_primary: bool = True
    county: str | None = "Genesee"
    state: str | None = "Michigan"
    posture: str | None = "pro se family / custody-parenting time"
    notes: str | None = Field(
        default="Docket, assigned judge, and party captions are omitted until a complete primary source is on disk."
    )


class CourtCaseRef(BaseModel):
    """Local projection of a specific proceeding inside a Matter."""

    court_case_id: UUID
    matter_id: UUID
    display_name: str
    docket_number: str | None = None
    court: str | None = Field(default="7th Judicial Circuit, Genesee County, Michigan")
    division: str | None = "Family Division"
    courthouse: str | None = "900 S. Saginaw St., Flint, Michigan"
    court_url: str | None = "https://7thcircuitcourt.com/"
    judge: str | None = None
    foc: str | None = None
    referee: str | None = None
    is_primary: bool = True
