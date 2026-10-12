"""Bound explicit current owner readmission without inventing historical authority.

Inputs: source version, fixed case, explicit LIVE and reason. Outputs: validated
reference-only request. Effects: none; select for existing unknown preview bindings.
"""
from uuid import UUID
from typing import Literal
from pydantic import BaseModel, ConfigDict, Field


class AdmissionRepairRequest(BaseModel):
    """Require explicit owner mode and actual source/case for a standalone Temporal repair.

    Inputs: existing version and scope, current LIVE intent, bounded reason.
    Output: engine repair body. Effects: none until forwarded to Temporal.
    """
    model_config = ConfigDict(extra="forbid")
    source_version_ref: UUID
    operating_mode: Literal["LIVE"]
    matter_id: UUID
    court_case_id: UUID
    reason: str = Field(min_length=1, max_length=512)
