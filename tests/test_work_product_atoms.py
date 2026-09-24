"""Adversarial synthetic tests for D06 mention identity and overlap accounting.

Byline: Codex · GPT-6 · 2026-09-23 (Unicode coordinate-contract remediation).
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


def test_unicode_code_point_boundaries_are_explicit_for_non_bmp_and_combining_marks() -> None:
    source_text = "A\U0001f600e\u0301Z"
    emoji = SourceSpan(turn_id="turn-unicode", char_start=1, char_end=2)
    decomposed_grapheme = SourceSpan(turn_id="turn-unicode", char_start=2, char_end=4)

    assert source_text[emoji.char_start : emoji.char_end] == "\U0001f600"
    assert source_text[decomposed_grapheme.char_start : decomposed_grapheme.char_end] == "e\u0301"
    assert emoji.model_dump(mode="json") == {
        "turn_id": "turn-unicode",
        "char_start": 1,
        "char_end": 2,
        "coordinate_unit": "unicode_code_point",
        "normalization": "none",
    }
    assert decomposed_grapheme.char_end - decomposed_grapheme.char_start == 2


@pytest.mark.parametrize(
    ("field", "value"),
    (("coordinate_unit", "utf16_code_unit"), ("normalization", "NFC")),
)
def test_source_span_rejects_alternate_coordinate_contracts(field: str, value: str) -> None:
    values = {"turn_id": "turn-unicode", "char_start": 1, "char_end": 2, field: value}
    with pytest.raises(ValidationError):
        SourceSpan.model_validate(values)


def test_unicode_coordinate_contract_is_bound_to_deterministic_atom_id() -> None:
    unicode_span = SourceSpan(turn_id="turn-unicode", char_start=1, char_end=4)
    candidate = atom(0, 8, spans=(unicode_span,), mention_key="unicode-assertion")

    assert candidate.atom_id == "wp-atom-v1:327061dd64456ef4735359a67356b74fb678e01d57e49b8800f6ffa4582c198b"
    assert MentionAtom.model_validate(candidate.model_dump(mode="json")).atom_id == candidate.atom_id


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


def test_span_order_is_canonical_without_changing_exact_coordinates() -> None:
    first = SourceSpan(turn_id="turn-16", char_start=0, char_end=8)
    second = SourceSpan(turn_id="turn-17", char_start=11, char_end=19)
    forward = atom(0, 8, spans=(first, second))
    reverse = atom(0, 8, spans=(second, first))
    assert forward.atom_id == reverse.atom_id
    assert forward.spans == reverse.spans == (first, second)
    assert forward.model_dump(mode="json") == reverse.model_dump(mode="json")


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


def test_record_class_disagreement_is_one_explicit_retained_conflict() -> None:
    event = atom(0, 8)
    statement = atom(0, 8, record_class="STATEMENT")
    assert event.atom_id == statement.atom_id
    result = reconcile_observations(
        (
            observation(statement, "run-2", "window-1"),
            observation(event, "run-1", "window-1"),
        )
    )
    assert result.atoms == ()
    assert len(result.conflicts) == 1
    conflict = result.conflicts[0]
    assert conflict.record_classes == ("EVENT", "STATEMENT")
    assert tuple(item.atom.record_class for item in conflict.observations) == ("EVENT", "STATEMENT")
    assert conflict.observation_coordinates == (("run-1", "window-1"), ("run-2", "window-1"))


def test_reconciliation_is_identical_under_input_reversal() -> None:
    first = SourceSpan(turn_id="turn-16", char_start=0, char_end=8)
    second = SourceSpan(turn_id="turn-17", char_start=11, char_end=19)
    inputs = (
        observation(atom(0, 8, spans=(second, first)), "run-2", "window-2"),
        observation(atom(0, 8, spans=(first, second)), "run-1", "window-1"),
        observation(atom(20, 28, record_class="EVENT"), "run-3", "window-2"),
        observation(atom(20, 28, record_class="STATEMENT"), "run-3", "window-1"),
    )
    forward = reconcile_observations(inputs)
    reverse = reconcile_observations(tuple(reversed(inputs)))
    assert forward.model_dump(mode="json") == reverse.model_dump(mode="json")
    assert len(forward.atoms) == len(forward.conflicts) == 1
    assert forward.atoms[0].atom.spans == (first, second)


@pytest.mark.parametrize("start,end", [(-1, 2), (2, 2), (3, 2)])
def test_invalid_source_offsets_fail_closed(start: int, end: int) -> None:
    with pytest.raises(ValidationError):
        SourceSpan(turn_id="turn-16", char_start=start, char_end=end)


def test_missing_version_or_spans_fail_closed() -> None:
    with pytest.raises(ValidationError):
        atom(0, 8, source_version_id=" ")
    with pytest.raises(ValidationError):
        atom(0, 8, spans=())


@pytest.mark.parametrize("field", ["turn_id", "run_id", "window_id"])
def test_whitespace_only_coordinates_fail_closed(field: str) -> None:
    with pytest.raises(ValidationError):
        if field == "turn_id":
            SourceSpan(turn_id=" \t ", char_start=0, char_end=8)
        else:
            observation(
                atom(0, 8), " \t " if field == "run_id" else "run-1", " \t " if field == "window_id" else "window-1"
            )


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
