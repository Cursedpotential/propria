"""Retained source/content/actor/event regression proofs under canonical operating policy.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Ported from test_proffer_mode_isolation.py; alternate-case expectations are superseded.
"""

from __future__ import annotations

import asyncio
import json
from datetime import UTC, datetime

import httpx
import pytest
from app.config import settings
from app.runtime import case_management, operating_mode
from app.service import proffer, source_context
from app.service.matter_mode import _clear_preview_modes_for_tests
from app.types.proffer import (
    ProfferDecisionActor,
    ProfferHandlerSelectionDecisionRequest,
    ProfferPreviewResponse,
)
from app.types.source_context import SourceContextCreateRequest
from fastapi import FastAPI
from fastapi.testclient import TestClient

TEST_MATTER_ID = "11111111-1111-4111-8111-111111111111"
TEST_COURT_CASE_ID = "22222222-2222-4222-8222-222222222222"
PREVIEW_HANDLE = "preview_handle_abcdefghijklmnopqrstuvwxyz"


@pytest.fixture(autouse=True)
def canonical_scope(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", TEST_MATTER_ID)
    monkeypatch.setattr(settings, "proffer_court_case_id", TEST_COURT_CASE_ID)
    async def verified_scope(_mode):
        return None
    monkeypatch.setattr(operating_mode, "verify_case_scope", verified_scope)
    _clear_preview_modes_for_tests()
    yield
    _clear_preview_modes_for_tests()


def test_generic_content_projection_preserves_exact_chunks_and_mode(monkeypatch) -> None:
    from app.service.matter_mode import bind_preview_mode

    bind_preview_mode(PREVIEW_HANDLE, "LIVE")

    async def fake_request(method, path, **kwargs):
        assert method == "GET"
        assert path.endswith("/content")
        assert kwargs["params"] == {"limit": 25, "record_cursor": "records-next"}
        return httpx.Response(
            200,
            json={
                "preview_handle": PREVIEW_HANDLE,
                "package": {
                    "source_version_ref": "source-version-1",
                    "declared_format": "document",
                    "status": "retained",
                    "metadata_count": 2,
                    "attachment_count": 0,
                },
                "attempt": {
                    "attempt_ref": "",
                    "projection_ref": "normalized-1",
                    "source_version_ref": "source-version-1",
                    "raw_generation_ref": "raw-1",
                    "normalized_generation_ref": "normalized-1",
                    "receipts": [],
                },
                "attempts_complete": False,
                "attempts_reason": "complete history is unavailable",
                "records": [
                    {
                        "record_id": "record-1",
                        "ordinal": 0,
                        "record_type": "document",
                        "payload": {"title": "Exact record"},
                        "source_locator_ref": "context.normalized_record_identity/record-1",
                    }
                ],
                "attachments": [],
                "chunk_generation": {
                    "generation_ref": "generation-1",
                    "generation_ordinal": 1,
                    "status": "sealed",
                    "policy_id": "document",
                    "policy_version": "1",
                    "chunker_id": "offsets",
                    "chunker_version": "1",
                    "schema_version": "1",
                    "source_view": "original",
                    "source_sha256": "a" * 64,
                    "receipt_ref": "receipt-1",
                },
                "chunks": [
                    {
                        "chunk_ref": "chunk-1",
                        "index": 0,
                        "content": "Exact chunk",
                        "sha256": "b" * 64,
                        "derivation_mode": "verbatim_span",
                        "locator_ref": "locator-1",
                        "byte_start": 0,
                        "byte_end": 11,
                    }
                ],
            },
        )

    monkeypatch.setattr(proffer, "_request", fake_request)
    result = asyncio.run(
        proffer.preview_content(PREVIEW_HANDLE, mode="LIVE", record_cursor="records-next", chunk_cursor=None, limit=25)
    )

    assert result.matter_mode == "LIVE"
    assert result.records[0].payload == {"title": "Exact record"}
    assert result.chunks[0].content == "Exact chunk"
    with pytest.raises(proffer.ProfferError, match="different matter mode"):
        asyncio.run(
            proffer.preview_content(PREVIEW_HANDLE, mode="DEV", record_cursor=None, chunk_cursor=None, limit=25)
        )


def test_handler_choice_is_flat_actor_bound_and_mode_correlated(monkeypatch) -> None:
    calls: list[tuple[str, str, dict]] = []

    async def fake_request(method, path, **kwargs):
        calls.append((method, path, kwargs))
        if method == "GET":
            return httpx.Response(
                200,
                json={
                    "preview_handle": PREVIEW_HANDLE,
                    "phase": "awaiting_handler_selection",
                    "handler_recommendation_ref": "handler-recommendation://123",
                    "detected_format": "callsbackuprestore_xml",
                    "detected_format_ref": "detected-format://123",
                    "signature_ref": "signature://calls-root-v1",
                    "recommended_handler": {
                        "handler_id": "duckdb.calls",
                        "handler_version": "1.0.0",
                        "execution_path": "duckdb",
                        "compatibility_ref": "compatibility://123",
                        "reason": "The root element is calls.",
                    },
                    "alternative_handlers": [],
                },
            )
        return httpx.Response(
            200,
            json={
                "preview_handle": PREVIEW_HANDLE,
                "decision_ref": "handler-decision://123",
                "status": "persisted",
            },
        )

    monkeypatch.setattr(proffer, "_request", fake_request)
    from app.service.matter_mode import bind_preview_mode

    bind_preview_mode(PREVIEW_HANDLE, "LIVE")
    choice = ProfferHandlerSelectionDecisionRequest(
        recommendation_ref="handler-recommendation://123",
        handler_id="duckdb.calls",
        handler_version="1.0.0",
        execution_path="duckdb",
        compatibility_ref="compatibility://123",
    )
    actor = ProfferDecisionActor(subject_uid="subject-1", username="operator")

    result = asyncio.run(proffer.decide_handler_selection(PREVIEW_HANDLE, choice, actor, mode="LIVE"))

    assert result.matter_mode == "LIVE"
    assert calls[1][2]["json"] == {"compatibility_ref": "compatibility://123"}
    assert calls[1][2]["headers"]["X-authentik-uid"] == "subject-1"
    assert calls[1][2]["headers"]["X-authentik-username"] == "operator"
    assert calls[1][2]["headers"]["Idempotency-Key"].startswith("proffer-handler-selection:")
    assert result.decision_ref == "handler-decision://123"


def test_handler_choice_outside_current_recommendation_never_posts(monkeypatch) -> None:
    calls: list[str] = []

    async def fake_request(method, path, **kwargs):
        calls.append(method)
        return httpx.Response(
            200,
            json={
                "preview_handle": PREVIEW_HANDLE,
                "phase": "awaiting_handler_selection",
                "handler_recommendation_ref": "handler-recommendation://123",
                "detected_format": "callsbackuprestore_xml",
                "detected_format_ref": "detected-format://123",
                "signature_ref": "signature://calls-root-v1",
                "recommended_handler": {
                    "handler_id": "duckdb.calls",
                    "handler_version": "1.0.0",
                    "execution_path": "duckdb",
                    "compatibility_ref": "compatibility://123",
                    "reason": "The root element is calls.",
                },
                "alternative_handlers": [],
            },
        )

    monkeypatch.setattr(proffer, "_request", fake_request)
    from app.service.matter_mode import bind_preview_mode

    bind_preview_mode(PREVIEW_HANDLE, "LIVE")
    actor = ProfferDecisionActor(subject_uid="subject-1", username="operator")
    incompatible = ProfferHandlerSelectionDecisionRequest(
        recommendation_ref="handler-recommendation://123",
        handler_id="decoder.sms",
        handler_version="1.0.0",
        execution_path="decoder",
        compatibility_ref="compatibility://not-offered",
    )

    with pytest.raises(proffer.ProfferError) as denied:
        asyncio.run(proffer.decide_handler_selection(PREVIEW_HANDLE, incompatible, actor, mode="LIVE"))

    assert denied.value.status_code == 409
    assert calls == ["GET"]


def test_source_context_mode_is_validated_and_forwarded_explicitly_upstream(monkeypatch) -> None:
    class Response:
        def json(self):
            return {
                "source_context_ref": "33333333-3333-4333-8333-333333333333",
                "receipt_ref": "source-context://33333333-3333-4333-8333-333333333333",
                "content_digest": "a" * 64,
                "revision": 1,
                "recorded_at": "2026-09-12T20:00:00Z",
            }

    captured: dict = {}

    async def fake_request(method, path, **kwargs):
        captured.update(method=method, path=path, kwargs=kwargs)
        return Response()

    monkeypatch.setattr(source_context, "_request", fake_request)
    body = SourceContextCreateRequest.model_validate(
        {
            "request_id": "request-1",
            "matter_id": TEST_MATTER_ID,
            "court_case_id": TEST_COURT_CASE_ID,
            "source_ref": "b2://salem-data/consignatio/casevault/source.xml",
            "observed_source": {
                "key": "source.xml",
                "name": "source.xml",
                "byte_length": 10,
                "etag": '"etag"',
                "preview_sha256": "b" * 64,
            },
            "assertions": {"source_class": "unknown"},
            "change_reason": "Initial operator context",
            "matter_mode": "LIVE",
        }
    )
    actor = ProfferDecisionActor(subject_uid="subject-1", username="operator")

    receipt = asyncio.run(source_context.create_source_context(body, actor, mode="LIVE"))

    assert receipt.matter_mode == "LIVE"
    assert "matter_mode" not in captured["kwargs"]["json"]
    assert captured["kwargs"]["json"]["operating_mode"] == "LIVE"
    assert captured["kwargs"]["json"]["matter_id"] == TEST_MATTER_ID
    headers = captured["kwargs"]["headers"]
    assert headers["X-authentik-uid"] == actor.subject_uid
    assert headers["X-authentik-username"] == actor.username
    key = headers["Idempotency-Key"]
    assert key.startswith("proffer-source-context:")
    asyncio.run(source_context.create_source_context(body, actor, mode="LIVE"))
    assert captured["kwargs"]["headers"]["Idempotency-Key"] == key
    before = dict(captured)
    with pytest.raises(proffer.ProfferError) as denied:
        asyncio.run(source_context.create_source_context(body.model_copy(update={"matter_mode": "DEV"}), actor, mode="DEV"))
    assert denied.value.status_code == 409
    assert captured == before
    for changed in ({"matter_mode": "DEV"}, {"matter_id": "99999999-9999-4999-8999-999999999999"},
                    {"court_case_id": "99999999-9999-4999-8999-999999999999"}):
        with pytest.raises(proffer.ProfferError) as mismatched:
            asyncio.run(source_context.create_source_context(body.model_copy(update=changed), actor, mode="LIVE"))
        assert mismatched.value.status_code == 409
        assert captured == before


def test_preview_exposes_complete_content_backed_recommendation() -> None:
    preview = ProfferPreviewResponse.model_validate(
        {
            "preview_handle": PREVIEW_HANDLE,
            "matter_mode": "LIVE",
            "phase": "awaiting_handler_selection",
            "handler_recommendation_ref": "recommendation://123",
            "detected_format": "callsbackuprestore_xml",
            "detected_format_ref": "detected-format://123",
            "signature_ref": "signature://calls-root-v1",
            "recommended_handler": {
                "handler_id": "duckdb.calls",
                "handler_version": "1.0.0",
                "execution_path": "duckdb",
                "compatibility_ref": "compatibility://123",
                "reason": "The root element is calls.",
            },
            "alternative_handlers": [],
        }
    )

    assert preview.detected_format == "callsbackuprestore_xml"
    assert preview.recommended_handler is not None
    assert preview.recommended_handler.execution_path == "duckdb"


def test_preview_event_is_re_emitted_with_mode() -> None:
    raw = {
        "event_id": 1,
        "event_type": "phase_changed",
        "occurred_at": "2026-09-12T20:00:00Z",
        "preview_handle": PREVIEW_HANDLE,
        "phase": "parser_execution",
    }
    response = httpx.Response(
        200,
        content=f"id: 1\ndata: {json.dumps(raw)}\n\n".encode(),
        headers={"content-type": "text/event-stream"},
    )

    async def collect() -> list[str]:
        return [
            item
            async for item in proffer.validated_preview_events(
                response,
                preview_handle=PREVIEW_HANDLE,
                mode="LIVE",
                last_event_id=None,
            )
        ]

    emitted = asyncio.run(collect())
    assert '"matter_mode":"LIVE"' in emitted[0]


def test_matters_route_fetches_only_exact_configured_id_and_echoes_mode(monkeypatch) -> None:
    calls: list[str] = []
    now = datetime(2026, 9, 12, tzinfo=UTC).isoformat()

    def fake_get(matter_id):
        calls.append(str(matter_id))
        return {
            "id": TEST_MATTER_ID,
            "title": "Configured DEV matter",
            "description": None,
            "status": "active",
            "partition_keys": ["test"],
            "court_cases": [],
            "created_at": now,
            "updated_at": now,
        }

    monkeypatch.setattr(case_management.service, "get_matter", fake_get)
    monkeypatch.setattr(
        case_management.service,
        "list_matters",
        lambda **_: (_ for _ in ()).throw(AssertionError("title/list inference is forbidden")),
    )
    app = FastAPI()
    app.include_router(case_management.router)

    response = TestClient(app).get("/api/matters?mode=LIVE")

    assert response.status_code == 200
    assert calls == [TEST_MATTER_ID]
    assert response.json()["matter_mode"] == "LIVE"
    assert response.json()["data"][0]["matter_mode"] == "LIVE"
