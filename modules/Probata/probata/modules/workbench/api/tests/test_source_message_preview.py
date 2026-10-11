"""Prove raw previews stay bounded, pinned, cancellable and outside ELT."""
import asyncio
import json
import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from app.runtime import source_message_preview as runtime
from app.service import source_message_preview as service
from app.service.proffer_errors import ProfferError
from app.service.source_inspection import SourceInspectionError
from app.types.source_message_preview import SourceMessagePreviewRequest

SOURCE = "b2://salem-data/original export/sms+original.xml"
REQUEST = dict(root_id="b2-salem-data", key="original export/sms+original.xml", source_ref=SOURCE,
               expected_byte_length=941300000, expected_etag='"etag"')


def ready(**changes):
    """Return bounded synthetic transport data; no genuine source is modified."""
    payload = dict(page_id="page-1", status="ready", source=dict(source_ref=SOURCE, etag='"etag"', version_id="v1", byte_length=941300000),
                   messages=[], scanned_bytes=1048576, preview_only=True)
    payload.update(changes)
    return payload


def transport(monkeypatch, payload, status=200):
    """Stub only the Go HTTP boundary and preserve streaming response handling."""
    captured = []
    original = httpx.AsyncClient
    def handle(request):
        captured.append(request)
        return httpx.Response(status, json=payload)
    monkeypatch.setattr(service.settings, "proffer_starter_url", "http://go.test")
    monkeypatch.setattr(service, "_service_authorization_headers", lambda: {"Authorization": "Bearer synthetic-token"})
    monkeypatch.setattr(httpx, "AsyncClient", lambda **kwargs: original(**kwargs, transport=httpx.MockTransport(handle)))
    return captured


def test_start_uses_head_pins_and_does_not_start_import(monkeypatch):
    calls = transport(monkeypatch, ready())
    monkeypatch.setattr(service, "_metadata", lambda request: (request.key, {"ETag": '"etag"', "VersionId": "v1", "ContentLength": 941300000}))
    page, status = asyncio.run(service.start_page(SourceMessagePreviewRequest(**REQUEST)))
    assert status == 200 and page.preview_only
    assert len(calls) == 1 and calls[0].url.path == "/reference-import/source-preview"
    assert json.loads(calls[0].content) == {"source_ref": SOURCE, "expected_etag": '"etag"', "version_id": "v1", "limit": 25}


def test_changed_listing_rejects_before_go(monkeypatch):
    calls = transport(monkeypatch, ready())
    def changed(request):
        raise SourceInspectionError(409, "changed")
    monkeypatch.setattr(service, "_metadata", changed)
    with pytest.raises(SourceInspectionError):
        asyncio.run(service.start_page(SourceMessagePreviewRequest(**REQUEST)))
    assert calls == []


@pytest.mark.parametrize("status", [409, 410, 422, 503, 504])
def test_upstream_expected_errors_keep_status(monkeypatch, status):
    transport(monkeypatch, {}, status)
    with pytest.raises(ProfferError) as error:
        asyncio.run(service.poll_page("page-1"))
    assert error.value.status_code == status


def test_poll_never_follows_supplied_url(monkeypatch):
    calls = transport(monkeypatch, {"page_id": "page-1", "status": "pending", "poll_url": "https://elsewhere.test/secret"}, 202)
    with pytest.raises(ProfferError) as error:
        asyncio.run(service.poll_page("page-1"))
    assert error.value.status_code == 502
    assert len(calls) == 1 and calls[0].url.path == "/reference-import/source-preview/page-1"


@pytest.mark.parametrize("payload", [ready(messages=[{}] * 51), ready(messages=[dict(id="m1", kind="sms", direction="incoming", body="x" * 16385)]), ready(messages=[dict(id="m1", kind="mms", direction="incoming", body="x" * 600000)])])
def test_huge_rows_and_response_are_rejected(monkeypatch, payload):
    transport(monkeypatch, payload)
    with pytest.raises(ProfferError) as error:
        asyncio.run(service.poll_page("page-1"))
    assert error.value.status_code == 502


def test_ready_source_must_match_head(monkeypatch):
    payload = ready()
    payload["source"]["version_id"] = "wrong"
    transport(monkeypatch, payload)
    monkeypatch.setattr(service, "_metadata", lambda request: (request.key, {"ETag": '"etag"', "VersionId": "v1", "ContentLength": 941300000}))
    with pytest.raises(ProfferError, match="correlation"):
        asyncio.run(service.start_page(SourceMessagePreviewRequest(**REQUEST)))


def test_metadata_truncation_is_preserved(monkeypatch):
    transport(monkeypatch, ready(messages=[dict(id="m1", kind="sms", direction="incoming", metadata_truncated=True)]))
    page, _ = asyncio.run(service.poll_page("page-1"))
    assert page.messages[0].metadata_truncated is True


def test_routes_have_no_case_or_owner_gate(monkeypatch):
    async def start(body):
        return service.SourceMessagePreviewReady.model_validate(ready()), 200
    monkeypatch.setattr(service, "start_page", start)
    app = FastAPI()
    app.include_router(runtime.router, prefix="/api/proffer")
    with TestClient(app) as client:
        response = client.post("/api/proffer/source-preview", json=REQUEST)
    assert response.status_code == 200 and response.headers["cache-control"] == "no-store"


def test_disconnect_cancels_pending_upstream():
    cancelled = []
    class Request:
        checks = 0
        async def is_disconnected(self):
            self.checks += 1
            return self.checks > 1
    async def blocked():
        try:
            await asyncio.sleep(100)
        finally:
            cancelled.append(True)
    async def run():
        task = asyncio.create_task(blocked())
        await asyncio.sleep(0)
        with pytest.raises(Exception) as error:
            await runtime._connected(Request(), task_wait(task))
        assert error.value.status_code == 499
        await asyncio.sleep(0)
        assert task.cancelled()
    async def task_wait(task):
        return await task
    asyncio.run(run())
    assert cancelled == [True]
