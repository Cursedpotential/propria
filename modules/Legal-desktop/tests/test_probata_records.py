"""Probata read-through checks use synthetic upstream responses, never live evidence."""
from types import SimpleNamespace

import httpx
import pytest

from legal_workspace.config import Settings
from legal_workspace.services import probata_records as reader
from pydantic import ValidationError


@pytest.fixture
def configured(monkeypatch, tmp_path):
    secret = tmp_path / "service-token"
    secret.write_text("synthetic-read-through-test-token-only-1234")
    monkeypatch.setattr(reader, "get_settings", lambda: SimpleNamespace(
        probata_records_base_url="https://probata.invalid", probata_records_token_file=str(secret),
        probata_records_mode="LIVE"))
    return secret


def upstream(record_id="native-entity", version="view-sha256:one", title="Original name", mode="LIVE"):
    return {"available": True, "mode": mode, "records": [{
        "origin": {"system": "probata", "kind": "entity", "record_id": record_id,
                   "record_version": version}, "title": title, "record": {"scope": "registry"}}]}


@pytest.mark.parametrize("input_mode,expected", [
    (None, "LIVE"), ("LIVE", "LIVE"), ("DEV", "DEV"),
    ("REAL", "LIVE"), ("TEST", "DEV"),
])
def test_operating_mode_config_is_canonical(monkeypatch, input_mode, expected):
    monkeypatch.delenv("PROBATA_RECORDS_MODE", raising=False)
    if input_mode is not None:
        monkeypatch.setenv("PROBATA_RECORDS_MODE", input_mode)
    assert Settings(_env_file=None).probata_records_mode == expected


@pytest.mark.parametrize("invalid", ["", "PROD", "test", "LIVE "])
def test_unknown_operating_mode_fails_configuration(monkeypatch, invalid):
    monkeypatch.setenv("PROBATA_RECORDS_MODE", invalid)
    with pytest.raises(ValidationError):
        Settings(_env_file=None)


@pytest.mark.parametrize("mode", ["DEV", "LIVE"])
def test_modes_share_native_case_and_record_identity(configured, monkeypatch, mode):
    monkeypatch.setattr(reader, "get_settings", lambda: SimpleNamespace(
        probata_records_base_url="https://probata.invalid",
        probata_records_token_file=str(configured), probata_records_mode=mode))
    data = upstream(mode=mode)
    data["matter_id"] = "same-matter-id"
    data["court_case_id"] = "same-court-case-id"

    def respond(request):
        assert request.url.params["mode"] == mode
        assert "matter_id" not in request.url.params
        assert "court_case_id" not in request.url.params
        return httpx.Response(200, json=data)

    with httpx.Client(transport=httpx.MockTransport(respond)) as http:
        result = reader.list_records("entity", record_id="native-entity", client=http)
    assert result.available
    assert result.mode == mode
    assert result.matter_id == "same-matter-id"
    assert result.court_case_id == "same-court-case-id"
    assert result.records[0].origin.record_id == "native-entity"


def test_readthrough_keeps_identity_and_reads_changes(configured):
    state = upstream()
    def respond(request):
        assert request.method == "GET"
        assert request.url.path == "/legal-context/records"
        assert request.url.params["mode"] == "LIVE"
        assert request.headers["authorization"].startswith("Bearer synthetic-")
        return httpx.Response(200, json=state)
    with httpx.Client(transport=httpx.MockTransport(respond)) as http:
        first = reader.list_records("entity", record_id="native-entity", client=http)
        state["records"][0]["title"] = "Corrected name"
        state["records"][0]["origin"]["record_version"] = "view-sha256:two"
        second = reader.list_records("entity", record_id="native-entity", client=http)
    assert first.available and second.available
    assert first.records[0].origin.record_id == second.records[0].origin.record_id
    assert first.records[0].origin.record_version != second.records[0].origin.record_version
    assert second.records[0].title == "Corrected name"


@pytest.mark.parametrize("change", ["mode", "identity", "kind", "duplicate", "malformed"])
def test_wrong_scope_or_identity_never_projects_data(configured, change):
    data = upstream()
    if change == "mode": data["mode"] = "DEV"
    elif change == "identity": data["records"][0]["origin"]["record_id"] = "other-id"
    elif change == "kind": data["records"][0]["origin"]["kind"] = "event"
    elif change == "duplicate": data["records"].append(data["records"][0])
    else: data["records"][0].pop("origin")
    with httpx.Client(transport=httpx.MockTransport(lambda _: httpx.Response(200, json=data))) as http:
        result = reader.list_records("entity", record_id="native-entity", client=http)
    assert not result.available and result.records == []


