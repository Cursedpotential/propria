"""Exercise discovery scope, actor provenance, receipts and cancellation without live services."""

import asyncio
from uuid import UUID

import httpx
import pytest
from app.config import settings
from app.runtime import discovery, operating_mode
from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.discovery import FeedbackRequest
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

MATTER = "11111111-1111-4111-8111-111111111111"
COURT = "22222222-2222-4222-8222-222222222222"
RUN = "33333333-3333-4333-8333-333333333333"
CANDIDATE = "44444444-4444-4444-8444-444444444444"
FEEDBACK = "55555555-5555-4555-8555-555555555555"
GENERATION = "66666666-6666-4666-8666-666666666666"
SCOPE = {"mode": "LIVE", "matter_id": MATTER, "court_case_id": COURT}
ACTOR = {"subject_uid": "session-user", "username": "owner"}
QUERY = f"mode=LIVE&matter_id={MATTER}&court_case_id={COURT}"


def run_payload(scope=None):
    return {
        "request": {"run_id": RUN, "scope": scope or SCOPE, "actor": ACTOR},
        "revision": 2,
        "dispatch_pending": True,
    }


def request_body():
    return {
        "run_id": RUN,
        "scope": SCOPE,
        "input_backend": "postgresql",
        "dataset_id": "recorded",
        "selection_revision": "selection-1",
        "generation_ids": [GENERATION],
        "hypotheses": [{"hypothesis_id": "description-1", "text": "Pressure about parenting time"}],
        "max_records": 200,
        "max_candidates": 25,
    }


