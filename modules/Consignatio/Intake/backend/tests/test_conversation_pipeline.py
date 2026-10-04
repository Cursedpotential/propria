"""Routing, message parsing and conversation chunks. Byline: Claude Code · Sonnet 5.5 ·
2026-10-02"""

import io
from pathlib import Path

import pytest

from casebible_index import routing
from casebible_index.chunk_identity import chunk_object_id
from casebible_index.conversation_chunks import ChunkCoreMissing, chunk_spool, chunk_thread
from casebible_index.message_sources import (
    CALLS_THREAD,
    MessageSpool,
    SourceMessage,
    parse_message_xml,
)

SMS_XML = (
    b"""<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="6">
  <sms protocol="0" address="+18105550101" date="1700000300000" """
    b"""type="1" body="third, received" contact_name="Resolved Name" />
  <sms protocol="0" address="+18105550101" date="1700000100000" type="2" body="first, sent" />
  <sms protocol="0" address="+18105550101" date="1700000200000" """
    b"""type="1" body="second &amp; received" />
  <sms protocol="0" address="+13135550199" date="1700000150000" type="1" body="other thread" />
  <mms date="1700000400000" address="+18105550101" msg_box="1"><parts>"""
    b"""<part ct="image/jpeg" /><part ct="text/plain" text="mms text" /></parts></mms>
  <call number="+18105550101" duration="61" date="1700000500000" """
    b"""type="2" contact_name="Resolved Name" />
</smses>
"""
)


class EveryTwo:
    """A stand-in engine: a new chunk every two messages. The real Neural chunker is exercised in
    the live proof."""

    def firsts(self, lines, beat=None):
        return list(range(0, len(lines), 2))


def test_routing_by_key_and_by_head():
    assert routing.classify_key("a/b/note.TXT") == routing.KIND_DOCUMENT
    assert routing.classify_key("a/photo.jpg") == routing.KIND_MEDIA
    assert routing.classify_key("a/clip.mp4") == routing.KIND_MEDIA
    assert routing.classify_key("a/blob.bin") == routing.KIND_UNSUPPORTED
    assert routing.classify_key("a/takeout.zip") == routing.KIND_ARCHIVE
    assert routing.is_indexable_kind(routing.KIND_DOCUMENT) and not routing.is_indexable_kind(
        routing.KIND_MEDIA
    )
    assert (
        routing.is_excluded("x/.git/config")
        and routing.is_excluded("../x")
        and not routing.is_excluded("x/y.txt")
    )
    chatgpt = b'[{"title": "t", "mapping": {"a": {}}, "current_node": "a"}]'
    claude = b'[{"uuid": "u", "chat_messages": [{"sender": "human", "text": "hi"}]}]'
    assert routing.sniff_ai_chat(chatgpt) and routing.sniff_ai_chat(claude)
    assert not routing.sniff_ai_chat(b'{"mapping": 1}') and not routing.sniff_ai_chat(b"[1, 2, 3]")
    assert routing.maybe_ai_chat_name("moved/data-2025/conversations.json")
    assert routing.sniff_message_export(SMS_XML) and not routing.sniff_message_export(
        b"<html><body/></html>"
    )


def test_sms_xml_is_parsed_with_source_stated_senders_not_resolved_names():
    notes: list[str] = []
    records = list(parse_message_xml(io.BytesIO(SMS_XML), notes))
    assert notes == []
    by_body = {m.body: (thread, m) for thread, m in records}
    thread, received = by_body["third, received"]
    assert (
        thread == "+18105550101" and received.sender == "+18105550101"
    )  # contact_name is NOT used
    assert by_body["first, sent"][1].sender == "device -> +18105550101"
    assert by_body["mms text"][1].kind == "mms" and by_body["mms text"][1].sender == "+18105550101"
    call_thread, call = by_body["call outgoing 61s"]
    assert call_thread == CALLS_THREAD and call.sender == "+18105550101"
    assert not any("Resolved Name" in m.sender for _, m in records)


def test_truncated_xml_keeps_the_records_read_and_says_so():
    cut = SMS_XML[: SMS_XML.index(b"<mms")]
    notes: list[str] = []
    records = list(parse_message_xml(io.BytesIO(cut), notes))
    assert len(records) == 4 and notes and "ends early" in notes[0]


def test_spool_orders_each_thread_by_time_and_quarantines_itself(tmp_path: Path):
    spool = MessageSpool(tmp_path, "t")
    for thread, message in parse_message_xml(io.BytesIO(SMS_XML), []):
        spool.add(thread, message)
    spool.finish()
    assert spool.threads() == sorted({"+18105550101", "+13135550199", CALLS_THREAD})
    assert [m.body for m in spool.messages("+18105550101")] == [
        "first, sent",
        "second & received",
        "third, received",
        "mms text",
    ]
    path = spool.path
    spool.close()
    assert not path.exists()
    retained = list((tmp_path / "to_be_deleted" / "message-spool").glob("*/t.sqlite"))
    assert len(retained) == 1
    import sqlite3

    with sqlite3.connect(retained[0]) as connection:
        assert connection.execute("SELECT count(*) FROM m").fetchone()[0] == 6


