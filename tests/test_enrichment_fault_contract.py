"""Focused W18 null/truncated enrichment and snapshot-retention regressions.

Byline: Codex · GPT-6-Astra · 2026-09-23
"""

from __future__ import annotations

from datetime import datetime, timezone

from pydantic import BaseModel, ConfigDict
import pytest

from server.contracts.enrichment import (
    EnrichmentAttempt,
    EnrichmentSnapshot,
    EnrichmentSource,
    decide_snapshot_publication,
    evaluate_enrichment_output,
    recover_last_valid_snapshot,
)


SHA_SOURCE = "11" * 32
SHA_CONFIG = "22" * 32
T1 = datetime(2026, 9, 23, 20, 0, tzinfo=timezone.utc)
T2 = datetime(2026, 9, 23, 20, 5, tzinfo=timezone.utc)


class _Payload(BaseModel):
    model_config = ConfigDict(extra="forbid")

    title: str
    date_basis: str | None
    summary: str


def _source(source_id: str = "source-1", version: str = "v1") -> EnrichmentSource:
    return EnrichmentSource(
        source_id=source_id,
        source_version=version,
        source_sha256=SHA_SOURCE,
        inventory_ref=f"inventory://{source_id}/{version}",
        extracted_text_ref=f"text://{source_id}/{version}",
    )


def _attempt(
    raw_output: str,
    *,
    attempt_id: str = "attempt-1",
    finish_reason: str = "stop",
    source: EnrichmentSource | None = None,
) -> EnrichmentAttempt:
    return EnrichmentAttempt(
        attempt_id=attempt_id,
        source=source or _source(),
        model_id="fixture-model-v1",
        model_config_sha256=SHA_CONFIG,
        raw_output=raw_output,
        finish_reason=finish_reason,
    )


def _valid(attempt_id: str = "attempt-valid", source: EnrichmentSource | None = None):
    return evaluate_enrichment_output(
        _attempt(
            '{"title":"Document","date_basis":null,"summary":"Retained text"}',
            attempt_id=attempt_id,
            source=source,
        ),
        _Payload,
    )


def _snapshot(
    snapshot_id: str = "snapshot-1",
    sequence: int = 1,
    source: EnrichmentSource | None = None,
    validated_at: datetime = T1,
) -> EnrichmentSnapshot:
    decision = decide_snapshot_publication(
        _valid(f"attempt-{snapshot_id}", source),
        prior_active=None,
        candidate_snapshot_id=snapshot_id,
        candidate_sequence=sequence,
        validated_at=validated_at,
    )
    assert decision.active_snapshot is not None
    return decision.active_snapshot


def test_nullable_field_is_validated_without_inventing_a_value() -> None:
    evaluation = _valid()

    assert evaluation.receipt.status == "completed"
    assert evaluation.receipt.reason == "validated"
    assert evaluation.payload is not None
    assert evaluation.payload.date_basis is None
    assert evaluation.receipt.source == _source()
    assert evaluation.receipt.validated_payload_sha256 is not None


@pytest.mark.parametrize(
    ("raw_output", "finish_reason"),
    [
        ('{"title":"Document","date_basis":', "stop"),
        ('{"title":"Document","date_basis":null}', "length"),
    ],
)
def test_truncated_output_is_partial_and_retains_source_linkage(raw_output: str, finish_reason: str) -> None:
    evaluation = evaluate_enrichment_output(
        _attempt(raw_output, finish_reason=finish_reason),  # type: ignore[arg-type]
        _Payload,
    )

    assert evaluation.receipt.status == "partial"
    assert evaluation.receipt.reason == "truncated_output"
    assert evaluation.receipt.retryable is True
    assert evaluation.receipt.source.inventory_ref == "inventory://source-1/v1"
    assert evaluation.receipt.source.extracted_text_ref == "text://source-1/v1"
    assert evaluation.receipt.validated_payload_sha256 is None
    assert evaluation.payload is None


def test_malformed_nontruncated_json_is_failed_not_completed() -> None:
    evaluation = evaluate_enrichment_output(
        _attempt('{"title": nope, "date_basis": null, "summary": "x"}'),
        _Payload,
    )

    assert evaluation.receipt.status == "failed"
    assert evaluation.receipt.reason == "invalid_json"
    assert evaluation.payload is None


def test_null_required_field_is_a_typed_schema_failure() -> None:
    evaluation = evaluate_enrichment_output(
        _attempt('{"title":null,"date_basis":null,"summary":"Retained text"}'),
        _Payload,
    )

    assert evaluation.receipt.status == "failed"
    assert evaluation.receipt.reason == "schema_validation"
    assert evaluation.receipt.retryable is True
    assert [(issue.path, issue.code) for issue in evaluation.receipt.issues] == [(("title",), "string_type")]
    assert evaluation.payload is None


