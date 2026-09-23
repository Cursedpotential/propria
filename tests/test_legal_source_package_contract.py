"""Synthetic D08 producer/consumer conformance; no evidence or live issuer used.

The test signer exercises the callback contract, not production Ed25519 trust.
Byline: Codex D08 · GPT-6 · 2026-09-23
"""

from __future__ import annotations

import hashlib
from dataclasses import replace
from datetime import UTC, datetime
from uuid import UUID

import pytest

from server.contracts.legal_source_package import (
    LegalSourceItem,
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
    return hashlib.sha256(b"synthetic-test-only" + message).digest()


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


def _accept(package, *, status="available", package_status="available", readback=None):
    verify_package(
        package,
        expected_issuer="indicia-probata",
        expected_matter_id=MATTER,
        verify_signature=_verify,
        current_package=lambda *_: package_status,
        current_status=lambda evidence_id, version: status if (evidence_id, version) == (EVIDENCE, 2) else None,
        current_item=lambda evidence_id, version, assertion_id, assertion_version: (
            (_item() if readback is None else readback)
            if (evidence_id, version, assertion_id, assertion_version) == (EVIDENCE, 2, ASSERTION, 3)
            else None
        ),
    )


def test_deterministic_package_identity_and_exact_readback() -> None:
    first = _package()
    assert _package() == first
    assert first.package_digest.startswith("sha256:")
    _accept(first)


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
            current_package=lambda *_: "available",
            current_status=lambda *_: "available",
            current_item=lambda *_: _item(),
        )
    with pytest.raises(ValueError, match="digest mismatch"):
        _accept(replace(package, items=(replace(package.items[0], span_locator="source:message:wrong"),)))
    with pytest.raises(ValueError, match="signature invalid"):
        _accept(replace(package, signature_hex="00"))
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(package, readback=replace(_item(), source_locator="r2://synthetic-bucket/moved"))


def test_wrong_version_and_source_identity_fail_at_readback() -> None:
    package = _package()
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(package, readback=replace(_item(), source_version_id=UUID(int=11)))
    with pytest.raises(ValueError, match="readback mismatch"):
        _accept(package, readback=replace(_item(), evidence_version=3))


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
