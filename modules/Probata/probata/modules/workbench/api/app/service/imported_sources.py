"""Project version outcomes into source files, export pages and summary totals.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

The facade owns shared cache, configuration and upstream injection seams.
Inputs/outputs remain the existing Imported read contract; no writes occur here.
"""

from __future__ import annotations

import asyncio
from datetime import UTC, datetime
from typing import Any

from app.service import imported


def _version_status(row: dict[str, Any], lifecycles: dict[str, str] | None) -> str:
    """One source version's own outcome: the publish receipt in PostgreSQL first, the run list second."""
    if row["published"] or row["approved"]:
        return "committed"
    if row["rejected"]:
        return "failed"
    if lifecycles is None:
        return "not_finished"
    lifecycle = lifecycles.get(row["id"])
    return imported._LIFECYCLE_STATUS.get(lifecycle or "", "not_finished")


def _is_media(key: str) -> bool:
    """Project version outcomes into source files, export pages and summary totals.
    """
    return ".derived/media/" in key


def _file_status(versions: list[dict[str, Any]], lifecycles: dict[str, str] | None) -> tuple[str, dict[str, Any], int]:
    """ONE status per FILE across all its source versions and attempts.

    A file that published on any attempt is Done (committed); earlier failed attempts are counted, not shown
    as the file's status. Otherwise the latest attempt's outcome stands. Derived media that never imported
    is skipped. Returns (status, the version that stands for the file's counts, failed attempts).
    """
    ordered = sorted(versions, key=lambda v: (v["version_ordinal"], v["acquired_at"]), reverse=True)
    statuses = [imported._version_status(v, lifecycles) for v in ordered]
    failed_attempts = sum(1 for status in statuses if status == "failed")
    for version, status in zip(ordered, statuses):
        if status == "committed":
            return "committed", version, failed_attempts
    key = ordered[0]["source_key"]
    if imported._is_media(key):
        return "skipped", ordered[0], failed_attempts
    return statuses[0], ordered[0], failed_attempts


def file_rows(rows: list[dict[str, Any]], lifecycles: dict[str, str] | None) -> list[dict[str, Any]]:
    """Every source FILE (one per source key) with its single status, its counts and its export."""
    by_key: dict[str, list[dict[str, Any]]] = {}
    for row in rows:
        by_key.setdefault(row["source_key"], []).append(row)
    out = []
    for key, versions in by_key.items():
        status, stand, failed_attempts = imported._file_status(versions, lifecycles)
        out.append({
            "key": key, "export_key": stand["export_key"], "status": status, "failed_attempts": failed_attempts,
            "attempts": len(versions), "raw": stand["raw_n"], "normalized": stand["norm_n"], "messages": stand["msgs"], "calls": stand["calls"],
            "first_at": stand["first_at"], "last_at": stand["last_at"], "acquired_at": max(v["acquired_at"] for v in versions),
            "is_parent": key == stand["export_key"], "is_media": imported._is_media(key), "version_id": stand["id"],
        })
    return out