def test_chunks_overlap_by_two_messages_and_are_content_addressed(tmp_path: Path):
    pytest.importorskip("server.context_chunks.chunker")
    from datetime import UTC, datetime

    messages = [
        SourceMessage(
            datetime(2026, 1, 1, 0, i, tzinfo=UTC), f"+1810555010{i % 2}", f"message {i}", "sms"
        )
        for i in range(9)
    ]
    chunks = chunk_thread(
        "file#thread", messages, chunker_name="neural_distilbert", overlap=2, engine=EveryTwo()
    )
    spans = [(c.first_index, c.last_index) for c in chunks]
    assert spans[0] == (0, 3) and spans[-1][1] == 8
    for a, b in zip(chunks, chunks[1:], strict=False):
        # at least two shared messages, or all of a short final chunk
        assert a.last_index - b.first_index + 1 >= min(2, b.last_index - b.first_index + 1)
    again = chunk_thread(
        "other-file#other", messages, chunker_name="neural_distilbert", overlap=2, engine=EveryTwo()
    )
    assert [c.content_hash for c in chunks] == [
        c.content_hash for c in again
    ]  # identity ignores where it came from
    assert chunk_object_id(chunks[0].chunker_version, chunks[0].content_hash) == chunk_object_id(
        again[0].chunker_version, again[0].content_hash
    )
    assert chunks[0].text.splitlines()[0] == "[2026-01-01 00:00] +18105550100: message 0"
    assert all(len(set(c.text.splitlines())) == len(c.text.splitlines()) for c in chunks)


def test_chunk_spool_runs_every_thread_and_calls_use_the_call_chunker(tmp_path: Path):
    pytest.importorskip("server.context_chunks.chunker")
    spool = MessageSpool(tmp_path, "t2")
    for thread, message in parse_message_xml(io.BytesIO(SMS_XML), []):
        spool.add(thread, message)
    spool.finish()
    chunks = list(
        chunk_spool(
            spool,
            "sha1:abc",
            chunker_name="neural_distilbert",
            overlap=2,
            engine=EveryTwo(),
            calls_engine=EveryTwo(),
        )
    )
    threads = {c.thread_id for c in chunks}
    assert threads == {"sha1:abc#+18105550101", "sha1:abc#+13135550199", f"sha1:abc#{CALLS_THREAD}"}
    assert [c.ordinal for c in chunks] == list(range(len(chunks)))
    assert next(c for c in chunks if c.thread_id.endswith(CALLS_THREAD)).chunker == "fast_1000"
    spool.close()


def test_missing_core_is_a_named_error(monkeypatch):
    import sys

    monkeypatch.setitem(sys.modules, "server", None)
    monkeypatch.setitem(sys.modules, "server.context_chunks", None)
    with pytest.raises(ChunkCoreMissing):
        chunk_thread(
            "t", [SourceMessage(None, "a", "b", "sms")], chunker_name="neural_distilbert", overlap=2
        )


def test_a_long_thread_is_chunked_in_segments_without_losing_overlap_or_coverage(tmp_path: Path):
    pytest.importorskip("server.context_chunks.chunker")
    from datetime import UTC, datetime, timedelta

    spool = MessageSpool(tmp_path, "seg")
    base = datetime(2026, 1, 1, tzinfo=UTC)
    for i in range(53):
        spool.add("t", SourceMessage(base + timedelta(minutes=i), "+1", f"m{i}", "sms"))
    spool.finish()
    chunks = list(
        chunk_spool(
            spool,
            "f",
            chunker_name="neural_distilbert",
            overlap=2,
            engine=EveryTwo(),
            segment_messages=10,
        )
    )
    covered = set()
    for chunk in chunks:
        covered.update(range(chunk.first_index, chunk.last_index + 1))
    assert covered == set(range(53))
    firsts = [c.first_index for c in chunks]
    assert firsts == sorted(set(firsts)) and firsts[0] == 0
    for a, b in zip(chunks, chunks[1:], strict=False):
        assert a.last_index - b.first_index + 1 >= min(2, b.last_index - b.first_index + 1)
    assert (
        chunks[3].text.splitlines()[0].endswith(f"m{chunks[3].first_index}")
    )  # global indexes match the text
    assert [c.ordinal for c in chunks] == list(range(len(chunks)))
    whole = list(
        chunk_spool(
            spool,
            "f",
            chunker_name="neural_distilbert",
            overlap=2,
            engine=EveryTwo(),
            segment_messages=1000,
        )
    )
    assert {c.content_hash for c in whole} != set() and covered == {
        i for c in whole for i in range(c.first_index, c.last_index + 1)
    }
    spool.close()
