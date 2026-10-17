"""Validate bounded Workbench knowledge-review projections independently of live backend.

Inputs: synthetic review receipts. Outputs: deterministic assertions. Effects: none.
Use these tests before wiring the canonical PostgreSQL/Temporal review queue.
"""
from __future__ import annotations

import pytest
from fastapi import HTTPException

from app.runtime import knowledge_review


def test_valid_queue_preserves_authoritative_ref_and_relative_review_link():
    result = knowledge_review._compact({
        "schema": "propria-knowledge-review-queue/v1",
        "total": 1,
        "items": [{
            "review_ref": "review-abc",
            "severity": "critical",
            "status": "queued",
            "summary": "Supersession conflicts with a newer decision.",
            "source_count": 2,
            "review_href": "/activity?review=review-abc",
        }],
    })
    assert result["status"] == "ready"
    assert result["items"][0]["review_ref"] == "review-abc"
    assert result["items"][0]["review_href"] == "/activity?review=review-abc"


@pytest.mark.parametrize("value", [
    "https://wrong.example/review", "//external.example/review",
    "javascript:alert(1)", "/not-authoritative/review",
    "/activity\\evil", "/review#fragment"
])
def test_review_links_must_stay_in_the_authenticated_workbench(value):
    assert knowledge_review._relative_review_href(value) is None


def test_queue_rejects_noncanonical_receipts():
    with pytest.raises(HTTPException) as ex:
        knowledge_review._compact({"schema": "wrong", "items": []})
    assert ex.value.status_code == 502


def test_queue_rejects_missing_persisted_reference():
    with pytest.raises(HTTPException):
        knowledge_review._compact({
            "schema": "propria-knowledge-review-queue/v1",
            "items": [{"severity": "critical", "status": "queued"}],
        })


def test_queue_does_not_misstate_missing_backend_as_empty(monkeypatch):
    monkeypatch.delenv("KNOWLEDGE_REVIEW_QUEUE_URL", raising=False)
    with pytest.raises(HTTPException) as ex:
        knowledge_review._queue_endpoint()
    assert ex.value.status_code == 503
