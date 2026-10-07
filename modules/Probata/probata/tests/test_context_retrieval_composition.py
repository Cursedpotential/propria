"""Exercise real LlamaIndex/LangGraph context composition on synthetic, network-free reader fixtures.

Inputs are fabricated citations and reader failures; assertions cover identity, branching and compact boundaries.
No real case, provider, database or index is touched. Run with the optional retrieval extra installed.
Byline: Codex · GPT-6.1-Sol · 2026-10-06.
"""

from __future__ import annotations

import asyncio
import json
import shutil
import subprocess
from pathlib import Path

import pytest

from server.core.retrieval_adapters import intake_hits, proffer_hits
from server.core.retrieval_composition import RetrievalLeg, retrieve_context, retrieve_flow
from server.core.retrieval_contracts import (
    Citation,
    ReadHit,
    RetrievalRequest,
    RetrievalResponse,
    compact_flow_result,
    request_from_flow,
)


def hit(object_id: str, *, source: str = "source-a", text: str = "same excerpt", score: float = 1) -> ReadHit:
    """Create a synthetic cited hit; output is in-memory only and never represents genuine case material."""
    return ReadHit(text=text, score=score, citation=Citation(
        collection="Synthetic", object_id=object_id, source_id=source, chunk_id=object_id,
        locator={"source_path": f"synthetic/{source}"},
    ))


def reader(*hits: ReadHit) -> RetrievalLeg:
    """Wrap synthetic ranked hits as an async read port without I/O; use for real-library composition tests."""
    async def read(request: RetrievalRequest) -> list[ReadHit]:
        """Accept a bounded request and return fixture hits without external effects."""
        return list(hits)
    return RetrievalLeg(read)


def run(legs: dict[str, RetrievalLeg], **bounds) -> RetrievalResponse:
    """Invoke the real graph synchronously for a fixture; return its response with no service or store access."""
    request = RetrievalRequest(request_id="synthetic-request", query="same", legs=tuple(legs), **bounds)
    return asyncio.run(retrieve_context(request, legs))


def test_equal_text_keeps_distinct_sources_and_collections(monkeypatch):
    """Fuse equal text with distinct provenance; assert no citation is erased and no LLM is called."""
    from llama_index.core.llms import MockLLM

    def forbidden(*args, **kwargs):
        """Reject any model call during fusion; this fixture has no provider effects."""
        raise AssertionError("query expansion/model invocation is forbidden")

    monkeypatch.setattr(MockLLM, "complete", forbidden)
    monkeypatch.setattr(MockLLM, "acomplete", forbidden)
    first = hit("object-1")
    second = hit("object-1", source="source-b")
    third = first.model_copy(update={"citation": first.citation.model_copy(update={"collection": "OtherSynthetic"})})
    result = run({"keyword": reader(first, second), "semantic": reader(third)})
    assert result.status == "success"
    assert len(result.items) == 3
    assert {item.citation.identity() for item in result.items} == {
        item.citation.identity() for item in (first, second, third)
    }
    assert all(item.text == first.text for item in result.items)


def test_shared_identity_fuses_legs_without_duplicate_weight_in_one_leg():
    """Fuse shared citations across legs; one repeated object within a leg contributes only once, without custom ranks."""
    shared, other = hit("shared", score=5), hit("other", score=4)
    result = run({"keyword": reader(shared, shared, other), "semantic": reader(shared)})
    assert len(result.items) == 2
    assert result.items[0].citation.object_id == "shared"
    assert result.items[0].legs == ("keyword", "semantic")
    assert result.legs[0].count == 2
    baseline = run({"keyword": reader(shared, other), "semantic": reader(shared)})
    assert result.items == baseline.items


def test_keyword_and_named_vector_share_source_but_retain_leg_targets():
    """Fuse one object across keyword/semantic legs; vector choices survive separately without multiplying source identity."""
    keyword = hit("shared")
    semantic = keyword.model_copy(update={"citation": keyword.citation.model_copy(update={"vector_name": "text_nim"})})
    result = run({"keyword": reader(keyword), "semantic": reader(semantic)})
    assert len(result.items) == 1
    assert result.items[0].vector_targets == {"keyword": None, "semantic": "text_nim"}


