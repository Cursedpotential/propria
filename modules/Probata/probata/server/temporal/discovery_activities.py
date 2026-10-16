# Byline: Codex · GPT-6.1-sol · 2026-10-11
"""Expose one reference-only CPU scoring operation on the evidence pipeline.

Inputs: run ID, immutable batch reference and feedback revision. Outputs: retained
result reference and bounded counts. Effects: scratch files and local CPU only;
choose downstream optional discovery without intake gates or canonical writes.
"""

from __future__ import annotations

import asyncio
import hashlib
from collections.abc import Iterator
from concurrent.futures import Future, ThreadPoolExecutor
from contextlib import contextmanager
from contextvars import copy_context
from threading import BoundedSemaphore, Event, Thread
from typing import Any

from temporalio import activity
from temporalio.exceptions import ApplicationError, CancelledError

from server.analysis.discovery import storage
from server.analysis.discovery.scoring import MODEL_VERSION, SCORE_REVISION, model_config, score_batch

_CPU_SLOT = BoundedSemaphore(1)
_ASYNC_CPU_SLOT = asyncio.Semaphore(1)
_CPU_EXECUTOR = ThreadPoolExecutor(max_workers=1, thread_name_prefix="discovery-cpu")


@contextmanager
def _heartbeats() -> Iterator[None]:
    """Keep bounded synchronous CPU model loading/scoring alive in Temporal.

    Inputs: active Activity context. Outputs: managed heartbeat lifetime. Effects:
    safe-label heartbeat thread stopped on exit; choose for local model calls
    that can exceed heartbeat timeout, never source text in heartbeat details.
    """
    active = activity.in_activity()
    stop = Event()

    def beat() -> None:
        """Report safe discovery progress and detect cancellation between CPU calls.

        Inputs: active Activity context. Outputs: none. Effects: safe-label
        heartbeat; choose over embedding records in progress or workflow history.
        """
        if active:
            if activity.is_cancelled():
                raise CancelledError("discovery scoring Activity was cancelled")
            activity.heartbeat("bounded discovery scoring")

    def pulse() -> None:
        """Pulse within a captured context while synchronous local model work blocks.

        Inputs: captured context and stop event. Outputs: none. Effects: periodic
        safe heartbeat; choose bounded thread lifetime over an orphan worker.
        """
        while not stop.wait(10):
            if not activity.is_cancelled():
                activity.heartbeat("bounded discovery scoring")

    context = copy_context()
    thread = Thread(target=lambda: context.run(pulse), daemon=True, name="discovery-heartbeat")
    if active:
        beat()
        thread.start()
    try:
        yield
        beat()
    finally:
        stop.set()
        if active:
            thread.join(timeout=2)


def score_retained_batch(params: dict[str, Any], *, encoder: Any = None) -> dict[str, Any]:
    """Score or exactly replay a pinned external batch without carrying source bodies.

    Inputs: run_id/batch_ref/feedback_revision and optional encoder. Outputs:
    result_ref/count/model receipt. Effects: bounded scratch reads and immutable
    derived writes; choose direct invocation for isolated writer/restart tests.
    """
    if set(params) != {"run_id", "batch_ref", "feedback_revision"}:
        raise storage.DiscoveryInvalid("discovery Activity requires only run_id, batch_ref and feedback_revision")
    batch = storage.read(params["batch_ref"])
    if batch.get("run_id") != params["run_id"] or batch.get("feedback_revision") != params["feedback_revision"]:
        raise storage.DiscoveryInvalid("discovery batch differs from Activity revision pins")
    identity = [params, MODEL_VERSION, SCORE_REVISION, model_config(batch, injected=encoder is not None)]
    existing = storage.root() / "results" / (storage.fingerprint(identity) + ".json")
    if existing.exists():
        path = storage.safe_path(existing.as_uri())
        with path.open("rb") as stream:
            data = stream.read(storage.MAX_BYTES + 1)

        ref = {"uri": path.as_uri(), "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
        result = storage.read(ref)
        if (
            result.get("run_id") != params["run_id"]
            or result.get("feedback_revision") != params["feedback_revision"]
            or result.get("request_fingerprint") != storage.fingerprint(identity)
            or result.get("result_fingerprint")
            != storage.fingerprint({key: value for key, value in result.items() if key != "result_fingerprint"})
        ):
            raise storage.DiscoveryInvalid("retained discovery result differs from pins")
    else:
        with _CPU_SLOT:
            result = score_batch(batch, encoder=encoder)
        result["request_fingerprint"] = storage.fingerprint(identity)
        result["result_fingerprint"] = storage.fingerprint(result)
        ref = storage.save(result, identity)
    return {
        "run_id": params["run_id"],
        "result_ref": ref,
        "candidate_count": len(result["candidates"]),
        "feedback_revision": params["feedback_revision"],
        "model_version": MODEL_VERSION,
    }


def _score_activity_sync(params: dict[str, Any]) -> dict[str, Any]:
    """Run CPU/file scoring with heartbeat context inside the dedicated executor.

    Inputs: reference-only request. Outputs: bounded score receipt. Effects:
    local CPU/files/heartbeats; choose this dedicated thread boundary so discovery
    never waits on the shared custody/parser/store Activity executor.
    """
    try:
        with _heartbeats():
            return score_retained_batch(params)
    except storage.DiscoveryInvalid as error:
        raise ApplicationError(str(error), type="DiscoveryInvalid", non_retryable=True) from None


@activity.defn(name="score_discovery_batch_activity")
async def score_discovery_batch_activity(params: dict[str, Any]) -> dict[str, Any]:
    """Score a pinned retained batch without occupying shared evidence executor threads.

    Inputs: reference-only Go request. Outputs: result reference and counts/model
    revision. Effects: async admission plus one dedicated CPU thread/file output;
    choose optional discovery on evidence-pipeline so waiting jobs and cancelled
    waiters leave custody, parsing and storage execution capacity available.
    """
    gate = _ASYNC_CPU_SLOT
    await gate.acquire()
    loop = asyncio.get_running_loop()
    context = copy_context()
    try:
        future = _CPU_EXECUTOR.submit(context.run, _score_activity_sync, params)
    except BaseException:
        gate.release()
        raise

    def release_slot(completed: Future[Any]) -> None:
        """Release CPU admission only when its actual thread operation finishes.

        Inputs: completed/cancelled executor future. Outputs: none. Effects:
        event-loop semaphore release; choose actual completion over freeing a
        running CPU slot merely because its awaiting Activity was cancelled.
        """
        try:
            loop.call_soon_threadsafe(gate.release)
        except RuntimeError:
            pass  # A closed worker loop has no surviving admission waiters.

    future.add_done_callback(release_slot)
    return await asyncio.wrap_future(future, loop=loop)


DISCOVERY_ACTIVITIES = [score_discovery_batch_activity]
