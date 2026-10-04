"""The Super Index's Temporal Activities and the worker that runs them (queue ``superindex``).

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The Go engine's ``superindex.CycleWorkflow`` (Probata modules/engine/superindex) sequences these;
each Activity is a
thin wrapper over one function in stage_runner.py and does nothing else (the ATOMICITY rules of
modules/Probata/probata
AGENTS.md: one unit, one job; references in, small results out; safely retryable; heartbeats).
Importing this module
has no side effects beyond temporalio; everything heavy is imported inside the bodies.

    superindex_discover_activity    catalog watermark + classification
    superindex_extract_activity     CocoIndex incremental pass: extract + chunk (a child process;
    killed on cancel)
    superindex_summarize_activity   the summary pass
    superindex_embed_activity       one vector slot
    superindex_publish_activity     chunks + vectors -> Weaviate; moved files
    superindex_graph_activity       the active snapshot -> surreal-intake file graph (at most daily)
    superindex_commit_activity      advance the watermark
    superindex_image_fetch_activity    a bounded slice of images or scanned PDF pages, bucket ->
    spool
    superindex_image_facts_activity    original time, device, screenshot / photo / scan
    superindex_image_ocr_activity      the text in the pictures
    superindex_image_embed_activity    one image vector slot (maxsim, single, colqwen)
    superindex_image_publish_activity  slice -> Weaviate IntakeImageV1, release the spool

Run:  python -m casebible_index.temporal_worker      (the ``superindex-worker`` compose service)

Env (names only): TEMPORAL_ADDRESS, TEMPORAL_NAMESPACE (default "default"), TEMPORAL_TASK_QUEUE
(default
"superindex"), plus every INTAKE_* / NIM_* / NVIDIA_API_KEY the stages read.
"""

from __future__ import annotations

import asyncio
import logging
import os
from dataclasses import dataclass, field
from typing import Any

from temporalio import activity
from temporalio.exceptions import ApplicationError

DISCOVER_ACTIVITY = "superindex_discover_activity"
EXTRACT_ACTIVITY = "superindex_extract_activity"
SUMMARIZE_ACTIVITY = "superindex_summarize_activity"
EMBED_ACTIVITY = "superindex_embed_activity"
PUBLISH_ACTIVITY = "superindex_publish_activity"
GRAPH_ACTIVITY = "superindex_graph_activity"
COMMIT_ACTIVITY = "superindex_commit_activity"
IMAGE_FETCH_ACTIVITY = "superindex_image_fetch_activity"
IMAGE_FACTS_ACTIVITY = "superindex_image_facts_activity"
IMAGE_OCR_ACTIVITY = "superindex_image_ocr_activity"
IMAGE_EMBED_ACTIVITY = "superindex_image_embed_activity"
IMAGE_PUBLISH_ACTIVITY = "superindex_image_publish_activity"

ACTIVITY_NAMES = (
    DISCOVER_ACTIVITY,
    EXTRACT_ACTIVITY,
    SUMMARIZE_ACTIVITY,
    EMBED_ACTIVITY,
    PUBLISH_ACTIVITY,
    GRAPH_ACTIVITY,
    COMMIT_ACTIVITY,
    IMAGE_FETCH_ACTIVITY,
    IMAGE_FACTS_ACTIVITY,
    IMAGE_OCR_ACTIVITY,
    IMAGE_EMBED_ACTIVITY,
    IMAGE_PUBLISH_ACTIVITY,
)
TASK_QUEUE_DEFAULT = "superindex"

log = logging.getLogger("superindex.worker")


@dataclass
class StageRequest:
    """The Go ``superindex.StageRequest`` (JSON field names match)."""

    cycle_id: str = ""
    params: dict[str, Any] = field(default_factory=dict)


def _beat(detail: str) -> None:
    if activity.in_activity():
        activity.heartbeat(detail)


async def _keepalive(stage: str) -> None:
    """A stage that is working but silent (a long query) still proves the worker is alive."""
    while True:
        await asyncio.sleep(30)
        _beat(f"{stage}: working")


