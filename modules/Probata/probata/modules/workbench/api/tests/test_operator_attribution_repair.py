"""Regress explicit speaker attribution and bounded existing-key overlap.

Inputs: synthetic rows. Outputs: assertions; effects: no database or model jobs.
Pick for the audit's role-as-direction and first-participant fallback regressions.
"""
from app.service import imported
from app.repo import case_identity_catalog as catalog


def test_legal_role_never_determines_direction_and_unknown_stays_unknown():
    """Keep legal roles independent of raw normalized speaker/recipient roles."""
    people = imported._People([{"person": "Person", "display_name": "Person", "role_in_case": "user", "entity_id": "p", "verification_state": "confirmed", "identifier": "5555550101", "kind": "phone"}])
    row = {"id": "record", "occurred_at": None, "body": "text", "projection_kind": "third_party", "attachment_count": 0, "has_attachments": False, "certainty": None}
    message = imported._message({**row, "participants": [{"role": "unknown", "identifier": "5555550101"}]}, people, "Other owner")
    assert message["sender"]["label"] == "Unknown"
    assert message["attribution"]["direction"] == "unknown"
    message = imported._message({**row, "participants": [{"role": "sender", "identifier": "5555550101"}, {"role": "recipient", "identifier": "self"}]}, people, "Other owner")
    assert message["sender"]["label"] == "Person" and not message["outgoing"]
    assert message["attribution"]["direction"] == "received"
    message = imported._message({**row, "participants": [{"role": "sender", "identifier": "self"}]}, people, "Other owner")
    assert message["outgoing"] and message["attribution"]["direction"] == "sent"


def test_catalog_overlap_uses_record_keys_and_reports_truncation(monkeypatch):
    """Count overlapping provenance keys without comparing or removing text."""
    monkeypatch.setattr(catalog.intake_discovery, "_query", lambda *a: [{"dedup_key": "a", "sender_match": True, "counterparty_match": True}, {"dedup_key": "a", "sender_match": True, "counterparty_match": False}, {"dedup_key": "b", "sender_match": False, "counterparty_match": True}])
    result = catalog.overlap("5555550101")
    assert result["unique_records"] == 2 and result["overlap"] == 1
    monkeypatch.setattr(catalog.intake_discovery, "_query", lambda *a: [{}] * 10001)
    assert catalog.overlap("5555550101")["truncated"]
