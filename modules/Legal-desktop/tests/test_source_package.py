"""Territory tests for LegalSourcePackage import.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

import pytest
from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    LegalSourcePackageItem,
    ReviewState,
)
from legal_workspace.services.source_package import import_legal_source_package
from legal_workspace.services.workspace import Workspace


def _item(state: ReviewState) -> LegalSourcePackageItem:
    return LegalSourcePackageItem(
        item_id=uuid4(),
        assertion_id=uuid4(),
        assertion_version=1,
        span_locator="src:msg:12:4-12:80",
        custody_locator="h1:abc",
        content_hash="sha256:deadbeef",
        review_state=state,
    )


def test_import_reports_omissions_but_blocks_unverified_approved_items() -> None:
    approved = _item(ReviewState.APPROVED)
    candidate = _item(ReviewState.CANDIDATE)
    revoked = _item(ReviewState.REVOKED)
    package = LegalSourcePackage(
        package_id=uuid4(),
        manifest_hash="sha256:pkg",
        matter_id=uuid4(),
        items=[approved, candidate, revoked],
        created_at=datetime.now(UTC),
    )

    result = import_legal_source_package(package)

    assert result.blocked is True
    assert result.accepted.items == []
    assert "D08 producer evidence unavailable" in (result.reason or "")
    assert set(result.omitted_item_ids) == {str(candidate.item_id), str(revoked.item_id)}


def test_import_blocks_when_nothing_is_approved() -> None:
    package = LegalSourcePackage(
        package_id=uuid4(),
        manifest_hash="sha256:pkg",
        matter_id=uuid4(),
        items=[_item(ReviewState.CANDIDATE)],
        created_at=datetime.now(UTC),
    )

    result = import_legal_source_package(package)

    assert result.blocked is True
    assert result.accepted.items == []
    assert result.reason == "no approved items in package"


def _valid_package(matter_id: UUID) -> LegalSourcePackage:
    return LegalSourcePackage(
        package_id=uuid4(),
        schema_version="1.0",
        manifest_hash="sha256:" + "a" * 64,
        matter_id=matter_id,
        items=[
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=uuid4(),
                assertion_version=1,
                span_locator="source:message:1",
                custody_locator="custody:1",
                content_hash="sha256:" + "b" * 64,
                review_state=ReviewState.APPROVED,
            )
        ],
        created_at=datetime.now(UTC),
    )


@pytest.mark.parametrize(
    ("field", "bad_value", "message"),
    [
        ("matter_id", uuid4(), "matter_id"),
        ("schema_version", "2.0", "schema_version"),
        ("package_id", UUID(int=0), "package_id"),
        ("manifest_hash", "sha256:not-a-digest", "manifest_hash"),
    ],
)
def test_consumer_rejects_invalid_package_without_storage_change(
    tmp_path, field, bad_value, message
) -> None:
    workspace = Workspace(tmp_path)
    package = _valid_package(workspace.load().matter.matter_id)
    package = package.model_copy(update={field: bad_value})
    before = {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()}

    with pytest.raises(ValueError, match=message):
        workspace.import_package(package)

    assert {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()} == before


@pytest.mark.parametrize(
    ("field", "bad_value", "message"),
    [
        ("assertion_version", 0, "assertion_version"),
        ("content_hash", "sha256:bad", "content_hash"),
    ],
)
def test_consumer_rejects_invalid_item_without_storage_change(
    tmp_path, field, bad_value, message
) -> None:
    workspace = Workspace(tmp_path)
    package = _valid_package(workspace.load().matter.matter_id)
    package.items[0] = package.items[0].model_copy(update={field: bad_value})
    before = {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()}

    with pytest.raises(ValueError, match=message):
        workspace.import_package(package)

    assert {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()} == before


def test_valid_looking_manifest_digest_cannot_adopt_changed_approved_payload(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    package = _valid_package(workspace.load().matter.matter_id)
    before = {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()}
    original = workspace.import_package(package)
    changed = package.model_copy(deep=True)
    changed.items[0].span_locator = "source:message:tampered"
    changed.manifest_hash = "sha256:" + "c" * 64
    tampered = workspace.import_package(changed)
    assert original.blocked is True
    assert tampered.blocked is True
    assert original.accepted.items == tampered.accepted.items == []
    assert "D08 producer evidence unavailable" in (tampered.reason or "")
    assert {path.name: path.read_bytes() for path in tmp_path.iterdir() if path.is_file()} == before
