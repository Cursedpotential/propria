"""Verify publish-stage DuckDB spill placement in a non-writable worker cwd."""

import json
import os
import uuid
from datetime import UTC, datetime, timedelta
from pathlib import Path

import pyarrow as pa
import pyarrow.parquet as pq
import pytest

from casebible_index.embed_stage import vector_schema, vectors_dir
from casebible_index.embedders import VectorSlot
from casebible_index.publish_stage import (
    LEDGER_SCHEMA,
    _publish_connection,
    pending_batch_sql,
    publish_pending,
    relocate_moved,
)


class RecordingStore:
    """Capture real publish-stage writes without a Weaviate network request."""

    def __init__(self):
        self.objects = []
        self.patches = []

    async def ensure_collection(self, vectors):
        """Accept named vector configuration; effects are limited to this test instance."""
        assert vectors == {"text_nim": 3}

    async def upsert(self, objects):
        """Record published objects and return the accepted count."""
        self.objects.extend(objects)
        return len(objects)

    async def patch(self, object_id, properties):
        """Record locator changes and report the object as present."""
        self.patches.append((object_id, properties))
        return True


def test_pending_batch_selects_only_unpublished_ready_identities_and_newest_good_vectors(
    tmp_path: Path,
):
    """Keep dedup, ledger exclusion and per-slot newest-good behavior after payload filtering."""
    out = tmp_path / "lake"
    chunks = out / "datasets" / "chunks"
    chunks.mkdir(parents=True)
    chunk_rows = []
    for content_hash, documents in {
        "a": ["doc-a", "doc-a-copy"],
        "b": ["doc-b"],
        "c": ["doc-c"],
        "d": ["doc-d"],
    }.items():
        for document_id in documents:
            chunk_rows.append(
                {
                    "chunker_version": "v1",
                    "content_hash": content_hash,
                    "document_id": document_id,
                    "chunk_ordinal": 0,
                    "text": content_hash,
                    "embedding": None,
                }
            )
    pq.write_table(pa.Table.from_pylist(chunk_rows), chunks / "chunks.parquet")

    now = datetime.now(UTC)
    slots = [
        VectorSlot("text_nim", "model-a", 3, "https://unused.test", "unused"),
        VectorSlot("legal", "model-b", 3, "https://unused.test", "unused"),
    ]
    for slot in slots:
        folder = vectors_dir(out, slot.name)
        folder.mkdir(parents=True)
        rows = []
        for content_hash in (["a", "b", "c", "d"] if slot == slots[0] else ["a", "b", "d"]):
            rows.append(
                {
                    "content_hash": content_hash,
                    "slot": slot.name,
                    "model": "old-good",
                    "dimensions": 3,
                    "status": "ok",
                    "embedding": [1.0, 1.0, 1.0],
                    "embedded_at": now - timedelta(days=1),
                }
            )
            if content_hash == "a":
                rows.extend(
                    [
                        {
                            "content_hash": content_hash,
                            "slot": slot.name,
                            "model": "new-good",
                            "dimensions": 3,
                            "status": "truncated",
                            "embedding": [2.0, 2.0, 2.0],
                            "embedded_at": now,
                        },
                        {
                            "content_hash": content_hash,
                            "slot": slot.name,
                            "model": "newer-failed",
                            "dimensions": 3,
                            "status": "failed",
                            "embedding": None,
                            "embedded_at": now + timedelta(days=1),
                        },
                    ]
                )
        pq.write_table(
            pa.Table.from_pylist(rows, schema=vector_schema()), folder / "vectors.parquet"
        )

    published = out / "datasets" / "published"
    published.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [{"chunker_version": "v1", "content_hash": "b"}], schema=LEDGER_SCHEMA
        ),
        published / "ledger.parquet",
    )
    sql = pending_batch_sql(out, slots, limit=20_001)
    assert sql is not None
    con = _publish_connection(out, "semantic-test")
    try:
        rows = con.execute(sql).fetchall()
        columns = [column[0] for column in con.description]
    finally:
        con.close()
    result = [dict(zip(columns, row, strict=True)) for row in rows]
    assert {row["content_hash"] for row in result} == {"a", "d"}
    assert len(result) == 2
    a = next(row for row in result if row["content_hash"] == "a")
    assert a["document_id"] == "doc-a"
    assert a["vec_0"] == a["vec_1"] == [2.0, 2.0, 2.0]
    assert a["model_0"] == a["model_1"] == "new-good"


