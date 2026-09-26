"""Contract tests for the held D08 LegalSourcePackage v1 consumer boundary.

Updated by: Codex (migration-passes/d09) | Date: 2026-09-23 | Rev: 2 | Platform: Codex / win32 | Changes: enforce receipt, rejection, replay, and keyword-only invariants | Context: independent review remediation
"""

from __future__ import annotations

from dataclasses import FrozenInstanceError
from datetime import UTC, datetime
from inspect import Parameter, signature
from uuid import UUID, uuid4

import pytest
from legal_workspace.integrations.legal_source_package_v1 import (
    SCHEMA_VERSION,
    SIGNATURE_ALGORITHM,
    HeldLegalSourcePackageV1Consumer,
    LegalSourcePackageV1Consumer,
    LegalSourcePackageV1Held,
    PackageAcceptanceRejectionCode,
    PackageAvailabilityReceipt,
)
from legal_workspace.services.workspace import Workspace


def _files(path) -> dict[str, bytes]:
    return {
        str(item.relative_to(path)): item.read_bytes() for item in path.rglob("*") if item.is_file()
    }


def _receipt(**updates) -> PackageAvailabilityReceipt:
    values = {
        "schema_version": SCHEMA_VERSION,
        "package_id": uuid4(),
        "issuer": "indicia-probata",
        "issuer_key_id": "legal-package-key-2026-09",
        "signature_algorithm": SIGNATURE_ALGORITHM,
        "matter_id": uuid4(),
        "package_digest": "sha256:" + "a" * 64,
        "signature_hex": "ab" * 64,
        "producer_revision": "sha256:" + "b" * 64,
        "accepted_at": datetime.now(UTC),
    }
    values.update(updates)
    return PackageAvailabilityReceipt(**values)


def test_held_consumer_fails_closed_and_exposes_only_atomic_acceptance() -> None:
    consumer = HeldLegalSourcePackageV1Consumer()

    assert isinstance(consumer, LegalSourcePackageV1Consumer)
    parameters = signature(consumer.accept_if_current).parameters
    assert tuple(parameters) == ("serialized_package", "expected_matter_id")
    assert all(parameter.kind is Parameter.KEYWORD_ONLY for parameter in parameters.values())
    assert not hasattr(consumer, "verify")
    assert not hasattr(consumer, "acknowledge")
    assert not hasattr(consumer, "persist")

    with pytest.raises(
        LegalSourcePackageV1Held,
        match="trusted issuer keys.*canonical producer snapshot.*atomic conditional inbox",
    ) as rejected:
        consumer.accept_if_current(serialized_package=b"{}", expected_matter_id=uuid4())

    assert rejected.value.code is PackageAcceptanceRejectionCode.INTEGRATION_HELD
    assert rejected.value.retryable is False

    with pytest.raises(TypeError, match="positional"):
        consumer.accept_if_current(b"{}", expected_matter_id=uuid4())  # type: ignore[misc]


def test_held_consumer_cannot_mutate_existing_workspace(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    matter_id = workspace.load().matter.matter_id
    before = _files(tmp_path)

    with pytest.raises(LegalSourcePackageV1Held):
        HeldLegalSourcePackageV1Consumer().accept_if_current(
            serialized_package=b'{"schema_version":"legal-source-package/v1"}',
            expected_matter_id=matter_id,
        )

    assert _files(tmp_path) == before
    assert workspace.load().package is None


def test_availability_receipt_is_immutable_and_revision_bound() -> None:
    receipt = _receipt()

    with pytest.raises(FrozenInstanceError):
        receipt.producer_revision = "sha256:" + "c" * 64  # type: ignore[misc]


def test_replay_identity_binds_signed_manifest_and_issuer_key() -> None:
    receipt = _receipt()

    assert receipt.replay_identity == (
        receipt.schema_version,
        receipt.matter_id,
        receipt.package_id,
        receipt.issuer,
        receipt.issuer_key_id,
        receipt.signature_algorithm,
        receipt.package_digest,
        receipt.signature_hex,
    )
    assert receipt.producer_revision not in receipt.replay_identity
    assert receipt.accepted_at not in receipt.replay_identity


@pytest.mark.parametrize(
    ("updates", "message"),
    [
        ({"schema_version": "1.0"}, "schema_version"),
        ({"package_id": UUID(int=0)}, "package_id"),
        ({"matter_id": UUID(int=0)}, "matter_id"),
        ({"issuer": ""}, "issuer"),
        ({"issuer_key_id": " key"}, "issuer_key_id"),
        ({"signature_algorithm": "none"}, "signature_algorithm"),
        ({"package_digest": "sha256:BAD"}, "package_digest"),
        ({"signature_hex": "ab"}, "signature_hex"),
        ({"signature_hex": "GG" * 64}, "signature_hex"),
        ({"producer_revision": "sha256:" + "B" * 64}, "producer_revision"),
        ({"accepted_at": datetime.fromisoformat("2026-09-23T12:00:00")}, "accepted_at"),
    ],
)
def test_availability_receipt_rejects_invalid_audit_identity(updates, message) -> None:
    with pytest.raises(ValueError, match=message):
        _receipt(**updates)
