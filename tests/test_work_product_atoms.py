"""Adversarial synthetic tests for D06 mention identity and overlap accounting.

Byline: Codex · GPT-6 · 2026-09-23.
"""

from __future__ import annotations

import pytest
from pydantic import ValidationError

from server.work_product.atoms import (
    AtomObservation,
    MentionAtom,
    SourceSpan,
    reconcile_observations,
)


def atom(start: int, end: int, **changes: object) -> MentionAtom:
    values = {
        "source_occurrence_id": "conversation-1",
        "source_version_id": "version-1",
        "provenance_origin_id": "origin-1",
        "spans": (SourceSpan(turn_id="turn-16", char_start=start, char_end=end),),
        "mention_key": "assertion-1",
        "record_class": "EVENT",
        "statement": "The speaker reported filing a motion.",
        "origin_class": "HUMAN_SOURCE",
    }
    values.update(changes)
    return MentionAtom.model_validate(values)


def observation(candidate: MentionAtom, run: str, window: str) -> AtomObservation:
    return AtomObservation(run_id=run, window_id=window, atom=candidate)


def test_two_mentions_in_one_turn_have_distinct_ids_and_exact_spans() -> None:
    first = atom(0, 8)
    second = atom(16, 27)
    same_span_second_assertion = atom(0, 8, mention_key="assertion-2")
    assert len({first.atom_id, second.atom_id, same_span_second_assertion.atom_id}) == 3


def test_overlap_and_retry_keep_observations_but_one_atom() -> None:
    candidate = atom(0, 8)
    result = reconcile_observations(
        (
            observation(candidate, "run-1", "window-1"),
            observation(candidate, "run-1", "window-2"),
            observation(candidate, "run-2", "window-1"),
        )
    )
    assert result.observations_seen == 3
    assert len(result.atoms) == 1
    assert result.atoms[0].observation_count == 3
    assert result.atoms[0].delivery_count == 3
    assert result.atoms[0].atom.provenance_origin_id == "origin-1"
    assert result.atoms[0].atom.spans[0].char_end == 8
    assert result.conflicts == ()


def test_redelivered_same_run_window_does_not_inflate_observation_count() -> None:
    candidate = atom(0, 8)
    result = reconcile_observations(
        (
            observation(candidate, "run-1", "window-1"),
            observation(candidate, "run-1", "window-1"),
        )
    )
    assert result.observations_seen == 2
    assert result.atoms[0].delivery_count == 2
    assert result.atoms[0].observation_count == 1


def test_source_version_and_occurrence_are_identity_boundaries() -> None:
    original = atom(0, 8)
    assert original.atom_id != atom(0, 8, source_version_id="version-2").atom_id
    assert original.atom_id != atom(0, 8, source_occurrence_id="conversation-2").atom_id
    assert original.atom_id == atom(0, 8, statement="A different paraphrase.").atom_id


def test_same_source_screenshot_and_summary_do_not_become_three_origins() -> None:
    original = atom(0, 8)
    screenshot = atom(0, 8, source_occurrence_id="screenshot", provenance_origin_id="origin-1")
    summary = atom(0, 8, source_occurrence_id="summary", provenance_origin_id="origin-1")
    result = reconcile_observations(
        tuple(
            observation(candidate, "run-1", name)
            for candidate, name in ((original, "original"), (screenshot, "screenshot"), (summary, "summary"))
        )
    )
    assert len(result.atoms) == 3  # distinct occurrences stay visible
    assert {item.atom.provenance_origin_id for item in result.atoms} == {"origin-1"}


def test_same_identity_disagreement_is_a_retained_conflict() -> None:
    result = reconcile_observations(
        (
            observation(atom(0, 8), "run-1", "window-1"),
            observation(atom(0, 8, statement="Conflicting extraction."), "run-2", "window-1"),
        )
    )
    assert result.observations_seen == 2
    assert result.atoms == ()
    assert len(result.conflicts) == 1
    assert len(result.conflicts[0].statements) == 2
    assert {item.run_id for item in result.conflicts[0].observations} == {"run-1", "run-2"}


@pytest.mark.parametrize("start,end", [(-1, 2), (2, 2), (3, 2)])
def test_invalid_source_offsets_fail_closed(start: int, end: int) -> None:
    with pytest.raises(ValidationError):
        SourceSpan(turn_id="turn-16", char_start=start, char_end=end)


def test_missing_version_or_spans_fail_closed() -> None:
    with pytest.raises(ValidationError):
        atom(0, 8, source_version_id=" ")
    with pytest.raises(ValidationError):
        atom(0, 8, spans=())


def test_ai_origin_is_explicit_and_cannot_be_inferred_as_human() -> None:
    ai = atom(0, 8, origin_class="AI_ORIGIN")
    assert ai.origin_class == "AI_ORIGIN"
    result = reconcile_observations((observation(ai, "run-1", "window-1"),))
    assert result.atoms[0].atom.origin_class == "AI_ORIGIN"
    with pytest.raises(ValidationError):
        atom(0, 8, origin_class="ADOPTED")


def test_same_identity_with_conflicting_origin_is_held_for_review() -> None:
    result = reconcile_observations(
        (
            observation(atom(0, 8), "run-1", "window-1"),
            observation(atom(0, 8, provenance_origin_id="origin-2"), "run-2", "window-1"),
        )
    )
    assert result.atoms == ()
    assert result.conflicts[0].provenance_origin_ids == ("origin-1", "origin-2")
