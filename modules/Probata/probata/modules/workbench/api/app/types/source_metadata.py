"""Metadata screen read model and the owner's append-only metadata corrections.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

The engine answers what the platform durably recorded for one file of a run
(registration, retained original, every source_metadata row verbatim, members,
projected attachments, hash receipts, corrections). The BFF adds the sidecars
that sit beside the file in its source root or its catalog folder. Observed
values are never written; a correction is an attributed overlay revision.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated, Any, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, model_validator

from app.types.matter_mode import MatterMode

Sha256Digest = Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]
Text = Annotated[str, StringConstraints(max_length=4096)]
FieldKey = Annotated[
    str,
    StringConstraints(min_length=3, max_length=512, pattern=r"^[a-z][a-z_]{0,31}:[^\x00-\x1f\x7f]+$"),
]


class ObjectFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    object_ref: str
    storage_class: str
    sha256: Sha256Digest
    byte_length: Annotated[int, Field(ge=0)]
    immutable_at: datetime


class SourceFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    source_version_ref: str
    source_key: Text = ""
    provenance_class: Text = ""
    declared_format: Text = ""
    original_filename: Text | None = None
    acquired_at: datetime | None = None
    status: Text = ""
    matter_id: str | None = None
    court_case_id: str | None = None
    source_context_ref: str | None = None
    original: ObjectFacts | None = None


class MetadataRow(BaseModel):
    """One context.source_metadata row, native JSON kept verbatim."""

    model_config = ConfigDict(extra="ignore")

    metadata_ref: str
    metadata_class: Literal["filesystem", "embedded", "container", "media_tool", "record_native"]
    extractor_id: Text
    extractor_version: Text | None = None
    generated_at: datetime
    receipt_ref: str
    fields: Any = None
    fields_bytes: Annotated[int, Field(ge=0)] = 0


class RetainedMember(BaseModel):
    model_config = ConfigDict(extra="ignore")

    object_ref: str
    role: str
    parent_object_ref: str | None = None
    sha256: Sha256Digest
    byte_length: Annotated[int, Field(ge=0)]
    storage_class: str
    member_locator: dict[str, Any] = Field(default_factory=dict)


class AttachmentFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    sha256: Sha256Digest
    filename: Text | None = None
    media_type: Text | None = None
    byte_length: Annotated[int, Field(ge=0)] | None = None
    source_locator_ref: Text = ""
    message_ids: list[str] = Field(default_factory=list)


class HashReceipt(BaseModel):
    model_config = ConfigDict(extra="ignore")

    hash_kind: str
    construction: str
    digest: Sha256Digest
    computed_at: datetime
    computed_by: str


class MetadataCorrection(BaseModel):
    model_config = ConfigDict(extra="ignore")

    correction_ref: str
    subject_sha256: Sha256Digest
    field_key: str
    revision: Annotated[int, Field(ge=1)]
    supersedes_ref: str | None = None
    action: Literal["correct", "retract"]
    source_value: Any = None
    corrected_value: Any = None
    change_reason: str
    actor_username: str
    receipt_ref: str
    recorded_at: datetime


class EngineMetadataView(BaseModel):
    """Exactly what the engine returns for GET .../metadata."""

    model_config = ConfigDict(extra="ignore")

    preview_handle: str
    request_id: str
    source_ref: str
    subject_kind: Literal["source", "member", "attachment"]
    subject_sha256: Sha256Digest | Literal[""] = ""
    source: SourceFacts | None = None
    metadata: list[MetadataRow]
    metadata_truncated: bool = False
    record_metadata_count: Annotated[int, Field(ge=0)] = 0
    members: list[RetainedMember]
    members_truncated: bool = False
    attachment: AttachmentFacts | None = None
    attachment_count: Annotated[int, Field(ge=0)] = 0
    hashes: list[HashReceipt]
    corrections: list[MetadataCorrection]
    corrections_available: bool = False


SidecarKind = Literal[
    "takeout_json", "json", "xmp", "apple_aae", "owner_sidecar_md", "owner_extraction_md", "companion"
]


class SidecarField(BaseModel):
    path: str
    value: Any = None


class Sidecar(BaseModel):
    """A file that accompanies the subject, with where it was found."""

    name: str
    key: str
    kind: SidecarKind
    found_via: Literal["beside_object", "catalog_folder"]
    byte_length: Annotated[int, Field(ge=0)] | None = None
    fields: list[SidecarField] = Field(default_factory=list)
    fields_truncated: bool = False
    text: str | None = None
    text_truncated: bool = False
    unread_reason: str | None = None


class SidecarConflict(BaseModel):
    """A sidecar value that disagrees with the file's own value. Never merged."""

    topic: Literal["capture_time", "gps"]
    sidecar_key: str
    sidecar_path: str
    sidecar_value: Any = None
    file_field: str
    file_value: Any = None
    detail: str


LookupState = Literal["ok", "unavailable", "not_configured", "not_applicable"]


class SidecarLookup(BaseModel):
    beside_object: LookupState
    catalog_folder: LookupState


class MetadataScreenResponse(EngineMetadataView):
    """The metadata screen: the engine's durable facts plus sidecars."""

    sidecars: list[Sidecar] = Field(default_factory=list)
    sidecar_conflicts: list[SidecarConflict] = Field(default_factory=list)
    sidecar_lookup: SidecarLookup
    matter_mode: MatterMode


class MetadataCorrectionRequest(BaseModel):
    """One owner correction as the browser sends it (actor comes from the session)."""

    model_config = ConfigDict(extra="forbid")

    subject_sha256: Sha256Digest
    field_key: FieldKey
    supersedes_ref: Annotated[str, StringConstraints(max_length=64)] = ""
    action: Literal["correct", "retract"]
    source_value: Any = None
    corrected_value: Any = None
    change_reason: Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=4000)]

    @model_validator(mode="after")
    def value_matches_action(self) -> MetadataCorrectionRequest:
        if self.action == "correct" and self.corrected_value is None:
            raise ValueError("a correction needs a corrected_value; use retract to withdraw one")
        if self.action == "retract" and (self.corrected_value is not None or not self.supersedes_ref):
            raise ValueError("a retract carries no corrected_value and names the correction it withdraws")
        return self


class MetadataCorrectionReceipt(BaseModel):
    model_config = ConfigDict(extra="ignore")

    correction_ref: str
    receipt_ref: str
    content_digest: str
    revision: Annotated[int, Field(ge=1)]
    recorded_at: datetime
    matter_mode: MatterMode
