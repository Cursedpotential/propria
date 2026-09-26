"""server/temporal/timeline_activities.py — the Timesketch projection step.

Byline: Claude Code · Opus 5.5 · 2026-09-25

``build_timeline_generation_activity`` is the last step of the Go
``extraction_commit_workflow`` (modules/engine/extraction/flow/workflows.go),
scheduled on this worker's task queue (``evidence-pipeline``) after committed
events became ``timeline.timeline_member`` rows. It REUSES
``server.timeline.generation.build_generation`` — the D02 builder — and adds
nothing of its own: no second timeline store, no second projection.

One unit, one job: it builds (or idempotently returns) one sealed projection
generation for a collection. The Timesketch-fork importer consumes it through
``PostgresTimelineProjectionSource``; this Activity never writes OpenSearch.

Import rule (as in n8n_activities.py): temporalio + stdlib at module level;
``server.timeline`` (whose db module reads the database URL at import) is
imported inside the body so importing this module stays side-effect free.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from temporalio import activity

BUILD_TIMELINE_GENERATION_ACTIVITY = "build_timeline_generation_activity"


@dataclass
class BuildGenerationParams:
    """The Go ``flow.ProjectionRequest`` (JSON field names match)."""

    collection_slug: str = "primary"
    created_by: str = "timeline_projector"
    commit_id: str = ""


@activity.defn(name=BUILD_TIMELINE_GENERATION_ACTIVITY)
def build_timeline_generation_activity(params: BuildGenerationParams) -> dict[str, Any]:
    """Build one sealed generation for ``params.collection_slug``; idempotent.

    Returns the Go ``flow.ProjectionResult`` shape. Runs in the worker's
    thread pool (sync SQLAlchemy); the transaction is the builder's contract.
    """
    from server.timeline.db import get_engine
    from server.timeline.generation import build_generation

    slug = (params.collection_slug or "primary").strip() or "primary"
    created_by = (params.created_by or "timeline_projector").strip()[:200] or "timeline_projector"
    with get_engine().begin() as conn:
        result = build_generation(conn, collection_slug=slug, created_by=created_by)
    activity.logger.info(
        "timeline generation %s (sequence %s, created=%s, members=%s) for commit %s",
        result.generation_id,
        result.sequence,
        result.created,
        result.member_count,
        params.commit_id,
    )
    return {
        "generation_id": result.generation_id,
        "sequence": result.sequence,
        "created": result.created,
        "member_count": result.member_count,
        "skipped_unresolved_governed_members": list(result.skipped_unresolved_governed_members),
    }
