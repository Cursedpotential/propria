"""Calendar HTTP: Court dates, deadlines, and Michigan family-law deadline calculations."""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from typing import Any, Dict, List, Optional
from uuid import UUID

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from legal_workspace.domain.calendar import DocketEvent, DocketEventCreate, EventKind
from legal_workspace.services.workspace import WORKSPACE

router = APIRouter()


class DeadlineCalculationRequest(BaseModel):
    """Request for calculating Michigan family-law deadlines."""
    trigger_event: str = Field(..., description="Description of the trigger event")
    trigger_date: datetime = Field(..., description="Date of the trigger event")
    event_type: str = Field(..., description="Type of event that triggered the calculation")


class DeadlineCalculationResponse(BaseModel):
    """Response containing calculated deadlines."""
    trigger_event: str
    trigger_date: datetime
    calculated_deadlines: List[Dict[str, Any]]


class MichiganFamilyLawDeadlineCalculator:
    """Calculate deadlines specific to Michigan family law cases."""

    @staticmethod
    def calculate_deadlines(trigger_date: datetime, event_type: str) -> List[Dict[str, Any]]:
        """
        Calculate Michigan family-law deadlines based on a trigger event.

        Args:
            trigger_date: The date of the triggering event
            event_type: Type of event (e.g., "complaint_filed", "served", "motion_filed")

        Returns:
            List of deadline calculations with descriptions and dates
        """
        deadlines = []

        # Ensure trigger_date is timezone-aware (UTC)
        if trigger_date.tzinfo is None:
            trigger_date = trigger_date.replace(tzinfo=UTC)

        if event_type.lower() in ["complaint_filed", "petition_filed"]:
            # Response to complaint/petition: 21 days after service
            # Note: In practice, this is 21 days after service, not filing
            # But we'll calculate from filing as a baseline
            response_due = trigger_date + timedelta(days=21)
            deadlines.append({
                "description": "Response to complaint/petition due",
                "deadline_date": response_due,
                "rule_reference": "MCR 3.205(A)",
                "days_offset": 21,
                "deadline_type": "response"
            })

            # Initial pretrial conference (varies by court, typically 60-90 days after filing)
            pretrial_date = trigger_date + timedelta(days=75)  # Approximate
            deadlines.append({
                "description": "Initial pretrial conference",
                "deadline_date": pretrial_date,
                "rule_reference": "Local court rule",
                "days_offset": 75,
                "deadline_type": "pretrial"
            })

        elif event_type.lower() in ["served", "service_completed"]:
            # Response to complaint/petition: 21 days after service
            response_due = trigger_date + timedelta(days=21)
            deadlines.append({
                "description": "Response to complaint/petition due",
                "deadline_date": response_due,
                "rule_reference": "MCR 3.205(A)",
                "days_offset": 21,
                "deadline_type": "response"
            })

        elif event_type.lower() in ["motion_filed", "motion_served"]:
            # Response to motion: varies but typically 7 days before hearing
            # Notice of hearing: at least 7 days before hearing (MCR 3.209)
            # For simplicity, we'll calculate a typical motion response deadline
            response_due = trigger_date + timedelta(days=7)
            deadlines.append({
                "description": "Response to motion due",
                "deadline_date": response_due,
                "rule_reference": "MCR 3.209",
                "days_offset": 7,
                "deadline_type": "motion_response"
            })

            # If this is a motion for parenting time changes, additional deadlines may apply

        elif event_type.lower() in ["answer_filed", "response_filed"]:
            # Counterpetition/counterclaim response: 21 days after service
            counter_response_due = trigger_date + timedelta(days=21)
            deadlines.append({
                "description": "Response to counterpetition/counterclaim due",
                "deadline_date": counter_response_due,
                "rule_reference": "MCR 3.205(A)",
                "days_offset": 21,
                "deadline_type": "counter_response"
            })

        elif event_type.lower() in ["discovery_served"]:
            # Discovery responses: 28 days after service (MCR 3.210(C)(3))
            discovery_response_due = trigger_date + timedelta(days=28)
            deadlines.append({
                "description": "Discovery responses due",
                "deadline_date": discovery_response_due,
                "rule_reference": "MCR 3.210(C)(3)",
                "days_offset": 28,
                "deadline_type": "discovery_response"
            })

        # Add some common family law deadlines that apply regardless of trigger
        # These would typically be set based on case scheduling order

        return deadlines


@router.post("/v1/calendar/deadlines/calculate", response_model=DeadlineCalculationResponse)
def calculate_michigan_deadlines(request: DeadlineCalculationRequest) -> DeadlineCalculationResponse:
    """Calculate Michigan family-law deadlines based on a trigger event."""
    try:
        deadlines = MichiganFamilyLawDeadlineCalculator.calculate_deadlines(
            request.trigger_date,
            request.event_type
        )

        return DeadlineCalculationResponse(
            trigger_event=request.trigger_event,
            trigger_date=request.trigger_date,
            calculated_deadlines=deadlines
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Error calculating deadlines: {str(e)}")


@router.get("/v1/calendar/events", response_model=List[DocketEvent])
def list_calendar_events() -> List[DocketEvent]:
    """List all calendar events (docket events) for the current matter."""
    return WORKSPACE.list_docket_events()


@router.post("/v1/calendar/events", response_model=DocketEvent)
def create_calendar_event(event: DocketEventCreate) -> DocketEvent:
    """Create a new calendar event (docket event)."""
    try:
        return WORKSPACE.add_docket_event(event)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/v1/calendar/events/{event_id}")
def delete_calendar_event(event_id: str) -> Dict[str, str]:
    """Delete a calendar event by ID."""
    try:
        event_uuid = UUID(event_id)
        WORKSPACE.delete_docket_event(event_uuid)
        return {"status": "deleted"}
    except ValueError:
        raise HTTPException(status_code=400, detail="Invalid event ID format")
    except StopIteration:
        raise HTTPException(status_code=404, detail="Calendar event not found")


# Helper function to convert calculated deadlines to docket events
def create_docket_events_from_deadlines(
    deadlines: List[Dict[str, Any]],
    source: str = "deadline_calculator"
) -> List[DocketEventCreate]:
    """Convert deadline calculations to DocketEventCreate objects."""
    events = []
    for deadline in deadlines:
        event = DocketEventCreate(
            occurs_at=deadline["deadline_date"],
            title=deadline["description"],
            kind=EventKind.DEADLINE,
            detail=f"Rule: {deadline.get('rule_reference', 'N/A')}",
            source=source,
            confirmed=False
        )
        events.append(event)
    return events