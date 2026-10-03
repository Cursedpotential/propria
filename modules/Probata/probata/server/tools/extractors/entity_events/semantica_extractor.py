"""Semantica as an entity and event extractor: the governed pattern worker over one window of messages.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Reuses ``server.analysis.semantica_worker.SemanticaPatternWorker`` (ADR-0043: candidates only, no store, no model) and
changes nothing in it. This module only converts a window of messages into its input and its candidates into the
shared page. The pattern methods are literal: they find capitalised phrases and dated sentences, so expect noise;
that is why the output is compare-only until the owner picks it.
"""

from __future__ import annotations

from datetime import datetime
from hashlib import sha256
from typing import Any

from server.tools.extractors.entity_events.pages import Message, PageBuilder

EXTRACTOR = "semantica"

# Semantica's event labels onto the shared event types; anything else is "other".
_EVENT_TYPE_WORDS = {
    "court": ("court", "hearing", "trial", "judge", "filing"),
    "medical": ("doctor", "medical", "hospital", "appointment_medical", "clinic"),
    "school": ("school", "teacher", "class"),
    "travel": ("travel", "trip", "flight", "move", "moved"),
    "financial": ("payment", "paid", "financial", "support"),
    "communication": ("call", "meeting", "communication", "message"),
    "appointment": ("appointment", "meeting_scheduled"),
    "incident": ("incident", "accident", "arrest"),
}

_KIND_OF = {"person": "people", "organization": "organizations", "location": "places"}


def _event_type(label: str) -> str:
    lowered = label.casefold()
    for shared, words in _EVENT_TYPE_WORDS.items():
        if any(word in lowered for word in words):
            return shared
    return "other"


def _date_of(candidate: Any) -> tuple[str | None, str | None]:
    """A full calendar date only when Semantica found one; a year or a vague window is kept as the stated phrase."""
    if candidate.occurred_at is not None:
        return candidate.occurred_at.date().isoformat(), None
    start, end = candidate.valid_from, candidate.valid_to
    if isinstance(start, datetime) and isinstance(end, datetime) and (end - start).days == 1:
        return start.date().isoformat(), None
    if isinstance(start, datetime):
        return None, str(start.year)
    return None, None


def extract_page(window: list[Message], after_ordinal: int, *, version: str | None = None) -> dict[str, Any]:
    """Run Semantica's pattern extractors over a window of messages and return the shared page.

    Messages with no text are skipped. Entities Semantica labels as a device or an account are dropped, as the
    governed worker already does for non-identity labels. Events keep only a full stated date.
    """
    from server.analysis.semantica_contracts import ExtractionBatch, ExtractionRecord
    from server.analysis.semantica_worker import SemanticaPatternWorker

    worker = SemanticaPatternWorker()
    builder = PageBuilder(EXTRACTOR, version or worker.extractor_version)
    records = []
    for message in window:
        if not message.body.strip():
            continue
        records.append(
            ExtractionRecord(
                source_raw_id=message.record_id,
                content=message.body,
                content_sha256=sha256(message.body.encode("utf-8")).hexdigest(),
                provenance_ref=f"context.normalized_record_identity:{message.record_id}",
                occurred_at=message.occurred_at,
            )
        )
    if records:
        batch = worker.extract(ExtractionBatch(batch_id=f"window-{after_ordinal}", matter_id="window", records=tuple(records)))
        if batch.failures:
            first = batch.failures[0]
            raise RuntimeError(f"semantica failed on {first.source_raw_id}: {first.error}")
        for entity in batch.entities:
            kind = _KIND_OF.get(entity.entity_type)
            if kind is None:
                continue
            surface = entity.provenance.evidence_quote or entity.name
            builder.add_entity(kind, entity.name, entity.provenance.source_raw_id, surface)
        for event in batch.events:
            date, stated = _date_of(event)
            builder.add_event(
                f"{event.event_type}: {event.summary}"[:200],
                event.provenance.source_raw_id,
                description=event.summary,
                date=date,
                when=stated,
                event_type=_event_type(event.event_type),
            )
    return builder.page(window, after_ordinal)
