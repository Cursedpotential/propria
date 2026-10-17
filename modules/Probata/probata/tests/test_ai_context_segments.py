"""Verify topic/domain segmentation and sentence chunking of AI conversations.

Byline: Claude Code · Opus 5.5 · 2026-10-16.

Inputs: fictional conversations and deterministic segmenter doubles. Outputs:
assertions on reply validation, exchange grouping, exact source slices, overlap
contexts and the untagged fallback. Effects: none beyond memory; no remote model
or case data. Real Chonkie SentenceChunker and OverlapRefinery are exercised.
"""
from __future__ import annotations

import json

import pytest

from server.analysis import ai_context_segments as seg

SOURCE = {"source_ref": "file:///fictional/original.json", "provider_version_id": None, "package_ref": None,
          "source_format": "chatgpt"}


def _records(turns: list[tuple[int, str, str]]) -> list[dict]:
    """Build decoded-record shapes for fictional turns.

    Inputs: (conversation, role, body) tuples. Outputs: records like decode_records.
    Effects: none.
    """
    out = []
    for i, (ci, role, body) in enumerate(turns):
        out.append({"conversation_index": ci, "conversation_id": f"conv-{ci}", "conversation_title": "Fictional",
                    "record_id": f"/{ci}/mapping/n{i}/message/content/parts/0", "native_message_index": i,
                    "role": role, "body": body, "source_span": {"start": 0, "end": len(body), "unit": "unicode_codepoint"}})
    return out


def _segmenter(reply: dict | str):
    """Return a fixed-reply segmenter double that counts its calls."""
    calls = []

    def call(prompt: str) -> str:
        calls.append(prompt)
        return reply if isinstance(reply, str) else json.dumps(reply)

    call.model_id = "fictional-segmenter"
    call.calls = calls
    return call


def test_parse_reply_derives_ends_from_ordered_starts():
    ids = ["E0", "E1", "E2", "E3", "E4"]
    reply = {"segments": [{"start_id": "E0", "domain": "other", "secondary_domain": None, "topic": "plants"},
                          {"start_id": "E2", "domain": "development", "secondary_domain": "other", "topic": "script"},
                          {"start_id": "E4", "domain": "unrelated", "secondary_domain": None, "topic": "weather"}]}
    parsed = seg.parse_reply(json.dumps(reply), ids)
    assert [(s["start"], s["end"]) for s in parsed] == [(0, 1), (2, 3), (4, 4)]
    assert parsed[1]["secondary_domain"] == "other"


@pytest.mark.parametrize("reply", [
    "not json",
    {"segments": []},
    {"segments": [{"start_id": "E1", "domain": "other", "secondary_domain": None, "topic": "x"}]},  # must start at E0
    {"segments": [{"start_id": "E0", "domain": "cooking", "secondary_domain": None, "topic": "x"}]},  # off taxonomy
    {"segments": [{"start_id": "E0", "domain": "other", "secondary_domain": "other", "topic": "x"}]},  # secondary == primary
    {"segments": [{"start_id": "E0", "domain": "other", "secondary_domain": None, "topic": "x"},
                  {"start_id": "E0", "domain": "development", "secondary_domain": None, "topic": "y"}]},  # not increasing
    {"segments": [{"start_id": "E0", "domain": "other", "secondary_domain": None, "topic": "x"},
                  {"start_id": "E5", "domain": "development", "secondary_domain": None, "topic": "y"}]},  # unknown id
    {"segments": [{"start_id": "E0", "domain": "other", "secondary_domain": None, "topic": ""}]},  # no topic
])
def test_parse_reply_rejects_off_contract_replies(reply):
    with pytest.raises(seg.SegmentReplyInvalid):
        seg.parse_reply(reply if isinstance(reply, str) else json.dumps(reply), ["E0", "E1", "E2"])


def test_exchanges_keep_prompt_with_all_replies_and_preamble():
    members = _records([(0, "system", "s"), (0, "user", "u1"), (0, "assistant", "a1"), (0, "tool", "t1"),
                        (0, "user", "u2"), (0, "assistant", "a2")])
    # A system preamble before the first user prompt joins the first exchange.
    assert seg.exchanges(members) == [[0, 1, 2, 3], [4, 5]]


def _conversation():
    long_answer = " ".join(f"Sentence number {i} explains the rename script in detail." for i in range(400))
    return _records([(0, "user", "How do I repot a snake plant?"), (0, "assistant", "Use cactus mix. Water after a week."),
                     (0, "user", "Write me a python script to rename files by date."), (0, "assistant", long_answer),
                     (0, "user", "Add a dry run flag."), (0, "assistant", "Added --dry-run. It prints planned renames."),
                     (0, "user", "What is the weather tomorrow?"), (0, "assistant", "I cannot check live weather.")])


