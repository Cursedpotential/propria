---
tags: [docstore, propria, receipt, memory, error-reporting, contextforge]
---

# Docstore 0.8.1-r5 — memory writes work, errors say why, and the write schema is published

> _Byline: Claude Code · Opus 5.5 · 2026-09-27. Subagent `docstore-memory-fix`, dispatched by the parent session "portal cut over"._
> _Owner, 10:01 EDT: "I want it fixed, and then I want the fact that it doesn't report the error also fixed. And then I want the fact that we have no idea how to write to it also fixed in the skill."_

**Result:**
- `docstore_memory_remember` through ctl writes.
- A second, similar write is refused with the conflicting id.
- Supersession works.
- Every Docstore error names its kind and carries the server's reason.
- `docstore_capabilities(operation="docstore_memory_remember")` returns the real field list.

This follows the r1–r4 precedent (`../2026-09-26-docstore-0.8.1-r2/`). The running server is built from `/data/propria/releases/docstore-0.8.1` on ovh-files, a tree in no git repository. The seven files r5 touches were byte-identical to this repository's copies apart from line endings, checked by LF-normalized sha256 on 2026-09-27. So r5 edits the repository copies, and `apply_r5.py` installs them whole, refusing any target that has drifted.

## Root causes

The symptom was `Docstore unavailable: the API answered HTTP 409; no filesystem fallback` for every payload. Five defects stacked up behind it:

| # | Layer | Defect | Fix |
|---|---|---|---|
| 1 | API `remote_memory.py` | Required `{kind, claim, detail, evidence, agent}` and raised a bare `ValueError('Memory fields required or payload oversized')`. Both payloads the parent tried (`content/tags/source`, `text/category`) lacked those fields. | Explicit validation. It reports every problem at once: unknown fields, missing fields, bad kind, bad scope, bad confidence. `detail` is optional, as it is in the table. |
| 2 | API `release_api.invoke` | Every `ValueError` became **HTTP 409 Conflict**, so a validation failure was labelled a conflict. | `MemoryFailure(status, detail)`: 422 invalid, 409 near-duplicate or exact claim exists, 404 unknown action, 502/503 memory service. Other operations keep their existing mapping; their reasons now reach the caller (row 5). |
| 3 | API + ctl defaults | Remember and recall defaulted to scope `probata`. The store's scope root moved to `propria` on 2026-09-19 (`scripts/docstore/schema/2026-09-19-memory-root-propria.surql`), and `ASSERT ^propria(/…)*$` rejects `probata`. Recall therefore matched nothing: the parent's `results: []`. | Default `propria`, validated in both layers. `claude/federation.py` had the same stale scope. |
| 4 | Memory DB `fn::remember` / `fn::supersede_memory` | (a) The vector duplicate guard took the 5 nearest rows **with no distance cutoff**. Once a scope held any embedded row, every write was refused as a near-duplicate; the unrelated owner rule's neighbours sat at 0.216–0.297. (b) `fn::remember` called `fn::supersede_memory` with 3 arguments and the function took 2. Every supersede failed: `Incorrect arguments for function fn::supersede_memory(). The function expects 2 arguments.` | `scripts/docstore/schema/2026-09-27-memory-remember-guard.surql`: cosine cutoff 0.20; supersede takes `$reason`, THROWs on an unknown or inactive target, accepts a string id, and logs the reason to `decision_log`. |
| 5 | ctl `server.request`, `governance.query` | Any HTTP error became "Docstore unavailable: the API answered HTTP n". The body was never read (0.8.1-r2 chose "upstream bodies are never echoed"). The native path collapsed every statement failure into "statement failed". | The API's JSON `detail` passes through, token redacted and bounded to 4000 characters. Messages are worded by kind (invalid / conflict / credentials / not found / too large / unavailable / API error). "Unavailable" is kept for 502/503/504, timeouts, unreachable APIs and redirects. Non-JSON bodies (proxy pages, tracebacks) are still not echoed. Native statement failures carry the database's reason, credential redacted, 1000 characters. |

The schema gap: `docstore_memory_remember(payload: dict)` published `payload: object, additionalProperties: true`. It now takes a typed `MemoryWrite` model (`extra="forbid"`), and its description states the duplicate and supersession rules and the error meanings.

### The 0.20 cutoff

The cutoff is a calibration, not a law. It was measured on 2026-09-27 against the live store with `nvidia/nemotron-3-embed-1b`.

- **Reworded duplicates:** nearest row at 0.061, 0.090, 0.113 and 0.184. The 0.184 case was a heavy rewording of the owner rule.
- **New, unrelated claims:** nearest row at 0.216, 0.313 and 0.325.

