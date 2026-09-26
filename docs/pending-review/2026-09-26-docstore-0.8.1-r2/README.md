---
tags: [docstore, propria, receipt, contextforge, tailscale, retraction-guard]
---

# Docstore 0.8.1-r2/r3 — search, sync, svc:docstore-api, and the retraction guard

> _Byline: Claude Code · Opus 5.5 · 2026-09-26. Subagent `docstore-fix`, dispatched by the parent session. Owner order relayed mid-task: "make lost docs impossible to miss."_

**Result:**
- Search through `ctl` answers long questions.
- The index is synced to the current desktop docs.
- `svc:docstore-api` serves the live API.
- Neither a sync nor an index run can retract a document unless someone names it.

The running Docstore server (Coolify service `propria-docstore-0-8-1`) is built from `/data/propria/releases/docstore-0.8.1` on ovh-files. That tree is in **no git repository**. This directory is its record:
- the apply scripts;
- the before-copies (`before/`, git-normalized to LF; original sha256 values below);
- the probes.

This follows the r1 precedent (`../2026-09-20-docstore-0.8.1-r1/`).

## Root causes

| Symptom | Cause | Fix |
|---|---|---|
| `coco_docstore_search` → "Docstore unavailable or invalid response" | Search was never down. A long question sent `/recall` into its any-term keyword fallback. That fallback ran one BM25 query per term, one after another, repeated terms included: 16 queries, 7.7 s warm and 30.25 s on the first cold call. The ctl→API client had a fixed 30 s budget, and every failure was reported as the same bare "unavailable". | **r2** `recall.py`: terms are de-duplicated and queried concurrently on the one multiplexed connection (1.7 s measured for the same 16 terms), and the vector leg overlaps the keyword leg. **r2** `server.py`: the budget is `DOCSTORE_API_TIMEOUT_S` (default 55 s, under ContextForge's 60 s tool timeout), and a timeout, HTTP status or unreachable API is named. Upstream bodies are still never echoed. |
| Sync stale since 2026-09-21 01:51 | 1) 0.8.1 has no automatic sync. The source mirror (`/data/propria/releases/docstore-0.8.1/sources`) changes only through `client.py sync`, and nothing had run it since 2026-09-20. 2) `sync()` hard-coded the pre-2026-09-19 folders (`Probata/probata/docs` …, now under `modules/`) and ignored the registry's exclusions. Advocatio's `planning/original-context` (files over 1 MiB) would have aborted it anyway. 3) The hosted ctl refused any MCP request body over 4 MiB with HTTP 413; that is the MCP SDK default. A full manifest is about 12 MB, so no client sync could ever reach the mirror. The 09-20 mirror was copied server-side. | **Client (0.8.2-sync-r2)** `patch_client_sync.py`: folders and exclusions come from `Propria/docs/docstore-source-registry.json`. **r2** `hosted.py`: the ctl accepts `DOCSTORE_MCP_MAX_BODY_BYTES` (default 40 MiB, the worker API's own upload bound). Then sync, then `docstore_index_full`. |
| `svc:docstore-api` 502 | On 2026-09-20, 0.8.1 stopped the old worker app, which served 8474 and 8072. Its own API listened only on loopback inside the new container. The owner deleted the old app on 2026-09-25. No decision retired the external API; see Evidence below. | **r2** `service.py`: `DOCSTORE_API_HOST` (loopback by default). The service compose sets `0.0.0.0` and publishes `100.91.190.107:8072:8000`, the canonical Docstore API port. `svc:docstore-api` was re-pointed from 8474 to 8072. |
| ADR enrichment failures 0085, 0016 (ValueError) | `nvidia/nemotron-3.5-lightning-30b-a3b` sometimes returns `statements` as one string instead of an array. `knowledge.extract()` retried only unparseable or truncated JSON, so a parseable reply of the wrong shape failed the document on the first try. An overloaded provider's 429/5xx failed it with no retry at all: 88 `HTTPStatusError` in run `6f397d56`. | **r4** `knowledge.py`: shape validation moved inside the retry loop, and 429/5xx get one retry after 5 s. A release test covers the shape retry. 0085 enriched in run `6206106a`, 0016 in run `e2b1ddf5`. |

## Evidence that `svc:docstore-api` is meant to exist

- Owner, 2026-09-10: "make sure there's an API exposed so I can put a front end on this".
- Owner decision, 2026-09-12 (port families): the Docstore API is canonical port 8072, and clients use only `https://docstore-api.tilapia-skilift.ts.net`. Recorded in `reference/2026-09-12-service-port-and-tailnet-addressing-standard.md` and `deploy/service-port-registry.json`.
- The 0.8.1 request (Codex, 2026-09-20) does not mention the external API. The loopback-only binding was an implementation choice. The 2026-09-20 TODO recorded "the API port 8072 is no longer published" as a defect.

## Retraction guard (r3 server, 0.8.2-sync-r3 client)

The owner's order followed the discovery of 24 Probata docs missing from the desktop. The first sync plan today would have retracted them.

- **Client `sync`:**
  - It never retracts on its own.
  - A document the mirror holds but the desktop lacks is restored from the mirror by default (`docstore_source_read`). The copy is hash-checked against the plan's `retracted_hashes` and again after writing.
  - `--retract PROJECT/PATH` names one document to retract.
  - `--hold PATTERN` / `--hold-file FILE` keep the mirror's version.
  - Every uploaded doc that git does not track is flagged, one `FLAG` line per file on stderr and as a count + list in the result.
  - Code: `patch_client_guard.py`, applied to four copies: Claude local-plugins and its 0.8.2 cache, Codex local-marketplace and its 0.8.2 cache. The same script updates the index skill and the Claude CHANGELOG.
- **Server `docstore_source_apply`:** refuses any retraction not named in `retract`. The plan lists `retracted_hashes`, and the new read-only operation `docstore_source_read` returns exact mirror copies.
- **Server index runs:** `retire_unexpected_projection` used to retract every stored document absent from the source, up to 1000, in every run. Now it retracts only the paths named in `docstore_index_full(retract_paths=…)`. Every other absent document is held and stays active. The run records `retraction_held` and a reason, and its CDC check stays degraded until someone decides.
- Code: `apply_r3.py`. The two retraction tests in `tests/test_release.py` follow the new contract.

## Live proof (2026-09-26, UTC)

- **Search:**
  - The failing question ("Docstore API deployment: which Coolify service or app serves the Docstore API, its port, and svc:docstore-api endpoint after the 0.8 release") returned 5 results through ctl/ContextForge in 6.1 s, on the first call to a freshly restarted container. Before the fix: 30.25 s and "unavailable".
  - `recall_compare.py`, same queries before and after: 11.0 → 5.5 s, 6.8 → 2.3 s and 7.5 → 5.3 s. Reranked top 5 identical in all four queries.
  - 298 control tests pass on the r2 image.
  - 314 control + release tests pass on the r3 image. The release tests ran with `surrealdb-embedded`, in a throwaway container.
- **svc:docstore-api:**
  - `tailscale serve` now proxies to `http://100.91.190.107:8072`.
  - `https://docstore-api.tilapia-skilift.ts.net/health` returns 200 `ok:true`.
  - `/stats` returns 401 without a token. With the container's token it answers through the service DNS.
- **Sync and index:**
  - The mirror took generations `2a00ad9b…` (22:23; 153 changed), `b5d49a5e…` (23:07; 43 changed, the 2026-09-20 flatten recovery) and `b394162f…` (23:19; 1 changed).
  - Run `6f397d56c50b48938335e2bb6769b647` ended 23:10:10:
    - 791 sources; CDC attribution **verified**, 791/791, 0 mismatch / missing / unexpected;
    - ADR projections 95/95;
    - 932 documents, 14,042 chunks, HNSW ready, 0 orphan chunks;
    - retirement: held 0, retired 0.
  - Status `degraded` with reason "provider enrichment remains pending". The r1 health rule keeps `ok:true`.
  - Earlier, run `54e7c905…` was cancelled during enrichment after its ingest had completed. 89 enrichments made before the cancel were kept.
- **The 62 docs of the 2026-09-20 flatten:**
  - Every one is indexed with the current desktop content: 59 by exact sha256; 3 (`BUILD_PLAN.md`, `PROJECT_CANON.md`, the 09-06 naming digest) equal after the store's non-BMP folding.
- **Retraction guard, live:**
  - A scratch Propria copy missing one doc gave a plan listing it with its mirror hash.
  - `docstore_source_apply` without naming it was refused (HTTP 409).
  - `docstore_source_read` returned the exact copy, and its hash matched the plan.
  - `client.py sync --apply` restored it (`FLAG restored from mirror (hash-verified)`), byte-identical to the canonical file. It then applied with 0 retractions.
- **Final state:**
  - Image `propria-docstore:0.8.1-r4`, 315 tests passing.
  - Run `e2b1ddf51fec4900b88e49ca80a85d06` ended 2026-09-26 23:34:59 UTC: 792 sources, CDC verified 792/792, 933 documents, 14,057 chunks, retirement held 0.
  - `docstore_health` through ctl: `ok:true`, `enrichment_pending: 2`. The two still pending are the propria WS00 reconciliation docs, which fail on the provider's reply shape.
- **Enrichment earlier today:**
  - `nvidia/nemotron-3.5-lightning-30b-a3b` was overloaded on NIM: a trivial prompt took 24.3 s, and the two ADRs hit a 180 s ReadTimeout twice.
  - The 23:10 run recorded 88 `HTTPStatusError` failures. They stay queued in `pending-enrichment.json`, 0085 and 0016 among them.
  - `moonshotai/kimi-k3` answered the same ping in 2.8 s.

## Open for the owner (lettered options; default first)

1. **Where the 0.8.x source lives.** The server tree (`/data/propria/releases/docstore-0.8.1`) and the 0.8.2 client are in no git repository; r1–r3 are patch receipts.
   - (A) Import the 0.8.1-r3 tree into this repository as the Docstore source, replacing the 0.6.5-era `plugins/docstore` and `scripts/docstore`, and build the image from git.
   - (B) Give it its own repository.
   - (C) Keep patch receipts.
2. **Enrichment model.**
   - (A) Point `DOCSTORE_LLM_MODEL` at `moonshotai/kimi-k3`, the primary since 2026-09-25. Its quirks need handling first: it takes the `thinking` kwarg, and empty or junk content must count as a failure and be retried.
   - (B) Keep nemotron-3.5-lightning and let later runs drain the queue.
3. **No automatic sync.** The mirror changes only when someone runs `client.py sync`, and indexing starts only on request.
   - (A) A Coolify scheduled `docstore_index_full`. It refreshes only what the mirror already holds.
   - (B) The VPS pulls committed docs from git. This misses desktop-only docs, which the new untracked flag lists.
   - (C) Keep the manual sync.
4. **ContextForge `ctl08` upstream is a raw IP** (`http://100.91.190.107:8175/mcp`). By the service-DNS rule it wants a Tailscale Service. The name needs your approval.
5. **Legacy `probata-docstore-control` (:8172).** Its API token is empty and its API URL is a raw IP: retire it or repoint it.
6. **FL-MCP and Vestigia docs are still outside the five server roots** (2026-09-24 TODO item).

## Files

| File | What |
|---|---|
| `apply_r2.py` | recall concurrency, named API failures, `DOCSTORE_API_HOST`, 40 MiB MCP body |
| `apply_r3.py` | retraction guard (server) + test updates |
| `apply_r4.py` | enrichment retries a mis-shaped reply and an overloaded provider once, plus a release test |
| `coolify_service_r2.py` | image tag, `DOCSTORE_API_HOST`, port 8072 in the service's stored compose (values never printed) |
| `patch_client_sync.py`, `patch_client_guard.py` | client repairs (four installed copies) |
| `recall_timing.py`, `recall_compare.py`, `enrichment_probe.py`, `run_progress.py` | read-only probes run inside the container |
| `recover_flatten_edits.py`, `hold_restored_copies.py` | the 2026-09-20 flatten recovery cross-check and the move of today's restored copies |
| `index_hashes.py`, `provider_ping.py`, `guard_live_test.py` | stored-hash proof, provider liveness, live negative test of the server guard (read-only) |
| `before/` | pre-change copies; sha256 of the originals listed below |

Original sha256 values, CRLF as found:
- `recall.py` `506ab4d28de63db2…`
- `server.py` `ffe70a599a355fd7…`
- `service.py` `ddc66506bbf16f8e…`
- `hosted.py` `160614fcf2f7bfd0…`
- client `a7aa7d250b2a0bf6…`
