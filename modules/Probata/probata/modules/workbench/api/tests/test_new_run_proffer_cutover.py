"""New imports use server-authored R2 references and the Go Proffer port."""

import hashlib
import io

import httpx
from app.runtime import proffer, runs
from app.service import proffer as service
from app.service import proffer_staged
from fastapi import FastAPI
from fastapi.testclient import TestClient

DATA = b'<smses><sms body="route regression" /></smses>'
SHA = hashlib.sha256(DATA).hexdigest()
KEY = f"workbench/staging/{SHA}/sms.xml"


def client():
    app = FastAPI()
    app.include_router(proffer.router)
    app.include_router(runs.router)
    return TestClient(app)


def test_staged_route_authors_reference_then_start_calls_go(monkeypatch):
    monkeypatch.setattr(proffer_staged.staging, "get", lambda value: {"r2_key": KEY} if value == SHA else None)
    opened = []

    def open_source(key):
        opened.append(key)
        return {"Body": io.BytesIO(DATA), "ContentLength": len(DATA)}

    monkeypatch.setattr(proffer_staged, "open_staged_object", open_source)
    captured = []

    async def request(method, path, **kwargs):
        captured.append((method, path, kwargs))
        if method == "GET":
            return httpx.Response(200, json={"preview_handle": "preview_abcdefghijklmnopqrstuvwxyz012345",
                "matter_id": "11111111-1111-4111-8111-111111111111", "operating_mode": "LIVE"})
        return httpx.Response(201, json={"preview_handle": "preview_abcdefghijklmnopqrstuvwxyz012345"})

    monkeypatch.setattr(service, "_request", request)
    with client() as browser:
        acquisition = browser.post(f"/api/proffer/staged/{SHA}/acquisition?mode=LIVE")
        assert acquisition.status_code == 201
        receipt = acquisition.json()
        assert receipt == {"acquisition_ref": f"r2://nexus/{KEY}", "sha256": SHA,
                           "byte_length": len(DATA), "matter_mode": "LIVE"}
        result = browser.post("/api/proffer/start?mode=LIVE", json={
            "request_id": "route-test", "source_ref": receipt["acquisition_ref"],
            "matter_id": "11111111-1111-4111-8111-111111111111",
            "court_case_id": "22222222-2222-4222-8222-222222222222",
            "declared_format": "sms_export_xml", "parser_options_ref": "pending-handler-selection/v1",
            "source_context_ref": "11111111-1111-4111-8111-111111111111", "matter_mode": "LIVE",
        })
        assert result.status_code == 201, result.text
    assert opened == [KEY]
    assert captured[0][:2] == ("POST", "/reference-import/start")
    assert "engine" not in captured[0][2]["json"]


def test_staged_changed_bytes_fail_closed(monkeypatch):
    monkeypatch.setattr(proffer_staged.staging, "get", lambda value: {"r2_key": KEY})
    body = io.BytesIO(b"changed")
    monkeypatch.setattr(proffer_staged, "open_staged_object", lambda key: {"Body": body, "ContentLength": 7})
    with client() as browser:
        result = browser.post(f"/api/proffer/staged/{SHA}/acquisition?mode=LIVE")
    assert result.status_code == 409
    assert body.closed


def test_staged_identity_escape_and_unconfigured_mode_fail_closed(monkeypatch):
    monkeypatch.setattr(proffer_staged.staging, "get", lambda value: {"r2_key": "elsewhere/sms.xml"})
    with client() as browser:
        assert browser.post(f"/api/proffer/staged/{SHA}/acquisition?mode=LIVE").status_code == 502
        assert browser.post(f"/api/proffer/staged/{SHA}/acquisition?mode=DEV").status_code == 409
        assert browser.post("/api/proffer/staged/not-a-hash/acquisition?mode=LIVE").status_code == 422


def test_legacy_create_and_retry_cannot_call_python(monkeypatch):
    async def forbidden(**kwargs):
        raise AssertionError("legacy Python ingest must never run")

    monkeypatch.setattr(runs, "start_run", forbidden)
    with client() as browser:
        assert browser.post("/api/runs", json={"workflow": "sms-xml"}).status_code == 410
        assert browser.post("/api/runs", files={"file": ("sms.xml", DATA)}).status_code == 410
        assert browser.post("/api/runs/original-failed/retry").status_code == 410
        assert browser.post("/api/runs/parse-dryrun", json={"sha256": SHA}).status_code == 410
