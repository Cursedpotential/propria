# Case store — embedded SurrealDB (search + graph)

> _Byline: Claude Code · Fable 5.1 · 2026-09-07._
> _Byline: Claude Code · Sonnet 5 · 2026-09-07 — owner orders 13:09-13:16: filed/draft works
> registers, court-event vs master-timeline split, memos, reference data, an evidence log,
> evals, case status, per-record source provenance, a case-extract/v1 importer, and a
> platform (probata) export bundle. All additive — see `docs/2026-09-07-case-store-registers.md`
> for the full design writeup, owner-order quotes, and deferred items._
> _Byline: Claude Code · Sonnet 5 · 2026-09-08 — "content lives in the SurrealDB store": added
> `src/content-store.ts` (read-only cache over `reference`/`source`) and
> `scripts/load-content-to-store.mjs` (the loader). See "Content in the store" below._

`src/store.ts` (+ `src/store-tools.ts`) adds an embedded SurrealDB case store with full-text
search, optional vector search, and graph traversal to the family-court-toolkit MCP server.
~~It is a separate, self-contained addition — it does not import from or modify `src/server.ts`
or `src/core.ts`.~~ **Corrected 2026-09-08:** `src/core.ts`, `src/court-language.ts`, and
`src/survival-guide.ts` now import `src/content-store.ts` (which imports `src/store.ts`) for
their store-first content reads — see "Content in the store" below. `src/store.ts` itself still
does not import from or modify any of those three files, and `src/server.ts` is still not
modified by `src/store.ts`/`src/store-tools.ts` beyond the pre-existing
`registerStoreTools(server)` call.

## A note on record ids with hyphens (e.g. every case-extract/v1 id)

**Verified live, fixed 2026-09-07:** this SurrealDB build's `RecordId.toString()` renders an
id containing any character outside `[A-Za-z0-9_]` — a hyphen, most commonly — wrapped in its
own quoted-identifier brackets: `memo:⟨probe-memo-1⟩` instead of `memo:probe-memo-1`. Since
every case-extract/v1 id is `<row>-<seq>` (e.g. `A1-7`), this would otherwise show up
everywhere (search hits, docket/timeline entries, `case_source`, graph neighbors) AND silently
break round-tripping — feeding a displayed id back into `case_source`/`case_graph`/`case_put`'s
`relations` would construct a record with the literal brackets baked into its id, not the
original record. `refToString()`/`normalize()` now strip a matching `⟨...⟩` pair on the way
out, and `parseRef()` strips it defensively on the way in too, so a hyphenated id always
displays and round-trips as the plain string a caller wrote.

## Install / build — read this before running anything

```
npm install surrealdb @surrealdb/node
npm ci --omit=dev
npm run build
```

**`npm ci --omit=dev` is required after installing these two packages, every time.**
`@surrealdb/node` ships native `.node` binaries (one per platform/arch) as real files under
`node_modules/@surrealdb/node/`. `build.mjs` marks both `surrealdb` and `@surrealdb/node` as
esbuild `external` — **verified live**: bundling them (no `external`) makes `esbuild` itself
succeed, but the resulting `dist/store.js` fails at runtime with:

```
Cannot find native binding. npm has a bug related to optional dependencies
(https://github.com/npm/cli/issues/4828). Please try `npm i` again after removing both
package-lock.json and node_modules directory.
```

This happens because `@surrealdb/node`'s platform-detection code does `require("./surrealdb-node.<platform>.node")` with a path relative to **its own package directory** — bundling inlines that
`require` call into `dist/store.js`, so the relative path then resolves against `dist/`
instead, and the native binary is never found. Externalizing keeps the `require` where it
belongs (inside `node_modules/@surrealdb/node/`), so the plugin **must ship real
`node_modules/surrealdb` and `node_modules/@surrealdb/node` directories** alongside
`dist/`, not just the bundled `dist/*.js` files. If this plugin is packaged for
distribution, include those two `node_modules` subtrees in the package (a full
`npm ci --omit=dev` in `mcp-app/` before packaging is the simplest way to guarantee this).

## Graceful degradation

