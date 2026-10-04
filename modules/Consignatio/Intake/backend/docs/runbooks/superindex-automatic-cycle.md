# Coco Super Index: the automatic cycle

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-02. Owner order 2026-10-02 19:58 EDT: "finish the Coco Super Index so it runs the way it was designed, automatically and not by hand", and 20:02: "make sure all these things get run as Temporal activities and are traceable."_

## What runs, and who starts it

```
Temporal Schedule  superindex-cycle   every 15 min, overlap SKIP      (cmd/superindex-schedule ensure)
  -> Go workflow   SuperIndexCycleWorkflow                            (Probata modules/engine/superindex, registered on the proffer worker)
       -> Python Activities on queue "superindex"                     (casebible_index/temporal_worker.py, compose service superindex-worker)
```

Nothing indexes outside that path. There is no daemon loop. Every tick is a workflow run in Temporal history; every
stage is an Activity with heartbeats, retries and a counted result; every stage also leaves an immutable receipt under
`<output>/cycles/<cycle_id>/` (the history carries counts, the files carry detail). `GET /index/cycles` on the API
returns the committed watermark and the newest cycles.

| Stage (Activity) | One job | Reads | Writes |
|---|---|---|---|
| `superindex_discover_activity` | catalog watermark per (provider, bucket); classify every object; ends the cycle when nothing changed | `raw_duck.bucket_objects_current` | `inventory/represented/<cycle>.parquet` (media and unsupported objects, with locator, size, sha1) |
| `superindex_extract_activity` | CocoIndex incremental pass: extract + chunk + document rows (a child process; killed on cancel or past `INTAKE_EXTRACT_RSS_LIMIT_MB`) | B2/R2 objects that are new or changed (catalog size + sha1 memo) | `datasets/documents`, `chunks`, `members`, `locators`; the active snapshot |
| `superindex_summarize_activity` | the separate summary pass over documents the index picks | Parquet only | `datasets/summaries` |
| `superindex_embed_activity` | one vector slot for every distinct chunk text without one (one Activity per slot, in parallel) | Parquet | `datasets/vectors/<slot>` |
| `superindex_publish_activity` | chunks + vectors -> Weaviate `CaseBibleChunks20261002`; patch the locator of moved files | Parquet | Weaviate; `datasets/published` |
| `superindex_graph_activity` | project the active snapshot into the `surreal-intake` file graph; skips unless the snapshot changed and 24 h passed (the contract is snapshot-scoped, so a projection per slice would rewrite the corpus each time) | Parquet | surreal-intake |
| `superindex_commit_activity` | advance the watermark, last, only after every stage succeeded | | `cycles/<id>/commit.json` |

Extract and chunk are ONE Activity on purpose: extracted text streams into the chunker and is never materialized.
That is what keeps a 1.3 GB export bounded. Splitting them would mean writing the whole text to disk between them.

### How "new or changed" is detected

The catalog is the only source. `bucket_objects_current` is the newest listing of each bucket; its watermark is
(listing time, object count, bytes). Unchanged watermark: one cheap Activity, outcome `no_change`. Changed: the full
key set goes to CocoIndex, whose memo state (catalog size + sha1) skips every unchanged object with zero bucket reads.
A file that only MOVED has the same sha1: it is not streamed again (`already_indexed`), a locator row records its new
key, and the publish stage patches `vault_key` and `catalog_path` on its Weaviate objects (no re-extract, no re-embed).

### Bounded memory, by construction

- An object is spooled to disk and read in 1 MiB windows. A message export is parsed into a SQLite spool on disk and chunked
  one 20,000-message segment of one thread at a time. Local measurement: a 250 MB SMS XML (1.22 M records, 152,581 chunks,
  one thread of ~730 k messages) peaked at 215 MB working set.
- CocoIndex no longer holds one declared Weaviate target state per chunk (the 3.16 GiB of the 2026-09-22 receipt). Chunks go
  to Parquet; the publish stage posts them in batches of 100 and keeps a ledger.
- The worker container is capped at 4 GiB (`mem_limit`), the extract child is killed cleanly at 3,000 MB, the first full
  run is time-sliced (`INTAKE_EXTRACT_TIME_BUDGET_S`, workflow default 4 h) so embeddings and Weaviate fill as it goes.
- `INTAKE_WEAVIATE_INDEX_ENABLED` still exists for the API's legacy `IntakeCorpus` search; the worker forces it off for the
  extract child.

