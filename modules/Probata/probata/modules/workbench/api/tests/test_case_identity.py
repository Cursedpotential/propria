"""Case page BFF: registry pass-through with actor + key, catalog counts labeled by store.

Byline: Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02
"""

from __future__ import annotations

import httpx
import pytest
from app.config import settings
from app.repo import case_identity_catalog as catalog
from app.runtime import case_identity as runtime
from app.runtime import operating_mode
from app.service import proffer
from app.service.proffer_errors import ProfferError
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

MATT = "01a0f751-e07b-76b6-afcb-63acfbba373e"


def _app(actor: bool = True) -> FastAPI:
    app = FastAPI()

    @app.middleware("http")
    async def identity(request: Request, call_next):
        if actor:
            request.state.subject_uid = "subject-1"
            request.state.principal = "matt"
        return await call_next(request)

    app.include_router(runtime.router)
    return app


def _view(mode: str = "LIVE") -> dict:
    return {
        "mode": mode,
        "matter": {"id": "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", "title": "Salem v Kinzel"},
        "court_case": {"id": "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", "caption": "Matthew S. Salem v Katrina Kinzel"},
        "people": [
            {
                "id": MATT,
                "display_name": "Matthew S. Salem",
                "identifiers": [
                    {"id": "a1", "normalized": "8102959302", "status": "confirmed"},
                    {"id": "a2", "normalized": "8102751930", "status": "retired"},
                ],
            }
        ],
        "history": [],
        "probata_counts": [],
        "probata_unknowns": [],
        "dismissed": [{"normalized": "34428"}],
        "count_store": "probata",
    }


class _Engine:
    def __init__(self):
        self.calls: list[dict] = []
        self.answers: dict[tuple[str, str], tuple[int, object]] = {}

    async def request(self, method: str, path: str, **kwargs):
        self.calls.append({"method": method, "path": path, **kwargs})
        status, payload = self.answers[(method, path)]
        if status >= 400:
            raise ProfferError(str(payload), status)
        return httpx.Response(status, json=payload)


