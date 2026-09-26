"""Synthetic D08 producer/consumer conformance; no evidence or live issuer used.

The test signer exercises the callback contract, not production Ed25519 trust.
Byline: Codex D08 · GPT-6 · 2026-09-23
"""

from __future__ import annotations

import hashlib
from dataclasses import replace
from datetime import UTC, datetime
from uuid import NAMESPACE_URL, UUID, uuid5

import pytest

from server.contracts.legal_source_package import (
    _DOMAIN,
    CanonicalItemReadback,
    CanonicalPackageReadback,
    LegalSourceItem,
    _canonical,
    _digest,
    _status_payload,
    assemble_package,
    assemble_status_event,
    verify_package,
    verify_status_event,
)

MATTER = UUID("11111111-1111-4111-8111-111111111111")
EVIDENCE = UUID("22222222-2222-4222-8222-222222222222")
ASSERTION = UUID("33333333-3333-4333-8333-333333333333")


def _item() -> LegalSourceItem:
    return LegalSourceItem(
        evidence_id=EVIDENCE,
        evidence_version=2,
        promotion_id=UUID("44444444-4444-4444-8444-444444444444"),
        assertion_id=ASSERTION,
        assertion_version=3,
        span_locator="source:message:12-28",
        occurrence_id=UUID("55555555-5555-4555-8555-555555555555"),
        source_version_id=UUID("66666666-6666-4666-8666-666666666666"),
        source_locator="r2://synthetic-bucket/object/version-2",
        source_content_hash="sha256:" + "a" * 64,
        content_hash="sha256:" + "b" * 64,
        source_verification_receipt_id=UUID("77777777-7777-4777-8777-777777777777"),
        human_authorization_receipt_id=UUID("88888888-8888-4888-8888-888888888888"),
    )


def _sign(message: bytes) -> bytes:
    # Signature-shape fixture only; this is not an Ed25519 implementation.
    return hashlib.sha512(b"synthetic-test-only" + message).digest()


def _verify(key_id: str, message: bytes, signature: bytes) -> bool:
    return key_id == "test-key-1" and signature == _sign(message)


def _package():
    return assemble_package(
        issuer="indicia-probata",
        issuer_key_id="test-key-1",
        matter_id=MATTER,
        package_version=7,
        committed_at=datetime(2026, 9, 23, 18, 0, tzinfo=UTC),
        items=(_item(),),
        sign=_sign,
    )


REVISION = "sha256:" + "c" * 64


def _accept(package, *, status="available", package_status="available", readback=None, acknowledge=None):
    return verify_package(
        package,
        expected_issuer="indicia-probata",
        expected_matter_id=MATTER,
        verify_signature=_verify,
        read_snapshot=lambda _: CanonicalPackageReadback(
            revision=REVISION,
            matter_id=package.matter_id,
            package_id=package.package_id,
            package_version=package.package_version,
            package_digest=package.package_digest,
            package_status=package_status,
            items=(CanonicalItemReadback(item=_item() if readback is None else readback, status=status),),
        ),
        acknowledge_if_current=(lambda *_: True) if acknowledge is None else acknowledge,
    )


def test_deterministic_package_identity_and_exact_readback() -> None:
    first = _package()
    assert _package() == first
    assert first.package_digest.startswith("sha256:")
    assert _accept(first) == REVISION


@pytest.mark.parametrize("status", [None, "revoked", "superseded", "unavailable"])
def test_current_status_fails_closed(status) -> None:
    with pytest.raises(ValueError, match="unavailable, revoked, or superseded"):
        _accept(_package(), status=status)


@pytest.mark.parametrize("status", [None, "revoked", "superseded", "unavailable"])
def test_package_currentness_fails_closed(status) -> None:
    with pytest.raises(ValueError, match="package current status"):
        _accept(_package(), package_status=status)


def test_forged_scope_digest_signature_and_locator_fail() -> None:
    package = _package()
    with pytest.raises(ValueError, match="issuer or matter mismatch"):
        verify_package(
            package,
            expected_issuer="indicia-probata",
            expected_matter_id=UUID(int=9),
            verify_signature=_verify,
            read_snapshot=lambda _: CanonicalPackageReadback(
                revision=REVISION,
                matter_id=package.matter_id,
                package_id=package.package_id,
                package_version=package.package_version,
                package_digest=package.package_digest,
                package_status="available",
                items=(CanonicalItemReadback(item=_item(), status="available"),),
            ),
            acknowledge_if_current=lambda *_: True,
        )
    with pytest.raises(ValueError, match="digest mismatch"):
        _accept(replace(package, items=(replace(package.items[0], span_locator="source:message:wrong"),)))
    with pytest.raises(ValueError, match="signature invalid"):
        _accept(replace(package, signature_hex="0" * 128))
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(package, readback=replace(_item(), source_locator="r2://synthetic-bucket/moved"))


