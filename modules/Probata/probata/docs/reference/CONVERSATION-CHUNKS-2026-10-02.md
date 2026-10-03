# Conversation chunks in Weaviate (ProfferChunks20261002)

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-02. Owner rules of 2026-10-02, built; deployed state is in `docs/URGENT-TODO.md`._

## The rule

- **Postgres holds every single message and every call.** It is the record of truth.
- **Weaviate holds only conversation chunks**, never one entry per message. Each chunk lists the ids of the messages it covers.
- **Calls:** one Weaviate entry per call-log **file** (one source version), listing its call ids. No per-call object.
- **Everything goes to Weaviate first** (owner ruling 2026-09-18, restated 2026-10-02): a run's chunks are searchable before the preview, before the owner's decision and before the Postgres commit.
- The Case Bible chunks belong to the Super Index (CocoIndex); this module is what it shares, see "Public API".

## Where a run publishes

`ProfferWorkflow`, behind marker `proffer-conversation-chunks-v1`, in this order:
`resolve_context_participants` -> `match_message_occurrences` -> `publish_context_search` (skips messages and calls; AI chats and documents unchanged) -> `propose_first_party_context` -> **chunk, publish chunks, publish the call-log file** -> preview -> owner's decision -> confirm and commit.

The chunks are cut from the run's **normalized generation** (`context.normalized_record_identity`), not from `working.*`. Every id a chunk carries is a normalized record id; the first-party import copies it unchanged into `working.message`, so after the commit the same ids are the Postgres message ids and nothing is rewritten. Chunks carry both `message_ids` and `normalized_record_ids` (equal), and `normalized_generation_id`.

