"""Validate bounded discovery configuration and attributed feedback.

Inputs: browser choices from server datasets and descriptions. Outputs: typed
request contracts. Effects: none; choose for discovery, never global ontology edits.
"""

from typing import Annotated, Literal
from uuid import UUID

from pydantic import AwareDatetime, BaseModel, ConfigDict, Field, model_validator

from app.types.matter_mode import MatterMode

Backend = Literal["postgresql", "surrealdb"]
Name = Annotated[str, Field(min_length=1, max_length=200)]
CandidateID = Annotated[str, Field(min_length=1, max_length=200, pattern=r"^[A-Za-z0-9_.:-]+$")]
Key = Annotated[str, Field(min_length=1, max_length=512, pattern=r"^[A-Za-z0-9_.:-]+$")]
FeedbackKind = Literal[
    "relevant",
    "irrelevant",
    "wrong_person",
    "wrong_behavior",
    "reported_speech",
    "missing_context",
    "partial_match",
    "skip",
    "free_text",
]


class Scope(BaseModel):
    """Bind one actual case pair to an explicit operating policy.

    Inputs/outputs: mode and registry IDs. Effects: none; never select an alternate case.
    """

    model_config = ConfigDict(extra="forbid")
    mode: MatterMode
    matter_id: UUID
    court_case_id: UUID


class Hypothesis(BaseModel):
    """Describe one run-local behavior to discover without changing an ontology.

    Inputs/outputs: identity and description. Effects: none; pick for a run definition.
    """

    model_config = ConfigDict(extra="forbid")
    hypothesis_id: Name
    text: str = Field(min_length=1, max_length=4000)


class Filters(BaseModel):
    """Bound episode inputs and historical visibility before discovery scoring.

    Inputs/outputs: optional dates, people/platforms and perspective. Effects: none.
    Pick for case-scoped discovery rather than unrestricted corpus scans.
    """

    model_config = ConfigDict(extra="forbid")
    start_at: AwareDatetime | None = None
    end_at: AwareDatetime | None = None
    participants: list[Name] = Field(default_factory=list, max_length=32)
    platforms: list[Name] = Field(default_factory=list, max_length=32)
    perspective: Literal["as_lived", "hindsight"] = "hindsight"
    cutoff: AwareDatetime | None = None

    @model_validator(mode="after")
    def validate_dates(self):
        """Require the as-lived cutoff and an ordered optional date range; no effects."""
        if self.perspective == "as_lived" and self.cutoff is None:
            raise ValueError("As-lived discovery requires a timezone-aware cutoff")
        if self.start_at and self.end_at and self.start_at > self.end_at:
            raise ValueError("Start must precede end")
        return self


class RunRequest(BaseModel):
    """Select recorded inputs and bounded run-local hypotheses for discovery.

    Inputs: server dataset pins and configured case. Output: validated run request.
    Effects: none; actor is injected by the authenticated server, never accepted here.
    """

    model_config = ConfigDict(extra="forbid")
    run_id: UUID
    scope: Scope
    input_backend: Backend
    dataset_id: Name
    selection_revision: Name
    generation_ids: list[UUID] = Field(min_length=1, max_length=32)
    hypotheses: list[Hypothesis] = Field(min_length=1, max_length=16)
    filters: Filters = Field(default_factory=Filters)
    max_records: int = Field(default=200, ge=1, le=400, strict=True)
    max_candidates: int = Field(default=25, ge=1, le=100, strict=True)


class FeedbackRequest(BaseModel):
    """Record one owner's correction against the displayed run revision.

    Inputs: candidate, explicit category, note and optional corrections. Output:
    validated feedback. Effects: none until submitted; publication is a separate operation.
    """

    model_config = ConfigDict(extra="forbid")
    feedback_id: UUID
    candidate_id: CandidateID
    expected_revision: int = Field(ge=1, strict=True)
    kind: FeedbackKind
    labels: list[Name] = Field(default_factory=list, max_length=32)
    attributed_actor_ids: list[Name] = Field(default_factory=list, max_length=32)
    note: str = Field(default="", max_length=4000)

    @model_validator(mode="after")
    def require_note(self):
        """Require content for a free-text correction; output self, no side effects."""
        if self.kind == "free_text" and not self.note.strip():
            raise ValueError("Free-text feedback requires a note")
        return self
