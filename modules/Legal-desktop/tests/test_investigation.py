"""EvidenceInvestigationRequest persists and writes the reserved event.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
No invented docket dates. court_safe=false.
SQLite is canonical; JSON debug mirror is opt-in via LEGAL_WORKSPACE_DEBUG_JSON.
"""

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.contracts.events import KNOWN_EVENT_TYPES
from legal_workspace.domain.factors import FactorLetter
from legal_workspace.domain.investigation import (
    EVIDENCE_REQUEST_CREATED,
    InvestigationCreate,
    InvestigationKind,
    to_created_event,
)
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_blank_workspace_has_no_investigations_or_docket_dates(tmp_path) -> None:
    state = Workspace(tmp_path).load()
    assert state.investigations == []
    assert state.docket_events == []


def test_request_persists_and_is_not_court_safe(tmp_path) -> None:
    first = Workspace(tmp_path)
    added = first.add_investigation(
        InvestigationCreate(
            needed="Last final custody / parenting-time order text and entry date.",
            why="Vodvarka lookback cannot start until the last order is clerk-confirmed.",
            kind=InvestigationKind.MISSING_PROOF,
            linked_issue="Proper cause or change of circumstances (if a final order exists)",
            factor_letter=FactorLetter.J,
        )
    )
    assert added.court_safe is False
    assert added.exportable is False
    assert added.event_type == EVIDENCE_REQUEST_CREATED
    assert added.event_type in KNOWN_EVENT_TYPES

    reloaded = Workspace(tmp_path)
    stored = reloaded.list_investigations()
    assert len(stored) == 1
    assert stored[0].request_id == added.request_id
    assert stored[0].court_safe is False
    assert stored[0].needed.startswith("Last final")
    assert reloaded.load().docket_events == []

    # The JSON event log is opt-in; verify the event envelope can still be built.
    envelope = to_created_event(
        stored[0],
        matter_id=reloaded.load().matter.matter_id,
        court_case_id=reloaded.load().court_case.court_case_id,
        aggregate_version=1,
    )
    assert envelope.event_type == EVIDENCE_REQUEST_CREATED
    envelope.fail_closed_if_unknown()


def test_http_investigations(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    listed = client.get("/v1/investigations")
    assert listed.status_code == 200
    assert listed.json() == []
    created = client.post(
        "/v1/investigations",
        json={
            "needed": "School attendance records for the current semester.",
            "why": "Factor (h) has no accepted package spans yet.",
            "kind": "missing_proof",
            "linked_issue": "What the judge must consider (a)–(l) considered individually",
            "factor_letter": "h",
        },
    )
    assert created.status_code == 200, created.text
    body = created.json()
    assert body["court_safe"] is False
    assert body["exportable"] is False
    assert body["event_type"] == EVIDENCE_REQUEST_CREATED
    assert body["factor_letter"] == "h"
    home = client.get("/v1/matter")
    assert home.status_code == 200
    assert any(row["path"] == "/missing-evidence" and row["label"] == "Missing evidence" for row in home.json()["next_surfaces"])
    assert home.json()["investigation_count"] == 1
    assert home.json()["upcoming_event_count"] == 0
    relisted = client.get("/v1/investigations")
    assert len(relisted.json()) == 1
