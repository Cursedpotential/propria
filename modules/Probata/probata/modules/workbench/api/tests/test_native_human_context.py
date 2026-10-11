"""Check native SMS BFF registration, read relays and search without touching genuine sources."""
import asyncio
import httpx
import pytest

from fastapi import FastAPI
from fastapi.testclient import TestClient
from starlette.requests import Request

from app.runtime import context_sources
from app.service import imported as service
from app.service import imported_search


def actor():
    """Return an existing synthetic actor context; no identity service or new authentication step."""
    request = Request({"type": "http", "method": "GET", "path": "/"})
    request.state.subject_uid = "synthetic-owner"
    request.state.principal = "synthetic-owner"
    return request


def test_sms_routes_relay_exact_source_and_native_pages(monkeypatch):
    """Native SMS POST and page GET use actual IDs/pins and the current Go endpoints."""
    calls = []
    async def upstream(method, path, **kwargs):
        calls.append((method, path, kwargs))
        return object()
    monkeypatch.setattr(context_sources.proffer, "_request", upstream)
    async def native_upstream(path, headers):
        return await upstream("GET", path, headers=headers)
    monkeypatch.setattr(context_sources, "_native_bounded_get", native_upstream)
    monkeypatch.setattr(context_sources.proffer, "_json_payload", lambda *_: {"workflow_id": "synthetic-workflow", "run_id": "synthetic-run"})
    source = context_sources.HumanContextSource(source_ref="b2://synthetic/sms.xml", source_kind="sms", declared_format="smsbackuprestore_xml", byte_length=987, etag='"synthetic-pin"', owner_person_ref="synthetic-owner")
    result = asyncio.run(context_sources.submit_ai_context_source(source, actor(), "synthetic-click"))
    assert result["workflow_id"] == "synthetic-workflow"
    assert calls[0][2]["json"]["byte_length"] == 987
    assert calls[0][2]["json"]["etag"] == '"synthetic-pin"'
    assert calls[0][2]["json"]["source_kind"] == "sms"
    handle = "human-11111111-1111-4111-8111-111111111111"
    asyncio.run(context_sources.human_context_content(handle, actor(), 25, 25))
    assert calls[-1][1] == f"/reference-import/context/previews/{handle}/content?offset=25&limit=25"
    app = FastAPI()
    app.include_router(context_sources.router)
    paths = app.openapi()["paths"]
    assert "post" in paths["/api/context/sources"]
    assert "get" in paths["/api/context/previews/{handle}/content"]
    assert "get" in paths["/api/context/sources/workflows/{workflow_id}/candidates"]


def test_native_search_does_not_need_legacy_case_or_thread_rows(monkeypatch):
    """A native chunk opens its registered source handle while legacy case scope stays separate."""
    def no_case():
        raise service.ImportedError("no fixed case")
    def unexpected(*args):
        raise AssertionError("native search attempted a legacy message lookup")
    monkeypatch.setattr(service, "live_matter", no_case)
    monkeypatch.setattr(service, "_people", unexpected)
    monkeypatch.setattr(service.pg, "versions_to_threads", unexpected)
    sent = []
    version = "11111111-1111-4111-8111-111111111111"
    class Client:
        """Synthetic read-only Weaviate transport returning one native conversation chunk."""
        def __init__(self, *a, **k): pass
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False
        async def post(self, url, json):
            sent.append(json["query"])
            class Response:
                status_code = 200
                def json(self):
                    return {"data": {"Get": {service.settings.imported_weaviate_class: [{"origin_system": "probata-native-context", "source_version_ids": [version], "record_kind": "conversation_chunk", "text": "synthetic SMS passage", "first_message_id": f"native-record:{version}:26:0:20"}]}}}
            return Response()
    monkeypatch.setattr(service.httpx, "AsyncClient", Client)
    result = asyncio.run(imported_search.search("synthetic", limit=10, offset=0))
    item = result["items"][0]
    assert item["preview_handle"] == "human-" + version
    assert item["thread_id"] is None
    assert item["source_version_id"] == version
    assert '"matter_id"' not in sent[0]
    assert 'origin_system' in sent[0]


def test_native_relay_stops_stream_before_collecting_oversize_body(monkeypatch):
    """Oversize native streams stop at the budget and close without reading the rest.

    Inputs: synthetic streaming transport. Outputs: budget error and bounded consumption.
    Effects: in-memory transport only; no genuine source reads or writes.
    """
    observed = {"chunks": 0, "closed": False}
    class Stream(httpx.AsyncByteStream):
        """Supply many bounded pieces to expose accidental whole-body buffering."""
        async def __aiter__(self):
            for _ in range(1000):
                observed["chunks"] += 1
                yield b"x" * 65536
        async def aclose(self):
            observed["closed"] = True
    original_client = httpx.AsyncClient
    transport = httpx.MockTransport(lambda request: httpx.Response(200, stream=Stream()))
    monkeypatch.setattr(context_sources.httpx, "AsyncClient", lambda **kwargs: original_client(transport=transport, **kwargs))
    monkeypatch.setattr(context_sources.proffer.settings, "proffer_starter_url", "http://synthetic")
    monkeypatch.setattr(context_sources.proffer, "_service_authorization_headers", lambda: {"Authorization": "Bearer synthetic"})
    with pytest.raises(context_sources.ProfferError, match="byte budget"):
        asyncio.run(context_sources._native_bounded_get("/native", {}))
    assert observed["chunks"] == 65
    assert observed["closed"]
