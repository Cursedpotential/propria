"""Platform PostgreSQL review backend for the whole n8n Knowledge Reconciliation workflow.

Inputs: authorized, bounded n8n proposals; native-source adapter snapshots; owner review decisions.
Outputs: immutable identity-bearing review receipts, Workbench queue and detail.
Effects: ops.knowledge_review_case/knowledge_review_action only. NO source-memory updates,
evidence promotion, deletion, rule activation or Temporal workflow creation.
Use this endpoint as a single native data/approval boundary for the n8n workflow.
The n8n workflow owns reconciliation sequencing, candidate comparisons, and alerts.
"""
from __future__ import annotations

import asyncio
import hashlib
import hmac
import ipaddress
import json
import os
from pathlib import Path
from typing import Any, Literal
from urllib.parse import urlsplit
from uuid import UUID

import httpx
from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, ConfigDict, Field, ValidationError
from sqlalchemy import create_engine, text

from server.api.platform_auth import require_platform_owner

SCHEMA = "propria-knowledge-review-proposals/v1"
SOURCES = ("memsearch", "engram", "cnf", "docstore", "casebible", "superindex", "ccc")
MAX_IN = 192 * 1024
MAX_RESPONSE = 1024 * 1024
MAX_ITEMS = 500
_engine: Any = None


class Strict(BaseModel):
    model_config = ConfigDict(extra="forbid")


class Trigger(Strict):
    schema: Literal["propria-knowledge-trigger/v1"]
    kind: Literal["scheduled", "manual", "event"]
    scope: str = Field(pattern=r"^propria(?:/[A-Za-z0-9_./-]+)?$", max_length=120)
    source: str | None = None
    record_ref: str | None = Field(default=None, max_length=1024)
    revision: str | None = Field(default=None, max_length=256)
    type: str | None = None
    run_id: str = Field(min_length=1, max_length=128)


class Revision(Strict):
    ref: str = Field(min_length=3, max_length=1024)
    revision: str = Field(min_length=1, max_length=256)


class Candidate(Strict):
    kind: str = Field(min_length=1, max_length=80, pattern=r"^[a-z][a-z0-9_]*$")
    scope: str = Field(pattern=r"^propria(?:/[A-Za-z0-9_./-]+)?$", max_length=120)
    source_refs: list[str] = Field(min_length=1, max_length=20)
    source_revisions: list[Revision] = Field(min_length=1, max_length=20)
    severity: Literal["low", "medium", "high", "critical"]
    summary: str = Field(min_length=5, max_length=240)
    idempotency_key: str = Field(min_length=1, max_length=800)
    proposed_action: Literal["review"]
    automatic: Literal[False]


class ProposalBatch(Strict):
    schema: Literal["propria-knowledge-review-proposals/v1"]
    run_id: str = Field(min_length=1, max_length=128)
    scope: str = Field(pattern=r"^propria(?:/[A-Za-z0-9_./-]+)?$", max_length=120)
    scanned: int = Field(ge=0, le=MAX_ITEMS)
    findings: int = Field(ge=0, le=100)
    review_candidates: list[Candidate] = Field(max_length=100)
    coverage: list[dict[str, Any]] = Field(max_length=30)


class Decision(Strict):
    expected_case_version: int = Field(ge=1)
    request_id: str = Field(min_length=16, max_length=256)
    decision: Literal["approve_plan", "reject", "defer"]
    rationale: str = Field(min_length=2, max_length=2000)
    reviewer_ref: str = Field(min_length=1, max_length=256)


def _db():
    """Reuse Platform's sole canonical PG URL; no secondary database or table auto-creation."""
    global _engine
    if _engine is None:
        from server.core.url import db_url
        _engine = create_engine(db_url, pool_pre_ping=True)
    return _engine


def _unpack(raw: Any) -> Any:
    return json.loads(raw) if isinstance(raw, str) else raw