@pytest.mark.parametrize("status", [401, 403, 404, 503])
def test_upstream_failures_remain_unavailable(configured, status):
    with httpx.Client(transport=httpx.MockTransport(lambda _: httpx.Response(status))) as http:
        result = reader.list_records("entity", client=http)
    assert not result.available
    assert result.records == []
    assert "token" not in result.reason.lower()


def test_missing_credentials_do_not_call_upstream(configured):
    configured.rename(configured.with_name("retained-token"))
    def unexpected(_): raise AssertionError("Upstream must not be called without credentials.")
    with httpx.Client(transport=httpx.MockTransport(unexpected)) as http:
        result = reader.list_records("entity", client=http)
    assert not result.available


def test_live_pointer_get_never_returns_synthetic_fallback(monkeypatch):
    monkeypatch.setattr(reader, "list_records", lambda *args, **kwargs: reader.Listing(
        available=False, reason="Unavailable"))
    with pytest.raises(reader.ProbataUnavailable):
        reader.get_record("event", "id")


def test_probata_route_requires_authenticated_actor(monkeypatch):
    from fastapi import FastAPI
    from fastapi.testclient import TestClient
    from legal_workspace.api import claim_routes, probata_record_routes

    app = FastAPI()
    app.include_router(probata_record_routes.router)
    monkeypatch.setattr(probata_record_routes, "list_records", lambda kind, q: reader.Listing(
        available=True, mode="LIVE", records=[]))
    with TestClient(app) as client:
        assert client.get("/v1/probata/records").status_code == 401
        app.dependency_overrides[claim_routes.actor] = lambda: "test:owner"
        assert client.get("/v1/probata/records?kind=event").json()["available"]
        assert client.get("/v1/probata/records?kind=other").status_code == 422


def test_probata_route_forwards_exact_native_identity(monkeypatch):
    from fastapi import FastAPI
    from fastapi.testclient import TestClient
    from legal_workspace.api import claim_routes, probata_record_routes

    calls = []
    def listing(kind, q, **kwargs):
        calls.append((kind, q, kwargs))
        return reader.Listing(available=True, mode="LIVE", records=[])
    monkeypatch.setattr(probata_record_routes, "list_records", listing)
    app = FastAPI()
    app.include_router(probata_record_routes.router)
    app.dependency_overrides[claim_routes.actor] = lambda: "test:owner"
    identity = "a5aaf531-9fb9-4d35-a56e-a41b2a8a2d60"
    with TestClient(app) as client:
        assert client.get(f"/v1/probata/records?kind=entity&record_id={identity}").status_code == 200
        assert calls == [("entity", "", {"record_id": identity})]
        assert client.get("/v1/probata/records?record_id=invalid").status_code == 422
        assert len(calls) == 1


def test_claim_reader_reuses_one_upstream_snapshot_per_kind(monkeypatch, tmp_path):
    from legal_workspace.api import claim_routes
    from legal_workspace.services.workspace import Workspace

    monkeypatch.setattr(claim_routes, "WORKSPACE", Workspace(tmp_path))
    calls = []

    def snapshot(kind):
        calls.append(kind)
        return reader.Listing(available=True, records=[reader.Record.model_validate(upstream()["records"][0])])

    monkeypatch.setattr(reader, "list_records", snapshot)
    service = claim_routes.get_claim_service()
    assert service.record_loader("entity", "native-entity") is not None
    assert service.record_loader("entity", "missing-entity") is None
    assert service.record_loader("entity", "native-entity") is not None
    assert calls == ["entity"]


def test_native_scope_preserved_separately_from_record_identity(configured):
    from uuid import uuid4
    data = upstream()
    data["matter_id"] = str(uuid4())
    data["court_case_id"] = str(uuid4())
    with httpx.Client(transport=httpx.MockTransport(lambda _: httpx.Response(200, json=data))) as http:
        result = reader.list_records("entity", client=http)
    assert result.available
    assert result.matter_id == data["matter_id"]
    assert result.court_case_id == data["court_case_id"]
