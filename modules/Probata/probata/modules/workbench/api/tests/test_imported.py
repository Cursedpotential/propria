"""Mobile Imported view: opaque ids, grouping, run-state labels, read-only routes.

Byline: Claude Code · Sonnet · 2026-10-02
"""

from __future__ import annotations

from datetime import datetime, timezone

import pytest
from app.repo import imported_pg as pg
from app.runtime import imported as runtime
from app.service import imported as service
from fastapi import FastAPI
from fastapi.testclient import TestClient

MATTER = "01a0f751-e07b-75cc-9ad5-63ad9449a8ba"
KEY = "b2://salem-data/consignatio/casevault/SourceCorpus/messaging/sms-backup-restore/8102689630/sms-2024-11-24.xml"
PEOPLE = [
    {"entity_id": "e-matt", "person": "Matt", "display_name": "Matthew S. Salem", "role_in_case": "user", "verification_state": "confirmed", "identifier": "8103535467", "kind": "phone"},
    {"entity_id": "e-kat", "person": "Katrina", "display_name": "Katrina Kinzel", "role_in_case": "co_parent", "verification_state": "confirmed", "identifier": "8102689630", "kind": "phone"},
    {"entity_id": "e-kat", "person": "Katrina", "display_name": "Katrina Kinzel", "role_in_case": "co_parent", "verification_state": "confirmed", "identifier": "katrina", "kind": "name"},
    {"entity_id": "e-ph", "person": "Unknown 313-555-0101", "display_name": "Unknown 313-555-0101", "role_in_case": "unknown", "verification_state": "proposed", "identifier": "3135550101", "kind": "phone"},
]


@pytest.fixture(autouse=True)
def _fresh(monkeypatch):
    service._cache.clear()
    monkeypatch.setattr(service.settings, "proffer_real_matter_id", MATTER)
    monkeypatch.setattr(pg, "people", lambda: PEOPLE)


def _app() -> FastAPI:
    app = FastAPI()
    app.include_router(runtime.router)
    return app


def test_opaque_ids_round_trip_and_reject_garbage():
    token = service.encode_id(KEY, "8103099590")
    assert service.decode_id(token, 2) == [KEY, "8103099590"]
    with pytest.raises(service.ImportedError) as error:
        service.decode_id("not-a-real-token", 2)
    assert error.value.status == 404


def test_cursor_rejects_a_forged_value():
    ts = datetime(2026, 1, 1, tzinfo=timezone.utc)
    cursor = service._cursor_encode(ts, "01a0fd2b-02bd-78ea-874e-dc97c6edd63b")
    assert service._cursor_decode(cursor) == (ts.isoformat(), "01a0fd2b-02bd-78ea-874e-dc97c6edd63b")
    with pytest.raises(service.ImportedError):
        service._cursor_decode(service.encode_id("not a time", "x"))


def test_export_description_names_format_device_and_owner():
    people = service._People(PEOPLE)
    sms = service.describe_export(KEY, people)
    assert (sms["format"], sms["device"], sms["owner"], sms["file_name"]) == ("SMS", "(810) 268-9630", "Katrina", "sms-2024-11-24.xml")
    calls = service.describe_export(KEY.replace("messaging/sms-backup-restore", "telephony/sms-backup-restore-calls"), people)
    assert calls["format"] == "Calls"
    fb = service.describe_export("b2://salem-data/consignatio/casevault/SourceCorpus/messaging/facebook-messenger/meta-1/katrinakinzel_1/message_1.json", people)
    assert (fb["format"], fb["owner"], fb["device"]) == ("Facebook", "Katrina", None)


def test_participants_mark_the_users_own_side():
    people = service._People(PEOPLE)
    me = service._participant("self", people, "Matt")
    assert (me["label"], me["mine"]) == ("Matthew S. Salem", True)
    her = service._participant("+18102689630", people, "Matt")
    assert (her["label"], her["mine"]) == ("Katrina Kinzel", False)
    stranger = service._participant("+13135550177", people, "Matt")
    assert stranger["label"] == "(313) 555-0177" and stranger["person"] is None
    assert stranger["number"] == "3135550177"


def test_a_placeholder_is_not_a_confirmed_person_and_offers_who_is_this():
    people = service._People(PEOPLE)
    placeholder = service._participant("+13135550101", people, "Matt")
    assert placeholder["placeholder"] is True and placeholder["entity_id"] == "e-ph"
    assert placeholder["label"] == "Unknown 313-555-0101" and placeholder["number"] == "3135550101"
    nobody = service._participant("+13135550199", people, "Matt")
    assert nobody["placeholder"] is False and nobody["entity_id"] is None and nobody["number"] == "3135550199"
    named = service._participant("+18102689630", people, "Matt")
    assert named["placeholder"] is False and named["number"] is None
    assert "Unknown 313-555-0101" not in people.by_person


