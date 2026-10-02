"""Re-chunk every committed thread already in Postgres into ProfferChunks20261002.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

One entry point, two modes. It is a long job: run it detached on the VPS (the temporal-worker image carries the model
and the NIM key), never as a child of an agent's shell.

    python -m server.context_chunks.rechunk --dry-run            # counts only: threads, messages, estimated chunks,
                                                                 # embed calls. Chunks a small sample; no embed, no writes.
    python -m server.context_chunks.rechunk --dry-run --exact    # chunks every thread (CPU, no embed, no writes)
    python -m server.context_chunks.rechunk --run                # chunk + embed + publish every thread, then call files
    python -m server.context_chunks.rechunk --run --thread first_party:<uuid> --thread acquired_third_party:<uuid>
    python -m server.context_chunks.rechunk --run --limit 5 --no-calls

Idempotent and resumable: chunk ids are deterministic, a thread whose current chunking is already in the collection is
skipped (``--force`` redoes it), and a thread that grew is replaced whole. Message text is never printed.
"""

from __future__ import annotations

import argparse
import math
import sys
import time
from typing import Any

from server.context_chunks.chunker import chunk_spans, chunker_version
from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP, EMBED_BATCH
from server.context_chunks.model import CORPUS_FIRST_PARTY, CORPUS_THIRD_PARTY, ThreadRef
from server.context_chunks.service import (
    plan_thread,
    publish_call_files,
    publish_thread,
    thread_lines,
)
from server.context_chunks.source import PgSource


def _parse_ref(value: str) -> ThreadRef:
    corpus, _, thread_id = value.partition(":")
    if corpus not in (CORPUS_FIRST_PARTY, CORPUS_THIRD_PARTY) or not thread_id:
        raise argparse.ArgumentTypeError(
            f"--thread wants first_party:<uuid> or acquired_third_party:<uuid>, got {value!r}"
        )
    return ThreadRef(corpus, thread_id)


def _sample(counts: dict[ThreadRef, int], size: int, cap: int) -> list[ThreadRef]:
    """Threads spread across the size distribution, none above ``cap`` messages (a sample, not the whole job)."""
    eligible = sorted((r for r, n in counts.items() if 2 <= n <= cap), key=lambda r: counts[r])
    if len(eligible) <= size:
        return eligible
    step = (len(eligible) - 1) / (size - 1)
    return [eligible[round(i * step)] for i in range(size)]


def dry_run(source: PgSource, chunker: str, overlap: int, sample: int, exact: bool, sample_cap: int) -> dict:
    counts = source.thread_message_counts()
    first = {r: n for r, n in counts.items() if r.corpus == CORPUS_FIRST_PARTY}
    third = {r: n for r, n in counts.items() if r.corpus == CORPUS_THIRD_PARTY}
    report: dict[str, Any] = {
        "chunker_version": chunker_version(chunker, overlap),
        "threads": {"first_party": len(first), "acquired_third_party": len(third), "total": len(counts)},
        "messages": {
            "first_party": sum(first.values()),
            "acquired_third_party": sum(third.values()),
            "total": sum(counts.values()),
        },
    }
    if exact:
        chunks_by_thread, started = {}, time.time()
        for i, ref in enumerate(sorted(counts, key=lambda r: (r.corpus, r.thread_id)), 1):
            plan = plan_thread(source.load_thread(ref), chunker, overlap)
            chunks_by_thread[ref] = len(plan.spans)
            print(
                f"  chunked {i}/{len(counts)} {ref.corpus} {ref.thread_id}: {counts[ref]} messages -> "
                f"{len(plan.spans)} chunks ({time.time() - started:.0f}s)",
                flush=True,
            )
        report["basis"] = "exact: every thread chunked"
    else:
        picked = _sample(counts, sample, sample_cap)
        sampled_messages = sampled_chunks = 0
        started = time.time()
        for ref in picked:
            thread = source.load_thread(ref)
            n_chunks = len(chunk_spans(thread_lines(thread), chunker, overlap))
            sampled_messages += len(thread.messages)
            sampled_chunks += n_chunks
            print(
                f"  sampled {ref.corpus} {ref.thread_id}: {len(thread.messages)} messages -> {n_chunks} chunks "
                f"({time.time() - started:.0f}s)",
                flush=True,
            )
        if not sampled_chunks:
            raise SystemExit("no thread to sample")
        ratio = sampled_messages / sampled_chunks
        chunks_by_thread = {r: max(1, math.ceil(n / ratio)) for r, n in counts.items()}
        report["basis"] = (
            f"estimate: {len(picked)} sampled threads, {ratio:.2f} messages per chunk overall "
            f"(widened chunks, overlap {overlap})"
        )
    chunks = sum(chunks_by_thread.values())
    report["chunks"] = chunks
    report["embed_calls"] = sum(math.ceil(n / EMBED_BATCH) for n in chunks_by_thread.values())
    versions = source.call_source_versions()
    report["call_log_files"] = len(versions)
    report["call_file_embed_calls"] = len(versions)  # one whole-file entry per request
    report["embed_batch"] = EMBED_BATCH
    return report


