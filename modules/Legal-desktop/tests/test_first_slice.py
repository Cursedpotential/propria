"""First vertical slice: Matter → import → factor → draft → gate.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Drives the shipped Workspace and the FastAPI app. No Docker.
"""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

from fastapi.testclient import TestClient

from legal_workspace.api.auth import AuthenticatedPrincipal
from legal_workspace.api.main import app
from legal_workspace.contracts.citations import EpistemicClass, EvidenceCitation
from legal_workspace.contracts.source_package import (
    LegalSourcePackage,
    LegalSourcePackageItem,
    ReviewState,
)
from legal_workspace.domain.factors import FactorCitationLink, FactorLetter
from legal_workspace.domain.release import ReleaseCreate
from legal_workspace.domain.review import ReviewCreate, ReviewVerdict
from legal_workspace.api import main as main_mod
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace
from conftest import seed_synthetic_approved_package

HUMAN = AuthenticatedPrincipal(
    subject="owner-subject",
    username="owner",
    email="owner@example.test",
    groups=("advocatio-users",),
    source="authentik",
)


def _approved_package(matter_id=None) -> tuple[LegalSourcePackage, EvidenceCitation]:
    assertion_id = uuid4()
    package = LegalSourcePackage(
        package_id=uuid4(),
        manifest_hash="sha256:" + "a" * 64,
        matter_id=matter_id or uuid4(),
        created_at=datetime.now(UTC),
        items=[
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=assertion_id,
                assertion_version=1,
                span_locator="sms:2024-03-12:14:02",
                custody_locator="h1:slice",
                content_hash="sha256:" + "b" * 64,
                review_state=ReviewState.APPROVED,
            ),
            LegalSourcePackageItem(
                item_id=uuid4(),
                assertion_id=uuid4(),
                assertion_version=1,
                span_locator="sms:unreviewed",
                custody_locator="h1:no",
                content_hash="sha256:" + "c" * 64,
                review_state=ReviewState.CANDIDATE,
            ),
        ],
    )
    citation = EvidenceCitation(
        statement="Parenting-time exchange occurred at the ordered location.",
        package_id=package.package_id,
        assertion_id=assertion_id,
        assertion_version=1,
        span_locator="sms:2024-03-12:14:02",
        epistemic_class=EpistemicClass.ESTABLISHED_FACT,
    )
    return package, citation


def test_workspace_walks_matter_import_factor_draft_gate(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    home = workspace.home()
    assert home.matter.display_name
    assert home.court_case.is_primary is True
    assert len(home.factors) == 12
    assert home.factors[0].letter is FactorLetter.A
    assert len(home.issue.children) == 3
    assert home.package is None

    package, citation = _approved_package(home.matter.matter_id)
    imported = workspace.import_package(package)
    assert imported.blocked is True
    assert "D08 producer evidence unavailable" in (imported.reason or "")
    assert workspace.load().package is None
    imported = seed_synthetic_approved_package(workspace, package)
    assert len(imported.accepted.items) == 1
    assert len(imported.omitted_item_ids) == 1

    entry = workspace.attach_factor_citation(
        FactorCitationLink(
            letter=FactorLetter.J,
            side="petitioner",
            citation=citation,
            package_id=package.package_id,
        )
    )
    assert len(entry.petitioner.citations) == 1

    section = workspace.draft_factor_section(
        FactorLetter.J,
        heading="Factor (j) — facilitation",
        body="Dated, documented interference mapped to MCL 722.23(j).",
    )
    assert section.factor_letter is FactorLetter.J
    assert len(section.citations) == 1

    gate = workspace.gate_draft(section.section_id)
    assert gate.ok is True

    review = workspace.add_review(
        ReviewCreate(
            section_id=section.section_id,
            verdict=ReviewVerdict.APPROVE,
            rationale="First-slice owner review.",
        ),
        principal=HUMAN,
    )
    assert review.verdict is ReviewVerdict.APPROVE
    built = workspace.build_release_candidate(ReleaseCreate(section_ids=[section.section_id]))
    assert built.blocked is False
    assert built.manifest is not None
    assert built.manifest.content_hash.startswith("sha256:")


def test_http_first_slice_on_shipped_app(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    home = client.get("/v1/matter")
    assert home.status_code == 200
    body = home.json()
    assert body["matter"]["display_name"]
    assert body["factor_count"] == 12
    paths = {row["path"] for row in body["next_surfaces"]}
    labels = {row["label"] for row in body["next_surfaces"]}
    assert "/custody-factors" in paths
    assert "/review" in paths
    assert "What the judge must consider" in labels
    assert "Your review" in labels
    assert all("cmd" not in row for row in body["next_surfaces"])
    assert body["upcoming_event_count"] == 0
    assert body["judge_confirmed"] is False

    package, citation = _approved_package(store.load().matter.matter_id)
    imported = client.post(
        "/v1/legal-source-packages:import",
        json=package.model_dump(mode="json"),
    )
    assert imported.status_code == 200
    assert imported.json()["blocked"] is True
    assert imported.json()["accepted_item_count"] == 0
    assert "D08 producer evidence unavailable" in imported.json()["reason"]
    assert store.load().package is None
    seed_synthetic_approved_package(store, package)

    linked = client.post(
        "/v1/factors/j/citations",
        json={
            "letter": "j",
            "side": "petitioner",
            "package_id": str(package.package_id),
            "citation": citation.model_dump(mode="json"),
        },
    )
    assert linked.status_code == 200, linked.text
    assert len(linked.json()["petitioner"]["citations"]) >= 1

    drafted = client.post(
        "/v1/drafts",
        json={
            "letter": "j",
            "heading": "Factor (j)",
            "body": "Facilitation shown by dated denials, not by a diagnostic label.",
        },
    )
    assert drafted.status_code == 200
    section_id = drafted.json()["section_id"]
    gated = client.post(f"/v1/drafts/{section_id}:gate")
    assert gated.status_code == 200
    assert gated.json()["ok"] is True

    reviewed = client.post(
        "/v1/reviews",
        json={
            "section_id": section_id,
            "verdict": "approve",
            "rationale": "HTTP first-slice owner review.",
            "reviewer": "owner",
        },
    )
    assert reviewed.status_code == 403, reviewed.text
    released = client.post("/v1/releases", json={"section_ids": [section_id]})
    assert released.status_code == 200
    assert released.json()["blocked"] is True
