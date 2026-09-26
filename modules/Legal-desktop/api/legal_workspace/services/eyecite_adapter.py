"""eyecite structure parse. Not a citator. Not subsequent-history.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Validate = parse succeeded + known reporter.
Normalize = cleaned / corrected citation string.
is_citator_verified is always false.
"""

from __future__ import annotations

from pydantic import BaseModel, Field


class ParsedCitation(BaseModel):
    raw: str
    normalized: str
    reporter: str | None = None
    volume: str | None = None
    page: str | None = None
    year: str | None = None
    court: str | None = None
    validated: bool
    known_reporter: bool
    is_citator_verified: bool = Field(
        default=False,
        description="Always false. eyecite does not Shepardize.",
    )


class ParseResult(BaseModel):
    citations: list[ParsedCitation]
    court_safe: bool = False
    is_citator_verified: bool = False


def parse_citations(text: str) -> ParseResult:
    """Drive eyecite.get_citations on cleaned text. Never marks citator-verified."""
    from eyecite import get_citations
    from eyecite.clean import clean_text

    cleaned = clean_text(text or "", ["html", "inline_whitespace", "all_whitespace", "underscores"])
    found = get_citations(cleaned)
    rows: list[ParsedCitation] = []
    for item in found:
        groups = getattr(item, "groups", {}) or {}
        meta = getattr(item, "metadata", None)
        reporter = groups.get("reporter")
        volume = groups.get("volume")
        page = groups.get("page")
        year = getattr(meta, "year", None) if meta is not None else None
        court = getattr(meta, "court", None) if meta is not None else None
        raw = getattr(item, "matched_text", None) or str(item)
        try:
            normalized = item.corrected_citation()
        except Exception:
            normalized = raw
        known = bool(reporter)
        rows.append(
            ParsedCitation(
                raw=str(raw),
                normalized=str(normalized),
                reporter=str(reporter) if reporter else None,
                volume=str(volume) if volume else None,
                page=str(page) if page else None,
                year=str(year) if year else None,
                court=str(court) if court else None,
                validated=known,
                known_reporter=known,
                is_citator_verified=False,
            )
        )
    return ParseResult(citations=rows, court_safe=False, is_citator_verified=False)