Every store tool resolves the store lazily on first use via `getStore()`. If `surrealdb` or
`@surrealdb/node` fails to `import()` (missing after install, wrong platform, corrupted
native binary, etc.), every `case_*` tool returns:

```json
{ "available": false, "reason": "surrealdb/@surrealdb/node failed to load: ..." }
```

instead of throwing or crashing the MCP server. The server itself starts fine either way —
only the store tools are affected.

## Where the database lives

`CUSTODY_CASE_DB` env var, else `~/.config/family-court-toolkit/case.db` (a `rocksdb://`
embedded database — namespace `fct`, database `case`). Schema migration
(`DEFINE ... IF NOT EXISTS`) runs idempotently on first connect each process.

### Windows absolute-path gotcha — fixed, worth knowing about

**Verified live while building this:** passing a Windows absolute path straight into
`rocksdb://C:/Users/...` mis-parses the drive letter as a URL "authority" (host) and
silently drops it. The connection reports success, but the database is actually created
**relative to the process's current working directory** (e.g. `<cwd>/C/Users/matts/...`),
not at the real `C:\Users\...` location — a silent, hard-to-notice data-location bug.

The fix (`toOpaqueIfWindowsDriveAbsolute()` in `src/store.ts`) rewrites any
`scheme://C:/...`-shaped URL into the scheme's **opaque** form, `scheme:C:/...` (no `//`),
which this SurrealDB build parses correctly as one absolute path. This is handled
automatically for both `CUSTODY_CASE_DB` and the default path — you do not need to do
anything — but if you ever construct a `rocksdb://` URL yourself (e.g. in a new script),
route it through `resolveDbUrl`-equivalent logic rather than string-concatenating
`rocksdb://` in front of a Windows path. `tests/store.test.mjs` has a regression test for
this (`regression: a Windows absolute drive path (C:/...) opens at the real location, not
relative to cwd`).

`surrealkv://` **hangs on Windows** — never use it (verified in an earlier session; not
re-verified here since `rocksdb://` already works). `mem://` is used for tests.

## Embeddings are optional

If `NVIDIA_API_KEY` or `NIM_API_KEY` is set (checked in that order: process env, then a
tolerant `KEY = value` scan of `~/.secrets/*.env` — never printed, never `source`d), vector
search calls `https://integrate.api.nvidia.com/v1/embeddings` with model
`nvidia/nemotron-3-embed-1b` (2048-dim by default; configurable via `CUSTODY_EMBED_DIM` at
first migration, recorded in the `meta:config` record). `data:image/...;base64,...` URIs are
stripped from text before embedding (this model 503s on them). Without a key, `case_search`
in `hybrid`/`vector` mode degrades to text-only search and sets `degraded: true` with a
`degraded_reason`. `case_search({ mode: "text" })` never touches embeddings at all.

## Schema

