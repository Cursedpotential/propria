"""Inert, source-bound context export contract (D11/SC10).

This module validates a proposed manifest. It neither reads source bytes nor
authorizes a route, evidence promotion, legal adoption, or external release.
"""

from __future__ import annotations

import hashlib
import json
import re
from dataclasses import dataclass
from datetime import UTC, datetime
from urllib.parse import unquote

_ID = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$")
_DIGEST = re.compile(r"^[0-9a-f]{64}$")
_LOCATOR = re.compile(r"^[a-z][a-z0-9+.-]*://[^\s?#]+$")
_DOMAINS = frozenset({"Vault", "CaseManagement", "KnowledgeBase", "Entities", "Code", "Triage", "Recovered", "Archive"})
_KINDS = frozenset({"family_story", "investigation_timeline", "case_timeline", "claim_support_matrix", "source_map"})
_AUDIENCES = frozenset({"private", "internal", "external"})


def _identifier(value: str, name: str) -> None:
    if not isinstance(value, str) or not _ID.fullmatch(value):
        raise ValueError(f"invalid {name}")


def _version(value: int, name: str) -> None:
    if type(value) is not int or value < 1:
        raise ValueError(f"invalid {name}")


def _digest(value: str, name: str) -> None:
    if not isinstance(value, str) or not _DIGEST.fullmatch(value):
        raise ValueError(f"invalid {name}")


def _path(value: str) -> str:
    """Return a canonical, route-relative POSIX path; reject encoded ambiguity."""
    if not isinstance(value, str) or not value or unquote(value) != value:
        raise ValueError("path must be unencoded and nonempty")
    if "\\" in value or ":" in value or "?" in value or "#" in value or any(ord(ch) < 32 for ch in value):
        raise ValueError("path contains a forbidden character")
    parts = value.split("/")
    if value.startswith("/") or any(part in {"", ".", ".."} for part in parts):
        raise ValueError("path is not a canonical relative path")
    return value


def _utc(value: datetime, name: str) -> None:
    if not isinstance(value, datetime) or value.tzinfo is None or value.utcoffset() is None:
        raise ValueError(f"{name} must be timezone-aware")


@dataclass(frozen=True)
class SourceRef:
    source_id: str
    source_version: int
    locator: str
    content_sha256: str
    status: str
    source_available_from: datetime

    def __post_init__(self) -> None:
        _identifier(self.source_id, "source_id")
        _version(self.source_version, "source_version")
        if not isinstance(self.locator, str) or not _LOCATOR.fullmatch(self.locator) or "@" in self.locator:
            raise ValueError("invalid source locator")
        _digest(self.content_sha256, "content_sha256")
        if self.status not in {"available", "unavailable", "revoked", "superseded"}:
            raise ValueError("invalid source status")
        _utc(self.source_available_from, "source_available_from")


@dataclass(frozen=True)
class NarrativeEntry:
    entry_id: str
    record_id: str
    record_version: int
    source_ids: tuple[str, ...]
    relation: str
    origin: str
    privacy: str
    time_text: str
    time_precision: str
    time_basis: str
    review_state: str
    evidence_status: str

    def __post_init__(self) -> None:
        _identifier(self.entry_id, "entry_id")
        _identifier(self.record_id, "record_id")
        _version(self.record_version, "record_version")
        if (
            type(self.source_ids) is not tuple
            or not self.source_ids
            or len(set(self.source_ids)) != len(self.source_ids)
        ):
            raise ValueError("entry needs distinct source IDs")
        for source_id in self.source_ids:
            _identifier(source_id, "source_id")
        if self.relation not in {"supports", "contradicts", "qualifies", "context", "unverified"}:
            raise ValueError("invalid claim-source relation")
        if self.origin not in {"source", "extraction", "inference", "recollection", "hypothesis"}:
            raise ValueError("invalid entry origin")
        if self.privacy not in {"ordinary", "private_strategy"}:
            raise ValueError("invalid entry privacy")
        if not self.time_text or self.time_precision not in {"exact", "approximate", "interval", "undated"}:
            raise ValueError("invalid time description")
        if self.time_basis not in {"event", "message", "source_created", "acquired", "unknown"}:
            raise ValueError("invalid time basis")
        if self.review_state not in {"unreviewed", "reviewed", "disputed"}:
            raise ValueError("invalid review state")
        if self.evidence_status not in {"not_promoted", "promoted", "revoked", "unknown"}:
            raise ValueError("invalid evidence status")


