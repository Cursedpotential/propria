"""Synthetic verification of the private authoritative case-header write boundary.

Byline: Codex · GPT-5 · 2026-10-05
Updated: Codex · gpt-6.1-sol · 2026-10-06 — total deadline, streaming cleanup and caller admission.

Every HTTP transport is stubbed; no database, Temporal or private service is used.
"""

from __future__ import annotations

import asyncio
import builtins
import json
import time
from types import SimpleNamespace

import httpx
import pytest

from server.case_management import authoritative_case_scope as scope

MATTER = "66666666-6666-4666-8666-666666666666"
COURT = "77777777-7777-4777-8777-777777777777"
OTHER = "88888888-8888-4888-8888-888888888888"
TOKEN = "synthetic-only-service-token-1234567890"


@pytest.fixture
def configured(monkeypatch):
    monkeypatch.setenv("PROFFER_MATTER_ID", MATTER)
    monkeypatch.setenv("PROFFER_COURT_CASE_ID", COURT)
    monkeypatch.setenv("PROFFER_STARTER_URL", "http://synthetic-starter.invalid:8089")


@pytest.fixture
def transport(monkeypatch, configured):
    """Use HTTPX's real stream lifecycle with a synthetic transport, never sockets."""
    observed = []
    state = SimpleNamespace(
        raw=json.dumps(
            {"mode": "LIVE", "matter": {"id": MATTER}, "court_case": {"id": COURT, "matter_id": MATTER}}
        ).encode(),
        error=None,
        status=200,
        chunks=None,
        delay=0,
        header_delay=0,
        closed=0,
        client_closed=0,
        cancelled=0,
        yielded=0,
    )

    class Stream(httpx.AsyncByteStream):
        async def __aiter__(self):
            try:
                for chunk in state.chunks if state.chunks is not None else [state.raw]:
                    await asyncio.sleep(state.delay)
                    state.yielded += 1
                    yield chunk
            except asyncio.CancelledError:
                state.cancelled += 1
                raise

        async def aclose(self):
            state.closed += 1

    async def open_request(request):
        observed.append(("get", request))
        await asyncio.sleep(state.header_delay)
        if state.error:
            raise state.error
        return httpx.Response(state.status, stream=Stream(), headers={"Location": "https://foreign.invalid/secret"})

    def headers():
        observed.append("credential")
        return {"Authorization": f"Bearer {TOKEN}"}

    real_client = httpx.AsyncClient

    class Client(real_client):
        async def __aexit__(self, *args):
            await super().__aexit__(*args)
            assert self.is_closed
            state.client_closed += 1

    def client(**options):
        observed.append(("options", options))
        return Client(transport=httpx.MockTransport(open_request), **options)

    monkeypatch.setattr(scope, "_service_authorization_headers", headers)
    monkeypatch.setattr(httpx, "AsyncClient", client)
    return observed, state


def test_each_live_admission_requires_a_fresh_authenticated_bounded_get(transport):
    observed, state = transport
    for _ in range(2):
        assert scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT) == (MATTER, COURT)
    gets = [item for item in observed if isinstance(item, tuple) and item[0] == "get"]
    assert len(gets) == 2
    request = gets[0][1]
    assert str(request.url) == "http://synthetic-starter.invalid:8089/case-identity/scope?mode=LIVE"
    assert request.method == "GET" and request.content == b""
    assert request.headers["Authorization"] == f"Bearer {TOKEN}"
    options = next(item[1] for item in observed if isinstance(item, tuple) and item[0] == "options")
    assert 0 < options["timeout"] <= 5
    assert options["trust_env"] is False and options["follow_redirects"] is False
    assert state.closed == 2 and state.client_closed == 2


def test_drip_response_hits_total_deadline_and_closes_stream(monkeypatch, transport):
    """An active stream cannot extend admission by continually resetting inactivity."""
    _, state = transport
    monkeypatch.setattr(scope, "_TIMEOUT_SECONDS", 0.05)
    state.chunks, state.delay = [b" "] * 1000, 0.005
    started = time.monotonic()
    with pytest.raises(scope.CaseScopeVerificationError) as denied:
        asyncio.run(scope.require_authoritative_live_case_scope_async("LIVE", MATTER, COURT))
    assert denied.value.http_status == 503 and str(denied.value) == scope._UPSTREAM_ERROR
    assert time.monotonic() - started < 1
    assert 0 < state.yielded < 1000 and state.cancelled == 1 and state.closed == 1
    assert state.client_closed == 1


