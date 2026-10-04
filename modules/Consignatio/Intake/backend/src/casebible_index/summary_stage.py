"""The summary stage: a document-level summary pass that runs after indexing. One unit, one job:
summarize.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner 2026-09-18 22:41 and 2026-09-21: "THE SUMMARY IS NEEDED", as its own pass after an item is
indexed, and "we can use
the index to at least roughly identify what actually needs to be summarized". So: discovery never
waits on a model,
and this stage chooses documents FROM THE INDEX (the documents Parquet of the active snapshot) by a
plain policy, then
summarizes a bounded number per pass:

* indexed documents with real text (``INTAKE_SUMMARY_MIN_CHARS``, default 800) that are
document-like (the extensions
  in ``INTAKE_SUMMARY_EXTENSIONS``), not message exports, chat exports, containers or data dumps;
* not yet summarized for this (document, version);
* the text is read from the document's own chunk shards (no bucket read): the whole text when it
fits
  ``summary_max_chars``, otherwise the beginning, middle and end chunks, with the coverage recorded
  honestly.

The result is an immutable Parquet under ``datasets/summaries`` that
``summaries.documents_relation`` lays over the
document row. A failed summary leaves a ``failed`` marker so the same document is not retried every
cycle; the model
is a setting (``NIM_SUMMARY_MODEL``), so switching providers is configuration.
"""

from __future__ import annotations

import asyncio
import hashlib
import os
import uuid
from collections.abc import Awaitable, Callable
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import duckdb
import pyarrow as pa
import pyarrow.parquet as pq

from .models import DocumentEnrichment
from .snapshots import newest_snapshot

DEFAULT_EXTENSIONS = (
    ".pdf",
    ".docx",
    ".rtf",
    ".eml",
    ".htm",
    ".html",
    ".txt",
    ".md",
    ".markdown",
    ".rst",
)
DEFAULT_MIN_CHARS = 800
DEFAULT_MAX_DOCS = 200
CONCURRENCY = 4

SUMMARY_SCHEMA = pa.schema(
    [
        ("document_id", pa.string()),
        ("version_id", pa.string()),
        ("artifact_id", pa.string()),
        ("summarized_at", pa.timestamp("us", tz="UTC")),
        ("summary_model", pa.string()),
        ("summary_coverage", pa.string()),
        ("summary_coverage_ratio", pa.float32()),
        ("title", pa.string()),
        ("document_type", pa.string()),
        ("document_date", pa.string()),
        ("date_basis", pa.string()),
        ("short_summary", pa.string()),
        ("detailed_summary", pa.string()),
        ("people", pa.list_(pa.string())),
        ("organizations", pa.list_(pa.string())),
        ("locations", pa.list_(pa.string())),
        ("dates_mentioned", pa.list_(pa.string())),
        ("topics", pa.list_(pa.string())),
        ("keywords", pa.list_(pa.string())),
        ("case_relevance", pa.string()),
        ("language", pa.string()),
        ("confidence", pa.float32()),
        ("review_notes", pa.list_(pa.string())),
    ]
)


def _posix(path: Path) -> str:
    return path.resolve().as_posix().replace("'", "''")


def policy_from_env() -> dict[str, Any]:
    raw = os.getenv("INTAKE_SUMMARY_EXTENSIONS", "")
    extensions = (
        tuple(e.strip().casefold() for e in raw.split(",") if e.strip()) or DEFAULT_EXTENSIONS
    )
    return {
        "extensions": extensions,
        "min_chars": int(os.getenv("INTAKE_SUMMARY_MIN_CHARS", str(DEFAULT_MIN_CHARS))),
        "max_docs": int(os.getenv("INTAKE_SUMMARY_MAX_DOCS_PER_PASS", str(DEFAULT_MAX_DOCS))),
    }


def select_candidates(output_dir: Path, policy: dict[str, Any]) -> list[dict[str, Any]]:
    documents = output_dir / "datasets" / "documents"
    snapshot = newest_snapshot(output_dir)
    if snapshot is None or not any(documents.glob("*.parquet")):
        return []
    summaries = output_dir / "datasets" / "summaries"
    done = (
        "SELECT DISTINCT document_id, version_id "
        f"FROM read_parquet('{_posix(summaries / '*.parquet')}', "
        "union_by_name = true)"
        if summaries.is_dir() and any(summaries.glob("*.parquet"))
        else "SELECT NULL::VARCHAR AS document_id, NULL::VARCHAR AS version_id WHERE false"
    )
    extensions = ", ".join("'" + e.replace("'", "") + "'" for e in policy["extensions"])
    con = duckdb.connect()
    try:
        rows = con.execute(
            f"""
            SELECT d.document_id, d.version_id, d.artifact_id,
                   d.relative_path, d.filename, d.text_char_count,
                   d.chunk_count
            FROM read_parquet('{_posix(documents / "*.parquet")}', union_by_name = true) d
            SEMI JOIN read_parquet('{_posix(snapshot)}') s
                USING (document_id, version_id, artifact_id)
            ANTI JOIN ({done}) x USING (document_id, version_id)
            WHERE d.index_status = 'indexed' AND d.chunk_count > 0 AND d.text_char_count >= ?
              AND lower(d.extension) IN ({extensions})
            ORDER BY d.text_char_count DESC, d.document_id
            LIMIT ?
            """,
            [policy["min_chars"], policy["max_docs"]],
        ).fetchall()
    finally:
        con.close()
    names = (
        "document_id",
        "version_id",
        "artifact_id",
        "relative_path",
        "filename",
        "text_char_count",
        "chunk_count",
    )
    return [dict(zip(names, row, strict=True)) for row in rows]


