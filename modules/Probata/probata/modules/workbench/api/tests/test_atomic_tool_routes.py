"""Check that the production Workbench app exposes the source-pinned action API."""

import asyncio

import pytest
from fastapi import HTTPException
from starlette.requests import Request

from app.runtime import atomic_tool_actions as actions


def test_atomic_tool_routes_are_in_the_production_app() -> None:
    """Require both action start and status paths in the real app's OpenAPI map.

    Inputs: the production FastAPI app. Outputs: route/method assertions.
    Effects: none; choose this to catch a BFF router omitted from main.py.
    """
    import main

    paths = main.app.openapi()["paths"]
    assert "post" in paths["/api/atomic-tool-actions"]
    assert "get" in paths["/api/atomic-tool-actions/{workflow_id}"]


def _request(uid: str = "owner-subject") -> Request:
    """Build an actor-bearing in-process request without network side effects.

    Inputs: subject UID. Output: Starlette request. Effects: none; use for
    start/status identity tests instead of calling a live Proffer service.
    """
    request = Request({"type": "http", "method": "POST", "path": "/api/atomic-tool-actions"})
    request.state.subject_uid = uid
    request.state.principal = "owner"
    return request


def test_start_forwards_canonical_identity_and_exact_source_pin(monkeypatch) -> None:
    """Require the BFF to forward the actor and immutable source to Proffer.

    Inputs: authenticated request and validated action. Output: Temporal IDs.
    Effects: in-process fake only; choose to catch lost source/actor fields.
    """
    calls: list[tuple[str, str, dict, dict]] = []

    async def fake_request(method: str, path: str, *, json: dict, headers: dict):
        """Capture one outbound request while avoiding all network IO."""
        calls.append((method, path, json, headers))
        return object()

    monkeypatch.setattr(actions.proffer, "_request", fake_request)
    monkeypatch.setattr(actions.proffer, "_json_payload", lambda _response, _label: {"workflow_id": "source-pinned-tool:run", "run_id": "run"})
    monkeypatch.setattr(actions, "configured_matter_id", lambda _mode: "matter-a")
    monkeypatch.setattr(actions, "configured_court_case_id", lambda _mode: "case-a")
    body = actions.SourceAction(tool_id="repair.detect", source_ref="upload://source", source_sha256="a" * 64)

    result = asyncio.run(actions.start_source_action(body, _request(), "click-1"))

    assert result == {"workflow_id": "source-pinned-tool:run", "run_id": "run"}
    method, path, payload, headers = calls[0]
    assert (method, path) == ("POST", "/reference-import/atomic-tools/actions")
    assert payload == {
        "operating_mode": "LIVE", "matter_id": "matter-a", "court_case_id": "case-a",
        "tool_id": "repair.detect", "source_ref": "upload://source",
        "source_sha256": "a" * 64, "args": {},
    }
    assert headers == {"X-authentik-uid": "owner-subject", "X-authentik-username": "owner", "Idempotency-Key": "click-1"}


def test_start_refuses_missing_actor_before_proffer(monkeypatch) -> None:
    """Reject missing verified identity before a tool workflow can start.

    Inputs: request without subject UID. Output: HTTP 401. Effects: none;
    choose to catch an auth-regression at the BFF boundary.
    """
    async def unexpected_request(*_args, **_kwargs):
        """Fail if an unauthenticated request ever reaches Proffer."""
        raise AssertionError("unauthenticated action reached Proffer")

    monkeypatch.setattr(actions.proffer, "_request", unexpected_request)
    body = actions.SourceAction(tool_id="repair.detect", source_ref="upload://source", source_sha256="a" * 64)
    with pytest.raises(HTTPException) as error:
        asyncio.run(actions.start_source_action(body, _request(uid=""), "click-2"))
    assert error.value.status_code == 401