def test_waiting_for_response_headers_is_in_the_total_deadline(monkeypatch, transport):
    _, state = transport
    monkeypatch.setattr(scope, "_TIMEOUT_SECONDS", 0.03)
    state.header_delay = 1
    with pytest.raises(scope.CaseScopeVerificationError, match=scope._UPSTREAM_ERROR):
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert state.yielded == 0
    assert state.client_closed == 1


@pytest.mark.parametrize("extra", [0, 1])
def test_stream_byte_boundary_and_early_close(transport, extra):
    _, state = transport
    state.chunks = [state.raw, b" " * (scope._MAX_RESPONSE_BYTES - len(state.raw) + extra), b"unused"]
    if extra:
        with pytest.raises(scope.CaseScopeVerificationError) as denied:
            scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
        assert denied.value.http_status == 502 and state.yielded == 2
    else:
        state.chunks.pop()
        assert scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT) == (MATTER, COURT)
    assert state.closed == 1
    assert state.client_closed == 1


def test_redirect_never_reaches_foreign_host_or_uses_environment_proxy(monkeypatch, transport):
    observed, state = transport
    monkeypatch.setenv("HTTP_PROXY", "http://foreign-proxy.invalid:8080")
    monkeypatch.setenv("HTTPS_PROXY", "http://foreign-proxy.invalid:8080")
    state.status = 302
    with pytest.raises(scope.CaseScopeVerificationError) as denied:
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert denied.value.http_status == 503 and state.closed == 1
    gets = [item for item in observed if isinstance(item, tuple) and item[0] == "get"]
    assert len(gets) == 1 and gets[0][1].url.host == "synthetic-starter.invalid"
    assert next(item[1] for item in observed if isinstance(item, tuple) and item[0] == "options")["trust_env"] is False


def test_external_cancellation_is_not_converted_to_approval_or_error(monkeypatch, transport):
    _, state = transport
    state.chunks, state.delay = [b" "] * 1000, 0.005

    async def cancel_admission():
        task = asyncio.create_task(scope.require_authoritative_live_case_scope_async("LIVE", MATTER, COURT))
        await asyncio.sleep(0.025)
        task.cancel()
        with pytest.raises(asyncio.CancelledError):
            await task

    asyncio.run(cancel_admission())
    assert state.cancelled == 1 and state.closed == 1
    assert state.client_closed == 1


def test_denied_mode_and_invalid_secret_do_not_import_httpx(monkeypatch, configured):
    real_import = builtins.__import__

    def fence(name, *args, **kwargs):
        if name == "httpx":
            pytest.fail("denied admission imported HTTPX")
        return real_import(name, *args, **kwargs)

    monkeypatch.setattr(builtins, "__import__", fence)
    monkeypatch.setenv("PROFFER_SERVICE_TOKEN_FILE", "relative-token")
    for mode in ("DEV", "LIVE"):
        with pytest.raises(scope.CaseScopeVerificationError):
            scope.require_authoritative_live_case_scope(mode, MATTER, COURT)


def test_sync_adapter_rejects_running_loop_without_detached_work(transport):
    observed, _ = transport

    async def nested():
        with pytest.raises(scope.CaseScopeVerificationError):
            scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)

    asyncio.run(nested())
    assert observed == []


