"""Protect shared Imported injection seams after the query-domain split.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
All upstream reads are synthetic; no live database or search credentials are used.
"""

from __future__ import annotations

import asyncio
from datetime import UTC, datetime, timedelta
from types import SimpleNamespace

import pytest

from app.repo import imported_pg as pg
from app.service import imported as service
from app.service.proffer_errors import ProfferError

MATTER = "11111111-1111-4111-8111-111111111111"
KEY = "b2://synthetic/casevault/messaging/sms-backup-restore/3135550100/export.xml"
ROW_ID = "33333333-3333-4333-8333-333333333333"
NOW = datetime(2026, 10, 5, tzinfo=UTC)


@pytest.fixture(autouse=True)
def isolated_cache(monkeypatch):
    """Give each test its own facade cache without changing other test modules."""
    monkeypatch.setattr(service, "_cache", {})
    monkeypatch.setattr(service, "live_matter", lambda: MATTER)
    monkeypatch.setattr(service, "_people", lambda: service._People([]))


@pytest.mark.parametrize(
    "name,args,kwargs,scoped",
    [
        ("source_versions", (MATTER,), {}, True),
        ("threads", (MATTER, KEY), {"limit": 2, "offset": 3}, True),
        ("thread_participants", (MATTER, KEY, ["conversation"]), {}, True),
        ("thread_last_messages", (MATTER, KEY, ["conversation"]), {}, True),
        ("messages", (MATTER, KEY, "conversation"), {"ts": None, "row_id": None, "direction": "before", "limit": 2}, True),
        ("message_time", (MATTER, ROW_ID), {}, True),
        ("versions_to_threads", (MATTER, [ROW_ID]), {}, True),
        ("call_log_has_rows", (), {}, False),
        ("calls_normalized", (MATTER,), {"ts": None, "row_id": None, "limit": 2}, True),
        ("calls_summary", (MATTER,), {}, True),
        ("calls_from_log", (), {"ts": None, "row_id": None, "limit": 2}, False),
        ("people", (), {}, False),
        ("numbers_activity", (MATTER,), {}, True),
        ("entity_activity", (), {}, False),
        ("working_unlinked_numbers", (), {}, False),
        ("number_records", (MATTER, "3135550101"), {"ts": None, "row_id": None, "limit": 2}, True),
        ("number_record_counts", (MATTER, "3135550101"), {}, True),
        ("review_queue", (MATTER,), {}, True),
    ],
)
def test_repo_exports_use_the_facade_query_and_cte(monkeypatch, name, args, kwargs, scoped):
    """Replacing the old query seam must intercept every split query domain."""
    seen = []

    def query(sql, params):
        seen.append((sql, params))
        return [{"has": True}]

    monkeypatch.setattr(pg, "_query", query)
    monkeypatch.setattr(pg, "_SV", "synthetic_scope_cte")
    getattr(pg, name)(*args, **kwargs)
    assert len(seen) == 1
    sql, params = seen[0]
    assert ("synthetic_scope_cte" in sql) is scoped
    assert params.get("matter") == (MATTER if scoped else None)
    if "limit" in kwargs:
        assert params["limit"] == 3
    if name.startswith("number_record"):
        assert params["forms"] == pg._number_forms("3135550101")


def test_empty_repo_inputs_still_skip_the_query(monkeypatch):
    """Empty conversation/version lists must remain no-I/O reads."""
    def unexpected(*args, **kwargs):
        pytest.fail("empty input queried the database")

    monkeypatch.setattr(pg, "_query", unexpected)
    assert pg.thread_participants(MATTER, KEY, []) == []
    assert pg.thread_last_messages(MATTER, KEY, []) == []
    assert pg.versions_to_threads(MATTER, []) == []


def test_cache_ttl_and_invalidation_preserve_lifecycle_state(monkeypatch):
    """Cache expiry and identity invalidation keep their original distinct scopes."""
    clock = [10.0]
    monkeypatch.setattr(service.time, "monotonic", lambda: clock[0])
    builds = []

    def build():
        builds.append(len(builds))
        return builds[-1]

    assert service._cached("people", 10, build) == 0
    clock[0] = 19.99
    assert service._cached("people", 10, build) == 0
    clock[0] = 20.0
    assert service._cached("people", 10, build) == 1
    for key in ("entity-activity", "unlinked", f"activity:{MATTER}", f"sv:{MATTER}", "lifecycles", "other"):
        service._cache[key] = (20.0, key)
    service.invalidate()
    assert set(service._cache) == {"lifecycles", "other"}


