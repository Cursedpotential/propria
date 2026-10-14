"""Verify staged embedding preserves legacy shards across schema evolution.

Byline: Codex GPT-6, 2026-10-14. Fixtures represent the deployed September schema
and current content-addressed schema; no provider or production data is used.
"""

import hashlib
from pathlib import Path

import pyarrow as pa
import pyarrow.parquet as pq

from casebible_index.embed_stage import select_pending


def test_legacy_shards_do_not_fail_or_gain_guessed_identity(tmp_path: Path) -> None:
    """Keep old embedded rows untouched while selecting only explicitly identified new chunks.

    Inputs: temporary test storage. Output: assertions over pending selection and bytes.
    Side effects: writes fixture Parquet files only. Pick this regression when changing
    the shared embedding/publication relation or migrating historical index schemas.
    """
    chunks = tmp_path / "datasets" / "chunks"
    chunks.mkdir(parents=True)
    legacy = chunks / "legacy.parquet"
    pq.write_table(
        pa.Table.from_pylist(
            [{"document_id": "old", "text": "retained original chunk", "text_sha256": "old-hash"}]
        ),
        legacy,
    )
    original = hashlib.sha256(legacy.read_bytes()).digest()

    assert select_pending(tmp_path, "text_nim", 10) == ([], False)

    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "document_id": "new",
                    "text": "new identified chunk",
                    "content_hash": "new-hash",
                    "chunker_version": "known-version",
                }
            ]
        ),
        chunks / "current.parquet",
    )
    assert select_pending(tmp_path, "text_nim", 10) == (
        [("new-hash", "new identified chunk")],
        False,
    )
    assert hashlib.sha256(legacy.read_bytes()).digest() == original
