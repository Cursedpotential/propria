"""The summaries dataset as an overlay on the documents dataset. One unit, one job: the SQL
relation.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner 2026-09-18/21: "THE SUMMARY IS NEEDED", run as its own pass after an item is indexed, never
inline. The
summary pass (summary_stage.py) writes immutable Parquet under ``datasets/summaries``; this relation
lays the newest
summary of each (document, version) over the document row the extract stage wrote, so every reader
(search, lake
SQL, lookups) sees the summary without the document row ever being rewritten. A document with no
summary reads
exactly as before.
"""

from __future__ import annotations

from pathlib import Path

SCALAR_COLUMNS = (
    "title",
    "document_type",
    "document_date",
    "date_basis",
    "short_summary",
    "detailed_summary",
    "case_relevance",
    "language",
)
LIST_COLUMNS = (
    "people",
    "organizations",
    "locations",
    "dates_mentioned",
    "topics",
    "keywords",
    "review_notes",
)


def _posix(path: Path) -> str:
    return path.resolve().as_posix().replace("'", "''")


def documents_relation(output_dir: Path) -> str:
    """A FROM-clause relation: the documents dataset, with the newest summary overlaid where one
    exists."""
    documents = output_dir / "datasets" / "documents" / "*.parquet"
    base = f"read_parquet('{_posix(documents)}', union_by_name = true)"
    summaries = output_dir / "datasets" / "summaries"
    if not summaries.is_dir() or not any(summaries.glob("*.parquet")):
        return base
    replaced = [
        f"COALESCE(NULLIF(s.{c}, ''), d0.{c}) AS {c}"
        for c in ("title", "short_summary", "detailed_summary", "case_relevance")
    ]
    replaced += [
        "COALESCE(NULLIF(s.document_type, 'unknown'), d0.document_type) AS document_type",
        "COALESCE(s.document_date, d0.document_date) AS document_date",
        "COALESCE(NULLIF(s.date_basis, 'unknown'), d0.date_basis) AS date_basis",
        "COALESCE(NULLIF(s.language, 'unknown'), d0.language) AS language",
    ]
    replaced += [f"COALESCE(s.{c}, d0.{c}) AS {c}" for c in LIST_COLUMNS]
    replaced += [
        "COALESCE(s.confidence, d0.confidence) AS confidence",
        "COALESCE(s.summary_model, d0.summary_model) AS summary_model",
        "COALESCE(s.summary_coverage, d0.summary_coverage) AS summary_coverage",
        "COALESCE(s.summary_coverage_ratio, d0.summary_coverage_ratio) AS summary_coverage_ratio",
    ]
    newest = (
        f"SELECT * FROM read_parquet('{_posix(summaries / '*.parquet')}', union_by_name = true) "
        "QUALIFY row_number() OVER (PARTITION BY document_id, version_id "
        "ORDER BY summarized_at DESC) = 1"
    )
    return (
        f"(SELECT d0.* REPLACE ({', '.join(replaced)}) FROM {base} d0 "
        f"LEFT JOIN ({newest}) s ON s.document_id = d0.document_id "
        "AND s.version_id = d0.version_id)"
    )
