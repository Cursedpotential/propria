"""Verify reference-only memory reconciliation contracts without touching production data.

Inputs: synthetic records/requests. Outputs: assertions for scope, fingerprints and fail-closed auth.
Effects: no source mutations, network, database transactions or workflow deployments.
"""
from __future__ import annotations

import pytest
from fastapi import HTTPException
from pydantic import ValidationError

from server.api import knowledge_reconcile_routes as kr


def test_reference_only_proposal_dedup_key_is_revision_sensitive():
    candidate = kr.Candidate(
        kind="possible_topic_conflict", scope="propria",
        source_refs=["engram:conversation-one", "memsearch:note-one"],
        source_revisions=[
            {"ref":"engram:conversation-one", "revision":"3"},
            {"ref":"memsearch:note-one", "revision":"2"}],
        severity="high", summary="Two versions require human verification",
        idempotency_key="same-reference-pair", proposed_action="review", automatic=False)
    refs, revisions, fp = kr._only_refs(candidate)
    assert refs == ["engram:conversation-one", "memsearch:note-one"]
    assert len(fp) == 64
    updated = candidate.model_copy(update={
        "source_revisions": [
            kr.Revision(ref="engram:conversation-one", revision="4"),
            kr.Revision(ref="memsearch:note-one", revision="2")]
    })
    assert kr._only_refs(updated)[2] != fp


def test_mismatched_source_pins_rejected_before_db_write():
    candidate = kr.Candidate(
        kind="duplicate_active_claim", scope="propria",
        source_refs=["memsearch:a"],
        source_revisions=[{"ref":"memsearch:b", "revision":"1"}],
        severity="low", summary="Claim appears twice",
        idempotency_key="duplicate-claim-id", proposed_action="review", automatic=False)
    with pytest.raises(HTTPException) as error:
        kr._only_refs(candidate)
    assert error.value.status_code == 422


def test_unknown_source_and_automatic_action_not_promoted():
    with pytest.raises(HTTPException):
        kr._only_refs(kr.Candidate(
            kind="possible_topic_conflict",scope="propria",source_refs=["invented:a"],
            source_revisions=[{"ref":"invented:a","revision":"1"}],
            severity="high",summary="Review possible conflict",
            idempotency_key="unknown-source",proposed_action="review",automatic=False))
    with pytest.raises(ValidationError):
        kr.Candidate.model_validate({
            "kind":"duplicate_active_claim","scope":"propria",
            "source_refs":["memsearch:a"],"source_revisions":[{"ref":"memsearch:a","revision":"1"}],
            "severity":"low","summary":"Review possible duplication",
            "idempotency_key":"x","proposed_action":"review","automatic":True})


@pytest.mark.asyncio
async def test_unavailable_source_does_not_claim_coverage(monkeypatch):
    monkeypatch.setattr(kr,"_configured_adapters",lambda:{})
    trigger=kr.Trigger(schema="propria-knowledge-trigger/v1",
       kind="scheduled",scope="propria",run_id="n8n-scan-test")
    result=await kr._snapshot(trigger)
    assert result["schema"]=="propria-knowledge-snapshot/v1"
    assert result["records"]==[]
    assert len(result["coverage"])==len(kr.SOURCES)
    assert all(row["state"]=="unavailable" for row in result["coverage"])


def test_owner_verdict_fails_closed_without_distinct_decision_capability(monkeypatch):
    from types import SimpleNamespace
    monkeypatch.delenv("KNOWLEDGE_REVIEW_DECISION_TOKEN_FILE",raising=False)
    assert kr._reviewer_token_ok(SimpleNamespace(headers={
        "x-knowledge-review-owner-token":"fabricated"})) is False
