"""Request-side binding for the preview message search filter.

Byline: Claude Code · Opus 5 · 2026-09-20.

This lives in its own module so the search parameters do not grow
``app/runtime/proffer.py``, which is already over the 300-line file cap.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated

from fastapi import Depends, HTTPException, Query
from pydantic import ValidationError

from app.types.proffer_search import MAX_QUERY_LENGTH, PreviewMessageFilter


def message_search_filter(
    q: Annotated[str | None, Query(max_length=MAX_QUERY_LENGTH)] = None,
    has_attachments: Annotated[bool, Query()] = False,
    sender: Annotated[str | None, Query(max_length=128)] = None,
    sent_from: Annotated[datetime | None, Query(alias="from")] = None,
    sent_to: Annotated[datetime | None, Query(alias="to")] = None,
) -> PreviewMessageFilter:
    """Build the bounded filter, failing closed on a contradictory request.

    A cursor is bound to the filter that minted it, so changing any of these
    parameters restarts paging from the first page rather than reusing a cursor
    that belonged to a different result set.
    """
    try:
        return PreviewMessageFilter(
            query=q,
            has_attachments=has_attachments,
            sender=sender,
            sent_from=sent_from,
            sent_to=sent_to,
        )
    except ValidationError as error:
        raise HTTPException(status_code=422, detail=error.errors(include_url=False)) from None


MessageSearchFilter = Annotated[PreviewMessageFilter, Depends(message_search_filter)]

__all__ = ["MessageSearchFilter", "message_search_filter"]
