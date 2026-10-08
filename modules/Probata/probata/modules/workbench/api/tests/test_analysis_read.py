"""Verify query scope, historical cutoff and exact artifact reads without live writes."""
import asyncio
import hashlib
import io
import json
from datetime import datetime, timezone

import pytest
from fastapi import HTTPException
from pydantic import ValidationError
from starlette.requests import Request

from app.repo import analysis_artifacts
from app.runtime import analysis
from app.service.proffer_errors import ProfferError
from app.types.analysis import AnalysisArtifact, AnalysisQuery


def pins():
    """Return bounded in-memory contract pins; never write them to a live store."""
    return dict(access_policy_id="personal-case", approved_revision_id="revision-one", approval_digest="a" * 64,
        projection_generation_id="generation-one", projection_hash="b" * 64)


def request():
    """Represent the already admitted tailnet owner for isolated route tests."""
    result = Request({"type": "http", "method": "GET", "path": "/api/analysis"})
    result.state.subject_uid = "tailscale:owner"
    result.state.principal = "owner"
    return result


def artifact(raw):
    """Create metadata for exact in-memory bytes, without object-store writes."""
    return AnalysisArtifact(uri="b2://salem-data/consignatio/casevault/DerivedKnowledge/analysis/queries/run/approved-context/one.json",
        version_id="provider-version", sha256=hashlib.sha256(raw).hexdigest(), bytes=len(raw))


def test_as_lived_cutoff_required_and_timezone_preserved():
    """Reject missing/naive historical cutoffs; hindsight must not inherit one."""
    for horizon in (None, "2020-01-01T12:00:00"):
        with pytest.raises(ValidationError):
            AnalysisQuery(**pins(), perspective="as_lived", horizon=horizon)
    query = AnalysisQuery(**pins(), perspective="as_lived", horizon="2020-01-01T12:00:00-05:00")
    assert query.horizon.astimezone(timezone.utc).hour == 17
    with pytest.raises(ValidationError):
        AnalysisQuery(**pins(), perspective="hindsight", horizon="2020-01-01T12:00:00Z")
    assert AnalysisQuery(**pins(), perspective="hindsight").horizon is None


def test_query_start_uses_existing_case_and_stable_click_key(monkeypatch):
    """Preserve actual query pins without taking a case or extra role from the browser."""
    calls = []

    async def upstream(method, path, **kwargs):
        calls.append((method, path, kwargs))
        return object()

    monkeypatch.setattr(analysis, "_case", lambda: dict(matter_id="matter-one", court_case_id="case-one"))
    monkeypatch.setattr(analysis.proffer, "_request", upstream)
    monkeypatch.setattr(analysis.proffer, "_json_payload", lambda *_: dict(workflow_id="approved-context-query:one", run_id="run-one"))
    query = AnalysisQuery(**pins(), perspective="as_lived", horizon="2020-01-01T12:00:00Z")
    result = asyncio.run(analysis.start_analysis_query(query, request(), "same-click"))
    assert result["workflow_id"] == "approved-context-query:one"
    assert calls[0][0:2] == ("POST", "/reference-import/analysis/queries")
    assert calls[0][2]["headers"]["Idempotency-Key"] == "same-click"
    assert calls[0][2]["headers"]["X-authentik-uid"] == "tailscale:owner"
    assert calls[0][2]["json"]["scope"]["projection_hash"] == "b" * 64
    assert calls[0][2]["json"]["scope"]["matter_id"] == "matter-one"


@pytest.mark.parametrize("changed", ["version", "length", "hash", "oversize"])
def test_artifact_version_length_hash_and_bound_fail_closed(monkeypatch, changed):
    """Close streams and reject changed or oversized bytes before rendering JSON."""
    raw = b'{"claims":[]}'
    ref = artifact(raw)
    stream = io.BytesIO(raw + (b"extra" if changed == "oversize" else b""))
    if changed == "hash":
        ref.sha256 = "c" * 64

    class Store:
        """Return one in-memory object response while capturing exact provider-version use."""
        def get_object(self, **kwargs):
            assert kwargs["VersionId"] == "provider-version"
            return dict(Body=stream, VersionId="other" if changed == "version" else ref.version_id,
                ContentLength=ref.bytes + (1 if changed == "length" else 0))

    monkeypatch.setattr(analysis_artifacts, "get_store_client", lambda _: Store())
    with pytest.raises(ProfferError):
        analysis_artifacts.read_query_artifact(ref)
    assert stream.closed


