"""Hand-marked source units, and the Takeout proposal that must be confirmed.

Byline: Claude Code · Opus 5 · 2026-09-22.

Owner, 2026-09-22 09:00: "i need the metadata in view and some way to signify
it's a unit of some kind". The catalog detects units it recognises; this is the
other half — the owner marking a folder himself. A Takeout is never recorded on
one click: its parts are proposed, gaps are shown, and it is written only on an
explicit confirm (bulk-intake owner requirement 7).
"""

from __future__ import annotations

import re
from datetime import datetime, timezone

from app.repo.source_unit_marks import UnitMarkStoreError, read_all, storage_description, write_all
from app.types.source_unit_marks import (
    SourceUnitMark,
    SourceUnitMarkList,
    SourceUnitMarkRequest,
    SourceUnitProposal,
    SourceUnitProposalRequest,
    TakeoutPartProposal,
)

# takeout-<stamp>-<job>-<NNN>, the export naming the corpus actually uses.
TAKEOUT_PART = re.compile(r"^takeout-(?P<stamp>[0-9a-z]+)-(?P<job>[0-9a-z]+)-(?P<part>\d{3})\b", re.IGNORECASE)
SUPERVISED_KINDS = {"takeout", "takeout_zip"}


def _now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def list_marks() -> SourceUnitMarkList:
    return SourceUnitMarkList(
        items=[SourceUnitMark.model_validate(item) for item in read_all()],
        storage=storage_description(),
    )


def record_mark(request: SourceUnitMarkRequest, marked_by: str) -> SourceUnitMark:
    """Record one hand-marked unit. A Takeout needs `confirm`."""
    if request.unit_type in SUPERVISED_KINDS and not request.confirm:
        raise UnitMarkStoreError(
            "A Takeout unit is a proposal: review its parts and confirm before it is recorded", 409
        )
    root = request.unit_root.rstrip("/")
    mark = SourceUnitMark(
        unit_root=root,
        unit_type=request.unit_type,
        label=request.label.strip(),
        marked_at=_now(),
        marked_by=marked_by,
    )
    items = [item for item in read_all() if str(item.get("unit_root", "")).rstrip("/") != root]
    items.append(mark.model_dump(mode="json"))
    write_all(items)
    return mark


def propose_unit(request: SourceUnitProposalRequest) -> SourceUnitProposal:
    """What this folder looks like, from the names the browser just listed.

    The basis is the observed listing, not the catalog: the catalog's own
    detections arrive separately through the discovery unit routes, and the two
    are never blended into one unlabelled answer.
    """
    sets: dict[tuple[str, str], set[int]] = {}
    for name in request.names:
        match = TAKEOUT_PART.match(name.rsplit("/", 1)[-1])
        if match:
            sets.setdefault((match["stamp"], match["job"]), set()).add(int(match["part"]))
    proposals = [
        TakeoutPartProposal(
            stamp=stamp,
            job=job,
            parts_present=sorted(parts),
            parts_missing=sorted(set(range(1, max(parts) + 1)) - parts),
            highest_part=max(parts),
        )
        for (stamp, job), parts in sorted(sets.items())
    ]
    looks_like = "takeout" if proposals else _kind_from_names(request.names)
    return SourceUnitProposal(
        unit_root=request.unit_root.rstrip("/"),
        looks_like=looks_like,
        basis="observed_listing",
        observed_count=len(request.names),
        takeout_sets=proposals,
        requires_confirmation=looks_like in SUPERVISED_KINDS,
    )


def _kind_from_names(names: list[str]) -> str | None:
    """A conservative guess for the marking dialog's default; never a decision."""
    lowered = {name.rsplit("/", 1)[-1].lower() for name in names}
    if ".obsidian" in lowered:
        return "obsidian_vault"
    if ".git" in lowered:
        return "git_repo"
    if any(name.startswith("cubeacr") or name.endswith(".cubeacr") for name in lowered):
        return "cube_acr"
    return None
