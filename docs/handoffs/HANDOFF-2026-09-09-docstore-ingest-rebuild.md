# HANDOFF — probata docstore ingest rebuild (2026-09-09)

> _Byline: Claude Code · Opus 5 · 2026-09-09 11:10 EDT_
STATUS: PARTIAL
BUILD_STATUS: UNKNOWN — `flow_docs.py` v5 compiles and imports, but has never been run end to end.

## Read this first

The SurrealDB docstore is **empty**. It held 489 documents and 11,436 verified chunks
at 09:45. The machine was taken down at ~10:40 by a runaway ingest process of mine
(17 GB resident), and on restart SurrealDB re-initialised its RocksDB directory:
`probata/.docstore/db` now holds 7 files / 0.1 MB, all stamped 10:41:36, and the query
`SELECT count() FROM document` answers `The namespace 'probata' does not exist`.

Nothing irreplaceable was lost. The index is derived data. All 564 source markdown
files, the 489-row mapping CSV, the schema, and the pipeline are intact on disk, so a
rebuild is a re-run, not a re-creation.

## Verified-live state

| Thing | State |
|---|---|
| SurrealDB | up, pid varies, `127.0.0.1:8462`, `rocksdb://…/probata/.docstore/db`, started by the Startup-folder launcher. `/health` 200. |
| Store contents | EMPTY. Namespace `probata` does not exist. Schema not applied. |
| Source docs | 564 `.md` files under `probata/docs/`, untouched. |
| Mapping CSV | `scripts/docstore/mapping/docs-ingest-mapping.csv`, 489 markdown rows + 14 non-markdown rows, fingerprint `67f2a8211600ba64`. |
| `flow_docs.py` | v5, 497 lines (was 874). Imports clean, mapping loads, `max_inflight=4`, RSS ceiling 1024 MB. **Never executed.** |
| Embeddings | NIM `nvidia/nemotron-3-embed-1b`, 2048-dim, API-only. Probed working earlier today. |
| Git | Everything under `scripts/docstore`, `plugins/docstore`, `codex/docstore` and today's `docs/` output is UNTRACKED. Nothing committed. |

## What v5 changed and why (the memory failure)

v1–v4 caused the crash. The cause was not CocoIndex, Python, or the machine — it was
my code ignoring the cookbook. The cookbook's core pattern is
`TargetState = Transform(SourceState)`: stream a source, one component per item,
**declare one small target row at a time**.

| Cookbook | v1–v4 did |
|---|---|
| one chunk table target, `declare_record` per chunk | one bundle per document holding every chunk and every 2048-float embedding |
| `coco.map(process_chunk, chunks, …)` inside the file component | a Python list of all chunk rows |
| `mount_each(process_file, files.items(), …)` | `asyncio.gather` over 489 `use_mount` calls, which pins every result until the last finishes |
| `memo=True` on `FileLike` (has `content_fingerprint`) | a custom memo-key function that re-read and re-hashed every file |
| the connector's table target | a hand-rolled TargetHandler, sink and fingerprinting |

v5 is written from Patterns 1 and 2 directly. Live memory is one chunk row per
in-flight slot. A `psutil` sampler thread aborts the process at
`DOCSTORE_MAX_RSS_MB` (default 1024) so a runaway can never take the desktop again.

## Real bugs found today, all still valid and worth keeping

1. **Change detection never worked.** The pipeline was memoized on the mapping-CSV row,
   not on file content, so editing a document's text never re-ingested it. v5 memoizes
   on `FileLike` (content fingerprint) plus the CSV fingerprint.
2. **`UPSERT … CONTENT` over an existing row always failed.** `CONTENT` is a full
   replace and a field's `DEFAULT` only fires on create, so `confidence` became NONE and
   the write died. Documents were writable exactly once. (v5 uses the connector, which
   sidesteps this, but the finding matters for any hand-written SurrealQL.)
