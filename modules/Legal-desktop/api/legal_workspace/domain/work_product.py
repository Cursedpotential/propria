"""Work-product version states. Released rows are never mutated.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class WorkProductState(str, Enum):
    PRIVATE_DRAFT = "private_draft"
    REVIEW_REQUIRED = "review_required"
    CITATION_VERIFIED = "citation_verified"
    RELEASE_CANDIDATE = "release_candidate"
    OWNER_APPROVED = "owner_approved"
    RELEASED = "released"
    SUPERSEDED = "superseded"
    WITHDRAWN = "withdrawn"
    STALE = "stale"
    REVALIDATION_REQUIRED = "revalidation_required"


IMMUTABLE_STATES = frozenset(
    {
        WorkProductState.RELEASED,
        WorkProductState.SUPERSEDED,
        WorkProductState.WITHDRAWN,
    }
)


class WorkProductVersion(BaseModel):
    work_product_id: UUID
    version: int = 1
    state: WorkProductState = WorkProductState.PRIVATE_DRAFT
    source_package_id: UUID | None = None
    content_hash: str = ""
    cited_assertion_ids: list[UUID] = Field(default_factory=list)
    cited_authority_ids: list[str] = Field(default_factory=list)
    superseded_by: UUID | None = None


def edit_creates_new_version(current: WorkProductVersion) -> WorkProductVersion:
    """Editing a released version must fork, never mutate."""
    if current.state in IMMUTABLE_STATES:
        return current.model_copy(
            update={
                "work_product_id": uuid4(),
                "version": current.version + 1,
                "state": WorkProductState.PRIVATE_DRAFT,
                "superseded_by": None,
            }
        )
    return current.model_copy(update={"version": current.version + 1})
