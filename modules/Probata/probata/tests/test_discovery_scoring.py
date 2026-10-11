"""Verify multi-label CPU retrieval, abstention and reversible scoped feedback."""

from copy import deepcopy

import pytest

from server.analysis.discovery.scoring import ABSTENTION_PROTOTYPES, score_batch
from server.analysis.discovery.storage import DiscoveryInvalid
from tests.test_discovery_context import record


class FixtureEncoder:
    """Supply independent fixed semantic vectors to exercise retrieval policy.

    Inputs: fixture text. Outputs: vectors. Effects: tracks bounded calls; choose
    deterministic test vectors over downloading a model or using live providers.
    """

    def __init__(self, values=None):
        self.values = {
            "threat description": [1, 0, 0],
            "control description": [0, 1, 0],
            "reported threat": [1, 0, 0],
            "neutral": [0, 0, 1],
            "negated threat": [0, 0, 1],
            "ambiguous": [0.7, 0.7, 0],
            "pressure": [0, 1, 0],
            "similar threat": [0.9, 0, 0.1],
        }
        self.values.update(values or {})
        self.calls = []
        self.values.update({text: [0, 0, 1] for text in ABSTENTION_PROTOTYPES})

    def encode(self, sentences, **kwargs):
        """Return fixed fixture embeddings without relying on scoring internals.

        Inputs: sentences/options. Outputs: vectors. Effects: records batch size;
        choose for contract tests that prohibit hidden model/provider loading.
        """
        self.calls.append((list(sentences), kwargs))
        return [self.values[text] for text in sentences]


def batch(text="reported threat"):
    """Build one explicitly scoped immutable discovery fixture.

    Inputs: synthetic text. Outputs: batch. Effects: none; choose for review and
    feedback policy tests without canonical writes.
    """
    return {
        "schema": "adaptive-discovery.batch.v1",
        "model_version": "discovery-cpu-v1",
        "run_id": "run",
        "dataset_id": "dataset",
        "selection_revision": "selection-v1",
        "identity_revision": "identity-v1",
        "input_revision": "input-v1",
        "group_revision": "group-v1",
        "feedback_revision": "feedback-v1",
        "scope": {"mode": "LIVE", "matter_id": "matter", "court_case_id": "case"},
        "records": [record("one", text=text)],
        "feedback": [],
        "hypotheses": [
            {"hypothesis_id": "threat", "category": "safety", "text": "threat description"},
            {"hypothesis_id": "control", "category": "coercion", "text": "control description"},
        ],
        "config": {"threshold": 0.65, "feedback_weight": 0.2},
    }


def feedback(kind, **fields):
    """Build a human-authored feedback exemplar in a synthetic immutable snapshot.

    Inputs: feedback kind/corrections. Outputs: dictionary. Effects: none; choose
    explicit reviewer identity over model-generated self-agreement.
    """
    return {
        "feedback_id": "feedback-one",
        "revision": 1,
        "kind": kind,
        "actor": {"subject_uid": "reviewer"},
        "created_at": "2026-01-01T10:00:00Z",
        **fields,
    }


@pytest.mark.parametrize("text", ["neutral", "negated threat"])
def test_neutral_and_negation_return_explicit_zero_label_abstention(text):
    result = score_batch(batch(text), encoder=FixtureEncoder())
    assert result["candidates"] == []
    assert result["counts"]["zero_labels"] == result["counts"]["abstained"] == 1


def test_reported_threat_is_relevant_without_inventing_a_behavioral_actor():
    snapshot = batch()
    snapshot["records"][0]["speech_form"] = "reported"
    candidate = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert candidate["labels"] == ["threat"]
    assert candidate["speaker_ids"] == ["speaker-a"]
    assert candidate["attributed_actor_ids"] == []
    citation = candidate["citations"][0]
    assert citation["quote"] == snapshot["records"][0]["text"][citation["start_char"] : citation["end_char"]]
    assert citation["native_locator"] == snapshot["records"][0]["locators"][0]


def test_partial_label_rejection_adjusts_only_rejected_label_and_reverses():
    snapshot = batch("ambiguous")
    baseline = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    snapshot["feedback"] = [feedback("reject", labels=["threat"], example_text="ambiguous")]
    adjusted = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert baseline["labels"] == ["control", "threat"]
    assert adjusted["labels"] == ["control"]
    assert adjusted["label_scores"]["control"] == baseline["label_scores"]["control"]
    snapshot["feedback"] = []
    assert score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0] == baseline


def test_semantically_similar_feedback_generalizes_only_within_scoped_label():
    snapshot = batch("similar threat")
    before = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    snapshot["feedback"] = [feedback("reject", labels=["threat"], example_text="reported threat")]
    after = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert after["score"] < before["score"]
    assert after["label_scores"]["control"] == before["label_scores"]["control"]
    snapshot["feedback"][0]["scope"] = {"mode": "LIVE", "matter_id": "different"}
    with pytest.raises(DiscoveryInvalid, match="scope"):
        score_batch(snapshot, encoder=FixtureEncoder())


