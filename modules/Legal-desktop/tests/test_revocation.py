"""Territory tests for evidence-event staleness.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

import pytest

from legal_workspace.contracts.events import EventEnvelope
from legal_workspace.domain.work_product import WorkProductState, WorkProductVersion
from legal_workspace.services.revocation import apply_evidence_event


def _event(event_type: str, payload: dict) -> EventEnvelope:
    return EventEnvelope(
        event_id=uuid4(),
        event_type=event_type,
        occurred_at=datetime.now(UTC),
        matter_id=uuid4(),
        aggregate_id=uuid4(),
        aggregate_version=1,
        trace_id="t-1",
        payload_hash="sha256:evt",
        payload=payload,
    )


def test_revocation_marks_dependent_released_product_stale() -> None:
    package_id = uuid4()
    released = WorkProductVersion(
        work_product_id=uuid4(),
        state=WorkProductState.RELEASED,
        source_package_id=package_id,
        content_hash="sha256:doc",
    )
    unrelated = WorkProductVersion(
        work_product_id=uuid4(),
        state=WorkProductState.RELEASED,
        source_package_id=uuid4(),
        content_hash="sha256:other",
    )

    updated = apply_evidence_event(
        _event("evidence.release.revoked.v1", {"package_id": str(package_id)}),
        [released, unrelated],
    )

    by_id = {row.work_product_id: row for row in updated}
    assert by_id[released.work_product_id].state is WorkProductState.STALE
    assert by_id[unrelated.work_product_id].state is WorkProductState.RELEASED


def test_unknown_event_schema_fails_closed() -> None:
    product = WorkProductVersion(work_product_id=uuid4(), content_hash="x")
    with pytest.raises(ValueError, match="unknown event schema"):
        apply_evidence_event(_event("legal.made_up.v9", {}), [product])
