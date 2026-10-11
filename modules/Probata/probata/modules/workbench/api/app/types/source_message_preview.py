"""Bounded, read-only raw message preview contracts; never an import request."""

from typing import Annotated, Literal
from pydantic import BaseModel, ConfigDict, Field, StringConstraints
from app.types.source_inspection import SourceInspectionRequest

Short = Annotated[str, StringConstraints(max_length=1024)]
PageID = Annotated[str, StringConstraints(pattern=r"^[A-Za-z0-9_-]{1,128}$")]


class SourceMessagePreviewRequest(SourceInspectionRequest):
    """Carry listing pins and one opaque page cursor, without case admission."""
    cursor: Annotated[str, StringConstraints(max_length=4096)] | None = None
    limit: Annotated[int, Field(ge=1, le=50)] = 25


class PreviewAttachment(BaseModel):
    """Describe an attachment without transferring its encoded payload."""
    model_config = ConfigDict(extra="forbid")
    content_type: Short = ""
    name: Short = ""
    encoded_bytes: Annotated[int, Field(ge=0)] = 0


class PreviewMessage(BaseModel):
    """Represent one bounded display row returned by the Go preview operation."""
    model_config = ConfigDict(extra="forbid")
    id: Short
    kind: Short
    date_ms: int | None = None
    direction: Short
    participants: Annotated[list[Short], Field(max_length=64)] = []
    body: Annotated[str, StringConstraints(max_length=16384)] = ""
    body_truncated: bool = False
    metadata_truncated: bool = False
    attachments: Annotated[list[PreviewAttachment], Field(max_length=64)] = []


class PreviewSource(BaseModel):
    """Retain the exact original locator and provider pins for a preview page."""
    source_ref: Annotated[str, StringConstraints(max_length=2048)]
    etag: Annotated[str, StringConstraints(min_length=1, max_length=512)]
    version_id: Short | None = None
    byte_length: Annotated[int, Field(ge=0)]


class SourceMessagePreviewReady(BaseModel):
    """Return at most fifty messages and one continuation, without corpus writes."""
    model_config = ConfigDict(extra="forbid")
    page_id: PageID
    status: Literal["ready"]
    source: PreviewSource
    messages: Annotated[list[PreviewMessage], Field(max_length=50)]
    next_cursor: Annotated[str, StringConstraints(max_length=4096)] | None = None
    scanned_bytes: Annotated[int, Field(ge=0, le=1048576)]
    truncation_reason: Short | None = None
    preview_only: Literal[True]


class SourceMessagePreviewPending(BaseModel):
    """Expose an opaque pending page handle, never an arbitrary upstream URL."""
    model_config = ConfigDict(extra="forbid")
    page_id: PageID
    status: Literal["pending"]
    poll_url: Annotated[str, StringConstraints(max_length=256)]


SourceMessagePreviewResponse = SourceMessagePreviewReady | SourceMessagePreviewPending
