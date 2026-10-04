"""Bounded discovery: read the catalog, decide whether it changed, represent every object. One unit,
one job.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Catalog first (owner 2026-09-16): listings land in ``raw_duck.bucket_objects`` and the index reads
``raw_duck.bucket_objects_current``; no bucket is ever listed here. Two cheap reads:

1. the WATERMARK, one row per (provider, bucket): the listing time, object count and bytes of the
newest snapshot. If it
   equals the last committed cycle's watermark for every configured bucket, nothing changed and the
   cycle ends here.
2. only when something changed (or forced): one streamed pass over the listing in bounded Arrow
batches that classifies
   every key (routing.classify_key) and writes the objects the extract stage will NOT read (media,
   unsupported) into
   a represented-inventory Parquet with their locator, size and sha1. Documents and archives go to
   the extract stage;
   their CocoIndex memo (catalog size + sha1) makes an unchanged object cost zero bucket reads.

The Case Bible holds everything; being in it does not mean a file is relevant. Nothing is filtered
by relevance.
"""

from __future__ import annotations

import os
from collections import Counter
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path, PurePosixPath
from typing import Any

import duckdb
import pyarrow as pa
import pyarrow.parquet as pq

from .catalog_source import _attach
from .routing import (
    KIND_ARCHIVE,
    KIND_DOCUMENT,
    classify_key,
    inventory_status,
    is_excluded,
    is_indexable_kind,
)

WATERMARK_SQL = """
SELECT provider, bucket, max(listed_at) AS listed_at, count(*) AS objects, sum(size) AS bytes
FROM catalog.raw_duck.bucket_objects_current
GROUP BY provider, bucket
ORDER BY provider, bucket
"""
LISTING_SQL = (
    "SELECT provider, bucket, key, size, sha1, listed_at "
    "FROM catalog.raw_duck.bucket_objects_current"
)
BATCH_ROWS = 50_000

REPRESENTED_SCHEMA = pa.schema(
    [
        ("provider", pa.string()),
        ("bucket", pa.string()),
        ("key", pa.string()),
        ("size", pa.int64()),
        ("sha1", pa.string()),
        ("kind", pa.string()),
        ("status", pa.string()),
        ("listed_at", pa.timestamp("us", tz="UTC")),
        ("cycle_id", pa.string()),
    ]
)


@dataclass(frozen=True)
class BucketMark:
    provider: str
    bucket: str
    listed_at: str
    objects: int
    bytes: int

    @property
    def name(self) -> str:
        return f"{self.provider}:{self.bucket}"

    def as_dict(self) -> dict[str, Any]:
        return {
            "provider": self.provider,
            "bucket": self.bucket,
            "listed_at": self.listed_at,
            "objects": self.objects,
            "bytes": self.bytes,
        }


def read_watermark(dsn: str, wanted: set[str] | None = None) -> list[BucketMark]:
    con = duckdb.connect()
    try:
        _attach(con, dsn)
        rows = con.execute(WATERMARK_SQL).fetchall()
    finally:
        con.close()
    marks = [
        BucketMark(
            p,
            b,
            listed.astimezone(UTC).isoformat() if hasattr(listed, "astimezone") else str(listed),
            int(n),
            int(sz or 0),
        )
        for p, b, listed, n, sz in rows
    ]
    return [m for m in marks if not wanted or m.name in wanted]


def changed_buckets(previous: list[dict[str, Any]] | None, current: list[BucketMark]) -> list[str]:
    """Buckets whose newest listing differs from the committed watermark (time, object count or
    bytes)."""
    before = {f"{m['provider']}:{m['bucket']}": m for m in (previous or [])}
    out = []
    for mark in current:
        old = before.get(mark.name)
        if old is None or (old["listed_at"], old["objects"], old["bytes"]) != (
            mark.listed_at,
            mark.objects,
            mark.bytes,
        ):
            out.append(mark.name)
    return out


