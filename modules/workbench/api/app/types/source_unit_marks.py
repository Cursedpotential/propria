"""Wire types for hand-marked source units.

Byline: Claude Code · Opus 5 · 2026-09-22.

The catalog's `raw_duck.atomic_units` is read-only to the Workbench. A unit the
owner marks by hand is therefore Workbench-owned state, kept apart from the
catalog's own detections and always labelled as hand-marked where it is shown.
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

UnitKind = Literal[
    "takeout",
    "takeout_zip",
    "facebook",
    "snapchat",
    "cube_acr",
    "git_repo",
    "obsidian_vault",
    "other",
]


class SourceUnitMark(BaseModel):
    model_config = ConfigDict(extra="forbid")
    unit_root: str = Field(min_length=1, max_length=2048)
    unit_type: UnitKind
    label: str = ""
    marked_at: str
    marked_by: str
    origin: Literal["hand_marked"] = "hand_marked"


class SourceUnitMarkRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    unit_root: str = Field(min_length=1, max_length=2048)
    unit_type: UnitKind
    label: str = Field(default="", max_length=200)
    # A Takeout stays a supervised proposal: it is recorded only on an
    # explicit confirm (bulk-intake requirement 7).
    confirm: bool = False


class SourceUnitMarkList(BaseModel):
    model_config = ConfigDict(extra="forbid")
    items: list[SourceUnitMark]
    storage: str  # where these marks live, said plainly in the surface
    catalog_units_are_read_only: Literal[True] = True


class TakeoutPartProposal(BaseModel):
    model_config = ConfigDict(extra="forbid")
    stamp: str
    job: str
    parts_present: list[int]
    parts_missing: list[int]
    highest_part: int


class SourceUnitProposalRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    unit_root: str = Field(min_length=1, max_length=2048)
    names: list[str] = Field(default_factory=list, max_length=2000)


class SourceUnitProposal(BaseModel):
    model_config = ConfigDict(extra="forbid")
    unit_root: str
    looks_like: UnitKind | None
    basis: Literal["observed_listing"]
    observed_count: int
    takeout_sets: list[TakeoutPartProposal]
    requires_confirmation: bool