## Routing: where each object goes

| Kind | What the index does | Setting |
|---|---|---|
| Document (text, PDF, DOCX, RTF, EML, HTML, ...) | extract, chunk (recursive splitter), content-addressed chunks | |
| Message export (smsbackuprestore XML: `<sms>`, `<mms>`, `<call>`) | threads by `address`, conversation chunks with the shared module (Chonkie Neural distilbert, windowed, >= 2 messages overlap) | `INTAKE_CONVERSATION_MODE=chunk` (default) or `route` (document row only) |
| AI chat export (`conversations.json` that sniffs as ChatGPT or Claude) | a document row with status `routed_ai_chat` and the locator; NEVER chunked here (Proffer owns it) | `INTAKE_AI_CHAT_MODE=route` (default) or `index` |
| ZIP | members listed over ranged reads; each text member is its own document (`member_path`), a nested ZIP is opened from a spool (depth 3), everything else is recorded in one members-inventory Parquet per archive | `INTAKE_INDEX_ARCHIVE_MEMBERS`, `INTAKE_ARCHIVE_MEMBER_LIMIT` |
| Image, audio, video, unsupported | represented in the discovery inventory (locator, size, sha1, status), not extracted here | |
| tar, tgz, 7z, rar | recorded as a container; members not listed (no central directory to range-read) | |

Senders are source-stated (see `message_sources.py`): received messages use the file's `address`, sent messages
`device -> <address>`, `contact_name` is never used.

## Chunk identity (shared with Proffer so it can COPY chunks)

```
chunk_text   = "\n".join(render_line(at, sender, body) for each member message, in order, overlap messages included)
content_hash = sha256(chunk_text, UTF-8)
chunk_key    = chunker_version + "|" + content_hash
object id    = uuid5(b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, "content-chunk-v1|" + chunk_key)
```

Frozen test vector (`tests/test_chunk_identity.py`): text `[2026-01-02 03:04] +18105550101: hello` + newline +
`[2026-01-02 03:05] device -> +18105550101: hi`, chunker version
`neural_distilbert|mirth/chonky_distilbert_base_uncased_1|chonkie-1.7.0|window=450|maxchars=7500|overlap=2|text=line-v1`
gives content_hash `ad70b309b1c71dd78f7417a1a739663672e919e843f3264d60eeafa5895c523b` and id
`d877f60e-322f-5339-9d4b-097e4e05d67c`. Every object carries `content_hash`, `chunker_version`, `text` and the vector, so a
reader can find it by id or by filter and copy the vector without re-embedding. A match needs identical rendered text:
the side that renders resolved names instead of source-stated senders will not match.

## Vectors: named slots

`INTAKE_VECTOR_SLOTS` (default `text_nim`): one Weaviate named vector per slot, one embed Activity per slot.
`text_nim` = NVIDIA NIM `nvidia/nemotron-3-embed-1b`, 2048-d, batched 32 per request, with the input guards (`data:image/`
rewritten to `data: image/`, blanks get a placeholder, 8,000-character cut). `legal` is a pluggable slot: set
`INTAKE_VECTOR_LEGAL_MODEL` (and `_DIMENSIONS`, `_BASE_URL`, `_KEY_SECRET`, `_STYLE`) when the legal embedder is chosen;
the Weaviate collection must already have the `legal` named vector (the store refuses to guess). Pattern follows
CocoIndex's named-vector support (https://cocoindex.io/docs-v1/connectors/qdrant, https://cocoindex.io/blogs/multi-vector/).

## Image stage: screenshots, photos and scanned pages (owner 2026-10-03; OFF by default)

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03._ It extends Intake's existing image index (collection `IntakeImageV1`:
> `image_embedders`, `image_facts`, `original_time`, `image_target`, `image_search`) with a catalog source; it is not a second
> lane. Switch: `INTAKE_IMAGE_STAGE=on` in the worker environment. Pattern source: CocoIndex's `image_search` (CLIP) and
> `multi_format_indexing` (ColPali) examples and the v1 named-vector docs (`image_search`, `image_search_colpali`, `multi_format_indexing`:
> one memoized unit per image, a named vector per model, MaxSim for the multi-vector); the `face_recognition` and
> `pdf_elements_embedding` examples were read and not used (faces are a separate step; scanned pages are rendered whole).

