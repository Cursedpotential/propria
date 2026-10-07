"""Expose bounded read-only context composition through existing Platform API owner authentication.

Inputs are typed queries or compact read_retrieval envelopes; outputs retain citations and explicit per-leg failures.
Only configured readers perform HTTP reads/query embeddings. Register on the existing app; no service, index,
evidence authority, worker binding or deployment is created. Pick direct retrieval for UI, compact flow for n8n/Temporal.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import importlib.util
import json
from typing import Any, Literal
from uuid import uuid4

from fastapi import FastAPI, HTTPException, Request
from pydantic import Field, ValidationError

from server.api.platform_auth import require_platform_owner
from server.core.retrieval_composition import retrieve_context
from server.core.retrieval_contracts import (
    Identifier,
    LegName,
    ReadModel,
    RetrievalRequest,
    compact_flow_result,
    request_from_flow,
)
from server.core.retrieval_readers import ContextReaders, ReaderConfig

MAX_REQUEST_BYTES = 32 << 10


class PublicScope(ReadModel):
    """Accept source locators without caller matter/authority; output is validated scope, with no permission or store effects."""

    source_id: Identifier | None = None
    source_version_ids: tuple[Identifier, ...] = Field(default=(), max_length=32)
    thread_id: Identifier | None = None
    path_prefix: Identifier | None = None


class ContextQuery(ReadModel):
    """Accept bounded public query knobs; output excludes reader URLs/credentials/matter, with no I/O."""

    request_id: Identifier = Field(default_factory=lambda: str(uuid4()))
    query: str = Field(min_length=1, max_length=2000, pattern=r"\S")
    mode: Literal["keyword", "hybrid", "vector"] = "keyword"
    scope: PublicScope = Field(default_factory=PublicScope)
    legs: tuple[LegName, ...] = Field(default=("intake", "proffer"), min_length=1, max_length=2)
    limit: int = Field(default=20, ge=1, le=50, strict=True)
    per_leg_limit: int = Field(default=50, ge=1, le=100, strict=True)
    leg_timeout_seconds: float = Field(default=10, gt=0, le=30, strict=True)

    def internal(self) -> RetrievalRequest:
        """Return a validated internal request without selecting authority; Proffer's reader injects configured matter only."""
        request = RetrievalRequest.model_validate(self.model_dump())
        if set(request.legs) - {"intake", "proffer"}:
            raise ValueError("reader is not allowlisted")
        return request


async def request_body(request: Request) -> dict[str, Any]:
    """Read bounded authenticated JSON; return an object or sanitized 4xx without recording query/corpus payloads."""
    require_platform_owner(request)
    raw = bytearray()
    async for chunk in request.stream():
        raw.extend(chunk)
        if len(raw) > MAX_REQUEST_BYTES:
            raise HTTPException(413, "retrieval request exceeds bound")
    try:
        value = json.loads(raw)
        if not isinstance(value, dict):
            raise TypeError("expected object")
        return value
    except (ValueError, TypeError):
        raise HTTPException(422, "invalid retrieval JSON") from None


def dependencies_available() -> bool:
    """Check optional framework availability without network or provider creation; return whether composition can be invoked."""
    try:
        return all(importlib.util.find_spec(name) is not None for name in ("llama_index.core", "langgraph.graph"))
    except ModuleNotFoundError:
        return False


def register_context_retrieval_routes(app: FastAPI, *, readers: ContextReaders | None = None) -> None:
    """Register three authenticated read routes on an existing app; inputs are app/server readers, effects are route bindings only.

    Outputs are route registrations. Default readers use server environment allowlists; inject a transport for synthetic tests.
    Parent owns app registration/deployment and installs the retrieval extra; this function creates no server or worker.
    """
    def configured_readers() -> ContextReaders:
        """Resolve current server-only reader settings without I/O; use injected ports only for trusted server/test callers."""
        return readers if readers is not None else ContextReaders(ReaderConfig.from_environment())

    async def execute(query: RetrievalRequest, *, compact: bool = False) -> dict[str, Any]:
        """Execute configured read ports; return cited/compact results, with sanitized dependency and envelope errors."""
        if not dependencies_available():
            raise HTTPException(503, "context retrieval dependencies are unavailable")
        response = await retrieve_context(query, configured_readers().legs())
        try:
            return compact_flow_result(response) if compact else response.model_dump(mode="json")
        except ValueError:
            raise HTTPException(422, "compact retrieval response exceeds bound; lower the result limit") from None

    @app.post("/v1/context/retrieve")
    async def retrieve(request: Request) -> dict[str, Any]:
        """Read Intake/Proffer context for a bounded query; return original citations and partial failures without writes.

        Inputs: ContextQuery JSON and Platform owner bearer. Output: RetrievalResponse; default legs intake/proffer.
        Proffer is server-matter scoped, Intake global unless unsupported caller scope fails that leg. No evidence access.
        """
        body = await request_body(request)
        try:
            query = ContextQuery.model_validate(body).internal()
        except (ValidationError, ValueError):
            raise HTTPException(422, "invalid or unsupported context retrieval request") from None
        return await execute(query)

    @app.get("/v1/context/retrieval-capabilities")
    async def capabilities(request: Request) -> dict[str, Any]:
        """Return configured reader capabilities after owner authentication; no health probe, provider or corpus read occurs.

        Input: Platform owner bearer. Output: configured modes/scopes, dependency state and explicit graph/evidence gaps.
        Pick before rendering controls; configured support does not establish live health or complete index coverage.
        """
        require_platform_owner(request)
        return {"modes": ["keyword", "hybrid", "vector"], "default_legs": ["intake", "proffer"],
                "composition_available": dependencies_available(), **configured_readers().capabilities()}

    @app.post("/v1/context/retrieve-flow")
    async def retrieve_flow(request: Request) -> dict[str, Any]:
        """Read the compact named-flow contract for n8n/Temporal; return citations without excerpt bodies or durable retries.

        Input: read_retrieval FlowRequest and Platform owner bearer. Output: bounded FlowResult with explicit partial status.
        Only allowlisted Intake/Proffer readers run. Server matter is never accepted through scope_json or other caller knobs.
        """
        body = await request_body(request)
        try:
            query = request_from_flow(body)
            scope = PublicScope.model_validate(query.scope.model_dump(exclude_unset=True))
            query = ContextQuery.model_validate({**query.model_dump(), "scope": scope.model_dump()}).internal()
        except (ValidationError, ValueError, TypeError):
            raise HTTPException(422, "invalid or unsupported retrieval flow request") from None
        return await execute(query, compact=True)
