"""Privilege first-pass is keyword-only and never a legal conclusion.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from uuid import uuid4

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.factors import FactorLetter
from legal_workspace.domain.privilege import (
    DISCLAIMER,
    MarkerKind,
    PrivilegeScan,
    scan_text,
)
from legal_workspace.services import workspace as workspace_mod


def test_scan_text_hits_each_hypothesized_kind() -> None:
    scan = scan_text(
        "This is attorney-client material and attorney work product. "
        "Litigation strategy and settlement posture stay private. "
        "Do not file the child's therapist notes or medical records. "
        "The child's date of birth and social security number are identifying."
    )
    kinds = set(scan.kinds_hit)
    assert kinds == {
        MarkerKind.ATTORNEY_CLIENT,
        MarkerKind.WORK_PRODUCT,
        MarkerKind.STRATEGY,
        MarkerKind.MEDICAL,
        MarkerKind.CHILD_IDENTIFYING,
    }
    labels = {item.label for item in scan.hypothesized_markers}
    assert labels == {
        "attorney-client",
        "work product",
        "strategy",
        "medical",
        "child-identifying",
    }
    assert scan.court_safe is False
    assert scan.legal_conclusion is False
    assert scan.exportable is False
    assert scan.method == "keyword_first_pass"
    assert "not a privilege" in scan.disclaimer.lower()


def test_clean_text_is_not_a_clearance() -> None:
    scan = scan_text("The parenting-time exchange occurred at the library.")
    assert scan.hypothesized_markers == []
    assert scan.kinds_hit == []
    assert scan.court_safe is False
    assert scan.legal_conclusion is False
    assert "not a privilege" in DISCLAIMER.lower()


def test_cannot_assert_privilege_as_legal_conclusion() -> None:
    forced = PrivilegeScan(
        hypothesized_markers=[],
        court_safe=True,
        legal_conclusion=True,
        exportable=True,
    )
    assert forced.court_safe is False
    assert forced.legal_conclusion is False
    assert forced.exportable is False
    dumped = forced.model_dump()
    assert "privilege_claimed" not in dumped
    assert "is_privileged" not in dumped
    assert dumped["legal_conclusion"] is False


def test_http_scan_text_and_section(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)

    empty = client.post("/v1/privilege:scan", json={})
    assert empty.status_code == 400

    missing = client.post("/v1/privilege:scan", json={"section_id": str(uuid4())})
    assert missing.status_code == 404

    pasted = client.post(
        "/v1/privilege:scan",
        json={"text": "Privileged and confidential advice of counsel."},
    )
    assert pasted.status_code == 200, pasted.text
    body = pasted.json()
    assert body["court_safe"] is False
    assert body["legal_conclusion"] is False
    assert body["source"] == "text"
    assert "attorney_client" in body["kinds_hit"]
    assert any(item["label"] == "attorney-client" for item in body["hypothesized_markers"])

    section = store.draft_factor_section(
        FactorLetter.J,
        "Facilitation notes",
        "Do not disclose this strategy. Child's school records stay out.",
    )
    by_section = client.post(
        "/v1/privilege:scan",
        json={"section_id": str(section.section_id)},
    )
    assert by_section.status_code == 200, by_section.text
    section_body = by_section.json()
    assert section_body["source"] == "section"
    assert section_body["section_id"] == str(section.section_id)
    assert section_body["court_safe"] is False
    assert section_body["legal_conclusion"] is False
    assert "strategy" in section_body["kinds_hit"]
    assert "child_identifying" in section_body["kinds_hit"]

    home = client.get("/v1/matter")
    assert any(row["path"] == "/confidentiality-check" and row["label"] == "Confidentiality check" for row in home.json()["next_surfaces"])
