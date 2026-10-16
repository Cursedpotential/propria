"""Regress unavailable analysis and failed-parent extraction history boundaries.

Inputs: synthetic upstream responses. Outputs: API assertions; effects: no jobs.
Pick for unavailable-versus-empty results, independently of output persistence.
"""
import asyncio
from types import SimpleNamespace
import httpx
from app.runtime import analysis
from app.service import conversation_actions, proffer, imported


def test_missing_analysis_catalog_is_explicit_unavailable(monkeypatch):
    """Preserve a supported unavailable state for a missing deployed catalog route."""
    monkeypatch.setattr(analysis, "_case", lambda: {})
    monkeypatch.setattr(analysis, "_actor_headers", lambda *a: {})
    async def missing(*a, **kw):
        raise proffer.ProfferError("not found", 404)
    monkeypatch.setattr(proffer, "_request", missing)
    result = asyncio.run(analysis.analysis_projections(SimpleNamespace(), ""))
    assert result["available"] is False and result["items"] == []
    assert "unavailable" in result["reason"]


def test_history_survives_when_no_output_run_exists(monkeypatch):
    """Return retained failed parents even if extraction_run is empty."""
    monkeypatch.setattr(imported, "live_matter", lambda: "matter")
    async def upstream(method, path, **kw):
        assert kw["params"] == {"matter_id": "matter", "export_key": "exact-source", "conv": "exact-conversation"}
        return httpx.Response(200, json={"available": True, "truncated": False, "coverage": "retained only", "attempts": [{"workflow_id": "conversation-extraction:failed", "run_id": "exact-run", "progress": {"outcome": "failed"}}]})
    monkeypatch.setattr(proffer, "_request", upstream)
    result = asyncio.run(conversation_actions.extraction_history("exact-source", "exact-conversation"))
    assert result["attempts"][0]["run_id"] == "exact-run"
