"""entity_events — selectable entity and event extractors over a window of conversation messages.

One extractor = one module = one Temporal Activity (``server/temporal/entity_event_activities.py``). Every extractor
takes the same input (a generation id and a position in its messages) and returns the same page shape
(:mod:`server.tools.extractors.entity_events.pages`), which the Go engine validates against the default extractor's
schema, grounds in the same messages and stages under a compare-only ``working.extraction_run`` tagged with the
extractor. The default extractor (Go + kimi-k3) lives in ``modules/engine/extraction``; these are the alternatives,
so outputs can be compared side by side.

    semantica    the vendored Semantica pattern extractors (no model call)
    langextract  Google LangExtract over kimi-k3 on NVIDIA NIM

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""
