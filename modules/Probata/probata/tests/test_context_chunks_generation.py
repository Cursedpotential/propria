"""Chunks cut from a run's normalized generation, before the commit (server/context_chunks/generation.py).

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta

import pytest

from server.context_chunks.generation import (
    GenerationSource,
    ResolutionError,
    conversation_key,
    normalize_party,
    owner_took_part,
    parse_parties,
    _AI_CHAT_FORMATS,
)
from server.context_chunks.model import CORPUS_FIRST_PARTY, CORPUS_THIRD_PARTY, ThreadRef
from server.context_chunks.service import build_chunk_objects, plan_thread

T0 = datetime(2026, 1, 5, 14, 30, tzinfo=UTC)
GEN = "01a0fd18-0000-7000-8000-000000000001"


def test_conversation_key_matches_the_go_vectors():
    """The same vectors as derive/smsthreads TestConversationKeyVectorsSharedWithThePythonPort."""
    long_in = [f"+1810555{i:04d}" for i in range(9)]
    cases = [
        (["+18102959303", "self"], "8102959303", ["8102959303"]),
        (["(810) 295-9303", "+18102959302", "SELF"], "8102959302_8102959303", ["8102959302", "8102959303"]),
        (["Katrina.Kinzel@Example.com", "self"], "katrina.kinzel@example.com", ["katrina.kinzel@example.com"]),
        (["Katrina Kinzel", "self"], "katrinakinzel", ["katrinakinzel"]),
        (["self", "null", ""], "unknown", []),
        (["12345", "insert-address-token"], "12345", ["12345"]),
        (long_in, "group-9-785bb6e5b107", [f"810555{i:04d}" for i in range(9)]),
    ]
    for given, key, participants in cases:
        assert conversation_key(given) == (key, participants), given
    assert normalize_party("  +1 (810) 295-9303 ") == "8102959303" and normalize_party("self") == ""


RESOLUTION = {
    "basis": "owner_participant:v1",
    "owner_person_id": "owner",
    "perspective_person_id": "owner",
    "identifiers": [
        {"raw": "+18105550100", "normalized": "8105550100", "entity_id": "owner", "is_owner": True},
        {"raw": "+18105550199", "normalized": "8105550199", "entity_id": "partner", "is_owner": False},
        {"raw": "+18105550123", "normalized": "8105550123", "is_owner": False},
    ],
}


def test_owner_took_part_is_the_go_disclosure_rule():
    assert owner_took_part(RESOLUTION, "+18105550199", ["+18105550100"]) is True
    assert owner_took_part(RESOLUTION, "+18105550199", ["+18105550123"]) is False
    assert owner_took_part(RESOLUTION, "self", ["+18105550199"]) is True  # the perspective person is the owner
    elsewhere = {**RESOLUTION, "perspective_person_id": "partner"}
    assert owner_took_part(elsewhere, "self", ["+18105550123"]) is False
    with pytest.raises(ResolutionError):
        owner_took_part({**RESOLUTION, "perspective_person_id": ""}, "self", [])
    with pytest.raises(ResolutionError):
        owner_took_part(RESOLUTION, "+18105559999", [])
    with pytest.raises(ResolutionError):
        owner_took_part({**RESOLUTION, "basis": "other"}, "", [])


def test_parse_parties_counts_unknown_roles_as_recipients_unless_they_repeat_the_sender():
    payload = {
        "participants": [
            {"role": "sender", "identifier": "+18102959303"},
            {"role": "recipient", "identifier": "self"},
            {"role": "unknown", "identifier": "+18102959302"},
            {"role": "unknown", "identifier": "+18102959303"},
        ]
    }
    parties, sender, recipients = parse_parties(payload)
    assert sender == "+18102959303" and recipients == ["self", "+18102959302"] and len(parties) == 4


class _Result:
    def __init__(self, rows):
        self.rows = rows

    def mappings(self):
        return self

    def all(self):
        return list(self.rows)

    def one_or_none(self):
        return self.rows[0] if self.rows else None


class _Conn:
    def __init__(self, answers):
        self.answers = answers

    def execute(self, statement, params=None):
        sql = str(statement)
        for marker, rows in self.answers.items():
            if marker in sql:
                return _Result(rows)
        raise AssertionError(f"unexpected SQL: {sql[:90]}")


def _msg(i, sender, recipients, body, *, minutes=0, rtype="message", svid="sv-1", extra=None):
    participants = [{"role": "sender", "identifier": sender}] + [
        {"role": "recipient", "identifier": r} for r in recipients
    ]
    content = {"body": body, **(extra or {})}
    return {
        "id": f"00000000-0000-4000-8000-{i:012d}",
        "record_type": rtype,
        "record_ordinal": i,
        "occurred_at": T0 + timedelta(minutes=minutes),
        "svid": svid,
        "matter_id": "matter-1",
        "payload": {"content": content, "participants": participants},
    }


def _source(records, *, matches=None, resolution=RESOLUTION):
    answers = {
        "FROM context.normalized_generation generation": [("smsbackuprestore_xml", "smsbackuprestore_xml")],
        "FROM context.normalized_record_identity": records,
        "FROM registry.vw_case_identifier": [
            {"entity_id": "partner", "display_name": "Katrina Kinzel", "identifier": "8105550199", "kind": "phone"},
            {"entity_id": "owner", "display_name": "Matthew S. Salem", "identifier": "8105550100", "kind": "phone"},
        ],
        "FROM registry.entity WHERE id": [("owner", "Matthew S. Salem")],
        "FROM context.activity_receipt receipt": [],
    }
    conn = _Conn(answers)
    source = GenerationSource(conn, GEN, "res-1", "match-1" if matches is not None else "")  # type: ignore[arg-type]
    source.resolution = lambda: resolution  # type: ignore[method-assign]
    source.matched = lambda: set(matches or [])  # type: ignore[method-assign]
    return source


def test_a_generation_is_cut_into_conversations_by_corpus_and_key_with_record_ids_as_message_ids():
    records = [
        _msg(1, "+18105550199", ["self"], "are you picking him up", minutes=0),
        _msg(2, "self", ["+18105550199"], "yes at six", minutes=2),
        _msg(3, "+18105550123", ["+18105550199"], "unrelated third party chatter", minutes=3),  # owner absent
        _msg(4, "+18105550199", ["self"], "thanks", minutes=5),
        _msg(
            5,
            "+18105550199",
            ["self"],
            "call attached",
            minutes=6,
            rtype="call",
            extra={"missed": True, "direction": "incoming", "duration_seconds": 0},
        ),
    ]
    source = _source(records)
    refs = source.all_threads()
    assert refs == [
        ThreadRef(CORPUS_THIRD_PARTY, f"gen-{GEN}/acquired_third_party/8105550123_8105550199"),
        ThreadRef(CORPUS_FIRST_PARTY, f"gen-{GEN}/first_party/8105550199"),
    ] or set(refs) == {
        ThreadRef(CORPUS_THIRD_PARTY, f"gen-{GEN}/acquired_third_party/8105550123_8105550199"),
        ThreadRef(CORPUS_FIRST_PARTY, f"gen-{GEN}/first_party/8105550199"),
    }
    first = source.load_thread(ThreadRef(CORPUS_FIRST_PARTY, f"gen-{GEN}/first_party/8105550199"))
    assert [m.id[-2:] for m in first.messages] == ["01", "02", "04"] and first.matter_id == "matter-1"
    assert [m.sender for m in first.messages] == ["+18105550199", "self", "+18105550199"]  # as the source stated them
    assert [m.sender_name for m in first.messages] == ["Katrina Kinzel", "Matthew S. Salem", "Katrina Kinzel"]
    plan = plan_thread(
        first, "token_1000", 2, engine=type("E", (), {"firsts": lambda self, lines, beat=None: [0, 2]})()
    )
    objects = build_chunk_objects(first, plan, embed_model="m", generation_id=GEN)
    props = objects[0]["properties"]
    assert (
        props["normalized_record_ids"] == props["message_ids"] == [m.id for m in first.messages[:3]]
    )  # ids copied, 1:1
    assert props["normalized_generation_id"] == GEN and props["thread_id"].startswith(f"gen-{GEN}/first_party/")
    calls = source.load_call_file("sv-1")
    assert calls.call_ids == [records[4]["id"]] and calls.lines[0].endswith("missed call from Katrina Kinzel, 0s")
    assert source.call_source_versions() == ["sv-1"]


def test_messages_already_held_by_an_earlier_source_are_left_out_and_a_calls_only_generation_needs_no_resolution():
    records = [_msg(1, "+18105550199", ["self"], "a"), _msg(2, "+18105550199", ["self"], "b", minutes=1)]
    source = _source(records, matches=[records[0]["id"]])
    (ref,) = source.all_threads()
    assert [m.id for m in source.load_thread(ref).messages] == [records[1]["id"]]
    only_calls = _source(
        [_msg(1, "+18105550199", ["self"], "", rtype="call", extra={"direction": "incoming", "duration_seconds": 7})],
        resolution=None,
    )  # type: ignore[arg-type]
    assert only_calls.all_threads() == [] and only_calls.call_source_versions() == ["sv-1"]
    with pytest.raises(LookupError):
        _source(records, resolution=None).all_threads()  # type: ignore[arg-type]


@pytest.mark.parametrize("format_id", sorted(_AI_CHAT_FORMATS))
@pytest.mark.parametrize("raw_only", [False, True])
def test_ai_generation_is_refused_before_human_resolution_or_publisher_resume(format_id, raw_only):
    """Reject declared/raw AI formats at source binding, before any records, registry or publisher are accessed.

    Inputs: synthetic persisted format rows; outputs: assertions. Effects: in-memory SQL fixture only.
    Choose to cover direct Activity/publisher construction independent of Go workflow dispatch.
    """
    formats = ("json", format_id) if raw_only else (format_id, "generic_message")
    conn = _Conn({"FROM context.normalized_generation generation": [formats]})
    with pytest.raises(ValueError, match="AI chat generations"):
        GenerationSource(conn, GEN, "stale-human-resolution", "stale-human-match")  # type: ignore[arg-type]


def test_human_generation_requires_real_provenance_and_keeps_missing_resolution_gate():
    """Preserve human source lookup and resolution requirements when no AI classification is verified.

    Inputs: absent provenance or normal SMS records; outputs: assertions. Effects: in-memory fixtures only.
    Choose as the strict-human regression counterpart of the AI direct-publisher boundary.
    """
    with pytest.raises(LookupError, match="no retained source/raw provenance"):
        GenerationSource(_Conn({"FROM context.normalized_generation generation": []}), GEN)  # type: ignore[arg-type]
    with pytest.raises(LookupError, match="no participant resolution"):
        _source([_msg(1, "user", ["assistant"], "human record labels do not classify sources")], resolution=None).all_threads()
