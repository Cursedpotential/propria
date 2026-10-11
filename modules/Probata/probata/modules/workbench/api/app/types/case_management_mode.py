"""Policy-echoing single-case views with an engine-admitted court coordinate.

Byline amendment: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from uuid import UUID

from pydantic import BaseModel

from app.types.case_management import Matter, MatterDetail
from app.types.matter_mode import MatterMode


class ModeBoundMatter(Matter):
    matter_mode: MatterMode


class ModeBoundMatterList(BaseModel):
    data: list[ModeBoundMatter]
    total: int
    limit: int
    offset: int
    matter_mode: MatterMode


class ModeBoundMatterDetail(MatterDetail):
    matter_mode: MatterMode
    admitted_court_case_id: UUID