The first r5 build used 0.15. It let the 0.184 paraphrase through, so the cutoff was raised to 0.20 within the hour, and that test row was deleted. The gap between the two groups is narrow. Revisit the cutoff if false conflicts or missed duplicates show up.

## Deployment

- DB migration applied to `probata_memory/memory` (surreal-case, `:8471`) through the memory MCP: `has_errors: false`. It was first run inside a transaction and rolled back, with every scenario checked there.
- `apply_r5.py` applied to `/data/propria/releases/docstore-0.8.1` (backups `*.bak-20260927T141827Z-r5`, `*.bak-20260927T142112Z-r5`).
- Image `propria-docstore:0.8.1-r5` (`sha256:992b6fb3…`): **334 passed, 3 skipped** (r4: 315) in a throwaway container, release and control suites.
- Coolify service `o8obobz576je1fbyygnywl83`: image tag patched through the API (`coolify_service_r2.py --image propria-docstore:0.8.1-r5 --apply`), then `POST /api/v1/deploy {"uuid": …}` twice (200 each; the second picked up the 0.20 rebuild). The container has run image `992b6fb3…` since 14:22:01 UTC.

## Live proof through ctl (ContextForge `ctl08`), 2026-09-27 UTC

| Step | Call | Result |
|---|---|---|
| Before | remember, any payload | `Docstore unavailable: the API answered HTTP 409; no filesystem fallback` |
| Schema | `docstore_capabilities(operation="docstore_memory_remember")` | full field list: required `kind, claim, evidence, agent`; enums, patterns, descriptions |
| Invalid (ctl) | `{content, tags, scope, source, kind:"note"}` | `7 validation errors …` naming each field: kind enum, three missing, three extra |
| Invalid (database) | valid payload, `supersede: "memory:doesnotexist"` | `Docstore rejected the request as invalid (HTTP 422): {"reason": "the memory database rejected the write", "database": "Error (Thrown): An error occurred: supersede target memory:doesnotexist does not exist"}` |
| Write | the owner rule below | `{"outcome":"written","id":"memory:z29uynwp9gdpj34m607t","scope":"propria"}` |
| Read back | `docstore_memory_recall` ("search with the owner's tools instead of grep…") | top hit `memory:z29uynwp9gdpj34m607t` |
| Duplicate | paraphrase of the rule | `Docstore refused the request as a conflict (HTTP 409): {"reason": "near-duplicate…", "conflicts": [{"id": "memory:z29uynwp9gdpj34m607t", "dist": 0.1838…}], "next": "pass supersede…"}` |
| Supersede | probe row, then `supersede` with a reworded claim | `{"outcome":"superseded","id":"memory:jbs3dq52…","superseded":"memory:qb9huqll…"}`; old row `superseded`, `->supersedes->` edge present, `decision_log` has the reason |

The kept row is the owner's rule, `memory:z29uynwp9gdpj34m607t`: "Owner rule 2026-09-27: search with the owner's tools (read-memories/DuckDB, memsearch, Docstore, ccc, claude-context, smart-explore), not grep; check all memory lanes before a decision (auto-memory note memory-tools-catalog)."

**Test data purged:**
- the paraphrase row;
- both supersession probe rows;
- their `supersedes` edge and their two `decision_log` rows.

The store holds 17 rows: the 16 from before plus the owner rule.

## Seen, not changed

- **The container reports `unhealthy`**, as it already did on r4. The API and store are up, but the latest index sync, run `5356f93f8018493d9849a81f381dd848` (2026-09-27 01:02 UTC, 795 sources), is `failed`. The r1 health rule marks that as not OK. It needs its own look.
- **Other operations still map a plain `ValueError` to 409** in `release_api.invoke` (upgrade, adr, knowledge, sources). Some of those are real conflicts (plan changed, worker busy), others are validation. Their reasons now reach the caller, but the status needs a per-operation audit to split 409 from 422.
- **The root `plugins/docstore/` in this monorepo is a preserved 0.6.3 snapshot** from the 2026-09-20 move. It still documents raw `fn::remember` calls with a `probata` scope. The canonical plugin is `modules/Probata/probata/plugins/docstore/claude` (0.8.3). Options for the owner:
  - (A) quarantine the root snapshot;
  - (B) replace it with the canonical tree;
  - (C) leave it.

## Files

| File | What |
|---|---|
| `apply_r5.py` | Installs the seven files and the release test after checking their pre-r5 hashes; idempotent, dry run by default. |
| `test_remote_memory.py` | Release test, installed as `tests/test_remote_memory.py`: 422 field listing, 409 conflicts, supersession, database error classes, recall scope. |
| `../../../scripts/docstore/schema/2026-09-27-memory-remember-guard.surql` | The memory-function migration. |
