"""Define bounded, transport-neutral context retrieval envelopes without storage or evidence authority.

Inputs are selected leg names, queries and existing result coordinates; outputs preserve citations.
No I/O occurs here. Use these contracts for composed context reads, not governed evidence search.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import json
from typing import Annotated, Any, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator

FLOW_NAME = "read_retrieval"
# Match engine/temporal/n8n_client.go's compact HTTP response bound.
MAX_FLOW_RESPONSE_BYTES = 64 << 10
Identifier = Annotated[str, StringConstraints(min_length=1, max_length=2048, pattern=r"\S")]
LegName = Annotated[str, StringConstraints(pattern=r"^[a-z][a-z0-9_]{0,63}$")]


class ReadModel(BaseModel):
    """Reject undeclared envelope fields; inputs become immutable models without I/O for context reads."""

    model_config = ConfigDict(extra="forbid", frozen=True, allow_inf_nan=False)


class Citation(ReadModel):
    """Retain supplied source coordinates; output is provenance metadata, without source validation or writes."""

    collection: Identifier
    object_id: Identifier
    source_id: Identifier | None = None
    source_version_ids: tuple[Identifier, ...] = Field(default=(), max_length=32)
    document_id: Identifier | None = None
    chunk_id: Identifier | None = None
    first_message_id: Identifier | None = None
    normalized_record_id: Identifier | None = None
    source_sha256: Identifier | None = None
    content_hash: Identifier | None = None
    vector_name: Identifier | None = None
    locator: dict[Identifier, Identifier] = Field(default_factory=dict, max_length=16)

    def identity(self) -> str:
        """Encode exact supplied coordinates canonically; return an in-memory key without minting corpus IDs."""
        # A vector target describes a retrieval leg, not a distinct source occurrence.
        data = self.model_dump(mode="json", exclude_none=True, exclude={"vector_name"})
        data["source_version_ids"] = sorted(data["source_version_ids"])
        return json.dumps(data, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


class ReadHit(ReadModel):
    """Carry an existing excerpt, score and citation; no parsing, indexing or provenance inference occurs."""

    text: str = Field(max_length=8192)
    score: float
    citation: Citation


class RankedHit(ReadHit):
    """Return library-fused score and contributing leg names while preserving the original excerpt/citation."""

    legs: tuple[LegName, ...]
    vector_targets: dict[LegName, Identifier | None]


class RetrievalScope(ReadModel):
    """Carry caller-authorized pre-ranking coordinates; output is typed scope without deriving permissions or filters."""

    matter_id: Identifier | None = None
    source_id: Identifier | None = None
    source_version_ids: tuple[Identifier, ...] = Field(default=(), max_length=32)
    thread_id: Identifier | None = None
    path_prefix: Identifier | None = None

    def requested_fields(self) -> frozenset[str]:
        """Return nonempty scope names for reader capability checks, without querying or applying post-filters."""
        return frozenset(self.model_dump(exclude_none=True, exclude_defaults=True))


class RetrievalRequest(ReadModel):
    """Bound a context query and selected injected legs; output validates without provider or storage access."""

    request_id: Identifier
    query: str = Field(min_length=1, max_length=2000, pattern=r"\S")
    mode: Literal["keyword", "hybrid"] = "keyword"
    scope: RetrievalScope = Field(default_factory=RetrievalScope)
    legs: tuple[LegName, ...] = Field(min_length=1, max_length=4)
    limit: int = Field(default=20, ge=1, le=50, strict=True)
    per_leg_limit: int = Field(default=50, ge=1, le=100, strict=True)
    leg_timeout_seconds: float = Field(default=10, gt=0, le=30, strict=True)

    @field_validator("legs")
    @classmethod
    def unique_legs(cls, value: tuple[str, ...]) -> tuple[str, ...]:
        """Reject repeated leg names in input; return unique requested order without duplicate rank weighting."""
        if len(value) != len(set(value)):
            raise ValueError("retrieval legs must be unique")
        return value


class LegOutcome(ReadModel):
    """Expose one leg's count or sanitized failure code; no provider error bodies or corpus data are logged."""

    name: LegName
    status: Literal["success", "failed"]
    count: int = Field(default=0, ge=0, le=100)
    error: Literal[
        "unavailable", "timeout", "invalid_result", "identity_conflict", "unsupported_mode", "unsupported_scope"
    ] | None = None


class RetrievalResponse(ReadModel):
    """Return fused context and explicit leg outcomes; use compact_flow_result before crossing Temporal history."""

    request_id: Identifier
    mode: Literal["keyword", "hybrid"]
    scope: RetrievalScope
    status: Literal["success", "partial", "failed"]
    items: tuple[RankedHit, ...]
    legs: tuple[LegOutcome, ...]
    verification: Literal["coordinates_preserved"] = "coordinates_preserved"


def request_from_flow(payload: dict[str, Any]) -> RetrievalRequest:
    """Adapt a compact named-flow request to a context query without I/O; reject arbitrary URLs and payloads.

    Inputs are flow/request_id and scalar inputs (query, leg_names, mode, scope_json, bounds).
    Output is RetrievalRequest. Pick this over direct model construction for the generic n8n caller.
    """
    if set(payload) != {"flow", "request_id", "inputs"} or payload["flow"] != FLOW_NAME:
        raise ValueError("expected read_retrieval flow, request_id and inputs only")
    inputs = payload["inputs"]
    allowed = {"query", "leg_names", "mode", "scope_json", "limit", "per_leg_limit", "leg_timeout_seconds"}
    if not isinstance(inputs, dict) or set(inputs) - allowed:
        raise ValueError("unsupported retrieval inputs")
    if not isinstance(inputs.get("leg_names"), str):
        raise ValueError("invalid leg_names input")  # noqa: TRY004 -- one validation-error boundary for flow envelopes
    scope = {}
    if "scope_json" in inputs:
        if not isinstance(inputs["scope_json"], str) or len(inputs["scope_json"]) > 16384:
            raise ValueError("invalid scope_json input")
        scope = json.loads(inputs["scope_json"])
    return RetrievalRequest.model_validate({
        "request_id": payload["request_id"],
        **{key: value for key, value in inputs.items() if key not in {"leg_names", "scope_json"}},
        "legs": inputs["leg_names"].split(","),
        "scope": scope,
    })


def compact_flow_result(response: RetrievalResponse) -> dict[str, Any]:
    """Project a read response to the existing FlowResult envelope without excerpt bodies or writes.

    Input is a verified response; output retains citation metadata, ranks and explicit partial status.
    Pick this for Temporal/n8n boundaries; use the full response only for direct context consumers.
    """
    result = {
        "flow": FLOW_NAME,
        "status": "failed" if response.status == "failed" else "success",
        "outputs": response.model_dump(mode="json", exclude={"items": {"__all__": {"text"}}}),
    }
    if len(json.dumps(result, ensure_ascii=False).encode("utf-8")) > MAX_FLOW_RESPONSE_BYTES:
        raise ValueError("compact retrieval response exceeds the n8n Activity response bound; lower the result limit")
    return result