def test_partial_leg_failure_is_visible_and_provider_error_is_redacted():
    """Fail one injected leg; retain successful citations and expose a safe failure code without the exception body."""
    async def unavailable(request):
        """Raise a fabricated provider error without contacting any provider."""
        raise RuntimeError("secret-provider-body synthetic corpus excerpt")

    result = run({"keyword": reader(hit("retained")), "semantic": RetrievalLeg(unavailable)})
    assert result.status == "partial"
    assert len(result.items) == 1
    assert result.legs[1].error == "unavailable"
    assert "secret-provider-body" not in result.model_dump_json()
    compact = compact_flow_result(result)
    assert compact["status"] == "success"
    assert compact["outputs"]["status"] == "partial"
    assert "text" not in compact["outputs"]["items"][0]
    assert compact["outputs"]["items"][0]["citation"]["object_id"] == "retained"


def test_deadline_cancels_only_slow_leg():
    """Bound an indefinitely waiting synthetic leg; verify cancellation, partial status and successful neighbor results."""
    cancelled = []

    async def waiting(request):
        """Wait on an unset event and observe deadline cancellation; no external work occurs."""
        try:
            await asyncio.Event().wait()
        finally:
            cancelled.append(True)

    result = run({"fast": reader(hit("fast")), "slow": RetrievalLeg(waiting)}, leg_timeout_seconds=0.01)
    assert result.status == "partial"
    assert result.legs[1].error == "timeout"
    assert cancelled == [True]


def test_scope_and_mode_are_never_silently_broadened():
    """Reject unsupported scoped/hybrid reads before their callbacks; a declared capable leg receives exact coordinates."""
    observed = []

    async def scoped(request):
        """Observe authorized synthetic request coordinates and return one fixture without external effects."""
        observed.append(request)
        return [hit("scoped")]

    request = RetrievalRequest(request_id="synthetic", query="example", mode="hybrid",
                               scope={"matter_id": "synthetic-matter", "source_version_ids": ["version-1"]},
                               legs=("keyword_only", "global", "scoped"))
    result = asyncio.run(retrieve_context(request, {
        "keyword_only": RetrievalLeg(scoped),
        "global": RetrievalLeg(scoped, supported_modes=frozenset({"hybrid"})),
        "scoped": RetrievalLeg(scoped, supported_modes=frozenset({"hybrid"}),
                               supported_scopes=frozenset({"matter_id", "source_version_ids"})),
    }))
    assert observed == [request]
    assert result.scope == request.scope
    assert result.mode == "hybrid"
    assert [leg.error for leg in result.legs] == ["unsupported_mode", "unsupported_scope", None]
    assert result.status == "partial"


@pytest.mark.parametrize("mode", ["unknown", "empty"])
def test_graph_skips_fusion_when_no_candidates(mode, monkeypatch):
    """Exercise the finish branch for unavailable/empty legs; no library fusion or fallback is invoked."""
    import server.core.retrieval_composition as composition

    def forbidden(state):
        """Reject fusion on the graph's no-candidate branch without side effects."""
        raise AssertionError("fusion must not run")

    monkeypatch.setattr(composition, "fuse_results", forbidden)
    request = RetrievalRequest(request_id="synthetic", query="nothing", legs=("one",))
    result = asyncio.run(retrieve_context(request, {} if mode == "unknown" else {"one": reader()}))
    assert result.status == ("failed" if mode == "unknown" else "success")
    assert result.items == ()


def test_verify_rejects_conflicting_text_for_same_identity():
    """Reject both conflicting origins for one exact identity; retain an independent valid leg without choosing a winner."""
    result = run({"first": reader(hit("conflict", text="first")),
                  "second": reader(hit("conflict", text="second")), "third": reader(hit("valid"))})
    assert result.status == "partial"
    assert [outcome.error for outcome in result.legs] == ["identity_conflict", "identity_conflict", None]
    assert [item.citation.object_id for item in result.items] == ["valid"]


