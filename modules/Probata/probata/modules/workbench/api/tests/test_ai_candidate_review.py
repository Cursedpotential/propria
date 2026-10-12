"""Verify native review preserves exact source pins and independent decision receipts.

Inputs: in-memory Go responses. Outputs: routing/provenance assertions. Effects:
no network or live writes; choose before connecting real staged candidates.
"""
import asyncio

import pytest
from fastapi import HTTPException
from starlette.requests import Request

from app.runtime import ai_candidate_review as review
from app.service.proffer_errors import ProfferError

SOURCE = {
    "source_version_id": "11111111-1111-4111-8111-111111111111",
    "source_ref": "b2://salem-data/consignatio/casevault/export.json",
    "version_id": "original-provider-version", "source_object_id": "", "source_sha256": "a" * 64,
    "prepared_ref": "b2://salem-data/consignatio/casevault/context/prepared.json",
}
ITEM = "22222222-2222-4222-8222-222222222222"


def actor():
    """Reuse middleware identity in a request without new authentication machinery."""
    request = Request({"type": "http", "method": "GET", "path": "/read"})
    request.state.subject_uid = "tailscale:owner"
    request.state.principal = "owner"
    return request


def fake_go(monkeypatch, payload):
    """Capture Go calls and return a selected bounded fake response."""
    calls = []

    async def call(method, path, **kwargs):
        """Record one request without a network or database side effect."""
        calls.append((method, path, kwargs))
        return object()

    monkeypatch.setattr(review.proffer, "_request", call)
    monkeypatch.setattr(review.proffer, "_json_payload", lambda *_: payload)
    return calls


def test_native_review_lists_exact_original_without_preview_or_normalized_id(monkeypatch):
    """Forward verified Stage pins and pagination without fabricating source identities."""
    payload = {"source_version_id": SOURCE["source_version_id"], "matter_mode": "LIVE", "next_cursor": "",
               "candidates": [{"candidate_id": ITEM, "content_sha256": "b" * 64, "candidate": dict(SOURCE)}]}
    calls = fake_go(monkeypatch, payload)
    result = asyncio.run(review.candidates({"review_source": SOURCE}, actor(), "previous-id"))
    assert result["source"] == SOURCE
    assert calls[0][0:2] == ("GET", "/reference-import/ai-candidates")
    assert calls[0][2]["params"] == {**SOURCE, "preview_handle": "", "matter_mode": "LIVE", "after_id": "previous-id"}
    assert calls[0][2]["headers"] == {"X-authentik-uid": "tailscale:owner", "X-authentik-username": "owner"}


def test_missing_stage_is_not_an_empty_success(monkeypatch):
    """Stop an unstaged read before any candidate query rather than imply completion."""
    calls = fake_go(monkeypatch, {})
    with pytest.raises(HTTPException) as error:
        asyncio.run(review.candidates({"status": "running"}, actor(), ""))
    assert error.value.status_code == 409
    assert not calls


def test_candidates_cannot_escape_exact_source_version(monkeypatch):
    """Reject a candidate from a different provider version even under the same source."""
    wrong = {**SOURCE, "version_id": "different-version"}
    fake_go(monkeypatch, {"source_version_id": SOURCE["source_version_id"], "matter_mode": "LIVE", "next_cursor": "",
                         "candidates": [{"candidate_id": ITEM, "content_sha256": "b" * 64, "candidate": wrong}]})
    with pytest.raises(ProfferError, match="exact original"):
        asyncio.run(review.candidates({"review_source": SOURCE}, actor(), ""))


@pytest.mark.parametrize("decision", ["approved", "rejected", "needs_info"])
def test_one_choice_preserves_digest_actor_retry_key_and_pending_graph_receipt(monkeypatch, decision):
    """Preserve a committed decision even when graph enqueue remains pending."""
    payload = {"candidate_id": ITEM, "decision": decision, "decision_id": "real-decision", "request_digest": "c" * 64,
               "matter_mode": "LIVE", "decision_committed": True,
               "projection": {"status": "pending" if decision == "approved" else "not_requested", "retryable": decision == "approved"}}
    calls = fake_go(monkeypatch, payload)
    body = review.CandidateDecision(candidate_id=ITEM, expected_content_sha256="b" * 64, decision=decision)
    result = asyncio.run(review.decide({"review_source": SOURCE}, body, actor(), "same-click-key"))
    assert result == payload
    assert calls[0][0:2] == ("POST", "/reference-import/ai-candidates/decision")
    assert calls[0][2]["json"] == {**SOURCE, **body.model_dump(mode="json"), "preview_handle": "", "matter_mode": "LIVE"}
    assert calls[0][2]["headers"]["Idempotency-Key"] == "same-click-key"
    assert "actor" not in calls[0][2]["json"]


