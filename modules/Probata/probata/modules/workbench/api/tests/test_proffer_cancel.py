"""Cancel one Proffer run: actor from request state, reason required, engine route only.

Byline: Claude Code · Opus 5.5 · 2026-09-28 (D05-C06).
"""

from __future__ import annotations

import asyncio

import pytest
from fastapi import HTTPException
from pydantic import ValidationError
from starlette.requests import Request

from app.runtime import proffer_cancel as runtime
from app.service import proffer, proffer_cancel
from app.service.matter_mode import _clear_preview_modes_for_tests, bind_preview_mode
from app.types.proffer import ProfferDecisionActor
from app.types.proffer_cancel import ProfferCancelRequest

PREVIEW_HANDLE = "preview_handle_abcdefghijklmnopqrstuvwxyz"


@pytest.fixture(autouse=True)
def preview_mode_binding():
    _clear_preview_modes_for_tests()
    bind_preview_mode(PREVIEW_HANDLE, "TEST")
    yield
    _clear_preview_modes_for_tests()


def test_cancel_requires_a_reason_and_rejects_browser_actor_fields() -> None:
    with pytest.raises(ValidationError):
        ProfferCancelRequest(reason="   ")
    with pytest.raises(ValidationError):
        ProfferCancelRequest.model_validate({"reason": "x", "actor": "someone"})


def test_cancel_posts_to_the_engine_route_with_the_actor_in_trusted_headers(monkeypatch) -> None:
    class Response:
        def json(self):
            return {"preview_handle": PREVIEW_HANDLE, "status": "cancel_requested"}

    captured = {}

    async def fake_request(method, path, **kwargs):
        captured.update(method=method, path=path, kwargs=kwargs)
        return Response()

    monkeypatch.setattr(proffer, "_request", fake_request)
    actor = ProfferDecisionActor(subject_uid="tailscale:owner@example.com", username="owner@example.com")
    result = asyncio.run(
        proffer_cancel.cancel(PREVIEW_HANDLE, ProfferCancelRequest(reason="started by mistake"), actor, mode="TEST")
    )

    assert result.status == "cancel_requested" and result.matter_mode == "TEST"
    assert captured["method"] == "POST"
    assert captured["path"] == f"/reference-import/previews/{PREVIEW_HANDLE}/cancel"
    assert captured["kwargs"]["json"] == {"reason": "started by mistake"}
    assert captured["kwargs"]["headers"] == {
        "X-authentik-uid": "tailscale:owner@example.com",
        "X-authentik-username": "owner@example.com",
    }


def test_cancel_route_fails_closed_without_an_authenticated_subject() -> None:
    request = Request({"type": "http", "headers": []})
    with pytest.raises(HTTPException) as error:
        asyncio.run(runtime.cancel_endpoint(PREVIEW_HANDLE, ProfferCancelRequest(reason="x"), request, "TEST"))
    assert error.value.status_code == 401


def test_main_app_mounts_the_cancel_route() -> None:
    import main

    assert "/api/proffer/previews/{preview_handle}/cancel" in main.app.openapi()["paths"]