def run(
    source: PgSource,
    store,
    embedder,
    chunker: str,
    overlap: int,
    refs: list[ThreadRef],
    *,
    force: bool,
    calls: bool,
    limit: int,
) -> dict:
    todo = refs or sorted(source.all_threads(), key=lambda r: (r.corpus, r.thread_id))
    if limit:
        todo = todo[:limit]
    totals: dict[str, Any] = {"threads": 0, "skipped_existing": 0, "chunks": 0, "stale_deleted": 0, "failed": 0}
    started = time.time()
    for i, ref in enumerate(todo, 1):
        try:
            plan = plan_thread(source.load_thread(ref), chunker, overlap)
            result = publish_thread(source, store, embedder, plan, resume=not force)
        except Exception as error:  # noqa: BLE001  one thread's failure is counted and printed, then the run goes on
            totals["failed"] += 1
            print(f"FAILED {ref.corpus} {ref.thread_id}: {type(error).__name__}: {str(error)[:200]}", flush=True)
            continue
        totals["threads"] += 1
        totals["skipped_existing"] += int(result.skipped_existing)
        totals["chunks"] += result.chunks_written
        totals["stale_deleted"] += result.stale_deleted
        print(
            f"{i}/{len(todo)} {ref.corpus} {ref.thread_id}: {plan.message_count} messages, {len(plan.spans)} chunks, "
            f"written {result.chunks_written}, stale deleted {result.stale_deleted}"
            f"{', already current' if result.skipped_existing else ''} ({time.time() - started:.0f}s)",
            flush=True,
        )
    if calls:
        totals["call_files"] = publish_call_files(source, store, embedder)
        print(f"call-log files: {totals['call_files']}", flush=True)
    totals["embed_requests"] = embedder.calls
    totals["embedded_texts"] = embedder.texts
    return totals


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--dry-run", action="store_true")
    mode.add_argument("--run", action="store_true")
    ap.add_argument("--chunker", default=DEFAULT_CHUNKER)
    ap.add_argument("--overlap", type=int, default=DEFAULT_OVERLAP)
    ap.add_argument("--exact", action="store_true", help="dry run: chunk every thread instead of estimating")
    ap.add_argument("--sample", type=int, default=6, help="dry run: threads to chunk for the estimate")
    ap.add_argument("--sample-cap", type=int, default=5000, help="dry run: largest thread (messages) to sample")
    ap.add_argument("--thread", action="append", type=_parse_ref, default=[], help="only these threads")
    ap.add_argument("--limit", type=int, default=0, help="run: stop after this many threads")
    ap.add_argument("--no-calls", action="store_true", help="run: skip the call-log files")
    ap.add_argument("--force", action="store_true", help="run: redo threads whose current chunking is already stored")
    args = ap.parse_args(argv)

    from server.context_chunks.db import read_only_connection

    with read_only_connection() as conn:
        source = PgSource(conn)
        if args.dry_run:
            report = dry_run(source, args.chunker, args.overlap, args.sample, args.exact, args.sample_cap)
            print("DRY RUN (nothing embedded, nothing written)")
            for key, value in report.items():
                print(f"  {key}: {value}")
            return 0
        from server.context_chunks.config import load_config
        from server.context_chunks.embed import NimEmbedder
        from server.context_chunks.store import ChunkStore

        config = load_config()
        totals = run(
            source,
            ChunkStore(config.weaviate_url, config.collection),
            NimEmbedder(config),
            args.chunker,
            args.overlap,
            args.thread,
            force=args.force,
            calls=not args.no_calls,
            limit=args.limit,
        )
        print("RUN COMPLETE")
        for key, value in totals.items():
            print(f"  {key}: {value}")
        return 1 if totals["failed"] else 0


if __name__ == "__main__":
    sys.exit(main())
