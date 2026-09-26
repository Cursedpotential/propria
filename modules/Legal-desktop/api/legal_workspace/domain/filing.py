"""Filing-readiness checklist. Never files or serves.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Computed checks plus owner verification. Agents cannot mark filed.
"""

from __future__ import annotations

from datetime import UTC, datetime
from enum import Enum
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class CheckState(str, Enum):
    PASS = "pass"
    FAIL = "fail"
    UNKNOWN = "unknown"


class FilingCheck(BaseModel):
    check_id: str
    label: str
    state: CheckState
    reason: str
    blocking: bool = True
    human: bool = False


class FilingOverride(BaseModel):
    override_id: UUID = Field(default_factory=uuid4)
    check_id: str
    state: CheckState
    note: str
    reviewer: str = "owner"
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))


class FilingOverrideCreate(BaseModel):
    check_id: str
    state: CheckState
    note: str
    reviewer: str = "owner"


class FilingReadiness(BaseModel):
    ready: bool
    filed: bool = False
    checks: list[FilingCheck]
    blocking_count: int


def evaluate_filing_readiness(
    *,
    package_imported: bool,
    docket_number: str | None,
    judge: str | None,
    review_approved: bool,
    release_count: int,
    last_release_filed: bool,
    overrides: list[FilingOverride],
    agno_verify_state: CheckState = CheckState.FAIL,
    agno_verify_reason: str = "Agno verify has not run.",
) -> FilingReadiness:
    computed = [
        FilingCheck(
            check_id="package",
            label="Approved LegalSourcePackage imported",
            state=CheckState.PASS if package_imported else CheckState.FAIL,
            reason="No accepted evidence package." if not package_imported else "Package present.",
        ),
        FilingCheck(
            check_id="agno-verify",
            label="Agno POST /v1/verify/{sha256} on package hashes",
            state=agno_verify_state,
            reason=agno_verify_reason,
        ),
        FilingCheck(
            check_id="docket",
            label="Caption / case number clerk-confirmed",
            state=CheckState.PASS if docket_number else CheckState.FAIL,
            reason="Docket number is blank until clerk-confirmed." if not docket_number else "Docket recorded.",
        ),
        FilingCheck(
            check_id="judge",
            label="Assigned judge / referee clerk-confirmed",
            state=CheckState.PASS if judge else CheckState.FAIL,
            reason="Judge/referee not confirmed." if not judge else "Judge recorded.",
        ),
        FilingCheck(
            check_id="review",
            label="Your review approved a draft",
            state=CheckState.PASS if review_approved else CheckState.FAIL,
            reason="No owner approval on the current draft hash." if not review_approved else "Owner approval recorded.",
        ),
        FilingCheck(
            check_id="release",
            label="Final review copy + manifest exist",
            state=CheckState.PASS if release_count else CheckState.FAIL,
            reason="Build a final-copy candidate first." if not release_count else f"{release_count} candidate(s).",
        ),
        FilingCheck(
            check_id="not-filed",
            label="System has not marked anything filed",
            state=CheckState.FAIL if last_release_filed else CheckState.PASS,
            reason="A candidate is marked filed — investigate." if last_release_filed else "Nothing is marked filed.",
        ),
        FilingCheck(
            check_id="signature",
            label="Signature / notary block current",
            state=CheckState.UNKNOWN,
            reason="Owner must verify the current form.",
            human=True,
        ),
        FilingCheck(
            check_id="form-revision",
            label="Current SCAO / local form revision",
            state=CheckState.UNKNOWN,
            reason="Link the official PDF; do not ship a competing form.",
            human=True,
        ),
        FilingCheck(
            check_id="redactions",
            label="Redactions / confidential handling",
            state=CheckState.UNKNOWN,
            reason="Your review of minors' identifiers and nonpublic content.",
            human=True,
        ),
        FilingCheck(
            check_id="service",
            label="Service and proof of service actually completed",
            state=CheckState.UNKNOWN,
            reason="Do not backfill a service date.",
            human=True,
        ),
        FilingCheck(
            check_id="fee",
            label="Fee / waiver",
            state=CheckState.UNKNOWN,
            reason="Clerk practice; owner confirms.",
            human=True,
        ),
    ]
    latest = {}
    for item in overrides:
        latest[item.check_id] = item
    checks: list[FilingCheck] = []
    for check in computed:
        override = latest.get(check.check_id)
        if override is not None:
            checks.append(
                check.model_copy(
                    update={
                        "state": override.state,
                        "reason": f"{check.reason} Owner: {override.note}",
                    }
                )
            )
        else:
            checks.append(check)
    blocking = [row for row in checks if row.blocking and row.state is not CheckState.PASS]
    return FilingReadiness(
        ready=not blocking,
        filed=False,
        checks=checks,
        blocking_count=len(blocking),
    )
