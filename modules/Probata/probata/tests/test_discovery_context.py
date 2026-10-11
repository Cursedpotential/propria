"""Verify discovery context keeps explicit links, clocks and occurrence identity."""

from copy import deepcopy

import pytest

from server.analysis.discovery.context import assemble_context, attribution
from server.analysis.discovery.storage import DiscoveryInvalid


def record(identity, *, source="source-a", conversation="conversation-a", ordinal=0, text="neutral"):
    """Construct a synthetic normalized occurrence with a native locator.

    Inputs: fixture identity/context. Outputs: dictionary. Effects: none; choose
    privacy-safe fixtures instead of live evidence.
    """
    return {
        "record_id": identity,
        "source_version_id": source,
        "generation_id": "generation-a",
        "conversation_id": conversation,
        "ordinal": ordinal,
        "text": text,
        "occurred_at": "2026-01-01T10:00:00Z",
        "source_available_from": "2026-01-01T10:00:00Z",
        "participants": [{"role": "speaker", "identifier": "speaker-a", "display_name": "Speaker A"}],
        "locators": [
            {
                "locator_id": "locator-" + identity,
                "source_version_id": source,
                "coordinate_system": "native-message",
                "range_start": ordinal,
                "range_end": ordinal + 1,
                "projection": {"message_id": identity},
            }
        ],
    }


def test_explicit_cross_platform_continuation_preserves_native_chronology():
    first = record("first", source="sms", text="Earlier statement")
    second = record("second", source="email", text="Continuation")
    second["occurred_at"] = "2026-01-01T11:00:00+01:00"
    first["ordinal"] = 1
    second["ordinal"] = 2
    first["explicit_episode_id"] = second["explicit_episode_id"] = "linked"
    result = assemble_context({"records": [second, first]})
    assert len(result) == 1
    assert [row["record_id"] for row in result[0]["records"]] == ["first", "second"]
    assert result[0]["records"][1]["locators"] == second["locators"]


def test_same_participant_unrelated_conversations_do_not_merge():
    first = record("first", conversation="one")
    second = record("second", conversation="two")
    assert len(assemble_context({"records": [first, second]})) == 2
    third = record("third", source="different-source", conversation="one")
    assert len(assemble_context({"records": [first, third]})) == 2


def test_duplicate_text_occurrences_remain_distinct_and_unknown_clock_is_retained():
    first = record("first", ordinal=1, text="Repeated sentence")
    second = record("second", ordinal=2, text="Repeated sentence")
    second["occurred_at"] = "around last spring"
    records = assemble_context({"records": [second, first]})[0]["records"]
    assert [r["record_id"] for r in records] == ["first", "second"]
    assert records[1]["occurred_at"] == "around last spring"
    assert any("uncertain" in item for item in attribution(records, [])["uncertainty"])
    duplicate = deepcopy(first)
    with pytest.raises(DiscoveryInvalid, match="duplicated"):
        assemble_context({"records": [first, duplicate]})


def test_reported_third_party_threat_keeps_speaker_separate_from_unknown_actor():
    reported = record("reported", text='He told me, "I will hurt you".')
    reported["speech_form"] = "reported"
    result = attribution([reported], [])
    assert result["speaker_ids"] == ["speaker-a"]
    assert result["attributed_actor_ids"] == []
    assert result["attribution_basis"] == "unknown"
    assert any("reported speech" in text for text in result["uncertainty"])


def test_cutoff_rejects_unknown_or_future_availability_before_scoring():
    batch = {"records": [record("record")], "filters": {"perspective": "as_lived", "cutoff": "2026-01-01T10:00:00Z"}}
    assert assemble_context(batch)
    batch["records"][0]["source_available_from"] = "2026-01-02T10:00:00Z"
    with pytest.raises(DiscoveryInvalid, match="cutoff"):
        assemble_context(batch)
    batch["records"][0]["source_available_from"] = None
    with pytest.raises(DiscoveryInvalid, match="cutoff"):
        assemble_context(batch)


def test_explicit_regrouping_cannot_reference_missing_or_conflicting_records():
    batch = {"records": [record("one")], "episodes": [{"episode_id": "episode", "record_ids": ["absent"]}]}
    with pytest.raises(DiscoveryInvalid, match="membership"):
        assemble_context(batch)


@pytest.mark.parametrize("occurred", [None, "uncertain-clock", "2026-01-02T10:00:00Z"])
def test_as_lived_rejects_future_or_unknown_occurrence_even_when_source_was_available(occurred):
    snapshot = {"records": [record("one")], "filters": {"perspective": "as_lived", "cutoff": "2026-01-01T10:00:00Z"}}
    snapshot["records"][0]["occurred_at"] = occurred
    with pytest.raises(DiscoveryInvalid, match="cutoff"):
        assemble_context(snapshot)
