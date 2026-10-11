"""Run native SMS chunk planning and persist actual reference-only Temporal receipts.

Inputs are registered source/manifest refs. Outputs are durable chunk-plan refs/counts.
Effects: derivative plans plus existing context execution/receipt/metadata rows only.
Choose instead of normalized-generation chunk Activities for native human XML.
"""
from __future__ import annotations

import json
import os
import contextvars
import threading
from datetime import datetime, timezone
from pathlib import Path

from temporalio import activity

ACTIVITY_NAME = "chunk_context_human_source_activity"


def _recover(raw, request):
    """Validate complete native request coordinates before recovering an idempotent receipt."""
    metadata = json.loads(raw) if isinstance(raw, str) else raw
    result = metadata["result"]
    if metadata["contract"] != "human-source-v1" or metadata["workflow_id"] != request["workflow_id"] or metadata["request_source"] != request["source"] or metadata["input_manifest_ref"] != request["manifest_ref"] or result["stage"] != "chunked" or not result.get("receipt_ref"):
        raise ValueError("native chunk recovered request differs")
    return result


def _read_prior(engine, request):
    """Read a completed chunk receipt without creating files or holding a long transaction."""
    from sqlalchemy import text
    with engine.connect() as connection:
        raw = connection.execute(text("SELECT metadata FROM context.source_metadata WHERE source_version_id=CAST(:v AS uuid) AND extractor_id='context-human-chunked' AND metadata->>'workflow_id'=:w ORDER BY generated_at LIMIT 1"), {"v": request["source"]["source_version_ref"], "w": request["workflow_id"]}).scalar()
    return None if raw is None else _recover(raw, request)


def persist_chunk_receipt(engine, request, result, info, *, completed_at=None):
    """Persist actual Temporal attempt/start and completed plan refs in existing metadata tables.

    Inputs: registered request, immutable result, SDK Info and observed completion clock.
    Outputs: committed result with its real UUIDv7 receipt. Effects: one metadata-only transaction.
    Native bodies/locators remain in files; no schema changes or normalized message writes.
    """
    from sqlalchemy import text
    from uuid6 import uuid7
    if not info.started_time or info.attempt < 1 or info.workflow_id != request["workflow_id"]:
        raise ValueError("native Temporal activity coordinates unavailable or different")
    completed_at = completed_at or datetime.now(timezone.utc)
    params = {"v": request["source"]["source_version_ref"], "w": request["workflow_id"]}
    with engine.begin() as connection:
        registered = connection.execute(text("SELECT v.workflow_id,s.source_key FROM context.source_version v JOIN context.source s ON s.id=v.source_id WHERE v.id=CAST(:v AS uuid) FOR UPDATE OF v"), params).one()
        if registered[0] != params["w"] or registered[1] != request["source"]["source_ref"]:
            raise ValueError("native chunk registration differs")
        prior = connection.execute(text("SELECT metadata FROM context.source_metadata WHERE source_version_id=CAST(:v AS uuid) AND extractor_id='context-human-chunked' AND metadata->>'workflow_id'=:w ORDER BY generated_at LIMIT 1"), params).scalar()
        if prior is not None:
            return _recover(prior, request)
        connection.execute(text("INSERT INTO context.activity_execution (source_version_id,workflow_id,activity_name,idempotency_key) VALUES (CAST(:v AS uuid),:w,:name,:w) ON CONFLICT (source_version_id,activity_name,idempotency_key) DO NOTHING"), {**params, "name": ACTIVITY_NAME})
        execution = connection.execute(text("SELECT id,workflow_id FROM context.activity_execution WHERE source_version_id=CAST(:v AS uuid) AND activity_name=:name AND idempotency_key=:w FOR UPDATE"), {**params, "name": ACTIVITY_NAME}).one()
        if execution[1] != params["w"]:
            raise ValueError("native chunk execution workflow differs")
        result = {**result, "receipt_ref": str(uuid7())}
        receipt_persisted_at = datetime.now(timezone.utc)
        metadata = {"contract": "human-source-v1", "workflow_id": params["w"], "request_source": request["source"],
                    "input_manifest_ref": request["manifest_ref"], "result": result,
                    "temporal": {"workflow_id": info.workflow_id, "workflow_run_id": info.workflow_run_id,
                                 "activity_id": info.activity_id, "attempt": info.attempt,
                                 "started_at": info.started_time.isoformat(), "operation_completed_at": completed_at.isoformat(),
                                 "receipt_persisted_at": receipt_persisted_at.isoformat()}}
        connection.execute(text("INSERT INTO context.activity_receipt (id,activity_execution_id,attempt,status,started_at,completed_at,result_ref) VALUES (CAST(:id AS uuid),:execution,:attempt,'success',:started,:completed,CAST(:ref AS jsonb))"),
                           {"id": result["receipt_ref"], "execution": execution[0], "attempt": info.attempt, "started": info.started_time,
                            "completed": completed_at, "ref": json.dumps({"ref_kind": "human_context_bundle", "ref_id": result["manifest_ref"]})})
        connection.execute(text("INSERT INTO context.source_metadata (source_version_id,metadata_class,metadata,extractor_id,extractor_version,extraction_activity_receipt_id,generated_at) VALUES (CAST(:v AS uuid),'record_native',CAST(:metadata AS jsonb),'context-human-chunked','1.0.0',CAST(:receipt AS uuid),:generated)"),
                           {**params, "metadata": json.dumps(metadata), "receipt": result["receipt_ref"], "generated": receipt_persisted_at})
    return result


@activity.defn(name=ACTIVITY_NAME)
def chunk_context_human_source_activity(request: dict) -> dict:
    """Plan bounded Neural native chunks then persist their actual reference-only stage receipt.

    Inputs: registered extraction refs. Outputs: durable chunked result. Effects: derivative and metadata writes.
    Original source, normalized tables, search schemas and vectors remain untouched in this Activity.
    """
    from server.context_chunks.native_human import plan_native_chunks
    from server.timeline.db import get_engine
    root = os.environ.get("HUMAN_CONTEXT_BUNDLE_ROOT", "").strip()
    if not root or not Path(root).is_absolute():
        raise ValueError("HUMAN_CONTEXT_BUNDLE_ROOT must be configured")
    info = activity.info()
    engine = get_engine()
    prior = _read_prior(engine, request)
    if prior is not None:
        return prior
    stopped = threading.Event()
    native_context = contextvars.copy_context()
    def heartbeat_loop():
        """Heartbeat during model loading/inference with the SDK Activity context retained."""
        while not stopped.wait(15):
            activity.heartbeat("native Neural chunk planning")
    thread = threading.Thread(target=lambda: native_context.run(heartbeat_loop), daemon=True)
    thread.start()
    def beat(detail):
        """Check native cancellation between bounded spool/window operations and heartbeat refs only."""
        if activity.is_cancelled():
            raise RuntimeError("native chunk planning cancelled")
        activity.heartbeat(detail)
    try:
        result = plan_native_chunks(Path(root), request, beat=beat)
    finally:
        stopped.set()
        thread.join()
    return persist_chunk_receipt(engine, request, result, info)
