"""Synthetic unit-read tests; no real messages, credentials or case writes.

Cover member identity, all-unit scope, generation isolation, literal search and
historical record readback. Byline: ChatGPT, 2026-10-17.
"""
import base64
import json
from datetime import datetime, timezone
from uuid import UUID
from fastapi import FastAPI
from fastapi.testclient import TestClient
import pytest

from app.runtime.imported import router
from app.service import imported, imported_units as service
from app.repo import imported_units_pg as repo

FOLDER = "b2://fixture/export/messages/inbox/thread_001"

def source(part, folder=FOLDER, fmt="Facebook"):
    key = f"{folder}/message_{part}.json"
    return {"id": imported.encode_id(key), "export_key": key, "file_name": f"message_{part}.json",
            "format": fmt, "owner": None, "owner_name": None}


def test_exact_registered_facebook_siblings_not_names_or_people():
    exports = [source(i) for i in [7, 1, 10, 2]] + [source(3, folder=FOLDER + "_different"), source(4, fmt="Other")]
    root, members, basis = service.select_members(exports[0], exports)
    assert root == FOLDER and basis == "registered_facebook_folder"
    assert [m["file_name"] for m in members] == ["message_1.json", "message_2.json", "message_7.json", "message_10.json"]
    assert len(exports) == 6


def test_unknown_structure_remains_single_export():
    exports = [source(1, "b2://fixture/random"), source(2, "b2://fixture/random")]
    root, members, basis = service.select_members(exports[0], exports)
    assert basis == "registered_export" and members == [exports[0]]
    assert root == exports[0]["export_key"]


@pytest.fixture
def unit_client(monkeypatch):
    exports = [source(i) for i in range(1, 8)]
    now = datetime(2020, 1, 1, tzinfo=timezone.utc)
    inventory = [{"source_version_id": str(UUID(int=i)), "source_key": s["export_key"],
                  "export_key": s["export_key"], "conv": s["file_name"], "version_ordinal": 3,
                  "retained_versions": 3, "acquired_at": now, "published": True,
                  "generation_id": str(UUID(int=100 + i)), "generation_ordinal": 2,
                  "raw_records": 4, "normalized_records": 4, "messages": 4}
                 for i, s in enumerate(exports, 1)]
    async def get_exports():
        return exports, True
    monkeypatch.setattr(imported, "_exports", get_exports)
    monkeypatch.setattr(imported, "live_matter", lambda: "configured-case")
    monkeypatch.setattr(imported, "_people", lambda: None)
    monkeypatch.setattr(imported, "_message", lambda row, people, owner: {
        "id": row["id"], "body": row["body"], "at": row["occurred_at"].isoformat() if row["occurred_at"] else None,
        "party": "first_party", "sender": {"label": "Fixture"},
    })
    monkeypatch.setattr(repo, "snapshot", lambda matter, keys, conv=None: [r for r in inventory if r["export_key"] in keys])
    calls = []
    def message_page(matter, generations, **args):
        calls.append({"matter": matter, "generations": generations, **args})
        allowed = [r for r in inventory if r["generation_id"] in generations]
        return [{"id": str(UUID(int=200 + i)), "occurred_at": now,
                 "export_key": allowed[i % len(allowed)]["export_key"],
                 "source_key": allowed[i % len(allowed)]["source_key"],
                 "generation_id": allowed[i % len(allowed)]["generation_id"],
                 "conv": allowed[i % len(allowed)]["conv"], "body": "Synthetic message", "projection_kind": "first_party",
                 "normalized_payload": {"content": {"body": "Synthetic message"}}}
                for i in range(args["limit"] + 1)]
    monkeypatch.setattr(repo, "message_page", message_page)
    app = FastAPI()
    app.include_router(router)
    return TestClient(app), exports, inventory, calls


