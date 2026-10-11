"""Check the real walk adapter with disposable in-memory objects and no live writes."""
import asyncio
import copy
import hashlib
import json

import pytest
from fastapi import HTTPException
from pydantic import ValidationError
from starlette.requests import Request

from app.repo import analysis_walks
from app.runtime import analysis
from app.types.analysis_walk import AnalysisWalk, WalkArtifact, WalkResult


def scope():
    """Supply a finite recorded-pin contract for isolated tests; no storage effects."""
    return dict(access_policy_id="personal-case", approved_revision_id="revision-one", approval_digest="a" * 64,
        projection_generation_id="generation-one", projection_hash="b" * 64,
        perspective="as_lived", horizon="2020-01-01T00:00:00Z", limit=25)


def req():
    """Reuse the already admitted personal actor in a local ASGI request; no auth changes."""
    value = Request({"type": "http", "method": "GET", "path": "/api/analysis"})
    value.state.subject_uid = "tailscale:owner"
    value.state.principal = "owner"
    return value


def fixture(monkeypatch, count=1):
    """Build exact immutable in-memory step/checkpoint objects without writing owner data."""
    case = dict(matter_id="matter-one", court_case_id="case-one")
    objects = {}
    reads = []
    root = "b2://salem-data/consignatio/casevault/DerivedKnowledge/analysis/walks/proof/"

    def put(name, body):
        """Pin one in-memory JSON body; input name/body, output ref, no remote effects."""
        raw = json.dumps(body).encode()
        ref = dict(uri=root + name + "?versionId=v1", version_id="v1", sha256="sha256:" + hashlib.sha256(raw).hexdigest(), bytes=len(raw))
        objects[root + name] = body
        return ref

    def read(ref, prefix):
        """Serve one exact fake object and record lazy-read coverage; no remote effects."""
        assert ref.version_id == "v1"
        assert prefix == analysis_walks.WALK_PREFIX
        reads.append(ref.uri)
        return copy.deepcopy(objects[ref.uri])

    cutoffs = [f"2020-01-{i + 1:02}T00:00:00Z" for i in range(count)]
    query_scope = {**scope(), **case, "horizon": cutoffs[-1]}
    scope_digest = analysis_walks._scope_digest("tailscale:owner", query_scope)
    inputs, refs, checkpoints = [], [], []
    for i in range(count + 1):
        perspective = "as_lived" if i < count else "hindsight"
        step = dict(schema="approved-walk-step-v1", perspective=perspective,
            horizon=cutoffs[i] if i < count else None, scope_digest=scope_digest, step_index=i, claims=[])
        if 0 < i < count:
            step["previous"] = {k: v for k, v in checkpoints[-1].items() if k in analysis_walks.CHECKPOINT_FIELDS}
        input_ref = put(f"input-{i}.json", step)
        checkpoint = dict(input_artifact=input_ref, schema="approved-walk-checkpoint-v1", perspective=perspective,
            horizon=step["horizon"], scope_digest=scope_digest, step_index=i,
            input_digest=analysis_walks._digest(step), model_input_digest="e" * 64, claim_ids=[], findings=[],
            model={"model_id": "test-model"}, prompt_version="approved-walk-balanced-v1", prompt_sha256="f" * 64,
            usage={}, review_status="pending", limitations=[])
        refs.append(dict(artifact=put(f"checkpoint-{i}.json", checkpoint)))
        inputs.append(dict(artifact=input_ref, has_more=False, claim_count=0))
        checkpoints.append(checkpoint)
    final = dict(schema="approved-walk-result-v1", query=dict(operating_mode="LIVE", actor=dict(subject_uid="tailscale:owner"), request_id="proof", scope=query_scope),
        cutoffs=cutoffs, scope_digest=scope_digest, checkpoints=refs, inputs=inputs, partial=False, review_status="pending",
        comparison=dict(hindsight_only_claim_ids=[], available_after_last_cutoff_claim_ids=[], unknown_availability_claim_ids=[], hindsight_findings_not_identical_to_last_as_lived=[]),
        coverage="Selected cutoff pages only.", validation="Exact source citations.", family_independence="Unresolved.")
    result = WalkResult(artifact=put("result.json", final), checkpoints=refs, partial=False, review_status="pending")
    monkeypatch.setattr(analysis_walks, "read_verified_artifact", read)
    return case, result, final, objects, reads