def test_verify_rejects_missing_source_locator_and_over_limit():
    """Fail malformed citation and over-limit legs explicitly; a bounded valid neighbor remains readable."""
    bad = ReadHit(text="unmapped", score=1, citation=Citation(collection="Synthetic", object_id="unmapped"))
    result = run({"unmapped": reader(bad), "oversized": reader(hit("a"), hit("b")),
                  "valid": reader(hit("valid"))}, per_leg_limit=1)
    assert result.status == "partial"
    assert [outcome.error for outcome in result.legs] == ["invalid_result", "invalid_result", None]


def test_intake_adapter_preserves_existing_vault_coordinates():
    """Adapt a synthetic existing Intake envelope; all source IDs and supplied locator fields survive without inferred hashes."""
    payload = {"collection": "SyntheticIntake", "target_vector": "text_vector", "hits": [{
        "object_id": "obj", "source_id": "src", "document_id": "doc", "chunk_id": "chunk",
        "source_path": "synthetic/path", "vault_key": "synthetic/vault-key", "resolution": "catalog",
        "text": "example", "score": 0.7,
    }]}
    adapted = intake_hits(payload)[0]
    assert adapted.citation.locator["vault_key"] == "synthetic/vault-key"
    assert adapted.citation.source_id == "src"
    assert adapted.citation.source_version_ids == ()
    assert adapted.citation.source_sha256 is None
    assert adapted.citation.vector_name == "text_vector"


def test_proffer_adapter_keeps_versions_message_and_real_object_id():
    """Adapt a synthetic Proffer row; preserve every returned version and message locator, refusing fabricated object IDs."""
    row = {"text": "example", "source_version_ids": ["version-1", "version-2"],
           "source_version_id": "version-3", "first_message_id": "message-1", "_additional": {"id": "obj", "score": 2}}
    adapted = proffer_hits("SyntheticProffer", [row], vector_name="text_nim")[0]
    assert adapted.citation.source_version_ids == ("version-1", "version-2", "version-3")
    assert adapted.citation.first_message_id == "message-1"
    assert adapted.citation.object_id == "obj"
    with pytest.raises(KeyError):
        proffer_hits("SyntheticProffer", [{**row, "_additional": {"score": 2}}])


def test_named_flow_contract_accepts_scalar_knobs_and_rejects_payloads():
    """Validate the wrapper's actual compact input; reject unknown fields, duplicate legs and source-body payloads."""
    payload = {"flow": "read_retrieval", "request_id": "synthetic", "inputs": {"query": "example", "leg_names": "intake,proffer"}}
    assert request_from_flow(payload).legs == ("intake", "proffer")
    scoped = {**payload, "inputs": {**payload["inputs"], "mode": "hybrid",
                                   "scope_json": json.dumps({"source_version_ids": ["exact-version"]})}}
    assert request_from_flow(scoped).scope.source_version_ids == ("exact-version",)
    for invalid in (
        {**payload, "url": "https://untrusted.invalid"},
        {**payload, "inputs": {**payload["inputs"], "content": "not allowed"}},
        {**payload, "inputs": {**payload["inputs"], "disclosure_tiers": "hindsight"}},
        {**payload, "inputs": {"query": "example", "leg_names": "intake,intake"}},
    ):
        with pytest.raises(ValueError):
            request_from_flow(invalid)
    with pytest.raises(ValueError):
        RetrievalRequest.model_validate({"request_id": "synthetic", "query": "example", "legs": ["intake"],
                                         "scope": {"horizon": "2099-01-01", "disclosure_tiers": ["hindsight"]}})


