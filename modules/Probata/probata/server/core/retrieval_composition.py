"""Compose injected context reads through LangGraph verification and LlamaIndex reciprocal-rank fusion.

Inputs are bounded RetrievalRequest and caller-owned async read legs; outputs are cited results and leg failures.
Effects are only those read callbacks: no index, checkpoint store, model/provider fallback or evidence writes.
Use directly or inside a Temporal Activity; Temporal retains durable retries and n8n retains visual integration.
There is no evidence adapter or raw-store handle here. An outer server must allowlist reader ports; any future
evidence reader must call the sanctioned native_evidence_search seam with server-resolved authority and audit intact.
Optional dependencies come from the retrieval extra; no framework is imported until composition is invoked.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable, Mapping
from dataclasses import dataclass
from typing import Any, Literal, TypedDict

from server.core.retrieval_contracts import (
    LegOutcome,
    RankedHit,
    ReadHit,
    RetrievalRequest,
    RetrievalResponse,
    RetrievalScope,
)


@dataclass(frozen=True)
class RetrievalLeg:
    """Inject one existing async reader; input is the request, output is bounded hits, effects belong to that reader."""

    read: Callable[[RetrievalRequest], Awaitable[list[ReadHit]]]
    supported_modes: frozenset[str] = frozenset({"keyword"})
    supported_scopes: frozenset[str] = frozenset()

    def __post_init__(self) -> None:
        """Validate declared reader capabilities without I/O; scopes must be enforced before ranking by the read callback."""
        if not self.supported_modes or not self.supported_modes <= {"keyword", "hybrid", "vector"}:
            raise ValueError("unsupported reader mode declaration")
        if not self.supported_scopes <= RetrievalScope.model_fields.keys():
            raise ValueError("unsupported reader scope declaration")


class RetrievalState(TypedDict, total=False):
    """Carry request-scoped hits/outcomes between graph nodes in memory; never checkpoint or persist source excerpts."""

    request: RetrievalRequest
    hits: dict[str, list[ReadHit]]
    outcomes: list[LegOutcome]
    items: tuple[RankedHit, ...]
    response: RetrievalResponse


async def collect_legs(request: RetrievalRequest, legs: Mapping[str, RetrievalLeg]) -> RetrievalState:
    """Read selected legs concurrently with individual deadlines and sanitized failures, without retries or writes.

    Inputs are a validated request and injected readers; output is per-leg hits/outcomes for graph verification.
    Pick this boundary to isolate one unavailable provider without silently declaring the whole query complete.
    """
    async def collect(name: str) -> tuple[str, list[ReadHit], LegOutcome]:
        """Read one named leg; return hits and a safe outcome, with only the injected callback's read effects."""
        if name not in legs:
            return name, [], LegOutcome(name=name, status="failed", error="unavailable")
        leg = legs[name]
        if request.mode not in leg.supported_modes:
            return name, [], LegOutcome(name=name, status="failed", error="unsupported_mode")
        if not request.scope.requested_fields() <= leg.supported_scopes:
            return name, [], LegOutcome(name=name, status="failed", error="unsupported_scope")
        code: Literal["timeout", "invalid_result", "unavailable"]
        try:
            async with asyncio.timeout(request.leg_timeout_seconds):
                rows = await leg.read(request)
            if not isinstance(rows, list) or len(rows) > request.per_leg_limit:
                raise ValueError("reader exceeded bounded result contract")
            hits = [ReadHit.model_validate(row) for row in rows]
            return name, hits, LegOutcome(name=name, status="success", count=len(hits))
        except TimeoutError:
            code = "timeout"
        except (ValueError, KeyError, TypeError):
            code = "invalid_result"
        except Exception:  # noqa: BLE001 -- isolate arbitrary injected reader failures without leaking provider bodies
            code = "unavailable"
        return name, [], LegOutcome(name=name, status="failed", error=code)

    results = await asyncio.gather(*(collect(name) for name in request.legs))
    return {"hits": {name: hits for name, hits, _ in results}, "outcomes": [outcome for _, _, outcome in results]}


def verify_coordinates(state: RetrievalState) -> RetrievalState:
    """Verify supplied citation coordinates and reject conflicting identities before fusion, without source lookups.

    Input is collected leg state; output retains valid legs and explicit failures. This proves coordinate preservation,
    not source authenticity. Equal text from different sources remains separate; conflicting text for one identity fails.
    """
    invalid: dict[str, Literal["invalid_result", "identity_conflict"]] = {}
    identities: dict[str, tuple[str, set[str]]] = {}
    unique: dict[str, list[ReadHit]] = {}
    for name, hits in state["hits"].items():
        seen = set()
        unique[name] = []
        for hit in hits:
            citation = hit.citation
            if not (citation.source_id or citation.source_version_ids) or not citation.locator:
                invalid[name] = "invalid_result"
            identity = citation.identity()
            if identity in identities:
                text, owners = identities[identity]
                if text != hit.text:
                    for owner in owners | {name}:
                        invalid[owner] = "identity_conflict"
                owners.add(name)
            else:
                identities[identity] = (hit.text, {name})
            if identity not in seen:
                seen.add(identity)
                unique[name].append(hit)
    outcomes = [
        LegOutcome(name=outcome.name, status="failed", error=invalid[outcome.name])
        if outcome.name in invalid else outcome.model_copy(update={"count": len(unique[outcome.name])})
        for outcome in state["outcomes"]
    ]
    return {
        "hits": {outcome.name: unique[outcome.name] if outcome.status == "success" else [] for outcome in outcomes},
        "outcomes": outcomes,
    }


