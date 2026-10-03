"""server/temporal/entity_event_activities.py — one Temporal Activity per selectable entity and event extractor.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The Go ``extraction_request_workflow`` (modules/engine/extraction/flow/conversation_workflows.go) schedules these on
this worker's task queue (``evidence-pipeline``), one call per window of up to 200 messages of one normalized
generation, then hands the reply to a Go Activity that validates it against the default extractor's schema, grounds it
in the same messages and stages it under a compare-only ``working.extraction_run`` tagged with the extractor.

    extract_entities_events_semantica_activity     semantica    the vendored pattern extractors, no model call
    extract_entities_events_langextract_activity   langextract  Google LangExtract over kimi-k3 on NVIDIA NIM

Same input, same output shape for every extractor (``server.tools.extractors.entity_events.pages``). The Activities
read PostgreSQL read-only and write nothing: staging is the Go side's, so there is one writer.

Import rule (as in timeline_activities.py): temporalio + stdlib at module level; the extractors, and through them the
heavy libraries, are imported inside the body.
"""

from __future__ import annotations

import threading
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass
from typing import Any

from temporalio import activity
from temporalio.exceptions import ApplicationError

SEMANTICA_ACTIVITY = "extract_entities_events_semantica_activity"
LANGEXTRACT_ACTIVITY = "extract_entities_events_langextract_activity"


@dataclass
class ExternalPageParams:
    """The Go ``flow.ExternalPageRequest`` (JSON field names match)."""

    generation_id: str = ""
    source_version_id: str = ""
    preview_handle: str = ""
    extractor: str = ""
    after_ordinal: int = -1
    limit: int = 200


@contextmanager
def _heartbeating(detail: str, every_seconds: float = 30.0) -> Iterator[None]:
    """Heartbeat from a side thread while a long model call runs, so the Go side's heartbeat timeout does not fire."""
    stop = threading.Event()

    def beat() -> None:
        while not stop.wait(every_seconds):
            if activity.in_activity():
                activity.heartbeat(detail)

    thread = threading.Thread(target=beat, daemon=True)
    thread.start()
    try:
        yield
    finally:
        stop.set()
        thread.join(timeout=2)


def _window(params: ExternalPageParams) -> list[Any]:
    from server.context_chunks.db import read_only_connection
    from server.tools.extractors.entity_events.pages import PAGE_MESSAGES, read_window

    if not params.generation_id.strip():
        raise ApplicationError("an extraction window needs a generation id", non_retryable=True)
    limit = params.limit if 0 < params.limit <= PAGE_MESSAGES else PAGE_MESSAGES
    with read_only_connection() as conn:
        return read_window(conn, params.generation_id.strip(), params.after_ordinal, limit)


@activity.defn(name=SEMANTICA_ACTIVITY)
def extract_entities_events_semantica_activity(params: ExternalPageParams) -> dict[str, Any]:
    """Run the vendored Semantica pattern extractors over one window of a generation's messages and return the shared page.

    No model call and no write; the reply is the people, places, organizations and dated events the patterns found,
    each tied to the message it came from. Literal and noisy by nature, so its output is compare-only.
    """
    from server.tools.extractors.entity_events import semantica_extractor

    window = _window(params)
    page = semantica_extractor.extract_page(window, params.after_ordinal)
    activity.logger.info(
        "semantica read %d messages after %d: %d people, %d events",
        len(window), params.after_ordinal, len(page["people"]), len(page["events"]),
    )
    return page


@activity.defn(name=LANGEXTRACT_ACTIVITY)
def extract_entities_events_langextract_activity(params: ExternalPageParams) -> dict[str, Any]:
    """Run LangExtract over kimi-k3 on NVIDIA NIM on one window of a generation's messages and return the shared page.

    Needs the default extractor's model environment (``ENTITY_MODEL_API_KEY_FILE``); without a mounted key the page
    comes back skipped with the reason instead of failing. A model reply that is empty, junk, cut off or not JSON is
    retried once with thinking on and then fails the Activity with the reason.
    """
    from server.tools.extractors.entity_events import langextract_extractor as lx
    from server.tools.extractors.entity_events.pages import skipped_page

    try:
        config = lx.config_from_env()
    except lx.ExtractorUnavailable as unavailable:
        return skipped_page(lx.EXTRACTOR, str(unavailable), params.after_ordinal)
    except ValueError as invalid:
        raise ApplicationError(str(invalid), non_retryable=True) from invalid
    window = _window(params)
    try:
        with _heartbeating(f"langextract {len(window)} messages after {params.after_ordinal}"):
            page = lx.extract_page(window, params.after_ordinal, config)
    except lx.ExtractorUnavailable as unavailable:
        return skipped_page(lx.EXTRACTOR, str(unavailable), params.after_ordinal)
    activity.logger.info(
        "langextract read %d messages after %d: %d people, %d events",
        len(window), params.after_ordinal, len(page["people"]), len(page["events"]),
    )
    return page
