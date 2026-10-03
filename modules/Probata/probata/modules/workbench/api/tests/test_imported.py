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


def test_an_unconfirmed_person_is_not_confirmed_and_offers_who_is_this():
    people = service._People(PEOPLE)
    unconfirmed = service._participant("+13135550101", people, "Matt")
    assert unconfirmed["unconfirmed"] is True and unconfirmed["entity_id"] == "e-ph"
    assert unconfirmed["label"] == "Unknown 313-555-0101" and unconfirmed["number"] == "3135550101"
    nobody = service._participant("+13135550199", people, "Matt")
    assert nobody["unconfirmed"] is False and nobody["entity_id"] is None and nobody["number"] == "3135550199"
    named = service._participant("+18102689630", people, "Matt")
    assert named["unconfirmed"] is False and named["number"] is None
    assert "Unknown 313-555-0101" not in people.by_person


def test_a_versions_own_status_prefers_the_publish_receipt_and_never_guesses_without_run_state():
    row = {"id": "a", "approved": False, "rejected": False, "published": False}
    assert service._version_status({**row, "published": True}, {"a": "failed"}) == "committed"
    assert service._version_status({**row, "approved": True}, None) == "committed"
    assert service._version_status(row, None) == "not_finished"
    assert service._version_status(row, {"a": "failed"}) == "failed"
    assert service._version_status(row, {"a": "awaiting_repair_decision"}) == "parked"
    assert service._version_status(row, {"a": "awaiting_preview_decision"}) == "awaiting_review"


def _version(i, key, *, raw=1, norm=1, approved=False, published=False, ordinal=1, msgs=None, calls=0, source=None):
    ts = datetime(2026, 10, 2, 12, i, tzinfo=timezone.utc)
    return {"id": f"v{i}", "source_key": key, "export_key": key.split(".derived/")[0], "acquired_at": ts, "version_ordinal": ordinal,
            "published": published, "raw_n": raw, "norm_n": norm, "msgs": norm if msgs is None else msgs, "calls": calls,
            "first_at": ts, "last_at": ts, "approved": approved, "rejected": False}


def _thread(n):
    return KEY + f".derived/threads/81025951{n:02d}.0001.ndjson"


def test_a_file_that_published_on_any_attempt_is_done_and_failed_attempts_are_only_counted():
    rows = [
        _version(1, _thread(1), ordinal=1),                    # first attempt failed
        _version(2, _thread(1), ordinal=2, published=True),    # second attempt published
        _version(3, _thread(1), ordinal=3),                    # a later retry that failed again
    ]
    (file,) = service.file_rows(rows, {"v1": "failed", "v3": "failed"})
    assert file["status"] == "committed" and file["failed_attempts"] == 2 and file["attempts"] == 3
    assert file["raw"] == 1  # counts come from the version that stands for the file, not summed across attempts


def test_a_split_parent_backup_reports_its_conversations_and_is_never_failed_or_not_finished(monkeypatch):
    rows = [_version(1, KEY)]                                      # the parent backup: no records, never "finished" itself
    rows += [_version(10 + n, _thread(n), published=True) for n in range(1, 4)]
    rows += [_version(20, _thread(9))]                              # one conversation that failed
    rows += [_version(30, KEY + ".derived/media/abc.png")]         # stray derived media that never imported
    monkeypatch.setattr(pg, "source_versions", lambda matter: rows)

    async def lifecycles():
        return {"v1": "failed", "v20": "failed", "v30": "failed"}

    monkeypatch.setattr(service, "_lifecycles", lifecycles)
    (item,) = TestClient(_app()).get("/api/imported/sources").json()["items"]
    assert item["split"] == {"total": 4, "done": 3, "failed": 1, "in_progress": 0}
    assert item["status"] == "failed" and item["status_counts"] == {"committed": 3, "failed": 1, "skipped": 1}
    assert item["files"] == 5  # the parent plus four conversation files; the stray media is not a file of the import


def test_a_split_parent_whose_conversations_all_published_is_done(monkeypatch):
    rows = [_version(1, KEY)] + [_version(10 + n, _thread(n), published=True) for n in range(1, 4)] + [_version(30, KEY + ".derived/media/abc.png")]
    monkeypatch.setattr(pg, "source_versions", lambda matter: rows)

    async def lifecycles():
        return {"v1": "failed", "v30": "failed"}

    monkeypatch.setattr(service, "_lifecycles", lifecycles)
    (item,) = TestClient(_app()).get("/api/imported/sources").json()["items"]
    assert item["status"] == "committed" and item["split"]["done"] == 3 and item["split"]["failed"] == 0


