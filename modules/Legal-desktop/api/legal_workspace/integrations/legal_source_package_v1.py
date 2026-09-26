"""Held consumer boundary for the D08 LegalSourcePackage v1 contract.

This module is intentionally not mounted by the API.  A production adapter must
verify the signed D08 manifest, read one canonical producer snapshot, and persist
the package plus its availability receipt only if that exact producer revision
is still current at commit time.

Updated by: Codex (migration-passes/d09) | Date: 2026-09-23 | Rev: 2 | Platform: Codex / win32 | Changes: bind receipts and rejection/replay semantics | Context: independent review found incomplete D08 audit identity
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from typing import Protocol, runtime_checkable
from uuid import UUID

SCHEMA_VERSION = "legal-source-package/v1"
SIGNATURE_ALGORITHM = "ed25519"
_DIGEST = re.compile(r"sha256:[0-9a-f]{64}\Z")
_ED25519_SIGNATURE = re.compile(r"[0-9a-f]{128}\Z")


class PackageAcceptanceRejectionCode(str, Enum):
    """Stable fail-closed outcomes for a future D08 consumer adapter."""

    INTEGRATION_HELD = "integration_held"
    MALFORMED_PACKAGE = "malformed_package"
    MATTER_MISMATCH = "matter_mismatch"
    UNTRUSTED_ISSUER = "untrusted_issuer"
    INVALID_SIGNATURE = "invalid_signature"
    SNAPSHOT_UNAVAILABLE = "snapshot_unavailable"
    PACKAGE_UNAVAILABLE = "package_unavailable"
    ITEM_UNAVAILABLE = "item_unavailable"
    REVISION_CHANGED = "revision_changed"
    IDENTITY_CONFLICT = "identity_conflict"
    DURABLE_WRITE_FAILED = "durable_write_failed"


class LegalSourcePackageV1Rejected(RuntimeError):
    """Typed rejection from the fail-closed package-consumer boundary."""

    def __init__(
        self,
        code: PackageAcceptanceRejectionCode,
        detail: str,
        *,
        retryable: bool = False,
    ) -> None:
        self.code = code
        self.retryable = retryable
        super().__init__(detail)


class LegalSourcePackageV1Held(LegalSourcePackageV1Rejected):
    """Raised while the trusted D08 consumer adapter is not activated."""


def _non_nil_uuid(value: UUID, name: str) -> None:
    if not isinstance(value, UUID) or value.int == 0:
        raise ValueError(f"{name} must be a non-nil UUID")


def _text(value: str, name: str) -> None:
    if (
        not isinstance(value, str)
        or not value
        or value.strip() != value
        or any(ord(character) < 32 for character in value)
    ):
        raise ValueError(f"{name} must be nonempty and contain no control characters")


def _digest(value: str, name: str) -> None:
    if not isinstance(value, str) or _DIGEST.fullmatch(value) is None:
        raise ValueError(f"{name} must be a lowercase sha256 digest")


def _aware_time(value: datetime, name: str) -> None:
    if not isinstance(value, datetime) or value.tzinfo is None or value.utcoffset() is None:
        raise ValueError(f"{name} must be timezone-aware")


@dataclass(frozen=True)
class PackageAvailabilityReceipt:
    """Durable audit identity for one atomic, exact-revision acceptance."""

    schema_version: str
    package_id: UUID
    issuer: str
    issuer_key_id: str
    signature_algorithm: str
    matter_id: UUID
    package_digest: str
    signature_hex: str
    producer_revision: str
    accepted_at: datetime

    def __post_init__(self) -> None:
        if self.schema_version != SCHEMA_VERSION:
            raise ValueError("unsupported package schema_version")
        _non_nil_uuid(self.package_id, "package_id")
        _non_nil_uuid(self.matter_id, "matter_id")
        _text(self.issuer, "issuer")
        _text(self.issuer_key_id, "issuer_key_id")
        if self.signature_algorithm != SIGNATURE_ALGORITHM:
            raise ValueError("unsupported package signature_algorithm")
        _digest(self.package_digest, "package_digest")
        if (
            not isinstance(self.signature_hex, str)
            or _ED25519_SIGNATURE.fullmatch(self.signature_hex) is None
        ):
            raise ValueError("signature_hex must encode one lowercase 64-byte Ed25519 signature")
        _digest(self.producer_revision, "producer_revision")
        _aware_time(self.accepted_at, "accepted_at")

    @property
    def replay_identity(self) -> tuple[str, UUID, UUID, str, str, str, str, str]:
        """Fields that must match an existing receipt before replay handling."""
        return (
            self.schema_version,
            self.matter_id,
            self.package_id,
            self.issuer,
            self.issuer_key_id,
            self.signature_algorithm,
            self.package_digest,
            self.signature_hex,
        )


@runtime_checkable
class LegalSourcePackageV1Consumer(Protocol):
    """Port for trusted and atomic D08 package acceptance.

    Implementations must fail without any consumer write unless all of these
    occur as one acceptance operation:

    * parse the exact ``legal-source-package/v1`` schema;
    * resolve ``issuer_key_id`` only through a configured trust store;
    * verify the Ed25519 signature and canonical manifest/package digest;
    * read one producer snapshot covering the package and every item;
    * require package and item status ``available`` at that snapshot revision;
    * conditionally persist the package and availability receipt only while the
      same producer revision remains current.

    Replay is idempotent only when every field in the original receipt's
    ``replay_identity`` matches. The adapter must reverify the signature and one
    current producer snapshot, then return the original durable receipt without
    another package, receipt, or event write. Reuse of a matter/package UUID with
    any different replay-identity field must reject with ``IDENTITY_CONFLICT``.

    Returning a receipt means availability at its recorded producer revision.
    It does not establish perpetual currentness and does not copy source bytes.
    """

    def accept_if_current(
        self,
        *,
        serialized_package: bytes,
        expected_matter_id: UUID,
    ) -> PackageAvailabilityReceipt: ...


class HeldLegalSourcePackageV1Consumer:
    """Safe default used until every D08 activation dependency is supplied."""

    _REASON = (
        "D08 LegalSourcePackage v1 consumer is held: trusted issuer keys, "
        "canonical producer snapshot readback, and an atomic conditional inbox "
        "adapter are not configured"
    )

    def accept_if_current(
        self,
        *,
        serialized_package: bytes,
        expected_matter_id: UUID,
    ) -> PackageAvailabilityReceipt:
        del serialized_package, expected_matter_id
        raise LegalSourcePackageV1Held(
            PackageAcceptanceRejectionCode.INTEGRATION_HELD,
            self._REASON,
        )
