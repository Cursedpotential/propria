"""Run source-context read-back behind the Review Actions panel.

Byline: Claude Code · Opus 5.5 · 2026-09-25.
"""

from __future__ import annotations

import asyncio

import httpx
import pytest
from app.service import source_context
from app.service.proffer_errors import ProfferError

HANDLE = "run_source_context_handle_abcdefghijklmnop"
SHA = "c" * 64


def _engine_payload(**overrides):
    payload = {
        "preview_handle": HANDLE,
        "request_id": "request-1",
        "source_ref": "b2://salem-data/consignatio/vault/v1/sms.xml",
        "parser_options_ref": "pending-handler-selection/v1",
        "registration": {
            "source_version_ref": "44444444-4444-4444-4444-444444444444",
            "declared_format": "xml",
            "source_context_ref": None,
            "original_filename": None,
            "original_sha256": SHA,
            "original_bytes": 4096,
        },
        "current": {
            "source_context_ref": "55555555-5555-4555-8555-555555555555",
            "revision": 2,
            "observed_source": {
                "key": "consignatio/vault/v1/sms.xml",
                "name": "sms.xml",
                "byte_length": 4096,
                "etag": f"sha256:{SHA}",
                "preview_sha256": SHA,
                "verification_state": "preview_only",
            },
            "assertions": {
                "source_class": "first_party",
                "context": "Phone backup",
                "acquired_at": "2026-01-10T02:13:38Z",
            },
            "change_reason": "Corrected context",
            "actor_username": "operator",
            "receipt_ref": "proffer-source-context://55555555-5555-4555-8555-555555555555",
            "recorded_at": "2026-09-25T00:00:00Z",
        },
    }
    payload.update(overrides)
    return payload


def _wire(monkeypatch: pytest.MonkeyPatch, payload: dict, calls: list) -> None:
    async def fake_require_mode(handle, mode):
        calls.append(("mode", handle, mode))

    async def fake_request(method, path, **kwargs):
        calls.append((method, path))
        return httpx.Response(200, json=payload)

    monkeypatch.setattr(source_context, "_require_mode", fake_require_mode)
    monkeypatch.setattr(source_context, "_request", fake_request)


def test_mode_is_proven_before_the_newest_revision_is_read(monkeypatch) -> None:
    calls: list = []
    _wire(monkeypatch, _engine_payload(), calls)

    result = asyncio.run(source_context.run_source_context(HANDLE, mode="TEST"))

    assert calls == [("mode", HANDLE, "TEST"), ("GET", f"/reference-import/previews/{HANDLE}/source-context")]
    assert result.matter_mode == "TEST"
    assert result.current is not None and result.current.revision == 2
    assert result.current.assertions.context == "Phone backup"
    assert result.current.observed_source.preview_sha256 == SHA
    assert result.registration is not None and result.registration.declared_format == "xml"


def test_a_run_without_operator_context_is_an_ordinary_answer(monkeypatch) -> None:
    _wire(monkeypatch, _engine_payload(current=None, registration=None), [])

    result = asyncio.run(source_context.run_source_context(HANDLE, mode="REAL"))

    assert result.current is None
    assert result.registration is None
    assert result.matter_mode == "REAL"


def test_a_crossed_handle_or_mode_fails_closed(monkeypatch) -> None:
    _wire(monkeypatch, _engine_payload(preview_handle="another_handle_abcdefghijklmnopqrstuv"), [])
    with pytest.raises(ProfferError) as crossed:
        asyncio.run(source_context.run_source_context(HANDLE, mode="TEST"))
    assert crossed.value.status_code == 502

    _wire(monkeypatch, _engine_payload(matter_mode="REAL"), [])
    with pytest.raises(ProfferError) as other_mode:
        asyncio.run(source_context.run_source_context(HANDLE, mode="TEST"))
    assert other_mode.value.status_code == 502