def test_one_unknown_numbers_list_merges_people_without_a_person_and_unconfirmed_people(monkeypatch):
    more = PEOPLE + [
        {"entity_id": "e-ph2", "person": "Jordan Reyes", "display_name": "Jordan Reyes", "role_in_case": "unknown", "verification_state": "proposed",
         "identifier": "3135550102", "kind": "phone", "alias_status": "confirmed"},
        {"entity_id": "e-ph2", "person": "Jordan Reyes", "display_name": "Jordan Reyes", "role_in_case": "unknown", "verification_state": "proposed",
         "identifier": "j. reyes", "kind": "name", "alias_status": "candidate", "alias_text_raw": "J. Reyes"},
        {"entity_id": "e-mail", "person": "Email Only", "display_name": "Email Only", "role_in_case": "unknown", "verification_state": "proposed",
         "identifier": "e@example.com", "kind": "email"},
    ]
    monkeypatch.setattr(pg, "people", lambda: more)
    monkeypatch.setattr(pg, "entity_activity", lambda: [
        {"entity_id": "e-ph", "calls": 0, "msgs": 3, "last_at": None},
        {"entity_id": "e-ph2", "calls": 9, "msgs": 2, "last_at": None},
    ])
    monkeypatch.setattr(pg, "working_unlinked_numbers", lambda: [
        {"number": "4195550123", "n": 7}, {"number": "2485550000", "n": 1}, {"number": "3135550101", "n": 99}, {"number": "1115", "n": 5},
    ])
    client = TestClient(_app())
    body = client.get("/api/imported/unknown-numbers").json()
    assert [(i["kind"], i["number"] or i["label"], i["total"]) for i in body["items"]] == [
        ("unconfirmed", "3135550102", 11), ("no_person", "4195550123", 7), ("unconfirmed", "3135550101", 3),
        ("no_person", "2485550000", 1), ("unconfirmed", "e@example.com", 0)]
    named = next(i for i in body["items"] if i["entity_id"] == "e-ph2")
    assert named["named"] is True and named["candidates"] == ["J. Reyes"]
    assert next(i for i in body["items"] if i["entity_id"] == "e-ph")["named"] is False
    only = client.get("/api/imported/unknown-numbers", params={"kind": "no_person"}).json()
    assert [i["number"] for i in only["items"]] == ["4195550123", "2485550000"]
    assert client.get("/api/imported/unlinked-numbers").status_code == 404


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
    assert body["3135550101"]["state"] == "unconfirmed" and body["3135550101"]["entity_id"] == "e-ph"
    assert body["+13135550177"] == {"state": "unknown", "number": "3135550177", "entity_id": None, "label": "(313) 555-0177"}
    assert "Katrina" not in body


def test_number_records_show_each_message_and_call_with_its_source_and_thread(monkeypatch):
    ts = datetime(2026, 5, 1, 12, 0, tzinfo=timezone.utc)
    base = {"source_key": KEY + ".derived/threads/8102595720.0001.ndjson", "export_key": KEY, "conv": "8102595720", "certainty": "exact",
            "projection_kind": None, "has_attachments": False, "attachment_count": 0, "source_version_id": "v1"}
    rows = [
        {**base, "id": "00000000-0000-7000-8000-000000000002", "occurred_at": ts, "record_type": "call", "body": None,
         "content": {"direction": "outgoing", "missed": False, "duration_seconds": 61, "disposition": "completed"},
         "participants": [{"role": "unknown", "identifier": "+13135550101"}]},
        {**base, "id": "00000000-0000-7000-8000-000000000001", "occurred_at": ts, "record_type": "message", "body": "hello",
         "content": {"body": "hello"}, "participants": [{"role": "sender", "identifier": "+13135550101"}, {"role": "recipient", "identifier": "self"}]},
    ]
    seen = {}

    def fake(matter, number, *, ts, row_id, limit):
        seen.update(number=number, limit=limit)
        return rows

    monkeypatch.setattr(pg, "number_records", fake)
    monkeypatch.setattr(pg, "number_record_counts", lambda matter, number: {"messages": 1, "calls": 1})
    body = TestClient(_app()).get("/api/imported/number-records", params={"number": "+1 (313) 555-0101", "limit": 5}).json()
    assert seen == {"number": "3135550101", "limit": 5}
    assert body["counts"] == {"messages": 1, "calls": 1} and body["next_cursor"] is None
    call, message = body["items"]
    assert call["type"] == "call" and call["call"]["duration_s"] == 61 and call["call"]["with"]["number"] == "3135550101"
    assert message["type"] == "message" and message["message"]["body"] == "hello"
    assert message["source"]["file_name"] == "8102595720.0001.ndjson" and message["source"]["casevault_key"].startswith("SourceCorpus/messaging/")
    assert message["source"]["thread_id"] and message["source"]["device"] == "(810) 268-9630"
    assert TestClient(_app()).get("/api/imported/number-records", params={"number": "34428"}).status_code == 422