**Original 12 tables** (unchanged): `person`, `child` (SCHEMAFULL — only
`{ initials, age, source?, extract_id?, imported_at? }`; a `name` field is rejected both by
`case_put`'s own validation and by the database schema itself), `order`, `hearing`,
`deadline`, `event`, `message`, `exhibit`, `factor` (seeded with the 12 MCL 722.23 letters
a-l), `source`, `note`, `court` (one record, `court:main`).

**New tables (owner orders 2026-09-07 13:09-13:16)**, all SCHEMALESS:

| Table | Purpose | Key fields |
|---|---|---|
| `court_event` | the REAL court-event timeline (never merged with `event`) | `kind` (hearing\|order\|filing\|service\|deadline\|referee-recommendation\|objection\|conference\|foc-appointment\|evaluation-appointment), `date`, `title`, `court`, `judge_or_referee`, `case_no`, `outcome`, `status` (past\|upcoming\|unknown — **computed from `date` vs now whenever a put omits `status`**, and never clobbered by a later status-less patch), `document_refs[]` |
| `filing` | the "filed works" register | `title`, `doc_type` (brief\|motion\|response\|objection\|affidavit\|proof-of-service\|exhibit-list\|letter\|other), `filed_or_planned` (filed\|planned\|draft-only), `date`, `court`, `served_on[]`, `service_method`, `status`, `path`, `sha256`, `document_refs[]` |
| `draft` | the "draft works" register | `title`, `doc_type`, `version`, `status` (working\|final\|abandoned\|superseded), `path`, `notes`, `date` |
| `memo` | analysis/strategy/weakness/direction/finding/risk/goal/issue/advice | `kind`, `title`, `text`, `status` (open\|active\|resolved\|superseded), `factors[]`, `author`, `created_at`, `supersedes` |
| `reference` | behavior-detection patterns, ontology, entity rules, lexicons, templates, trusted authorities — **the ruler, not the subject: loaded whole, never analyzed** | `kind` (behavior_pattern\|ontology_term\|entity_rule\|lexicon\|template\|authority), `key`, `category`, `pattern`, `definition`, `aliases[]`, `severity`, `factors[]`, `source`, `version` |
| `evidence_log` | "for when we get to that point" — independent of the platform's H1/H2/H3 custody hashing | `logged_at`, `action` (received\|collected\|hashed\|reviewed\|produced\|disclosed\|admitted\|excluded\|returned), `by`, `hash`, `path`, `notes` (the exhibit is RELATEd via the `logs` edge, not a plain field) |
| `eval` | a place to store evals and reports | `kind` (eval\|report\|review\|audit), `title`, `subject`, `score`, `verdict`, `text`, `path`, `tool_or_model`, `created_at` |
| `case_status` | one singleton record, `case_status:current` | `phase`, `posture`, `next_court_event`, `open_deadlines[]`, `last_updated` (auto-stamped on every set), `notes` |

Standing orders reuse the existing `order` table: `kind` (standing\|interim\|final\|
referee-recommendation) and `in_force` (boolean) are just ordinary SCHEMALESS fields on it —
no new table was needed.

**Source linking (any table):** every data record may carry a `source` object —
`{ path, sha256, r2_path, locator, url, row, extract_id }` — preserved as-is by `case_put`
(SCHEMALESS tables accept it with no DDL change; `child` needed an explicit
`FLEXIBLE TYPE option<object>` field since it is SCHEMAFULL — a plain `TYPE option<object>`
enforces the nested object's own shape too, verified live). `case_search`, `case_docket`, and
`case_timeline` results include each row's `source` (or `null`). `case_source({ id })` returns
a record's `source` block, whether `source.path` exists **locally** (no network calls), and
the r2 pointer.

**New edges (owner orders 2026-09-07 13:09-13:16)**: `drafted_as` (draft→filing),
`responds_to` (filing→filing|court_event), `entered_at` (order→court_event), `logs`
(evidence_log→exhibit), `evaluates` (eval→draft|filing|memo), `about` (memo→any),
`matches_pattern` (event|message→reference) — created the same way as every other edge, via
`case_put`'s `relations[]`. Original 6 edges unchanged: `evidences` (exhibit→event),
`supports_factor` / `contradicts_factor` (event|exhibit→factor), `sent_by` / `sent_to`
(message→person), `filed_in` (order|hearing→court).

`event`, `message`, and `exhibit` carry `occurred_at` and `known_at`; `court_event.date`,
`filing.date`, `draft.date`, `evidence_log.logged_at`, and `eval.created_at` are the other
date-shaped fields — all cast from an incoming ISO string to a real `Date` object before they
cross into SurrealQL (`castDateFields()`, renamed from `castTwoClocks` when the single-date
tables were added; a plain string is stored as a string, not a datetime, and `<=`/`ORDER BY`
need the real type). Full-text (`FULLTEXT ANALYZER ... BM25`) indexes on `event.description`,
`message.body`, `note.text`, `exhibit.label`, `memo.text`, `filing.title`, `draft.title`,
`court_event.title`, `reference.definition`, `eval.text` (10 tables total). HNSW (cosine)
indexes on `embedding` for `event`, `message`, `note`, `memo`, `eval` (5 tables total).

## Tools

**Original 9** (unchanged request shapes; `case_export`/`case_import`/`case_timeline` gained
new OPTIONAL params, see below): `case_put`, `case_query`, `case_search`, `case_graph`,
`case_factor_map`, `case_timeline`, `case_export`, `case_import`, `case_summary`.

**New 7 (owner orders 2026-09-07 13:09-13:16)**:

| Tool | action(s) | Use it to |
|---|---|---|
| `case_status` | get \| set | Read or MERGE-update the `case_status:current` singleton. |
| `case_docket` | (none — filters only) | List filings + drafts + orders + **upcoming** court_events, filterable by `status`/`doc_type` (filing/draft) or `in_force` (order). |
| `case_memo` | put \| list \| latest | Upsert a memo; list by kind/status; get the newest memo per kind. |
| `case_evidence_log` | append \| list | Append one entry (optionally RELATEd to an exhibit via `logs`); list by exhibit/action. |
| `case_eval` | put \| list | Upsert an eval/report row; list by kind/subject. |
| `case_reference` | load \| list \| match | Load reference data from a file or `content/reference/` (+ the existing court-language lexicon, see below); list by kind/category; match `pattern`/`aliases[]` against text, with spans. |
| `case_source` | (none) | Given a `"table:id"` ref, return its `source` block + local path existence + r2 pointer. |

**`case_timeline`** gained `mode: "court" | "master" | "merged"` (default `merged`) and
`upcoming: boolean` (court lane only). `mode: "court"` = `court_event ∪ hearing ∪ deadline ∪
order`; `mode: "master"` = `event ∪ message ∪ exhibit` (the corpus-extracted timeline, kept
DELIBERATELY separate from the real court-event timeline — owner order). Every entry now
carries a `lane` tag; `known_by` still applies to the master lane only (the two-clock
discipline never applied to the court lane, which has no `known_at` concept).

**`case_export`** gained `format: "snapshot" | "platform"` (default `snapshot`, unchanged
behavior). `format: "platform"` writes one NDJSON file per table plus `edges.ndjson` and
`manifest.json` (`schema: "fct-platform-bundle/v1"`) to a new
`~/.config/family-court-toolkit/exports/platform-<timestamp>/` directory — the export
mechanism to the probata evidence platform (owner order). Every row is
`{ id, type, ...the record's own fields }`; children stay redacted to initials+age (the store
never held anything else for them).

**`case_import`** now auto-detects a case-extract/v1 envelope (a single file whose top-level
`schema` is `"case-extract/v1"`, OR a directory of them) in addition to the existing snapshot
and Vincent-schema shapes — see "Importing case-extract/v1" below.

**Sidecar compatibility aliases** (`caseMemo`, `caseStatus`, `caseSource`, `caseReference`,
`caseEvidenceLog`, `caseEvals` — single-arity, store.ts function-name level only, not MCP
tools): `app/` (a separate, concurrently-developed Tauri work surface — see `app/README.md`
"Where the store lives") dynamically dispatches store.ts exports BY NAME via
`app/sidecar/lib/store-client.mjs`'s `callStoreFn`, using a different naming convention
(single combined read functions) than this file's own established one (one verb-function per
concern — `caseMemoPut`/`caseMemoList`/`caseMemoLatestByKind`, `caseStatusGet`/`caseStatusSet`,
etc. are the real API). Thin read-only aliases matching the app's expected names were added so
`app/`'s `callStoreFn` stops returning `{ queued: true }` for these surfaces without this file
adopting a second naming convention — see the "Sidecar compatibility aliases" section in
`src/store.ts`. **Known fallout, not fixed here (out of scope — `app/` is another agent's
active work):** `app/sidecar/tests/server.test.mjs` and `store-client.test.mjs` contain tests
that assert `queued: true` for exactly these tool names ("until the companion agent lands
them") — those two test files will need updating by whoever owns `app/` now that the real
functions exist. `app/src/types/store.ts`'s `Queued<T>` stub shapes (e.g. `CaseStatusSnapshot`,
`CaseMemo`) also do not exactly match this store's real field names/kinds (the owner's literal
order for `memo.kind`, for example, is far broader than the app stub's 4-value union) —
reconciling those types is that same follow-up, not done here.

## Importing case-extract/v1 (Case Bible intake)

`caseImportExtract(store, fileOrDir)` (wired into `case_import`/the `case_import` tool
automatically) imports one case-extract/v1 envelope file, or every envelope file under a
directory tree (`<row>/<slug>.json`, skipping `_INDEX.json` and any `_SCHEMA*` file — by NAME,
so a malformed sidecar file never even gets parsed, let alone aborts the import). Schema doc:
`E:\AI_Workspace\casebible\_intake\extracted-json-20260907\_SCHEMA-case-extract-v1.md`
(read-only reference — this store never writes there). Record-type mapping: `person`→`person`
(role `child` → the `child` table, `{ initials, age }` ONLY), `event`→`event`,
`court_event`→`court_event`, `filing`→`filing`, `draft`→`draft`, `message`→`message`,
`exhibit`→`exhibit`, `source_authority`→`source`, `issue`/`risk`/`goal`→`memo` (kind = the
record type), `note`→`memo` when its own `kind` is strategy/advice/finding/pattern, else
plain `note`, `entity_rule`→`reference` (kind: `behavior_pattern`).

**Every child's real name AND aliases are collected across the WHOLE batch first**, then
substituted for that child's initials in every free-text field of every record before
anything is written — closing the same name-in-prose leak `importVincentSchema`'s
`redactChildNames()` closes for the older Vincent-schema importer (verified: the owner's real
corpus narrates events like "Birth of \<child's full name\>" in prose, not just in a formal
`person` entity). Every record gets a provenance block (`source: {path, sha256, row, locator}`
from the envelope + `extract_id` (`<row>-<seq>`) + `imported_at`) and is UPSERTed by
`extract_id` used directly as the record id — **idempotent**: re-running the same import
merges onto the same records rather than duplicating them.

## Tools (full list)

`case_put`, `case_query`, `case_search`, `case_graph`, `case_factor_map`, `case_timeline`,
`case_export`, `case_import`, `case_summary`, `case_status`, `case_docket`, `case_memo`,
`case_evidence_log`, `case_eval`, `case_reference`, `case_source` — see `src/store-tools.ts`
for full descriptions/schemas, or `skills/family-court-toolkit/references/case-store/SKILL.md`
for usage guidance. Register them into `server.ts` with:

```ts
import { registerStoreTools } from "./store-tools.js";
// ...
registerStoreTools(server);
```

## CLI twin

`scripts/case-cli.mjs` (Node — Python cannot open the embedded RocksDB store concurrently
with the MCP server process) exposes every tool as a subcommand: `put`, `query`, `search`,
`graph`, `factor-map`, `timeline`, `export`, `import`, `summary`. Run
`node scripts/case-cli.mjs` with no arguments for full usage. It reads `CUSTODY_CASE_DB` the
same way the MCP tools do, and closes its connection cleanly before exiting (a rocksdb LOCK
file left held by an abrupt `process.exit()` would otherwise hang the *next* open against the
same path — verified live).

## Seeding from the Vincent-style case schema

`scripts/import-vincent-schema.mjs <path> [--dry-run|--commit]` reads a Vincent-style intake
file and loads it via `store.importVincentSchema()`. Defaults to `--dry-run` (imports into a
throwaway `mem://` store and only prints counts) — pass `--commit` to actually write into the
real case store. **Never run `--commit` without the owner's explicit go-ahead**; seeding the
real store is the owner's decision, not something a script does silently.

Two real findings from building this against the owner's actual intake file
(`~/.config/family-court-toolkit/intake/vincent_case_schema_2025-12-15.md`):

1. **The file has no ` ```json ` fence at all** — the JSON object is embedded mid-sentence
   ("...legal argumentation.json{"parties": [...`). `extractVincentJson()` handles three
   shapes in order: plain JSON, a proper fenced block, and a brace-balance scan for this
   unfenced case.