async def _run(stage: str, request: StageRequest) -> dict[str, Any]:
    from . import stage_runner

    if not request.cycle_id.strip():
        raise ApplicationError(f"{stage} requires a cycle id", non_retryable=True)
    keepalive = asyncio.create_task(_keepalive(stage))
    try:
        result = await stage_runner.STAGES[stage](request.cycle_id, request.params or {}, _beat)
    except ValueError as error:  # configuration or request errors do not heal by retrying
        raise ApplicationError(f"{stage}: {error}", non_retryable=True) from error
    finally:
        keepalive.cancel()
    activity.logger.info("%s cycle %s done", stage, request.cycle_id)
    return result


@activity.defn(name=DISCOVER_ACTIVITY)
async def superindex_discover_activity(request: StageRequest) -> dict[str, Any]:
    """Discover catalog changes and classify the represented source inventory.

    Inputs: StageRequest cycle reference and catalog selection params.
    Outputs: counted discovery result and watermark comparison.
    Side effects: reads the catalog and writes inventory Parquet and a stage receipt.
    Pick this before extraction to identify source changes without reading bucket payloads.
    """
    return await _run("discover", request)


@activity.defn(name=EXTRACT_ACTIVITY)
async def superindex_extract_activity(request: StageRequest) -> dict[str, Any]:
    """Extract source text and chunk it within the configured process budget.

    Inputs: StageRequest cycle reference and extraction bounds.
    Outputs: counted extraction result with process status and resource measurements.
    Side effects: runs a bounded child process, reads bucket objects and writes the text lake.
    Pick this for source parsing before summary or embedding of retained derived text.
    """
    return await _run("extract", request)


@activity.defn(name=SUMMARIZE_ACTIVITY)
async def superindex_summarize_activity(request: StageRequest) -> dict[str, Any]:
    """Summarize eligible retained documents independently of extraction.

    Inputs: StageRequest cycle reference and summary selection params.
    Outputs: counted summary result with remaining work.
    Side effects: reads document Parquet, calls the summary provider and writes summary receipts.
    Pick this to fill document summaries without parsing the source again.
    """
    return await _run("summarize", request)


@activity.defn(name=EMBED_ACTIVITY)
async def superindex_embed_activity(request: StageRequest) -> dict[str, Any]:
    """Fill one configured text embedding slot for retained chunks.

    Inputs: StageRequest cycle reference and a named text slot with selection bounds.
    Outputs: counted embedding result and remaining work.
    Side effects: reads chunk Parquet, calls the embedding provider and writes vector Parquet.
    Pick this for text vectors; image vectors use the image embedding Activity.
    """
    return await _run("embed", request)


@activity.defn(name=PUBLISH_ACTIVITY)
async def superindex_publish_activity(request: StageRequest) -> dict[str, Any]:
    """Publish retained text chunks and vectors to the configured search collection.

    Inputs: StageRequest cycle reference and publication bounds.
    Outputs: counted publication result, including locator changes.
    Side effects: reads the lake, upserts Weaviate text objects and writes a stage receipt.
    Pick this after text embedding; image publication uses its separate collection and ledger.
    """
    return await _run("publish", request)


@activity.defn(name=GRAPH_ACTIVITY)
async def superindex_graph_activity(request: StageRequest) -> dict[str, Any]:
    """Project the active source snapshot into Intake's file relationship graph.

    Inputs: StageRequest cycle reference and graph refresh params.
    Outputs: counted graph result or a refresh interval skip reason.
    Side effects: reads retained source state, writes the Surreal file graph and a stage receipt.
    Pick this for file relationships after publication, rather than vector search materialization.
    """
    return await _run("graph", request)


@activity.defn(name=COMMIT_ACTIVITY)
async def superindex_commit_activity(request: StageRequest) -> dict[str, Any]:
    """Advance the catalog watermark after the cycle's required stages complete.

    Inputs: StageRequest cycle reference and commit params.
    Outputs: counted commit result.
    Side effects: writes the accepted watermark and commit receipt in the owning output folder.
    Pick this as the last production stage; bounded proof runs leave the watermark unchanged.
    """
    return await _run("commit", request)


