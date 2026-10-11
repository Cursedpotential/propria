"""Typed read-only source inspection contract for immediate Workbench preview.

Byline: Codex · GPT-5.6-Sol · 2026-08-30.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints

from app.types.matter_mode import MatterMode


SourceKey = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=1024)]
OpaqueETag = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=512)]
Sha256Digest = Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]


class SourceInspectionRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    root_id: Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=64)]
    source_ref: Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=2048)]
    key: SourceKey
    expected_byte_length: Annotated[int, Field(ge=0)]
    expected_etag: OpaqueETag | None = None


class SourceVersionResponse(BaseModel):
    """Expose one selected object's exact provider version from a metadata HEAD.

    Inputs: validated selected source. Output: original locator, B2 version,
    size and ETag. Effects: none; use before context registration, not preview.
    """

    model_config = ConfigDict(extra="forbid")
    source_ref: str
    provider_version_id: Annotated[str, StringConstraints(min_length=1)]
    byte_length: Annotated[int, Field(ge=0)]
    etag: OpaqueETag


class ParserPreflight(BaseModel):
    """Non-authoritative routing hint derived only from the source filename."""

    model_config = ConfigDict(extra="forbid")

    declared_format: str
    route_label: str
    basis: Literal["filename_extension"] = "filename_extension"
    authoritative: Literal[False] = False


class SourceReaderDescriptor(BaseModel):
    """Carry selected source metadata and reader coordinates without a digest claim.

    Inputs: validated provider metadata. Output: reader fields. Effects: none.
    Shared by checksum inspection and metadata-only file selection responses.
    """
    model_config = ConfigDict(extra="forbid")

    source: str
    root_id: str
    active_root_id: str
    matter_mode: MatterMode
    source_location: str = "r2"
    bucket: str
    key: SourceKey
    source_ref: str
    name: str
    byte_length: Annotated[int, Field(ge=0)]
    etag: OpaqueETag
    last_modified: datetime | None = None
    content_type: str
    preview_text: str = ""
    preview_url: str | None = None
    parser_preflight: ParserPreflight


class SourceInspectionResponse(SourceReaderDescriptor):
    """Return the explicit small-source checksum and its preview, outside custody."""

    sha256: Sha256Digest
    digest_status: Literal["preview_only"] = "preview_only"
    preview_kind: Literal["pdf", "text", "image", "unsupported"]


class SourcePreviewResponse(SourceReaderDescriptor):
    """Describe a selected file without claiming to have read or hashed its bytes."""

    sha256: None = None
    digest_status: Literal["not_computed"] = "not_computed"
    preview_kind: Literal["pdf", "text", "image", "audio", "video", "unsupported"]
