"""Cycle receipts and the catalog watermark. One unit, one job: remember what a cycle did.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Every Super Index cycle (discover, extract+chunk, summary, embed, publish, commit) leaves one \
small immutable JSON
per stage under ``cycles/<cycle_id>/``. The watermark (the catalog listings the index has fully \
caught up to) advances
ONLY when a cycle commits, so a cycle that fails midway is simply run again from the same \
watermark: every stage is
idempotent (memoized extraction, content-addressed vectors, upserted objects). These files are \
the receipt trail the
Temporal history points at; the history carries counts, the files carry the detail.
"""

from __future__ import annotations

import json
import re
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .parquet_store import write_json_immutable

CYCLE_ID_FORMAT = "%Y%m%dT%H%M%SZ"
_PLAIN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")


def new_cycle_id(now: datetime | None = None) -> str:
    return (now or datetime.now(UTC)).strftime(CYCLE_ID_FORMAT)


def cycle_dir(output_dir: Path, cycle_id: str) -> Path:
    if not _PLAIN.match(cycle_id) or ".." in cycle_id:
        raise ValueError("cycle id must be a plain name")
    return output_dir / "cycles" / cycle_id


def write_stage_receipt(
    output_dir: Path, cycle_id: str, stage: str, payload: dict[str, Any]
) -> Path:
    now = datetime.now(UTC)
    path = cycle_dir(output_dir, cycle_id) / f"{now.strftime('%H%M%S.%fZ')}-{stage}.json"
    write_json_immutable(
        path, {"cycle_id": cycle_id, "stage": stage, "recorded_at": now.isoformat(), **payload}
    )
    return path


def read_stage_receipts(output_dir: Path, cycle_id: str) -> list[dict[str, Any]]:
    folder = cycle_dir(output_dir, cycle_id)
    if not folder.is_dir():
        return []
    return [json.loads(p.read_text(encoding="utf-8")) for p in sorted(folder.glob("*.json"))]


def last_commit(output_dir: Path) -> dict[str, Any] | None:
    """The newest committed cycle's receipt (its ``watermark`` is what the index has caught up \
to)."""
    root = output_dir / "cycles"
    if not root.is_dir():
        return None
    for folder in sorted(root.glob("*"), reverse=True):
        for receipt in sorted(folder.glob("*-commit.json"), reverse=True):
            return json.loads(receipt.read_text(encoding="utf-8"))
    return None


def commit_cycle(
    output_dir: Path, cycle_id: str, watermark: list[dict[str, Any]], summary: dict[str, Any]
) -> Path:
    """Advance the watermark. Called once, last, by the workflow, after every stage of the \
cycle succeeded."""
    return write_stage_receipt(output_dir, cycle_id, "commit", {"watermark": watermark, \
"summary": summary})
