"""The publish stage: write embedded chunks to Weaviate, and follow moved files. One unit, one job:
publish.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Reads Parquet, never holds more than one batch. A chunk is published once it has a good vector for
EVERY enabled slot
and is not yet in the publish ledger (``datasets/published``), keyed by (chunker_version,
content_hash), the same pair
that makes its Weaviate id (chunk_identity.py). Identical text in two files is one object. Writes
are idempotent
upserts, so a crash between the Weaviate write and the ledger write only repeats a write.

Two passes in one unit because both are "project the lake into the collection":

* ``publish_pending``  new chunks -> objects (+ vectors per slot);
* ``relocate_moved``   a file that moved has a newer locator (``datasets/locators``) than the
ledger's ``vault_key``
  for its chunks -> PATCH ``vault_key``/``catalog_path`` on those objects (no re-extract, no
  re-embed; owner
  2026-09-22 10:48).

Retirement (an object that left the catalog) is REPORTED in the cycle receipt, not performed: a
content-addressed
object can be shared by another live file, and the owner has not asked for derived objects to be
deleted.
"""

from __future__ import annotations

import uuid
from collections.abc import Callable
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import duckdb
import pyarrow as pa
import pyarrow.parquet as pq

from .cas_store import ChunkCollection, chunk_properties
from .chunk_identity import chunk_object_id
from .embed_stage import chunk_source_sql, vectors_dir
from .embedders import VectorSlot

DEFAULT_MAX_OBJECTS = 20_000
FETCH_ROWS = 256

LEDGER_SCHEMA = pa.schema(
    [
        ("chunker_version", pa.string()),
        ("content_hash", pa.string()),
        ("object_id", pa.string()),
        ("document_id", pa.string()),
        ("vault_key", pa.string()),
        ("relative_path", pa.string()),
        ("run_id", pa.string()),
        ("published_at", pa.timestamp("us", tz="UTC")),
    ]
)


def _posix(path: Path) -> str:
    return path.resolve().as_posix().replace("'", "''")


def ledger_dir(output_dir: Path) -> Path:
    return output_dir / "datasets" / "published"


def _ledger_relation(output_dir: Path) -> str | None:
    folder = ledger_dir(output_dir)
    if folder.is_dir() and any(folder.glob("*.parquet")):
        return f"read_parquet('{_posix(folder / '*.parquet')}', union_by_name = true)"
    return None


def _ready_vectors(output_dir: Path, slots: list[VectorSlot]) -> list[str] | None:
    """One relation per slot of good vectors, or None when any slot has none yet."""
    relations = []
    for slot in slots:
        folder = vectors_dir(output_dir, slot.name)
        if not (folder.is_dir() and any(folder.glob("*.parquet"))):
            return None
        relations.append(
            f"(SELECT content_hash, embedding, model FROM "
            f"read_parquet('{_posix(folder / '*.parquet')}', "
            f"union_by_name = true) WHERE status IN ('ok', 'truncated') "
            f"QUALIFY row_number() OVER (PARTITION BY content_hash ORDER BY embedded_at DESC) = 1)"
        )
    return relations


def pending_batch_sql(output_dir: Path, slots: list[VectorSlot], limit: int) -> str | None:
    """SQL returning up to ``limit`` chunk rows, one per (chunker_version, content_hash), with one
    vector per slot."""
    vectors = _ready_vectors(output_dir, slots)
    if vectors is None or not any((output_dir / "datasets" / "chunks").glob("*.parquet")):
        return None
    ledger = _ledger_relation(output_dir)
    chunks = chunk_source_sql(output_dir)
    ready_join = " ".join(f"SEMI JOIN {v} v{i} USING (content_hash)" for i, v in enumerate(vectors))
    not_published = (
        f"AND NOT EXISTS (SELECT 1 FROM {ledger} p WHERE p.content_hash = c.content_hash "
        "AND p.chunker_version = c.chunker_version)"
        if ledger
        else ""
    )
    picks = f"""
        SELECT c.chunker_version, c.content_hash,
               min(c.document_id || '#' || lpad(CAST(c.chunk_ordinal AS VARCHAR), 9, '0')) AS pick
        FROM {chunks} c {ready_join}
        WHERE c.content_hash IS NOT NULL AND c.chunker_version IS NOT NULL {not_published}
        GROUP BY c.chunker_version, c.content_hash
        LIMIT {limit}
    """
    vector_columns = ", ".join(
        f"v{i}.embedding AS vec_{i}, v{i}.model AS model_{i}" for i in range(len(vectors))
    )
    vector_joins = " ".join(
        f"JOIN {v} v{i} ON v{i}.content_hash = c.content_hash" for i, v in enumerate(vectors)
    )
    return f"""
        WITH picked AS ({picks})
        SELECT c.* EXCLUDE (embedding), {vector_columns}
        FROM {chunks} c
        JOIN picked k ON c.chunker_version = k.chunker_version AND c.content_hash = k.content_hash
             AND (c.document_id || '#' || lpad(CAST(c.chunk_ordinal AS VARCHAR), 9, '0')) = k.pick
        {vector_joins}
    """