@pytest.mark.parametrize(
    ("finish_reason", "expected_status"),
    [
        ("provider_error", "failed"),
        ("timeout", "failed"),
        ("cancelled", "failed"),
        ("unsupported", "unsupported"),
    ],
)
def test_noncontent_terminal_outcomes_are_never_completed(finish_reason: str, expected_status: str) -> None:
    evaluation = evaluate_enrichment_output(
        _attempt("", finish_reason=finish_reason),  # type: ignore[arg-type]
        _Payload,
    )

    assert evaluation.receipt.status == expected_status
    assert evaluation.receipt.reason == finish_reason
    assert evaluation.receipt.validated_payload_sha256 is None


def test_partial_attempt_keeps_last_valid_snapshot_active() -> None:
    active = _snapshot()
    truncated = evaluate_enrichment_output(_attempt('{"title":"new"'), _Payload)

    decision = decide_snapshot_publication(
        truncated,
        prior_active=active,
        candidate_snapshot_id="snapshot-2",
        candidate_sequence=2,
        validated_at=T2,
    )

    assert decision.published is False
    assert decision.active_snapshot == active
    assert decision.receipt.status == "partial"


def test_schema_failure_keeps_last_valid_snapshot_active() -> None:
    active = _snapshot()
    invalid = evaluate_enrichment_output(
        _attempt('{"title":null,"date_basis":null,"summary":"new"}'),
        _Payload,
    )

    decision = decide_snapshot_publication(
        invalid,
        prior_active=active,
        candidate_snapshot_id="snapshot-2",
        candidate_sequence=2,
        validated_at=T2,
    )

    assert decision.published is False
    assert decision.active_snapshot == active
    assert decision.receipt.status == "failed"


def test_valid_attempt_advances_snapshot_with_exact_source_linkage() -> None:
    active = _snapshot()
    next_source_version = _source(version="v2")

    decision = decide_snapshot_publication(
        _valid("attempt-2", next_source_version),
        prior_active=active,
        candidate_snapshot_id="snapshot-2",
        candidate_sequence=2,
        validated_at=T2,
    )

    assert decision.published is True
    assert decision.active_snapshot is not None
    assert decision.active_snapshot.snapshot_id == "snapshot-2"
    assert decision.active_snapshot.sequence == 2
    assert decision.active_snapshot.source.source_version == "v2"
    assert decision.active_snapshot.source.extracted_text_ref == "text://source-1/v2"


def test_identical_publication_retry_is_an_idempotent_noop() -> None:
    evaluation = _valid()
    first = decide_snapshot_publication(
        evaluation,
        prior_active=None,
        candidate_snapshot_id="snapshot-1",
        candidate_sequence=1,
        validated_at=T1,
    )
    assert first.active_snapshot is not None

    retry = decide_snapshot_publication(
        evaluation,
        prior_active=first.active_snapshot,
        candidate_snapshot_id="snapshot-1",
        candidate_sequence=1,
        validated_at=T1,
    )

    assert retry.published is False
    assert retry.active_snapshot == first.active_snapshot


def test_same_sequence_with_different_validated_payload_fails_closed() -> None:
    active = _snapshot()
    changed = evaluate_enrichment_output(
        _attempt('{"title":"Changed","date_basis":null,"summary":"Different"}', attempt_id="attempt-2"),
        _Payload,
    )

    with pytest.raises(ValueError, match="conflicts with or precedes"):
        decide_snapshot_publication(
            changed,
            prior_active=active,
            candidate_snapshot_id="snapshot-conflict",
            candidate_sequence=1,
            validated_at=T2,
        )


def test_publication_refuses_prior_snapshot_from_another_source() -> None:
    wrong_prior = _snapshot(source=_source("other-source"))

    with pytest.raises(ValueError, match="different source"):
        decide_snapshot_publication(
            _valid(),
            prior_active=wrong_prior,
            candidate_snapshot_id="snapshot-2",
            candidate_sequence=2,
            validated_at=T2,
        )


def test_restart_recovers_highest_validated_snapshot_not_failed_attempt() -> None:
    first = _snapshot()
    second = _snapshot("snapshot-2", 2, validated_at=T2)
    failed = evaluate_enrichment_output(_attempt('{"title":"broken"'), _Payload)
    failed_decision = decide_snapshot_publication(
        failed,
        prior_active=second,
        candidate_snapshot_id="snapshot-3",
        candidate_sequence=3,
        validated_at=T2,
    )

    recovered = recover_last_valid_snapshot("source-1", (second, first))

    assert failed_decision.published is False
    assert failed_decision.active_snapshot == second
    assert recovered == second


def test_restart_fails_closed_on_conflicting_validated_sequence() -> None:
    first = _snapshot("snapshot-1", 1)
    conflict = _snapshot("snapshot-conflict", 1, validated_at=T2)

    with pytest.raises(ValueError, match="conflicting validated snapshots"):
        recover_last_valid_snapshot("source-1", (first, conflict))
