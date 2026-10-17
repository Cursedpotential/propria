"""Validate Workbench's canonical knowledge-review proxy, version and link safety.

Inputs: fake review snapshots and fake identity. Outputs: focused deterministic tests.
Effects: no network, original case data, secrets, or approvals.
"""
from __future__ import annotations

from types import SimpleNamespace
import pytest
from fastapi import HTTPException

from app.runtime import knowledge_review


def test_queue_preserves_canonical_source_identity_and_version():
    result = knowledge_review._compact({
        "schema": "propria-knowledge-review-queue/v1",
        "total": 1, "next_cursor": None,
        "items": [{"review_ref": "7f707e7a-32ed-4f99-9a6b-1d61ab9f8122",
                   "severity": "critical", "status": "queued",
                   "kind": "possible_topic_conflict", "scope": "propria",
                   "summary": "Conflicting memory versions", "source_count": 2,
                   "case_version": 3,
                   "review_href": "/activity?knowledge_review=7f707e7a-32ed-4f99-9a6b-1d61ab9f8122"}],
    })
    assert result["status"] == "ready"
    assert result["items"][0]["case_version"] == 3
    assert result["items"][0]["source_count"] == 2


@pytest.mark.parametrize("link", [
    "https://wrong.example/review", "//external.example/review",
    "javascript:alert(1)", "/not-authoritative/review",
    "/activity\\evil", "/review#fragment"
])
def test_links_are_only_internal_review_urls(link):
    assert knowledge_review._review_href(link) is None


def test_noncanonical_or_missing_persisted_reference_rejected():
    with pytest.raises(HTTPException) as error:
        knowledge_review._compact({"schema": "wrong", "items": [], "total": 0})
    assert error.value.status_code == 502
    with pytest.raises(HTTPException):
        knowledge_review._compact({"schema": "propria-knowledge-review-queue/v1",
                                   "total": 1,
                                   "items": [{"severity": "critical", "status": "queued"}]})


def test_no_identity_is_not_authenticated():
    with pytest.raises(HTTPException) as error:
        knowledge_review._reviewer(SimpleNamespace(state=SimpleNamespace()))
    assert error.value.status_code == 401
    assert knowledge_review._reviewer(
        SimpleNamespace(state=SimpleNamespace(subject_uid="authentik-owner"))
    ) == "authentik-owner"


@pytest.mark.asyncio
async def test_owner_verdict_not_represented_as_source_write(monkeypatch):
    from unittest.mock import AsyncMock
    from uuid import UUID
    from app.runtime.knowledge_review import DecisionRequest

    review_id = UUID("7f707e7a-32ed-4f99-9a6b-1d61ab9f8122")
    request = SimpleNamespace(state=SimpleNamespace(subject_uid="real-owner"))
    stub = AsyncMock(return_value={
        "review_ref": str(review_id), "decision": "approve_plan",
        "source_applied": False, "case_version": 2
    })
    monkeypatch.setattr(knowledge_review, "_request", stub)
    result = await knowledge_review.decide(
        request, review_id, DecisionRequest(expected_case_version=1,
            request_id="unique-owner-request-01234",
            decision="approve_plan", rationale="Versions verified")
    )
    assert result["source_applied"] is False
    assert stub.await_args.kwargs["owner"] is True
    assert stub.await_args.kwargs["body"]["reviewer_ref"] == "real-owner"