async def publish_pending(
    output_dir: Path,
    store: ChunkCollection,
    slots: list[VectorSlot],
    *,
    max_objects: int = DEFAULT_MAX_OBJECTS,
    run_id: str,
    beat: Callable[[str], None] | None = None,
) -> dict[str, Any]:
    await store.ensure_collection({s.name: s.dimensions for s in slots})
    sql = pending_batch_sql(output_dir, slots, max_objects + 1)
    if sql is None:
        return {"published": 0, "more": False, "reason": "nothing embedded yet"}
    con = duckdb.connect()
    published = 0
    ledger_rows: list[dict[str, Any]] = []
    more = False
    try:
        con.execute("SET memory_limit='1GB'")
        reader = con.execute(sql).fetch_record_batch(FETCH_ROWS)
        indexed_at = datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")
        models = [s.model for s in slots]
        total_seen = 0
        while True:
            try:
                batch = reader.read_next_batch()
            except StopIteration:
                break
            objects = []
            for row in batch.to_pylist():
                total_seen += 1
                if total_seen > max_objects:
                    more = True
                    break
                object_id = chunk_object_id(row["chunker_version"], row["content_hash"])
                objects.append(
                    {
                        "id": object_id,
                        "properties": chunk_properties(
                            row, embed_models=models, run_id=run_id, indexed_at=indexed_at
                        ),
                        "vectors": {s.name: row[f"vec_{i}"] for i, s in enumerate(slots)},
                    }
                )
                ledger_rows.append(
                    {
                        "chunker_version": row["chunker_version"],
                        "content_hash": row["content_hash"],
                        "object_id": object_id,
                        "document_id": row["document_id"],
                        "vault_key": row.get("vault_key") or "",
                        "relative_path": row.get("relative_path") or "",
                        "run_id": run_id,
                        "published_at": datetime.now(UTC),
                    }
                )
            if objects:
                published += await store.upsert(objects)
                _write_ledger(output_dir, run_id, ledger_rows)
                ledger_rows = []
                if beat is not None:
                    beat(f"published {published} objects")
            if more:
                break
    finally:
        con.close()
    return {"published": published, "more": more}


def _write_ledger(output_dir: Path, run_id: str, rows: list[dict[str, Any]]) -> Path:
    folder = ledger_dir(output_dir)
    folder.mkdir(parents=True, exist_ok=True)
    path = folder / f"{run_id}--{uuid.uuid4().hex[:8]}.parquet"
    pq.write_table(
        pa.Table.from_pylist(rows, schema=LEDGER_SCHEMA),
        path.with_suffix(".partial"),
        compression="zstd",
    )
    path.with_suffix(".partial").replace(path)
    return path


def moved_documents_sql(output_dir: Path, limit: int) -> str | None:
    """Documents whose newest locator differs from the vault_key their objects were published
    with."""
    ledger = _ledger_relation(output_dir)
    locators = output_dir / "datasets" / "locators"
    if ledger is None or not (locators.is_dir() and any(locators.glob("*.parquet"))):
        return None
    return f"""
        WITH newest AS (
            SELECT document_id, vault_key, relative_path
            FROM read_parquet('{_posix(locators / "*.parquet")}', union_by_name = true)
            QUALIFY row_number() OVER (PARTITION BY document_id ORDER BY observed_at DESC) = 1
        ), published AS (
            SELECT document_id, vault_key AS published_key, object_id FROM {ledger}
            QUALIFY row_number() OVER (PARTITION BY object_id ORDER BY published_at DESC) = 1
        )
        SELECT n.document_id, n.vault_key, n.relative_path, p.object_id
        FROM newest n JOIN published p ON p.document_id = n.document_id
        WHERE n.vault_key <> p.published_key
        LIMIT {limit}
    """


async def relocate_moved(
    output_dir: Path, store: ChunkCollection, *, max_objects: int = DEFAULT_MAX_OBJECTS, run_id: str
) -> dict[str, Any]:
    sql = moved_documents_sql(output_dir, max_objects)
    if sql is None:
        return {"patched": 0, "missing": 0}
    con = duckdb.connect()
    try:
        rows = con.execute(sql).fetchall()
    finally:
        con.close()
    patched = missing = 0
    updated: list[dict[str, Any]] = []
    for document_id, vault_key, relative_path, object_id in rows:
        if await store.patch(object_id, {"vault_key": vault_key, "catalog_path": relative_path}):
            patched += 1
            updated.append(
                {
                    "chunker_version": "",
                    "content_hash": "",
                    "object_id": object_id,
                    "document_id": document_id,
                    "vault_key": vault_key,
                    "relative_path": relative_path,
                    "run_id": run_id,
                    "published_at": datetime.now(UTC),
                }
            )
        else:
            missing += 1
    if updated:
        # Same ledger, newest row wins on read: the patched key becomes the published key.
        _write_ledger(output_dir, f"{run_id}-relocate", updated)
    return {"patched": patched, "missing": missing}
