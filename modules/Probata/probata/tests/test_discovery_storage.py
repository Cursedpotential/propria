"""Verify immutable external scoring, process restart, corruption and path bounds."""

import asyncio
import hashlib
import json
import os
import subprocess
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

import pytest
from temporalio.exceptions import ApplicationError

from server.analysis.discovery import storage
from server.temporal.discovery_activities import score_discovery_batch_activity, score_retained_batch
from tests.test_discovery_scoring import FixtureEncoder, batch


def retain_input(directory, snapshot):
    """Retain a synthetic externally written batch under the configured test root.

    Inputs: directory/snapshot. Outputs: immutable reference. Effects: test file
    write; choose for the real external-reader boundary rather than mocking I/O.
    """
    data = storage.encode(snapshot)
    path = directory / (hashlib.sha256(data).hexdigest() + ".json")
    path.write_bytes(data)
    return {"uri": path.as_uri(), "sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data)}


@pytest.fixture
def params(tmp_path, monkeypatch):
    """Bind one existing scratch root and reference-only test request.

    Inputs: pytest temporary path. Outputs: request. Effects: test env/file write;
    choose disposable storage rather than creating local infrastructure.
    """
    monkeypatch.setenv("ADAPTIVE_DISCOVERY_ROOT", str(tmp_path))
    return {"run_id": "run", "batch_ref": retain_input(tmp_path, batch()), "feedback_revision": "feedback-v1"}


def test_restart_replays_exact_result_without_loading_or_calling_encoder(params):
    initial = score_retained_batch(params, encoder=FixtureEncoder())
    script = """
import json, sys
from server.temporal.discovery_activities import score_retained_batch
class NoCalls:
    def encode(self, *args, **kwargs):
        raise AssertionError('replay must not load or call model')
print(json.dumps(score_retained_batch(json.loads(sys.argv[1]), encoder=NoCalls()), sort_keys=True))
"""
    completed = subprocess.run(
        [sys.executable, "-c", script, json.dumps(params)],
        check=True,
        capture_output=True,
        text=True,
        env=os.environ.copy(),
    )
    assert json.loads(completed.stdout) == initial
    assert set(initial) == {"run_id", "result_ref", "candidate_count", "feedback_revision", "model_version"}
    assert "reported threat" not in json.dumps(initial)


def test_concurrent_external_writers_return_one_exact_receipt(params):
    with ThreadPoolExecutor(max_workers=4) as pool:
        receipts = list(pool.map(lambda _: score_retained_batch(params, encoder=FixtureEncoder()), range(4)))
    assert all(receipt == receipts[0] for receipt in receipts)
    assert len(list(storage.root().glob("results/*.json"))) == 1
    assert storage.read(receipts[0]["result_ref"])["candidates"]


def test_changed_feedback_or_input_revisions_use_new_result_identity(params, tmp_path):
    first = score_retained_batch(params, encoder=FixtureEncoder())
    snapshot = batch()
    snapshot["feedback_revision"] = "feedback-v2"
    params["feedback_revision"] = "feedback-v2"
    params["batch_ref"] = retain_input(tmp_path, snapshot)
    second = score_retained_batch(params, encoder=FixtureEncoder())
    assert first["result_ref"] != second["result_ref"]
    snapshot["identity_revision"] = "identity-v2"
    params["batch_ref"] = retain_input(tmp_path, snapshot)
    assert score_retained_batch(params, encoder=FixtureEncoder())["result_ref"] != second["result_ref"]


def test_retained_result_corruption_is_not_silently_replayed(params):
    receipt = score_retained_batch(params, encoder=FixtureEncoder())
    path = storage.safe_path(receipt["result_ref"]["uri"])
    value = storage.read(receipt["result_ref"])
    value["candidates"][0]["score"] = 0.01
    path.chmod(0o600)
    path.write_bytes(storage.encode(value))
    with pytest.raises(storage.DiscoveryInvalid, match="pins"):
        score_retained_batch(params, encoder=FixtureEncoder())


def test_references_reject_bad_digest_byte_size_and_symlink_escapes(params, tmp_path):
    wrong = dict(params["batch_ref"], sha256="0" * 64)
    with pytest.raises(storage.DiscoveryInvalid, match="digest"):
        storage.read(wrong)
    with pytest.raises(storage.DiscoveryInvalid, match="bytes"):
        storage.read(dict(params["batch_ref"], bytes=storage.MAX_BYTES + 1))
    link = tmp_path / "escape.json"
    link.symlink_to(Path(params["batch_ref"]["uri"].removeprefix("file://")))
    with pytest.raises(storage.DiscoveryInvalid, match="symlinks"):
        storage.read(dict(params["batch_ref"], uri=link.as_uri()))
    with pytest.raises(storage.DiscoveryInvalid, match="escaped"):
        storage.safe_path((tmp_path / ".." / "outside.json").as_uri())


def test_atomic_writer_refuses_different_bytes_for_same_identity(params):
    storage.save({"one": 1}, "same-key")
    with pytest.raises(storage.DiscoveryInvalid, match="different"):
        storage.save({"one": 2}, "same-key")


def test_activity_classifies_bad_input_nonretryable_before_cpu_load(params):
    invalid = dict(params, feedback_revision="wrong")
    with pytest.raises(ApplicationError) as caught:
        asyncio.run(score_discovery_batch_activity(invalid))
    assert caught.value.non_retryable
    assert caught.value.type == "DiscoveryInvalid"


def test_extra_activity_payload_is_rejected_to_keep_history_reference_only(params):
    with pytest.raises(storage.DiscoveryInvalid, match="only"):
        score_retained_batch(dict(params, records=[{"text": "body"}]), encoder=FixtureEncoder())


def test_explicit_existing_root_is_required(monkeypatch):
    monkeypatch.delenv("ADAPTIVE_DISCOVERY_ROOT", raising=False)
    with pytest.raises(storage.DiscoveryInvalid, match="existing absolute"):
        storage.root()


def test_shared_worker_limits_cpu_model_calls_to_one_at_a_time(params, tmp_path):
    class TrackingEncoder(FixtureEncoder):
        """Track concurrent synthetic CPU calls without using a real model.

        Inputs: fixture text. Outputs: fixture vectors. Effects: bounded simulated
        CPU delay and counters; choose to verify the shared worker resource gate.
        """

        def __init__(self):
            super().__init__()
            self.active = self.maximum = 0
            self.lock = threading.Lock()

        def encode(self, sentences, **kwargs):
            """Observe concurrent entry during independently scheduled scoring calls.

            Inputs: sentences/options. Outputs: vectors. Effects: counter update;
            choose to verify the process semaphore rather than timing an inference.
            """
            with self.lock:
                self.active += 1
                self.maximum = max(self.maximum, self.active)
            try:
                time.sleep(0.01)
                return super().encode(sentences, **kwargs)
            finally:
                with self.lock:
                    self.active -= 1

    requests = []
    for index in range(3):
        snapshot = batch()
        snapshot["run_id"] = "concurrent-" + str(index)
        requests.append(dict(params, run_id=snapshot["run_id"], batch_ref=retain_input(tmp_path, snapshot)))
    encoder = TrackingEncoder()
    with ThreadPoolExecutor(max_workers=3) as pool:
        receipts = list(pool.map(lambda request: score_retained_batch(request, encoder=encoder), requests))
    assert len(receipts) == 3
    assert encoder.maximum == 1


def test_active_activity_heartbeats_safe_labels_and_stops_thread(monkeypatch):
    from server.temporal import discovery_activities

    heartbeats = []
    monkeypatch.setattr(discovery_activities.activity, "in_activity", lambda: True)
    monkeypatch.setattr(discovery_activities.activity, "is_cancelled", lambda: False)
    monkeypatch.setattr(discovery_activities.activity, "heartbeat", lambda *details: heartbeats.append(details))
    monkeypatch.setattr(discovery_activities, "score_retained_batch", lambda _: {"candidate_count": 0})
    assert asyncio.run(discovery_activities.score_discovery_batch_activity({})) == {"candidate_count": 0}
    assert heartbeats == [("bounded discovery scoring",), ("bounded discovery scoring",)]
    assert not any(thread.name == "discovery-heartbeat" for thread in threading.enumerate())


def test_async_discovery_waiters_leave_shared_evidence_executor_available(monkeypatch):
    from server.temporal import discovery_activities

    entered = threading.Event()
    release = threading.Event()
    calls = []

    def blocked_score(request):
        """Hold synthetic model execution until unrelated evidence work completes.

        Inputs: reference fixture. Outputs: fixture receipt. Effects: bounded test
        event wait; choose to observe executor isolation without real inference.
        """
        calls.append(threading.current_thread().name)
        entered.set()
        assert release.wait(5)
        return {"candidate_count": 0}

    monkeypatch.setattr(discovery_activities, "score_retained_batch", blocked_score)

    async def exercise():
        """Schedule concurrent discovery waits beside shared synchronous evidence work.

        Inputs: test executor/events. Outputs: none. Effects: temporary concurrent
        tasks; choose to prove discovery cannot occupy shared evidence threads.
        """
        monkeypatch.setattr(discovery_activities, "_ASYNC_CPU_SLOT", asyncio.Semaphore(1))
        jobs = [asyncio.create_task(score_discovery_batch_activity({})) for _ in range(8)]
        try:
            assert await asyncio.to_thread(entered.wait, 1)
            with ThreadPoolExecutor(max_workers=1, thread_name_prefix="shared-evidence") as shared:
                result = await asyncio.wait_for(
                    asyncio.get_running_loop().run_in_executor(shared, lambda: "custody-parser-store-completed"),
                    timeout=1,
                )
                assert result == "custody-parser-store-completed"
            assert len(calls) == 1
        finally:
            release.set()
        await asyncio.gather(*jobs)

    asyncio.run(exercise())
    assert len(calls) == 8
    assert all(name.startswith("discovery-cpu") for name in calls)


def test_cancelled_async_waiter_does_not_admit_another_cpu_job(monkeypatch):
    from server.temporal import discovery_activities

    entered = threading.Event()
    release = threading.Event()
    calls = []

    def blocked_score(request):
        """Keep one synthetic CPU operation running while a queued waiter cancels.

        Inputs: fixture request. Outputs: receipt. Effects: bounded event wait;
        choose to verify cancelled admission consumes no executor thread.
        """
        calls.append(request)
        entered.set()
        assert release.wait(5)
        return {"candidate_count": 0}

    monkeypatch.setattr(discovery_activities, "score_retained_batch", blocked_score)

    async def exercise():
        """Cancel a waiting Activity while retaining one actual CPU operation.

        Inputs: events and monkeypatched scorer. Outputs: none. Effects: test
        cancellation; choose to verify asynchronous waiting releases cleanly.
        """
        monkeypatch.setattr(discovery_activities, "_ASYNC_CPU_SLOT", asyncio.Semaphore(1))
        first = asyncio.create_task(score_discovery_batch_activity({"request": "first"}))
        try:
            assert await asyncio.to_thread(entered.wait, 1)
            waiter = asyncio.create_task(score_discovery_batch_activity({"request": "cancelled-waiter"}))
            await asyncio.sleep(0)
            waiter.cancel()
            with pytest.raises(asyncio.CancelledError):
                await waiter
            assert calls == [{"request": "first"}]
        finally:
            release.set()
        await first

    asyncio.run(exercise())
