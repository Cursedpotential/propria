"""Fresh imports use canonical upload receipts; retired R2 stays historical only.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

import hashlib
import tempfile

import httpx
import pytest
from app.repo import staging, staged_acquisition
from app.runtime import files, proffer, runs, upload
from app.service import proffer as service
from app.service import proffer_streams
from app.service import upload as legacy_upload
from fastapi import FastAPI
from fastapi.testclient import TestClient

DATA = b'<smses><sms body="route regression" /></smses>'
SHA = hashlib.sha256(DATA).hexdigest()
KEY = f"workbench/staging/{SHA}/sms.xml"
HANDLE = "preview_abcdefghijklmnopqrstuvwxyz012345"


def client():
    app = FastAPI()
    for router in (proffer.router, runs.router, upload.router, files.router):
        app.include_router(router)
    return TestClient(app)


def test_fresh_upload_streams_exact_bytes_then_start_calls_go(monkeypatch, tmp_path):
    token = tmp_path / "service-token"
    token.write_text("t" * 40)
    monkeypatch.setattr(service.settings, "proffer_service_token_file", str(token))
    monkeypatch.setattr(service.settings, "proffer_starter_url", "https://starter.internal")
    upstream = []
    original_client = httpx.AsyncClient

    async def accept(request):
        upstream.append(request)
        assert request.url.path == "/acquisition/upload"
        assert await request.aread() == DATA
        assert request.headers["authorization"] == "Bearer " + "t" * 40
        assert request.headers["content-type"] == "application/xml"
        assert request.headers["content-length"] == str(len(DATA))
        return httpx.Response(201, json={"acquisition_ref": f"upload://{SHA}", "sha256": SHA, "byte_length": len(DATA)})

    monkeypatch.setattr(proffer_streams.httpx, "AsyncClient",
                        lambda **kwargs: original_client(transport=httpx.MockTransport(accept), **kwargs))
    captured = []

    async def request(method, path, **kwargs):
        captured.append((method, path, kwargs))
        if method == "GET":
            return httpx.Response(200, json={"preview_handle": HANDLE,
                "matter_id": "11111111-1111-4111-8111-111111111111", "operating_mode": "LIVE"})
        return httpx.Response(201, json={"preview_handle": HANDLE})

    monkeypatch.setattr(service, "_request", request)
    monkeypatch.setattr(staging, "get", lambda *_: pytest.fail("new flow must not query staging"))
    monkeypatch.setattr(staged_acquisition, "open_staged_object", lambda *_: pytest.fail("new flow must not read R2"))
    with client() as browser:
        acquisition = browser.post("/api/proffer/upload?mode=LIVE", content=DATA, headers={"content-type": "application/xml"})
        assert acquisition.status_code == 201, acquisition.text
        receipt = acquisition.json()
        assert receipt == {"acquisition_ref": f"upload://{SHA}", "sha256": SHA,
                           "byte_length": len(DATA), "matter_mode": "LIVE"}
        result = browser.post("/api/proffer/start?mode=LIVE", json={
            "request_id": "route-test", "source_ref": receipt["acquisition_ref"],
            "matter_id": "11111111-1111-4111-8111-111111111111",
            "court_case_id": "22222222-2222-4222-8222-222222222222",
            "declared_format": "sms_export_xml", "parser_options_ref": "pending-handler-selection/v1",
            "source_context_ref": "11111111-1111-4111-8111-111111111111", "matter_mode": "LIVE",
        })
        assert result.status_code == 201, result.text
    assert len(upstream) == 1
    assert captured[0][:2] == ("POST", "/reference-import/start")
    assert captured[0][2]["json"]["operating_mode"] == "LIVE"
    assert "engine" not in captured[0][2]["json"]


@pytest.mark.parametrize("mode", ["LIVE", "REAL", "DEV"])
def test_retired_staged_acquisition_rejects_before_lookup_or_object_io(monkeypatch, mode):
    monkeypatch.setattr(staging, "get", lambda *_: pytest.fail("retired flow must not query staging"))
    monkeypatch.setattr(staged_acquisition, "open_staged_object", lambda *_: pytest.fail("retired flow must not read R2"))
    with client() as browser:
        result = browser.post(f"/api/proffer/staged/{SHA}/acquisition?mode={mode}")
        assert browser.post("/api/proffer/staged/not-a-hash/acquisition?mode=LIVE").status_code == 422
    assert result.status_code == 409
    if mode != "DEV":
        assert "re-upload" in result.json()["detail"]


def test_legacy_upload_refuses_before_body_tempfile_or_storage_io(monkeypatch):
    monkeypatch.setattr(tempfile, "mkstemp", lambda *_args, **_kwargs: pytest.fail("must not create tempfile"))
    monkeypatch.setattr(legacy_upload, "stage_upload", lambda *_: pytest.fail("must not stage bytes"))
    with client() as browser:
        for kwargs in ({}, {"content": DATA}, {"files": {"file": ("sms.xml", DATA)}}):
            response = browser.post("/api/upload", **kwargs)
            assert response.status_code == 410
            assert "/api/proffer/upload" in response.json()["detail"]


def test_historical_listing_and_provenance_keep_original_r2_identity(monkeypatch):
    historical = {"id": SHA, "name": "sms.xml", "r2_key": KEY, "source_ref": f"r2://nexus/{KEY}",
                  "size": len(DATA), "status": "staged", "meta": {"original": True}}
    monkeypatch.setattr(files, "list_staged", lambda **_: [historical])
    monkeypatch.setattr(files, "get_staged_detail", lambda _id: historical)
    with client() as browser:
        assert browser.get("/api/files").json() == [historical]
        assert browser.get(f"/api/files/{SHA}").json() == historical


@pytest.mark.parametrize("mode", ["DEV", "TEST"])
def test_dev_canonical_upload_rejects_before_stream_open(monkeypatch, mode):
    async def forbidden(*_args, **_kwargs):
        pytest.fail("Dev must not dispatch")
    monkeypatch.setattr(service, "open_upload_stream", forbidden)
    with client() as browser:
        assert browser.post(f"/api/proffer/upload?mode={mode}", content=DATA).status_code == 409


def test_legacy_create_and_retry_cannot_call_python(monkeypatch):
    async def forbidden(**kwargs):
        raise AssertionError("legacy Python ingest must never run")

    monkeypatch.setattr(runs, "start_run", forbidden)
    with client() as browser:
        assert browser.post("/api/runs", json={"workflow": "sms-xml"}).status_code == 410
        assert browser.post("/api/runs", files={"file": ("sms.xml", DATA)}).status_code == 410
        assert browser.post("/api/runs/original-failed/retry").status_code == 410
        assert browser.post("/api/runs/parse-dryrun", json={"sha256": SHA}).status_code == 410
