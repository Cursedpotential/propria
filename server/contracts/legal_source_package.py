"""Versioned, import-light exchange contract for promoted legal source references.

This module does not promote evidence or read a source. Its issuer input must come
from the single, committed promotion authority after human authorization and a
source reread. No production issuer or route is wired while D-152 holds.

Byline: Codex D08 · GPT-6 · 2026-09-23
"""

from __future__ import annotations

import hashlib
import json
import re
from collections.abc import Callable
from dataclasses import asdict, dataclass, replace
from datetime import UTC, datetime
from typing import Literal
from urllib.parse import urlsplit
from uuid import NAMESPACE_URL, UUID, uuid5

SCHEMA_VERSION = "legal-source-package/v1"
SIGNATURE_ALGORITHM = "ed25519"
_DOMAIN = b"propria.legal-source-package.v1\n"
_HASH = re.compile(r"sha256:[0-9a-f]{64}\Z")
_SPAN = re.compile(r"[A-Za-z][A-Za-z0-9+.-]*:[^\s?#]+\Z")
PackageStatus = Literal["available", "revoked", "superseded", "unavailable"]
Signer = Callable[[bytes], bytes]
Verifier = Callable[[str, bytes, bytes], bool]


def _uuid(value: UUID, name: str) -> None:
    if not isinstance(value, UUID) or value.int == 0:
        raise ValueError(f"{name} must be a non-nil UUID")


def _hash(value: str, name: str) -> None:
    if not isinstance(value, str) or _HASH.fullmatch(value) is None:
        raise ValueError(f"{name} must be a lowercase sha256 digest")


def _text(value: str, name: str) -> None:
    if not isinstance(value, str) or not value or value.strip() != value or any(ord(c) < 32 for c in value):
        raise ValueError(f"{name} must be nonempty and contain no control characters")


def _locator(value: str, name: str) -> None:
    _text(value, name)
    parsed = urlsplit(value)
    if parsed.scheme not in {"r2", "upload", "file", "sealed"} or not (parsed.netloc or parsed.path):
        raise ValueError(f"{name} must be a retained-source locator")
    if parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError(f"{name} must not contain credentials, query, or fragment")


def _time(value: datetime, name: str) -> None:
    if not isinstance(value, datetime) or value.tzinfo is None or value.utcoffset() is None:
        raise ValueError(f"{name} must be timezone-aware")


def _positive_int(value: int, name: str) -> None:
    if type(value) is not int or value < 1:
        raise ValueError(f"{name} must be a positive integer")


