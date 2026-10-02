"""Workbench BFF routes for the Case page (case identity).

Byline: Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02

    GET  /api/case-identity?mode=TEST|REAL       case header, people, identifiers, counts, unknowns
    GET  /api/case-identity/lookup?value=...     who used these identifiers (read tool for other apps)
    GET  /api/case-identity/catalog-events       the Case Bible events behind one catalog count
    POST /api/case-identity/identifiers          add an identifier
    POST /api/case-identity/identifiers/{id}     fix an identifier in place
    POST /api/case-identity/identifiers/{id}/delete  remove an identifier
    POST /api/case-identity/header?mode=         edit the matter or its court case
    POST /api/case-identity/people               add a person
    POST /api/case-identity/people/{person_id}   edit a person
    POST /api/case-identity/triage               dismiss / reopen an identifier tied to nobody

Every write needs the Authentik actor (tailnet identity, or a machine JWT) and
an Idempotency-Key; the engine validates and persists. Bodies pass through as
JSON objects: the engine refuses unknown fields, so the BFF does not keep a
second copy of the contract.
"""

from __future__ import annotations

from typing import Annotated, Any

from fastapi import APIRouter, Body, Header, HTTPException, Path, Query, Request

from app.service import case_identity as service
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor

router = APIRouter(prefix="/api/case-identity", tags=["case identity"])

IdempotencyKey = Annotated[str, Header(alias="Idempotency-Key", min_length=1, max_length=200)]
JsonObject = Annotated[dict[str, Any], Body()]


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


def _actor(request: Request) -> ProfferDecisionActor:
    subject_uid = str(getattr(request.state, "subject_uid", "")).strip()
    username = str(getattr(request.state, "principal", "")).strip()
    if not subject_uid or not username:
        raise HTTPException(status_code=401, detail="authenticated subject identity is unavailable")
    try:
        return ProfferDecisionActor(subject_uid=subject_uid, username=username)
    except ValueError:
        raise HTTPException(status_code=401, detail="authenticated subject identity is invalid") from None


@router.get("")
async def read_endpoint(mode: Annotated[MatterMode, Query()]):
    try:
        return await service.read(mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/lookup")
async def lookup_endpoint(value: Annotated[list[str], Query(min_length=1, max_length=200)]):
    try:
        return await service.lookup(value)
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/catalog-events")
async def catalog_events_endpoint(
    identifier: Annotated[str, Query(min_length=1, max_length=512)],
    match_on: Annotated[str, Query(pattern="^(counterparty_phone|sender)$")],
    before: Annotated[str | None, Query(max_length=64)] = None,
    limit: Annotated[int, Query(ge=1, le=200)] = 50,
):
    try:
        return await service.catalog_events(identifier, match_on, before, limit)
    except ProfferError as error:
        raise _translate(error) from None


UuidPath = Annotated[str, Path(pattern="^[0-9a-fA-F-]{36}$")]


@router.post("/identifiers", status_code=201)
async def identifier_endpoint(body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.add_identifier(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/identifiers/{alias_id}", status_code=201)
async def identifier_edit_endpoint(alias_id: UuidPath, body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.edit_identifier(alias_id, body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/identifiers/{alias_id}/delete", status_code=201)
async def identifier_delete_endpoint(alias_id: UuidPath, body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.delete_identifier(alias_id, body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/header", status_code=201)
async def header_endpoint(body: JsonObject, request: Request, key: IdempotencyKey, mode: Annotated[MatterMode, Query()]):
    actor = _actor(request)
    try:
        return await service.edit_header(body, actor, key, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/people", status_code=201)
async def add_person_endpoint(body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.add_person(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/people/{person_id}", status_code=201)
async def edit_person_endpoint(person_id: UuidPath, body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.edit_person(person_id, body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/triage", status_code=201)
async def triage_endpoint(body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.triage(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/placeholders", status_code=201)
async def placeholders_endpoint(body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.add_placeholders(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/contact-people", status_code=201)
async def contact_people_endpoint(body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.add_contact_people(body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/people/{person_id}/merge", status_code=201)
async def merge_person_endpoint(person_id: UuidPath, body: JsonObject, request: Request, key: IdempotencyKey):
    actor = _actor(request)
    try:
        return await service.merge_person(person_id, body, actor, key)
    except ProfferError as error:
        raise _translate(error) from None
