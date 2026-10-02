"""Case page: the registry read and the owner's edits, plus catalog counts.

Byline: Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02

Owner order 2026-10-01 07:56/07:57: Probata registry is the ONE identity
store; the Case page edits it. Owner 2026-10-02 02:12: identifiers are fixed
in place or deleted, and every change is one registry.identity_change row
(before, after, who, why). The engine
(Proffer starter, `/case-identity/*`) owns every registry read and write; this
module forwards the Authentik actor and the Idempotency-Key, checks the shape
of what comes back, and adds what the Case Bible catalog holds per identifier,
labeled with the store each number comes from.
"""

from __future__ import annotations

import asyncio
from typing import Any
from urllib.parse import quote

from app.repo import case_identity_catalog as catalog
from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor

_VIEW_KEYS = {"mode", "matter", "court_case", "people", "history", "probata_counts", "probata_unknowns", "dismissed"}
_RECEIPT_KEYS = {"ref", "kind", "recorded_at"}


def _headers(actor: ProfferDecisionActor, key: str) -> dict[str, str]:
    return {"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username, "Idempotency-Key": key}


def _object(response, label: str, keys: set[str]) -> dict[str, Any]:
    payload = proffer._json_payload(response, label)
    if not isinstance(payload, dict) or not keys <= payload.keys():
        raise ProfferError(f"Proffer starter returned an invalid {label}", 502)
    return payload


def _known_identifiers(view: dict[str, Any]) -> list[str]:
    known = {
        identifier.get("normalized")
        for person in view.get("people") or []
        for identifier in person.get("identifiers") or []
        if identifier.get("status") != "retired"
    }
    known.update(item.get("normalized") for item in view.get("dismissed") or [])
    return sorted(value for value in known if isinstance(value, str) and value)


def _catalog_section(view: dict[str, Any]) -> dict[str, Any]:
    """Blocking catalog reads, run in a worker thread."""
    if not catalog.configured():
        return {**catalog.STORE, "available": False, "error": "Case Bible catalog connection is not configured",
                "counts": [], "unknowns": {"phone": [], "name": []}}
    keys = [
        identifier.get("normalized")
        for person in view.get("people") or []
        for identifier in person.get("identifiers") or []
    ]
    known = _known_identifiers(view)
    try:
        return {
            **catalog.STORE,
            "available": True,
            "counts": catalog.counts([key for key in keys if isinstance(key, str)]),
            "unknowns": {"phone": catalog.unknowns(known, kind="phone"), "name": catalog.unknowns(known, kind="name")},
        }
    except catalog.CatalogError as error:
        return {**catalog.STORE, "available": False, "error": error.message, "counts": [], "unknowns": {"phone": [], "name": []}}


async def read(mode: MatterMode) -> dict[str, Any]:
    response = await proffer._request("GET", "/case-identity", params={"mode": mode})
    view = _object(response, "case identity", _VIEW_KEYS)
    if view.get("mode") != mode:
        raise ProfferError("Proffer starter returned the case identity of a different mode", 502)
    view["catalog"] = await asyncio.to_thread(_catalog_section, view)
    return view


async def lookup(values: list[str]) -> dict[str, Any]:
    response = await proffer._request("GET", "/case-identity/lookup", params=[("value", value) for value in values])
    return _object(response, "identifier lookup", {"matches", "store"})


async def catalog_events(identifier: str, match_on: str, before: str | None, limit: int) -> dict[str, Any]:
    try:
        return await asyncio.to_thread(catalog.events, identifier, match_on, before=before, limit=limit)
    except catalog.CatalogError as error:
        raise ProfferError(error.message, error.status) from None


async def _write(path: str, body: dict[str, Any], actor: ProfferDecisionActor, key: str, *, params=None) -> dict[str, Any]:
    response = await proffer._request("POST", path, json=body, headers=_headers(actor, key), params=params)
    receipt = _object(response, "case identity receipt", _RECEIPT_KEYS)
    receipt["replayed"] = bool(receipt.get("replayed"))
    return receipt


async def add_identifier(body: dict[str, Any], actor: ProfferDecisionActor, key: str) -> dict[str, Any]:
    return await _write("/case-identity/identifiers", body, actor, key)


async def edit_identifier(alias_id: str, body: dict[str, Any], actor: ProfferDecisionActor, key: str) -> dict[str, Any]:
    return await _write(f"/case-identity/identifiers/{quote(alias_id, safe='')}", body, actor, key)


async def delete_identifier(alias_id: str, body: dict[str, Any], actor: ProfferDecisionActor, key: str) -> dict[str, Any]:
    return await _write(f"/case-identity/identifiers/{quote(alias_id, safe='')}/delete", body, actor, key)


async def edit_header(body: dict[str, Any], actor: ProfferDecisionActor, key: str, *, mode: MatterMode) -> dict[str, Any]:
    return await _write("/case-identity/header", body, actor, key, params={"mode": mode})


async def add_person(body: dict[str, Any], actor: ProfferDecisionActor, key: str) -> dict[str, Any]:
    return await _write("/case-identity/people", body, actor, key)


async def edit_person(person_id: str, body: dict[str, Any], actor: ProfferDecisionActor, key: str) -> dict[str, Any]:
    return await _write(f"/case-identity/people/{quote(person_id, safe='')}", body, actor, key)


async def triage(body: dict[str, Any], actor: ProfferDecisionActor, key: str) -> dict[str, Any]:
    return await _write("/case-identity/triage", body, actor, key)
