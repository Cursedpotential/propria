"""Thin starter for the two conversation-chunk workflows. It runs nothing itself; Temporal does, and keeps the record.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

    python -m server.context_chunks.start rechunk --dry-run            # counts: threads, messages, chunks, embed calls
    python -m server.context_chunks.start rechunk --dry-run --exact    # chunk every thread (CPU), no embed, no writes
    python -m server.context_chunks.start rechunk                      # chunk, embed and publish every committed thread
    python -m server.context_chunks.start remove --dry-run             # verify, then count what would be removed
    python -m server.context_chunks.start remove                       # verify, then remove the per-message objects

It starts the Go workflow (queue ``TEMPORAL_PROFFER_TASK_QUEUE``, default ``proffer-v1``) and prints the workflow id,
the run id and, unless ``--no-wait``, the result. Everything after that is in the Temporal UI under that id.
Env: TEMPORAL_ADDRESS, TEMPORAL_NAMESPACE (as server/temporal/worker.py).
"""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import sys
import time
import uuid
from typing import Any

BACKFILL_WORKFLOW = "proffer_conversation_chunks_backfill_workflow"
REMOVAL_WORKFLOW = "proffer_conversation_chunks_removal_workflow"


def build_input(args: argparse.Namespace) -> tuple[str, str, dict[str, Any]]:
    """(workflow name, workflow id, input) for the parsed command line. The input uses the Go struct's JSON names."""
    request_id = args.request_id or f"{args.command}-{time.strftime('%Y%m%dT%H%M%S')}-{uuid.uuid4().hex[:6]}"
    if args.command == "rechunk":
        body: dict[str, Any] = {
            "request_id": request_id,
            "dry_run": args.dry_run,
            "exact": args.exact,
            "chunker": args.chunker,
            "overlap": args.overlap,
            "skip_covered": args.skip_covered,
            "thread_limit": args.limit,
            "no_calls": args.no_calls,
        }
        return BACKFILL_WORKFLOW, f"conversation-chunks-backfill-{request_id}", body
    body = {
        "request_id": request_id,
        "dry_run": args.dry_run,
        "only_covered": args.only_covered,
        "old_collection": args.old_collection,
    }
    return REMOVAL_WORKFLOW, f"conversation-chunks-removal-{request_id}", body


def parser() -> argparse.ArgumentParser:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="command", required=True)
    for name in ("rechunk", "remove"):
        p = sub.add_parser(name)
        p.add_argument("--dry-run", action="store_true")
        p.add_argument("--request-id", default="")
        p.add_argument("--no-wait", action="store_true")
    rc = sub.choices["rechunk"]
    rc.add_argument("--exact", action="store_true", help="dry run: chunk every thread instead of estimating")
    rc.add_argument("--chunker", default="")
    rc.add_argument("--overlap", type=int, default=0)
    rc.add_argument(
        "--skip-covered", action="store_true", help="leave a thread alone when some chunk covers all its messages"
    )
    rc.add_argument("--limit", type=int, default=0, help="stop after this many threads")
    rc.add_argument("--no-calls", action="store_true", help="skip the call-log files")
    rm = sub.choices["remove"]
    rm.add_argument("--only-covered", action="store_true", help="remove just the covered objects when some are not")
    rm.add_argument("--old-collection", default="")
    return ap


async def start(args: argparse.Namespace) -> int:
    from temporalio.client import Client

    name, workflow_id, body = build_input(args)
    client = await Client.connect(
        os.environ.get("TEMPORAL_ADDRESS", "temporal-server:7233"),
        namespace=os.environ.get("TEMPORAL_NAMESPACE", "default"),
    )
    handle = await client.start_workflow(
        name, body, id=workflow_id, task_queue=os.environ.get("TEMPORAL_PROFFER_TASK_QUEUE", "proffer-v1")
    )
    print(f"started {name}: workflow id {handle.id}, run id {handle.result_run_id}", flush=True)
    if args.no_wait:
        return 0
    result = await handle.result()
    print(json.dumps(result, indent=2, default=str))
    return 0


def main(argv: list[str] | None = None) -> int:
    return asyncio.run(start(parser().parse_args(argv)))


if __name__ == "__main__":
    sys.exit(main())
