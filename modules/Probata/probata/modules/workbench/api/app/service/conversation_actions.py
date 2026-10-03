"""Conversation actions: start Extract and Send to Surreal on the engine, follow them, and read what the extractors found.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The engine decides and runs; this module resolves the opaque conversation ids the Imported view lists, forwards to the Proffer
starter (``/reference-import/conversations/*``) with the Authentik actor and an Idempotency-Key, and reads the
extraction tables read-only (``repo/extractions_pg``). Always the live case: no matter or mode parameter.

AI chats never come here: the conversation ids are the Imported view's message threads, and an AI chat is never a
message conversation.
"""

from __future__ import annotations

import asyncio
import time
from typing import Any

from app.repo import extractions_pg as pg
from app.service import imported, proffer
from app.service.proffer_errors import ProfferError
from app.types.conversation_actions import (
    ExtractConversationsRequest,
    SendConversationsRequest,
    WorkflowStarted,
    WorkflowStatus,
)
from app.types.proffer import ProfferDecisionActor

_WORKFLOW_PREFIXES = ("conversation-extraction:", "send-to-surreal:", "auto-extraction:")

# The extractor registry changes with a deploy, not with a request.
_REGISTRY_TTL = 60.0
_registry: tuple[float, dict[str, Any]] | None = None


