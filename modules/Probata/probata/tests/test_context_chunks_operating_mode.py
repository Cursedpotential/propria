"""Import-light boundary tests; no DB, embedding, vector, or Temporal service is used.

Byline: Codex · GPT-5 · 2026-10-05
"""

from __future__ import annotations

import asyncio
import builtins
import sys
from contextlib import contextmanager
from dataclasses import asdict
from types import ModuleType, SimpleNamespace
from uuid import uuid4

import pytest
from temporalio.converter import DataConverter
from temporalio.exceptions import ApplicationError

from server.context_chunks import start as starter
from server.temporal import chunk_activities as chunks
from server.temporal import chunk_backfill_activities as backfill
from server.temporal.chunk_write_guard import canonical_operating_mode, configured_case_scope


@pytest.fixture
def approved(monkeypatch):
    # Ephemeral unit-only UUIDs are never persisted, dispatched, or used as production defaults.
    pair = str(uuid4()), str(uuid4())
    monkeypatch.setenv("PROFFER_MATTER_ID", pair[0])
    monkeypatch.setenv("PROFFER_COURT_CASE_ID", pair[1])
    return {"matter_id": pair[0], "court_case_id": pair[1]}


WRITERS = (
    (chunks.publish_context_chunks_activity, chunks.PublishChunksParams),
    (chunks.publish_call_log_files_activity, chunks.PublishCallLogFilesParams),
    (backfill.remove_per_message_objects_activity, backfill.RemovalParams),
)


def params_for(cls, **policy):
    """Build an in-memory mutation input from a dataclass and policy, without I/O.

    Outputs: params with destructive removal explicitly non-dry-run. Pick this helper
    for write-boundary tests rather than default read-only RemovalParams.
    """
    return cls(**policy, **({"dry_run": False} if cls is backfill.RemovalParams else {}))


def fence_imports(monkeypatch):
    """Reject body imports and return observations for a denied-activity assertion.

    Inputs: scoped pytest monkeypatch. Outputs: observed import names. Effects: temporary
    import interception only. Pick body_stubs instead when proving admitted behavior.
    """
    observed = []
    original = builtins.__import__

    def guarded(name, *args, **kwargs):
        if name.startswith("server.context_chunks"):
            observed.append(name)
            raise AssertionError(f"denied activity imported body dependency {name}")
        return original(name, *args, **kwargs)

    monkeypatch.setattr(builtins, "__import__", guarded)
    return observed


@pytest.mark.parametrize("fn,cls", WRITERS)
@pytest.mark.parametrize("mode", ["", " ", None, "DEV", "TEST", "unrecognized"])
def test_denied_modes_never_import_the_body(monkeypatch, approved, fn, cls, mode):
    params = params_for(cls, operating_mode=mode, **approved)
    imports = fence_imports(monkeypatch)
    with pytest.raises(ApplicationError) as error:
        fn(params)
    assert error.value.non_retryable and error.value.type == "ChunkWriteDenied"
    assert imports == []


@pytest.mark.parametrize("fn,cls", WRITERS)
@pytest.mark.parametrize(
    "defect", ["missing-input", "invalid-input", "mismatch", "no-env", "partial-env", "nil-env", "invalid-env"]
)
def test_denied_scope_never_imports_the_body(monkeypatch, approved, fn, cls, defect):
    policy = {"operating_mode": "LIVE", **approved}
    if defect == "missing-input":
        policy["matter_id"] = ""
    elif defect == "invalid-input":
        policy["court_case_id"] = "bad-uuid"
    elif defect == "mismatch":
        policy["court_case_id"] = str(uuid4())
    elif defect == "no-env":
        monkeypatch.delenv("PROFFER_MATTER_ID")
        monkeypatch.delenv("PROFFER_COURT_CASE_ID")
    elif defect == "partial-env":
        monkeypatch.delenv("PROFFER_COURT_CASE_ID")
    elif defect == "nil-env":
        monkeypatch.setenv("PROFFER_MATTER_ID", "00000000-0000-0000-0000-000000000000")
    else:
        monkeypatch.setenv("PROFFER_MATTER_ID", "bad-uuid")
    imports = fence_imports(monkeypatch)
    with pytest.raises(ApplicationError) as error:
        fn(params_for(cls, **policy))
    assert error.value.non_retryable
    assert imports == []


