"""Authenticated live Probata projections for the claims workspace."""
from typing import Literal

from fastapi import APIRouter, Depends, Query

from legal_workspace.api.claim_routes import actor
from legal_workspace.services.probata_records import Listing, list_records

router = APIRouter(dependencies=[Depends(actor)])


@router.get("/v1/probata/records", response_model=Listing)
def probata_records(kind: Literal["entity", "event"] = "event",
                   q: str = Query("", max_length=200)):
    return list_records(kind, q)