def test_mismatched_decision_receipt_is_not_success(monkeypatch):
    """Reject a receipt for a different item instead of marking this choice saved."""
    fake_go(monkeypatch, {"candidate_id": "other", "decision": "approved", "decision_id": "receipt",
                         "request_digest": "c" * 64, "matter_mode": "LIVE"})
    body = review.CandidateDecision(candidate_id=ITEM, expected_content_sha256="b" * 64, decision="approved")
    with pytest.raises(ProfferError, match="receipt"):
        asyncio.run(review.decide({"review_source": SOURCE}, body, actor(), "click"))


def test_existing_context_router_exposes_read_and_explicit_decision(monkeypatch):
    """Exercise both BFF routes with server-resolved source pins and existing identity.

    Inputs: workflow URL and one explicit decision. Outputs: page/receipt. Effects:
    in-memory HTTP and fake Go calls only; no user approval is performed live.
    """
    from fastapi import FastAPI
    from fastapi.testclient import TestClient
    from app.runtime import context_sources

    async def status(_workflow_id, _request):
        """Return the workflow's verified Stage result, never a browser source pin."""
        return {"workflow_id": "context-source:one", "review_source": SOURCE}

    monkeypatch.setattr(context_sources, "ai_context_source_status", status)
    payload = {"source_version_id": SOURCE["source_version_id"], "matter_mode": "LIVE", "next_cursor": "",
               "candidates": [{"candidate_id": ITEM, "content_sha256": "b" * 64, "candidate": dict(SOURCE)}]}
    calls = fake_go(monkeypatch, payload)
    app = FastAPI()

    @app.middleware("http")
    async def identity(request, call_next):
        """Supply the existing personal-tailnet middleware identity in the test."""
        request.state.subject_uid = "tailscale:owner"
        request.state.principal = "owner"
        return await call_next(request)

    app.include_router(context_sources.router)
    client = TestClient(app)
    path = "/api/context/sources/workflows/context-source:one/candidates"
    assert client.get(path).status_code == 200
    payload.clear()
    payload.update({"candidate_id": ITEM, "decision": "approved", "decision_id": "real-receipt",
                    "request_digest": "c" * 64, "matter_mode": "LIVE", "decision_committed": True,
                    "projection": {"status": "pending", "retryable": True}})
    response = client.post(path + "/decision", headers={"Idempotency-Key": "one-click"}, json={
        "candidate_id": ITEM, "expected_content_sha256": "b" * 64, "decision": "approved",
        "source_ref": "b2://untrusted/browser.json", "actor": "fabricated",
    })
    assert response.status_code == 200
    assert response.json()["projection"]["status"] == "pending"
    assert calls[1][2]["json"]["source_ref"] == SOURCE["source_ref"]
    assert "actor" not in calls[1][2]["json"]


def test_native_projection_keeps_empty_object_identity_with_real_original_pin():
    """Accept the catalog's honest native variant without inventing a retained object."""
    from app.types.analysis import ProjectionSourcePin
    from pydantic import ValidationError

    pin = {"source_id": ITEM, "source_version_id": SOURCE["source_version_id"], "source_object_id": "",
           "source_object_sha256": SOURCE["source_sha256"], "source_object_uri": SOURCE["source_ref"]}
    assert ProjectionSourcePin.model_validate(pin).source_object_id == ""
    for invalid in ({**pin, "source_id": "invented"}, {**pin, "source_object_uri": "https://unverified"}):
        with pytest.raises(ValidationError):
            ProjectionSourcePin.model_validate(invalid)


def test_native_claim_requires_full_native_locator_instead_of_a_fictional_object():
    """Preserve Unicode start zero and require source/hash/span when object ID is empty."""
    from app.types.analysis import AnalysisClaim
    from pydantic import ValidationError

    claim = dict(id="claim", kind="statement", text="Original account", matter_id="matter", court_case_id="case",
        approved_revision_id="decision", approval_digest="a" * 64, control_generation_id="generation",
        candidate_id=ITEM, candidate_sha256="b" * 64, source_id=ITEM, source_version_id=SOURCE["source_version_id"],
        source_object_id="", source_object_uri=SOURCE["source_ref"], source_sha256=SOURCE["source_sha256"],
        record_id="native-record", record_sha256="c" * 64, native_span_start=0, native_span_end=5,
        native_span_unit="unicode_codepoint", native_span_sha256="d" * 64,
        approved_at="2026-10-08T12:00:00Z", approved_by="tailscale:owner")
    assert AnalysisClaim.model_validate(claim).native_span_start == 0
    for key in ("native_span_start", "native_span_end", "native_span_unit", "native_span_sha256"):
        with pytest.raises(ValidationError):
            AnalysisClaim.model_validate({**claim, key: None})