It runs as a loop of bounded SLICES (default 40 files / 256 MiB), after the text passes and also on a tick where the catalog
did not change (it is driven by its own ledger, not by the watermark, so a backlog drains over successive cycles; at most 150
slices per cycle keeps Temporal history small). Five single-purpose Activities per slice:

| Activity | One job | Reads | Writes |
|---|---|---|---|
| `superindex_image_fetch_activity` | select the next slice of images (inventory `media`, formats the embedders accept, at most 25 MiB, exact copies grouped by SHA-1) or, when none are left, scanned PDFs (extract stage status `skipped_no_text` joined to `inventory/pdf/<cycle>.parquet`, rendered to PNG pages by pypdfium2, at most 20 pages); stream each object from the bucket to the spool in 4 MiB windows | inventory, bucket | spool, `datasets/images/slices/<slice>.json` |
| `superindex_image_facts_activity` | original time, device, GPS, size, and screenshot / photo / scan (`image_kind.py`, rule order recorded as `image_kind_basis`) | spool | `images/facts` |
| `superindex_image_ocr_activity` | the text in the picture, engine `INTAKE_IMAGES_OCR_ENGINE` (`tesseract` default; `doctr` selectable once installed) | spool | `images/ocr` |
| `superindex_image_embed_activity` | ONE slot, one Activity per slot in parallel (below) | spool | `images/vectors/<slot>` |
| `superindex_image_publish_activity` | facts + text + vectors -> Weaviate `IntakeImageV1` (schema added if needed), ledger, release the spool | the tables above | Weaviate, `images/published` |

Why Python Activities and not n8n flows (owner prefers n8n for image/PDF steps, 2026-10-02): a flow is one n8n execution per file
and one Activity pair per file in Temporal history; 677,730 occurrences cannot ride that. One Activity per bounded slice can.
The per-file gateway tools (`ocr_image.*`, `ocr_pdf.*`, `image_meta.*`, `image_thumb.*`, built in worktree
`pdf-image-steps-20261002`, uncommitted) stay the right shape for the Proffer lane's single-document steps.

Identity is content: the catalog SHA-1 (`k-<md5 of the locator>` when the catalog has none), `<sha1>#p<n>` for a PDF page; the
Weaviate id is `uuid5("intake-image-content:<identity>")`, so an exact copy elsewhere is the same object and carries an
`occurrences` count; every occurrence stays in the catalog. A moved file needs no work (the locator `vault_key` is a
property; relocation patching for images is not built, see below).