@activity.defn(name=IMAGE_FETCH_ACTIVITY)
async def superindex_image_fetch_activity(request: StageRequest) -> dict[str, Any]:
    """Fetch a bounded image slice or scanned PDF page window from approved bucket objects.

    Inputs: StageRequest cycle reference, locator or prefix fence and remaining item budget.
    Outputs: counted slice result with manifest reference, enabled slots and remaining work.
    Side effects: reads bucket objects, renders PDF pages and writes retained spool and manifest.
    Pick this before image facts; text sources use the extraction Activity instead.
    """
    return await _run("image_fetch", request)


@activity.defn(name=IMAGE_FACTS_ACTIVITY)
async def superindex_image_facts_activity(request: StageRequest) -> dict[str, Any]:
    """Read original time, device, dimensions and image kind for a retained image slice.

    Inputs: StageRequest cycle reference and image slice_id.
    Outputs: counted image facts result.
    Side effects: reads spooled images and writes facts Parquet and a stage receipt.
    Pick this before image embedding to apply screenshot, scan and photo coverage policies.
    """
    return await _run("image_facts", request)


@activity.defn(name=IMAGE_OCR_ACTIVITY)
async def superindex_image_ocr_activity(request: StageRequest) -> dict[str, Any]:
    """Read text visible in a retained image slice using the selected OCR engine.

    Inputs: StageRequest cycle reference, image slice_id and optional engine override.
    Outputs: counted OCR result with text coverage and confidence measurements.
    Side effects: runs OCR over spool files and writes OCR Parquet and a stage receipt.
    Pick this for image-derived text; native exported message text stays in the text lane.
    """
    return await _run("image_ocr", request)


@activity.defn(name=IMAGE_EMBED_ACTIVITY)
async def superindex_image_embed_activity(request: StageRequest) -> dict[str, Any]:
    """Fill one configured image vector slot for the slice's eligible images.

    Inputs: StageRequest cycle reference, image slice_id and named image slot.
    Outputs: counted embedding result with coverage and provider failures.
    Side effects: reads spool files and facts, calls the embedder and writes vector Parquet.
    Pick this once per enabled image slot; text slots use the text embedding Activity.
    """
    return await _run("image_embed", request)


@activity.defn(name=IMAGE_PUBLISH_ACTIVITY)
async def superindex_image_publish_activity(request: StageRequest) -> dict[str, Any]:
    """Publish retained image facts, OCR and vectors to the image search collection.

    Inputs: StageRequest cycle reference and image slice_id.
    Outputs: counted publication result with failures and quarantine file and byte totals.
    Side effects: verifies schema, upserts Weaviate images, writes the ledger and moves spool
    material into the owning output folder's quarantine without deleting it.
    Pick this after the image slots finish; text publication uses a separate collection.
    """
    return await _run("image_publish", request)


ACTIVITIES = [
    superindex_discover_activity,
    superindex_extract_activity,
    superindex_summarize_activity,
    superindex_embed_activity,
    superindex_publish_activity,
    superindex_graph_activity,
    superindex_commit_activity,
    superindex_image_fetch_activity,
    superindex_image_facts_activity,
    superindex_image_ocr_activity,
    superindex_image_embed_activity,
    superindex_image_publish_activity,
]


async def main() -> None:
    from temporalio.client import Client
    from temporalio.worker import Worker

    logging.basicConfig(
        level=os.environ.get("LOG_LEVEL", "INFO"),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    address = os.environ.get("TEMPORAL_ADDRESS", "").strip()
    if not address:
        raise SystemExit("TEMPORAL_ADDRESS is required")
    namespace = os.environ.get("TEMPORAL_NAMESPACE", "default")
    queue = os.environ.get("TEMPORAL_TASK_QUEUE", TASK_QUEUE_DEFAULT)
    log.info("connecting to Temporal at %s (namespace=%s)", address, namespace)
    client = await Client.connect(address, namespace=namespace)
    # One Activity of each kind at a time per stage that touches the lake: the stages write
    # immutable Parquet and
    # the extract pass is a heavy child process. Embed slots may run in parallel (different slot
    # directories).
    worker = Worker(
        client,
        task_queue=queue,
        activities=ACTIVITIES,
        max_concurrent_activities=int(os.environ.get("SUPERINDEX_MAX_CONCURRENT_ACTIVITIES", "4")),
    )
    log.info("worker running on queue %r: %s", queue, ", ".join(ACTIVITY_NAMES))
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