def test_segment_chunks_are_exact_slices_with_segment_identity():
    records = _conversation()
    reply = {"segments": [{"start_id": "E0", "domain": "other", "secondary_domain": None, "topic": "plant care"},
                          {"start_id": "E1", "domain": "development", "secondary_domain": None, "topic": "rename script"},
                          {"start_id": "E3", "domain": "unrelated", "secondary_domain": None, "topic": "weather"}]}
    segmenter = _segmenter(reply)
    chunks, segments, stats = seg.segment_chunks(records, SOURCE, segmenter=segmenter)
    assert stats["segment_model_calls"] == 1 and stats["untagged_segments"] == 0
    assert [(s["first_exchange"], s["last_exchange"], s["domain"]) for s in segments] == [
        (0, 0, "other"), (1, 2, "development"), (3, 3, "unrelated")]
    full = "\n\n".join(r["body"] for r in records)
    for chunk in chunks:
        assert chunk["text"] == full[chunk["text_start"]:chunk["text_end"]]
        assert chunk["context"] == full[chunk["context_start"]:chunk["text_start"]]
        assert len(chunk["text"]) <= seg.SENTENCE_CHUNK_CHARS
        assert len(chunk["context"]) + len(chunk["text"]) <= seg.MAX_EMBED_CHARS
        for piece in chunk["segments"]:
            body = next(r["body"] for r in records if r["record_id"] == piece["record_id"])
            assert piece["text"] == body[piece["body_start"]:piece["body_end"]]
    # The long development answer splits into several chunks that share one segment_id.
    dev = [c for c in chunks if c["domain"] == "development"]
    assert len(dev) > 1 and len({c["segment_id"] for c in dev}) == 1
    assert [c["segment_chunk_index"] for c in dev] == list(range(len(dev)))
    assert all(c["segment_chunk_count"] == len(dev) for c in dev)
    # Overlap never crosses a segment: first chunk of each segment has no context.
    firsts = [c for c in chunks if c["segment_chunk_index"] == 0]
    assert all(c["context"] == "" for c in firsts)
    assert any(c["context"] for c in dev[1:])
    # Chunks tile each segment exactly, without gaps or overlap.
    for s in segments:
        own = [c for c in chunks if c["segment_id"] == s["segment_id"]]
        assert own[0]["text_start"] == s["text_start"] and own[-1]["text_end"] == s["text_end"]
        assert all(a["text_end"] == b["text_start"] for a, b in zip(own, own[1:]))
    # Same input and recipe give the same segment IDs.
    again, _, _ = seg.segment_chunks(records, SOURCE, segmenter=_segmenter(reply))
    assert [c["segment_id"] for c in again] == [c["segment_id"] for c in chunks]


def test_invalid_replies_are_retried_then_marked_untagged_never_guessed():
    records = _conversation()
    segmenter = _segmenter({"segments": [{"start_id": "E9", "domain": "other", "secondary_domain": None, "topic": "x"}]})
    chunks, segments, stats = seg.segment_chunks(records, SOURCE, segmenter=segmenter)
    assert len(segmenter.calls) == seg.SEGMENT_ATTEMPTS
    assert stats["untagged_windows"] == 1 and segments[0]["status"] == "untagged"
    assert segments[0]["domain"] is None and segments[0]["reason"].startswith("invalid_reply")
    assert all(c["segment_status"] == "untagged" for c in chunks) and chunks


def test_unconfigured_segmenter_is_explicit():
    chunks, segments, stats = seg.segment_chunks(_conversation(), SOURCE, segmenter=False)
    assert stats["segment_model_calls"] == 0
    assert stats["untagged_reasons"] == {"segmenter_unconfigured": 1}
    assert len(segments) == 1 and chunks


def test_long_conversations_are_windowed_with_marked_boundaries(monkeypatch):
    monkeypatch.setattr(seg, "WINDOW_EXCHANGES", 2)
    records = _conversation()  # four exchanges -> two windows
    seen = []

    def call(prompt: str) -> str:
        first = prompt.split("must start at ", 1)[1].split(";", 1)[0]
        seen.append(first)
        return json.dumps({"segments": [{"start_id": first, "domain": "other", "secondary_domain": None, "topic": "t"}]})

    call.model_id = "fictional"
    _, segments, stats = seg.segment_chunks(records, SOURCE, segmenter=call)
    assert sorted(seen) == ["E0", "E2"] and stats["segment_windows"] == 2
    assert [s["window_start"] for s in segments] == [False, True]