def test_wrong_version_and_source_identity_fail_at_readback() -> None:
    package = _package()
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(package, readback=replace(_item(), source_version_id=UUID(int=11)))
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(package, readback=replace(_item(), evidence_version=3))


def test_revocation_between_snapshot_and_acknowledgment_fails_closed() -> None:
    state = {"revision": REVISION, "status": "available"}

    def snapshot(package):
        observed = CanonicalPackageReadback(
            revision=state["revision"],
            matter_id=MATTER,
            package_id=package.package_id,
            package_version=7,
            package_digest=package.package_digest,
            package_status=state["status"],
            items=(CanonicalItemReadback(item=_item(), status="available"),),
        )
        state.update(revision="sha256:" + "d" * 64, status="revoked")
        return observed

    def acknowledge(_package, expected_revision):
        return state["revision"] == expected_revision and state["status"] == "available"

    with pytest.raises(ValueError, match="revision changed"):
        verify_package(
            _package(),
            expected_issuer="indicia-probata",
            expected_matter_id=MATTER,
            verify_signature=_verify,
            read_snapshot=snapshot,
            acknowledge_if_current=acknowledge,
        )


def test_non_boolean_acknowledgment_cannot_establish_availability() -> None:
    with pytest.raises(ValueError, match="revision changed"):
        _accept(_package(), acknowledge=lambda *_: "accepted")


def test_bool_versions_are_rejected_before_signing_or_readback() -> None:
    with pytest.raises(ValueError, match="evidence_version"):
        assemble_package(
            issuer="indicia-probata",
            issuer_key_id="test-key-1",
            matter_id=MATTER,
            package_version=7,
            committed_at=datetime(2026, 9, 23, 18, tzinfo=UTC),
            items=(replace(_item(), evidence_version=True),),
            sign=_sign,
        )
    with pytest.raises(ValueError, match="package_version"):
        assemble_package(
            issuer="indicia-probata",
            issuer_key_id="test-key-1",
            matter_id=MATTER,
            package_version=True,
            committed_at=datetime(2026, 9, 23, 18, tzinfo=UTC),
            items=(_item(),),
            sign=_sign,
        )
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(_package(), readback=replace(_item(), assertion_version=True))


def test_invalid_issuer_inputs_and_unsupported_locators_do_not_sign() -> None:
    signed: list[bytes] = []

    def signer(message: bytes) -> bytes:
        signed.append(message)
        return _sign(message)

    with pytest.raises(ValueError, match="source_locator"):
        assemble_package(
            issuer="indicia-probata",
            issuer_key_id="test-key-1",
            matter_id=MATTER,
            package_version=7,
            committed_at=datetime.now(UTC),
            items=(replace(_item(), source_locator="https://other.example/source?token=secret"),),
            sign=signer,
        )
    assert signed == []


@pytest.mark.parametrize("signature", [b"", b"s" * 32, b"s" * 63, b"s" * 65])
def test_issuer_rejects_wrong_length_signature_bytes(signature: bytes) -> None:
    with pytest.raises(ValueError, match="exactly 64 Ed25519 signature bytes"):
        assemble_package(
            issuer="indicia-probata",
            issuer_key_id="test-key-1",
            matter_id=MATTER,
            package_version=7,
            committed_at=datetime(2026, 9, 23, 18, tzinfo=UTC),
            items=(_item(),),
            sign=lambda _: signature,
        )


@pytest.mark.parametrize("signature_hex", ["", "00", "a" * 126, "a" * 130, "A" * 128])
def test_consumer_rejects_wrong_length_or_case_signature_hex(signature_hex: str) -> None:
    with pytest.raises(ValueError, match="exactly 64 Ed25519 bytes"):
        _accept(replace(_package(), signature_hex=signature_hex))