@pytest.mark.asyncio
async def test_publish_spills_inside_output_bind_and_writes_ledger():
    """Force disk spill, then publish and relocate one synthetic chunk from the same lake."""
    workspace = Path(os.environ["PUBLISH_SPILL_WORKSPACE"])
    out = workspace / f"lake-{uuid.uuid4().hex}"
    out.mkdir(parents=True)
    slot = VectorSlot("text_nim", "fake-model", 3, "https://unused.test", "unused")

    con = _publish_connection(out, "forced-spill")
    try:
        spill_dir = Path(con.execute("SELECT current_setting('temp_directory')").fetchone()[0])
        assert spill_dir.is_absolute() and out in spill_dir.parents
        con.execute("SET threads=1")
        con.execute("SET memory_limit='32MB'")
        profile = out / "spill-profile.json"
        con.execute("SET enable_profiling='json'")
        con.execute(f"SET profiling_output='{profile.as_posix()}'")
        con.execute(
            "SELECT i, md5(CAST(i AS VARCHAR)) AS h "
            "FROM range(1000000) t(i) ORDER BY h"
        ).fetchall()
        assert json.loads(profile.read_text())["system_peak_temp_dir_size"] > 0
    finally:
        con.close()

    chunks = out / "datasets" / "chunks"
    chunks.mkdir(parents=True)
    content_hash = "a" * 64
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "chunker_version": "fixture-v1",
                    "content_hash": content_hash,
                    "document_id": "synthetic-doc",
                    "version_id": "synthetic-version",
                    "artifact_id": "synthetic-artifact",
                    "chunk_ordinal": 0,
                    "text": "synthetic fixture text",
                    "vault_key": "fixture/original.txt",
                    "relative_path": "fixture/original.txt",
                    "embedding": [0.1, 0.2, 0.3],
                }
            ]
        ),
        chunks / "fixture.parquet",
    )
    vectors = vectors_dir(out, slot.name)
    vectors.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "content_hash": content_hash,
                    "slot": slot.name,
                    "model": slot.model,
                    "dimensions": 3,
                    "status": "ok",
                    "embedding": [0.1, 0.2, 0.3],
                    "embedded_at": datetime.now(UTC),
                }
            ],
            schema=vector_schema(),
        ),
        vectors / "fixture.parquet",
    )

    store = RecordingStore()
    result = await publish_pending(out, store, [slot], max_objects=1, run_id="spill-test")
    assert result == {"published": 1, "more": False}
    assert len(store.objects) == 1
    ledger_files = list((out / "datasets" / "published").glob("*.parquet"))
    assert len(ledger_files) == 1
    assert pq.read_table(ledger_files[0]).column("content_hash").to_pylist() == [content_hash]

    locators = out / "datasets" / "locators"
    locators.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "document_id": "synthetic-doc",
                    "vault_key": "fixture/moved.txt",
                    "relative_path": "fixture/moved.txt",
                    "observed_at": datetime.now(UTC),
                }
            ]
        ),
        locators / "fixture.parquet",
    )
    relocated = await relocate_moved(out, store, max_objects=1, run_id="spill-relocate")
    assert relocated == {"patched": 1, "missing": 0}
    assert store.patches[0][1]["vault_key"] == "fixture/moved.txt"
    assert any((out / "scratch" / "duckdb").glob("relocate-*"))
    assert not Path("/app/.tmp").exists()
