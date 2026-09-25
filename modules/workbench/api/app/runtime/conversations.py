"""Read-only bout review endpoints for conversations with people. Byline: Claude Code · Opus 5.5 · 2026-09-24."""
from fastapi import APIRouter, HTTPException, Query
from starlette.concurrency import run_in_threadpool

from app.repo.conversations import conversations
from app.repo.intake_discovery import DiscoveryError
from app.service.conversations import bout_review, day

router = APIRouter(prefix="/api/conversations", tags=["conversations"])


@router.get("")
async def list_endpoint():
    """Registered conversations with their bout sets and label passes."""
    try:
        return {"conversations": await run_in_threadpool(conversations)}
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/{conv_key}/bouts")
async def bouts_endpoint(conv_key: str, rules: str | None = Query(None, max_length=200),
                         label_pass: str | None = Query(None, alias="pass", max_length=200)):
    """Every bout of one bout set with the chosen label pass, normalized to segments, shifts and a summary."""
    try:
        return await run_in_threadpool(bout_review, conv_key, rules, label_pass)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None


@router.get("/{conv_key}/days/{day_value}")
async def day_endpoint(conv_key: str, day_value: str, rules: str = Query(..., max_length=200)):
    """Messages of every bout on one day, with text and source files."""
    try:
        return await run_in_threadpool(day, conv_key, rules, day_value)
    except DiscoveryError as error:
        raise HTTPException(error.status, error.message) from None
