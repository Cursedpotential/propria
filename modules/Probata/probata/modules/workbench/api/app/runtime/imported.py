"""Workbench BFF routes for the mobile Imported view. Read-only: there is no write route here.

Byline: Claude Code · Sonnet · 2026-10-02

    GET /api/imported/summary                       totals for the live case
    GET /api/imported/sources                       imported files (exports), newest first
    GET /api/imported/sources/{source_id}/threads   the conversations inside one source
    GET /api/imported/threads/{thread_id}/messages  one page of a thread, chat order
    GET /api/imported/calls                         calls (working.call_log when filled, else normalized records)
    GET /api/imported/search?q=                     message search (Weaviate)
    GET /api/imported/identity                      named people a number can be merged into
    GET /api/imported/unknown-numbers?kind=         who still needs naming, most frequent first, as ONE list: numbers with no
                                                    person (no_person) and people still unconfirmed (unconfirmed)
    GET /api/imported/review-queue                  previews waiting for a decision (decide on the Review routes)

Always the live case (the configured live matter); there is no matter or mode parameter. The
decision on a preview stays on the existing Review routes.
"""

from __future__ import annotations

from typing import Annotated, Literal

from fastapi import APIRouter, HTTPException, Path, Query

from app.service import imported as service

router = APIRouter(prefix="/api/imported", tags=["imported"])

Opaque = Annotated[str, Path(min_length=1, max_length=2048, pattern=r"^[A-Za-z0-9_-]+$")]


def _translate(error: service.ImportedError) -> HTTPException:
    return HTTPException(status_code=error.status, detail=error.message)


@router.get("/summary")
async def summary_endpoint():
    try:
        return await service.summary()
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/sources")
async def sources_endpoint(
    limit: Annotated[int, Query(ge=1, le=50)] = 20,
    offset: Annotated[int, Query(ge=0, le=100000)] = 0,
    format: Annotated[Literal["SMS", "Calls", "Facebook", "Other"] | None, Query()] = None,
):
    try:
        return await service.sources(limit=limit, offset=offset, fmt=format)
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/sources/{source_id}/threads")
async def threads_endpoint(
    source_id: Opaque,
    limit: Annotated[int, Query(ge=1, le=50)] = 25,
    offset: Annotated[int, Query(ge=0, le=100000)] = 0,
):
    try:
        return await service.source_threads(source_id, limit=limit, offset=offset)
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/threads/{thread_id}/messages")
async def messages_endpoint(
    thread_id: Opaque,
    cursor: Annotated[str | None, Query(max_length=512)] = None,
    direction: Annotated[Literal["before", "after"], Query()] = "before",
    limit: Annotated[int, Query(ge=1, le=100)] = 40,
    around: Annotated[str | None, Query(max_length=40)] = None,
):
    try:
        return await service.thread_messages(thread_id, cursor=cursor, direction=direction, limit=limit, around=around)
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/calls")
async def calls_endpoint(
    cursor: Annotated[str | None, Query(max_length=512)] = None,
    limit: Annotated[int, Query(ge=1, le=100)] = 40,
):
    try:
        return await service.calls(cursor=cursor, limit=limit)
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/search")
async def search_endpoint(
    q: Annotated[str, Query(min_length=1, max_length=200)],
    limit: Annotated[int, Query(ge=1, le=50)] = 20,
    offset: Annotated[int, Query(ge=0, le=1000)] = 0,
):
    try:
        return await service.search(q, limit=limit, offset=offset)
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/review-queue")
async def review_queue_endpoint():
    try:
        return await service.review_queue()
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/identity")
async def identity_endpoint():
    try:
        return await service.identity()
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/unknown-numbers")
async def unknown_numbers_endpoint(
    kind: Annotated[Literal["all", "no_person", "unconfirmed"], Query()] = "all",
    limit: Annotated[int, Query(ge=1, le=1000)] = 30,
    offset: Annotated[int, Query(ge=0, le=100000)] = 0,
    q: Annotated[str | None, Query(max_length=40)] = None,
):
    try:
        return await service.unknown_numbers(kind=kind, limit=limit, offset=offset, q=q)
    except service.ImportedError as error:
        raise _translate(error) from None


@router.get("/number-status")
async def number_status_endpoint(numbers: Annotated[list[str], Query(max_length=50)]):
    try:
        return await service.number_status([value[:40] for value in numbers])
    except service.ImportedError as error:
        raise _translate(error) from None