@pytest.fixture
def engine(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", _view()["matter"]["id"])
    monkeypatch.setattr(settings, "proffer_court_case_id", _view()["court_case"]["id"])
    async def verified_scope(_mode):
        return None
    monkeypatch.setattr(operating_mode, "verify_case_scope", verified_scope)
    stub = _Engine()
    monkeypatch.setattr(proffer, "_request", stub.request)
    return stub


@pytest.fixture
def catalog_calls(monkeypatch):
    calls: dict[str, list] = {"counts": [], "unknowns": []}
    monkeypatch.setattr(catalog, "configured", lambda: True)

    def counts(identifiers):
        calls["counts"].append(identifiers)
        return [{"identifier": "8102959302", "match_on": "counterparty_phone", "event_kind": "message", "events": 23032}]

    def unknowns(known, *, kind="phone", limit=50):
        calls["unknowns"].append((tuple(known), kind))
        return []

    monkeypatch.setattr(catalog, "counts", counts)
    monkeypatch.setattr(catalog, "unknowns", unknowns)
    return calls


def test_read_merges_registry_and_catalog_with_store_labels(engine, catalog_calls) -> None:
    engine.answers[("GET", "/case-identity")] = (200, _view())
    response = TestClient(_app()).get("/api/case-identity", params={"mode": "LIVE"})
    assert response.status_code == 200, response.text
    body = response.json()
    assert engine.calls[0]["params"] == {"mode": "LIVE"}
    assert body["count_store"] == "probata"
    assert body["catalog"]["store"] == "casebible" and body["catalog"]["available"] is True
    assert body["catalog"]["counts"][0]["events"] == 23032
    # Counts are asked for every identifier, retired included; the unknowns queue
    # leaves out what registry carries (not retired) and what the owner dismissed.
    assert sorted(catalog_calls["counts"][0]) == ["8102751930", "8102959302"]
    assert catalog_calls["unknowns"][0] == (("34428", "8102959302"), "phone")


def test_read_rejects_an_engine_answer_for_another_mode(engine, catalog_calls) -> None:
    engine.answers[("GET", "/case-identity")] = (200, _view("DEV"))
    response = TestClient(_app()).get("/api/case-identity", params={"mode": "LIVE"})
    assert response.status_code == 502


def test_read_survives_an_unavailable_catalog(engine, monkeypatch) -> None:
    engine.answers[("GET", "/case-identity")] = (200, _view())
    monkeypatch.setattr(catalog, "configured", lambda: True)

    def broken(_):
        raise catalog.CatalogError("Pre-ingest catalog query unavailable or timed out")

    monkeypatch.setattr(catalog, "counts", broken)
    body = TestClient(_app()).get("/api/case-identity", params={"mode": "LIVE"}).json()
    assert body["catalog"]["available"] is False
    assert body["people"][0]["display_name"] == "Matthew S. Salem"


def test_identifier_write_forwards_actor_and_key(engine) -> None:
    engine.answers[("POST", "/case-identity/identifiers")] = (
        201,
        {"ref": "r1", "kind": "registry.entity_alias", "recorded_at": "2026-10-01T12:00:00Z", "replayed": False},
    )
    body = {"entity_id": MATT, "raw_value": "810-252-2779", "kind": "phone", "status": "confirmed",
            "basis": "her phone saves it as Matthew Salem", "change_reason": "owner confirmed"}
    response = TestClient(_app()).post("/api/case-identity/identifiers", json=body, headers={"Idempotency-Key": "k-1"})
    assert response.status_code == 201, response.text
    call = engine.calls[0]
    assert call["json"] == body
    assert call["headers"] == {"X-authentik-uid": "subject-1", "X-authentik-username": "matt", "Idempotency-Key": "k-1"}


def test_identifier_edit_and_delete_reach_their_engine_routes(engine) -> None:
    alias = "01a0f751-e07b-7000-8000-00000000a001"
    receipt = {"ref": "c1", "kind": "registry.identity_change", "recorded_at": "2026-10-02T06:00:00Z"}
    engine.answers[("POST", f"/case-identity/identifiers/{alias}")] = (201, receipt)
    engine.answers[("POST", f"/case-identity/identifiers/{alias}/delete")] = (201, receipt)
    client = TestClient(_app())
    edit = {"fields": {"kind": "legal"}, "change_reason": "the caption names her"}
    assert client.post(f"/api/case-identity/identifiers/{alias}", json=edit, headers={"Idempotency-Key": "e"}).status_code == 201
    removal = {"change_reason": "typed into the wrong person"}
    response = client.post(f"/api/case-identity/identifiers/{alias}/delete", json=removal, headers={"Idempotency-Key": "d"})
    assert response.status_code == 201 and response.json()["replayed"] is False
    assert [call["json"] for call in engine.calls] == [edit, removal]
    bad = client.post("/api/case-identity/identifiers/not-a-uuid/delete", json=removal, headers={"Idempotency-Key": "d2"})
    assert bad.status_code == 422


def test_writes_need_an_actor_and_a_key(engine) -> None:
    no_key = TestClient(_app()).post("/api/case-identity/triage", json={"raw_value": "34428"})
    assert no_key.status_code == 422
    no_actor = TestClient(_app(actor=False)).post(
        "/api/case-identity/triage", json={"raw_value": "34428"}, headers={"Idempotency-Key": "k"}
    )
    assert no_actor.status_code == 401
    assert engine.calls == []


def test_engine_errors_keep_their_status(engine) -> None:
    engine.answers[("POST", "/case-identity/header")] = (409, "the record changed since it was read")
    response = TestClient(_app()).post(
        "/api/case-identity/header", params={"mode": "LIVE"}, json={"target": "matter"}, headers={"Idempotency-Key": "k"}
    )
    assert response.status_code == 409
    assert engine.calls[0]["params"] == {"mode": "LIVE"}


def test_person_edit_path_is_a_uuid(engine) -> None:
    response = TestClient(_app()).post(
        "/api/case-identity/people/../../reference-import", json={}, headers={"Idempotency-Key": "k"}
    )
    assert response.status_code in (404, 422)
    assert engine.calls == []


def test_lookup_passes_every_value(engine) -> None:
    engine.answers[("GET", "/case-identity/lookup")] = (200, {"matches": [], "store": "probata.registry"})
    response = TestClient(_app()).get("/api/case-identity/lookup", params=[("value", "8102689630"), ("value", "Me")])
    assert response.status_code == 200
    assert engine.calls[0]["params"] == [("value", "8102689630"), ("value", "Me")]


def test_catalog_events_rejects_an_unknown_match(monkeypatch) -> None:
    response = TestClient(_app()).get(
        "/api/case-identity/catalog-events", params={"identifier": "8102959302", "match_on": "body"}
    )
    assert response.status_code == 422


def test_router_is_mounted_on_the_application() -> None:
    from main import app

    paths = set(app.openapi()["paths"])
    assert {"/api/case-identity", "/api/case-identity/identifiers", "/api/case-identity/lookup"} <= paths



# Byline: Claude Code · Sonnet · 2026-10-02
def test_placeholders_and_merge_pass_through_with_actor_and_key(engine) -> None:
    receipt = {"ref": "k", "kind": "registry.placeholders", "recorded_at": "2026-10-02T00:00:00Z", "replayed": False,
               "detail": {"created": 1, "dry_run": True}}
    engine.answers[("POST", "/case-identity/placeholders")] = (201, receipt)
    engine.answers[("POST", f"/case-identity/people/{MATT}/merge")] = (201, {**receipt, "kind": "registry.identity_change", "detail": {"rows_moved": {}}})
    client = TestClient(_app())
    batch = {"numbers": ["8105550142"], "change_reason": "seen in calls", "dry_run": True}
    response = client.post("/api/case-identity/placeholders", json=batch, headers={"Idempotency-Key": "p1"})
    assert response.status_code == 201 and response.json()["detail"]["created"] == 1
    merge = {"into_id": "01a0f751-e07b-76c7-8c0f-65692ad656b8", "change_reason": "her other phone"}
    assert client.post(f"/api/case-identity/people/{MATT}/merge", json=merge, headers={"Idempotency-Key": "m1"}).status_code == 201
    assert [call["json"] for call in engine.calls] == [batch, merge]
    assert client.post("/api/case-identity/people/not-a-uuid/merge", json=merge, headers={"Idempotency-Key": "m2"}).status_code == 422
    assert TestClient(_app(actor=False)).post("/api/case-identity/placeholders", json=batch, headers={"Idempotency-Key": "p2"}).status_code == 401
