"""Define browser inputs for bounded, read-only combined search.

Inputs choose query, mode and approved source families; outputs validate them.
No I/O occurs. Use these inputs instead of accepting upstream URLs or authority.
Byline: Codex · 2026-10-06.
"""
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator


class SearchRequest(BaseModel):
    """Validate a browser search without accepting a caller-selected case or provider.

    Inputs are a request ID, text, mode, source families and limit; output is a
    typed request. No side effects; this is the Workbench's public search input.
    """
    model_config = ConfigDict(extra="forbid")
    request_id: str = Field(min_length=1, max_length=128, pattern=r"^[A-Za-z0-9_-]+$")
    query: str = Field(min_length=1, max_length=2000, pattern=r"\S")
    mode: Literal["keyword", "hybrid", "vector"] = "hybrid"
    legs: list[Literal["intake", "proffer"]] = Field(default_factory=lambda: ["intake", "proffer"], min_length=1, max_length=2)
    limit: int = Field(default=20, ge=1, le=50, strict=True)

    @field_validator("legs")
    @classmethod
    def distinct_legs(cls, values: list[str]) -> list[str]:
        """Reject duplicate source families so one backend cannot gain extra ranking weight."""
        if len(set(values)) != len(values):
            raise ValueError("source families must be distinct")
        return values