@pytest.fixture
def body_stubs(monkeypatch):
    """Replace body dependencies with in-memory calls for admitted-path verification.

    Inputs: scoped pytest monkeypatch. Outputs: call observations. Effects: temporary
    module/attribute stubs, never service I/O. Pick fence_imports for denial proof.
    """
    calls = []

    def install(name, **members):
        module = ModuleType(f"server.context_chunks.{name}")
        module.__dict__.update(members)
        monkeypatch.setitem(sys.modules, module.__name__, module)

    def config():
        calls.append("config")
        return SimpleNamespace(weaviate_url="stub-only", collection="stub", reuse_collection="")

    @contextmanager
    def connection():
        calls.append("db")
        yield object()

    def embedder(_):
        calls.append("embed")
        return SimpleNamespace(calls=1)

    def store(*_):
        calls.append("store")
        return object()

    def publish(*args, **kwargs):
        calls.append("publish")
        return SimpleNamespace(
            corpus="context",
            thread_id="thread",
            chunks_written=1,
            stale_deleted=0,
            embed_texts=1,
            skipped_existing=0,
            reused=0,
        )

    def publish_calls(*args, **kwargs):
        calls.append("publish-calls")
        return {"files": 1, "calls": 2}

    def remove(*args, **kwargs):
        calls.append(("remove", kwargs))
        return {"dry_run": kwargs["dry_run"]}

    install("config", load_config=config)
    install("db", read_only_connection=connection)
    install("embed", NimEmbedder=embedder)
    install("store", ChunkStore=store)
    install("model", ThreadPlan=SimpleNamespace(from_dict=lambda _: SimpleNamespace(thread_id="thread")))
    install("service", PlanStale=RuntimeError, publish_thread=publish, publish_call_files=publish_calls)
    install("source", PgSource=lambda _: object())
    install("remove_per_message", NotVerified=RuntimeError, remove=remove)
    monkeypatch.setattr(chunks, "_source", lambda *_: object())
    monkeypatch.setattr(backfill, "_stores", lambda *_: (store(), store()))
    return calls


@pytest.mark.parametrize("mode", ["LIVE", " live ", "REAL"])
@pytest.mark.parametrize("fn,cls", WRITERS)
def test_admitted_live_runs_expected_stubbed_body(approved, body_stubs, fn, cls, mode):
    params = params_for(cls, operating_mode=mode, **approved)
    if cls is chunks.PublishCallLogFilesParams:
        params.source_version_id = "unit-only-source-reference"
    result = fn(params)
    assert "db" in body_stubs and "store" in body_stubs
    if cls is backfill.RemovalParams:
        assert result == {"dry_run": False}
        assert body_stubs[-1] == ("remove", {"dry_run": False, "only_covered": False})
    else:
        assert "config" in body_stubs and "embed" in body_stubs
        assert result["embed_requests"] == 1


def test_read_only_removal_needs_no_mode_or_scope(monkeypatch, body_stubs):
    monkeypatch.delenv("PROFFER_MATTER_ID", raising=False)
    monkeypatch.delenv("PROFFER_COURT_CASE_ID", raising=False)
    assert backfill.remove_per_message_objects_activity(backfill.RemovalParams()) == {"dry_run": True}
    assert body_stubs[-1] == ("remove", {"dry_run": True, "only_covered": False})


def test_read_only_siblings_do_not_require_write_authorization(monkeypatch, body_stubs):
    def denied(*_):
        raise AssertionError("read-only sibling requested write admission")

    monkeypatch.setattr(chunks, "require_live_chunk_write", denied)
    monkeypatch.setattr(backfill, "require_live_chunk_write", denied)
    config = sys.modules["server.context_chunks.config"]
    config.DEFAULT_CHUNKER, config.DEFAULT_OVERLAP = "stub", 4
    model = sys.modules["server.context_chunks.model"]
    model.ThreadRef = lambda *args: args
    service = sys.modules["server.context_chunks.service"]
    service.plan_source_version = lambda *args, **kwargs: []
    service.plan_threads = lambda *args, **kwargs: []
    rechunk = ModuleType("server.context_chunks.rechunk")
    rechunk.list_threads = lambda *args, **kwargs: ([], [])
    rechunk.dry_run = lambda *args, **kwargs: {"read_only": True}
    monkeypatch.setitem(sys.modules, rechunk.__name__, rechunk)
    removal = sys.modules["server.context_chunks.remove_per_message"]
    removal.chunk_coverage = lambda _: (set(), set())
    removal.verify = lambda *_: {"read_only": True}
    removal.public = lambda report: report

    plan = chunks.chunk_context_threads_activity(
        chunks.ChunkThreadsParams(source_version_id="unit-only-reference", operating_mode="DEV")
    )
    assert plan["threads"] == []
    assert backfill.list_context_threads_activity(backfill.ListThreadsParams()) == {
        "threads": [],
        "call_source_versions": [],
    }
    assert backfill.estimate_context_chunks_activity(backfill.EstimateParams()) == {"read_only": True}
    assert backfill.verify_chunk_coverage_activity(backfill.RemovalParams()) == {"read_only": True}