def _only_refs(candidate: Candidate) -> tuple[list[str], list[dict[str, str]], str]:
    """Validate stable source identity and return server-computed idempotency hash."""
    refs = sorted(set(candidate.source_refs))
    revisions = sorted((row.model_dump() for row in candidate.source_revisions),
                       key=lambda row: row["ref"])
    if len(refs) != len(candidate.source_refs) or refs != [row["ref"] for row in revisions]:
        raise HTTPException(422, "source references and revisions must match exactly")
    if any(row["ref"].split(":", 1)[0] not in SOURCES for row in revisions):
        raise HTTPException(422, "unrecognized memory/source identity")
    fingerprint = hashlib.sha256(json.dumps({
        "scope": candidate.scope, "kind": candidate.kind, "revisions": revisions
    }, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    return refs, revisions, fingerprint


def _write_batch(batch: ProposalBatch) -> dict:
    """Persist review identities transactionally. Existing fingerprints are never reopened or re-alerted."""
    if batch.findings != len(batch.review_candidates):
        raise HTTPException(422, "candidate count and finding count disagree")
    seen: set[str] = set()
    saved: list[dict[str, Any]] = []
    try:
        with _db().begin() as conn:
            for row in batch.review_candidates:
                if row.scope != batch.scope:
                    raise HTTPException(422, "proposal escaped its input scope")
                refs, revisions, fingerprint = _only_refs(row)
                if fingerprint in seen:
                    raise HTTPException(422, "repeated proposal in one request")
                seen.add(fingerprint)
                inserted = conn.execute(text("""
                    INSERT INTO ops.knowledge_review_case
                        (fingerprint, scope, kind, severity, summary, source_refs, source_revisions)
                    VALUES (:fp, :scope, :kind, :severity, :summary,
                        CAST(:refs AS jsonb), CAST(:revisions AS jsonb))
                    ON CONFLICT (fingerprint) DO NOTHING
                    RETURNING review_ref
                """), {"fp": fingerprint, "scope": row.scope, "kind": row.kind,
                       "severity": row.severity, "summary": row.summary,
                       "refs": json.dumps(refs), "revisions": json.dumps(revisions)}).scalar()
                if inserted is None:
                    data = conn.execute(text("""
                        UPDATE ops.knowledge_review_case SET last_seen_at=now()
                        WHERE fingerprint=:fp
                        RETURNING review_ref, status, severity, kind
                    """), {"fp": fingerprint}).mappings().one()
                    created = False
                else:
                    data = {"review_ref": inserted, "status": "queued",
                            "severity": row.severity, "kind": row.kind}
                    created = True
                saved.append({
                    "review_ref": str(data["review_ref"]), "status": data["status"],
                    "severity": data["severity"], "kind": data["kind"],
                    "newly_created": created
                })
    except HTTPException:
        raise
    except Exception:
        raise HTTPException(503, "canonical PostgreSQL knowledge review writer unavailable") from None
    return {"schema": "propria-knowledge-review-write/v1", "run_id": batch.run_id,
            "review_queue": saved, "source_mutations": 0}


def _list_review(limit: int = 100) -> dict:
    """List only pending cases for the existing Activity surface; never render raw source text."""
    limit = max(1, min(100, int(limit)))
    try:
        with _db().connect() as conn:
            total = conn.execute(text("""
                SELECT count(*) FROM ops.knowledge_review_case
                WHERE status IN ('queued', 'deferred')
            """)).scalar_one()
            rows = conn.execute(text("""
                SELECT review_ref, kind, severity, status, scope, summary, source_refs, case_version
                FROM ops.knowledge_review_case
                WHERE status IN ('queued','deferred')
                ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1
                         WHEN 'medium' THEN 2 ELSE 3 END,
                         created_at DESC
                LIMIT :n
            """), {"n": limit}).mappings().all()
    except Exception:
        raise HTTPException(503, "canonical review inbox unavailable") from None
    items = [
        {"review_ref": str(r["review_ref"]), "kind": r["kind"],
         "severity": r["severity"], "status": r["status"], "scope": r["scope"],
         "summary": r["summary"],
         "source_count": len(_unpack(r["source_refs"]) or []),
         "case_version": r["case_version"],
         "review_href": f"/activity?knowledge_review={r['review_ref']}"}
        for r in rows
    ]
    return {"schema": "propria-knowledge-review-queue/v1", "total": int(total),
            "items": items, "next_cursor": "more" if int(total) > len(items) else None}


def _detail(review_ref: UUID) -> dict:
    """Read one version-pinned proposal; source references, not derived evidence, appear in review UI."""
    try:
        with _db().connect() as conn:
            record = conn.execute(text("""
                SELECT review_ref,kind,severity,status,scope,summary,
                       source_refs,source_revisions,case_version,final_disposition,
                       created_at,decided_at,native_write_receipt
                FROM ops.knowledge_review_case WHERE review_ref=:ref
            """), {"ref": str(review_ref)}).mappings().first()
    except Exception:
        raise HTTPException(503, "canonical review case unavailable") from None
    if record is None:
        raise HTTPException(404, "review case not found")
    return {"schema": "propria-knowledge-review-detail/v1",
            **{k: (str(v) if k == "review_ref" else v.isoformat()
                   if k in ("created_at", "decided_at") and v else _unpack(v)
                   if k in ("source_refs", "source_revisions", "native_write_receipt") else v)
               for k, v in dict(record).items()}}


def _reviewer_token_ok(request: Request) -> bool:
    """Separate human-decision capability: n8n's normal Platform bearer is insufficient."""
    configured = os.environ.get("KNOWLEDGE_REVIEW_DECISION_TOKEN_FILE", "")
    if not configured:
        return False
    try:
        secret = Path(configured).read_text(encoding="utf-8").strip()
    except OSError:
        return False
    provided = request.headers.get("x-knowledge-review-owner-token", "")
    return bool(32 <= len(secret) <= 4096 and
                hmac.compare_digest(secret.encode(), provided.encode()))


def _decide(review_ref: UUID, payload: Decision) -> dict:
    """Append a checked human verdict. Approve_plan does not mutate a source or certify an apply."""
    try:
        with _db().begin() as conn:
            prior = conn.execute(text("""
                SELECT action_id,decision,case_version FROM ops.knowledge_review_action
                WHERE idempotency_key=:key
            """), {"key": payload.request_id}).mappings().first()
            if prior:
                return {"review_ref": str(review_ref), "decision": prior["decision"],
                        "case_version": prior["case_version"],
                        "action_ref": str(prior["action_id"]), "duplicate": True,
                        "source_applied": False}
            locked = conn.execute(text("""
                SELECT review_ref,status,case_version FROM ops.knowledge_review_case
                WHERE review_ref=:ref FOR UPDATE
            """), {"ref": str(review_ref)}).mappings().first()
            if not locked:
                raise HTTPException(404, "review case not found")
            if locked["case_version"] != payload.expected_case_version:
                raise HTTPException(409, "review case revision changed; reload before deciding")
            if locked["status"] == "decided":
                raise HTTPException(409, "already decided")
            action_id = conn.execute(text("""
                INSERT INTO ops.knowledge_review_action
                (review_ref, case_version, idempotency_key, reviewer_ref, decision, rationale)
                VALUES (:ref,:v,:key,:reviewer,:decision,:rationale)
                RETURNING action_id
            """), {"ref": str(review_ref), "v": locked["case_version"]+1,
                   "key": payload.request_id, "reviewer": payload.reviewer_ref,
                   "decision": payload.decision, "rationale": payload.rationale}).scalar_one()
            new_status = "deferred" if payload.decision == "defer" else "decided"
            conn.execute(text("""
                UPDATE ops.knowledge_review_case
                SET status=:status, case_version=case_version+1,
                    final_disposition=:disposition,
                    decided_at=CASE WHEN :status='decided' THEN now() ELSE NULL END
                WHERE review_ref=:ref
            """), {"ref": str(review_ref), "status": new_status,
                   "disposition": payload.decision if new_status == "decided" else None})
            return {"review_ref": str(review_ref), "decision": payload.decision,
                    "case_version": locked["case_version"]+1,
                    "action_ref": str(action_id), "duplicate": False,
                    "source_applied": False}
    except HTTPException:
        raise
    except Exception:
        raise HTTPException(503, "review decision could not be committed") from None


def _configured_adapters() -> dict[str, tuple[str, str]]:
    """Use only operator-configured native reader URLs and runtime token files."""
    config = os.environ.get("KNOWLEDGE_RECONCILE_SOURCES_JSON", "")
    if not config:
        return {}
    try:
        data = json.loads(config)
        if not isinstance(data, list):
            raise ValueError()
        configured: dict[str, tuple[str, str]] = {}
        for row in data:
            if not isinstance(row, dict) or set(row) != {"store", "url", "token_file"}:
                raise ValueError()
            store, url, token_file = row["store"], row["url"], row["token_file"]
            if store not in SOURCES or store in configured or not isinstance(token_file, str):
                raise ValueError()
            address = urlsplit(url)
            if (address.scheme not in ("https", "http") or not address.hostname or
                address.username or address.password or address.query or address.fragment or
                not address.path.startswith("/")):
                raise ValueError()
            if address.scheme == "http":
                ip = ipaddress.ip_address(address.hostname)
                if not (ip.is_loopback or ip in ipaddress.ip_network("100.64.0.0/10")):
                    raise ValueError()
            configured[store] = (url, token_file)
        return configured
    except (ValueError, TypeError, KeyError):
        raise HTTPException(503, "source adapter configuration is invalid") from None


async def _one_snapshot(store: str, config: tuple[str, str] | None, trigger: Trigger):
    if config is None:
        return [], {"store": store, "state": "unavailable",
                    "reason": "no authorized native source adapter configured"}
    url, token_file = config
    try:
        token = Path(token_file).read_text(encoding="utf-8").strip()
        if not token:
            raise ValueError()
        async with httpx.AsyncClient(timeout=20, follow_redirects=False,
                                     trust_env=False) as client:
            response = await client.post(url, headers={
                "authorization": "Bearer " + token,
                "content-type": "application/json"
            }, json={"schema": "propria-knowledge-snapshot-request/v1",
                     "scope": trigger.scope, "kind": trigger.kind,
                     "record_ref": trigger.record_ref if trigger.source == store else None,
                     "revision": trigger.revision if trigger.source == store else None,
                     "limit": 100})
        if response.status_code != 200 or len(response.content) > MAX_RESPONSE:
            raise ValueError()
        result = response.json()
        if result.get("schema") != "propria-knowledge-source-snapshot/v1" or not isinstance(result.get("records"), list):
            raise ValueError()
        if len(result["records"]) > 100:
            raise ValueError()
        output = []
        for row in result["records"]:
            if (not isinstance(row, dict) or row.get("store") != store or
                not isinstance(row.get("id"), str) or not row["id"] or
                not isinstance(row.get("revision"), str) or not row["revision"] or
                not isinstance(row.get("scope"), str) or
                not row["scope"].startswith(trigger.scope) or
                not isinstance(row.get("subject_key"), str) or
                not isinstance(row.get("text"), str) or len(row["text"]) > 2000 or
                len(row["subject_key"]) > 240):
                raise ValueError()
            output.append({k: row[k] for k in
               ("store", "id", "revision", "scope", "subject_key", "text",
                "source_hash", "state", "supersedes", "expires_at") if k in row})
        coverage = {"store": store, "state": "partial" if result.get("has_more") else "queried",
                    "reason": "source result was bounded at 100" if result.get("has_more") else ""}
        return output, coverage
    except (OSError, ValueError, TypeError, json.JSONDecodeError, httpx.HTTPError):
        return [], {"store": store, "state": "unavailable",
                    "reason": "native source adapter unreachable, unauthorized, or returned invalid data"}


async def _snapshot(trigger: Trigger):
    """Concurrent bounded native reads; nonresponding stores stay visibly unavailable."""
    configured = _configured_adapters()
    selected = [trigger.source] if trigger.source else list(SOURCES)
    outcomes = await asyncio.gather(*[
        _one_snapshot(store, configured.get(store), trigger) for store in selected
    ])
    all_rows = [row for rows, _ in outcomes for row in rows]
    if len(all_rows) > MAX_ITEMS:
        all_rows = all_rows[:MAX_ITEMS]
        coverage = [dict(item, state="partial", reason="global 500-row bound")
                    for _, item in outcomes]
    else:
        coverage = [item for _, item in outcomes]
    return {"schema": "propria-knowledge-snapshot/v1", "records": all_rows,
            "coverage": coverage}


async def _json(request: Request, max_bytes: int = MAX_IN) -> dict:
    require_platform_owner(request)
    contents = bytearray()
    async for chunk in request.stream():
        contents.extend(chunk)
        if len(contents) > max_bytes:
            raise HTTPException(413, "knowledge request too large")
    try:
        value = json.loads(contents)
        if not isinstance(value, dict):
            raise ValueError()
        return value
    except (ValueError, TypeError):
        raise HTTPException(422, "knowledge request must be a JSON object") from None


def register_knowledge_reconcile_routes(app: FastAPI) -> None:
    """Attach one batch source and review API to the EXISTING Platform FastAPI app."""
    @app.post("/v1/knowledge-reconcile/snapshot")
    async def snapshot(request: Request):
        raw = await _json(request, 16384)
        try:
            trigger = Trigger.model_validate(raw)
        except ValidationError:
            raise HTTPException(422, "invalid source snapshot request") from None
        if trigger.source and trigger.source not in SOURCES:
            raise HTTPException(422, "unknown native source")
        return await _snapshot(trigger)

    @app.post("/v1/knowledge-reconcile/reviews")
    async def save_reviews(request: Request):
        raw = await _json(request)
        try:
            batch = ProposalBatch.model_validate(raw)
        except ValidationError:
            raise HTTPException(422, "invalid review batch") from None
        return await asyncio.to_thread(_write_batch, batch)

    @app.get("/v1/knowledge-reconcile/reviews")
    async def review_queue(request: Request, limit: int = 100):
        require_platform_owner(request)
        return await asyncio.to_thread(_list_review, limit)

    @app.get("/v1/knowledge-reconcile/reviews/{review_ref}")
    async def review_detail(request: Request, review_ref: UUID):
        require_platform_owner(request)
        return await asyncio.to_thread(_detail, review_ref)

    @app.post("/v1/knowledge-reconcile/reviews/{review_ref}/decision")
    async def review_decision(request: Request, review_ref: UUID):
        raw = await _json(request, 4096)
        if not _reviewer_token_ok(request):
            raise HTTPException(403, "owner review capability is not configured or unauthorized")
        try:
            payload = Decision.model_validate(raw)
        except ValidationError:
            raise HTTPException(422, "invalid review decision") from None
        return await asyncio.to_thread(_decide, review_ref, payload)
