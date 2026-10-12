"""Verify category-first package reads and revision-bound correction capabilities.

Inputs: synthetic persisted rows and native reader responses. Outputs: assertions
on pagination, complete fields and provenance. Effects: no live source changes;
run in the existing VPS Workbench Python environment.
"""
import asyncio

import pytest

from app.service import context_packages as service
from app.service.proffer_errors import ProfferError

SOURCE = {"source_version_ref": "11111111-1111-4111-8111-111111111111",
    "source_ref": "b2://case/SourceCorpus/sms/export.xml", "source_type": "smsbackuprestore_xml",
    "workflow_id": "context-source:actual", "revision": 1, "package_ref": "b2://case/unit/export",
    "category": "sms", "domain": "sms", "path": "SourceCorpus/sms/export.xml",
    "classification_basis": {"kind": "persisted_source_metadata"}}


def test_cursor_cannot_cross_category_domain_version_or_section():
    """Reject a continuation reused for a different persisted query boundary."""
    binding = {"category": "sms", "domain": "sms", "version": SOURCE["source_version_ref"], "section": "chunks"}
    cursor = service._cursor(binding, "native-key")
    assert service._after(cursor, binding) == "native-key"
    for key in binding:
        with pytest.raises(ProfferError) as error:
            service._after(cursor, {**binding, key: "different"})
        assert error.value.status_code == 409


def test_original_fields_unknown_edges_and_native_locator_survive_correction():
    """Keep original unknown metadata and actual directed edges beside owner edits."""
    edge = {"kind": "real-unrecognized-relation", "direction": "in", "target": "actual-id", "sourcebasis": {"locator": "line:7"}}
    row = {"chunk_ref": "file:///actual/chunks/3", "content": "native body", "locator_ref": "xml:/sms[3]",
        "new_native_field": {"unknown": True}, "attrs": {"relations": [edge]}}
    correction = {"correction_ref": "22222222-2222-4222-8222-222222222222",
        "corrected_value": {"fields": {"label": "reviewed"}, "review_state": "accepted"}}
    item = service._item(row, SOURCE, "chunks", 0, correction, "human-" + SOURCE["source_version_ref"])
    assert item["fields"] == row
    assert item["id"] == row["chunk_ref"]
    assert item["edges"] == [edge]
    assert item["source_refs"][0]["locator"] == "xml:/sms[3]"
    assert item["correction"]["fields"] == {"label": "reviewed"}
    assert item["placement"]["inherited_from"] == SOURCE["package_ref"]
    assert next(cap for cap in item["capabilities"] if cap["operation"] == "commit")["available"]
    assert "content" in item["capabilities"][0]["immutable_fields"]


def test_unaccepted_proposal_cannot_advertise_commit_and_literal_edge_is_exposed():
    """Expose a real edge row without inventing relations or enabling an unaccepted commit."""
    row = {"edge_id": "actual-edge", "kind": "before", "from_node_id": "a", "to_node_id": "b", "source_pins": []}
    item = service._item(row, SOURCE, "edges", 0, None, "actual-handle")
    assert item["edges"] == [row]
    assert not next(cap for cap in item["capabilities"] if cap["operation"] == "commit")["available"]


def test_category_first_list_preserves_atomic_package_and_unresolved_sources(monkeypatch):
    """List persisted categories then their packages without hiding missing reader bindings."""
    monkeypatch.setattr(service, "_matter", lambda: "actual-matter")
    monkeypatch.setattr(service.repo, "categories", lambda *_: [{"id": "sms", "category": "sms", "domain": "sms", "count": 1, "cursor_key": "sms"}])
    calls = []

    def packages(matter, category, domain, after, limit):
        """Return one registered package and record exact taxonomy coordinates."""
        calls.append((matter, category, domain, after, limit))
        return [dict(SOURCE), {**SOURCE, "source_version_ref": "other", "source_type": "json", "workflow_id": None, "preview_handle": None}]

    monkeypatch.setattr(service.repo, "packages", packages)
    landing = asyncio.run(service.categories(None))
    assert landing["items"][0]["id"] == "sms"
    listing = asyncio.run(service.packages("sms", "sms", None))
    assert calls == [("actual-matter", "sms", "sms", "", 50)]
    assert listing["items"][0]["handle"] == "human-" + SOURCE["source_version_ref"]
    assert all(item["package_ref"] == SOURCE["package_ref"] for item in listing["items"])
    assert listing["items"][1]["available"] is False


def test_metadata_landing_keeps_all_fields_and_paginated_history(monkeypatch):
    """Expose native metadata, placement and complete correction continuations on the landing."""
    monkeypatch.setattr(service, "_matter", lambda: "actual-matter")
    monkeypatch.setattr(service.repo, "source", lambda *_: dict(SOURCE))
    monkeypatch.setattr(service.repo, "counts", lambda *_: {"metadata": 1, "documents": 0, "corrections": 51})
    monkeypatch.setattr(service.repo, "corrections", lambda *_: [{"correction_ref": str(i)} for i in range(51)])
    monkeypatch.setattr(service.repo, "correction_heads", lambda *_: {})
    monkeypatch.setattr(service.repo, "candidates", lambda *_: ([], 0))
    row = {"id": "actual-metadata", "fields": {"metadata_class": "new_class", "metadata": {"unrecognized": [1, 2, 3]}}}
    monkeypatch.setattr(service.repo, "metadata", lambda *_: [row])

    async def engine(*_, **__):
        """Return real-shaped native progress without normalized identifiers."""
        return {"manifest_ref": "file:///native/manifest", "stages": [{"records": 3}, {"records": 0, "chunks": 2}]}

    monkeypatch.setattr(service, "_engine", engine)
    value = asyncio.run(service.snapshot("human-" + SOURCE["source_version_ref"], "allmetadata", None, None))
    assert value["items"][0]["id"] == "placement"
    assert value["items"][1]["fields"] == row["fields"]
    assert next(section for section in value["sections"] if section["id"] == "allmetadata")["count"] == 2
    assert next(section for section in value["sections"] if section["id"] == "records")["count"] == 3
    assert len(value["metadata"]["corrections"]) == 50
    assert value["metadata"]["correction_history_next_cursor"]


def test_ai_page_cannot_continue_after_attempt_changes(monkeypatch):
    """Pin native extraction bundle revisions so a retry cannot silently mix pages."""
    monkeypatch.setattr(service, "_matter", lambda: "actual-matter")
    monkeypatch.setattr(service.repo, "source", lambda *_: {**SOURCE, "source_type": "chatgpt_official_json"})
    monkeypatch.setattr(service.repo, "counts", lambda *_: {"metadata": 0, "documents": 0, "corrections": 0})
    monkeypatch.setattr(service.repo, "corrections", lambda *_: [])
    monkeypatch.setattr(service.repo, "correction_heads", lambda *_: {})
    binding = {"handle": "context-source:actual", "version": SOURCE["source_version_ref"], "section": "chunks", "revision": 1, "reader_revision": "previous-attempt"}

    async def engine(*_, **__):
        """Return the subsequent immutable bundle generation without source mutation."""
        return {"revision": "new-attempt", "source_version_ref": SOURCE["source_version_ref"], "source": {"source_ref": SOURCE["source_ref"]}}

    monkeypatch.setattr(service, "_engine", engine)
    with pytest.raises(ProfferError) as error:
        asyncio.run(service.snapshot("context-source:actual", "chunks", service._cursor(binding, 50), None))
    assert error.value.status_code == 409
