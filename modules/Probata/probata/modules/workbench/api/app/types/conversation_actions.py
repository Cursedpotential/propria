"""Conversation actions: request bodies for Extract and Send to Surreal, and the shapes the Workbench shows.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

A conversation is the id the Imported view already lists (``ImportedThread.id``, an opaque token for one export file plus
one conversation key). The engine (Proffer starter, ``/reference-import/conversations/*``) owns the workflows; these models
only bound what crosses the BFF. Progress and extraction objects are passed through as validated dicts so a new engine
field reaches the panel without a BFF release.
"""

from __future__ import annotations

from typing import Annotated, Any

from pydantic import BaseModel, ConfigDict, Field, StringConstraints

ThreadId = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=2048, pattern=r"^[A-Za-z0-9_-]+$")]
ExtractorId = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=64, pattern=r"^[a-z0-9][a-z0-9_-]*$")]
IdempotencyKey = Annotated[str, StringConstraints(strip_whitespace=True, min_length=8, max_length=200)]
WorkflowId = Annotated[str, StringConstraints(strip_whitespace=True, min_length=8, max_length=512)]


class ExtractConversationsRequest(BaseModel):
    """Extract entities and events from the chosen conversations with the chosen extractors."""

    model_config = ConfigDict(extra="forbid")

    thread_ids: Annotated[list[ThreadId], Field(min_length=1, max_length=200)]
    # Empty means the default extractor. Every registered extractor is selectable; the picker lists them.
    extractors: Annotated[list[ExtractorId], Field(max_length=16)] = []


class SendConversationsRequest(BaseModel):
    """Send whole conversations to surreal-case."""

    model_config = ConfigDict(extra="forbid")

    thread_ids: Annotated[list[ThreadId], Field(min_length=1, max_length=200)]
    include_extractions: bool = True


class WorkflowStarted(BaseModel):
    """The engine accepted the workflow; poll its status route."""

    model_config = ConfigDict(extra="allow")

    workflow_id: str
    run_id: str
    kind: str
    conversations: int


class WorkflowStatus(BaseModel):
    """Progress of an extraction or a send."""

    model_config = ConfigDict(extra="allow")

    workflow_id: str
    kind: str
    outcome: str
    steps: list[dict[str, Any]]
    receipts: list[dict[str, Any]]
