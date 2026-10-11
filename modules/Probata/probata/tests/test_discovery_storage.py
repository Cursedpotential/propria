"""Verify immutable external scoring, process restart, corruption and path bounds."""

import hashlib
import json
import os
import subprocess
import sys
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
        score_discovery_batch_activity(invalid)
    assert caught.value.non_retryable
    assert caught.value.type == "DiscoveryInvalid"


def test_extra_activity_payload_is_rejected_to_keep_history_reference_only(params):
    with pytest.raises(storage.DiscoveryInvalid, match="only"):
        score_retained_batch(dict(params, records=[{"text": "body"}]), encoder=FixtureEncoder())


def test_explicit_existing_root_is_required(monkeypatch):
    monkeypatch.delenv("ADAPTIVE_DISCOVERY_ROOT", raising=False)
    with pytest.raises(storage.DiscoveryInvalid, match="existing absolute"):
        storage.root()