def _canonical(value: object) -> bytes:
    def normalize(obj: object) -> object:
        if isinstance(obj, UUID):
            return str(obj)
        if isinstance(obj, datetime):
            return obj.astimezone(UTC).isoformat(timespec="microseconds").replace("+00:00", "Z")
        if isinstance(obj, dict):
            return {key: normalize(item) for key, item in obj.items()}
        if isinstance(obj, (list, tuple)):
            return [normalize(item) for item in obj]
        return obj

    return json.dumps(normalize(value), sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def _digest(value: bytes) -> str:
    return "sha256:" + hashlib.sha256(value).hexdigest()


@dataclass(frozen=True)
class LegalSourceItem:
    evidence_id: UUID
    evidence_version: int
    promotion_id: UUID
    assertion_id: UUID
    assertion_version: int
    span_locator: str
    occurrence_id: UUID
    source_version_id: UUID
    source_locator: str
    source_content_hash: str
    content_hash: str
    source_verification_receipt_id: UUID
    human_authorization_receipt_id: UUID

    def validate(self) -> None:
        for name in (
            "evidence_id",
            "promotion_id",
            "assertion_id",
            "occurrence_id",
            "source_version_id",
            "source_verification_receipt_id",
            "human_authorization_receipt_id",
        ):
            _uuid(getattr(self, name), name)
        _positive_int(self.evidence_version, "evidence_version")
        _positive_int(self.assertion_version, "assertion_version")
        if _SPAN.fullmatch(self.span_locator) is None:
            raise ValueError("span_locator must be an exact, nonempty source span reference")
        _locator(self.source_locator, "source_locator")
        _hash(self.source_content_hash, "source_content_hash")
        _hash(self.content_hash, "content_hash")


@dataclass(frozen=True)
class LegalSourcePackage:
    schema_version: str
    package_id: UUID
    issuer: str
    issuer_key_id: str
    signature_algorithm: str
    matter_id: UUID
    package_version: int
    issued_at: datetime
    status: PackageStatus
    items: tuple[LegalSourceItem, ...]
    package_digest: str
    signature_hex: str


@dataclass(frozen=True)
class CanonicalItemReadback:
    item: LegalSourceItem
    status: PackageStatus


@dataclass(frozen=True)
class CanonicalPackageReadback:
    """One producer snapshot revision covering package and every exact item."""

    revision: str
    matter_id: UUID
    package_id: UUID
    package_version: int
    package_digest: str
    package_status: PackageStatus
    items: tuple[CanonicalItemReadback, ...]


ReadSnapshot = Callable[[LegalSourcePackage], CanonicalPackageReadback | None]
ConditionalAcknowledge = Callable[[LegalSourcePackage, str], bool]


@dataclass(frozen=True)
class LegalPackageStatusEvent:
    """Append-only transition; it never rewrites the signed package snapshot."""

    event_id: UUID
    package_id: UUID
    package_digest: str
    sequence: int
    previous_digest: str
    status: PackageStatus
    reason: str
    effective_at: datetime
    event_digest: str
    signature_hex: str


def manifest_bytes(package: LegalSourcePackage) -> bytes:
    """Canonical UTF-8 bytes signed by the producer and recomputed by consumers."""
    return _canonical(
        {
            "schema_version": package.schema_version,
            "package_id": package.package_id,
            "issuer": package.issuer,
            "issuer_key_id": package.issuer_key_id,
            "signature_algorithm": package.signature_algorithm,
            "matter_id": package.matter_id,
            "package_version": package.package_version,
            "issued_at": package.issued_at,
            "status": package.status,
            "items": [asdict(item) for item in package.items],
        }
    )


def _validate_shape(package: LegalSourcePackage) -> None:
    if package.schema_version != SCHEMA_VERSION:
        raise ValueError("unsupported package schema_version")
    if package.signature_algorithm != SIGNATURE_ALGORITHM:
        raise ValueError("unsupported package signature_algorithm")
    _uuid(package.package_id, "package_id")
    _uuid(package.matter_id, "matter_id")
    _text(package.issuer, "issuer")
    _text(package.issuer_key_id, "issuer_key_id")
    _time(package.issued_at, "issued_at")
    _positive_int(package.package_version, "package_version")
    if package.status not in {"available", "revoked", "superseded", "unavailable"}:
        raise ValueError("unsupported package status")
    if not package.items:
        raise ValueError("package must contain at least one item")
    if type(package.items) is not tuple:
        raise ValueError("package items must be an immutable tuple")
    identities: set[tuple[UUID, int, UUID, int]] = set()
    for item in package.items:
        item.validate()
        identity = (item.evidence_id, item.evidence_version, item.assertion_id, item.assertion_version)
        if identity in identities:
            raise ValueError("duplicate evidence assertion version")
        identities.add(identity)
    if (
        tuple(
            sorted(
                package.items,
                key=lambda item: (
                    str(item.evidence_id),
                    item.evidence_version,
                    str(item.assertion_id),
                    item.assertion_version,
                ),
            )
        )
        != package.items
    ):
        raise ValueError("package items must be in canonical identity order")
    _hash(package.package_digest, "package_digest")
    if not package.signature_hex or re.fullmatch(r"[0-9a-f]+", package.signature_hex) is None:
        raise ValueError("signature_hex must be lowercase hexadecimal")


def assemble_package(
    *,
    issuer: str,
    issuer_key_id: str,
    matter_id: UUID,
    package_version: int,
    committed_at: datetime,
    items: tuple[LegalSourceItem, ...],
    sign: Signer,
) -> LegalSourcePackage:
    """Assemble from verified committed-promotion readback; caller owns that gate.

    The signing key must remain in the producer's separate permission boundary.
    Passing unverified context or a UI flag to this function is not authorization.
    """
    ordered = tuple(
        sorted(
            items,
            key=lambda item: (
                str(item.evidence_id),
                item.evidence_version,
                str(item.assertion_id),
                item.assertion_version,
            ),
        )
    )
    draft = LegalSourcePackage(
        schema_version=SCHEMA_VERSION,
        package_id=UUID(int=1),
        issuer=issuer,
        issuer_key_id=issuer_key_id,
        signature_algorithm=SIGNATURE_ALGORITHM,
        matter_id=matter_id,
        package_version=package_version,
        issued_at=committed_at,
        status="available",
        items=ordered,
        package_digest="sha256:" + "0" * 64,
        signature_hex="00",
    )
    _validate_shape(draft)
    # Identity excludes signing metadata and is stable across transport retries.
    identity = _canonical(
        {
            "issuer": issuer,
            "matter_id": matter_id,
            "package_version": package_version,
            "evidence": [(item.evidence_id, item.evidence_version) for item in ordered],
        }
    )
    draft = replace(draft, package_id=uuid5(NAMESPACE_URL, _DOMAIN.decode() + identity.decode()))
    payload = manifest_bytes(draft)
    signature = sign(_DOMAIN + payload)
    if not isinstance(signature, bytes) or not signature:
        raise ValueError("issuer signer returned no signature")
    return replace(draft, package_digest=_digest(payload), signature_hex=signature.hex())


def verify_package(
    package: LegalSourcePackage,
    *,
    expected_issuer: str,
    expected_matter_id: UUID,
    verify_signature: Verifier,
    read_snapshot: ReadSnapshot,
    acknowledge_if_current: ConditionalAcknowledge,
) -> str:
    """Fail closed on every import boundary; success means availability only.

    The verifier must resolve issuer_key_id to a trusted issuer public key. The
    The producer snapshot must atomically cover package and all item currentness.
    The acknowledgment callback must compare that same producer revision and
    durably record one consumer availability receipt before returning True. A
    missing/changed revision or unavailable producer fails closed. Success is
    availability at the acknowledged revision, not perpetual currentness.
    """
    _validate_shape(package)
    if package.issuer != expected_issuer or package.matter_id != expected_matter_id:
        raise ValueError("package issuer or matter mismatch")
    payload = manifest_bytes(package)
    if _digest(payload) != package.package_digest:
        raise ValueError("package digest mismatch")
    expected_id = uuid5(
        NAMESPACE_URL,
        _DOMAIN.decode()
        + _canonical(
            {
                "issuer": package.issuer,
                "matter_id": package.matter_id,
                "package_version": package.package_version,
                "evidence": [(item.evidence_id, item.evidence_version) for item in package.items],
            }
        ).decode(),
    )
    if package.package_id != expected_id:
        raise ValueError("package identity mismatch")
    if verify_signature(package.issuer_key_id, _DOMAIN + payload, bytes.fromhex(package.signature_hex)) is not True:
        raise ValueError("package issuer signature invalid")
    if package.status != "available":
        raise ValueError("package is not available")
    snapshot = read_snapshot(package)
    if type(snapshot) is not CanonicalPackageReadback:
        raise ValueError("canonical producer snapshot unavailable")
    _hash(snapshot.revision, "snapshot revision")
    _uuid(snapshot.matter_id, "snapshot matter_id")
    _uuid(snapshot.package_id, "snapshot package_id")
    _positive_int(snapshot.package_version, "snapshot package_version")
    _hash(snapshot.package_digest, "snapshot package_digest")
    if (
        snapshot.matter_id != package.matter_id
        or snapshot.package_id != package.package_id
        or snapshot.package_version != package.package_version
        or snapshot.package_digest != package.package_digest
    ):
        raise ValueError("canonical producer package identity readback mismatch")
    if snapshot.package_status != "available":
        raise ValueError("package current status unavailable, revoked, or superseded")
    if type(snapshot.items) is not tuple or len(snapshot.items) != len(package.items):
        raise ValueError("canonical producer item readback mismatch")
    for item, current in zip(package.items, snapshot.items, strict=True):
        if type(current) is not CanonicalItemReadback:
            raise ValueError("canonical producer item readback mismatch")
        if current.status != "available":
            raise ValueError("evidence version unavailable, revoked, or superseded")
        if type(current.item) is not LegalSourceItem or _canonical(asdict(current.item)) != _canonical(asdict(item)):
            raise ValueError("source, assertion, version, digest, or locator readback mismatch")
    if acknowledge_if_current(package, snapshot.revision) is not True:
        raise ValueError("canonical producer revision changed before durable acknowledgment")
    return snapshot.revision


def _status_payload(event: LegalPackageStatusEvent) -> bytes:
    return _canonical(
        {
            "event_id": event.event_id,
            "package_id": event.package_id,
            "package_digest": event.package_digest,
            "sequence": event.sequence,
            "previous_digest": event.previous_digest,
            "status": event.status,
            "reason": event.reason,
            "effective_at": event.effective_at,
        }
    )


def assemble_status_event(
    package: LegalSourcePackage,
    *,
    sequence: int,
    previous_digest: str,
    status: PackageStatus,
    reason: str,
    effective_at: datetime,
    sign: Signer,
) -> LegalPackageStatusEvent:
    """Create one deterministic signed event for a committed status transition."""
    _validate_shape(package)
    _positive_int(sequence, "status sequence")
    _hash(previous_digest, "previous_digest")
    if sequence == 1 and previous_digest != package.package_digest:
        raise ValueError("first status event must chain from the package digest")
    if status not in {"revoked", "superseded", "unavailable"}:
        raise ValueError("status event must invalidate availability")
    _text(reason, "reason")
    _time(effective_at, "effective_at")
    if effective_at < package.issued_at:
        raise ValueError("status event cannot predate package issuance")
    identity = _canonical(
        {
            "package_id": package.package_id,
            "sequence": sequence,
            "previous_digest": previous_digest,
            "status": status,
        }
    )
    draft = LegalPackageStatusEvent(
        event_id=uuid5(NAMESPACE_URL, _DOMAIN.decode() + identity.decode()),
        package_id=package.package_id,
        package_digest=package.package_digest,
        sequence=sequence,
        previous_digest=previous_digest,
        status=status,
        reason=reason,
        effective_at=effective_at,
        event_digest="sha256:" + "0" * 64,
        signature_hex="00",
    )
    payload = _status_payload(draft)
    signature = sign(_DOMAIN + b"status\n" + payload)
    if not isinstance(signature, bytes) or not signature:
        raise ValueError("issuer signer returned no status signature")
    return replace(draft, event_digest=_digest(payload), signature_hex=signature.hex())


def verify_status_event(
    event: LegalPackageStatusEvent,
    *,
    package: LegalSourcePackage,
    expected_sequence: int,
    expected_previous_digest: str,
    verify_signature: Verifier,
) -> None:
    """Check a chained transition before applying it to a durable consumer inbox."""
    _validate_shape(package)
    _uuid(event.event_id, "event_id")
    _uuid(event.package_id, "status package_id")
    _positive_int(event.sequence, "status sequence")
    _positive_int(expected_sequence, "expected status sequence")
    if event.package_id != package.package_id or event.package_digest != package.package_digest:
        raise ValueError("status event package mismatch")
    if event.sequence != expected_sequence or event.previous_digest != expected_previous_digest:
        raise ValueError("status event sequence or chain mismatch")
    _hash(event.previous_digest, "previous_digest")
    _time(event.effective_at, "effective_at")
    if event.effective_at < package.issued_at:
        raise ValueError("status event cannot predate package issuance")
    _text(event.reason, "reason")
    if event.status not in {"revoked", "superseded", "unavailable"}:
        raise ValueError("status event cannot activate a package")
    identity = _canonical(
        {
            "package_id": package.package_id,
            "sequence": event.sequence,
            "previous_digest": event.previous_digest,
            "status": event.status,
        }
    )
    if event.event_id != uuid5(NAMESPACE_URL, _DOMAIN.decode() + identity.decode()):
        raise ValueError("status event identity mismatch")
    payload = _status_payload(event)
    if event.event_digest != _digest(payload):
        raise ValueError("status event digest mismatch")
    if (
        verify_signature(package.issuer_key_id, _DOMAIN + b"status\n" + payload, bytes.fromhex(event.signature_hex))
        is not True
    ):
        raise ValueError("status event signature invalid")
