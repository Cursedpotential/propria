"""Define browser inputs for bounded, read-only combined search.

Inputs choose query, mode and approved source families; outputs validate them.
No I/O occurs. Use these inputs instead of accepting upstream URLs or authority.
Byline: Codex · 2026-10-06.
"""
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


class SearchRequest(BaseModel):
    """Validate a browser search without accepting a caller-selected case or provider.

    Inputs are a request ID, text, mode, source families and limit; output is a
    typed request. No side effects; this is the Workbench's public search input.
    """
    model_config = ConfigDict(extra="forbid")
    request_id: str = Field(min_length=1, max_length=128, pattern=r"^[A-Za-z0-9_-]+$")
    query: str = Field(min_length=1, max_length=2000, pattern=r"\S")
    mode: Literal["keyword", "hybrid", "vector"] = "hybrid"
    legs: list[Literal["intake", "proffer", "casebible"]] = Field(
        default_factory=lambda: ["intake", "proffer"], min_length=1, max_length=3
    )
    # Explicit, limited Super Index pre-ranking filters. Neither is a case or authority selector.
    source_id: str | None = Field(default=None, min_length=1, max_length=255, pattern=r"\S")
    path_prefix: str | None = Field(default=None, min_length=1, max_length=1024, pattern=r"\S")
    limit: int = Field(default=20, ge=1, le=50, strict=True)

    @field_validator("legs")
    @classmethod
    def distinct_legs(cls, values: list[str]) -> list[str]:
        """Reject duplicate source families so one backend cannot gain extra ranking weight."""
        if len(set(values)) != len(values):
            raise ValueError("source families must be distinct")
        return values

    @model_validator(mode="after")
    def pre_ranked_superindex_filters(self) -> "SearchRequest":
        """Only Super Index supports these scopes; do not silently broaden other legs."""
        if self.path_prefix is None and self.source_id is None:
            return self
        if self.legs != ["casebible"]:
            raise ValueError("source and path filters require Super Index as the only search source")
        if self.path_prefix is not None:
            path = self.path_prefix
            if path.startswith("/") or "://" in path or "\\" in path or "*" in path or "?" in path:
                raise ValueError("path prefix must be a literal catalog-relative key")
            if any(part in {".", ".."} for part in path.split("/")):
                raise ValueError("path prefix cannot contain traversal segments")
        return self

class GraphResolveRequest(BaseModel):
    """Carry exact Intake source identity for a read-only relationship lookup.

    Inputs are source/document and optional version IDs; output is validated
    identity. No I/O or authority changes occur; use before choosing a snapshot.
    """
    model_config = ConfigDict(extra="forbid")
    source_id: str = Field(min_length=1, max_length=255, pattern=r"\S")
    document_id: str = Field(min_length=1, max_length=255, pattern=r"\S")
    version_id: str | None = Field(default=None, min_length=1, max_length=255, pattern=r"\S")
