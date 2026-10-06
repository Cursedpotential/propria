"""Real HTTPX scope transport contracts; no live secrets, sockets or mutations.

Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

import asyncio
import json
import time

import httpx
import pytest
from app.config import settings
from app.runtime import operating_mode
from app.service import case_scope, proffer
from app.service.proffer_errors import ProfferError
from fastapi import FastAPI
from fastapi.testclient import TestClient

MATTER = "11111111-1111-4111-8111-111111111111"
COURT = "22222222-2222-4222-8222-222222222222"
TOKEN = "synthetic-token-" + "x" * 32
CLIENT = httpx.AsyncClient


class Bytes(httpx.AsyncByteStream):
    """Yield controlled upstream chunks and record closure without socket I/O."""

    def __init__(self, chunks, delay=0):
        self.chunks, self.delay, self.reads, self.closed = chunks, delay, 0, False

    async def __aiter__(self):
        for chunk in self.chunks:
            if self.delay:
                await asyncio.sleep(self.delay)
            self.reads += 1
            yield chunk

    async def aclose(self):
        self.closed = True


def header(mode="LIVE"):
    return {"mode": mode, "matter": {"id": MATTER}, "court_case": {"id": COURT, "matter_id": MATTER}}


@pytest.fixture(autouse=True)
def configured(monkeypatch, tmp_path):
    token = tmp_path / "mounted-token"
    token.write_text(TOKEN + "\r\n")
    monkeypatch.setattr(settings, "proffer_matter_id", MATTER)
    monkeypatch.setattr(settings, "proffer_court_case_id", COURT)
    monkeypatch.setattr(settings, "proffer_starter_url", "http://starter.private:8000")
    monkeypatch.setattr(settings, "proffer_service_token_file", str(token))
    monkeypatch.setattr(operating_mode, "verify_case_scope", case_scope.verify_case_scope)
    return token


def transport(monkeypatch, handler):
    calls = []

    def factory(**kwargs):
        calls.append(kwargs)
        return CLIENT(transport=httpx.MockTransport(handler), **kwargs)

    monkeypatch.setattr(case_scope.httpx, "AsyncClient", factory)
    return calls


@pytest.mark.parametrize("mode", ["LIVE", "DEV"])
def test_private_authenticated_scope_excludes_environment_proxies_and_is_fresh(monkeypatch, mode):
    for key in ("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"):
        monkeypatch.setenv(key, "http://untrusted-proxy.invalid:9999")
    monkeypatch.setenv("NO_PROXY", "")
    monkeypatch.setenv("SSL_CERT_FILE", "/nonexistent/untrusted-cert")
    monkeypatch.setattr(CLIENT, "_init_proxy_transport", lambda *_a, **_k: pytest.fail("no proxy may be initialized"))
    requests = []
    streams = []

    def upstream(request):
        requests.append(request)
        assert str(request.url) == f"http://starter.private:8000/case-identity/scope?mode={mode}"
        assert request.method == "GET"
        assert request.headers["authorization"] == f"Bearer {TOKEN}"
        assert request.headers["accept-encoding"] == "identity"
        stream = Bytes([json.dumps(header(mode)).encode()])
        streams.append(stream)
        return httpx.Response(200, stream=stream)

    options = transport(monkeypatch, upstream)
    for _ in range(2):
        asyncio.run(case_scope.verify_case_scope(mode))
    assert len(requests) == len(options) == 2
    assert all(option == {"timeout": 5.0, "trust_env": False, "follow_redirects": False} for option in options)
    assert all(stream.closed for stream in streams)


@pytest.mark.parametrize("size", [65535, 65536, 65537])
def test_streaming_size_boundary_counts_actual_bytes_without_length_header(monkeypatch, size):
    data = json.dumps(header()).encode()
    data += b" " * (size - len(data))
    stream = Bytes([data[:40000], data[40000:]])
    transport(monkeypatch, lambda _: httpx.Response(200, stream=stream))
    if size <= 65536:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    else:
        with pytest.raises(ProfferError) as denied:
            asyncio.run(case_scope.verify_case_scope("LIVE"))
        assert denied.value.status_code == 502
    assert stream.closed


@pytest.mark.parametrize("headers", [{"content-length": "65537"}, {"content-length": "-1"},
                                     {"content-length": "invalid"}, {"content-encoding": "gzip"}])
def test_invalid_bounds_or_compression_reject_before_reading_body(monkeypatch, headers):
    stream = Bytes([b"unbounded secret-bearing error body"])
    transport(monkeypatch, lambda _: httpx.Response(200, headers=headers, stream=stream))
    with pytest.raises(ProfferError) as denied:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    assert denied.value.status_code == 502
    assert stream.reads == 0 and stream.closed


@pytest.mark.parametrize("status", [201, 301, 302, 307, 308, 401, 500])
def test_non_200_is_denied_without_redirect_body_read_or_bearer_forwarding(monkeypatch, status):
    requests = []
    stream = Bytes([TOKEN.encode()])

    def upstream(request):
        requests.append(request)
        return httpx.Response(status, headers={"location": "http://untrusted-target.invalid"}, stream=stream)

    transport(monkeypatch, upstream)
    with pytest.raises(ProfferError) as denied:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    assert denied.value.status_code == 503
    assert TOKEN not in str(denied.value)
    assert len(requests) == 1 and requests[0].url.host == "starter.private"
    assert stream.reads == 0 and stream.closed


@pytest.mark.parametrize("raw", [b'{"mode":"DEV","mode":"LIVE"}',
                                 b'{"matter":{"id":"foreign","id":"approved"}}',
                                 b"not-json", b"[]", b"\xff", b"{" * 2000])
def test_ambiguous_or_invalid_json_fails_closed(monkeypatch, raw):
    stream = Bytes([raw])
    transport(monkeypatch, lambda _: httpx.Response(200, stream=stream))
    with pytest.raises(ProfferError) as denied:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    assert denied.value.status_code == 502
    assert stream.closed


@pytest.mark.parametrize("deadline", [0.08, 5.0])
def test_total_deadline_stops_dripping_response_and_closes_stream(monkeypatch, deadline):
    # Each chunk arrives inside the read-inactivity budget, but total duration exceeds it.
    monkeypatch.setattr(case_scope, "_TIMEOUT_SECONDS", deadline)
    stream = Bytes([b" "] * 100, delay=deadline / 4)
    transport(monkeypatch, lambda _: httpx.Response(200, stream=stream))
    started = time.monotonic()
    with pytest.raises(ProfferError) as denied:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    assert denied.value.status_code == 503
    assert deadline <= time.monotonic() - started < deadline + 0.5
    assert 0 < stream.reads < 100 and stream.closed


@pytest.mark.parametrize("url", ["", "ftp://starter", "http://user:password@starter", "http://starter?",
                                "http://starter#", "http://starter?q=1", "http://starter/#fragment",
                                "http://starter:99999", "http://starter:0", "http://starter\\evil", "http://%73tarter",
                                "http://starter\n", "http://starter/path with space"])
def test_invalid_url_is_denied_before_auth_or_network(monkeypatch, url):
    monkeypatch.setattr(settings, "proffer_starter_url", url)
    monkeypatch.setattr(proffer, "_service_authorization_headers", lambda: pytest.fail("must not read secret"))
    monkeypatch.setattr(case_scope.httpx, "AsyncClient", lambda **_: pytest.fail("must not open client"))
    with pytest.raises(ProfferError) as denied:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    assert denied.value.status_code == 503
    assert "password" not in str(denied.value)


@pytest.mark.parametrize("kind", ["relative", "missing", "directory", "symlink", "short", "oversized", "invalid"])
def test_auth_requires_existing_regular_bounded_valid_mounted_file(monkeypatch, configured, tmp_path, kind):
    path = configured
    if kind == "relative":
        path = "relative-token"
    elif kind == "missing":
        path = tmp_path / "missing"
    elif kind == "directory":
        path = tmp_path
    elif kind == "symlink":
        path = tmp_path / "link"
        path.symlink_to(configured)
    else:
        path.write_bytes({"short": b"short", "oversized": b"a" * 4099, "invalid": b"a" * 40 + b"\x00"}[kind])
    monkeypatch.setattr(settings, "proffer_service_token_file", str(path))
    monkeypatch.setattr(case_scope.httpx, "AsyncClient", lambda **_: pytest.fail("must not open client"))
    with pytest.raises(ProfferError) as denied:
        asyncio.run(case_scope.verify_case_scope("LIVE"))
    assert denied.value.status_code == 503 and TOKEN not in str(denied.value)


@pytest.mark.parametrize("mode,field,value", [("unknown", None, None), ("LIVE", "proffer_matter_id", ""),
                                            ("DEV", "proffer_court_case_id", "cafebabe-cafe-babe-cafe-babecafebabe")])
def test_bad_mode_or_case_config_precedes_secret_and_network(monkeypatch, mode, field, value):
    if field:
        monkeypatch.setattr(settings, field, value)
        monkeypatch.setattr(settings, "proffer_real_matter_id", "")
        monkeypatch.setattr(settings, "proffer_real_court_case_id", "")
    monkeypatch.setattr(proffer, "_service_authorization_headers", lambda: pytest.fail("must not read secret"))
    monkeypatch.setattr(case_scope.httpx, "AsyncClient", lambda **_: pytest.fail("must not open client"))
    with pytest.raises(ProfferError):
        asyncio.run(case_scope.verify_case_scope(mode))


@pytest.mark.parametrize("failure", ["redirect", "huge", "duplicate", "timeout", "network", "foreign-parent"])
def test_failed_scope_transport_cannot_dispatch_downstream_mutation(monkeypatch, failure):
    monkeypatch.setattr(case_scope, "_TIMEOUT_SECONDS", 0.08)
    requests, mutations = [], []

    def upstream(request):
        requests.append(request)
        if failure == "network":
            raise httpx.ReadError("private URL " + TOKEN)
        payload = header()
        payload["court_case"]["matter_id"] = "foreign" if failure == "foreign-parent" else MATTER
        raw = b'{"mode":"DEV","mode":"LIVE"}' if failure == "duplicate" else json.dumps(payload).encode()
        stream = Bytes([b"a" * 65537] if failure == "huge" else [raw], delay=0.2 if failure == "timeout" else 0)
        return httpx.Response(302 if failure == "redirect" else 200, stream=stream)

    transport(monkeypatch, upstream)
    app = FastAPI()

    @app.post("/mutate")
    async def mutation(mode: operating_mode.OperatingMode):
        mutations.append(mode)
        return {}

    response = TestClient(app).post("/mutate?mode=LIVE")
    assert response.status_code in {502, 503}
    assert len(requests) == 1 and mutations == []
    assert TOKEN not in response.text and "private URL" not in response.text
