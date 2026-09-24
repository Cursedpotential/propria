"""Fail-closed enrichment output and snapshot-publication contracts.

The evaluator is deliberately framework-neutral: it performs no provider call,
persistence, orchestration, or projection. A caller can run it directly or as
one Temporal/n8n Activity, persist the returned receipt, and separately apply
the publication decision.

Byline: Codex · GPT-6-Astra · 2026-09-23
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from hashlib import sha256
import json
from typing import Literal, TypeVar

from pydantic import BaseModel, ConfigDict, Field, ValidationError, field_validator, model_validator


_HEX_SHA256 = r"^[0-9a-f]{64}$"
PayloadT = TypeVar("PayloadT", bound=BaseModel)


class EnrichmentSource(BaseModel):
    """Exact retained-source and extracted-text linkage for one attempt."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    source_id: str = Field(min_length=1)
    source_version: str = Field(min_length=1)
    source_sha256: str = Field(pattern=_HEX_SHA256)
    inventory_ref: str = Field(min_length=1)
    extracted_text_ref: str = Field(min_length=1)

    @field_validator("source_id", "source_version", "inventory_ref", "extracted_text_ref")
    @classmethod
    def _nonblank(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("source linkage values must not be blank")
        return value


FinishReason = Literal["stop", "length", "provider_error", "timeout", "cancelled", "unsupported"]
OutcomeStatus = Literal["completed", "partial", "failed", "unsupported"]
OutcomeReason = Literal[
    "validated",
    "empty_output",
    "truncated_output",
    "invalid_json",
    "schema_validation",
    "provider_error",
    "timeout",
    "cancelled",
    "unsupported",
]


class EnrichmentAttempt(BaseModel):
    """Exact provider result presented to schema validation."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    attempt_id: str = Field(min_length=1)
    source: EnrichmentSource
    model_id: str = Field(min_length=1)
    model_config_sha256: str = Field(pattern=_HEX_SHA256)
    raw_output: str
    finish_reason: FinishReason = "stop"

    @field_validator("attempt_id", "model_id")
    @classmethod
    def _nonblank(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("attempt and model identifiers must not be blank")
        return value


class EnrichmentIssue(BaseModel):
    """Bounded, non-payload-bearing validation detail."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    path: tuple[str | int, ...] = ()
    code: str = Field(min_length=1)


class EnrichmentReceipt(BaseModel):
    """Persistable outcome that always retains source and exact-output identity."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    attempt_id: str
    source: EnrichmentSource
    model_id: str
    model_config_sha256: str = Field(pattern=_HEX_SHA256)
    finish_reason: FinishReason
    status: OutcomeStatus
    reason: OutcomeReason
    retryable: bool
    raw_output_sha256: str = Field(pattern=_HEX_SHA256)
    validated_payload_sha256: str | None = Field(default=None, pattern=_HEX_SHA256)
    issues: tuple[EnrichmentIssue, ...] = ()

    @model_validator(mode="after")
    def _outcome_is_consistent(self) -> EnrichmentReceipt:
        if self.status == "completed":
            if self.reason != "validated" or self.validated_payload_sha256 is None or self.issues:
                raise ValueError("completed enrichment requires one validated payload and no issues")
        elif self.validated_payload_sha256 is not None:
            raise ValueError("non-completed enrichment cannot claim a validated payload")
        if self.status == "unsupported" and self.reason != "unsupported":
            raise ValueError("unsupported status requires unsupported reason")
        return self


@dataclass(frozen=True)
class EnrichmentEvaluation:
    """In-process result; only ``receipt`` crosses a durable activity boundary."""

    receipt: EnrichmentReceipt
    payload: BaseModel | None


class EnrichmentSnapshot(BaseModel):
    """One immutable, schema-validated snapshot eligible for activation."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    snapshot_id: str = Field(min_length=1)
    sequence: int = Field(ge=1)
    source: EnrichmentSource
    attempt_id: str = Field(min_length=1)
    model_id: str = Field(min_length=1)
    model_config_sha256: str = Field(pattern=_HEX_SHA256)
    raw_output_sha256: str = Field(pattern=_HEX_SHA256)
    payload_sha256: str = Field(pattern=_HEX_SHA256)
    validated_at: datetime

    @field_validator("validated_at")
    @classmethod
    def _aware_time(cls, value: datetime) -> datetime:
        if value.tzinfo is None or value.utcoffset() is None:
            raise ValueError("validated_at requires an explicit timezone")
        return value


class SnapshotPublicationDecision(BaseModel):
    """Atomic decision: publish the new validated snapshot or retain the prior one."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    receipt: EnrichmentReceipt
    published: bool
    active_snapshot: EnrichmentSnapshot | None

    @model_validator(mode="after")
    def _publication_matches_receipt(self) -> SnapshotPublicationDecision:
        if self.published and self.receipt.status != "completed":
            raise ValueError("only completed enrichment may publish a snapshot")
        if self.published and self.active_snapshot is None:
            raise ValueError("published decision requires the new active snapshot")
        return self


def _text_sha256(value: str) -> str:
    return sha256(value.encode("utf-8")).hexdigest()


def _payload_sha256(payload: BaseModel) -> str:
    encoded = json.dumps(
        payload.model_dump(mode="json"),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    )
    return _text_sha256(encoded)


def _receipt(
    attempt: EnrichmentAttempt,
    *,
    status: OutcomeStatus,
    reason: OutcomeReason,
    retryable: bool,
    payload_sha256: str | None = None,
    issues: tuple[EnrichmentIssue, ...] = (),
) -> EnrichmentReceipt:
    return EnrichmentReceipt(
        attempt_id=attempt.attempt_id,
        source=attempt.source,
        model_id=attempt.model_id,
        model_config_sha256=attempt.model_config_sha256,
        finish_reason=attempt.finish_reason,
        status=status,
        reason=reason,
        retryable=retryable,
        raw_output_sha256=_text_sha256(attempt.raw_output),
        validated_payload_sha256=payload_sha256,
        issues=issues,
    )


def _looks_truncated(error: json.JSONDecodeError, raw_output: str) -> bool:
    stripped = raw_output.rstrip()
    if not stripped:
        return False
    if error.msg.startswith("Unterminated"):
        return True
    trailing_incomplete = stripped[-1] in "{[,:"
    at_end = error.pos >= max(0, len(stripped) - 1)
    return trailing_incomplete or (at_end and error.msg.startswith("Expecting"))


def evaluate_enrichment_output(
    attempt: EnrichmentAttempt,
    payload_schema: type[PayloadT],
) -> EnrichmentEvaluation:
    """Validate one exact provider output and return a typed, source-linked receipt."""

    terminal_failures: dict[FinishReason, tuple[OutcomeStatus, OutcomeReason, bool]] = {
        "length": ("partial", "truncated_output", True),
        "provider_error": ("failed", "provider_error", True),
        "timeout": ("failed", "timeout", True),
        "cancelled": ("failed", "cancelled", True),
        "unsupported": ("unsupported", "unsupported", False),
    }
    if attempt.finish_reason != "stop":
        status, reason, retryable = terminal_failures[attempt.finish_reason]
        return EnrichmentEvaluation(
            receipt=_receipt(attempt, status=status, reason=reason, retryable=retryable),
            payload=None,
        )

    if not attempt.raw_output.strip():
        return EnrichmentEvaluation(
            receipt=_receipt(attempt, status="failed", reason="empty_output", retryable=True),
            payload=None,
        )

    try:
        decoded = json.loads(attempt.raw_output)
    except json.JSONDecodeError as error:
        truncated = _looks_truncated(error, attempt.raw_output)
        issue = EnrichmentIssue(path=(), code="json_truncated" if truncated else "json_invalid")
        return EnrichmentEvaluation(
            receipt=_receipt(
                attempt,
                status="partial" if truncated else "failed",
                reason="truncated_output" if truncated else "invalid_json",
                retryable=True,
                issues=(issue,),
            ),
            payload=None,
        )

    try:
        payload = payload_schema.model_validate(decoded)
    except ValidationError as error:
        issues = tuple(
            EnrichmentIssue(path=tuple(item["loc"]), code=str(item["type"]))
            for item in error.errors(include_input=False)
        )
        return EnrichmentEvaluation(
            receipt=_receipt(
                attempt,
                status="failed",
                reason="schema_validation",
                retryable=True,
                issues=issues,
            ),
            payload=None,
        )

    payload_digest = _payload_sha256(payload)
    return EnrichmentEvaluation(
        receipt=_receipt(
            attempt,
            status="completed",
            reason="validated",
            retryable=False,
            payload_sha256=payload_digest,
        ),
        payload=payload,
    )


def decide_snapshot_publication(
    evaluation: EnrichmentEvaluation,
    *,
    prior_active: EnrichmentSnapshot | None,
    candidate_snapshot_id: str,
    candidate_sequence: int,
    validated_at: datetime,
) -> SnapshotPublicationDecision:
    """Publish only validated output; otherwise leave the last valid snapshot active."""

    receipt = evaluation.receipt
    if prior_active is not None and prior_active.source.source_id != receipt.source.source_id:
        raise ValueError("prior active snapshot belongs to a different source")
    if receipt.status != "completed":
        return SnapshotPublicationDecision(receipt=receipt, published=False, active_snapshot=prior_active)
    if evaluation.payload is None or receipt.validated_payload_sha256 is None:
        raise ValueError("completed receipt is missing its validated payload")
    if not candidate_snapshot_id.strip():
        raise ValueError("candidate snapshot id must not be blank")

    snapshot = EnrichmentSnapshot(
        snapshot_id=candidate_snapshot_id,
        sequence=candidate_sequence,
        source=receipt.source,
        attempt_id=receipt.attempt_id,
        model_id=receipt.model_id,
        model_config_sha256=receipt.model_config_sha256,
        raw_output_sha256=receipt.raw_output_sha256,
        payload_sha256=receipt.validated_payload_sha256,
        validated_at=validated_at,
    )
    if prior_active is not None and candidate_sequence <= prior_active.sequence:
        if snapshot == prior_active:
            return SnapshotPublicationDecision(receipt=receipt, published=False, active_snapshot=prior_active)
        raise ValueError("candidate sequence conflicts with or precedes the active snapshot")
    return SnapshotPublicationDecision(receipt=receipt, published=True, active_snapshot=snapshot)


def recover_last_valid_snapshot(
    source_id: str,
    snapshots: tuple[EnrichmentSnapshot, ...],
) -> EnrichmentSnapshot | None:
    """Recover the highest nonconflicting validated sequence for one source."""

    if not source_id.strip():
        raise ValueError("source id must not be blank")
    matching = [snapshot for snapshot in snapshots if snapshot.source.source_id == source_id]
    by_sequence: dict[int, EnrichmentSnapshot] = {}
    for snapshot in matching:
        prior = by_sequence.get(snapshot.sequence)
        if prior is not None and prior != snapshot:
            raise ValueError(f"conflicting validated snapshots at sequence {snapshot.sequence}")
        by_sequence[snapshot.sequence] = snapshot
    return max(by_sequence.values(), key=lambda snapshot: snapshot.sequence, default=None)
