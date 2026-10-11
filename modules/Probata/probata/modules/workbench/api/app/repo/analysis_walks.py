"""Resolve a completed walk into verified historical steps and separate hindsight.

Inputs: workflow-owned exact references, actor and personal case. Outputs: cited
checkpoint documents. Effects: bounded B2 reads only; reuse for Read, not ingestion.
"""
import hashlib
import json

from pydantic import AwareDatetime, TypeAdapter, ValidationError

from app.repo.analysis_artifacts import read_verified_artifact
from app.service.proffer_errors import ProfferError
from app.types.analysis import AnalysisClaim
from app.types.analysis_walk import AnalysisWalk, WALK_PREFIX, WalkArtifact, WalkResult

MAX_TOTAL_BYTES = 12 * 1024 * 1024
CHECKPOINT_FIELDS = {"input_artifact", "schema", "perspective", "horizon", "scope_digest", "step_index",
    "input_digest", "claim_ids", "findings", "model", "prompt_version", "prompt_sha256", "usage", "review_status", "limitations"}
FINDING_KINDS = {"supporting", "contrary", "positive", "missing_context", "alternative", "uncertainty"}


def _require(condition: bool, detail: str):
    """Stop a mismatched result before rendering; input condition/detail, output none, no effects."""
    if not condition:
        raise ProfferError(detail, 502)


def _date(value):
    """Parse only supplied aware dates; input ISO date/null, output datetime/null, no effects."""
    return TypeAdapter(AwareDatetime).validate_python(value) if value is not None else None


def _digest(value):
    """Hash the exact Python canonical step contract; input JSON, output SHA-256, no effects."""
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()


def _findings(findings, claims):
    """Resolve every quotation to the exact supplied eligible claim and root citation.

    Inputs: bounded findings and ID-indexed claims. Output: unchanged findings.
    Effects: none; choose before displaying reasoning or reusing earlier state.
    """
    _require(isinstance(findings, list) and len(findings) <= 16, "Walk finding count is invalid")
    for finding in findings:
        _require(isinstance(finding, dict) and finding.get("kind") in FINDING_KINDS
            and isinstance(finding.get("text"), str) and 1 <= len(finding["text"]) <= 1000,
            "Walk finding is invalid")
        citations = finding.get("citations")
        _require(isinstance(citations, list) and 1 <= len(citations) <= 8, "Walk citation count is invalid")
        for cite in citations:
            _require(isinstance(cite, dict), "Walk citation is invalid")
            claim = claims.get(cite.get("claim_id"))
            quote = cite.get("quote")
            _require(claim is not None and cite.get("source_pin") == claim
                and isinstance(quote, str) and bool(quote.strip()) and quote in claim["text"],
                "Walk citation no longer resolves to its exact source")
    return findings