def test_status_prefers_the_committed_fact_and_never_guesses_without_run_state():
    row = {"id": "a", "approved": False, "rejected": False}
    assert service._version_status({**row, "approved": True}, None) == "committed"
    assert service._version_status(row, None) == "not_finished"
    assert service._version_status(row, {"a": "failed"}) == "failed"
    assert service._version_status(row, {"a": "awaiting_repair_decision"}) == "parked"
    assert service._version_status(row, {"a": "awaiting_preview_decision"}) == "awaiting_review"


def _version(i, key, *, raw, norm, approved, msgs=None, calls=0):
    ts = datetime(2026, 10, 2, 12, i, tzinfo=timezone.utc)
    return {"id": i and f"v{i}", "source_key": key, "export_key": key.split(".derived/")[0], "acquired_at": ts,
            "raw_n": raw, "norm_n": norm, "msgs": norm if msgs is None else msgs, "calls": calls,
            "first_at": ts, "last_at": ts, "approved": approved, "rejected": False}


def test_sources_group_derived_files_under_their_export(monkeypatch):
    rows = [
        _version(2, KEY + ".derived/threads/8103099590.0001.ndjson", raw=4, norm=4, approved=True),
        _version(1, KEY + ".derived/threads/8102595720.0001.ndjson", raw=3, norm=3, approved=False),
    ]
    monkeypatch.setattr(pg, "source_versions", lambda matter: rows)

    async def lifecycles():
        return {"v1": "failed"}

    monkeypatch.setattr(service, "_lifecycles", lifecycles)
    client = TestClient(_app())
    body = client.get("/api/imported/sources").json()
    assert body["total"] == 1
    item = body["items"][0]
    assert (item["files"], item["raw"], item["normalized"], item["committed"]) == (2, 7, 7, 4)
    assert item["status"] == "failed" and item["status_counts"] == {"committed": 1, "failed": 1}
    assert item["format"] == "SMS" and item["owner"] == "Katrina"


def test_unknown_numbers_list_most_frequent_first(monkeypatch):
    more = PEOPLE + [{"entity_id": "e-ph2", "person": "Unknown 313-555-0102", "display_name": "Unknown 313-555-0102",
                      "role_in_case": "unknown", "verification_state": "proposed", "identifier": "3135550102", "kind": "phone"}]
    monkeypatch.setattr(pg, "people", lambda: more)
    monkeypatch.setattr(pg, "entity_activity", lambda: [
        {"entity_id": "e-ph", "calls": 0, "msgs": 3, "last_at": None},
        {"entity_id": "e-ph2", "calls": 9, "msgs": 2, "last_at": None},
    ])
    body = TestClient(_app()).get("/api/imported/unknown-numbers").json()
    assert [item["number"] for item in body["items"]] == ["3135550102", "3135550101"]
    assert body["items"][0]["total"] == 11 and body["total"] == 2


def test_unlinked_numbers_skip_anyone_the_registry_carries(monkeypatch):
    monkeypatch.setattr(pg, "working_unlinked_numbers", lambda: [{"number": "4195550123", "n": 7}, {"number": "2485550000", "n": 1}])
    body = TestClient(_app()).get("/api/imported/unlinked-numbers").json()
    assert [(i["number"], i["total"]) for i in body["items"]] == [("4195550123", 7), ("2485550000", 1)]


def test_the_view_has_no_write_route():
    methods = {method for route in _app().routes for method in getattr(route, "methods", set())}
    assert methods <= {"GET", "HEAD"}


def test_search_failure_is_a_clean_503(monkeypatch):
    class Boom:
        def __init__(self, *a, **k): ...
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False
        async def post(self, *a, **k):
            raise service.httpx.ConnectError("down")

    monkeypatch.setattr(service.httpx, "AsyncClient", Boom)
    response = TestClient(_app()).get("/api/imported/search", params={"q": "title"})
    assert response.status_code == 503 and "unavailable" in response.json()["detail"]


def test_number_status_tells_named_placeholder_and_unknown_apart():
    body = TestClient(_app()).get("/api/imported/number-status", params=[
        ("numbers", "+18102689630"), ("numbers", "3135550101"), ("numbers", "+13135550177"), ("numbers", "Katrina")]).json()["items"]
    assert body["+18102689630"]["state"] == "known" and body["+18102689630"]["label"] == "Katrina Kinzel"
    assert body["3135550101"]["state"] == "placeholder" and body["3135550101"]["entity_id"] == "e-ph"
    assert body["+13135550177"] == {"state": "unknown", "number": "3135550177", "entity_id": None, "label": "(313) 555-0177"}
    assert "Katrina" not in body
