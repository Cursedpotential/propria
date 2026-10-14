"""Bounded summary retry and safe failure-marker tests."""

from pathlib import Path

import httpx
import pyarrow as pa
import pyarrow.parquet as pq
import pytest

from casebible_index.models import DocumentEnrichment
from casebible_index.nim import NimError
from casebible_index.summary_stage import (
    _safe_failure_notes,
    run_summary,
    select_candidates,
    write_summary,
)

POLICY = {"extensions": (".txt",), "min_chars": 100, "max_docs": 10}


def _lake(root: Path) -> Path:
    """Write one synthetic indexed document, active snapshot and text chunk to a test lake."""
    out = root / "lake"
    documents = out / "datasets" / "documents"
    chunks = out / "datasets" / "chunks"
    snapshots = out / "snapshots" / "fixture"
    for folder in (documents, chunks, snapshots):
        folder.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [{
                "document_id": "doc", "version_id": "v1", "artifact_id": "artifact",
                "relative_path": "fixture.txt", "filename": "fixture.txt",
                "text_char_count": 900, "chunk_count": 1, "index_status": "indexed",
                "extension": ".txt",
            }]
        ),
        documents / "doc.parquet",
    )
    pq.write_table(
        pa.Table.from_pylist(
            [{"document_id": "doc", "version_id": "v1", "artifact_id": "artifact"}]
        ),
        snapshots / "active_documents.parquet",
    )
    pq.write_table(
        pa.Table.from_pylist([{"artifact_id": "artifact", "chunk_ordinal": 0, "text": "x" * 900}]),
        chunks / "doc--v1--0.parquet",
    )
    return out


def _doc() -> dict[str, str]:
    """Return the synthetic document identity used by immutable marker writes."""
    return {"document_id": "doc", "version_id": "v1", "artifact_id": "artifact"}


def test_wrapped_validation_exposes_field_only() -> None:
    """Extract a validation field through a NIM wrapper without storing input values."""
    secret = "private-input-value-and-api-key"
    try:
        DocumentEnrichment.model_validate({"confidence": secret})
    except ValueError as validation:
        try:
            raise NimError("NIM summary was not valid schema-conformant JSON") from validation
        except NimError as wrapped:
            notes = _safe_failure_notes(wrapped)
    assert notes == [
        "summary_error:NimError",
        "validation_field:confidence",
        "validation:schema_json",
    ]
    assert secret not in " ".join(notes)


def test_legacy_failure_marker_is_retryable_and_success_is_final(tmp_path: Path) -> None:
    """Count a prior marker once and exclude all versions with any success."""
    out = _lake(tmp_path)
    old = write_summary(out, _doc(), "model", "failed", 0.0, None)
    assert len(select_candidates(out, POLICY)) == 1
    success = write_summary(
        out, _doc(), "model", "full", 1.0,
        DocumentEnrichment(short_summary="Supported summary"),
    )
    assert old.is_file() and success.is_file()
    assert select_candidates(out, POLICY) == []


@pytest.mark.asyncio
async def test_failed_attempt_then_valid_success_stays_complete(tmp_path: Path) -> None:
    """Retry on the next pass and preserve both immutable attempt rows."""
    out = _lake(tmp_path)
    calls = 0

    async def summarize(**_kwargs: object) -> DocumentEnrichment:
        """Fail once, then return a valid model object for the same document."""
        nonlocal calls
        calls += 1
        if calls == 1:
            raise RuntimeError("transient provider failure")
        return DocumentEnrichment(short_summary="A supported summary")

    first = await run_summary(out, summarize, "model", max_chars=2000, policy=POLICY)
    second = await run_summary(out, summarize, "model", max_chars=2000, policy=POLICY)
    third = await run_summary(out, summarize, "model", max_chars=2000, policy=POLICY)
    assert (first["failed"], second["summarized"], third["selected"], calls) == (1, 1, 0, 2)
    assert len(list((out / "datasets" / "summaries").glob("*.parquet"))) == 2


@pytest.mark.asyncio
async def test_permanent_failure_stops_at_three_and_keeps_safe_cause(tmp_path: Path) -> None:
    """Persist only allowlisted error metadata and cap attempts without deleting rows."""
    out = _lake(tmp_path)
    secret = "secret-key-private-input"

    async def broken(**_kwargs: object) -> DocumentEnrichment:
        """Raise an HTTP error carrying private text that must not enter Parquet."""
        request = httpx.Request("POST", "https://example.invalid/chat")
        response = httpx.Response(503, request=request, text=secret)
        raise httpx.HTTPStatusError(secret, request=request, response=response)

    results = [
        await run_summary(out, broken, "model", max_chars=2000, policy=POLICY)
        for _ in range(4)
    ]
    assert [result["selected"] for result in results] == [1, 1, 1, 0]
    files = list((out / "datasets" / "summaries").glob("*.parquet"))
    assert len(files) == 3
    for path in files:
        notes = pq.read_table(path, columns=["review_notes"])["review_notes"][0].as_py()
        assert notes == ["summary_error:HTTPStatusError", "http_status:503"]
        assert secret.encode() not in path.read_bytes()