def read_walk_content(workflow_id: str, result: WalkResult, actor: str, case: dict[str, str], index: int | None = None) -> dict:
    """Read the final walk document or one lazily selected validated checkpoint.

    Inputs: completed status, server identity/case. Output: full final document
    or one dereferenced step. Effects: exact B2 reads, capped at 12 MiB combined;
    no graph/model calls, storage credentials or new owner-authorization step.
    """
    total = 0
    cached = {}

    def read(value):
        """Read a unique exact reference once within the combined byte budget."""
        nonlocal total
        ref = WalkArtifact.model_validate(value)
        identity = ref.model_dump_json()
        if identity not in cached:
            total += ref.bytes
            _require(total <= MAX_TOTAL_BYTES, "Walk display exceeds its combined result byte budget")
            cached[identity] = read_verified_artifact(ref.reader_reference(), WALK_PREFIX)
        return cached[identity]

    try:
        _require(result.artifact.bytes <= 1024 * 1024, "Walk summary exceeds its display byte budget")
        final = read(result.artifact)
        query = final.get("query", {})
        scope = query.get("scope", {})
        _require(isinstance(query, dict) and isinstance(scope, dict), "Walk query binding is invalid")
        _require(query.get("actor", {}).get("subject_uid") == actor
            and query.get("operating_mode") == "LIVE"
            and workflow_id == "approved-context-walk:" + query.get("request_id", "")
            and all(scope.get(k) == v for k, v in case.items()), "Walk returned a different actor, case or execution")
        contract = AnalysisWalk.model_validate({"scope": {k: v for k, v in scope.items() if k not in case}, "cutoffs": final.get("cutoffs")})
        refs = [entry.model_dump(mode="json") for entry in result.checkpoints]
        inputs = final.get("inputs")
        digest = final.get("scope_digest")
        _require(final.get("schema") == "approved-walk-result-v1"
            and final.get("checkpoints") == refs and len(refs) == len(contract.cutoffs) + 1
            and isinstance(inputs, list) and len(inputs) == len(refs)
            and final.get("partial") is result.partial and final.get("review_status") == result.review_status
            and isinstance(digest, str) and len(digest) == 64 and all(c in "0123456789abcdef" for c in digest),
            "Walk final document differs from its workflow result")
        comparison = final.get("comparison")
        _require(isinstance(comparison, dict), "Walk comparison is invalid")
        for name in ("hindsight_only_claim_ids", "available_after_last_cutoff_claim_ids", "unknown_availability_claim_ids"):
            values = comparison.get(name)
            _require(isinstance(values, list) and len(values) <= contract.scope.limit
                and all(isinstance(value, str) and 0 < len(value) <= 240 for value in values), "Walk comparison IDs exceed their bounds")
        changes = comparison.get("hindsight_findings_not_identical_to_last_as_lived")
        _require(isinstance(changes, list) and len(changes) <= 16, "Walk comparison findings exceed their bounds")
        cited = {}
        for finding in changes:
            _require(isinstance(finding, dict) and isinstance(finding.get("citations"), list)
                and len(finding["citations"]) <= 8, "Walk comparison citations are invalid")
            for citation in finding["citations"]:
                raw = citation["source_pin"]
                claim = AnalysisClaim.model_validate(raw)
                _require(all(getattr(claim, k) == v for k, v in case.items())
                    and claim.approved_revision_id == scope["approved_revision_id"]
                    and claim.approval_digest == scope["approval_digest"]
                    and claim.control_generation_id == scope["projection_generation_id"]
                    and (claim.id not in cited or cited[claim.id] == raw), "Walk comparison source escaped its projection")
                cited[claim.id] = raw
        _findings(changes, cited)
        for name in ("coverage", "validation", "family_independence"):
            _require(isinstance(final.get(name), str) and len(final[name]) <= 4096, "Walk summary text is invalid")
        if index is None:
            return final
        _require(type(index) is int and 0 <= index < len(refs), "Unknown walk step")
        # One final document plus selected input/checkpoint and at most one predecessor.
        # Never preload all 21 potentially large checkpoint objects.
        checkpoints = []
        for index, reference in [(index, refs[index])]:
            _require(isinstance(inputs[index], dict), "Walk input reference is invalid")
            input_ref = inputs[index].get("artifact")
            _require(WalkArtifact.model_validate(input_ref).bytes <= 96 * 1024, "Walk step exceeds the reasoning input budget")
            step = read(input_ref)
            checkpoint = read(reference["artifact"])
            horizon = contract.cutoffs[index] if index < len(contract.cutoffs) else None
            perspective = "as_lived" if horizon else "hindsight"
            for body, schema in ((step, "approved-walk-step-v1"), (checkpoint, "approved-walk-checkpoint-v1")):
                _require(body.get("schema") == schema and body.get("scope_digest") == digest
                    and body.get("step_index") == index and body.get("perspective") == perspective
                    and _date(body.get("horizon")) == horizon, "Walk checkpoint crossed its selected horizon or scope")
            previous = read(refs[index - 1]["artifact"]) if 0 < index < len(contract.cutoffs) else None
            if previous is not None:
                _require(previous.get("schema") == "approved-walk-checkpoint-v1"
                    and previous.get("scope_digest") == digest and previous.get("step_index") == index - 1
                    and previous.get("perspective") == "as_lived"
                    and _date(previous.get("horizon")) == contract.cutoffs[index - 1],
                    "Walk predecessor crossed its horizon or scope")
            expected_previous = {k: previous[k] for k in CHECKPOINT_FIELDS if k in previous} if previous else None
            _require(step.get("previous") == expected_previous, "Walk prior-state chain changed or seeded hindsight")
            _require(checkpoint.get("input_artifact") == input_ref
                and checkpoint.get("input_digest") == _digest(step)
                and checkpoint.get("review_status") == "pending"
                and checkpoint.get("prompt_version") == "approved-walk-balanced-v1",
                "Walk checkpoint does not match its exact reasoning input")
            _require(isinstance(checkpoint.get("prompt_sha256"), str) and len(checkpoint["prompt_sha256"]) == 64
                and all(c in "0123456789abcdef" for c in checkpoint["prompt_sha256"])
                and isinstance(checkpoint.get("model"), dict)
                and isinstance(checkpoint.get("limitations"), list) and len(checkpoint["limitations"]) <= 50
                and all(isinstance(value, str) and len(value) <= 4096 for value in checkpoint["limitations"]),
                "Walk model or limitation metadata is invalid")
            rows = step.get("claims")
            _require(isinstance(rows, list) and len(rows) <= contract.scope.limit, "Walk claims exceed the selected page")
            claims = {}
            for raw in rows:
                claim = AnalysisClaim.model_validate(raw)
                _require(claim.id not in claims and all(getattr(claim, k) == v for k, v in case.items())
                    and claim.approved_revision_id == scope["approved_revision_id"]
                    and claim.approval_digest == scope["approval_digest"]
                    and claim.control_generation_id == scope["projection_generation_id"]
                    and (horizon is None or claim.source_available_from is not None and claim.source_available_from <= horizon),
                    "Walk claim escaped its approved projection or historical cutoff")
                claims[claim.id] = raw
            _require(isinstance(inputs[index].get("has_more"), bool)
                and inputs[index].get("claim_count") == len(claims)
                and isinstance(checkpoint.get("claim_ids"), list)
                and sorted(checkpoint["claim_ids"]) == sorted(claims), "Walk claim coverage changed")
            _findings(checkpoint.get("findings"), claims)
            if expected_previous:
                _findings(expected_previous.get("findings"), claims)
            _require(not inputs[index]["has_more"] or result.partial, "Walk result concealed incomplete retrieval")
            return {"artifact": reference["artifact"], "checkpoint": checkpoint,
                "has_more": inputs[index]["has_more"], "claim_count": len(claims)}
    except ProfferError:
        raise
    except (ValidationError, ValueError, TypeError, KeyError, AttributeError):
        raise ProfferError("Walk result has an invalid protected document contract", 502) from None
