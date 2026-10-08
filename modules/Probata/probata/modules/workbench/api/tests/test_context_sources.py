"""Check Workbench's actor-bound AI context handoff to the Go engine."""

from __future__ import annotations

import asyncio

import pytest
from fastapi import HTTPException
from pydantic import ValidationError
from starlette.requests import Request

from app.runtime import context_sources
from app.runtime import source_inspection as source_runtime
from app.service import source_inspection
from app.repo import object_store_client
from app.types.source_roots import parse_source_roots
from fastapi import FastAPI
from fastapi.testclient import TestClient


def _request(uid: str = "owner-subject") -> Request:
    """Construct a verified-actor request without opening a network door.

    Input: subject UID. Output: Starlette request. Effects: none; use for BFF
    identity tests instead of invoking a live Go workflow.
    """
    request = Request({"type": "http", "method": "POST", "path": "/api/context/sources"})
    request.state.subject_uid = uid
    request.state.principal = "owner"
    return request


@pytest.mark.parametrize("declared_format", ["chatgpt", "claude", "gemini_markdown"])
def test_ai_context_start_keeps_complete_source_version_and_actor(monkeypatch, declared_format) -> None:
    """Require exact original refs and actor identity at the Go boundary.

    Inputs: explicitly selected ChatGPT export and provider version. Output:
    Temporal IDs. Effects: in-process fake only; choose to catch accidental
    source normalization, case gating or direct Python extraction.
    """
    calls: list[tuple[str, str, dict, dict]] = []

    async def fake_request(method: str, path: str, *, json: dict, headers: dict):
        """Record the one Go starter call without network IO."""
        calls.append((method, path, json, headers))
        return object()

    monkeypatch.setattr(context_sources.proffer, "_request", fake_request)
    monkeypatch.setattr(context_sources.proffer, "_json_payload", lambda _response, _label: {"workflow_id": "context-source:one", "run_id": "run-one"})
    source = context_sources.AIContextSource(
        source_ref="b2://casevault/export.json", provider_version_id="b2-version-one",
        package_ref="b2://casevault/export-package/", declared_format=declared_format,
    )

    result = asyncio.run(context_sources.submit_ai_context_source(source, _request(), "click-one"))

    assert result == {"workflow_id": "context-source:one", "run_id": "run-one"}
    assert calls == [(
        "POST", "/reference-import/context/sources",
        {
            "contract_version": "context-v1", "source_ref": "b2://casevault/export.json",
            "provider_version_id": "b2-version-one", "package_ref": "b2://casevault/export-package/",
            "source_kind": "ai_chat", "declared_format": declared_format,
        },
        {"X-authentik-uid": "owner-subject", "X-authentik-username": "owner", "Idempotency-Key": "click-one"},
    )]


def test_ai_context_requires_explicit_provider_and_version() -> None:
    """Prevent generic JSON or unpinned remote objects from entering the AI path.

    Inputs: missing provider/version. Output: validation errors. Effects: none;
    choose before presenting the AI action for a selected B2 object.
    """
    with pytest.raises(ValidationError):
        context_sources.AIContextSource(source_ref="b2://casevault/file.json", provider_version_id="", declared_format="chatgpt")
    with pytest.raises(ValidationError):
        context_sources.AIContextSource(source_ref="b2://casevault/file.json", provider_version_id="version-one", declared_format="message_export_json")


def test_ai_context_rejects_missing_actor_before_go(monkeypatch) -> None:
    """Reject unauthenticated submissions before any Go request can be sent.

    Input: missing verified actor. Output: HTTP 401. Effects: none; choose to
    protect context workflow starts at the Workbench BFF boundary.
    """
    async def unexpected_request(*_args, **_kwargs):
        """Fail if a missing actor reaches the Go starter."""
        raise AssertionError("missing actor reached Go")

    monkeypatch.setattr(context_sources.proffer, "_request", unexpected_request)
    source = context_sources.AIContextSource(source_ref="b2://casevault/file.json", provider_version_id="version-one", declared_format="claude")
    with pytest.raises(HTTPException) as error:
        asyncio.run(context_sources.submit_ai_context_source(source, _request(uid=""), "click-two"))
    assert error.value.status_code == 401


def test_ai_context_routes_are_mounted() -> None:
    """Require both AI context routes on the production Workbench app.

    Inputs: production FastAPI app. Output: route assertions. Effects: none;
    choose to catch an omitted router include during integration.
    """
    import main

    paths = main.app.openapi()["paths"]
    assert "post" in paths["/api/context/sources"]
    assert "get" in paths["/api/context/sources/workflows/{workflow_id}"]


def test_selected_source_version_is_head_only_even_for_large_export(monkeypatch) -> None:
    """Return the provider VersionId without hashing or opening a large export.

    Inputs: selected B2 object and matching listing facts. Output: real VersionId.
    Effects: in-process fake HEAD only; choose to protect large AI-chat exports.
    """
    raw = '[{"id":"b2-vault","label":"B2 Casevault","url":"b2://salem-data/consignatio/casevault/"}]'
    monkeypatch.setattr(object_store_client, "SOURCE_ROOTS", parse_source_roots(raw))
    calls = []

    def fake_head(root_id, key):
        """Record the one selected-object metadata call."""
        calls.append((root_id, key))
        return {"ContentLength": 600_000_000, "ETag": '"listed-etag"', "VersionId": "actual-provider-version"}

    def unexpected_open(*_args, **_kwargs):
        """Fail if metadata lookup reads an AI-chat body."""
        raise AssertionError("metadata lookup opened source body")

    monkeypatch.setattr(source_inspection, "head_source_object", fake_head)
    monkeypatch.setattr(source_inspection, "open_source_object", unexpected_open)
    app = FastAPI()
    app.include_router(source_runtime.router)
    response = TestClient(app).post("/api/proffer/source-version", json={
        "root_id": "b2-vault", "source_ref": "b2://salem-data/consignatio/casevault/chat/export.json",
        "key": "chat/export.json", "expected_byte_length": 600_000_000, "expected_etag": '"listed-etag"',
    })
    assert response.status_code == 200
    assert response.json() == {
        "source_ref": "b2://salem-data/consignatio/casevault/chat/export.json",
        "provider_version_id": "actual-provider-version",
        "byte_length": 600_000_000, "etag": '"listed-etag"',
    }
    assert calls == [("b2-vault", "chat/export.json")]


def test_selected_source_without_provider_version_cannot_start_context(monkeypatch) -> None:
    """Reject unversioned B2 metadata instead of substituting an ETag.

    Input: selected source whose provider omits VersionId. Output: HTTP 409.
    Effects: in-process fake only; choose before offering a context import.
    """
    raw = '[{"id":"b2-vault","label":"B2 Casevault","url":"b2://salem-data/consignatio/casevault/"}]'
    monkeypatch.setattr(object_store_client, "SOURCE_ROOTS", parse_source_roots(raw))
    monkeypatch.setattr(source_inspection, "head_source_object", lambda _root_id, _key: {
        "ContentLength": 1, "ETag": '"listed-etag"',
    })
    app = FastAPI()
    app.include_router(source_runtime.router)
    response = TestClient(app).post("/api/proffer/source-version", json={
        "root_id": "b2-vault", "source_ref": "b2://salem-data/consignatio/casevault/chat/export.json",
        "key": "chat/export.json", "expected_byte_length": 1, "expected_etag": '"listed-etag"',
    })
    assert response.status_code == 409
    assert "no provider version ID" in response.json()["detail"]
