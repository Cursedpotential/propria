"""Metadata screen for one file of a Review run, and the owner's corrections.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

The engine answers what the platform durably recorded (it extracts nothing);
this adapter proves TEST/REAL ownership first, validates the answer fail-closed,
adds the file's sidecars, and reports any sidecar value that disagrees with the
file's own value. Corrections go to the engine as attributed, append-only
overlays with an idempotency key derived from the exact submission.
"""

from __future__ import annotations

import asyncio
import hashlib
import json
from datetime import UTC, datetime, timedelta
from typing import Any

from app.service.proffer import ProfferError, _json_payload, _mode_payload, _request, _require_mode, _validated
from app.service.source_sidecars import find_sidecars
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferDecisionActor
from app.types.source_metadata import (
    EngineMetadataView,
    MetadataCorrectionReceipt,
    MetadataCorrectionRequest,
    MetadataRow,
    MetadataScreenResponse,
    Sidecar,
    SidecarConflict,
    SidecarLookup,
)

CAPTURE_TAGS = ("DateTimeOriginal", "SubSecDateTimeOriginal", "CreateDate", "CreationDate", "MediaCreateDate")
NAIVE_TOLERANCE = timedelta(hours=26)  # a device-local wall clock can sit up to ±14 h from UTC
ZONED_TOLERANCE = timedelta(hours=1)
GPS_TOLERANCE = 0.001


def _file_values(rows: list[MetadataRow]):
    """(field key, tag, value) for every embedded / media-tool value the file itself carries."""
    for row in rows:
        if row.metadata_class not in ("embedded", "media_tool") or not isinstance(row.fields, dict):
            continue
        for key, value in row.fields.items():
            yield f"{row.metadata_class}:{key}", str(key).rsplit(":", 1)[-1], value


def _exif_time(value: Any) -> datetime | None:
    text = str(value).strip()
    for pattern in ("%Y:%m:%d %H:%M:%S%z", "%Y:%m:%d %H:%M:%S", "%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%dT%H:%M:%S"):
        try:
            return datetime.strptime(text[:25].replace("Z", "+0000"), pattern)
        except ValueError:
            continue
    return None


def _number(value: Any) -> float | None:
    try:
        return float(str(value).strip())
    except ValueError:
        return None


def sidecar_conflicts(sidecars: list[Sidecar], rows: list[MetadataRow]) -> list[SidecarConflict]:
    """Capture time and GPS disagreements between a Takeout sidecar and the file's own values."""
    file_values = list(_file_values(rows))
    conflicts: list[SidecarConflict] = []
    for sidecar in sidecars:
        if sidecar.kind != "takeout_json":
            continue
        fields = {field.path: field.value for field in sidecar.fields}
        stamp = fields.get("photoTakenTime.timestamp")
        taken = datetime.fromtimestamp(int(stamp), UTC) if str(stamp or "").isdigit() else None
        for field_key, tag, value in file_values:
            if taken is None or tag not in CAPTURE_TAGS:
                continue
            own = _exif_time(value)
            if own is None:
                continue
            zoned = own.tzinfo is not None
            gap = abs((own.astimezone(UTC) if zoned else own) - (taken if zoned else taken.replace(tzinfo=None)))
            if gap > (ZONED_TOLERANCE if zoned else NAIVE_TOLERANCE):
                conflicts.append(
                    SidecarConflict(
                        topic="capture_time",
                        sidecar_key=sidecar.key,
                        sidecar_path="photoTakenTime.timestamp",
                        sidecar_value=stamp,
                        file_field=field_key,
                        file_value=value,
                        detail=f"differ by {gap.total_seconds() / 3600:.1f} hours",
                    )
                )
        for axis, tag in (("latitude", "GPSLatitude"), ("longitude", "GPSLongitude")):
            theirs = _number(fields.get(f"geoData.{axis}"))
            if theirs is None or theirs == 0.0:
                continue
            for field_key, own_tag, value in file_values:
                ours = _number(value) if own_tag == tag else None
                if ours is not None and abs(ours - theirs) > GPS_TOLERANCE:
                    conflicts.append(
                        SidecarConflict(
                            topic="gps",
                            sidecar_key=sidecar.key,
                            sidecar_path=f"geoData.{axis}",
                            sidecar_value=theirs,
                            file_field=field_key,
                            file_value=value,
                            detail=f"{axis} differs by {abs(ours - theirs):.4f}°",
                        )
                    )
    return conflicts


async def metadata_screen(
    preview_handle: str, *, mode: MatterMode, subject_sha256: str | None = None
) -> MetadataScreenResponse:
    """Every metadata fact the platform holds for one file of the run, plus its sidecars."""
    await _require_mode(preview_handle, mode)
    path = f"/reference-import/previews/{preview_handle}/metadata"
    if subject_sha256:
        path += f"?subject_sha256={subject_sha256}"
    response = await _request("GET", path)
    view = _validated(EngineMetadataView, _json_payload(response, "metadata view"), "metadata view")
    if view.preview_handle != preview_handle:
        raise ProfferError("Proffer metadata view correlation failed", 502)
    if subject_sha256 and view.subject_sha256 != subject_sha256:
        raise ProfferError("Proffer metadata view answered for a different file", 502)
    if view.subject_kind == "source":
        sidecars, lookup = await asyncio.to_thread(find_sidecars, view.source_ref)
    else:
        sidecars, lookup = [], SidecarLookup(beside_object="not_applicable", catalog_folder="not_applicable")
    return MetadataScreenResponse(
        **view.model_dump(),
        sidecars=sidecars,
        sidecar_conflicts=sidecar_conflicts(sidecars, view.metadata),
        sidecar_lookup=lookup,
        matter_mode=mode,
    )


async def correct_metadata(
    preview_handle: str,
    request: MetadataCorrectionRequest,
    actor: ProfferDecisionActor,
    *,
    mode: MatterMode,
) -> MetadataCorrectionReceipt:
    """Append one attributed correction; the observed value is never written."""
    await _require_mode(preview_handle, mode)
    body = request.model_dump(mode="json")
    canonical = json.dumps(body, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    key = hashlib.sha256(f"{preview_handle}\x00{actor.subject_uid}\x00{canonical}".encode()).hexdigest()
    response = await _request(
        "POST",
        f"/reference-import/previews/{preview_handle}/metadata/corrections",
        json=body,
        headers={
            "X-authentik-uid": actor.subject_uid,
            "X-authentik-username": actor.username,
            "Idempotency-Key": f"metadata-correction:{key}",
        },
    )
    return _validated(
        MetadataCorrectionReceipt,
        _mode_payload(_json_payload(response, "metadata correction receipt"), "metadata correction receipt", mode),
        "metadata correction receipt",
    )