**Slots** (`INTAKE_IMAGES_SLOTS`, default `image_maxsim,image_single`; every one stays selectable; receipts carry each slot's licence basis):

| Slot | Model | Default coverage | Notes |
|---|---|---|---|
| `image_maxsim` | Jina `jina-embeddings-v4` multi-vector (MaxSim) | `INTAKE_IMAGES_MAXSIM=all` (owner 2026-10-03: primary, screenshots and images) | needs `JINA_API_KEY`; Qwen Research License, owner accepted non-commercial personal case use; ~0.39 MB per image |
| `image_single` | NIM `nvidia/llama-nemotron-embed-vl-1b-v2`, 2048-d (or Google `gemini-embedding-2`) | every image | selectable and ready |
| `image_colqwen` | ColQwen2.5 on a GPU endpoint you run | `INTAKE_IMAGES_COLQWEN=screenshots` | client `ColQwenEmbedder`; wrapper `deploy/colqwen_modal.py` (not deployed); needs `INTAKE_IMAGES_COLQWEN_URL` |
| `image_clip` | Weaviate `multi2vec-clip` ViT-B/32, computed by Weaviate on insert from a 224-px thumbnail | when `INTAKE_IMAGES_CLIP=1` | needs the CLIP container in `Probata/deploy/data-weaviate-native-v1.yaml`; restarts Weaviate |

**Photo tier** (`INTAKE_IMAGES_PHOTO_TIER`): `all` (default) gives every image every enabled slot, CLIP, when on, being a second
opinion. `clip_first` gives photos CLIP only (the free first pass) and keeps the hosted slots for screenshots and scans; a
photo a search shortlists is promoted by embedding it into the hosted slots later (promotion is not built, see below).

**Search**: `POST /images/search {query, limit, mode: hybrid|keyword, kind: screenshot|photo|scan}`; literal OCR text (BM25,
verified as a substring), plus one vector lane per enabled slot; hits carry `provider`, `bucket`, `vault_key`, `content_sha256`,
`image_kind` and their channel. `/filesystem/search` still adds image hits for the directory lane.

Memory: a slice is at most 256 MiB on disk and one image is decoded at a time; Tesseract is a subprocess; embeds run 4 at a time;
the worker stays inside its 4 GiB cap. Weaviate storage: a MaxSim bag is ~0.39 MB per image (5.8 GB for the 14,948 named
screenshots, 264 GB for 677,730 images): enable `all` only after Weaviate's multi-vector encoding is on.

## Deploy order (each step is a shared write; the parent session runs them)

1. Apply `casebible/tools/superindex_source_current_20261003.sql` on the catalog (one view, one grant).
2. Merge; redeploy Coolify app `superindex` (uuid `f12skzwshwp85b1k4lbgm0pp`): the compose now builds two services. Needs
   `INTAKE_CATALOG_DSN` (already set) and Coolify env for the new settings (names in `deploy/superindex.compose.yml`).
3. Rebuild and redeploy the Go proffer worker (registers `SuperIndexCycleWorkflow`; `worker_test.go` expects 5 workflows).
4. Run `superindex-schedule ensure` (TEMPORAL_ADDRESS, TEMPORAL_TASK_QUEUE = the proffer worker's queue) with the schedule
   **paused** (`Paused: true`), then `describe`. Unpause only after the proof below.
5. Image stage (optional, after 1-4): set `INTAKE_IMAGE_STAGE=on`, `JINA_API_KEY`, `INTAKE_IMAGES_WEAVIATE_URL` in the Coolify
   environment; redeploy. `INTAKE_IMAGES_CLIP=1` and the CLIP container are a separate step: deploy `data-weaviate-native-v1`
   (Weaviate restarts) first, read the container's real memory, then set the flag. The collection is created on first publish
   (`ensure_schema`); a collection that already exists without `image_clip` / `image_colqwen` must be recreated to add them.
6. Refresh the catalog counts: `python scripts/image_counts.py` on the VPS (unique images by SHA-1, screenshot-named, PDFs).

**Five-file proof** (a bounded proof run, never committed, test objects purged afterwards): start the workflow by hand with
`path_prefix` set to a folder holding five synthetic files you place in the bucket first: (1) a phone screenshot PNG named
`Screenshot_...png`, (2) a JPEG photo with camera EXIF, (3) a second copy of the screenshot in another folder (one object,
`occurrences = 2`), (4) a two-page scanned PDF with no text layer, (5) a HEIC (counted `unsupported_format`, not processed).
Read back with `POST /images/search`: the question about the screenshot's text returns it first by OCR literal and by the
vector lanes; `kind=screenshot` excludes the photo; the PDF's pages appear as `source_kind: pdf_page`, `page` 1 and 2. Then
delete the five bucket objects, set the five Weaviate objects `active=false`, and remove the test ledger rows.


## Not built, stated

- Image and PDF steps: PDFs with a text layer are extracted in-process (pypdf). Images and scanned PDFs are handled by the
  image stage above (off by default). Not built there: relocation patching of a moved image's `vault_key`; promotion of a
  shortlisted photo into the hosted slots; PDF pages beyond `INTAKE_IMAGES_PDF_MAX_PAGES` and PDFs inside ZIPs; HEIC; thumbnails
  for hits (a hit opens the original by locator); faces; Surreal entity writes for images; retirement of image objects.
- Two Activities the owner asked for on 2026-10-03 are NOT built because each needs a storage decision first: entities and
  events from screenshot OCR text (LangExtract / the default extractor), and corroboration links between screenshots and
  existing message records (`screenshot_match.py` is the tested matcher; nothing stores its result yet). The options are in
  `docs/LOG.md` (2026-10-03, image stage).
- Retirement of Weaviate objects for files that left the catalog: reported by count, not performed (a content-addressed
  object can be shared by another live file).
- Parquet `/search` for staged documents has no vectors (they live in `datasets/vectors` and Weaviate); semantic search of
  staged chunks is `POST /chunks/search`. A moved file's Parquet document rows keep its first key; Weaviate is patched.
- `IntakeCorpus` (37,857 objects, the 2026-09-22 proofs) is untouched and still searched by `/filesystem/search`.