@dataclass(frozen=True)
class PackageFile:
    path: str
    sha256: str
    source_id: str | None = None

    def __post_init__(self) -> None:
        _path(self.path)
        _digest(self.sha256, "file sha256")
        if self.source_id is not None:
            _identifier(self.source_id, "file source_id")


@dataclass(frozen=True)
class ApprovedRoute:
    route_id: str
    route_map_version: int
    output_kind: str
    scope_id: str
    domain: str
    relative_directory: str
    approval_ref: str
    effective_at: datetime

    def __post_init__(self) -> None:
        _identifier(self.route_id, "route_id")
        _version(self.route_map_version, "route_map_version")
        _identifier(self.scope_id, "scope_id")
        _identifier(self.approval_ref, "approval_ref")
        if self.output_kind not in _KINDS or self.domain not in _DOMAINS:
            raise ValueError("unsupported route kind or domain")
        _path(self.relative_directory)
        _utc(self.effective_at, "effective_at")


@dataclass(frozen=True)
class ContextExportManifest:
    schema_version: int
    export_id: str
    version: int
    predecessor_digest: str | None
    case_id: str
    creator_id: str
    purpose: str
    generator_version: str
    template_version: str
    redaction_profile: str
    audience: str
    output_kind: str
    snapshot_at: datetime
    time_mode: str
    as_lived_cutoff: datetime | None
    route_id: str
    route_map_version: int
    destination: str
    requested_entry_ids: tuple[str, ...]
    sources: tuple[SourceRef, ...]
    entries: tuple[NarrativeEntry, ...]
    requested_source_copies: tuple[str, ...]
    files: tuple[PackageFile, ...]
    offline_links: tuple[str, ...]
    excluded_source_ids: tuple[str, ...]
    release_state: str = "prepared_unreleased"