2. **The file's own JSON is not quite valid JSON** — every
   `<ungrounded-authority probable-jurisdiction="US-MI">` tag has unescaped inner double
   quotes around `US-MI`, which breaks JSON string syntax. `repairKnownVincentQuoteBug()`
   fixes this one specific, describable pattern (swaps those two inner quotes for single
   quotes) before parsing; it is a no-op on any file that doesn't contain it.

A third, more important finding: the source file's own `timeline` narrative text can contain
a child's real name in prose (e.g. `"event": "Birth of Kailah Salem"`) even though the
`parties[]` entity itself is correctly identified as a child and reduced to initials.
Rejecting the `child` table's `name` field alone does **not** stop a name embedded in free
text from flowing into `event.description` / `exhibit.label` / `note.text` verbatim.
`importVincentSchema()` therefore captures every child's real name from `parties[]` *before*
discarding it, and runs a redaction pass (`redactChildNames()`) over every free-text field it
writes afterward, replacing the child's full name and first name (case-insensitive, word-
boundary matched) with their initials. Verified live: before this fix, a full case-store
export of the real file contained the child's name once (in `event:vincent-timeline-1`'s
description); after the fix, it does not.

## Content in the store

> _Byline: Claude Code · Sonnet 5 · 2026-09-08._

The plugin's static content — the verification ledger, the master source directory, the
court-language lexicon/examples/templates, and the survival-guide templates + event context
packs — now *also* lives in the case store, alongside the case data. `src/content-store.ts`
is a thin, read-only cache over the `reference` and `source` tables:

