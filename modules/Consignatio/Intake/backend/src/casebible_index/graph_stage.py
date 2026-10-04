"""The graph stage: project the active snapshot into the Intake file graph (surreal-intake). One
unit, one job.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The projection contract (projections/index_run.py) is snapshot-scoped: every projection writes one
occurrence node per
document under a new snapshot key. Run after every extract slice of a 570,000-object catalog it
would write the whole
corpus again each time. So it is its own stage with its own cadence: it runs when the active
snapshot has changed since
the last projection AND at least ``INTAKE_GRAPH_MIN_INTERVAL_HOURS`` (default 24) have passed, or
when forced. The
Surreal instance is ``surreal-intake``; this unit never touches the Docstore.
"""

from __future__ import annotations

import json
import os
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .config import Settings
from .snapshots import newest_snapshot

DEFAULT_MIN_INTERVAL_HOURS = 24.0


def last_projection(output_dir: Path) -> dict[str, Any] | None:
    """The newest cycle receipt of a COMPLETED projection, from the cycle trail."""
    root = output_dir / "cycles"
    if not root.is_dir():
        return None
    for folder in sorted(root.glob("*"), reverse=True):
        for receipt in sorted(folder.glob("*-graph.json"), reverse=True):
            data = json.loads(receipt.read_text(encoding="utf-8"))
            if data.get("projected"):
                return data
    return None


def graph_due(
    output_dir: Path,
    snapshot: Path | None,
    *,
    now: datetime | None = None,
    force: bool = False,
    min_hours: float | None = None,
) -> tuple[bool, str]:
    if snapshot is None:
        return False, "no active snapshot yet"
    if force:
        return True, "forced"
    hours = (
        float(os.getenv("INTAKE_GRAPH_MIN_INTERVAL_HOURS", str(DEFAULT_MIN_INTERVAL_HOURS)))
        if min_hours is None
        else min_hours
    )
    previous = last_projection(output_dir)
    if previous is None:
        return True, "never projected"
    if previous.get("snapshot") == str(snapshot):
        return False, "snapshot unchanged since the last projection"
    age = (
        (now or datetime.now(UTC)) - datetime.fromisoformat(previous["recorded_at"])
    ).total_seconds() / 3600
    if age < hours:
        return False, f"last projection {age:.1f} h ago, minimum interval {hours:g} h"
    return True, "snapshot changed and the interval has passed"


async def project_snapshot(settings: Settings, snapshot: Path) -> dict[str, Any]:
    from .projections.index_run import project_index_run
    from .projections.runtime import connect_graph

    catalog_mode = settings.source_mode == "catalog"
    locator = (
        f"{settings.object_store_scheme}://{settings.vault_bucket}/"
        if catalog_mode
        else Path(settings.source_dir).as_uri()
    )
    async with await connect_graph() as graph:
        return await project_index_run(
            graph,
            output_dir=settings.output_dir,
            snapshot=snapshot,
            source_id=settings.source_id,
            root_locator=locator,
            store_kind="object_store" if catalog_mode else "filesystem",
        )


async def run_graph_stage(settings: Settings, *, force: bool = False) -> dict[str, Any]:
    snapshot = newest_snapshot(settings.output_dir)
    due, reason = graph_due(settings.output_dir, snapshot, force=force)
    if not due:
        return {"projected": False, "reason": reason, "snapshot": str(snapshot) if snapshot else ""}
    counts = await project_snapshot(settings, snapshot)  # type: ignore[arg-type]
    return {"projected": True, "reason": reason, "snapshot": str(snapshot), **counts}
