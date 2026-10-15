"""Folder-batch passthrough: scope, mode echo and identity checks.

Byline: Claude Code · Opus 5 · 2026-09-22.
"""

import json
from uuid import uuid4

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.runtime import operating_mode
from app.runtime.proffer_batch import router
from app.service import proffer_batch as service
from app.service.proffer_errors import ProfferError

BATCH_ID = "batch" + "0123456789abcdef" * 2
MATTER_ID = "11111111-1111-4111-8111-111111111111"
COURT_CASE_ID = "22222222-2222-4222-8222-222222222222"


def _body(**overrides):
    body = {
        "batch_id": BATCH_ID,
        "matter_id": MATTER_ID,
        "court_case_id": COURT_CASE_ID,
        "folder_ref": "b2://salem-data/consignatio/vault/v1/calls/",
        "declared_format": "xml",
        "parser_options_ref": "pending-handler-selection/v1",
        "matter_mode": "LIVE",
    }
    body.update(overrides)
    return body


@pytest.fixture
def client(monkeypatch):
    async def verified_scope(_mode):
        return None
    monkeypatch.setattr(operating_mode, "verify_case_scope", verified_scope)
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


def _response(status: int, payload: dict) -> httpx.Response:
    return httpx.Response(status, content=json.dumps(payload), headers={"content-type": "application/json"})


def _status():
    return {"batch_id": BATCH_ID, "prefix": "folder/", "terminal": False, "counts": {}, "items": [],
            "operating_mode": "LIVE", "matter_id": MATTER_ID, "court_case_id": COURT_CASE_ID}


def test_start_batch_passes_the_folder_through_and_echoes_the_mode(client, monkeypatch):
    seen = {}

    async def request(method, path, **kwargs):
        if method == "GET":
            return _response(200, _status())
        seen["method"], seen["path"], seen["json"] = method, path, kwargs.get("json")
        return _response(201, {"batch_id": BATCH_ID})

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    response = client.post("/api/proffer/start-batch?mode=LIVE", json=_body())
    assert response.status_code == 201
    assert response.json() == {"batch_id": BATCH_ID, "matter_mode": "LIVE"}
    assert seen["method"] == "POST" and seen["path"] == "/reference-import/start-batch"
    # The BFF's own mode echo is never forwarded to the engine.
    assert "matter_mode" not in seen["json"]
    assert seen["json"]["operating_mode"] == "LIVE"
    assert seen["json"]["folder_ref"] == "b2://salem-data/consignatio/vault/v1/calls/"
    assert "unit_roots" not in seen["json"]


def test_three_package_roots_are_forwarded_as_one_batch_without_combining_units(client, monkeypatch):
    parent = "b2://fixture/vault/Incoming/"
    roots = [parent + name + "/" for name in ("+15550000001 - Ada", "+15550000002 - Beth", "+15550000003 - Chris")]
    monkeypatch.setenv("SOURCE_ROOTS_JSON", json.dumps([{"id": "fixture", "label": "Fixture", "url": "b2://fixture/vault/"}]))
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    calls = []

    async def request(method, path, **kwargs):
        calls.append((method, path, kwargs.get("json")))
        return _response(200, _status()) if method == "GET" else _response(201, {"batch_id": BATCH_ID})

    monkeypatch.setattr(service, "_request", request)
    response = client.post("/api/proffer/start-batch?mode=LIVE", json=_body(folder_ref=parent, unit_roots=roots))
    assert response.status_code == 201
    starts = [body for method, _, body in calls if method == "POST"]
    assert len(starts) == 1
    assert starts[0]["folder_ref"] == parent
    assert starts[0]["unit_roots"] == roots