def test_unit_inventory_counts_selected_generations_without_claiming_verification(unit_client):
    client, exports, _, _ = unit_client
    response = client.get(f"/api/imported/sources/{exports[6]['id']}/unit")
    assert response.status_code == 200
    payload = response.json()
    assert len(payload["members"]) == 7 and payload["message_count"] == 28
    assert payload["normalization_verified"] is False
    assert len(payload["extraction_targets"]) == 7
    assert payload["members"][0]["versions"][0]["retained_versions"] == 3
    assert "Unimported" in payload["coverage"]


def test_all_members_searched_before_pagination_and_literal_query_preserved(unit_client):
    client, exports, inventory, calls = unit_client
    path = f"/api/imported/sources/{exports[0]['id']}/unit/messages"
    response = client.get(path, params={"q": "50%_done", "limit": 2})
    assert response.status_code == 200
    assert calls[0]["q"] == "50%_done"
    assert calls[0]["generations"] == [r["generation_id"] for r in inventory]
    assert len(response.json()["items"]) == 2
    assert response.json()["items"][0]["source"]["source_key"].startswith(FOLDER)
    assert response.json()["next_cursor"]
    assert client.get(path, params={"q": "changed", "cursor": response.json()["next_cursor"]}).status_code == 409
    assert len(calls) == 1  # changed cursor failed before a broader read


def test_member_filter_does_not_discard_parent_membership(unit_client):
    client, exports, inventory, calls = unit_client
    path = f"/api/imported/sources/{exports[0]['id']}/unit/messages"
    assert client.get(path, params={"member_id": exports[5]["id"]}).status_code == 200
    assert calls[-1]["generations"] == [inventory[5]["generation_id"]]
    assert client.get(path, params={"member_id": "foreign"}).status_code == 422
    assert client.get("/api/imported/sources/notfound/unit").status_code == 404
    assert client.get(path, params={"limit": 10001}).status_code == 422
    assert client.get(path, params={"q": "x" * 201}).status_code == 422


def test_cursor_can_retain_unknown_time_without_inventing_date():
    payload = {"scope": "sig", "id": str(UUID(int=17)), "time": None}
    cursor = base64.urlsafe_b64encode(json.dumps(payload).encode()).decode().rstrip("=")
    assert service.decode_cursor(cursor, "sig") == (None, str(UUID(int=17)))
    for invalid in ["garbage", base64.urlsafe_b64encode(b'[]').decode(), "@@"]:
        with pytest.raises(imported.ImportedError) as error:
            service.decode_cursor(invalid, "sig")
        assert error.value.status == 422


def test_historical_record_is_visible_without_marking_current(unit_client, monkeypatch):
    client, exports, _, _ = unit_client
    record_id = str(UUID(int=999))
    observed = []
    def get_record(matter, keys, requested):
        observed.append(keys)
        return {"id": requested, "generation_id": str(UUID(int=987)), "occurred_at": None,
                "normalized_payload": {"content": {"body": "Historical fixture"}},
                "source_key": exports[0]["export_key"]}
    monkeypatch.setattr(repo, "record", get_record)
    path = f"/api/imported/sources/{exports[0]['id']}/unit/records/{record_id}"
    response = client.get(path)
    assert response.status_code == 200 and response.json()["current_generation"] is False
    assert len(observed[0]) == 7 and response.json()["occurred_at"] is None
    monkeypatch.setattr(repo, "record", lambda *args: None)
    assert client.get(path).status_code == 404


def test_pg_read_is_bound_to_generation_and_filters_before_keyset(monkeypatch):
    observed = []
    monkeypatch.setattr(repo.pg, "_query", lambda sql, values: observed.append((sql, values)) or [])
    repo.message_page("configured-case", [str(UUID(int=4))], q="quote'_%", after_time=None, after_id=None, limit=5)
    sql, params = observed[0]
    assert "normalized_generation_id = ANY" in sql
    assert sql.index("strpos") < sql.index("ORDER BY f.occurred_at")
    assert "quote'_%" not in sql and params["q"] == "quote'_%"
    assert "ASC NULLS LAST" in sql and params["limit"] == 6
