"""Verify the source query carries current bucket identity into the configured fence.

Byline: Codex GPT-6, 2026-10-14. This reproduces the deployed zero-file regression
without connecting to a live catalog or reading source payloads.
"""

from pathlib import Path

import duckdb


def test_current_catalog_rows_retain_bucket_identity_and_occurrences() -> None:
    """Read current rows and preserve the occurrence metadata needed by the source selector.

    Inputs: an in-memory current-view fixture. Output: assertions on query results.
    Side effects: temporary DuckDB tables only. Pick this regression before changing
    the default catalog SQL or its provider/bucket projection.
    """
    query = (
        Path(__file__).parents[1] / "src/casebible_index/sql/catalog_source.sql"
    ).read_text(encoding="utf-8")
    with duckdb.connect() as con:
        con.execute("ATTACH ':memory:' AS catalog")
        con.execute("CREATE SCHEMA catalog.raw_duck")
        con.execute(
            "CREATE TABLE catalog.raw_duck.superindex_source_current ("
            "provider VARCHAR, bucket VARCHAR, key VARCHAR, size BIGINT, sha1 VARCHAR, "
            "md5 VARCHAR, modtime TIMESTAMP, listed_at TIMESTAMP, name VARCHAR, "
            "occurrence_count BIGINT, occurrences JSON)"
        )
        occurrences = '[{"disposition":"content_on_b2","original_path":"native/source.txt"}]'
        con.execute(
            "INSERT INTO catalog.raw_duck.superindex_source_current VALUES "
            "('b2', 'salem-data', 'new/source.txt', 42, 'sha1', NULL, NULL, "
            "'2026-10-14', 'source.txt', 1, ?)",
            [occurrences],
        )
        row = con.execute(query).fetchone()
        columns = [item[0] for item in con.description]
        result = dict(zip(columns, row, strict=True))
    assert f"{result['provider']}:{result['bucket']}" in {"b2:salem-data"}
    assert result["key"] == "new/source.txt"
    assert result["occurrences"] == occurrences
    assert result["occurrence_count"] == 1
    assert result["resolution"] == "content_on_b2"