def validate_manifest(manifest: ContextExportManifest, route: ApprovedRoute) -> str:
    """Validate the pinned proposal and return its deterministic SHA-256 digest.

    The caller must independently verify the approved route, source readback,
    physical path containment (including symlinks), output nonexistence, file
    bytes/rendered links, and any human external-release authorization.
    """
    _version(manifest.schema_version, "schema_version")
    if manifest.schema_version != 1:
        raise ValueError("unsupported schema version")
    _identifier(manifest.export_id, "export_id")
    _version(manifest.version, "version")
    if manifest.version == 1 and manifest.predecessor_digest is not None:
        raise ValueError("first version has no predecessor")
    if manifest.version > 1:
        _digest(manifest.predecessor_digest, "predecessor_digest")
    for name in (
        "case_id",
        "creator_id",
        "purpose",
        "generator_version",
        "template_version",
        "redaction_profile",
        "route_id",
    ):
        _identifier(getattr(manifest, name), name)
    if manifest.audience not in _AUDIENCES or manifest.output_kind not in _KINDS:
        raise ValueError("unsupported audience or output kind")
    _utc(manifest.snapshot_at, "snapshot_at")
    if manifest.time_mode not in {"as_lived", "hindsight"}:
        raise ValueError("invalid time mode")
    if manifest.time_mode == "as_lived":
        _utc(manifest.as_lived_cutoff, "as_lived_cutoff")
        if manifest.as_lived_cutoff > manifest.snapshot_at:
            raise ValueError("cutoff exceeds snapshot")
    elif manifest.as_lived_cutoff is not None:
        raise ValueError("hindsight must not carry as-lived cutoff")
    _version(manifest.route_map_version, "route_map_version")
    if (manifest.route_id, manifest.route_map_version, manifest.output_kind, manifest.case_id) != (
        route.route_id,
        route.route_map_version,
        route.output_kind,
        route.scope_id,
    ):
        raise ValueError("route binding mismatch")
    if route.effective_at > manifest.snapshot_at:
        raise ValueError("route not yet effective")
    destination = _path(manifest.destination)
    expected_prefix = f"{route.domain}/{route.relative_directory}/{manifest.export_id}/v{manifest.version}"
    if destination != expected_prefix:
        raise ValueError("destination is not fresh versioned approved route")
    if manifest.release_state != "prepared_unreleased":
        raise ValueError("this contract cannot authorize release")
    for name in (
        "requested_entry_ids",
        "sources",
        "entries",
        "requested_source_copies",
        "files",
        "offline_links",
        "excluded_source_ids",
    ):
        if type(getattr(manifest, name)) is not tuple:
            raise ValueError(f"{name} must be an immutable tuple")
    if any(type(source) is not SourceRef for source in manifest.sources):
        raise ValueError("invalid source item")
    if any(type(entry) is not NarrativeEntry for entry in manifest.entries):
        raise ValueError("invalid entry item")
    if any(type(file) is not PackageFile for file in manifest.files):
        raise ValueError("invalid file item")
    source_ids = [source.source_id for source in manifest.sources]
    if not source_ids or len(set(source_ids)) != len(source_ids):
        raise ValueError("sources must be nonempty and distinct")
    if not manifest.requested_entry_ids or len(set(manifest.requested_entry_ids)) != len(manifest.requested_entry_ids):
        raise ValueError("requested entries must be nonempty and distinct")
    for entry_id in manifest.requested_entry_ids:
        _identifier(entry_id, "requested_entry_id")
    if len({entry.entry_id for entry in manifest.entries}) != len(manifest.entries):
        raise ValueError("duplicate entry")
    if set(manifest.requested_entry_ids) != {entry.entry_id for entry in manifest.entries}:
        raise ValueError("selected entries not represented exactly")
    source_by_id = {source.source_id: source for source in manifest.sources}
    if manifest.time_mode == "as_lived" and any(
        source.source_available_from > manifest.as_lived_cutoff for source in manifest.sources
    ):
        raise ValueError("as-lived selection contains future source")
    if any(source.source_available_from > manifest.snapshot_at for source in manifest.sources):
        raise ValueError("selection contains source beyond snapshot")
    for entry in manifest.entries:
        if any(source_id not in source_by_id for source_id in entry.source_ids):
            raise ValueError("entry has missing source")
        if manifest.audience != "private" and entry.privacy == "private_strategy":
            raise ValueError("private strategy cannot enter shared output")
    excluded = set(manifest.excluded_source_ids)
    if len(excluded) != len(manifest.excluded_source_ids) or not excluded <= source_by_id.keys():
        raise ValueError("invalid excluded source list")
    unavailable = {source.source_id for source in manifest.sources if source.status != "available"}
    if unavailable != excluded:
        raise ValueError("unavailable sources must be explicitly excluded")
    copies = set(manifest.requested_source_copies)
    if len(copies) != len(manifest.requested_source_copies) or not copies <= source_by_id.keys() - excluded:
        raise ValueError("invalid requested source copies")
    paths = [file.path for file in manifest.files]
    if not paths or len(set(paths)) != len(paths):
        raise ValueError("files must be nonempty and distinct")
    for file in manifest.files:
        if file.source_id is not None and file.source_id not in copies:
            raise ValueError("unrequested source copy")
    if {file.source_id for file in manifest.files if file.source_id is not None} != copies:
        raise ValueError("requested source copy missing")
    if len(set(manifest.offline_links)) != len(manifest.offline_links):
        raise ValueError("duplicate offline link")
    for link in manifest.offline_links:
        if _path(link) not in paths:
            raise ValueError("offline link has no package member")
    payload = {
        "schema_version": manifest.schema_version,
        "export_id": manifest.export_id,
        "version": manifest.version,
        "predecessor_digest": manifest.predecessor_digest,
        "case_id": manifest.case_id,
        "creator_id": manifest.creator_id,
        "purpose": manifest.purpose,
        "generator_version": manifest.generator_version,
        "template_version": manifest.template_version,
        "redaction_profile": manifest.redaction_profile,
        "audience": manifest.audience,
        "output_kind": manifest.output_kind,
        "snapshot_at": manifest.snapshot_at.astimezone(UTC).isoformat(),
        "time_mode": manifest.time_mode,
        "as_lived_cutoff": manifest.as_lived_cutoff.astimezone(UTC).isoformat() if manifest.as_lived_cutoff else None,
        "route_id": manifest.route_id,
        "route_map_version": manifest.route_map_version,
        "route_binding": {
            "domain": route.domain,
            "relative_directory": route.relative_directory,
            "approval_ref": route.approval_ref,
            "effective_at": route.effective_at.astimezone(UTC).isoformat(),
        },
        "destination": manifest.destination,
        "requested_entry_ids": manifest.requested_entry_ids,
        "sources": [
            {**source.__dict__, "source_available_from": source.source_available_from.astimezone(UTC).isoformat()}
            for source in manifest.sources
        ],
        "entries": [entry.__dict__ for entry in manifest.entries],
        "requested_source_copies": manifest.requested_source_copies,
        "files": [file.__dict__ for file in manifest.files],
        "offline_links": manifest.offline_links,
        "excluded_source_ids": manifest.excluded_source_ids,
        "release_state": manifest.release_state,
    }
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()
