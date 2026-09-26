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
| ADR enrichment failures 0085, 0016 (ValueError) | The provider is intermittent. `nvidia/nemotron-3.5-lightning-30b-a3b` returned invalid or truncated JSON on both attempts in the 09-21 run. Reproduced today (`enrichment_probe.py`): both documents extract cleanly on the first attempt (about 8 s each), while one retry hung to a 180 s ReadTimeout. The documents and the code are not at fault. | Retried by the next run: `pending-enrichment.json` carries them. |

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

## Files

| File | What |
|---|---|
| `apply_r2.py` | recall concurrency, named API failures, `DOCSTORE_API_HOST`, 40 MiB MCP body |
| `apply_r3.py` | retraction guard (server) + test updates |
| `coolify_service_r2.py` | image tag, `DOCSTORE_API_HOST`, port 8072 in the service's stored compose (values never printed) |
| `patch_client_sync.py`, `patch_client_guard.py` | client repairs (four installed copies) |
| `recall_timing.py`, `recall_compare.py`, `enrichment_probe.py`, `run_progress.py` | read-only probes run inside the container |
| `recover_flatten_edits.py`, `hold_restored_copies.py` | the 2026-09-20 flatten recovery cross-check and the move of today's restored copies |
| `before/` | pre-change copies; sha256 of the originals listed below |

Original sha256 values, CRLF as found:
- `recall.py` `506ab4d28de63db2…`
- `server.py` `ffe70a599a355fd7…`
- `service.py` `ddc66506bbf16f8e…`
- `hosted.py` `160614fcf2f7bfd0…`
- client `a7aa7d250b2a0bf6…`