def test_lifecycle_refresh_pages_filters_case_and_retains_previous_on_failure(monkeypatch):
    """Lifecycle refresh stays scoped and preserves stale data when upstream fails."""
    seen = []

    async def request(method, path, *, params):
        seen.append((method, path, dict(params)))
        payload = {
            "items": [
                {"source_version_ref": "keep", "matter_id": MATTER, "lifecycle": "running"},
                {"source_version_ref": "foreign", "matter_id": "other", "lifecycle": "failed"},
            ],
            "next_cursor": "page2" if len(seen) == 1 else None,
        }
        return SimpleNamespace(json=lambda: payload)

    monkeypatch.setattr(service.proffer, "_request", request)
    asyncio.run(service._refresh_lifecycles())
    assert seen[1][2] == {"limit": 100, "cursor": "page2"}
    assert service._cache["lifecycles"][1] == {"keep": "running"}

    async def unavailable(*args, **kwargs):
        raise ProfferError("synthetic outage", 503)

    monkeypatch.setattr(service.proffer, "_request", unavailable)
    asyncio.run(service._refresh_lifecycles())
    assert service._cache["lifecycles"][1] == {"keep": "running"}


def test_message_window_uses_patched_exports_people_and_pg(monkeypatch):
    """Around-message paging retains the old facade helpers and cursor direction."""
    source = {"export_key": KEY, "id": "source", "file_name": "export.xml", "format": "SMS", "device": None, "owner": None, "owner_name": None}

    async def exports():
        return [source], True

    monkeypatch.setattr(service, "_exports", exports)
    monkeypatch.setattr(pg, "message_time", lambda matter, row_id: {"occurred_at": NOW})
    seen = []
    rows = [
        {"id": ROW_ID, "occurred_at": NOW, "body": "newer", "participants": [], "projection_kind": None, "attachment_count": 0, "has_attachments": False, "certainty": "exact"},
        {"id": "44444444-4444-4444-8444-444444444444", "occurred_at": NOW - timedelta(hours=1), "body": "older", "participants": [], "projection_kind": None, "attachment_count": 0, "has_attachments": False, "certainty": "exact"},
    ]

    def messages(matter, export, conv, **kwargs):
        seen.append((matter, export, conv, kwargs))
        return rows.copy()

    monkeypatch.setattr(pg, "messages", messages)
    result = asyncio.run(service.thread_messages(service.encode_id(KEY, "conversation"), cursor=None, direction="after", limit=2, around=ROW_ID))
    assert seen == [(MATTER, KEY, "conversation", {"ts": (NOW + timedelta(hours=12)).isoformat(), "row_id": "ffffffff-ffff-ffff-ffff-ffffffffffff", "direction": "before", "limit": 60})]
    assert [item["body"] for item in result["items"]] == ["older", "newer"]
    assert result["older_cursor"] is None
    assert service._cursor_decode(result["newer_cursor"]) == (NOW.isoformat(), ROW_ID)


@pytest.mark.parametrize("use_log", [False, True])
def test_calls_follow_patched_repo_fallback_and_keep_summary(monkeypatch, use_log):
    """Both call read paths must resolve patched facade repository methods."""
    monkeypatch.setattr(pg, "call_log_has_rows", lambda: use_log)
    summary = {"total": 1, "first_at": NOW, "last_at": NOW}
    monkeypatch.setattr(pg, "calls_summary", lambda matter: summary)
    monkeypatch.setattr(pg, "calls_from_log", lambda **kwargs: [{"id": ROW_ID, "occurred_at": NOW, "direction": "inbound", "from_e164": "+13135550101", "from_raw": None, "to_e164": None, "to_raw": None, "call_type": "missed", "duration_s": 0}])
    monkeypatch.setattr(pg, "calls_normalized", lambda matter, **kwargs: [{"id": ROW_ID, "occurred_at": NOW, "export_key": KEY, "content": {"missed": True, "direction": "incoming", "duration_seconds": 0}, "participants": [{"identifier": "+13135550101"}]}])
    result = asyncio.run(service.calls(cursor=None, limit=2))
    assert result["items"][0]["missed"] is True
    assert result["summary"] == {"total": 1, "first_at": NOW.isoformat(), "last_at": NOW.isoformat()}
    assert result["next_cursor"] is None
    assert result["read_from"] == ("working.call_log" if use_log else "context.normalized_record_identity (record_type call)")
