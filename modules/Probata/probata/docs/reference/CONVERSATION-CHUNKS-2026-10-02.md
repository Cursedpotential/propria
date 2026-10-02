# Conversation chunks in Weaviate (ProfferChunks20261002)

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-02. Owner rules of 2026-10-02 (15:40-15:46 EDT), built, not yet deployed._

## The rule

- **Postgres holds every single message and every call.** It is the record of truth.
- **Weaviate holds only conversation chunks**, never one entry per message. Each chunk lists the Postgres message ids it covers.
- **Calls:** one Weaviate entry per call-log **file** (one source version), listing its `working.call_log` ids. No per-call object.
- The Case Bible collection `MsgEvents20260918` is a separate search engine and is never touched by Proffer. Its existing 366,912 per-message objects stay; only NEW Case Bible loads are chunked (into `CaseBibleChunks20261002`).

## Units (one job each; the engine orchestrates)

| Unit | Where | Does |
|---|---|---|
| `chunk_context_threads_activity` | `server/temporal/chunk_activities.py`, queue `evidence-pipeline` | reads the threads a source version touched, runs the chunker, returns a plan (message-index spans + digest). No embed, no writes. |
| `publish_context_chunks_activity` | same | one thread: embeds its chunks (NIM `nvidia/nemotron-3-embed-1b`, 2048-d, batched, input guards), writes them, deletes that thread's chunks of any other chunking. |
| `publish_call_log_files_activity` | same | one entry per call-log file of the source version. |
| `ProfferWorkflow` branch | `modules/engine/proffer/context_chunks.go`, marker `proffer-conversation-chunks-v1` | after `commit_first_party_context_threads` and `commit_call_log`, before `seal_generation`. A failure stops the run before the seal. |
| `publish_context_search_activity` | `modules/engine/activities/publish_context_search.go` | on a run with the marker, skips messages and calls (`skip_record_kinds=message,call`). AI chats and documents are unchanged. |

The Python Activities are not stages of the stage graph (the graph is the Go worker's 26 stages); they run on the Python queue like the extraction flow's `build_timeline_generation_activity`.

## Chunker

Chonkie 1.7.0 Neural, `mirth/chonky_distilbert_base_uncased_1`, `min_characters_per_chunk=1` (the settings of `scripts/jev_eval/chunk_all.py`). One line per message, `[YYYY-MM-DD HH:MM] Sender: body` (UTC). Three additions, each measured:

1. **Windowed.** On the pinned stack the Neural chunker classifies only the first ~512 tokens it is given: a 1000-message thread came back as 3 chunks, every split in the first window. The head-to-head fed it stretches of at most 89 messages, where that did not show. A thread is fed to it in half-overlapping windows of at most 450 tokens, and each split is taken from the window that has context on both sides.
2. **Overlap.** Every chunk is extended forward by 2 messages, so each chunk shares at least 2 messages with the next.
3. **Size cap.** A chunk longer than 7500 characters is cut into even parts by message, so the whole chunk fits the embedder's 8000-character input.

Other Chonkie chunkers stay selectable (`token_1000`, `fast_1000`, `sentence_1000`, `recursive_1000`, `neural_modernbert`; `chunker.register` adds more).

## Identity

- Chunk object id: `uuid5(b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, "proffer-chunk-v1|<thread_id>|<first message id>|<last message id>|<chunker version>")`.
- The chunker version names the exact construction: chunker, model, library version, window, size cap, overlap, text format.
- A thread that grew is re-chunked whole; its new chunks are written first, then every chunk of that thread with another `thread_digest` is deleted.
- Property `source_version_ids[0]` is the source version of the chunk's first message; the Workbench opens the thread from it, at `first_message_id`.

## Operations (the parent session runs these; nothing here runs by itself)

- Re-chunk what is already in Postgres: `python -m server.context_chunks.rechunk --dry-run` (counts), then `--run`, detached on the VPS (temporal-worker image).
- Remove the old per-message objects, after the chunks are verified: `python -m server.context_chunks.remove_per_message --verify`, `--dry-run`, `--execute`.
- Case Bible: `chunk_publish.py` in `casebible/tools/comm_timeline_mvp`; `publish_bundle.py MODE=probe` prints the counts.