@pytest.mark.parametrize("caller", ["api", "cli", "publish", "call-log", "remove"])
def test_actual_drip_transport_denies_callers_before_sql_body_or_dispatch(monkeypatch, transport, caller):
    """Exercise each real adapter against cancellation, with all mutation effects fenced."""
    _, state = transport
    monkeypatch.setattr(scope, "_TIMEOUT_SECONDS", 0.04)
    state.chunks, state.delay = [b" "] * 1000, 0.005
    if caller == "api":
        import hashlib

        from fastapi import FastAPI
        from fastapi.testclient import TestClient

        from server.api import inspect_routes

        app = FastAPI()
        inspect_routes.register_inspect_routes(app, knowledge=None)
        monkeypatch.setattr(inspect_routes, "_verify_proffer_delegation", lambda *_: None)
        monkeypatch.setattr(inspect_routes, "_get_engine", lambda: pytest.fail("denied admission accessed SQL"))
        body = {
            "preview_handle": "preview_handle_abcdefghijklmnopqrstuvwxyz",
            "matter_mode": "LIVE",
            "scope": "record",
            "target_id": OTHER,
            "attempt_id": OTHER,
            "actor_subject_uid": "synthetic-subject",
            "actor_username": "synthetic-operator",
            "claim": "Synthetic review only",
        }
        body["idempotency_key"] = hashlib.sha256(
            json.dumps(body, sort_keys=True, separators=(",", ":")).encode()
        ).hexdigest()
        with TestClient(app) as client:
            response = client.post("/v1/flags/proffer-potential-promotion", json=body)
        assert response.status_code == 503 and response.json()["detail"] == scope._UPSTREAM_ERROR
    else:
        from temporalio.exceptions import ApplicationError

        from server.context_chunks import start as starter
        from server.temporal import chunk_activities as chunks
        from server.temporal import chunk_backfill_activities as backfill

        real_import = builtins.__import__

        def fence(name, *args, **kwargs):
            if name in (
                "temporalio.client",
                "server.config",
                "server.db",
                "server.context_chunks.embed",
                "server.context_chunks.store",
            ):
                pytest.fail(f"denied admission imported mutation dependency {name}")
            return real_import(name, *args, **kwargs)

        monkeypatch.setattr(builtins, "__import__", fence)
        policy = {"operating_mode": "LIVE", "matter_id": MATTER, "court_case_id": COURT}
        with pytest.raises(ApplicationError) as denied:
            if caller == "cli":
                asyncio.run(starter.start(starter.parser().parse_args(["rechunk", "--no-wait"])))
            elif caller == "publish":
                chunks.publish_context_chunks_activity(chunks.PublishChunksParams(**policy))
            elif caller == "call-log":
                chunks.publish_call_log_files_activity(chunks.PublishCallLogFilesParams(**policy))
            else:
                backfill.remove_per_message_objects_activity(backfill.RemovalParams(**policy, dry_run=False))
        assert denied.value.type == "ChunkWriteDenied" and denied.value.non_retryable
    assert state.cancelled == 1 and state.closed == 1
    assert state.client_closed == 1


@pytest.mark.parametrize("mode", [None, "", "DEV", "TEST", "REAL", "unknown"])
def test_denied_modes_do_no_credential_or_header_io(transport, mode):
    observed, _ = transport
    with pytest.raises(scope.CaseScopeVerificationError):
        scope.require_authoritative_live_case_scope(mode, MATTER, COURT)
    assert observed == []


@pytest.mark.parametrize("field", ["PROFFER_MATTER_ID", "PROFFER_COURT_CASE_ID"])
@pytest.mark.parametrize(
    "value",
    [
        "",
        "invalid",
        "00000000-0000-0000-0000-000000000000",
        "deadbeef-dead-beef-dead-beefdeadbeef",
        "cafebabe-cafe-babe-cafe-babecafebabe",
        OTHER,
    ],
)
def test_invalid_or_mismatching_config_denies_before_all_io(monkeypatch, transport, field, value):
    observed, _ = transport
    monkeypatch.setenv(field, value)
    with pytest.raises(scope.CaseScopeVerificationError):
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert observed == []


@pytest.mark.parametrize("matter,court", [(OTHER, COURT), (MATTER, OTHER), ("bad", COURT), (MATTER, "")])
def test_local_request_mismatch_denies_before_all_io(transport, matter, court):
    observed, _ = transport
    with pytest.raises(scope.CaseScopeVerificationError):
        scope.require_authoritative_live_case_scope("LIVE", matter, court)
    assert observed == []


@pytest.mark.parametrize(
    "url",
    [
        "",
        "starter",
        "file:///secret",
        "http://",
        "http://user:secret@starter",
        "http://starter?",
        "http://starter?mode=LIVE",
        "http://starter#",
        "http://starter#x",
        "http://starter\n",
        "http://starter:99999",
        "http://starter\\foreign",
        "http://%73tarter",
    ],
)
def test_invalid_url_denies_before_credential_io(monkeypatch, transport, url):
    observed, _ = transport
    monkeypatch.setenv("PROFFER_STARTER_URL", url)
    with pytest.raises(scope.CaseScopeVerificationError):
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert observed == []


