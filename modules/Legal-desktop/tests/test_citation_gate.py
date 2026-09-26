"""Territory tests for the release citation gate.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

from legal_workspace.contracts.citations import AuthorityCitation, AuthorityLevel, EvidenceCitation
from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    LegalSourcePackageItem,
    ReviewState,
)
from legal_workspace.services.citation_gate import (
    validate_authority_citations,
    validate_factual_citations,
)


def test_factual_sentence_without_approved_assertion_is_unsupported() -> None:
    assertion_id = uuid4()
    package = LegalSourcePackage(
        package_id=uuid4(),
        manifest_hash="sha256:pkg",
        matter_id=uuid4(),
        created_at=datetime.now(UTC),
        items=[
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=assertion_id,
                assertion_version=1,
                span_locator="span:1",
                custody_locator="h1:1",
                content_hash="sha256:a",
                review_state=ReviewState.APPROVED,
            )
        ],
    )
    unsupported = EvidenceCitation(
        statement="The child was withheld on 12 March.",
        package_id=package.package_id,
        assertion_id=uuid4(),
        assertion_version=1,
        span_locator="span:missing",
    )
    supported = EvidenceCitation(
        statement="Parenting time occurred as ordered on 1 April.",
        package_id=package.package_id,
        assertion_id=assertion_id,
        assertion_version=1,
        span_locator="span:1",
    )

    bad = validate_factual_citations([unsupported], package)
    good = validate_factual_citations([supported], package)

    assert bad.ok is False
    assert any(item.startswith("UNSUPPORTED") for item in bad.blockers)
    assert good.ok is True


def test_authority_without_snapshot_blocks_and_citator_flag_is_rejected() -> None:
    missing_snapshot = AuthorityCitation(
        proposition="Best interests include the love and affection factor.",
        identifier="MCL 722.23(a)",
        authority_level=AuthorityLevel.STATUTE,
        snapshot_hash=None,
    )
    claimed_citator = AuthorityCitation(
        proposition="This case remains good law.",
        identifier="Pierron v Pierron",
        authority_level=AuthorityLevel.PUBLISHED_OPINION,
        snapshot_hash="sha256:snap",
        is_citator_verified=True,
    )
    pinned = AuthorityCitation(
        proposition="The court shall consider the factors.",
        identifier="MCL 722.23",
        pinpoint="(a)",
        authority_level=AuthorityLevel.STATUTE,
        snapshot_hash="sha256:mcl72223",
    )

    assert validate_authority_citations([missing_snapshot]).ok is False
    assert validate_authority_citations([claimed_citator]).ok is False
    assert validate_authority_citations([pinned]).ok is True
