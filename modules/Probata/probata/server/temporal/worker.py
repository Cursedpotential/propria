"""server/temporal/worker.py — the Temporal worker entrypoint (P1 task 1).

Byline: Claude Code · Fable 5 · 2026-08-24
Byline: Codex · GPT-6 · 2026-10-07 (register real graph preparation Activity).

Runs as its OWN Coolify app (docker/temporal-worker/Dockerfile, CMD
``python -m server.temporal.worker``). Joins task queue ``evidence-pipeline``
and registers:

  workflows:  ChatTranscriptIngest (P1), P0DurabilityProbe (the P0 exit test)
  activities: custody_activity, parse_activity, store_activity (sync — run in
              a thread pool), knowledge_activity (async), plus one
              extract_html_<tool>_activity per HTML library (html_tool_activities.py), and
              extract_entities_events_<extractor>_activity per selectable entity/event extractor
              (entity_event_activities.py: semantica, langextract)

Env:
  TEMPORAL_ADDRESS    frontend address (default temporal-server:7233 on the
                      shared `agno` network; the deployed app overrides with
                      the tailnet address 100.91.190.107:7233)
  TEMPORAL_NAMESPACE  default: "default"
  TEMPORAL_TASK_QUEUE default: "evidence-pipeline"
  TEMPORAL_ACTIVITY_THREADS  sync-activity pool size, default 8
"""

from __future__ import annotations

import asyncio
import logging
import os
from concurrent.futures import ThreadPoolExecutor

from temporalio.client import Client
from temporalio.worker import Worker

from server.temporal.activities import (
    custody_activity,
    knowledge_activity,
    parse_activity,
    store_activity,
)
from server.temporal.ai_content_activities import AI_CONTENT_ACTIVITIES, AI_CONTEXT_ACTIVITIES
from server.temporal.ai_context_graph_activities import AI_CONTEXT_GRAPH_ACTIVITIES
from server.temporal.chunk_activities import (
    chunk_context_threads_activity,
    publish_call_log_files_activity,
    publish_context_chunks_activity,
)
from server.temporal.chunk_backfill_activities import (
    estimate_context_chunks_activity,
    list_context_threads_activity,
    remove_per_message_objects_activity,
    verify_chunk_coverage_activity,
)
from server.temporal.classification_workflow import ClassificationBatchPipeline
from server.temporal.entity_event_activities import (
    extract_entities_events_langextract_activity,
    extract_entities_events_semantica_activity,
)
from server.temporal.html_tool_activities import HTML_TOOL_ACTIVITIES
from server.temporal.n8n_activities import n8n_webhook_activity
from server.temporal.timeline_activities import build_timeline_generation_activity
from server.temporal.workflows import ChatTranscriptIngest, P0DurabilityProbe

log = logging.getLogger("temporal.worker")

TASK_QUEUE_DEFAULT = "evidence-pipeline"


async def main() -> None:
    """Run the Python Temporal worker with context and historical Activities.

    Inputs: Temporal connection and task-queue environment. Outputs: none.
    Effects: registers Activities and polls Temporal. Choose for the managed
    Python worker so ai-context-v1 runs without replacing replayed workflows.
    """
    logging.basicConfig(
        level=os.environ.get("LOG_LEVEL", "INFO"),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    address = os.environ.get("TEMPORAL_ADDRESS", "temporal-server:7233")
    namespace = os.environ.get("TEMPORAL_NAMESPACE", "default")
    task_queue = os.environ.get("TEMPORAL_TASK_QUEUE", TASK_QUEUE_DEFAULT)
    threads = int(os.environ.get("TEMPORAL_ACTIVITY_THREADS", "8"))

    log.info("connecting to Temporal at %s (namespace=%s)", address, namespace)
    client = await Client.connect(address, namespace=namespace)
    log.info("connected; joining task queue %r (%d activity threads)", task_queue, threads)

    with ThreadPoolExecutor(max_workers=threads) as executor:
        worker = Worker(
            client,
            task_queue=task_queue,
            workflows=[ChatTranscriptIngest, P0DurabilityProbe, ClassificationBatchPipeline],
            # build_timeline_generation_activity: projection step of the Go
            # extraction_commit_workflow (Claude Code · Opus 5.5 · 2026-09-25).
            # chunk_context_threads_activity / publish_context_chunks_activity /
            # publish_call_log_files_activity: the conversation-chunk units the Go ProfferWorkflow calls once
            # the first-party context is proposed and BEFORE the preview (Claude Code · Sonnet 5.5 · 2026-10-02).
            # list/estimate/verify/remove: the re-chunk and the per-message removal, as Activities of two Go workflows.
            activities=[
                # Six independent AI content operations, external retained payloads.
                # Codex / GPT-6.1-Sol / 2026-10-06.
                *AI_CONTENT_ACTIVITIES,
                # Context-only AI source stages; historical names stay registered
                # for Temporal replay while new requests use ai-context-v1.
                *AI_CONTEXT_ACTIVITIES,
                *AI_CONTEXT_GRAPH_ACTIVITIES,
                custody_activity,
                parse_activity,
                store_activity,
                knowledge_activity,
                n8n_webhook_activity,
                build_timeline_generation_activity,
                # One Activity per HTML text-extraction library (Claude Code · Sonnet · 2026-10-02).
                *HTML_TOOL_ACTIVITIES,
                chunk_context_threads_activity,
                publish_context_chunks_activity,
                publish_call_log_files_activity,
                list_context_threads_activity,
                estimate_context_chunks_activity,
                verify_chunk_coverage_activity,
                remove_per_message_objects_activity,
                # The selectable entity/event extractors the Go extraction_request_workflow schedules, one per
                # library (Claude Code · Sonnet 5.5 · 2026-10-02).
                extract_entities_events_semantica_activity,
                extract_entities_events_langextract_activity,
            ],
            activity_executor=executor,
            # Keep admitted Activities within the configured bounded executor;
            # the trial worker sets TEMPORAL_ACTIVITY_THREADS=1.
            max_concurrent_activities=threads,
        )
        log.info("worker running — workflows: ChatTranscriptIngest, P0DurabilityProbe, ClassificationBatchPipeline")
        await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