def test_actual_flow_callable_returns_compact_read_results():
    """Invoke the future wrapper target directly with synthetic readers; preserve request identity without excerpt payloads."""
    payload = {"flow": "read_retrieval", "request_id": "synthetic", "inputs": {"query": "example", "leg_names": "intake"}}
    result = asyncio.run(retrieve_flow(payload, {"intake": reader(hit("object"))}))
    assert result["outputs"]["request_id"] == "synthetic"
    assert "same excerpt" not in json.dumps(result)
    assert result["outputs"]["items"][0]["citation"]["object_id"] == "object"


def test_compact_flow_result_refuses_oversized_citation_history():
    """Exceed the existing Go caller's 64-KiB envelope bound with fabricated locators; fail explicitly without writes."""
    large = hit("large")
    large = large.model_copy(update={"citation": large.citation.model_copy(update={
        "locator": {str(i): "x" * 2048 for i in range(16)},
    })})
    result = run({"one": reader(large, large.model_copy(update={"citation": large.citation.model_copy(update={"object_id": "two"})}))})
    with pytest.raises(ValueError, match="response bound"):
        compact_flow_result(result)


def test_n8n_export_is_explicitly_unbound_and_preserves_compact_contract():
    """Inspect source-only n8n wiring without executing it; assert auth, disabled activation and explicit missing endpoint."""
    root = Path(__file__).resolve().parents[1]
    workflow = json.loads((root / "deploy/docker/n8n/workflows/retrieval/wf-read-retrieval.json").read_text())
    nodes = {node["name"]: node for node in workflow["nodes"]}
    assert workflow["active"] is False
    assert nodes["Read retrieval webhook"]["parameters"]["authentication"] == "headerAuth"
    assert nodes["UNBOUND read composition HTTP"]["parameters"]["url"] == ""
    assert "No API route or worker binding" in nodes["Binding seam and contract"]["parameters"]["content"]
    assert "Idempotency-Key" in json.dumps(nodes["UNBOUND read composition HTTP"])
    assert "read_retrieval" in nodes["Validate flow request"]["parameters"]["jsCode"]


def test_n8n_code_nodes_execute_contract_and_reject_hidden_partial_failure():
    """Execute exported Code-node validators on synthetic envelopes in Node; no browser, provider or n8n execution occurs."""
    node = shutil.which("node")
    if node is None:
        pytest.skip("Node required for exported n8n Code-node validation")
    root = Path(__file__).resolve().parents[1]
    workflow = json.loads((root / "deploy/docker/n8n/workflows/retrieval/wf-read-retrieval.json").read_text())
    code = {n["name"]: n["parameters"].get("jsCode") for n in workflow["nodes"]}
    request = {"flow": "read_retrieval", "request_id": "synthetic", "inputs": {"query": "example", "leg_names": "one"}}
    response = asyncio.run(retrieve_flow(request, {"one": reader(hit("object"))}))
    script = """
const fixture = JSON.parse(require('fs').readFileSync(0, 'utf8'));
const execute = (code, body) => new Function('$input', '$', code)(
  {first:()=>({json:body})}, ()=>({first:()=>({json:fixture.request})}));
execute(fixture.requestCode, {body:fixture.request});
execute(fixture.responseCode, fixture.response);
for (const bad of [
  {...fixture.request, inputs:{query:'example',leg_names:'one,one'}},
  {...fixture.request, inputs:{query:'example',leg_names:'one',limit:'20'}}
]) { let rejected=false; try { execute(fixture.requestCode, {body:bad}); } catch { rejected=true; }
  if(!rejected) throw new Error('invalid request accepted'); }
const hidden=structuredClone(fixture.response);
hidden.outputs.legs[0]={name:'one',status:'failed',count:0,error:'unavailable'};
let rejected=false; try {execute(fixture.responseCode, hidden);} catch {rejected=true;}
if(!rejected) throw new Error('hidden partial failure accepted');
"""
    completed = subprocess.run([node, "-e", script], input=json.dumps({
        "request": request, "response": response, "requestCode": code["Validate flow request"],
        "responseCode": code["Validate compact result"],
    }), capture_output=True, text=True, timeout=10, check=False)
    assert completed.returncode == 0, completed.stderr