@pytest.mark.parametrize(
    "raw",
    [
        b"",
        b"not-json",
        b"null",
        b"[]",
        b"{}",
        b"\xff",
        b'{"mode":"LIVE","mode":"DEV"}',
        b"x" * (scope._MAX_RESPONSE_BYTES + 1),
        json.dumps(
            {"mode": "REAL", "matter": {"id": MATTER}, "court_case": {"id": COURT, "matter_id": MATTER}}
        ).encode(),
        json.dumps(
            {"mode": "DEV", "matter": {"id": MATTER}, "court_case": {"id": COURT, "matter_id": MATTER}}
        ).encode(),
        json.dumps(
            {"mode": "LIVE", "matter": {"id": OTHER}, "court_case": {"id": COURT, "matter_id": MATTER}}
        ).encode(),
        json.dumps(
            {"mode": "LIVE", "matter": {"id": MATTER}, "court_case": {"id": OTHER, "matter_id": MATTER}}
        ).encode(),
        json.dumps({"mode": "LIVE", "matter": None, "court_case": {"id": COURT, "matter_id": MATTER}}).encode(),
    ],
    ids=lambda value: f"payload-{len(value)}-bytes",
)
def test_malformed_or_foreign_header_fails_closed_without_payload_exposure(transport, raw):
    _, state = transport
    state.raw = raw
    with pytest.raises(scope.CaseScopeVerificationError) as error:
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert error.value.http_status == 502
    assert str(error.value) == scope._HEADER_ERROR
    assert state.closed == 1 and state.client_closed == 1


@pytest.mark.parametrize(
    "error",
    [TimeoutError(TOKEN), httpx.ConnectError(TOKEN), httpx.ReadTimeout(TOKEN), httpx.InvalidURL(TOKEN), OSError(TOKEN)],
)
def test_upstream_failure_has_only_safe_error_and_no_positive_cache(transport, error):
    observed, state = transport
    scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    state.error = error
    with pytest.raises(scope.CaseScopeVerificationError) as denied:
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert denied.value.http_status == 503 and TOKEN not in str(denied.value)
    assert sum(isinstance(item, tuple) and item[0] == "get" for item in observed) == 2


@pytest.mark.parametrize(
    "court_matter_id",
    [
        "missing",
        None,
        "",
        "not-a-uuid",
        OTHER,
        "00000000-0000-0000-0000-000000000000",
        "deadbeef-dead-beef-dead-beefdeadbeef",
    ],
)
def test_correct_scope_ids_require_the_court_to_belong_to_the_approved_matter(transport, court_matter_id):
    """Reject missing, malformed or foreign court-parent correlation from the scope endpoint."""
    _, state = transport
    header = {"mode": "LIVE", "matter": {"id": MATTER}, "court_case": {"id": COURT}}
    if court_matter_id != "missing":
        header["court_case"]["matter_id"] = court_matter_id
    state.raw = json.dumps(header).encode()
    with pytest.raises(scope.CaseScopeVerificationError) as error:
        scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT)
    assert error.value.http_status == 502
    assert str(error.value) == scope._HEADER_ERROR
    assert state.closed == 1 and state.client_closed == 1


@pytest.mark.parametrize(
    "raw",
    [
        b"",
        b"x" * 31,
        b"x" * 4097,
        b"x" * 4099,
        b"x" * 32 + b"\x00",
        b"x" * 32 + b"\nembedded",
        b"x" * 32 + b" ",
        b"x" * 32 + b"\xff",
    ],
)
def test_token_file_size_and_content_are_bounded(monkeypatch, tmp_path, raw):
    path = tmp_path / "token"
    path.write_bytes(raw)
    monkeypatch.setenv("PROFFER_SERVICE_TOKEN_FILE", str(path))
    with pytest.raises(scope.CaseScopeVerificationError) as error:
        scope._service_authorization_headers()
    assert str(error.value) == scope._AUTH_ERROR


@pytest.mark.parametrize("ending", [b"", b"\n", b"\r\n"])
def test_token_file_validates_absolute_regular_file_and_strips_crlf(monkeypatch, tmp_path, ending):
    path = tmp_path / "token"
    path.write_bytes(TOKEN.encode() + ending)
    monkeypatch.setenv("PROFFER_SERVICE_TOKEN_FILE", str(path))
    assert scope._service_authorization_headers() == {"Authorization": f"Bearer {TOKEN}"}


@pytest.mark.parametrize("path_kind", ["missing", "directory", "relative"])
def test_token_path_must_be_absolute_existing_regular_file(monkeypatch, tmp_path, path_kind):
    path = {"missing": str(tmp_path / "absent"), "directory": str(tmp_path), "relative": "token"}[path_kind]
    monkeypatch.setenv("PROFFER_SERVICE_TOKEN_FILE", path)
    with pytest.raises(scope.CaseScopeVerificationError):
        scope._service_authorization_headers()