def classify_listing(
    dsn: str,
    wanted: set[str],
    out_path: Path,
    cycle_id: str,
    *,
    path_prefix: str = "",
    batch_rows: int = BATCH_ROWS,
) -> dict[str, Any]:
    """One streamed pass: count every kind, write the objects the extract stage will not read to
    ``out_path``."""
    con = duckdb.connect()
    kinds: Counter[str] = Counter()
    sizes: Counter[str] = Counter()
    extensions: Counter[str] = Counter()
    excluded = 0
    represented = 0
    out_path.parent.mkdir(parents=True, exist_ok=True)
    partial = out_path.with_suffix(".partial")
    writer = pq.ParquetWriter(partial, REPRESENTED_SCHEMA, compression="zstd")
    # Top-level PDFs are documents (the extract stage reads them), but the ones with no text layer
    # are scanned pages
    # for the image stage (image_slice.select_scanned_pdfs). Their locators are listed here, once,
    # so that stage never
    # re-reads the catalog (Claude Code · Sonnet 5.5 · 2026-10-03).
    pdf_path = out_path.parent.parent / "pdf" / out_path.name
    pdf_path.parent.mkdir(parents=True, exist_ok=True)
    pdf_partial = pdf_path.with_suffix(".partial")
    pdf_writer = pq.ParquetWriter(pdf_partial, REPRESENTED_SCHEMA, compression="zstd")
    pdf_rows = 0
    try:
        _attach(con, dsn)
        reader = con.execute(LISTING_SQL).fetch_record_batch(batch_rows)
        while True:
            try:
                batch = reader.read_next_batch()
            except StopIteration:
                break
            rows: list[dict[str, Any]] = []
            pdfs: list[dict[str, Any]] = []
            for row in batch.to_pylist():
                if wanted and f"{row['provider']}:{row['bucket']}" not in wanted:
                    continue
                key = row["key"]
                if path_prefix and not key.startswith(path_prefix):
                    continue
                if is_excluded(key):
                    excluded += 1
                    continue
                kind = classify_key(key)
                kinds[kind] += 1
                sizes[kind] += int(row["size"] or 0)
                if kind in (KIND_DOCUMENT, KIND_ARCHIVE):
                    extensions[PurePosixPath(key).suffix.casefold()] += 1
                if kind == KIND_DOCUMENT and key.casefold().endswith(".pdf"):
                    pdfs.append(
                        {
                            "provider": row["provider"],
                            "bucket": row["bucket"],
                            "key": key,
                            "size": int(row["size"] or 0),
                            "sha1": row["sha1"] or "",
                            "kind": kind,
                            "status": "pdf_candidate",
                            "listed_at": row["listed_at"],
                            "cycle_id": cycle_id,
                        }
                    )
                if not is_indexable_kind(kind):
                    rows.append(
                        {
                            "provider": row["provider"],
                            "bucket": row["bucket"],
                            "key": key,
                            "size": int(row["size"] or 0),
                            "sha1": row["sha1"] or "",
                            "kind": kind,
                            "status": inventory_status(kind),
                            "listed_at": row["listed_at"],
                            "cycle_id": cycle_id,
                        }
                    )
            if rows:
                writer.write_table(pa.Table.from_pylist(rows, schema=REPRESENTED_SCHEMA))
                represented += len(rows)
            if pdfs:
                pdf_writer.write_table(pa.Table.from_pylist(pdfs, schema=REPRESENTED_SCHEMA))
                pdf_rows += len(pdfs)
    finally:
        writer.close()
        pdf_writer.close()
        con.close()
    os.replace(partial, out_path)
    os.replace(pdf_partial, pdf_path)
    return {
        "kinds": dict(kinds),
        "bytes_by_kind": dict(sizes),
        "excluded": excluded,
        "represented_inventory": str(out_path),
        "represented_rows": represented,
        "pdf_inventory": str(pdf_path),
        "pdf_rows": pdf_rows,
        "top_text_extensions": dict(extensions.most_common(15)),
    }


def discover(
    settings,
    dsn: str,
    cycle_id: str,
    previous_watermark: list[dict[str, Any]] | None,
    *,
    force: bool = False,
) -> dict[str, Any]:
    """The discovery stage. Returns the receipt: whether anything changed, the new watermark, the
    classification."""
    wanted = set(settings.source_buckets)
    marks = read_watermark(dsn, wanted)
    changed = changed_buckets(previous_watermark, marks)
    receipt: dict[str, Any] = {
        "started_at": datetime.now(UTC).isoformat(),
        "watermark": [m.as_dict() for m in marks],
        "changed_buckets": changed,
        "forced": force,
        "changed": bool(changed) or force,
        "buckets_configured": sorted(wanted),
        "buckets_missing_from_catalog": sorted(wanted - {m.name for m in marks}),
    }
    if receipt["changed"]:
        out = settings.output_dir / "inventory" / "represented" / f"{cycle_id}.parquet"
        receipt.update(
            classify_listing(dsn, wanted, out, cycle_id, path_prefix=settings.catalog_path_prefix)
        )
    return receipt