3. **`age_days` blocked every new document.** Inside a transaction, `RETURN VALUE id` on
   a new row evaluates the COMPUTED field before `observed_at`'s DEFAULT is visible →
   `Cannot perform subtraction with 'datetime' and 'none'`. A naive `OR time::now()`
   fallback then produced a negative duration. **Already fixed in
   `schema/010_documents.surql`** with a guard returning `0s`; verified live.
4. **My verification method was wrong**: hashing raw file bytes made 97 correctly-written
   documents look stale, because `read_text()` normalises newlines and the pipeline
   hashes the normalised text. Any checker must hash the way the pipeline hashes.

## UNRESOLVED

- **v5 has never run.** It cannot run against the current schema — see the next section.
- **The store must be rebuilt from zero**: apply schema, then full ingest.
- **14 non-markdown mapping rows** (3 Go files, 9 `.txt`/`.err` dumps, 1 OpenAPI `.yaml`,
  1 recovered `.txt`) are typed in the CSV but excluded by the `**/*.md` matcher. Listed
  in `scripts/docstore/mapping/mapping-exceptions.md`. Needs an owner ruling.
- **The mirror `--apply`** (496 moves, 225 link rewrites, dead-link gate must not rise)
  has not run. `scripts/docstore/mirror_docs.py`.
- **Nothing is committed to git.**
- **Weaviate skills** were installed into `probata/.agents/skills/` instead of
  `~/.agents/skills`, because `npx skills add` installs relative to cwd.

## Pending owner decisions

- **Schema change (blocking the rebuild).** The built-in SurrealDB connector serialises
  rows as JSON and cannot write `chunk.document` as `record<document>` — a JSON string is
  not coerced into a record link (verified live). The cookbook's answers are to
  denormalise (Pattern 2 keeps `filename` on the chunk row) or use a relation target.
  v5 denormalises, so `scripts/docstore/schema/020_chunks.surql` needs:
  ```surql
  DEFINE FIELD OVERWRITE source_path ON chunk TYPE string;
  DEFINE FIELD OVERWRITE document    ON chunk TYPE option<record<document>>;
  ```
  and `fn::docs_search` in `090_docs_api.surql` must filter on `chunk.source_path`
  instead of traversing `chunk.document`. WHY it matters: without it every chunk write is
  rejected by the SCHEMAFULL table. The store is empty, so this is now a zero-risk edit
  with no migration. SHORTCOMING: the graph link from chunk to document goes away unless
  a relation target is added later.
- **Budget.** This session consumed roughly 18% of a weekly 20x Max allowance, largely on
  churn: eight ingest runs and repeated wrong diagnoses. Keep the next session short and
  scoped.

## Next steps, in order

1. Apply the two `chunk` field definitions above and update `fn::docs_search` to use
   `source_path`. Re-apply `schema/000`–`090` to the empty store (`POST /sql` with basic
   auth and `surreal-ns` / `surreal-db` headers — **not** `surreal sql`, which reads stdin
   line by line and breaks multi-line DEFINEs, and **not** `surreal import`, which needs
   `OPTION IMPORT`).
2. Scoped smoke test: `DOCSTORE_ONLY_FILES="docs/BUILD_PLAN.md,docs/adr/0001-fresh-build-from-skeleton.md"`
   with the RSS ceiling armed. Confirm rows land and report peak resident memory.
3. Full ingest, 489 files. Watch peak RSS; the ceiling should never trigger.
4. Verify with a checker that hashes `read_text().encode()`: expect 489 documents,
   0 absent, 0 stale, 0 chunks with a wrong embedding dimension.
5. Then the mirror `--apply` and the dead-link gate.
6. Then commit.

## Owner working-style contract

- **Get approval before making a change.** Stated repeatedly today and broken repeatedly.
  Propose, wait, then act — for file edits, schema application, and any run.
- Read the supplied cookbooks and use them as templates. Do not skim and invent.
- Verify before claiming; never report success from a proxy signal.
- This is a nine-year-old workstation, not a server. Check memory before running
  anything at scale, and keep the ceiling armed.
- Structured answer-first replies: bullets, labelled blocks, white space.
- Never hard-delete; quarantine. Byline every artifact.
