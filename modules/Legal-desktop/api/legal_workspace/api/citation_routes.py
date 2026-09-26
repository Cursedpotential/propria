"""Citation structure HTTP. Not a citator. Not subsequent-history.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from legal_workspace.services.eyecite_adapter import ParseResult

router = APIRouter()

_MISSING = (
    "Citation parser is not installed on this machine. "
    "This screen parses structure only. It does not Shepardize."
)


class CitationParseRequest(BaseModel):
    text: str = Field(min_length=1)


@router.post("/v1/citations:parse", response_model=ParseResult)
def parse_citation_text(body: CitationParseRequest) -> ParseResult:
    try:
        from legal_workspace.services.eyecite_adapter import parse_citations

        return parse_citations(body.text)
    except ImportError as exc:
        raise HTTPException(status_code=503, detail=_MISSING) from exc
    except ModuleNotFoundError as exc:
        raise HTTPException(status_code=503, detail=_MISSING) from exc