@pytest.mark.parametrize("roots", [
    [], ["b2://fixture/vault/Incoming/A/"] * 2,
    ["b2://fixture/vault/Incoming/A/", "b2://fixture/vault/Incoming/A/attachments/"],
    ["b2://fixture/vault/Incoming/A/", "b2://other/vault/Incoming/B/"],
    ["b2://fixture/vault/IncomingWrong/A/"], ["b2://fixture/outside/Incoming/A/"],
    ["b2://fixture/vault/Incoming/../outside/"], ["b2://fixture/vault/Incoming/%2e%2e/outside/"],
    ["b2://fixture/vault/Incoming/A/?versionId=not-a-folder-version"],
    ["b2://fixture/vault/Incoming/A/", "b2://fixture/vault/Incoming/%41/"],
    [f"b2://fixture/vault/Incoming/{index}/" for index in range(33)],
])
def test_invalid_package_roots_never_reach_engine(client, monkeypatch, roots):
    monkeypatch.setenv("SOURCE_ROOTS_JSON", json.dumps([{"id": "fixture", "label": "Fixture", "url": "b2://fixture/vault/"}]))
    monkeypatch.setattr(service, "require_scope", lambda *args: None)

    async def never(*args, **kwargs):
        raise AssertionError("invalid unit roots must not reach engine")

    monkeypatch.setattr(service, "_request", never)
    response = client.post("/api/proffer/start-batch?mode=LIVE", json=_body(folder_ref="b2://fixture/vault/Incoming/", unit_roots=roots))
    assert response.status_code == 422


def test_package_parent_may_be_the_explicitly_configured_bucket_root(monkeypatch):
    monkeypatch.setenv("SOURCE_ROOTS_JSON", json.dumps([{"id": "fixture", "label": "Fixture", "url": "b2://fixture/"}]))
    service.validate_unit_roots("b2://fixture/", ["b2://fixture/Incoming/A/", "b2://fixture/Incoming/B/"])


def test_a_body_mode_that_contradicts_the_query_is_refused(client, monkeypatch):
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    response = client.post("/api/proffer/start-batch?mode=DEV", json=_body())
    assert response.status_code == 409


def test_a_different_batch_id_coming_back_is_a_bad_gateway(client, monkeypatch):
    async def request(method, path, **kwargs):
        return _response(201, {"batch_id": "batch" + "f" * 32})

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    assert client.post("/api/proffer/start-batch?mode=LIVE", json=_body()).status_code == 502


def test_batch_status_carries_counts_and_the_mode(client, monkeypatch):
    async def request(method, path, **kwargs):
        assert path == f"/reference-import/batches/{BATCH_ID}"
        return _response(
            200,
            {
                "batch_id": BATCH_ID,
                "operating_mode": "LIVE", "matter_id": MATTER_ID, "court_case_id": COURT_CASE_ID,
                "prefix": "consignatio/vault/v1/calls/",
                "terminal": False,
                "listing_truncated": False,
                "items_truncated": False,
                "counts": {"total": 2, "queued": 1, "running": 1, "waiting_on_gate": 0, "done": 0, "failed": 0, "skipped": 0},
                "items": [{"key": "a.xml", "status": "running"}],
            },
        )

    monkeypatch.setattr(service, "_request", request)
    body = client.get(f"/api/proffer/batches/{BATCH_ID}?mode=LIVE").json()
    assert body["matter_mode"] == "LIVE" and body["counts"]["total"] == 2
    assert body["items"][0]["status"] == "running"


def test_an_engine_error_keeps_its_status(client, monkeypatch):
    async def request(method, path, **kwargs):
        raise ProfferError("batch import is not configured on this service", 503)

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(service, "require_scope", lambda *args: None)
    assert client.post("/api/proffer/start-batch?mode=LIVE", json=_body()).status_code == 503


def test_a_malformed_batch_id_never_reaches_the_engine(client, monkeypatch):
    def fail(*args, **kwargs):
        raise AssertionError("the engine must not be called")

    monkeypatch.setattr(service, "_request", fail)
    assert client.get(f"/api/proffer/batches/{uuid4().hex[:8]}?mode=LIVE").status_code == 422


@pytest.mark.parametrize("patch", [{"operating_mode": ""}, {"operating_mode": "REAL"},
                                  {"operating_mode": "DEV"}, {"matter_id": COURT_CASE_ID},
                                  {"court_case_id": MATTER_ID}])
def test_missing_or_wrong_durable_batch_binding_is_never_echoed_as_live(client, monkeypatch, patch):
    async def request(_method, _path, **_kwargs):
        return _response(200, {**_status(), **patch})
    monkeypatch.setattr(service, "_request", request)
    response = client.get(f"/api/proffer/batches/{BATCH_ID}?mode=LIVE")
    assert response.status_code == 409
    assert "matter_mode" not in response.json()