- **Grouping** is the first-party import's: (corpus, conversation key). The key is `smsthreads.ConversationKey`, ported to `generation.py`; the corpus is `first_party` when the owner is a stated party under the run's participant resolution, else `acquired_third_party` (`disclosure`, ported). The Go test `TestConversationKeyVectorsSharedWithThePythonPort` and `tests/test_context_chunks.py` assert the same vectors; change one, change both.
- A message the match-up found already held by an earlier source is left out (the earlier source's chunks hold it).
- A chunk's `thread_id` is `gen-<generation id>/<corpus>/<key>`. A later backup of the same conversation is a new generation with its own chunks; the match-up keeps them from duplicating messages.
- **A rejected run keeps its chunks**, exactly as it keeps its per-message search objects today: nothing marks or removes them. They carry `ingest_run_id` (the Proffer run), `normalized_generation_id` and `source_version_ids`, so a reader can join them to the rejected preview decision. A mark or removal for both kinds of object would be one follow-up.
- A failure stops the run before the preview with the Activity's reason. Every write is idempotent.

## Units (one job each)

| Unit | Where | Does |
|---|---|---|
| `chunk_context_threads_activity` | `server/temporal/chunk_activities.py`, queue `evidence-pipeline` | a generation's conversations (or named committed threads) -> plans (message-index spans plus a digest, no text) |
| `publish_context_chunks_activity` | same | one thread: embed (or copy, see Reuse), write, delete that thread's chunks of any other chunking |
| `publish_call_log_files_activity` | same | one entry per call-log file |
| `list_context_threads_activity`, `estimate_context_chunks_activity` | `chunk_backfill_activities.py` | the re-chunk's thread list and its dry-run counts |
| `verify_chunk_coverage_activity`, `remove_per_message_objects_activity` | same | the removal's verification and deletion |
| `ProfferWorkflow` branch | `modules/engine/proffer/context_chunks.go` | schedules the first three before the preview |
| `proffer_conversation_chunks_backfill_workflow`, `proffer_conversation_chunks_removal_workflow` | `proffer/context_chunks_backfill.go` | re-chunk what `working.*` holds, and remove the old per-message objects; dry-run is an input, one Activity per thread, the result is the receipt, progress is a query |

`python -m server.context_chunks.start rechunk|remove [--dry-run]` only starts those workflows and prints the ids.
The Python Activities are not stages of the stage graph (the Go worker's 26 stages); they run on the Python queue like the extraction flow's projection step.

## Chunker

Chonkie 1.7.0 Neural, `mirth/chonky_distilbert_base_uncased_1`, `min_characters_per_chunk=1` (the settings of `scripts/jev_eval/chunk_all.py`). Three additions, each measured:

1. **Windowed.** On the pinned stack the Neural chunker classifies only the first ~512 tokens it is given: a 1000-message thread came back as 3 chunks, every split in the first window. A thread is fed to it in half-overlapping windows of at most 450 tokens, each split taken from the window with context on both sides.
2. **Overlap.** Every chunk is extended forward by 2 messages: each chunk shares at least 2 messages with the next.
3. **Size cap.** A chunk over 7500 characters is cut into even parts by message, so the whole chunk fits the embedder's 8000-character input.

Other Chonkie chunkers stay selectable (`token_1000`, `fast_1000`, `sentence_1000`, `recursive_1000`, `neural_modernbert`; `chunker.register` adds more).

## Public API (`server/context_chunks`, import these; everything else is Proffer wiring)

```python
from server.context_chunks import (
    render_line, chunk_spans, chunker_version,
    chunk_text, chunk_content_hash, chunk_key, content_chunk_id, nim_input,
)
render_line(at, sender, body)            # "[YYYY-MM-DD HH:MM] <sender>: <one-line body>", UTC; "(attachment)" when empty
chunk_spans(lines, name, overlap)        # [(first, last)], inclusive message indexes, overlap included
chunker_version(name, overlap)           # e.g. "neural_distilbert|mirth/chonky_distilbert_base_uncased_1|chonkie-1.7.0|window=450|maxchars=7500|overlap=2|text=line-v1"
chunk_text(lines)                        # "\n".join(lines), the member messages in thread order, overlap included
chunk_content_hash(lines)                # sha256(chunk_text(lines).encode("utf-8")).hexdigest()
chunk_key(version, content_hash)         # version + "|" + content_hash   (THE shared chunk identity)
content_chunk_id(key)                    # uuid5(b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, "content-chunk-v1|" + key)
```

**The sender in `render_line` is the source-stated string** (a number, a name, `self`; for SMS XML a received message uses the address as in the file, a sent one `device -> <address>`). Resolved names are never in the embedded text; they ride as `participant_names` and `participant_entity_ids`. Both systems must render this way or their chunks do not match.

The shared test vector is `tests/test_context_chunks_reuse.py::test_the_shared_chunk_key_vector`: three lines, hash `473919a1...7bb0`, id `96961b7f-e21d-5269-8a85-c79dc69d526a`. The other side asserts the same.

Properties on every chunk: `text`, `content_hash`, `chunker`, `chunker_version`, `overlap`, the `text_nim` named vector, plus the links (`message_ids`, `normalized_record_ids`, `thread_id`, `source_version_ids`, ...). Proffer's own object id stays `uuid5(ns, "proffer-chunk-v1|<thread id>|<first message id>|<last message id>|<chunker version>")`.

## Reuse

When `CONTEXT_CHUNKS_REUSE_COLLECTION` names a collection (`CaseBibleChunks20261002`), Proffer looks up its chunks by `content_hash` and `chunker_version` before embedding. A match: the object's vectors are copied (every named vector this collection has; a chunk missing one is not reused), and Proffer adds its own id, the Postgres message ids, resolved names and entity ids, plus `reused_from_object_id` and `reused_from_collection`. No match, or no such collection: Proffer chunks and embeds itself. The match needs the same chunker version, the same boundaries, the same rendered text; the sender string is where two systems most easily differ.

## Operations (the parent session runs these; nothing here starts by itself)

- Re-chunk what is committed: `python -m server.context_chunks.start rechunk --dry-run` (counts), then without `--dry-run`. `--skip-covered` leaves a thread alone when chunks already cover all its messages.
- Remove the old per-message objects after the chunks are verified: `start remove --dry-run`, then `start remove`. It verifies first (by message id), refuses when anything is uncovered unless `--only-covered`.