async def _exports() -> tuple[list[dict[str, Any]], bool]:
    """Project version outcomes into source files, export pages and summary totals.
    """
    matter = imported.live_matter()
    rows = await asyncio.to_thread(lambda: imported._cached(f"sv:{matter}", 15, lambda: imported.pg.source_versions(matter)))
    lifecycles = await imported._lifecycles()
    people = await asyncio.to_thread(imported._people)
    grouped: dict[str, dict[str, Any]] = {}
    for file in imported.file_rows(rows, lifecycles):
        group = grouped.setdefault(
            file["export_key"],
            {
                "export_key": file["export_key"], "files": 0, "raw": 0, "normalized": 0, "committed": 0,
                "messages": 0, "calls": 0, "first_at": None, "last_at": None, "imported_at": None,
                "status_counts": {}, "split": None, "_parent": None, "_children": [],
            },
        )
        if file["is_parent"]:
            group["_parent"] = file
        elif not file["is_media"]:
            group["_children"].append(file)
        if file["is_media"] and file["status"] == "skipped":
            group["status_counts"]["skipped"] = group["status_counts"].get("skipped", 0) + 1
            continue
        group["raw"] += file["raw"]
        group["normalized"] += file["normalized"]
        group["messages"] += file["messages"]
        group["calls"] += file["calls"]
        if file["status"] == "committed":
            group["committed"] += file["normalized"]
        if file["first_at"] and (group["first_at"] is None or file["first_at"] < group["first_at"]):
            group["first_at"] = file["first_at"]
        if file["last_at"] and (group["last_at"] is None or file["last_at"] > group["last_at"]):
            group["last_at"] = file["last_at"]
        if group["imported_at"] is None or file["acquired_at"] > group["imported_at"]:
            group["imported_at"] = file["acquired_at"]
    exports = []
    for group in grouped.values():
        children, parent = group.pop("_children"), group.pop("_parent")
        child_counts: dict[str, int] = {}
        for child in children:
            child_counts[child["status"]] = child_counts.get(child["status"], 0) + 1
        is_split = parent is not None and bool(children)
        if is_split:
            # The parent backup was split into one file per conversation: its own status is the children's.
            done, failed = child_counts.get("committed", 0), child_counts.get("failed", 0)
            group["split"] = {"total": len(children), "done": done, "failed": failed,
                              "in_progress": len(children) - done - failed}
            counted = children
        else:
            counted = ([parent] if parent else []) + children
        for file in counted:
            group["status_counts"][file["status"]] = group["status_counts"].get(file["status"], 0) + 1
        group["files"] = len(counted) + (1 if is_split else 0)  # the parent counts as a file
        counts = group["status_counts"]
        group["status"] = next((name for name in imported._STATUS_ORDER if counts.get(name)), "committed" if counts.get("skipped") else "not_finished")
        group["failed_attempts"] = sum(f["failed_attempts"] for f in counted)
        group["id"] = imported.encode_id(group["export_key"])
        group.update(imported.describe_export(group["export_key"], people))
        for key in ("first_at", "last_at", "imported_at"):
            group[key] = imported._iso(group[key])
        exports.append(group)
    exports.sort(key=lambda item: item["imported_at"] or "", reverse=True)
    return exports, lifecycles is not None


async def sources(*, limit: int, offset: int, fmt: str | None) -> dict[str, Any]:
    """Project version outcomes into source files, export pages and summary totals.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    exports, lifecycle_ok = await imported._exports()
    if fmt:
        exports = [item for item in exports if item["format"].lower() == fmt.lower()]
    page = exports[offset: offset + limit]
    totals = {"files": sum(i["files"] for i in exports), "raw": sum(i["raw"] for i in exports),
              "normalized": sum(i["normalized"] for i in exports), "committed": sum(i["committed"] for i in exports)}
    return {
        "items": page, "total": len(exports), "next_offset": offset + limit if offset + limit < len(exports) else None,
        "totals": totals, "run_state_available": lifecycle_ok,
    }


async def _export_for(source_id: str) -> dict[str, Any]:
    """Project version outcomes into source files, export pages and summary totals.
    """
    (export_key,) = imported.decode_id(source_id, 1)
    exports, _ = await imported._exports()
    for item in exports:
        if item["export_key"] == export_key:
            return item
    raise imported.ImportedError("Unknown source", 404)


async def summary() -> dict[str, Any]:
    """Project version outcomes into source files, export pages and summary totals.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    exports, lifecycle_ok = await imported._exports()
    totals = {"sources": len(exports), "files": sum(i["files"] for i in exports),
              "raw": sum(i["raw"] for i in exports), "normalized": sum(i["normalized"] for i in exports),
              "committed": sum(i["committed"] for i in exports),
              "messages": sum(i["messages"] for i in exports), "calls": sum(i["calls"] for i in exports)}
    status: dict[str, int] = {}
    for item in exports:
        for name, count in item["status_counts"].items():
            status[name] = status.get(name, 0) + count
    return {"matter_id": imported.live_matter(), "totals": totals, "files_by_status": status, "run_state_available": lifecycle_ok,
            "generated_at": datetime.now(UTC).isoformat()}