@pytest.fixture
def client(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", MATTER)
    monkeypatch.setattr(settings, "proffer_court_case_id", COURT)

    async def verify(mode):
        assert mode in {"LIVE", "DEV"}

    monkeypatch.setattr(operating_mode, "verify_case_scope", verify)
    app = FastAPI()

    @app.middleware("http")
    async def auth(request: Request, call_next):
        request.state.subject_uid = ACTOR["subject_uid"]
        request.state.principal = ACTOR["username"]
        return await call_next(request)

    app.include_router(discovery.router)
    return TestClient(app)


def wire(monkeypatch, replies):
    calls = []

    async def request(method, path, **kwargs):
        calls.append((method, path, kwargs))
        return httpx.Response(200, json=replies.pop(0))

    monkeypatch.setattr(proffer, "_request", request)
    return calls


def test_actor_is_session_derived_and_body_cannot_spoof(client, monkeypatch):
    calls = wire(monkeypatch, [run_payload()])
    response = client.post(
        "/api/discovery/runs?mode=LIVE",
        json=request_body(),
        headers={"Idempotency-Key": RUN, "X-authentik-uid": "spoofed", "X-authentik-username": "spoofed"},
    )
    assert response.status_code == 202
    assert calls[0][2]["headers"] == {
        "X-authentik-uid": ACTOR["subject_uid"],
        "X-authentik-username": "owner",
        "Idempotency-Key": RUN,
    }
    assert "actor" not in calls[0][2]["json"]
    assert response.json()["dispatch_pending"] is True
    spoof = request_body() | {"actor": {"subject_uid": "spoofed", "username": "spoofed"}}
    assert client.post("/api/discovery/runs?mode=LIVE", json=spoof, headers={"Idempotency-Key": RUN}).status_code == 422
    assert len(calls) == 1


def test_dev_and_mode_mismatch_never_dispatch(client, monkeypatch):
    calls = wire(monkeypatch, [])
    body = request_body()
    body["scope"] = {**SCOPE, "mode": "DEV"}
    response = client.post("/api/discovery/runs?mode=DEV", json=body, headers={"Idempotency-Key": RUN})
    assert response.status_code == 409
    assert "isolated" in response.json()["detail"]
    assert client.post("/api/discovery/runs?mode=LIVE", json=body, headers={"Idempotency-Key": RUN}).status_code == 409
    assert calls == []


def test_wrong_requested_or_returned_case_fails_closed(client, monkeypatch):
    calls = wire(monkeypatch, [{"items": [{"scope": {**SCOPE, "court_case_id": CANDIDATE}}]}])
    assert client.get(f"/api/discovery/datasets?{QUERY}").status_code == 502
    assert (
        client.get(f"/api/discovery/datasets?mode=LIVE&matter_id={CANDIDATE}&court_case_id={COURT}").status_code == 409
    )
    assert len(calls) == 1
    calls = wire(monkeypatch, [run_payload({**SCOPE, "mode": "DEV"})])
    assert client.get(f"/api/discovery/runs/{RUN}?{QUERY}").status_code == 502


def test_feedback_returns_only_matching_saved_receipt(client, monkeypatch):
    body = {
        "feedback_id": FEEDBACK,
        "candidate_id": CANDIDATE,
        "expected_revision": 2,
        "kind": "reported_speech",
        "labels": [],
        "attributed_actor_ids": [],
        "note": "Quotation of somebody else",
    }
    saved = {**body, "revision": 3, "actor": ACTOR, "created_at": "2026-10-11T01:00:00Z"}
    calls = wire(monkeypatch, [run_payload(), {"feedback": saved, "run": run_payload()}])
    response = client.post(
        f"/api/discovery/runs/{RUN}/feedback?{QUERY}", json=body, headers={"Idempotency-Key": FEEDBACK}
    )
    assert response.status_code == 200
    assert response.json()["feedback"]["kind"] == "reported_speech"
    assert calls[1][2]["json"]["expected_revision"] == 2
    assert calls[1][2]["headers"]["Idempotency-Key"] == FEEDBACK
    wire(monkeypatch, [run_payload(), {"feedback": {**saved, "candidate_id": RUN}, "run": run_payload()}])
    assert (
        client.post(
            f"/api/discovery/runs/{RUN}/feedback?{QUERY}", json=body, headers={"Idempotency-Key": FEEDBACK}
        ).status_code
        == 502
    )


def test_context_preserves_native_citations_without_missing_hash_gate(client, monkeypatch):
    record = {
        "source_ref": "b2://registered/source",
        "locators": [
            {
                "coordinate_system": "unicode_codepoints",
                "range_start": 0,
                "range_end": 3,
                "projection": {"json_pointer": "/messages/2"},
            }
        ],
    }
    wire(monkeypatch, [run_payload(), {"candidate": {"candidate_id": CANDIDATE}, "records": [record], "revision": 2}])
    response = client.get(f"/api/discovery/runs/{RUN}/context/{CANDIDATE}?{QUERY}")
    assert response.status_code == 200
    assert response.json()["records"] == [record]


def test_revision_conflict_is_not_reported_as_saved(client, monkeypatch):
    async def rejected(method, path, **kwargs):
        if method == "GET":
            return httpx.Response(200, json=run_payload())
        raise ProfferError("Discovery revision changed; refresh", 409)

    monkeypatch.setattr(proffer, "_request", rejected)
    response = client.post(
        f"/api/discovery/runs/{RUN}/feedback?{QUERY}",
        json={"feedback_id": FEEDBACK, "candidate_id": CANDIDATE, "expected_revision": 1, "kind": "wrong_person"},
        headers={"Idempotency-Key": FEEDBACK},
    )
    assert response.status_code == 409
    assert "feedback" not in response.json()


def test_disconnected_scope_cancels_upstream_work():
    cancelled = []

    class Disconnected:
        async def is_disconnected(self):
            await asyncio.sleep(0)
            return True

    async def operation():
        try:
            await asyncio.Event().wait()
        finally:
            cancelled.append(True)

    async def check():
        task = asyncio.create_task(operation())
        await asyncio.sleep(0)

        async def awaiting():
            return await task

        with pytest.raises(Exception) as caught:
            await discovery._connected(Disconnected(), awaiting())
        assert caught.value.status_code == 499
        assert task.cancelled()

    asyncio.run(check())
    assert cancelled == [True]


def test_feedback_key_actor_and_scope_are_required(client, monkeypatch):
    wire(monkeypatch, [])
    assert (
        client.post(f"/api/discovery/runs/{RUN}/feedback?{QUERY}", json={"candidate_id": CANDIDATE}).status_code == 422
    )
    with pytest.raises(ValueError):
        FeedbackRequest(
            feedback_id=UUID(FEEDBACK), candidate_id=CANDIDATE, expected_revision=2, kind="free_text", note=""
        )


def test_mismatched_retry_keys_fail_before_engine_dispatch(client, monkeypatch):
    calls = wire(monkeypatch, [])
    assert (
        client.post(
            "/api/discovery/runs?mode=LIVE", json=request_body(), headers={"Idempotency-Key": FEEDBACK}
        ).status_code
        == 409
    )
    assert (
        client.post(
            f"/api/discovery/runs/{RUN}/feedback?{QUERY}",
            json={"feedback_id": FEEDBACK, "candidate_id": CANDIDATE, "expected_revision": 2, "kind": "skip"},
            headers={"Idempotency-Key": RUN},
        ).status_code
        == 409
    )
    assert calls == []
