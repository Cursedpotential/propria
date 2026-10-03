"""Re-chunk what is already in Postgres: the list and the dry-run, as library functions behind Temporal Activities.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner order 2026-10-02: everything runs as Temporal Activities and is traceable. The re-chunk of the committed threads is
the Go workflow ``proffer_conversation_chunks_backfill_workflow`` (modules/engine/proffer/context_chunks_backfill.go):
one Activity lists the threads, then each thread is one chunk Activity and one publish Activity, then the call-log files;
dry-run is a workflow input. ``python -m server.context_chunks.start rechunk [--dry-run]`` starts it. This module holds
what the list and estimate Activities call.

It reads working.* (PgSource): what is already committed. A run that publishes its chunks before the commit
(proffer.ProfferWorkflow) has its own, generation-scoped chunks; ``skip_covered`` leaves a thread alone when its
messages are all in some chunk already.
"""

from __future__ import annotations

import math
from typing import Any

from server.context_chunks.chunker import chunk_spans, chunker_version
from server.context_chunks.config import EMBED_BATCH
from server.context_chunks.model import CORPUS_FIRST_PARTY, CORPUS_THIRD_PARTY, ThreadRef
from server.context_chunks.service import thread_lines
from server.context_chunks.source import Source


def _sample(counts: dict[ThreadRef, int], size: int, cap: int) -> list[ThreadRef]:
    """Threads spread across the size distribution, none above ``cap`` messages (a sample, not the whole job)."""
    eligible = sorted((r for r, n in counts.items() if 2 <= n <= cap), key=lambda r: counts[r])
    if len(eligible) <= size:
        return eligible
    step = (len(eligible) - 1) / (size - 1)
    return [eligible[round(i * step)] for i in range(size)]


def list_threads(
    source: Source, *, covered: set[str] | None = None, limit: int = 0
) -> tuple[list[dict[str, Any]], list[str]]:
    """(the threads to re-chunk, the call-log source versions). With ``covered`` (message ids already in some chunk) a
    thread whose messages are all covered is left out."""
    counts = source.thread_message_counts()
    threads = []
    for ref in sorted(counts, key=lambda r: (r.corpus, r.thread_id)):
        if covered is not None and all(m in covered for m in source.thread_message_ids(ref)):
            continue
        threads.append({"corpus": ref.corpus, "thread_id": ref.thread_id, "messages": counts[ref]})
    if limit:
        threads = threads[:limit]
    return threads, source.call_source_versions()


def dry_run(
    source: Source, chunker: str, overlap: int, sample: int, exact: bool, sample_cap: int, beat: Any = None
) -> dict[str, Any]:
    """Counts only: threads, messages, estimated (or exact) chunks, embed calls. Nothing is embedded or written."""
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
        chunks_by_thread = {}
        for i, ref in enumerate(sorted(counts, key=lambda r: (r.corpus, r.thread_id)), 1):
            if beat:
                beat(f"exact {i}/{len(counts)}")
            chunks_by_thread[ref] = len(chunk_spans(thread_lines(source.load_thread(ref)), chunker, overlap, beat=beat))
        report["basis"] = "exact: every thread chunked"
    else:
        picked = _sample(counts, sample, sample_cap)
        sampled_messages = sampled_chunks = 0
        for ref in picked:
            if beat:
                beat(f"sample {ref.thread_id}")
            thread = source.load_thread(ref)
            sampled_messages += len(thread.messages)
            sampled_chunks += len(chunk_spans(thread_lines(thread), chunker, overlap, beat=beat))
        if not sampled_chunks:
            raise ValueError("no thread to sample")
        ratio = sampled_messages / sampled_chunks
        chunks_by_thread = {r: max(1, math.ceil(n / ratio)) for r, n in counts.items()}
        report["basis"] = (
            f"estimate: {len(picked)} sampled threads, {ratio:.2f} messages per chunk overall "
            f"(widened chunks, overlap {overlap})"
        )
    report["chunks"] = sum(chunks_by_thread.values())
    report["embed_calls"] = sum(math.ceil(n / EMBED_BATCH) for n in chunks_by_thread.values())
    versions = source.call_source_versions()
    report["call_log_files"] = len(versions)
    report["call_file_embed_calls"] = len(versions)  # one whole-file entry per request
    report["embed_batch"] = EMBED_BATCH
    return report
