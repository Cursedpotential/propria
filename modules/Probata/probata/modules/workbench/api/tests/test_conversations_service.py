"""Bout review label shapes and input checks. Byline: Claude Code · Opus 5.5 · 2026-09-24."""
import pytest
from app.repo.intake_discovery import DiscoveryError
from app.service import conversations as service


def test_stretches_shape_keeps_segments_shifts_and_summary():
    out = service.normalize_label({
        "stretches": [{"from_i": 0, "to_i": 2, "tone": "tense", "driver": "A", "note": "n"},
                      {"from_i": 3, "to_i": 3, "tone": "hostile", "driver": "B"}],
        "shifts": [{"at_i": 3, "from_tone": "tense", "to_tone": "hostile", "speed": "abrupt", "trigger_i": 2, "note": "x"}],
        "summary": "s"})
    assert out["shape"] == "stretches"
    assert out["segments"] == [[0, 2, "tense", "A", "n"], [3, 3, "hostile", "B", ""]]
    assert out["shifts"] == [[3, "tense", "hostile", "abrupt", 2, "x"]]
    assert out["summary"] == "s"


def test_overlapping_windows_take_the_latest_window_and_merge_runs():
    def w(a, b, tone):
        return {"from_i": a, "to_i": b, "answers": {"main_tone": {"choice": tone, "confidence": 0.9}}}
    out = service.normalize_label({"windows": [w(0, 3, "neutral"), w(2, 5, "tense"), w(6, 6, "tense")]})
    assert out["shape"] == "windows"
    assert [s[:3] for s in out["segments"]] == [[0, 1, "neutral"], [2, 6, "tense"]]
    assert [s[:3] for s in out["shifts"]] == [[2, "neutral", "tense"]]


def test_unknown_and_missing_labels_are_flagged_not_guessed():
    assert service.normalize_label({"something": 1})["shape"] == "unknown"
    assert service.normalize_label(None)["shape"] is None


def test_day_must_be_a_date_before_any_query(monkeypatch):
    monkeypatch.setattr(service.repo, "conversation", lambda key: pytest.fail("queried"))
    with pytest.raises(DiscoveryError, match="YYYY-MM-DD"):
        service.day("x", "rules", "2024-07-27'; drop")