def _actor_headers(actor: ProfferDecisionActor, idempotency_key: str | None) -> dict[str, str]:
    headers = {"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username}
    if idempotency_key is not None:
        headers["Idempotency-Key"] = idempotency_key
    return headers


def conversations(thread_ids: list[str]) -> list[dict[str, str]]:
    """Resolve opaque thread ids to the (export file, conversation) pairs the engine takes. An unknown id is a 404."""
    out: list[dict[str, str]] = []
    seen: set[str] = set()
    for thread_id in thread_ids:
        if thread_id in seen:
            continue
        seen.add(thread_id)
        try:
            export_key, conv = imported.decode_id(thread_id, 2)
        except imported.ImportedError as error:
            raise ProfferError(error.message, error.status) from None
        out.append({"export_key": export_key, "conv": conv})
    return out


async def extractors() -> dict[str, Any]:
    """Every selectable extractor and the default, as the engine's registry lists them."""
    response = await proffer._request("GET", "/reference-import/extractors")
    payload = proffer._json_payload(response, "extractor registry")
    if not isinstance(payload, dict) or not isinstance(payload.get("extractors"), list):
        raise ProfferError("Proffer starter returned an invalid extractor registry", 502)
    return payload


async def _registry_by_run_extractor() -> dict[str, dict[str, Any]]:
    """Run extractor value -> registry entry, cached; empty when the engine cannot be reached (runs then group under their own names)."""
    global _registry
    now = time.monotonic()
    if _registry is None or now - _registry[0] > _REGISTRY_TTL:
        try:
            listing = await extractors()
        except ProfferError:
            return {}
        mapping: dict[str, dict[str, Any]] = {}
        for entry in listing["extractors"]:
            for run_extractor in entry.get("run_extractors") or [entry.get("run_extractor")]:
                if run_extractor:
                    mapping[run_extractor] = entry
        _registry = (now, mapping)
    return _registry[1]


async def start_extraction(request: ExtractConversationsRequest, actor: ProfferDecisionActor, key: str) -> WorkflowStarted:
    """Start extraction_request_workflow for the chosen conversations and extractors."""
    body = {
        "matter_id": imported.live_matter(),
        "conversations": conversations(request.thread_ids),
        "extractors": list(dict.fromkeys(request.extractors)),
    }
    response = await proffer._request(
        "POST", "/reference-import/conversations/extract", json=body, headers=_actor_headers(actor, key)
    )
    return proffer._validated(WorkflowStarted, proffer._json_payload(response, "extraction start"), "extraction start")


async def start_send(request: SendConversationsRequest, actor: ProfferDecisionActor, key: str) -> WorkflowStarted:
    """Start send_to_surreal_workflow for the chosen conversations."""
    body = {
        "matter_id": imported.live_matter(),
        "conversations": conversations(request.thread_ids),
        "include_extractions": request.include_extractions,
    }
    response = await proffer._request(
        "POST", "/reference-import/conversations/send-to-surreal", json=body, headers=_actor_headers(actor, key)
    )
    return proffer._validated(WorkflowStarted, proffer._json_payload(response, "send start"), "send start")


async def workflow_status(workflow_id: str) -> WorkflowStatus:
    """Progress of an extraction or a send. Only workflows this surface starts can be read."""
    if not workflow_id.startswith(_WORKFLOW_PREFIXES):
        raise ProfferError("unknown workflow", 404)
    response = await proffer._request("GET", f"/reference-import/conversations/workflows/{workflow_id}")
    return proffer._validated(WorkflowStatus, proffer._json_payload(response, "workflow status"), "workflow status")


def _iso(value: Any) -> str | None:
    return value.isoformat() if hasattr(value, "isoformat") else (str(value) if value is not None else None)


def _run(row: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": row["run_id"], "extractor": row["extractor"], "version": row["extractor_version"], "model_id": row["model_id"] or None,
        "status": row["status"], "error": row["error"] or None, "started_at": _iso(row["started_at"]),
        "finished_at": _iso(row["finished_at"]), "stats": row["stats"] or {},
    }


def _entity(row: dict[str, Any]) -> dict[str, Any]:
    aliases = [a.get("text") for a in (row["aliases"] or []) if isinstance(a, dict) and a.get("text")]
    return {
        "id": row["id"], "run_id": row["run_id"], "name": row["name"], "type": row["entity_type"], "confidence": row["confidence"],
        "review_state": row["review_state"], "aliases": aliases, "mention_count": row["mention_count"] or row["model_mentions"],
        "mentions": [
            {"record_id": m.get("record_id"), "surface": m.get("surface"), "snippet": m.get("snippet")} for m in (row["mentions"] or [])
        ],
    }


def _event(row: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": row["id"], "run_id": row["run_id"], "title": row["title"], "type": row["event_type"], "occurred_at": _iso(row["occurred_at"]),
        "precision": row["precision"], "confidence": row["confidence"], "review_state": row["review_state"],
        "record_ids": [r for r in (row["record_ids"] or []) if r], "description": row["description"] or "",
        "when_stated": row["when_stated"] or None,
    }


def group_by_extractor(
    runs: list[dict[str, Any]], entities: list[dict[str, Any]], events: list[dict[str, Any]], registry: dict[str, dict[str, Any]]
) -> list[dict[str, Any]]:
    """Group runs, entities and events under the extractor that made them.

    The default extractor's rule, model and reconcile steps are three runs of one extractor (the registry says which
    run extractors belong together); an extractor the registry does not know is grouped under its own run name, so a
    result is never hidden because the registry changed. Order: registry order, then unknown names.
    """
    group_of: dict[str, tuple[str, str]] = {}
    groups: dict[str, dict[str, Any]] = {}

    def group_for(run_extractor: str) -> dict[str, Any]:
        entry = registry.get(run_extractor)
        key, label = (entry["id"], entry["label"]) if entry else (run_extractor, run_extractor)
        if key not in groups:
            groups[key] = {
                "id": key, "label": label, "kind": entry.get("kind") if entry else None,
                "compare_only": bool(entry.get("compare_only")) if entry else False,
                "runs": [], "entities": [], "events": [],
            }
        return groups[key]

    for row in runs:
        group = group_for(row["extractor"])
        group_of[row["run_id"]] = (group["id"], row["extractor"])
        group["runs"].append(_run(row))
    for row in entities:
        key = group_of.get(row["run_id"])
        if key:
            groups[key[0]]["entities"].append(_entity(row))
    for row in events:
        key = group_of.get(row["run_id"])
        if key:
            groups[key[0]]["events"].append(_event(row))
    registry_order = {entry["id"]: i for i, entry in enumerate(_unique_entries(registry))}
    ordered = sorted(groups.values(), key=lambda g: (registry_order.get(g["id"], len(registry_order)), g["id"]))
    for group in ordered:
        group["counts"] = {"entities": len(group["entities"]), "events": len(group["events"]), "runs": len(group["runs"])}
        group["status"] = _group_status(group["runs"])
    return ordered


def _unique_entries(registry: dict[str, dict[str, Any]]) -> list[dict[str, Any]]:
    seen: dict[str, dict[str, Any]] = {}
    for entry in registry.values():
        seen.setdefault(entry["id"], entry)
    return list(seen.values())


def _group_status(runs: list[dict[str, Any]]) -> str:
    states = {run["status"] for run in runs}
    if "running" in states:
        return "running"
    if "failed" in states:
        return "failed" if states == {"failed"} else "completed_with_failures"
    return "completed"


async def extractions(thread_id: str) -> dict[str, Any]:
    """What the extractors found for one conversation, grouped by extractor. Read-only; empty groups are not invented."""
    matter = imported.live_matter()
    try:
        export_key, conv = imported.decode_id(thread_id, 2)
    except imported.ImportedError as error:
        raise ProfferError(error.message, error.status) from None
    try:
        run_rows, entity_rows, event_rows = await asyncio.gather(
            asyncio.to_thread(pg.runs, matter, export_key, conv),
            asyncio.to_thread(pg.entities, matter, export_key, conv),
            asyncio.to_thread(pg.events, matter, export_key, conv),
        )
    except pg.ImportedError as error:
        raise ProfferError(error.message, error.status) from None
    truncated = len(entity_rows) > pg.MAX_ROWS or len(event_rows) > pg.MAX_ROWS
    registry = await _registry_by_run_extractor()
    grouped = group_by_extractor(run_rows, entity_rows[: pg.MAX_ROWS], event_rows[: pg.MAX_ROWS], registry)
    return {"thread_id": thread_id, "extractors": grouped, "truncated": truncated}


def _reset_registry_cache() -> None:
    """Forget the cached registry (tests)."""
    global _registry
    _registry = None