```ts
getReference(key: string): Promise<{ kind?; key?; body?; data?; source_path?; sha256?; loaded_at? } | null>
getSources(): Promise<Record<string, unknown>[] | null>   // every row of the `source` table
```

Both return `null` on a store failure or a missing row — they never throw. Results are cached
in-process for 5 minutes.

### Keys loaded (`scripts/load-content-to-store.mjs`)

| Source file | Store location | `kind` | Payload field |
|---|---|---|---|
| `content/toolkit/ledger.json` (each entry) | `source:<ledger id>` | — | every original field + `r2_path`/`sha256`/`loaded_at` |
| `content/custody-guide/master_source_directory.md` | `reference:master-source-directory` | `directory` | `body` (whole file) |
| `content/tools/court-language/lexicon.json` | `reference:court-language-lexicon` | `lexicon` | `data` (parsed JSON) |
| `skills/court-language/EXAMPLES.md` | `reference:court-language-examples` | `template` | `body` (whole file) |
| `skills/court-language/TEMPLATES.md` | `reference:court-language-templates` | `template` | `body` (whole file) |
| `content/tools/survival-guide/TEMPLATE.md` | `reference:survival-guide-template` | `template` | `body` (whole file) |
| `content/tools/survival-guide/CARD_TEMPLATE.md` | `reference:survival-guide-card-template` | `template` | `body` (whole file) |
| `content/tools/survival-guide/events/<id>.json` (each file) | `reference:<id>` | `event_pack` | `data` (parsed JSON) |

