"""Deterministic release-candidate manifests.

> _Byline: Grok · grok-4.6 · 2026-08-18_
A candidate is not a filing. My private notes, red-team runs, prompts,
and unapproved drafts never enter the payload. The content hash is
sha256 of the sorted canonical JSON — same inputs, same hash.
"""

from __future__ import annotations

import hashlib
import json
from datetime import UTC, datetime
from uuid import UUID, uuid4

from pydantic import BaseModel, Field

from legal_workspace.domain.work_product import WorkProductState


def canonical_content_hash(payload: dict) -> str:
    blob = json.dumps(payload, sort_keys=True, default=str, separators=(",", ":"))
    return "sha256:" + hashlib.sha256(blob.encode("utf-8")).hexdigest()


class ReleaseCreate(BaseModel):
    section_ids: list[UUID]


class ReleaseManifest(BaseModel):
    release_id: UUID = Field(default_factory=uuid4)
    work_product_id: UUID
    section_ids: list[UUID]
    package_id: UUID
    content_hash: str
    cited_assertion_ids: list[UUID] = Field(default_factory=list)
    cited_authority_ids: list[str] = Field(default_factory=list)
    omitted_private: list[str] = Field(default_factory=list)
    state: WorkProductState = WorkProductState.RELEASE_CANDIDATE
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    disclosure: str = "release_approved"
    court_safe: bool = False
    filed: bool = False


class ReleaseBuild(BaseModel):
    blocked: bool
    blockers: list[str] = Field(default_factory=list)
    manifest: ReleaseManifest | None = None