def read_document_text(
    output_dir: Path, doc: dict[str, Any], max_chars: int
) -> tuple[str, str, float]:
    """(text, coverage, ratio) from the document's own chunk shards: whole text, or
    beginning/middle/end chunks."""
    pattern = (
        output_dir / "datasets" / "chunks" / f"{doc['document_id']}--{doc['version_id']}--*.parquet"
    )
    con = duckdb.connect()
    try:
        chunks = con.execute(
            "SELECT chunk_ordinal, text "
            f"FROM read_parquet('{_posix(pattern)}', union_by_name = true) "
            "WHERE artifact_id = ? ORDER BY chunk_ordinal",
            [doc["artifact_id"]],
        ).fetchall()
    finally:
        con.close()
    total = sum(len(t) for _, t in chunks)
    if total <= max_chars:
        return "\n".join(t for _, t in chunks), "full", 1.0
    third = max_chars // 3
    picked: list[str] = []
    for label, group in (
        ("BEGINNING", _take(chunks, third, "head")),
        ("MIDDLE", _take(chunks, third, "middle")),
        ("END", _take(chunks, third, "tail")),
    ):
        picked.append(f"[{label}]\n" + group)
    text = "\n\n".join(picked)
    return text, "representative", min(1.0, len(text) / total)


def _take(chunks: list[tuple[int, str]], budget: int, where: str) -> str:
    if not chunks:
        return ""
    ordered = chunks if where != "tail" else list(reversed(chunks))
    if where == "middle":
        mid = len(chunks) // 2
        ordered = chunks[mid:] + list(reversed(chunks[:mid]))
    out: list[str] = []
    used = 0
    for _, text in ordered:
        if used + len(text) > budget and out:
            break
        out.append(text[: budget - used] if not out else text)
        used += len(out[-1])
    return "\n".join(reversed(out) if where == "tail" else out)


def write_summary(
    output_dir: Path,
    doc: dict[str, Any],
    model: str,
    coverage: str,
    ratio: float,
    enrichment: DocumentEnrichment | None,
) -> Path:
    e = enrichment or DocumentEnrichment()
    row = {
        "document_id": doc["document_id"],
        "version_id": doc["version_id"],
        "artifact_id": doc["artifact_id"],
        "summarized_at": datetime.now(UTC),
        "summary_model": model,
        "summary_coverage": coverage,
        "summary_coverage_ratio": ratio,
        "title": e.title,
        "document_type": e.document_type,
        "document_date": e.document_date,
        "date_basis": e.date_basis,
        "short_summary": e.short_summary,
        "detailed_summary": e.detailed_summary,
        "people": e.people,
        "organizations": e.organizations,
        "locations": e.locations,
        "dates_mentioned": e.dates_mentioned,
        "topics": e.topics,
        "keywords": e.keywords,
        "case_relevance": e.case_relevance,
        "language": e.language,
        "confidence": e.confidence,
        "review_notes": e.review_notes,
    }
    folder = output_dir / "datasets" / "summaries"
    folder.mkdir(parents=True, exist_ok=True)
    stem = (
        f"{doc['document_id']}--{doc['version_id']}--"
        f"{hashlib.sha256(model.encode()).hexdigest()[:8]}"
    )
    path = folder / f"{stem}--{uuid.uuid4().hex[:6]}.parquet"
    pq.write_table(
        pa.Table.from_pylist([row], schema=SUMMARY_SCHEMA),
        path.with_suffix(".partial"),
        compression="zstd",
    )
    path.with_suffix(".partial").replace(path)
    return path


Summarize = Callable[..., Awaitable[DocumentEnrichment]]


async def run_summary(
    output_dir: Path,
    summarize: Summarize,
    model: str,
    *,
    max_chars: int,
    policy: dict[str, Any] | None = None,
    beat: Callable[[str], None] | None = None,
) -> dict[str, Any]:
    policy = policy or policy_from_env()
    candidates = await asyncio.to_thread(select_candidates, output_dir, policy)
    done = failed = 0
    semaphore = asyncio.Semaphore(CONCURRENCY)

    async def one(doc: dict[str, Any]) -> bool:
        async with semaphore:
            text, coverage, ratio = await asyncio.to_thread(
                read_document_text, output_dir, doc, max_chars
            )
            try:
                enrichment = await summarize(
                    filename=doc["filename"],
                    relative_path=doc["relative_path"],
                    text=text,
                    source_created_at=None,
                    source_modified_at=None,
                    coverage=coverage,
                )
            # noqa: BLE001 - a provider failure is recorded, never fatal to the pass
            except Exception:
                await asyncio.to_thread(write_summary, output_dir, doc, model, "failed", 0.0, None)
                return False
            await asyncio.to_thread(
                write_summary, output_dir, doc, model, coverage, ratio, enrichment
            )
            return True

    for start in range(0, len(candidates), CONCURRENCY * 4):
        results = await asyncio.gather(
            *(one(d) for d in candidates[start : start + CONCURRENCY * 4])
        )
        done += sum(results)
        failed += len(results) - sum(results)
        if beat is not None:
            beat(f"summarized {done}, failed {failed} of {len(candidates)}")
    return {
        "selected": len(candidates),
        "summarized": done,
        "failed": failed,
        "model": model,
        "more": len(candidates) >= policy["max_docs"],
        "policy": {k: policy[k] for k in ("min_chars", "max_docs")},
    }
