"""Tests for scripts/import_fct_bundle.py (owner order 2026-09-07 13:16).

No live database — every test exercises dry-run parsing, mapping, and
validation against a tiny synthetic bundle written under ``tmp_path``.

Byline: Claude Code · Sonnet 5 · 2026-09-07
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from scripts.import_fct_bundle import (
    KNOWN_BUNDLE_TABLES,
    MAPPED_BUNDLE_TABLES,
    ValidationError,
    build_plans,
    confidence_value,
    content_sha256,
    map_record,
    parse_env_file,
    parse_when,
    read_manifest,
    read_ndjson,
)

RUN_ID = "00000000-0000-0000-0000-000000000000"


def _write_ndjson(path: Path, records: list[dict]) -> None:
    path.write_text("\n".join(json.dumps(r) for r in records) + "\n", encoding="utf-8")


def _write_bundle(bundle_dir: Path, tables: dict[str, list[dict]], manifest: dict | None = None) -> None:
    bundle_dir.mkdir(parents=True, exist_ok=True)
    if manifest is not None:
        (bundle_dir / "manifest.json").write_text(json.dumps(manifest), encoding="utf-8")
    for table, records in tables.items():
        _write_ndjson(bundle_dir / f"{table}.ndjson", records)


# ---------------------------------------------------------------------------
# small helpers
# ---------------------------------------------------------------------------


def test_confidence_value_numeric_and_word():
    assert confidence_value(0.75) == 0.75
    assert confidence_value("high") == 0.9
    assert confidence_value("Medium") == 0.6
    assert confidence_value("low") == 0.3
    assert confidence_value("nonsense") is None
    assert confidence_value(None) is None
    assert confidence_value(1.5) is None  # out of [0,1]


def test_parse_when_handles_iso_date_datetime_and_unknown():
    assert parse_when("2026-05-01").year == 2026
    assert parse_when("2026-05-01T10:30:00Z").hour == 10
    assert parse_when("unknown") is None
    assert parse_when(None) is None
    assert parse_when("not a date") is None


def test_content_sha256_is_32_bytes_and_stable():
    record = {"id": "C1-1", "name": "Jane Doe"}
    digest1 = content_sha256(record)
    digest2 = content_sha256(dict(record))  # different dict instance, same content
    assert len(digest1) == 32
    assert digest1 == digest2
    assert content_sha256({"id": "C1-1", "name": "Different"}) != digest1


def test_parse_env_file_never_executes_and_strips_quotes(tmp_path: Path):
    env_path = tmp_path / "fake.env"
    env_path.write_text('DB_USER=alice\nDB_PASS="s3cr3t"\n# comment\nNOT_A_LINE\n', encoding="utf-8")
    parsed = parse_env_file(env_path)
    assert parsed["DB_USER"] == "alice"
    assert parsed["DB_PASS"] == "s3cr3t"
    assert "NOT_A_LINE" not in parsed


def test_parse_env_file_missing_file_returns_empty(tmp_path: Path):
    assert parse_env_file(tmp_path / "absent.env") == {}


# ---------------------------------------------------------------------------
# read_ndjson / read_manifest
# ---------------------------------------------------------------------------


def test_read_ndjson_round_trips(tmp_path: Path):
    path = tmp_path / "person.ndjson"
    _write_ndjson(path, [{"id": "p1", "name": "A"}, {"id": "p2", "name": "B"}])
    records = read_ndjson(path)
    assert [r["id"] for r in records] == ["p1", "p2"]


def test_read_ndjson_skips_blank_lines(tmp_path: Path):
    path = tmp_path / "person.ndjson"
    path.write_text('{"id": "p1", "name": "A"}\n\n\n{"id": "p2", "name": "B"}\n', encoding="utf-8")
    assert len(read_ndjson(path)) == 2


def test_read_ndjson_raises_validation_error_on_bad_json(tmp_path: Path):
    path = tmp_path / "person.ndjson"
    path.write_text('{"id": "p1", "name": "A"}\nnot json at all\n', encoding="utf-8")
    with pytest.raises(ValidationError):
        read_ndjson(path)


def test_read_manifest_absent_returns_none(tmp_path: Path):
    assert read_manifest(tmp_path) is None


def test_read_manifest_reads_json(tmp_path: Path):
    (tmp_path / "manifest.json").write_text(json.dumps({"schema": "fct-platform-bundle/v1"}), encoding="utf-8")
    manifest = read_manifest(tmp_path)
    assert manifest is not None
    assert manifest["schema"] == "fct-platform-bundle/v1"


# ---------------------------------------------------------------------------
# map_record — one happy-path case per mapped bundle table
# ---------------------------------------------------------------------------


def test_map_person():
    plan = map_record("person", {"id": "p1", "name": "Jane Doe", "role": "owner"}, RUN_ID)
    assert plan.target_table == "working.candidate_entity"
    assert "candidate_entity" in plan.sql
    assert plan.params[0] == RUN_ID
    assert plan.params[1] == "fct_bundle:person"
    assert plan.params[2] == "p1"
    assert plan.params[3] == "person"  # entity_type
    assert plan.params[4] == "Jane Doe"


def test_map_person_requires_name():
    with pytest.raises(ValidationError):
        map_record("person", {"id": "p1"}, RUN_ID)


def test_map_child_strips_dob_and_contact():
    record = {
        "id": "c1",
        "name": "J.D.",
        "role": "child",
        "age": 9,
        "dob": "2017-01-01",
        "contact": "555-1234",
        "address": "123 Main St",
    }
    plan = map_record("child", record, RUN_ID)
    attrs = json.loads(plan.params[7])
    assert "dob" not in attrs
    assert "address" not in attrs
    assert "contact" not in attrs
    assert attrs["age"] == 9
    assert attrs["role"] == "child"


def test_map_court_uses_organization_entity_type():
    plan = map_record("court", {"id": "ct1", "court": "Wayne County Circuit Court"}, RUN_ID)
    assert plan.params[3] == "organization"
    attrs = json.loads(plan.params[7])
    assert attrs["entity_subtype"] == "court"


def test_map_event_happy_path():
    record = {"id": "e1", "description": "Something happened", "occurred_at": "2026-01-15", "factors": ["a", "c"]}
    plan = map_record("event", record, RUN_ID)
    assert plan.target_table == "working.candidate_event"
    assert plan.params[4] == "Something happened"  # summary
    assert plan.params[5].year == 2026  # occurred_at


def test_map_event_requires_occurred_at():
    record = {"id": "e1", "description": "Something happened", "occurred_at": "unknown"}
    with pytest.raises(ValidationError):
        map_record("event", record, RUN_ID)


def test_map_note():
    plan = map_record("note", {"id": "n1", "text": "watch for this", "kind": "strategy"}, RUN_ID)
    assert plan.target_table == "working.candidate_fact"
    assert plan.params[3] == "strategy"  # predicate
    assert plan.params[4] == "watch for this"  # statement


@pytest.mark.parametrize(
    "table,event_type", [("order", "court_order"), ("hearing", "hearing"), ("deadline", "deadline")]
)
def test_map_timeline_events(table, event_type):
    record = {"id": f"{table}-1", "title": f"A {table}", "date": "2026-03-01"}
    plan = map_record(table, record, RUN_ID)
    assert plan.target_table == "timeline.event_candidate"
    assert plan.params[0] == "fct_bundle"  # source_system
    assert plan.params[5] == "point"  # temporal_precision
    assert plan.params[8] == event_type


def test_map_court_event_uses_kind_as_event_type():
    record = {"id": "ce1", "title": "Status conference", "kind": "conference", "date": "2026-04-01"}
    plan = map_record("court_event", record, RUN_ID)
    assert plan.params[8] == "conference"


def test_map_filing_ignores_filed_or_planned_as_a_date():
    record = {"id": "f1", "title": "Motion to modify", "filed_or_planned": "filed", "date": "2026-02-01"}
    plan = map_record("filing", record, RUN_ID)
    assert plan.params[6].year == 2026  # occurred_at came from 'date', not 'filed_or_planned'
    assert plan.params[5] == "point"


def test_map_filing_without_date_is_uncertain_not_an_error():
    record = {"id": "f2", "title": "Motion to modify", "filed_or_planned": "planned"}
    plan = map_record("filing", record, RUN_ID)
    assert plan.params[5] == "uncertain"
    assert plan.params[6] is None


def test_map_timeline_event_requires_title():
    with pytest.raises(ValidationError):
        map_record("hearing", {"id": "h1", "date": "2026-01-01"}, RUN_ID)


def test_map_source_provenance_class_heuristics():
    owner_plan = map_record("source", {"id": "s1", "path": "/a.md", "authored_by": "owner"}, RUN_ID)
    assert owner_plan.params[1] == "first_party_authored"

    court_plan = map_record("source", {"id": "s2", "path": "/b.pdf", "authored_by": "court"}, RUN_ID)
    assert court_plan.params[1] == "acquired_third_party"

    ai_plan = map_record("source", {"id": "s3", "path": "/c.json", "authored_by": "ai:claude"}, RUN_ID)
    assert ai_plan.params[1] == "system_generated"

    unknown_plan = map_record("source", {"id": "s4", "path": "/d.txt"}, RUN_ID)
    assert unknown_plan.params[1] == "unknown"


# ---------------------------------------------------------------------------
# build_plans — the whole-bundle dry-run pass, no DB
# ---------------------------------------------------------------------------


def test_build_plans_maps_valid_records_and_reports_unmapped(tmp_path: Path):
    bundle_dir = tmp_path / "bundle"
    _write_bundle(
        bundle_dir,
        {
            "person": [{"id": "p1", "name": "Jane Doe"}],
            "child": [{"id": "c1", "name": "J.D.", "age": 9}],
            "event": [{"id": "e1", "description": "narrated thing", "occurred_at": "2026-01-01"}],
            "message": [{"id": "m1", "from": "a", "to": ["b"], "body": "hi", "sent_at": "2026-01-01T00:00:00Z"}],
            "exhibit": [{"id": "x1", "label_or_name": "Screenshot 1"}],
        },
        manifest={"schema": "fct-platform-bundle/v1", "exported_at": "2026-09-07T13:16:00Z"},
    )
    plans, reports, edge_count = build_plans(bundle_dir, RUN_ID)

    mapped_ids = {(p.bundle_table, p.record_id) for p in plans}
    assert ("person", "p1") in mapped_ids
    assert ("child", "c1") in mapped_ids
    assert ("event", "e1") in mapped_ids

    assert reports["message"].unmapped
    assert reports["message"].total_rows == 1
    assert reports["exhibit"].unmapped
    assert reports["exhibit"].total_rows == 1

    # Tables with no ndjson file at all are still reported (rows=0), never
    # silently dropped from the summary.
    for table in KNOWN_BUNDLE_TABLES:
        assert table in reports
    assert edge_count == 0


def test_build_plans_records_validation_errors_without_crashing(tmp_path: Path):
    bundle_dir = tmp_path / "bundle"
    _write_bundle(
        bundle_dir,
        {
            "person": [{"id": "p1", "name": "Jane Doe"}, {"id": "p2"}],  # p2 has no name
            "event": [{"id": "e1", "description": "x", "occurred_at": "unknown"}],  # unparseable date
        },
    )
    plans, reports, _edges = build_plans(bundle_dir, RUN_ID)

    person_report = reports["person"]
    assert person_report.planned == 1
    assert len(person_report.validation_errors) == 1
    assert person_report.validation_errors[0][0] == "p2"

    event_report = reports["event"]
    assert event_report.planned == 0
    assert len(event_report.validation_errors) == 1

    planned_person_ids = {p.record_id for p in plans if p.bundle_table == "person"}
    assert planned_person_ids == {"p1"}


def test_build_plans_reports_edges_count(tmp_path: Path):
    bundle_dir = tmp_path / "bundle"
    bundle_dir.mkdir()
    _write_ndjson(bundle_dir / "edges.ndjson", [{"from": "p1", "to": "e1", "edge": "participant"}])
    _plans, _reports, edge_count = build_plans(bundle_dir, RUN_ID)
    assert edge_count == 1


def test_build_plans_flags_unexpected_table(tmp_path: Path):
    bundle_dir = tmp_path / "bundle"
    bundle_dir.mkdir()
    _write_ndjson(bundle_dir / "totally_new_table.ndjson", [{"id": "z1"}])
    _plans, reports, _edges = build_plans(bundle_dir, RUN_ID)
    assert "totally_new_table" in reports
    assert reports["totally_new_table"].unmapped
    assert any("unexpected table" in msg for _rid, msg in reports["totally_new_table"].validation_errors)


def test_every_known_table_is_either_mapped_or_documented_unmapped():
    # Sanity check on the module's own bookkeeping: MAPPED_BUNDLE_TABLES must
    # be a subset of KNOWN_BUNDLE_TABLES (no typos creating a phantom table).
    assert MAPPED_BUNDLE_TABLES.issubset(set(KNOWN_BUNDLE_TABLES))
