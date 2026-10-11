"""Expose one reference-only analytical step to the existing evidence-pipeline worker.

Inputs: exact Go-prepared step reference. Outputs: exact checkpoint reference.
Effects: B2 derivative I/O and one NIM call; Go owns sequencing and final validation.
"""
from __future__ import annotations

import asyncio
from dataclasses import dataclass
from typing import Any
from temporalio import activity
from temporalio.exceptions import ApplicationError


@dataclass
class StepRequest:
    """Decode one Go step reference; input pinned artifact, output dataclass, no effects; source bytes stay outside history."""
    artifact: dict[str, Any]


@activity.defn(name="reason_approved_walk_step_activity")
async def reason_approved_walk_step_activity(request: StepRequest) -> dict[str, Any]:
    """Reason over one finite horizon and retain a versioned cited checkpoint.

Inputs: exact step ArtifactRef. Outputs: exact checkpoint ArtifactRef only.
Effects: bounded B2 reads/writes and one NIM call; choose from Go approvedwalkflow.
"""
    from server.analysis.approved_walk import Artifacts, NIMModel, reason
    async def pulse() -> None:
        """Pulse through artifact/model I/O; input Activity context, output none, effects heartbeats; no source payloads."""
        while True:
            await asyncio.sleep(15)
            activity.heartbeat("reasoning")

    activity.heartbeat("reasoning")
    heartbeat = asyncio.create_task(pulse())
    model = None
    try:
        artifacts = await asyncio.to_thread(Artifacts)
        step = await asyncio.to_thread(artifacts.read, request.artifact)
        # Retry after successful persistence returns the retained checkpoint without another call.
        # Temporal heartbeat details retain only this versioned ref, never source/model text.
        if activity.info().heartbeat_details and isinstance(activity.info().heartbeat_details[0], dict):
            ref = activity.info().heartbeat_details[0]
            checkpoint = await asyncio.to_thread(artifacts.read, ref)
            from server.analysis.approved_walk import canonical
            import hashlib
            if checkpoint.get("input_digest") != hashlib.sha256(canonical(step)).hexdigest():
                raise ValueError("retry checkpoint identity mismatch")
            return {"artifact": ref}
        model = NIMModel()
        checkpoint = await reason(step, model)
        checkpoint["input_artifact"] = request.artifact
        ref = await asyncio.to_thread(artifacts.put, request.artifact, checkpoint)
        activity.heartbeat(ref)
        return {"artifact": ref}
    except (ValueError, KeyError, TypeError) as error:
        raise ApplicationError("approved walk step validation failed", type="ApprovedWalkContract", non_retryable=True) from error
    finally:
        heartbeat.cancel()
        try:
            await heartbeat
        except asyncio.CancelledError:
            pass
        if model is not None:
            await model.close()


APPROVED_WALK_ACTIVITIES = [reason_approved_walk_step_activity]
