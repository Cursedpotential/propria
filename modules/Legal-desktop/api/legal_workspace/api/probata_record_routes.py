"""Authenticated live Probata projections for the claims workspace."""
from typing import Literal
from uuid import UUID

from fastapi import APIRouter, Depends, Query

from legal_workspace.api.claim_routes import actor
from legal_workspace.services.probata_records import Listing, list_records

router = APIRouter(dependencies=[Depends(actor)])


@router.get("/v1/probata/records", response_model=Listing)
def probata_records(kind: Literal["entity", "event"] = "event",
                   q: str = Query("", max_length=200), record_id: UUID | None = None):
    return list_records(kind, q, record_id=str(record_id)) if record_id else list_records(kind, q)