@pytest.mark.parametrize("cutoffs", [[], ["2020-01-01T00:00:00"], ["2020-01-02T00:00:00Z", "2020-01-01T00:00:00Z"], ["2019-01-01T00:00:00Z"]])
def test_walk_requires_explicit_ordered_aware_dates(cutoffs):
    """Match Go's request rules without silently sorting or inventing dates."""
    with pytest.raises(ValidationError):
        AnalysisWalk(scope=scope(), cutoffs=cutoffs)


def test_real_artifact_shape_keeps_version_query_and_prefixed_hash():
    """Normalize only the reader's format; retain actual Go references in response models."""
    ref = WalkArtifact(uri="b2://salem-data/consignatio/casevault/DerivedKnowledge/analysis/walks/run/a.json?versionId=v%2F1", version_id="v/1", sha256="sha256:" + "a" * 64, bytes=2)
    normalized = ref.reader_reference()
    assert normalized.version_id == "v/1" and normalized.sha256 == "a" * 64
    assert "?" not in normalized.uri and ref.uri.endswith("v%2F1")
    for uri in (ref.uri.replace("walks/", "queries/"), ref.uri.replace("v%2F1", "other"), ref.uri.replace("salem-data", "other")):
        with pytest.raises(ValueError):
            ref.model_copy(update={"uri": uri}).reader_reference()


def test_summary_does_not_preload_twenty_one_checkpoints(monkeypatch):
    """A twenty-cutoff summary reads only its finite final document."""
    case, result, final, objects, reads = fixture(monkeypatch, 20)
    value = analysis_walks.read_walk_content("approved-context-walk:proof", result, "tailscale:owner", case)
    assert len(reads) == 1 and len(value["checkpoints"]) == 21


@pytest.mark.parametrize("index, expected_reads", [(0, 3), (1, 4), (2, 3)])
def test_selected_step_reads_only_input_checkpoint_and_one_predecessor(monkeypatch, index, expected_reads):
    """Keep historical chain checks bounded and start hindsight without a prior state."""
    case, result, final, objects, reads = fixture(monkeypatch, 2)
    value = analysis_walks.read_walk_content("approved-context-walk:proof", result, "tailscale:owner", case, index)
    assert len(reads) == expected_reads
    assert value["checkpoint"]["step_index"] == index
    assert value["checkpoint"]["perspective"] == ("hindsight" if index == 2 else "as_lived")


@pytest.mark.parametrize("violation", ["actor", "case", "execution", "chain", "hindsight-seed", "input-hash", "cutoff", "checkpoint-ref"])
def test_scope_chain_and_exact_input_mismatches_fail_closed(monkeypatch, violation):
    """Reject cross-actor/case/execution, changed input, forged chain and leaked hindsight."""
    case, result, final, objects, reads = fixture(monkeypatch, 2)
    index = 1
    root = next(key.rsplit("/", 1)[0] for key in objects)
    if violation == "actor": final["query"]["actor"]["subject_uid"] = "other"
    elif violation == "case": final["query"]["scope"]["matter_id"] = "other"
    elif violation == "execution": final["query"]["request_id"] = "other"
    elif violation == "chain": objects[root + "/input-1.json"]["previous"]["findings"] = [{"changed": True}]
    elif violation == "hindsight-seed":
        index = 2
        objects[root + "/input-2.json"]["previous"] = objects[root + "/checkpoint-1.json"]
    elif violation == "input-hash": objects[root + "/checkpoint-1.json"]["input_digest"] = "0" * 64
    elif violation == "cutoff": objects[root + "/checkpoint-1.json"]["horizon"] = "2021-01-01T00:00:00Z"
    else: final["checkpoints"][0]["artifact"]["version_id"] = "different"
    with pytest.raises(analysis_walks.ProfferError):
        analysis_walks.read_walk_content("approved-context-walk:proof", result, "tailscale:owner", case, index)


