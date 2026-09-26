"""Import and validate LegalSourcePackage objects.

> _Byline: Grok · grok-4.6 · 2026-08-18_
> _Producer-evidence hold: Codex · GPT-6 · 2026-09-23_
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from uuid import UUID

from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    ReviewState,
)

_SHA256 = re.compile(r"sha256:[0-9a-fA-F]{64}\Z")
_SUPPORTED_SCHEMA_VERSION = "1.0"


def validate_consumer_package(package: LegalSourcePackage, matter_id: UUID) -> None:
    """Reject malformed or out-of-scope package references before persistence.

    This checks the consumer envelope. It does not verify producer signatures,
    recompute the producer manifest, or establish source-byte authenticity.
    """
    if package.matter_id != matter_id:
        raise ValueError("package matter_id does not match this workspace")
    if package.schema_version != _SUPPORTED_SCHEMA_VERSION:
        raise ValueError("unsupported package schema_version")
    if package.package_id.int == 0:
        raise ValueError("package_id must be non-nil")
    if not _SHA256.fullmatch(package.manifest_hash):
        raise ValueError("manifest_hash must be sha256 followed by 64 hex characters")
    for item in package.items:
        if item.assertion_version < 1:
            raise ValueError("assertion_version must be positive")
        if not _SHA256.fullmatch(item.content_hash):
            raise ValueError("item content_hash must be sha256 followed by 64 hex characters")


@dataclass(frozen=True)
class ImportResult:
    accepted: LegalSourcePackage
    omitted_item_ids: tuple[str, ...]
    blocked: bool
    reason: str | None = None


def import_legal_source_package(package: LegalSourcePackage) -> ImportResult:
    """Inspect eligibility without claiming an unverified producer package.

    D08 has not supplied an issuer, canonical manifest digest algorithm, or
    verifiable status receipt. A syntactically valid digest is caller input,
    so even APPROVED rows cannot be installed as usable legal evidence.
    """
    omitted = [
        str(item.item_id)
        for item in package.items
        if item.review_state is not ReviewState.APPROVED
    ]
    accepted_items = [
        item for item in package.items if item.review_state is ReviewState.APPROVED
    ]
    inspected = package.model_copy(update={"items": []})
    if not accepted_items:
        return ImportResult(
            accepted=inspected,
            omitted_item_ids=tuple(omitted),
            blocked=True,
            reason="no approved items in package",
        )
    return ImportResult(
        accepted=inspected,
        omitted_item_ids=tuple(omitted),
        blocked=True,
        reason="D08 producer evidence unavailable: approved package manifest and status cannot be independently verified",
    )
