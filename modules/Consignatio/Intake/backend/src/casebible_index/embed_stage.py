"""The embed stage: give every distinct chunk text a vector for one slot. One unit, one job: embed.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Reads the chunk Parquet of the ACTIVE snapshot, finds the distinct ``content_hash`` values that have
no vector yet for
this slot, embeds that text in provider-sized batches, and writes vector shards under
``datasets/vectors/<slot>/``.
Keyed by ``content_hash``, so the same text found in two files, or in a file and its copy, is
embedded once; a vector
already on disk is never recomputed (re-runs and crashes cost nothing).

Memory is bounded by ``max_texts`` (default 20,000 texts, ~50 MB) and ``shard_size`` (512 vectors,
~4 MB): the lake can
hold millions of chunks, one pass touches a bounded slice of them and the workflow runs passes until
none are left.
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

from .embedders import SlotEmbedder
from .snapshots import newest_snapshot

DEFAULT_MAX_TEXTS = 20_000
DEFAULT_SHARD = 512


def _posix(path: Path) -> str:
    return path.resolve().as_posix().replace("'", "''")


def vectors_dir(output_dir: Path, slot: str) -> Path:
    return output_dir / "datasets" / "vectors" / slot


def vector_schema() -> pa.Schema:
    return pa.schema(
        [
            ("content_hash", pa.string()),
            ("slot", pa.string()),
            ("model", pa.string()),
            ("dimensions", pa.int32()),
            ("status", pa.string()),
            ("embedding", pa.list_(pa.float32())),
            ("embedded_at", pa.timestamp("us", tz="UTC")),
        ]
    )


def chunk_source_sql(output_dir: Path) -> str:
    """A FROM relation: the chunks of the active snapshot's documents (every chunk when there is no
    snapshot)."""
    chunks = (
        f"read_parquet('{_posix(output_dir / 'datasets' / 'chunks' / '*.parquet')}', "
        "union_by_name = true)"
    )
    snapshot = newest_snapshot(output_dir)
    if snapshot is None:
        return chunks
    return (
        f"(SELECT c.* FROM {chunks} c SEMI JOIN read_parquet('{_posix(snapshot)}') s "
        "USING (document_id, version_id, artifact_id))"
    )


def select_pending(output_dir: Path, slot: str, limit: int) -> tuple[list[tuple[str, str]], bool]:
    """Up to ``limit`` (content_hash, text) pairs lacking a vector for ``slot``, and whether more
    remain."""
    if not any((output_dir / "datasets" / "chunks").glob("*.parquet")):
        return [], False
    vec = vectors_dir(output_dir, slot)
    have = (
        "SELECT DISTINCT content_hash "
        f"FROM read_parquet('{_posix(vec / '*.parquet')}', union_by_name = true)"
        if vec.is_dir() and any(vec.glob("*.parquet"))
        else "SELECT NULL::VARCHAR AS content_hash WHERE false"
    )
    con = duckdb.connect()
    try:
        con.execute("SET memory_limit='1GB'")
        sql = f"""
            WITH want AS (
                SELECT DISTINCT content_hash FROM {chunk_source_sql(output_dir)}
                WHERE content_hash IS NOT NULL
            ), have AS ({have}),
            pending AS (
                SELECT w.content_hash FROM want w ANTI JOIN have h USING (content_hash)
                LIMIT {limit + 1}
            )
            SELECT c.content_hash, any_value(c.text) AS text
            FROM {chunk_source_sql(output_dir)} c SEMI JOIN pending p USING (content_hash)
            GROUP BY c.content_hash
            ORDER BY c.content_hash
        """
        rows = con.execute(sql).fetchall()
    finally:
        con.close()
    return [(h, t) for h, t in rows[:limit]], len(rows) > limit


async def run_embed(
    output_dir: Path,
    embedder: SlotEmbedder,
    *,
    max_texts: int = DEFAULT_MAX_TEXTS,
    shard_size: int = DEFAULT_SHARD,
    beat: Callable[[str], None] | None = None,
) -> dict[str, Any]:
    slot = embedder.slot
    pending, more = select_pending(output_dir, slot.name, max_texts)
    counts = {"ok": 0, "truncated": 0, "failed": 0}
    run_id = datetime.now(UTC).strftime("%Y%m%dT%H%M%S") + "-" + uuid.uuid4().hex[:8]
    target = vectors_dir(output_dir, slot.name)
    target.mkdir(parents=True, exist_ok=True)
    for part, start in enumerate(range(0, len(pending), shard_size)):
        group = pending[start : start + shard_size]
        vectors, statuses = await embedder.embed([text for _, text in group])
        now = datetime.now(UTC)
        table = pa.Table.from_pylist(
            [
                {
                    "content_hash": h,
                    "slot": slot.name,
                    "model": slot.model,
                    "dimensions": slot.dimensions,
                    "status": status,
                    "embedding": vector,
                    "embedded_at": now,
                }
                for (h, _), vector, status in zip(group, vectors, statuses, strict=True)
            ],
            schema=vector_schema(),
        )
        path = target / f"{run_id}--p{part:05d}.parquet"
        pq.write_table(table, path.with_suffix(".partial"), compression="zstd")
        path.with_suffix(".partial").replace(path)
        for status in statuses:
            counts[status] += 1
        if beat is not None:
            beat(f"embedded {start + len(group)}/{len(pending)} texts for {slot.name}")
    return {
        "slot": slot.name,
        "model": slot.model,
        "selected": len(pending),
        "embedded_ok": counts["ok"],
        "embedded_truncated": counts["truncated"],
        "embedded_failed": counts["failed"],
        "requests": embedder.requests,
        "more": more,
    }