def fuse_results(state: RetrievalState) -> RetrievalState:
    """Fuse verified ranked legs with QueryFusionRetriever; return scores and every contributing source citation.

    Inputs are verified hits/request; output is bounded RankedHit items. No embedding or LLM calls occur.
    Use library reciprocal-rank fusion rather than provider-score arithmetic or a second local index.
    """
    from llama_index.core.llms import MockLLM
    from llama_index.core.retrievers import BaseRetriever, QueryFusionRetriever
    from llama_index.core.retrievers.fusion_retriever import FUSION_MODES
    from llama_index.core.schema import NodeWithScore, QueryBundle, TextNode

    class ResultRetriever(BaseRetriever):
        """Expose existing verified results to LlamaIndex; output fresh nodes without querying or indexing."""

        def __init__(self, hits: list[ReadHit]) -> None:
            """Accept a verified ranked leg; retain it in memory without provider access or settings mutation."""
            super().__init__()
            self.hits = hits

        def _retrieve(self, query_bundle: QueryBundle) -> list[NodeWithScore]:
            """Return nodes for an already-executed query; source identity participates in library deduplication."""
            return [NodeWithScore(
                # TextNode.hash includes metadata. Text-only deduplication would erase distinct origins.
                node=TextNode(id_=hit.citation.identity(), text=hit.text,
                              metadata={"citation_identity": hit.citation.identity()}),
                score=hit.score,
            ) for hit in self.hits]

    originals: dict[str, ReadHit] = {}
    origins: dict[str, list[str]] = {}
    vector_targets: dict[str, dict[str, str | None]] = {}
    retrievers = []
    for name, hits in state["hits"].items():
        if not hits:
            continue
        retrievers.append(ResultRetriever(hits))
        for hit in hits:
            identity = hit.citation.identity()
            originals[identity] = hit
            origins.setdefault(identity, []).append(name)
            vector_targets.setdefault(identity, {})[name] = hit.citation.vector_name
    # QueryFusionRetriever resolves an LLM even with expansion disabled. Explicit inert MockLLM prevents
    # resolving global Settings.llm (and an implicit OpenAI credential/provider); it is never invoked.
    fusion = QueryFusionRetriever(
        retrievers, llm=MockLLM(), num_queries=1, use_async=False,
        mode=FUSION_MODES.RECIPROCAL_RANK, similarity_top_k=state["request"].limit,
    )
    ranked = fusion.retrieve(state["request"].query)
    return {"items": tuple(RankedHit(
        **originals[node.node.id_].model_dump(exclude={"score"}),
        score=node.score, legs=tuple(origins[node.node.id_]), vector_targets=vector_targets[node.node.id_],
    ) for node in ranked)}


def finish_results(state: RetrievalState) -> RetrievalState:
    """Return success, partial or failed based on explicit leg outcomes; no successful empty read becomes a failure."""
    failed = sum(outcome.status == "failed" for outcome in state["outcomes"])
    status: Literal["success", "partial", "failed"] = (
        "failed" if failed == len(state["outcomes"]) else "partial" if failed else "success"
    )
    return {"response": RetrievalResponse(
        request_id=state["request"].request_id, mode=state["request"].mode, scope=state["request"].scope, status=status,
        items=state.get("items", ()), legs=tuple(state["outcomes"]),
    )}


def route_hits(state: RetrievalState) -> str:
    """Branch on remaining candidates; return continue/finish without I/O, retries or implicit provider fallback."""
    return "continue" if any(state["hits"].values()) else "finish"


async def retrieve_context(request: RetrievalRequest, legs: Mapping[str, RetrievalLeg]) -> RetrievalResponse:
    """Execute one bounded LangGraph read/verify/fuse operation for direct callers or a Temporal Activity.

    Inputs are request and existing async reader ports; output includes original citations and partial-leg failures.
    Only reader callbacks may access providers. No checkpoint store, durable retry, server or promotion is created.
    """
    try:
        from langgraph.graph import END, START, StateGraph
        from llama_index.core.retrievers import QueryFusionRetriever  # noqa: F401 -- validate before reader I/O
    except ImportError as exc:
        raise RuntimeError("context retrieval requires the Probata retrieval extra") from exc

    async def read_node(state: RetrievalState) -> RetrievalState:
        """Read injected legs for the graph request; return collected state with per-leg read effects only."""
        return await collect_legs(state["request"], legs)

    graph = StateGraph(RetrievalState)
    graph.add_node("read", read_node)
    graph.add_node("verify", verify_coordinates)
    graph.add_node("fuse", fuse_results)
    graph.add_node("finish", finish_results)
    graph.add_edge(START, "read")
    graph.add_conditional_edges("read", route_hits, {"continue": "verify", "finish": "finish"})
    graph.add_conditional_edges("verify", route_hits, {"continue": "fuse", "finish": "finish"})
    graph.add_edge("fuse", "finish")
    graph.add_edge("finish", END)
    result = await graph.compile().ainvoke({"request": request}, config={"recursion_limit": 8})
    return result["response"]


async def retrieve_flow(payload: dict[str, Any], legs: Mapping[str, RetrievalLeg]) -> dict[str, Any]:
    """Run the declared read_retrieval envelope and return compact citations for an existing runtime/Activity adapter.

    Inputs are FlowRequest and caller-injected readers; output matches Go FlowResult without source excerpts.
    Effects are bounded reader calls only. This callable registers no API route, worker or n8n binding.
    """
    from server.core.retrieval_contracts import compact_flow_result, request_from_flow

    response = await retrieve_context(request_from_flow(payload), legs)
    return compact_flow_result(response)
