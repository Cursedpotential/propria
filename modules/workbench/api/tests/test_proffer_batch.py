"""Folder-batch passthrough: scope, mode echo and identity checks.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

import json
from uuid import uuid4

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.runtime.proffer_batch import router
from app.service import proffer_batch as service
from app.service.proffer_errors import ProfferError

BATCH_ID = "batch" + "0123456789abcdef" * 2
MATTER_ID = "deadbeef-dead-beef-dead-beefdeadbeef"
COURT_CASE_ID = "cafebabe-cafe-babe-cafe-babecafebabe"


def _body(**overrides):
    body = {
        "batch_id": BATCH_ID,
        "matter_id": MATTER_ID,
        "court_case_id": COURT_CASE_ID,
        "folder_ref": "b2://salem-data/consignatio/vault/v1/calls/",
        "declared_format": "xml",
        "parser_options_ref": "pending-handler-selection/v1",
        "matter_mode": "TEST",
    }
    body.update(overrides)
    return body


@pytest.fixture
def client():
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


def _response(status: int, payload: dict) -> httpx.Response:
    return httpx.Response(status, content=json.dumps(payload), headers={"content-type": "application/json"})


def test_start_batch_passes_the_folder_through_and_echoes_the_mode(client, monkeypatch):
    seen = {}

    async def request(method, path, **kwargs):
        seen["method"], seen["path"], seen["json"] = method, path, kwargs.get("json")
        return _response(201, {"batch_id": BATCH_ID})

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    response = client.post("/api/proffer/start-batch?mode=TEST", json=_body())
    assert response.status_code == 201
    assert response.json() == {"batch_id": BATCH_ID, "matter_mode": "TEST"}
    assert seen["method"] == "POST" and seen["path"] == "/reference-import/start-batch"
    # The BFF's own mode echo is never forwarded to the engine.
    assert "matter_mode" not in seen["json"]
    assert seen["json"]["folder_ref"] == "b2://salem-data/consignatio/vault/v1/calls/"


def test_a_body_mode_that_contradicts_the_query_is_refused(client, monkeypatch):
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    response = client.post("/api/proffer/start-batch?mode=REAL", json=_body())
    assert response.status_code == 409


def test_a_different_batch_id_coming_back_is_a_bad_gateway(client, monkeypatch):
    async def request(method, path, **kwargs):
        return _response(201, {"batch_id": "batch" + "f" * 32})

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    assert client.post("/api/proffer/start-batch?mode=TEST", json=_body()).status_code == 502


def test_batch_status_carries_counts_and_the_mode(client, monkeypatch):
    async def request(method, path, **kwargs):
        assert path == f"/reference-import/batches/{BATCH_ID}"
        return _response(
            200,
            {
                "batch_id": BATCH_ID,
                "prefix": "consignatio/vault/v1/calls/",
                "terminal": False,
                "listing_truncated": False,
                "items_truncated": False,
                "counts": {"total": 2, "queued": 1, "running": 1, "waiting_on_gate": 0, "done": 0, "failed": 0, "skipped": 0},
                "items": [{"key": "a.xml", "status": "running"}],
            },
        )

    monkeypatch.setattr(service, "_request", request)
    body = client.get(f"/api/proffer/batches/{BATCH_ID}?mode=TEST").json()
    assert body["matter_mode"] == "TEST" and body["counts"]["total"] == 2
    assert body["items"][0]["status"] == "running"


def test_an_engine_error_keeps_its_status(client, monkeypatch):
    async def request(method, path, **kwargs):
        raise ProfferError("batch import is not configured on this service", 503)

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    assert client.post("/api/proffer/start-batch?mode=TEST", json=_body()).status_code == 503


def test_a_malformed_batch_id_never_reaches_the_engine(client, monkeypatch):
    def fail(*args, **kwargs):
        raise AssertionError("the engine must not be called")

    monkeypatch.setattr(service, "_request", fail)
    assert client.get(f"/api/proffer/batches/{uuid4().hex[:8]}?mode=TEST").status_code == 422
