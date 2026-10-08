"""Expose six independent, bounded AI content stages to the evidence worker.

Inputs: exact retained-source/case pins, optional legacy generation pins and file refs.
Outputs: bounded stage receipts. Effects: delegated per Activity, never source
payloads in Temporal history. Pick for verified standard AI exports rather than
human chunks or the legacy chat-file/pending projector.
Byline: Codex / GPT-6.1-Sol / 2026-10-06.
"""
from __future__ import annotations

from contextlib import contextmanager
from contextvars import copy_context
from dataclasses import asdict, dataclass
from datetime import timedelta
from threading import Event, Thread
from typing import Any, Iterator

from temporalio import activity
from temporalio.exceptions import ApplicationError, CancelledError


@dataclass
class AIContentParams:
    """Decode the shared Go AIContentRequest for native and legacy content runs.

    Inputs: exact source/case and native original fields, optional legacy
    generation pins, stage refs and approved bounds. Outputs: dataclass request.
    Effects: none; choose for all six Activities while bodies stay outside history.
    """
    request_id: str = ""
    source_version_id: str = ""
    normalized_generation_id: str = ""
    verification_id: str = ""
    original_ref: str = ""
    native_source_only: bool = False
    source_object_id: str = ""
    source_sha256: str = ""
    version_id: str | None = None
    operating_mode: str = ""
    matter_id: str = ""
    court_case_id: str = ""
    prepared_ref: str = ""
    work_products_ref: str = ""
    candidates_ref: str = ""
    embeddings_ref: str = ""
    publication_ref: str = ""
    max_records: int = 1024
    max_text_bytes: int = 2097152
    max_chunks: int = 256
    max_model_calls: int = 512


@contextmanager
def _heartbeats() -> Iterator[Any]:
    """Keep a bounded sync provider operation heartbeating in its Activity context.

    Inputs: active Temporal context. Outputs: safe heartbeat callback. Effects:
    heartbeat thread stopped on exit; no source/model text in heartbeat details.
    Choose for provider stages that can block longer than a heartbeat interval.
    """
    active = activity.in_activity()
    stop = Event()

    def beat(detail: str) -> None:
        """Report a stage label without source payloads.

        Inputs: safe label. Outputs: none. Effects: Temporal heartbeat when active;
        choose instead of putting source text in progress events.
        """
        if active:
            if activity.is_cancelled():
                raise CancelledError("AI content Activity cancellation was requested")
            activity.heartbeat(detail)

    def pulse() -> None:
        """Pulse while a synchronous provider call blocks.

        Inputs: captured Activity context and stop event. Outputs: none. Effects:
        periodic heartbeat; choose over blocking provider-specific progress hooks.
        """
        while not stop.wait(20):
            beat("AI content stage running")

    context = copy_context()
    thread = Thread(target=lambda: context.run(pulse), daemon=True, name="ai-content-heartbeat")
    if active:
        beat("AI content stage starting")
        thread.start()
    try:
        yield beat
    finally:
        stop.set()
        if active:
            thread.join(timeout=2)


def _run(name: str, params: AIContentParams) -> dict[str, Any]:
    """Delegate one lazy stage and classify permanent invalid inputs for Temporal.

    Inputs: owned stage function and common request. Outputs: small receipt.
    Effects: that function's independent operation; invalid source/ref/grounding
    fails nonretryably, transport failures remain retryable with no skip/fallback.
    Choose instead of one monolithic ingestion Activity.
    """
    from server.analysis import ai_content
    from server.analysis.ai_content_provider import ProviderDeferred, ProviderUnavailable
    try:
        with _heartbeats() as beat:
            function = getattr(ai_content, name)
            if name in {"extract_candidates", "publish_content", "verify_publication"}:
                return function(asdict(params), beat=beat)
            return function(asdict(params))
    except ai_content.ContentInvalid as error:
        raise ApplicationError(str(error), type="AIContentInvalid", non_retryable=True) from None
    except ProviderDeferred as error:
        raise ApplicationError(str(error), error.metadata, type="AIContentProviderDeferred",
                               next_retry_delay=timedelta(seconds=error.delay)) from None
    except ProviderUnavailable as error:
        raise ApplicationError(str(error), error.metadata, type="AIContentProviderUnavailable", non_retryable=True) from None


@activity.defn(name="ai_prepare_content_activity")
def ai_prepare_content_activity(params: AIContentParams) -> dict[str, Any]:
    """Prepare Neural topic chunks from one exact retained native AI original.

    Inputs: source/case pins, original_ref, native_source_only and bounds. Outputs:
    prepared bundle_ref, source hash and counts. Effects: retained-original read,
    Neural inference and derived file; choose before extraction and embedding.
    """
    return _run("prepare_content", params)


@activity.defn(name="ai_extract_work_products_activity")
def ai_extract_work_products_activity(params: AIContentParams) -> dict[str, Any]:
    """Retain complete fenced artifacts and explicitly marked draft occurrences.

    Inputs: exact pins and prepared_ref. Outputs: work_products bundle_ref and
    count. Effects: retained derived files; choose independently before model
    candidate extraction so complete useful content survives short quotations.
    """
    return _run("extract_work_products", params)


@activity.defn(name="ai_extract_candidates_activity")
def ai_extract_candidates_activity(params: AIContentParams) -> dict[str, Any]:
    """Extract useful AI content candidates and ground their exact source quotes.

    Inputs: pins, prepared_ref and model bound. Outputs: candidates bundle_ref.
    Effects: approved remote AI provider calls and retained unreviewed candidates;
    choose independently from preparation, embedding and canonical fact decisions.
    """
    return _run("extract_candidates", params)


@activity.defn(name="ai_embed_content_activity")
def ai_embed_content_activity(params: AIContentParams) -> dict[str, Any]:
    """Embed prepared conversation windows using the configured remote provider.

    Inputs: pins and prepared_ref. Outputs: embedded bundle_ref and counts.
    Effects: remote NIM calls and retained vectors; choose separately from
    publication so retries reuse a pinned model/content embedding bundle.
    """
    return _run("embed_content", params)


@activity.defn(name="ai_publish_content_activity")
def ai_publish_content_activity(params: AIContentParams) -> dict[str, Any]:
    """Add coherent AI conversation chunks to the existing AI search collection.

    Inputs: pins and four prerequisite refs. Outputs: published bundle_ref and
    complete object count. Effects: fresh LIVE authority and additive search
    inserts; choose after providers, never publish one object per message.
    """
    return _run("publish_content", params)


@activity.defn(name="ai_verify_content_publication_activity")
def ai_verify_content_publication_activity(params: AIContentParams) -> dict[str, Any]:
    """Independently verify published AI properties, citations, models and vectors.

    Inputs: pins, prerequisites and publication_ref. Outputs: verified bundle_ref
    and exact readback count. Effects: read-only search/DB and retained proof;
    choose after publication, never equate successful writes with verification.
    """
    return _run("verify_publication", params)


AI_CONTENT_ACTIVITIES = (
    ai_prepare_content_activity,
    ai_extract_work_products_activity,
    ai_extract_candidates_activity,
    ai_embed_content_activity,
    ai_publish_content_activity,
    ai_verify_content_publication_activity,
)