def test_exact_artifact_reads_only_query_prefix(monkeypatch):
    """Read exact valid bytes and reject source objects or caller-selected endpoints."""
    raw = b'{"claims":[]}'
    ref = artifact(raw)

    class Store:
        """Serve exact query bytes only for the intended bucket and version."""
        def get_object(self, **kwargs):
            assert kwargs["Bucket"] == "salem-data"
            assert kwargs["VersionId"] == ref.version_id
            return dict(Body=io.BytesIO(raw), VersionId=ref.version_id, ContentLength=len(raw))

    monkeypatch.setattr(analysis_artifacts, "get_store_client", lambda _: Store())
    assert analysis_artifacts.read_query_artifact(ref) == {"claims": []}
    for uri in ("https://example.com/one.json", "b2://other/consignatio/casevault/DerivedKnowledge/analysis/queries/one.json",
        "b2://salem-data/consignatio/casevault/Original/one.json"):
        ref.uri = uri
        with pytest.raises(ProfferError):
            analysis_artifacts.read_query_artifact(ref)


@pytest.mark.parametrize("violation", [None, "actor", "case", "revision", "future", "unknown-clock"])
def test_findings_validate_actor_scope_and_as_lived_source_clock(monkeypatch, violation):
    """Keep mismatched scope and unverified precise source timing out of cutoff reads."""
    case = dict(matter_id="matter-one", court_case_id="case-one")
    content = {**pins(), **case, "actor_subject_uid": "tailscale:owner", "perspective": "as_lived",
        "horizon": "2020-01-01T12:00:00Z", "limit": 25, "has_more": False, "claims": [{
            "id": "claim-one", "kind": "statement", "text": "Recorded finding", **case,
            "approved_revision_id": "revision-one", "approval_digest": "a" * 64, "control_generation_id": "generation-one",
            "candidate_id": "candidate-one", "candidate_sha256": "c" * 64, "source_id": "source-one",
            "source_version_id": "source-version-one", "source_object_id": "source-object-one",
            "source_object_uri": "b2://salem-data/consignatio/casevault/Original/one.json", "source_sha256": "d" * 64,
            "record_id": "record-one", "record_sha256": "e" * 64, "source_available_from": "2019-01-01T12:00:00Z",
            "predicate": "recorded_statement", "native_json_pointer": "/messages/0/text",
            "native_span_start": 0, "native_span_end": 5, "native_span_unit": "unicode_codepoint",
            "native_span_sha256": "f" * 64,
            "approved_at": "2026-01-01T12:00:00Z", "approved_by": "tailscale:owner",
        }]}
    result = {**pins(), **case, "perspective": "as_lived", "claim_count": 1, "has_more": False,
        "artifact": artifact(json.dumps(content).encode()).model_dump()}
    if violation == "actor":
        content["actor_subject_uid"] = "other"
    elif violation == "case":
        content["claims"][0]["court_case_id"] = "other"
    elif violation == "revision":
        content["claims"][0]["approved_revision_id"] = "other"
    elif violation == "future":
        content["claims"][0]["source_available_from"] = "2021-01-01T12:00:00Z"
    elif violation == "unknown-clock":
        content["claims"][0]["source_available_from"] = None

    async def status(*_):
        return dict(workflow_id="approved-context-query:one", outcome="completed", result=result)

    monkeypatch.setattr(analysis, "_status", status)
    monkeypatch.setattr(analysis, "read_query_artifact", lambda _: content)
    if violation:
        with pytest.raises(HTTPException) as error:
            asyncio.run(analysis.analysis_query_content("approved-context-query:one", request()))
        assert error.value.status_code == 502
    else:
        got = asyncio.run(analysis.analysis_query_content("approved-context-query:one", request()))
        assert got.claims[0].text == "Recorded finding"
        assert got.claims[0].record_id == "record-one"
        assert got.claims[0].predicate == "recorded_statement"
        assert got.claims[0].native_json_pointer == "/messages/0/text"
        assert got.claims[0].native_span_start == 0
        assert got.claims[0].native_span_end == 5
        assert got.claims[0].native_span_unit == "unicode_codepoint"
        assert got.claims[0].native_span_sha256 == "f" * 64
