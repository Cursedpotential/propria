"""Regress operation artifact availability, terminal gates and exact-source fingerprints."""

import asyncio
from uuid import uuid4

import pytest
from pydantic import ValidationError

from app.service import proffer_operator
from app.types.proffer import ProfferPreviewResponse
from tests.test_proffer_operator_surface import HANDLE, _operation, _preview, _wire


@pytest.mark.parametrize("phase", ["awaiting_handler_selection", "awaiting_repair_decision", "awaiting_decision"])
@pytest.mark.parametrize("lifecycle", ["cancelled", "failed", "completed", "rerun_required", "unavailable"])
def test_terminal_or_unknown_lifecycle_overrides_stale_gate(phase, lifecycle):
    actions = proffer_operator._valid_actions(phase, lifecycle, mutation_eligible=True)
    assert not {"select_handler", "retain_original", "approve_preview", "reject_preview"} & {action.action for action in actions}


def test_unknown_mode_never_offers_a_mutation():
    assert [action.action for action in proffer_operator._valid_actions("awaiting_handler_selection", "running")] == ["refresh"]


def artifact_payload():
    source, raw, normalized = str(uuid4()), str(uuid4()), str(uuid4())
    return {
        "preview_handle": HANDLE, "phase": "failed", "lifecycle": "failed", "terminal": True,
        "matter_mode": "LIVE", "receipts": [],
        "correlation": {"request_id": "request-1", "source_version_id": source, "raw_generation_id": raw, "normalized_generation_id": normalized},
        "results": {"status": "available", "source_version_id": source, "raw_generation_id": raw, "normalized_generation_id": normalized, "raw_receipt_ref": "raw-receipt", "normalized_receipt_ref": "normalized-receipt", "preview_available": False},
        "fingerprint": {"status": "available", "source_version_id": source, "hash_kind": "context_source_fingerprint", "algorithm": "sha256", "digest": "a" * 64, "construction": "context-source-fingerprint-v1", "computed_at": "2026-10-15T12:00:00Z", "computed_by": "fixture", "receipt_ref": "hash-receipt", "activity_receipt_ref": "fingerprint-stage-receipt"},
    }


def test_persisted_generation_remains_readable_after_downstream_failure(monkeypatch):
    preview = ProfferPreviewResponse.model_validate(artifact_payload())
    _wire(monkeypatch, preview, _operation("failed", "publish_context_search_activity"))
    result = asyncio.run(proffer_operator.operator_snapshot(HANDLE, mode="LIVE"))
    assert result.package.normalized_products.status == "available"
    assert result.surfaces["records"].status == "available"
    assert result.package.original_fingerprint.status == "available"
    assert result.package.original_fingerprint.ref == "a" * 64
    assert not {"approve_preview", "reject_preview"} & {action.action for action in result.valid_actions}


def test_artifacts_cannot_become_approval_or_cross_generation_preview():
    payload = artifact_payload()
    payload["phase"] = "awaiting_decision"
    with pytest.raises(ValidationError, match="cannot authorize"):
        ProfferPreviewResponse.model_validate(payload)
    payload["phase"] = "failed"
    payload["correlation"]["normalized_generation_id"] = str(uuid4())
    with pytest.raises(ValidationError, match="exact operation"):
        ProfferPreviewResponse.model_validate(payload)


def test_fingerprint_cannot_refer_to_another_source_or_generation_hash():
    payload = artifact_payload()
    payload["fingerprint"]["source_version_id"] = str(uuid4())
    with pytest.raises(ValidationError, match="exact source bytes"):
        ProfferPreviewResponse.model_validate(payload)
    payload = artifact_payload()
    payload["fingerprint"]["hash_kind"] = "context_raw_generation_fingerprint"
    with pytest.raises(ValidationError, match="exact source bytes"):
        ProfferPreviewResponse.model_validate(payload)


def test_inconsistent_receipt_is_unavailable_with_reason(monkeypatch):
    preview = _preview("failed", "failed")
    payload = artifact_payload()
    payload.pop("correlation")
    payload.pop("receipts")
    payload["results"]["status"] = "inconsistent"
    payload["results"]["reason"] = "Persisted normalized-generation reference has no matching generation."
    preview = ProfferPreviewResponse.model_validate(payload)
    _wire(monkeypatch, preview, _operation("failed"))
    result = asyncio.run(proffer_operator.operator_snapshot(HANDLE, mode="LIVE"))
    assert result.surfaces["records"].status == "unavailable"
    assert "no matching generation" in result.surfaces["records"].reason
    assert result.surfaces["records"].count is None


def test_ai_chat_uses_unqueried_native_route_without_pg_generation_requirement(monkeypatch):
    payload = artifact_payload()
    source = payload["results"]["source_version_id"]
    payload.pop("correlation")
    payload.pop("receipts")
    payload["results"] = {"status": "not_queried", "content_store": "weaviate", "source_kind": "ai_chat", "source_ref": "b2://synthetic/conversations.json", "source_version_id": source, "preview_available": False, "reason": "Historical operation has no verified current native index reference; Weaviate availability was not queried."}
    preview = ProfferPreviewResponse.model_validate(payload)
    _wire(monkeypatch, preview, _operation("failed"))
    result = asyncio.run(proffer_operator.operator_snapshot(HANDLE, mode="LIVE"))
    assert result.storage_state.context_target.ref == "weaviate"
    assert result.surfaces["records"].count is None
    assert "not queried" in result.surfaces["records"].reason
    assert not {"select_handler", "retain_original", "approve_preview", "reject_preview"} & {action.action for action in result.valid_actions}
    payload["results"]["raw_generation_id"] = str(uuid4())
    with pytest.raises(ValidationError, match="never PostgreSQL chat"):
        ProfferPreviewResponse.model_validate(payload)
