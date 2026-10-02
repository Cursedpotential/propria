"""The Case Bible chunk publisher (modules/Consignatio/casebible/tools/comm_timeline_mvp/chunk_publish.py).

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

It imports Probata's server/context_chunks, so its tests live here. The chunker is a fake that cuts at known rows; the
real model and the live Weaviate are the deploy's proof.
"""

from __future__ import annotations

import sys
import uuid
from datetime import datetime, timedelta
from itertools import pairwise
from pathlib import Path

import pytest

TOOLS = Path(__file__).resolve().parents[3] / "Consignatio" / "casebible" / "tools" / "comm_timeline_mvp"
if not TOOLS.is_dir():  # a checkout without the Consignatio module
    pytest.skip("Consignatio module not present", allow_module_level=True)
sys.path.insert(0, str(TOOLS))
import chunk_publish as cp

from server.context_chunks import ids

T0 = datetime(2024, 11, 24, 14, 0)  # noqa: DTZ001  naive, as sort_ts_final is


class CutAtRows:
    """Starts a new chunk at the given row indexes of every conversation it is asked to cut."""

    def __init__(self, cuts):
        self.cuts = cuts

    def firsts(self, lines, beat=None):
        return sorted({0, *(c for c in self.cuts if c < len(lines))})


def make_rows(n, vault_key="b2://v/sms-1.xml", conv="conv-A", *, offset=0, minutes=None):
    return [
        {
            "dedup_key": f"dk-{vault_key[-9:]}-{conv}-{i + offset:03d}",
            "content_key": f"ck-{(i + offset) // 2:03d}",
            "record_index": i + offset,
            "event_ts_utc": None,
            "sort_ts_final": T0 + timedelta(minutes=(minutes or 1) * (i + offset)),
            "conversation_id": conv,
            "conversation_title": "Title",
            "participants": ["Katrina Kinzel", "Matt Salem"],
            "sender": "Katrina Kinzel" if i % 2 else "Matt Salem",
            "body": f"message {i + offset}",
            "attachments": None,
            "event_kind": "sms",
            "direction": "in",
            "vault_key": vault_key,
            "sha1": "sha1abc",
            "catalog_rel": "SourceCorpus/sms-1.xml",
            "source_format": "sms_backup_xml",
            "platform": "carrier_sms_mms",
            "custodian": "Matt",
            "source_device": "dev-1",
        }
        for i in range(n)
    ]


class FakeEmbedder:
    model = "nvidia/nemotron-3-embed-1b"

    def __init__(self):
        self.calls = 0

    def embed(self, texts):
        self.calls += 1
        return [[0.0] * 4 for _ in texts]


class FakeStore:
    def __init__(self):
        self.objects = {}

    def ensure_collection(self):
        return []

    def upsert(self, objects):
        for o in objects:
            self.objects[o["id"]] = o
        return len(objects)

    def _match(self, where, o):
        p = o["properties"]
        ok = []
        for op in where["operands"]:
            key, want, kind = op["path"][0], op["valueText"], op["operator"]
            ok.append((p.get(key) == want) if kind == "Equal" else (p.get(key) != want))
        return all(ok)

    def count(self, where):
        return sum(1 for o in self.objects.values() if self._match(where, o))

    def delete_matching(self, where, *, dry_run=False):
        hits = [k for k, o in self.objects.items() if self._match(where, o)]
        if not dry_run:
            for k in hits:
                del self.objects[k]
        return len(hits)

    def delete_other_generations(self, thread_id, digest):
        return self.delete_matching(
            {
                "operands": [
                    {"path": ["thread_id"], "operator": "Equal", "valueText": thread_id},
                    {"path": ["thread_digest"], "operator": "NotEqual", "valueText": digest},
                ]
            }
        )


def test_rows_group_by_vault_key_and_conversation_in_conversation_order():
    rows = make_rows(3, conv="B") + make_rows(3, conv="A") + make_rows(2, vault_key="b2://v/other.xml", conv="A")
    rows[0], rows[2] = rows[2], rows[0]  # out of order on purpose
    groups = cp.group_rows(rows)
    assert set(groups) == {("b2://v/sms-1.xml", "B"), ("b2://v/sms-1.xml", "A"), ("b2://v/other.xml", "A")}
    for g in groups.values():
        assert [r["record_index"] for r in g] == sorted(r["record_index"] for r in g)


