"""Authenticated claims, evidence relationships, gaps and local follow-ups."""

from typing import Annotated, Literal
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Request

from legal_workspace.contracts.claims import (
    ClaimCreate,
    ClaimPatch,
    ClaimRecord,
    ClaimRevision,
    FollowupCreate,
    FollowupPatch,
    FromProbataCreate,
    GapCreate,
    GapPatch,
    GapReportRow,
    LinkCreate,
    LinkPatch,
    SourceOptions,
)
from legal_workspace.services.claims import ClaimNotFound, ClaimRevisionConflict, ClaimService
from legal_workspace.services.workspace import WORKSPACE


def actor(request: Request) -> str:
    principal = getattr(request.state, "auth", None)
    if principal is None:
        raise HTTPException(401, "Authentication required.")
    return f"{principal.source}:{principal.subject}"


router = APIRouter(dependencies=[Depends(actor)])


def get_claim_service() -> ClaimService:
    from legal_workspace.services.probata_records import ProbataUnavailable, get_record, list_records

    # One current upstream snapshot per kind in this request. Opening a saved
    # list must not issue one network request per claim, especially on outage.
    snapshots = {}

    def record_loader(kind, record_id):
        if kind not in snapshots:
            snapshots[kind] = list_records(kind)
        listing = snapshots[kind]
        if not listing.available:
            raise ProbataUnavailable(listing.reason)
        for row in listing.records:
            if row.origin.record_id == record_id:
                return row.model_dump(mode="json")
        return get_record(kind, record_id) if listing.truncated else None

    return ClaimService(
        WORKSPACE.store_dir,
        WORKSPACE.load().matter.matter_id,
        lambda: WORKSPACE.load().package,
        record_loader=record_loader,
    )


Service = Annotated[ClaimService, Depends(get_claim_service)]
Actor = Annotated[str, Depends(actor)]


def invoke(operation):
    try:
        return operation()
    except ClaimNotFound as exc:
        raise HTTPException(404, "Claim or linked record not found.") from exc
    except ClaimRevisionConflict as exc:
        raise HTTPException(
            409, {"message": str(exc), "current_revision": exc.current_revision}
        ) from exc
    except ValueError as exc:
        raise HTTPException(400, str(exc)) from exc


@router.get("/v1/claim-sources", response_model=SourceOptions)
def claim_sources(service: Service):
    return service.sources()


@router.get("/v1/claim-gaps", response_model=list[GapReportRow])
def claim_gaps(service: Service):
    return service.gap_report()


@router.get("/v1/claims", response_model=list[ClaimRecord])
def list_claims(service: Service):
    return service.list()


@router.post("/v1/claims", response_model=ClaimRecord, status_code=201)
def create_claim(body: ClaimCreate, service: Service, identity: Actor):
    return invoke(lambda: service.create(body, actor=identity))


@router.post("/v1/claims/from-probata", response_model=ClaimRecord)
def from_probata(body: FromProbataCreate, service: Service, identity: Actor):
    return invoke(lambda: service.from_probata(body, actor=identity))


@router.get("/v1/claims/by-origin", response_model=ClaimRecord | None)
def claim_by_origin(kind: Literal["entity", "event"], record_id: UUID, service: Service):
    return service.by_origin(kind, str(record_id))


@router.get("/v1/claims/{claim_id}", response_model=ClaimRecord)
def get_claim(claim_id: UUID, service: Service):
    return invoke(lambda: service.get(claim_id))


@router.get("/v1/claims/{claim_id}/history", response_model=list[ClaimRevision])
def claim_history(claim_id: UUID, service: Service):
    return invoke(lambda: service.history(claim_id))


@router.patch("/v1/claims/{claim_id}", response_model=ClaimRecord)
def patch_claim(claim_id: UUID, body: ClaimPatch, service: Service, identity: Actor):
    return invoke(lambda: service.patch(claim_id, body, actor=identity))


@router.post("/v1/claims/{claim_id}/links", response_model=ClaimRecord)
def add_link(claim_id: UUID, body: LinkCreate, service: Service, identity: Actor):
    return invoke(lambda: service.add_link(claim_id, body, actor=identity))


@router.patch("/v1/claims/{claim_id}/links/{link_id}", response_model=ClaimRecord)
def patch_link(claim_id: UUID, link_id: UUID, body: LinkPatch, service: Service, identity: Actor):
    return invoke(lambda: service.patch_link(claim_id, link_id, body, actor=identity))


@router.post("/v1/claims/{claim_id}/gaps", response_model=ClaimRecord)
def add_gap(claim_id: UUID, body: GapCreate, service: Service, identity: Actor):
    return invoke(lambda: service.add_gap(claim_id, body, actor=identity))


@router.patch("/v1/claims/{claim_id}/gaps/{gap_id}", response_model=ClaimRecord)
def patch_gap(claim_id: UUID, gap_id: UUID, body: GapPatch, service: Service, identity: Actor):
    return invoke(lambda: service.patch_gap(claim_id, gap_id, body, actor=identity))


@router.post("/v1/claims/{claim_id}/followups", response_model=ClaimRecord)
def add_followup(claim_id: UUID, body: FollowupCreate, service: Service, identity: Actor):
    return invoke(lambda: service.add_followup(claim_id, body, actor=identity))


@router.patch("/v1/claims/{claim_id}/followups/{followup_id}", response_model=ClaimRecord)
def patch_followup(
    claim_id: UUID, followup_id: UUID, body: FollowupPatch, service: Service, identity: Actor
):
    return invoke(lambda: service.patch_followup(claim_id, followup_id, body, actor=identity))
