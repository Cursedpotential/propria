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
_WINDOWS_RESERVED = frozenset({"CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$"}) | frozenset(
    f"{prefix}{number}" for prefix in ("COM", "LPT") for number in range(1, 10)
)


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
    if any(ch in value for ch in '\\:?#%<>|*"') or any(ord(ch) < 32 for ch in value):
        raise ValueError("path contains a forbidden character")
    parts = value.split("/")
    if value.startswith("/") or any(part in {"", ".", ".."} for part in parts):
        raise ValueError("path is not a canonical relative path")
    if any(part.endswith((".", " ")) or part.split(".", 1)[0].upper() in _WINDOWS_RESERVED for part in parts):
        raise ValueError("path has a Windows alias or reserved component")
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
    entry_available_from: datetime

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
        if self.source_ids != tuple(sorted(self.source_ids)):
            raise ValueError("entry source IDs must be sorted")
        if self.relation not in {"supports", "contradicts", "qualifies", "context", "unverified"}:
            raise ValueError("invalid claim-source relation")
        if self.origin not in {"source", "extraction", "inference", "recollection", "hypothesis"}:
            raise ValueError("invalid entry origin")
        if self.privacy not in {"ordinary", "private_strategy"}:
            raise ValueError("invalid entry privacy")
        if (
            type(self.time_text) is not str
            or not 1 <= len(self.time_text) <= 512
            or self.time_text != self.time_text.strip()
            or any(ord(ch) < 32 for ch in self.time_text)
            or self.time_precision not in {"exact", "approximate", "interval", "undated"}
        ):
            raise ValueError("invalid time description")
        if self.time_basis not in {"event", "message", "source_created", "acquired", "unknown"}:
            raise ValueError("invalid time basis")
        if self.review_state not in {"unreviewed", "reviewed", "disputed"}:
            raise ValueError("invalid review state")
        if self.evidence_status not in {"not_promoted", "promoted", "revoked", "unknown"}:
            raise ValueError("invalid evidence status")
        if self.time_precision == "undated" and (self.time_text != "undated" or self.time_basis != "unknown"):
            raise ValueError("undated time must be explicit and unknown-basis")
        if self.time_precision != "undated" and self.time_basis == "unknown":
            raise ValueError("dated time needs a known basis")
        _utc(self.entry_available_from, "entry_available_from")


@dataclass(frozen=True)
class PackageFile:
    path: str
    sha256: str
    kind: str
    source_id: str | None = None

    def __post_init__(self) -> None:
        _path(self.path)
        _digest(self.sha256, "file sha256")
        if self.kind not in {"representation", "raw_source_copy"}:
            raise ValueError("invalid package file kind")
        if self.kind == "raw_source_copy":
            if self.source_id is None:
                raise ValueError("raw source copy needs a source ID")
            _identifier(self.source_id, "file source_id")
        elif self.source_id is not None:
            raise ValueError("representation cannot claim raw source identity")


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
        predecessor_digest = manifest.predecessor_digest
        if predecessor_digest is None:
            raise ValueError("later version needs a predecessor digest")
        _digest(predecessor_digest, "predecessor_digest")
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
    cutoff = manifest.as_lived_cutoff
    if manifest.time_mode == "as_lived":
        if cutoff is None:
            raise ValueError("as-lived mode needs a cutoff")
        _utc(cutoff, "as_lived_cutoff")
        if cutoff > manifest.snapshot_at:
            raise ValueError("cutoff exceeds snapshot")
    elif cutoff is not None:
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
    if source_ids != sorted(source_ids):
        raise ValueError("sources must be sorted by source ID")
    if not manifest.requested_entry_ids or len(set(manifest.requested_entry_ids)) != len(manifest.requested_entry_ids):
        raise ValueError("requested entries must be nonempty and distinct")
    if manifest.requested_entry_ids != tuple(sorted(manifest.requested_entry_ids)):
        raise ValueError("requested entry IDs must be sorted")
    for entry_id in manifest.requested_entry_ids:
        _identifier(entry_id, "requested_entry_id")
    if len({entry.entry_id for entry in manifest.entries}) != len(manifest.entries):
        raise ValueError("duplicate entry")
    if set(manifest.requested_entry_ids) != {entry.entry_id for entry in manifest.entries}:
        raise ValueError("selected entries not represented exactly")
    source_by_id = {source.source_id: source for source in manifest.sources}
    if cutoff is not None and any(source.source_available_from > cutoff for source in manifest.sources):
        raise ValueError("as-lived selection contains future source")
    if any(source.source_available_from > manifest.snapshot_at for source in manifest.sources):
        raise ValueError("selection contains source beyond snapshot")
    if any(entry.entry_available_from > manifest.snapshot_at for entry in manifest.entries):
        raise ValueError("entry was not available at snapshot")
    if cutoff is not None and any(entry.entry_available_from > cutoff for entry in manifest.entries):
        raise ValueError("as-lived selection contains future entry")
    excluded = set(manifest.excluded_source_ids)
    if len(excluded) != len(manifest.excluded_source_ids) or not excluded <= source_by_id.keys():
        raise ValueError("invalid excluded source list")
    if manifest.excluded_source_ids != tuple(sorted(manifest.excluded_source_ids)):
        raise ValueError("excluded source IDs must be sorted")
    for entry in manifest.entries:
        if any(source_id not in source_by_id for source_id in entry.source_ids):
            raise ValueError("entry has missing source")
        if any(source_id in excluded for source_id in entry.source_ids):
            raise ValueError("entry cites excluded source")
        if manifest.audience != "private" and entry.privacy == "private_strategy":
            raise ValueError("private strategy cannot enter shared output")
    unavailable = {source.source_id for source in manifest.sources if source.status != "available"}
    if unavailable != excluded:
        raise ValueError("unavailable sources must be explicitly excluded")
    copies = set(manifest.requested_source_copies)
    if len(copies) != len(manifest.requested_source_copies) or not copies <= source_by_id.keys() - excluded:
        raise ValueError("invalid requested source copies")
    if manifest.requested_source_copies != tuple(sorted(manifest.requested_source_copies)):
        raise ValueError("requested source copies must be sorted")
    paths = [file.path for file in manifest.files]
    if not paths or len(set(paths)) != len(paths):
        raise ValueError("files must be nonempty and distinct")
    if paths != sorted(paths) or len({path.casefold() for path in paths}) != len(paths):
        raise ValueError("file paths must be sorted and case-insensitively distinct")
    raw_source_ids = []
    for file in manifest.files:
        if file.kind == "raw_source_copy":
            if file.source_id not in copies or file.sha256 != source_by_id[file.source_id].content_sha256:
                raise ValueError("unrequested or digest-mismatched raw source copy")
            raw_source_ids.append(file.source_id)
    if len(set(raw_source_ids)) != len(raw_source_ids) or set(raw_source_ids) != copies:
        raise ValueError("requested source copy missing")
    if len(set(manifest.offline_links)) != len(manifest.offline_links):
        raise ValueError("duplicate offline link")
    if manifest.offline_links != tuple(sorted(manifest.offline_links)):
        raise ValueError("offline links must be sorted")
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
        "entries": [
            {**entry.__dict__, "entry_available_from": entry.entry_available_from.astimezone(UTC).isoformat()}
            for entry in manifest.entries
        ],
        "requested_source_copies": manifest.requested_source_copies,
        "files": [file.__dict__ for file in manifest.files],
        "offline_links": manifest.offline_links,
        "excluded_source_ids": manifest.excluded_source_ids,
        "release_state": manifest.release_state,
    }
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()
