"""Synthetic verification of the private authoritative case-header write boundary.

Byline: Codex · GPT-5 · 2026-10-05

Every HTTP transport is stubbed; no database, Temporal or private service is used.
"""

from __future__ import annotations

import json
from types import SimpleNamespace
from urllib.error import HTTPError, URLError

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
    """Stub the entire HTTP transport and record admission and byte-limit ordering."""
    observed = []
    state = SimpleNamespace(
        raw=json.dumps({"mode": "LIVE", "matter": {"id": MATTER}, "court_case": {"id": COURT}}).encode(),
        error=None,
        status=200,
    )

    class Response:
        @property
        def status(self):
            return state.status

        def __enter__(self):
            return self

        def __exit__(self, *_):
            return False

        def read(self, limit):
            observed.append(("read", limit))
            return state.raw[:limit]

    def open_request(request, timeout):
        observed.append(("get", request, timeout))
        if state.error:
            raise state.error
        return Response()

    def headers():
        observed.append("credential")
        return {"Authorization": f"Bearer {TOKEN}"}

    def build(*handlers):
        observed.append(("handlers", handlers))
        return SimpleNamespace(open=open_request)

    monkeypatch.setattr(scope, "_service_authorization_headers", headers)
    monkeypatch.setattr(scope, "build_opener", build)
    return observed, state


def test_each_live_admission_requires_a_fresh_authenticated_bounded_get(transport):
    observed, _ = transport
    for _ in range(2):
        assert scope.require_authoritative_live_case_scope("LIVE", MATTER, COURT) == (MATTER, COURT)
    gets = [item for item in observed if isinstance(item, tuple) and item[0] == "get"]
    assert len(gets) == 2
    request = gets[0][1]
    assert request.full_url == "http://synthetic-starter.invalid:8089/case-identity?mode=LIVE"
    assert request.get_method() == "GET" and request.data is None
    assert request.get_header("Authorization") == f"Bearer {TOKEN}"
    assert 0 < gets[0][2] <= 5
    assert ("read", scope._MAX_RESPONSE_BYTES + 1) in observed
    handlers = next(item[1] for item in observed if isinstance(item, tuple) and item[0] == "handlers")
    assert handlers[0].proxies == {}
    assert handlers[1].redirect_request(None, None, 302, "redirect", {}, "https://foreign.invalid") is None


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
        json.dumps({"mode": "REAL", "matter": {"id": MATTER}, "court_case": {"id": COURT}}).encode(),
        json.dumps({"mode": "DEV", "matter": {"id": MATTER}, "court_case": {"id": COURT}}).encode(),
        json.dumps({"mode": "LIVE", "matter": {"id": OTHER}, "court_case": {"id": COURT}}).encode(),
        json.dumps({"mode": "LIVE", "matter": {"id": MATTER}, "court_case": {"id": OTHER}}).encode(),
        json.dumps({"mode": "LIVE", "matter": None, "court_case": {"id": COURT}}).encode(),
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


@pytest.mark.parametrize(
    "error",
    [TimeoutError(TOKEN), URLError(TOKEN), OSError(TOKEN), HTTPError("http://foreign.invalid", 302, TOKEN, {}, None)],
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
