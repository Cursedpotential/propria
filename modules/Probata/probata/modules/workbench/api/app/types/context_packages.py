"""Define lossless package review envelopes over existing native source stores.

Inputs: source/version references, persisted placement and bounded reader pages.
Outputs: dynamic sections and attributed review requests. Effects: none; use for
Read's category-to-package navigation without normalizing native source bodies.
"""
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field


class PackageReview(BaseModel):
    """Describe one explicit revision-bound review action.

    Inputs: displayed item/revision, edited fields, source locators and reason.
    Output: existing engine writer input. Effects: none until submitted; source
    content and search never require this action before becoming readable.
    """
    model_config = ConfigDict(extra="forbid")
    operation: Literal["metadata", "edit", "accept", "reject", "commit"]
    item_id: str = Field(min_length=1, max_length=512)
    revision: str = Field(default="", max_length=128)
    fields: dict[str, Any] = Field(default_factory=dict)
    source_refs: list[dict[str, Any]] = Field(default_factory=list, max_length=64)
    reason: str = Field(min_length=1, max_length=4000)


class PackageSection(BaseModel):
    """Report one actual reader's availability and full count without implying extraction.

    Inputs: reader status/count. Output: section descriptor. Effects: none;
    choose an explicit reason when a required reader or extraction is absent.
    """
    id: str
    label: str
    count: int | None
    available: bool
    reason: str | None = None
