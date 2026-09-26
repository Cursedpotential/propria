"""Exhibit candidates come only from approved package items.

> _Byline: Grok · grok-4.6 · 2026-08-18_
No evidence bytes. No seeded Bates numbers. Annotations persist.
"""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    LegalSourcePackageItem,
    ReviewState,
)
from legal_workspace.domain.exhibits import (
    ExhibitAnnotationCreate,
    ExhibitReadiness,
    bates_prefix,
    next_bates_number,
)
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace
from conftest import seed_synthetic_approved_package


def _package(matter_id=None) -> LegalSourcePackage:
    return LegalSourcePackage(
        package_id=uuid4(),
        manifest_hash="sha256:" + "a" * 64,
        matter_id=matter_id or uuid4(),
        created_at=datetime.now(UTC),
        items=[
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=uuid4(),
                assertion_version=1,
                span_locator="sms:2024-03-12:14:02",
                custody_locator="h1:exh",
                content_hash="sha256:" + "b" * 64,
                review_state=ReviewState.APPROVED,
            ),
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=uuid4(),
                assertion_version=1,
                span_locator="sms:unreviewed",
                custody_locator="h1:skip",
                content_hash="sha256:" + "c" * 64,
                review_state=ReviewState.CANDIDATE,
            ),
        ],
    )


def test_blank_workspace_has_no_exhibits_or_bates(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    assert workspace.list_exhibits() == []
    assert workspace.load().exhibit_annotations == []


def test_candidates_are_approved_package_items_only(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    package = _package(workspace.load().matter.matter_id)
    imported = seed_synthetic_approved_package(workspace, package)
    assert imported.blocked is False
    rows = workspace.list_exhibits()
    assert len(rows) == 1
    assert rows[0].item_id == imported.accepted.items[0].item_id
    assert rows[0].review_state is ReviewState.APPROVED
    assert rows[0].bates_number == ""
    assert rows[0].bytes_present is False
    assert "000001" not in rows[0].exhibit_label


def test_annotation_persists_and_unknown_item_is_rejected(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    imported = seed_synthetic_approved_package(workspace, _package(workspace.load().matter.matter_id))
    item_id = imported.accepted.items[0].item_id
    annotated = workspace.annotate_exhibit(
        ExhibitAnnotationCreate(
            item_id=item_id,
            exhibit_label="Petitioner A",
            bates_number="OWNER-12",
            foundation="MRE 901 personal knowledge",
            relevance="Factor (j) facilitation",
            readiness=ExhibitReadiness.TECHNICALLY_READY,
        )
    )
    assert annotated.exhibit_label == "Petitioner A"
    assert annotated.bates_number == "OWNER-12"
    assert annotated.bytes_present is False
    reloaded = Workspace(tmp_path)
    rows = reloaded.list_exhibits()
    assert len(rows) == 1
    assert rows[0].exhibit_label == "Petitioner A"
    assert rows[0].bates_number == "OWNER-12"
    assert reloaded.load().exhibit_annotations
    try:
        workspace.annotate_exhibit(ExhibitAnnotationCreate(item_id=uuid4(), exhibit_label="nope"))
        raise AssertionError("unknown item must fail")
    except ValueError as exc:
        assert "approved package items" in str(exc)


def test_http_exhibits(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    empty = client.get("/v1/exhibits")
    assert empty.status_code == 200
    assert empty.json() == []
    package = _package(store.load().matter.matter_id)
    imported = client.post(
        "/v1/legal-source-packages:import",
        json=package.model_dump(mode="json"),
    )
    assert imported.status_code == 200
    assert imported.json()["blocked"] is True
    assert client.get("/v1/exhibits").json() == []
    seed_synthetic_approved_package(store, package)
    listed = client.get("/v1/exhibits")
    assert listed.status_code == 200
    rows = listed.json()
    assert len(rows) == 1
    assert rows[0]["bates_number"] == ""
    assert rows[0]["bytes_present"] is False
    created = client.post(
        "/v1/exhibits",
        json={
            "item_id": rows[0]["item_id"],
            "exhibit_label": "A",
            "foundation": "business record",
        },
    )
    assert created.status_code == 200, created.text
    assert created.json()["exhibit_label"] == "A"
    assert created.json()["bytes_present"] is False
    blocked = client.post(
        "/v1/exhibits",
        json={"item_id": str(uuid4()), "exhibit_label": "invented"},
    )
    assert blocked.status_code == 409
    home = client.get("/v1/matter")
    assert any(row["path"] == "/evidence" and row["label"] == "Evidence list" for row in home.json()["next_surfaces"])
    assert home.json()["exhibit_count"] == 1


def test_bates_sequence_is_owner_triggered_not_seeded(tmp_path) -> None:
    assert next_bates_number("GENESEE", []) == "GENESEE-000001"
    assert next_bates_number("GENESEE", ["GENESEE-000001", "OWNER-12"]) == "GENESEE-000002"
    assert bates_prefix("Genesee County custody matter").startswith("GENESEE")
    workspace = Workspace(tmp_path)
    imported = seed_synthetic_approved_package(workspace, _package(workspace.load().matter.matter_id))
    item_id = imported.accepted.items[0].item_id
    assert workspace.list_exhibits()[0].bates_number == ""
    stamped = workspace.assign_bates(item_id)
    assert stamped.bates_number.endswith("-000001")
    assert stamped.bytes_present is False
    reloaded = Workspace(tmp_path).list_exhibits()[0]
    assert reloaded.bates_number == stamped.bates_number
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    http = TestClient(app).post(f"/v1/exhibits/{item_id}:bates")
    assert http.status_code == 200
    assert http.json()["bates_number"].endswith("-000001")