`r2_path` is only ever set when a local file actually exists at
`content/custody-guide/sources/primary/<ledger entry's file_path>` — as of 2026-09-08 the
ledger's `file_path` values are relative to `content/toolkit/` (the ledger's own write-ups), not
to `sources/primary/` (the 143 raw downloaded primary-source files), so this almost never fires
against the current corpus; the convention (`casebible-sorted/fct-sources/primary/<relative
path under sources/primary>`) is in place for when those two corpora are reconciled.

### Loader command

```
node scripts/load-content-to-store.mjs [--dry-run]
```

Every write is an idempotent UPSERT (re-running is always safe — see
`tests/content_store.test.mjs`'s idempotency test). `CUSTODY_CASE_DB` selects the target store
exactly as every other script/tool in this plugin does. `--dry-run` prints what it would write
and touches nothing.

### Fallback rule

Every store-first reader (`core.ts`'s `loadLedgerRecords()`/`loadDirectoryRecords()`,
`court-language.ts`'s `loadLexicon()`/`loadPhrasebanks()`, `survival-guide.ts`'s
`loadContextPack()`/`loadTemplate()`) tries `getReference()`/`getSources()` FIRST and falls back
to the ORIGINAL direct file read whenever the store is unavailable or has no row for that key —
this is exact, not approximate: with an empty store (the `mem://` test situation, or a fresh
plugin install before the loader has ever run) every one of these functions returns precisely
what it always returned. `tests/content_store.test.mjs` proves this by comparing the fallback
output against a direct read of the same file, byte-for-byte / field-for-field.

`auditSources()`, `searchRecords()`, `reviewCourtLanguage()`, `loadLexicon()`,
`buildSurvivalGuide()`, and `loadContextPack()` are all `async` now (they weren't before
2026-09-08) because their store-first reads are. `listSurvivalGuideEvents()` stays a plain
filesystem listing on purpose — `buildServer()` calls it synchronously to fix the `zod` enum of
valid `survival_guide` event ids at tool-registration time, which must not depend on an async
store round trip.

### Bundling: local modules must stay real runtime imports

`build.mjs` bundles each `src/*.ts` entry point independently (`esbuild`, `bundle: true`).
Before this change that was harmless — every entry's local imports were self-contained. Now that
`core.ts`/`court-language.ts`/`survival-guide.ts` transitively depend on `store.ts` (via
`content-store.ts`), naively bundling each entry point would give EACH ONE its own private,
independent copy of `store.ts`'s module-level `getStore()` cache — invisible to each other. That
is silently wrong for the embedded engines (two independent opens of the same `mem://` are two
DIFFERENT, mutually invisible in-memory databases; two independent opens of the same
`rocksdb://` path from one process can lock-conflict) even though it would go unnoticed against
a shared remote `ws://` store. `build.mjs`'s `localExternalsPlugin()` fixes this by leaving every
cross-cutting local import (`./store.js`, `./store-tools.js`, `./core.js`,
`./court-language.js`, `./survival-guide.js`, `./content-store.js`) as a real runtime import to
the ONE compiled sibling file in `dist/`, scoped so it only matches an import written in one of
OUR OWN `src/*.ts` files — `zod` v4 has its own internal `./core.js` relative import
(`node_modules/zod/v4/core/errors.js`) that must NOT be redirected at our `dist/core.js`.

## Backup / export

`case_export` (tool) or `node scripts/case-cli.mjs export [path]` writes every table and edge
to one JSON file (default `~/.config/family-court-toolkit/exports/<timestamp>.json`), or, with
`format: "platform"`, one NDJSON-per-table platform bundle (see above). `case_import` /
`case-cli.mjs import <path>` loads that snapshot shape (`{tables, edges}`), a Vincent-style
schema, or a case-extract/v1 file/directory back in — all three paths run the same table/edge
UPSERT logic, so import is safe to re-run (idempotent by record id / `extract_id`).

## Testing

`tests/store.test.mjs` (30 tests as of 2026-09-07, up from 12 before the case-store register
extension) runs entirely against `mem://` (plus one real `rocksdb://` regression test for the
Windows absolute-path fix, in a real OS temp directory). `after()` closes every store opened
during the run and force-exits the process — verified live: the embedded engine's native
connection otherwise keeps `node --test`'s process alive indefinitely even though every
assertion already finished in under a second.

`npm test` runs the whole suite (84 tests across all `tests/*.test.mjs` files as of
2026-09-08, up from 73 — the 11 added are `tests/content_store.test.mjs`, covering the
"content lives in the SurrealDB store" fallback/load/store-backed-read contract above; 72/73
passed before this addition and 83/84 pass now — the one pre-existing failure,
`court_language_review: self-consistency — every EXAMPLES.md 'After' text never trips the
lexicon`, is unrelated to this change and untouched). A sequential MCP client
probe (a scratch `rocksdb://` path via
`CUSTODY_CASE_DB`, never the owner's real case store; each `callTool` awaited to completion
before the next — batched stdin writes race and must never be used to judge a write→read pair)
additionally proved `case_status` set→get, `case_memo` put→`case_search` hit,
`case_put(court_event)`→`case_timeline({mode:"court", upcoming:true})`, and
`case_export({format:"platform"})` writing real files, end-to-end through the built
`dist/server.js`.