def test_start_forwards_only_existing_identity_case_and_stable_key(monkeypatch):
    """Use the existing personal-tailnet seam with the actual Go route; no new role gate."""
    calls = []
    async def upstream(method, path, **kwargs):
        """Capture one isolated start request; output dummy transport, no remote effects."""
        calls.append((method, path, kwargs))
        return object()
    monkeypatch.setattr(analysis, "_case", lambda: dict(matter_id="matter-one", court_case_id="case-one"))
    monkeypatch.setattr(analysis.proffer, "_request", upstream)
    monkeypatch.setattr(analysis.proffer, "_json_payload", lambda *_: dict(workflow_id="approved-context-walk:proof", run_id="run"))
    body = AnalysisWalk(scope=scope(), cutoffs=["2020-01-01T00:00:00Z"])
    value = asyncio.run(analysis.start_analysis_walk(body, req(), "same-click"))
    assert value["run_id"] == "run"
    assert calls[0][:2] == ("POST", "/reference-import/analysis/walks")
    assert calls[0][2]["headers"]["Idempotency-Key"] == "same-click"
    assert calls[0][2]["headers"]["X-authentik-uid"] == "tailscale:owner"
    assert calls[0][2]["json"]["scope"]["matter_id"] == "matter-one"


def test_unfinished_walk_does_not_read_artifacts(monkeypatch):
    """Prevent queued/running execution from being presented as completed reasoning."""
    async def status(*_):
        """Return real-shaped running metadata without remote effects."""
        return dict(outcome="running")
    monkeypatch.setattr(analysis, "_walk_status", status)
    with pytest.raises(HTTPException) as error:
        asyncio.run(analysis.analysis_walk_content("approved-context-walk:proof", req()))
    assert error.value.status_code == 409


@pytest.mark.parametrize("violation", [None, "future", "unknown-clock", "revision", "citation-pin", "quote"])
def test_selected_native_citations_keep_exact_spans_and_cutoff(monkeypatch, violation):
    """Retain a native span at zero and reject temporal/source/citation mismatches."""
    case, result, final, objects, reads = fixture(monkeypatch)
    root = next(key.rsplit("/", 1)[0] for key in objects)
    claim = dict(id="claim-one", kind="statement", text="Original attributed statement", **case,
        approved_revision_id="revision-one", approval_digest="a" * 64, control_generation_id="generation-one",
        candidate_id="candidate-one", candidate_sha256="c" * 64,
        source_id="019c7714-3b77-74d1-9866-e1f484aae2ab", source_version_id="019c6e27-e55b-73d1-87d8-4e01f1f75043",
        source_object_id="", source_object_uri="b2://salem-data/consignatio/casevault/Original/chat.json",
        source_sha256="d" * 64, record_id="native-span", record_sha256="e" * 64,
        native_json_pointer="/original/text", native_span_start=0, native_span_end=29,
        native_span_unit="unicode_codepoint", native_span_sha256="f" * 64,
        source_available_from="2019-01-01T00:00:00Z", approved_at="2026-01-01T00:00:00Z", approved_by="tailscale:owner")
    step = objects[root + "/input-0.json"]
    checkpoint = objects[root + "/checkpoint-0.json"]
    if violation == "future": claim["source_available_from"] = "2021-01-01T00:00:00Z"
    elif violation == "unknown-clock": claim["source_available_from"] = None
    elif violation == "revision": claim["approved_revision_id"] = "other"
    step["claims"] = [claim]
    final["inputs"][0]["claim_count"] = 1
    checkpoint["claim_ids"] = [claim["id"]]
    checkpoint["input_digest"] = analysis_walks._digest(step)
    citation = dict(claim_id=claim["id"], quote="attributed statement", source_pin=copy.deepcopy(claim))
    if violation == "citation-pin": citation["source_pin"]["native_span_start"] = 1
    elif violation == "quote": citation["quote"] = "Invented text"
    checkpoint["findings"] = [dict(kind="uncertainty", text="An attributed account.", citations=[citation])]
    if violation:
        with pytest.raises(analysis_walks.ProfferError):
            analysis_walks.read_walk_content("approved-context-walk:proof", result, "tailscale:owner", case, 0)
    else:
        actual = analysis_walks.read_walk_content("approved-context-walk:proof", result, "tailscale:owner", case, 0)
        pin = actual["checkpoint"]["findings"][0]["citations"][0]["source_pin"]
        assert pin["source_object_id"] == "" and pin["native_span_start"] == 0
        assert pin["native_span_sha256"] == "f" * 64