@pytest.mark.parametrize(
    "cls",
    [chunks.ChunkThreadsParams, chunks.PublishChunksParams, chunks.PublishCallLogFilesParams, backfill.RemovalParams],
)
def test_historical_json_decodes_with_missing_authority_not_live(cls):
    async def decode():
        converter = DataConverter.default
        payloads = await converter.encode([{}])
        return (await converter.decode(payloads, [cls]))[0]

    decoded = asyncio.run(decode())
    assert decoded.operating_mode == decoded.matter_id == decoded.court_case_id == ""
    assert asdict(decoded)["operating_mode"] != "LIVE"


@pytest.mark.parametrize("command", ["rechunk", "remove"])
def test_cli_default_and_explicit_mode_preserve_same_scope(approved, command):
    _, _, live = starter.build_input(starter.parser().parse_args([command, "--dry-run"]))
    _, _, dev = starter.build_input(starter.parser().parse_args([command, "--dry-run", "--operating-mode", "DEV"]))
    assert live["operating_mode"] == "LIVE" and dev["operating_mode"] == "DEV"
    assert {k: live[k] for k in approved} == {k: dev[k] for k in approved} == approved


def test_cli_admitted_live_dispatch_preserves_scope_and_flags(monkeypatch, approved):
    observed = []

    class Client:
        @staticmethod
        async def connect(address, **kwargs):
            observed.append("connect")
            return Client()

        async def start_workflow(self, name, body, **kwargs):
            observed.append((name, body, kwargs))
            return SimpleNamespace(id=kwargs["id"], result_run_id="stub-only")

    module = ModuleType("temporalio.client")
    module.Client = Client
    monkeypatch.setitem(sys.modules, module.__name__, module)
    args = starter.parser().parse_args(["remove", "--only-covered", "--no-wait", "--request-id", "unit-only"])
    assert asyncio.run(starter.start(args)) == 0
    assert observed[0] == "connect"
    name, body, options = observed[1]
    assert name == starter.REMOVAL_WORKFLOW
    assert body == {
        "operating_mode": "LIVE",
        **approved,
        "request_id": "unit-only",
        "dry_run": False,
        "only_covered": True,
        "old_collection": "",
    }
    assert options["id"].endswith("unit-only")


@pytest.mark.parametrize("command", ["rechunk", "remove"])
@pytest.mark.parametrize("defect", ["DEV", "missing-scope", "unknown-mode"])
def test_cli_denies_before_temporal_import_or_connect(monkeypatch, approved, command, defect):
    args = [command, "--operating-mode", "DEV" if defect == "DEV" else "LIVE"]
    if defect == "missing-scope":
        monkeypatch.delenv("PROFFER_MATTER_ID")
    elif defect == "unknown-mode":
        args[-1] = "unknown"
    original = builtins.__import__

    def no_client(name, *a, **kw):
        assert name != "temporalio.client", "denied starter imported the Temporal client"
        return original(name, *a, **kw)

    monkeypatch.setattr(builtins, "__import__", no_client)
    with pytest.raises((ValueError, ApplicationError)):
        asyncio.run(starter.start(starter.parser().parse_args(args)))


def test_read_only_cli_can_be_unconfigured_and_old_namespace_defaults_fresh_live(monkeypatch):
    monkeypatch.delenv("PROFFER_MATTER_ID", raising=False)
    monkeypatch.delenv("PROFFER_COURT_CASE_ID", raising=False)
    args = starter.parser().parse_args(["remove", "--dry-run"])
    del args.operating_mode
    body = starter.build_input(args)[2]
    assert body["operating_mode"] == "LIVE" and body["matter_id"] == body["court_case_id"] == ""


def test_no_mode_specific_configuration_fallback(monkeypatch):
    monkeypatch.delenv("PROFFER_MATTER_ID", raising=False)
    monkeypatch.delenv("PROFFER_COURT_CASE_ID", raising=False)
    monkeypatch.setenv("PROFFER_TEST_MATTER_ID", str(uuid4()))
    monkeypatch.setenv("PROFFER_TEST_COURT_CASE_ID", str(uuid4()))
    with pytest.raises(ValueError):
        configured_case_scope()
    assert canonical_operating_mode("TEST") == "DEV"
