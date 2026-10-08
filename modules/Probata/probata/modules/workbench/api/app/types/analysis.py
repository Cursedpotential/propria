"""Validate bounded analysis queries and their exact source-cited results.

Inputs: recorded projection pins and perspective. Outputs: typed Go contracts.
Effects: none; use for reading analysis, never ingestion or evidence promotion.
"""
from typing import Annotated, Literal
from uuid import UUID

from pydantic import AwareDatetime, BaseModel, ConfigDict, Field, model_validator

Identifier = Annotated[str, Field(min_length=1, max_length=240, pattern=r"^[^\s\x00]+$")]
Digest = Annotated[str, Field(pattern=r"^[0-9a-f]{64}$")]


class ProjectionPin(BaseModel):
    """Identify one recorded completed graph generation without selecting a latest version.

    Inputs/outputs: revision, approval digest and projection hash. Effects: none.
    Use unchanged pins from the graph catalog when requesting a query page.
    """
    model_config = ConfigDict(extra="forbid")
    access_policy_id: Identifier
    approved_revision_id: Identifier
    approval_digest: Digest
    projection_generation_id: Identifier
    projection_hash: Digest


class AnalysisQuery(ProjectionPin):
    """Select a recorded projection and an explicit historical reading perspective.

    Inputs: pins, perspective, timezone-aware cutoff and bounded cursor/limit.
    Output: validated query; effects: none. Use before the Temporal query start.
    """
    perspective: Literal["as_lived", "hindsight"]
    horizon: AwareDatetime | None = None
    limit: int = Field(default=25, ge=1, le=100, strict=True)
    cursor: str = Field(default="", max_length=2048)

    @model_validator(mode="after")
    def historical_cutoff(self):
        """Require a real cutoff for as-lived reads and omit it for hindsight."""
        if self.perspective == "as_lived" and self.horizon is None:
            raise ValueError("as-lived reading requires a timezone-aware cutoff")
        if self.perspective == "hindsight" and self.horizon is not None:
            raise ValueError("hindsight reading does not take an as-lived cutoff")
        return self


class ProjectionSourcePin(BaseModel):
    """Retain the original source coordinates supplied by a completed graph checkpoint.

    Inputs/outputs: existing object/version/hash pins. Effects: none; use for
    snapshot labels and citations, never to treat a current object as that version.
    """
    model_config = ConfigDict(extra="forbid")
    source_id: Identifier
    source_version_id: Identifier
    source_object_id: Identifier | Literal[""]
    source_object_sha256: Digest
    source_object_uri: str = Field(min_length=1, max_length=2048)


    @model_validator(mode="after")
    def native_original_identity(self):
        """Permit no retained object only for real registered native B2 sources.

        Inputs: graph source coordinates. Output: unchanged honest pin. Effects:
        none; source and version remain actual UUIDs, never substitute object IDs.
        """
        if self.source_object_id == "":
            UUID(self.source_id)
            UUID(self.source_version_id)
            if not self.source_object_uri.startswith("b2://"):
                raise ValueError("native original needs its actual B2 locator")
        return self


class ProjectionSnapshot(ProjectionPin):
    """Describe one real completed checkpoint and its retained original sources.

    Inputs/outputs: Go catalog entry. Effects: none; use for explicit snapshot
    selection rather than inferring completion from working candidate rows.
    """
    matter_id: Identifier
    court_case_id: Identifier
    claim_count: int = Field(ge=0, strict=True)
    completed_at: AwareDatetime
    source_pins: list[ProjectionSourcePin] = Field(min_length=1, max_length=100)


class AnalysisArtifact(BaseModel):
    """Pin the exact version and hash of a bounded query-result object.

    Inputs/outputs: Go artifact reference. Effects: none; use only after status.
    """
    model_config = ConfigDict(extra="forbid")
    uri: str = Field(min_length=1, max_length=2048)
    version_id: str = Field(min_length=1, max_length=512)
    sha256: Digest
    bytes: int = Field(ge=1, le=8 * 1024 * 1024, strict=True)


class AnalysisResult(ProjectionPin):
    """Describe a completed query page without embedding findings in workflow status.

    Inputs/outputs: exact case, projection, artifact and counts. Effects: none.
    Use to verify the separately retrieved content before displaying it.
    """
    artifact: AnalysisArtifact
    matter_id: Identifier
    court_case_id: Identifier
    claim_count: int = Field(ge=0, le=100, strict=True)
    perspective: Literal["as_lived", "hindsight"]
    has_more: bool
    next_cursor: str = Field(default="", max_length=2048)


class AnalysisClaim(BaseModel):
    """Retain a finding's original object, record and approval provenance.

    Inputs/outputs: one Go claim. Effects: none; use as a citation, not evidence.
    """
    model_config = ConfigDict(extra="forbid")
    id: Identifier
    kind: Literal["entity_mention", "statement", "event_account"]
    text: str = Field(min_length=1, max_length=1024 * 1024)
    matter_id: Identifier
    court_case_id: Identifier
    approved_revision_id: Identifier
    approval_digest: Digest
    control_generation_id: Identifier
    candidate_id: Identifier
    candidate_sha256: Digest
    source_id: Identifier
    source_version_id: Identifier
    source_object_id: Identifier | Literal[""]
    source_object_uri: str = Field(min_length=1, max_length=2048)
    source_sha256: Digest
    record_id: Identifier
    record_sha256: Digest
    predicate: str | None = None
    native_json_pointer: str | None = None
    native_span_start: int | None = Field(default=None, ge=0, strict=True)
    native_span_end: int | None = Field(default=None, ge=0, strict=True)
    native_span_unit: Literal["unicode_codepoint"] | None = None
    native_span_sha256: Digest | None = None
    occurred_at: AwareDatetime | None = None
    source_available_from: AwareDatetime | None = None
    approved_at: AwareDatetime
    approved_by: Identifier


    @model_validator(mode="after")
    def native_original_locator(self):
        """Require the native citation when no retained object exists.

        Inputs: registered source UUIDs, original B2 URI/hash and native span.
        Output: unchanged cited claim. Effects: none; never normalize AI messages.
        """
        if self.source_object_id == "":
            UUID(self.source_id)
            UUID(self.source_version_id)
            if (not self.source_object_uri.startswith("b2://")
                    or self.native_span_unit != "unicode_codepoint"
                    or self.native_span_start is None or self.native_span_end is None
                    or self.native_span_end <= self.native_span_start
                    or self.native_span_sha256 is None):
                raise ValueError("native original requires its genuine source and span citation")
        return self


class AnalysisContent(AnalysisQuery):
    """Expose only verified bounded findings from one actor-bound query artifact.

    Inputs/outputs: artifact body and exact source citations. Effects: none.
    Use for Read results after object version/hash and scope validation.
    """
    actor_subject_uid: Identifier
    matter_id: Identifier
    court_case_id: Identifier
    cursor: str = Field(default="", exclude=True)
    request_cursor: str = Field(default="", max_length=2048)
    has_more: bool
    next_cursor: str = Field(default="", max_length=2048)
    claims: list[AnalysisClaim] = Field(max_length=100)