def test_wrong_person_and_reported_speech_change_attribution_not_behavior_score():
    snapshot = batch()
    initial = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    snapshot["feedback"] = [
        feedback(
            "wrong_person",
            candidate_id=initial["candidate_id"],
            labels=["threat"],
            example_text="reported threat",
            attributed_actor_ids=["third-party"],
        ),
        feedback("reported_speech", record_ids=["one"], feedback_id="form-two"),
    ]
    adjusted = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert adjusted["score"] == initial["score"]
    assert adjusted["attributed_actor_ids"] == ["third-party"]
    assert "reported" in adjusted["speech_forms"]


@pytest.mark.parametrize("kind", ["skip", "partial_match", "missing_context", "free_text"])
def test_unreviewed_or_non_label_feedback_does_not_become_negative(kind):
    snapshot = batch()
    before = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    snapshot["feedback"] = [feedback(kind, labels=["threat"], example_text="reported threat")]
    after = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert after["label_scores"] == before["label_scores"]


def test_encoder_calls_are_bounded_and_all_revision_pins_are_retained():
    snapshot = batch()
    snapshot["config"]["batch_size"] = 2
    encoder = FixtureEncoder()
    result = score_batch(snapshot, encoder=encoder)
    assert all(len(texts) <= 2 for texts, _ in encoder.calls)
    assert result["pins"]["identity_revision"] == "identity-v1"
    snapshot["config"]["max_window_chars"] = 2
    snapshot["config"]["max_windows"] = 1
    with pytest.raises(DiscoveryInvalid, match="windows"):
        score_batch(snapshot, encoder=FixtureEncoder())


def test_feedback_from_model_agreement_and_future_human_feedback_are_rejected():
    snapshot = batch()
    snapshot["feedback"] = [feedback("confirm", source="model", labels=["threat"])]
    with pytest.raises(DiscoveryInvalid, match="human"):
        score_batch(snapshot, encoder=FixtureEncoder())
    snapshot["feedback"][0]["source"] = "human"
    snapshot["filters"] = {"perspective": "as_lived", "cutoff": "2026-01-01T10:00:00Z"}
    snapshot["feedback"][0]["created_at"] = "2026-01-02T10:00:00Z"
    with pytest.raises(DiscoveryInvalid, match="cutoff"):
        score_batch(snapshot, encoder=FixtureEncoder())


def test_missing_native_locator_remains_explicit_without_fabricated_citation():
    snapshot = deepcopy(batch())
    snapshot["records"][0]["locators"] = []
    candidate = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert candidate["citations"] == []
    assert any("no native" in item for item in candidate["uncertainty"])


def test_neutral_neighbor_does_not_hide_threat_and_citations_follow_winning_window():
    snapshot = batch("neutral........")
    snapshot["records"].append(record("two", ordinal=1, text="reported threat"))
    snapshot["config"]["max_window_chars"] = 16
    encoder = FixtureEncoder({"neutral........\n": [0, 0, 1]})
    candidate = score_batch(snapshot, encoder=encoder)["candidates"][0]
    assert candidate["labels"] == ["threat"]
    assert candidate["citations"][0]["record_id"] == "two"
    assert candidate["citations"][0]["quote"] == "reported threat"
    assert [r["record_id"] for r in candidate["context_records"]] == ["one", "two"]


def test_cited_offsets_are_unicode_characters_and_include_later_retrieved_span():
    snapshot = batch("🧭" * 16 + "reported threat")
    snapshot["config"]["max_window_chars"] = 16
    encoder = FixtureEncoder({"🧭" * 16: [0, 0, 1]})
    citation = score_batch(snapshot, encoder=encoder)["candidates"][0]["citations"][0]
    assert citation["start_char"] == 16
    assert citation["end_char"] == 31
    assert citation["quote"] == "reported threat"


def test_supplied_episode_membership_is_authoritative_over_original_native_link():
    snapshot = batch()
    snapshot["records"][0]["explicit_episode_id"] = "original-link"
    snapshot["episodes"] = [{"episode_id": "episode:go-remapped", "record_ids": ["one"]}]
    candidate = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert candidate["episode_id"] == "episode:go-remapped"


def test_go_nullable_optional_arrays_and_initial_zero_feedback_revision():
    snapshot = batch()
    snapshot["feedback"] = None
    snapshot["feedback_revision"] = 0
    snapshot["episodes"] = None
    snapshot["records"][0]["participants"] = None
    snapshot["records"][0]["attributed_actor_ids"] = None
    candidate = score_batch(snapshot, encoder=FixtureEncoder())["candidates"][0]
    assert candidate["speaker_ids"] == candidate["attributed_actor_ids"] == []


def test_competing_negation_description_abstains_even_above_label_threshold():
    snapshot = batch("near denial")
    result = score_batch(snapshot, encoder=FixtureEncoder({"near denial": [0.7, 0, 0.75]}))
    assert result["candidates"] == []
    assert result["counts"]["abstained"] == 1