def test_chunks_link_back_to_the_case_bible_records_and_overlap_by_two():
    rows = make_rows(12)
    digest, objects = cp.chunk_group(
        "b2://v/sms-1.xml", "conv-A", rows, run_id="RUN1", model="m", engine=CutAtRows([5, 9])
    )
    assert [(o["properties"]["chunk_index"], o["properties"]["message_count"]) for o in objects] == [
        (0, 7),
        (1, 6),
        (2, 3),
    ]
    first = objects[0]["properties"]
    assert first["vault_key"] == "b2://v/sms-1.xml" and first["catalog_path"] == "SourceCorpus/sms-1.xml"
    assert first["conversation_id"] == "conv-A" and first["thread_id"] == "b2://v/sms-1.xml#conv-A"
    assert first["dedup_keys"] == [r["dedup_key"] for r in rows[:7]]
    assert first["content_keys"] == ["ck-000", "ck-001", "ck-002", "ck-003"]  # distinct, in order
    assert first["first_dedup_key"] == rows[0]["dedup_key"] and first["last_dedup_key"] == rows[6]["dedup_key"]
    assert (
        first["ingest_run_id"] == "RUN1" and first["origin_system"] == "casebible" and first["thread_digest"] == digest
    )
    assert (
        first["text"].split("\n")[0] == "[2024-11-24 14:00] Matt Salem: message 0"
        and first["start_at"] == "2024-11-24T14:00:00Z"
    )
    for a, b in pairwise(objects):
        assert len(set(a["properties"]["dedup_keys"]) & set(b["properties"]["dedup_keys"])) >= 2
    version = objects[0]["properties"]["chunker_version"]
    key = f"casebible-chunk-v1|b2://v/sms-1.xml#conv-A|{rows[0]['dedup_key']}|{rows[6]['dedup_key']}|{version}"
    assert objects[0]["id"] == str(uuid.uuid5(ids.CHUNK_NAMESPACE, key))


def test_publish_file_is_idempotent_and_replaces_a_grown_conversation_and_retires_older_runs():
    store, emb = FakeStore(), FakeEmbedder()
    engine = CutAtRows([5, 9])
    out = cp.publish_file(store, emb, make_rows(12), run_id="RUN1", engine=engine, batch=2)
    assert out["conversations"] == 1 and out["chunks"] == 3 and len(store.objects) == 3 and emb.calls == 2
    again = cp.publish_file(store, emb, make_rows(12), run_id="RUN1", engine=engine, batch=2)  # a resumed run
    assert again["skipped_current"] == 1 and again["chunks"] == 0 and emb.calls == 2
    # a later run id, the conversation grew: whole conversation re-chunked, the old chunks of the file retired
    grown = cp.publish_file(store, emb, make_rows(16), run_id="RUN2", engine=CutAtRows([6, 11]), batch=4)
    assert grown["chunks"] == 3
    assert {o["properties"]["ingest_run_id"] for o in store.objects.values()} == {"RUN2"}
    assert {k for o in store.objects.values() for k in o["properties"]["dedup_keys"]} == {
        r["dedup_key"] for r in make_rows(16)
    }


def test_a_new_run_with_identical_data_keeps_its_chunks_and_never_retires_them():
    store, emb = FakeStore(), FakeEmbedder()
    engine = CutAtRows([5])
    cp.publish_file(store, emb, make_rows(10), run_id="RUN1", engine=engine)
    out = cp.publish_file(store, emb, make_rows(10), run_id="RUN2", engine=engine)
    assert out["skipped_current"] == 0 and out["chunks"] == 2 and out["retired"] == 0
    assert len(store.objects) == 2 and {o["properties"]["ingest_run_id"] for o in store.objects.values()} == {"RUN2"}


def test_other_files_and_other_conversations_are_untouched():
    store, emb = FakeStore(), FakeEmbedder()
    cp.publish_file(
        store, emb, make_rows(6, vault_key="b2://v/other.xml", conv="Z"), run_id="RUN0", engine=CutAtRows([3])
    )
    cp.publish_file(store, emb, make_rows(6) + make_rows(6, conv="conv-B"), run_id="RUN1", engine=CutAtRows([3]))
    assert {o["properties"]["vault_key"] for o in store.objects.values()} == {"b2://v/other.xml", "b2://v/sms-1.xml"}
    assert {
        o["properties"]["conversation_id"]
        for o in store.objects.values()
        if o["properties"]["vault_key"].endswith("sms-1.xml")
    } == {"conv-A", "conv-B"}
    assert any(o["properties"]["ingest_run_id"] == "RUN0" for o in store.objects.values())


def test_a_call_row_without_a_body_is_described_like_the_per_message_loader_did():
    row = make_rows(1)[0] | {"body": None, "event_kind": "call", "direction": "missed", "conversation_title": "Katrina"}
    _, objects = cp.chunk_group(row["vault_key"], "conv-A", [row], run_id="R", model="m", engine=CutAtRows([]))
    assert objects[0]["properties"]["text"].endswith("Matt Salem: Katrina call missed")


def test_estimate_counts_conversations_chunks_and_embed_calls():
    assert cp.estimate(10, 3200, batch=32) == {
        "conversations": 10,
        "messages": 3200,
        "estimated_chunks": 100,
        "estimated_embed_calls": 10,
    }
    assert cp.estimate(2, 5, batch=32)["estimated_chunks"] == 2
    assert cp.PROPERTIES and {"vault_key", "catalog_path", "conversation_id", "content_keys", "dedup_keys"} <= {
        p["name"] for p in cp.PROPERTIES
    }