def test_status_event_is_immutable_chained_and_signed() -> None:
    package = _package()
    event = assemble_status_event(
        package,
        sequence=1,
        previous_digest=package.package_digest,
        status="revoked",
        reason="synthetic correction",
        effective_at=datetime(2026, 9, 23, 19, tzinfo=UTC),
        sign=_sign,
    )
    assert (
        assemble_status_event(
            package,
            sequence=1,
            previous_digest=package.package_digest,
            status="revoked",
            reason="synthetic correction",
            effective_at=datetime(2026, 9, 23, 19, tzinfo=UTC),
            sign=_sign,
        )
        == event
    )
    verify_status_event(
        event,
        package=package,
        expected_sequence=1,
        expected_previous_digest=package.package_digest,
        verify_signature=_verify,
    )
    later = assemble_status_event(
        package,
        sequence=2,
        previous_digest=event.event_digest,
        status="superseded",
        reason="synthetic replacement",
        effective_at=datetime(2026, 9, 23, 20, tzinfo=UTC),
        sign=_sign,
    )
    verify_status_event(
        later,
        package=package,
        expected_sequence=2,
        expected_previous_digest=event.event_digest,
        verify_signature=_verify,
    )
    with pytest.raises(ValueError, match="sequence or chain mismatch"):
        verify_status_event(
            event,
            package=package,
            expected_sequence=2,
            expected_previous_digest=event.event_digest,
            verify_signature=_verify,
        )
    with pytest.raises(ValueError, match="digest mismatch"):
        verify_status_event(
            replace(event, reason="changed"),
            package=package,
            expected_sequence=1,
            expected_previous_digest=package.package_digest,
            verify_signature=_verify,
        )
    with pytest.raises(ValueError, match="identity mismatch"):
        verify_status_event(
            replace(event, event_id=UUID(int=10)),
            package=package,
            expected_sequence=1,
            expected_previous_digest=package.package_digest,
            verify_signature=_verify,
        )
    with pytest.raises(ValueError, match="status sequence"):
        verify_status_event(
            replace(event, sequence=True),
            package=package,
            expected_sequence=1,
            expected_previous_digest=package.package_digest,
            verify_signature=_verify,
        )
    with pytest.raises(ValueError, match="expected status sequence"):
        verify_status_event(
            event,
            package=package,
            expected_sequence=True,
            expected_previous_digest=package.package_digest,
            verify_signature=_verify,
        )
    with pytest.raises(ValueError, match="predate"):
        assemble_status_event(
            package,
            sequence=1,
            previous_digest=package.package_digest,
            status="revoked",
            reason="invalid ordering",
            effective_at=datetime(2026, 9, 23, 17, tzinfo=UTC),
            sign=_sign,
        )


def test_signed_first_status_event_cannot_start_from_unrelated_digest() -> None:
    package = _package()
    event = assemble_status_event(
        package,
        sequence=1,
        previous_digest=package.package_digest,
        status="revoked",
        reason="synthetic correction",
        effective_at=datetime(2026, 9, 23, 19, tzinfo=UTC),
        sign=_sign,
    )
    unrelated = "sha256:" + "e" * 64
    identity = _canonical(
        {
            "package_id": package.package_id,
            "sequence": 1,
            "previous_digest": unrelated,
            "status": "revoked",
        }
    )
    forged = replace(
        event,
        previous_digest=unrelated,
        event_id=uuid5(NAMESPACE_URL, _DOMAIN.decode() + identity.decode()),
    )
    payload = _status_payload(forged)
    forged = replace(
        forged,
        event_digest=_digest(payload),
        signature_hex=_sign(_DOMAIN + b"status\n" + payload).hex(),
    )
    with pytest.raises(ValueError, match="first status event"):
        verify_status_event(
            forged,
            package=package,
            expected_sequence=1,
            expected_previous_digest=unrelated,
            verify_signature=_verify,
        )


@pytest.mark.parametrize("signature", [b"", b"s" * 32, b"s" * 63, b"s" * 65])
def test_status_issuer_rejects_wrong_length_signature_bytes(signature: bytes) -> None:
    package = _package()
    with pytest.raises(ValueError, match="exactly 64 Ed25519 status signature bytes"):
        assemble_status_event(
            package,
            sequence=1,
            previous_digest=package.package_digest,
            status="revoked",
            reason="synthetic correction",
            effective_at=datetime(2026, 9, 23, 19, tzinfo=UTC),
            sign=lambda _: signature,
        )


@pytest.mark.parametrize("signature_hex", ["", "00", "a" * 126, "a" * 130, "A" * 128])
def test_status_consumer_rejects_wrong_length_or_case_signature_hex(signature_hex: str) -> None:
    package = _package()
    event = assemble_status_event(
        package,
        sequence=1,
        previous_digest=package.package_digest,
        status="revoked",
        reason="synthetic correction",
        effective_at=datetime(2026, 9, 23, 19, tzinfo=UTC),
        sign=_sign,
    )
    with pytest.raises(ValueError, match="exactly 64 Ed25519 bytes"):
        verify_status_event(
            replace(event, signature_hex=signature_hex),
            package=package,
            expected_sequence=1,
            expected_previous_digest=package.package_digest,
            verify_signature=lambda *_: True,
        )
