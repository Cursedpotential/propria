---
title: Vestigia (TraceIQ) continuation plan
status: proposed
date: 2026-10-02
owner_order: "2026-09-20 13:56 — reconcile and plan the traceiq continuation"
tags: [vestigia, traceiq, geodata, plan, custody-case, propria]
---

# Vestigia (TraceIQ) — continuation plan

> _Byline: Claude Code · Opus 5.5 · 2026-10-02. Proposed, not approved: nothing here is locked,
> and every choice below is a lettered option with a default for the owner to confirm or change.
> Sources: a static read of this folder (no data files opened, no scripts run), the Claude session
> logs from 2026-07-24 onward, Docstore (`vestigia`, `docs`, `infra` domains), Probata
> `docs/NAMING.md`, and the 2026-09-20 and 2026-09-30 TODO files. The Codex and memsearch lanes
> were not swept (a hook blocks scripted sweeps of them)._

## Why it exists

Vestigia turns Google location-history exports into a forensic geodata store for the custody case.
The owner's words (2026-07-25): comparing the expected custody schedule with where the devices
actually were is "arguably the point of the whole tool". He wants several geocoding providers so he
can see "where they agree or disagree", and rounding so lookups do not cost "hundreds of dollars".
The 2026-07-24 rebuild exists because `home_base` kept getting lost across AI-session handoffs.

## Where it stands (static read, 2026-10-02)

Built in two days (2026-07-24/25), untouched for feature work since 2026-08-01.

| Area | State |
|---|---|
| Database | Postgres + PostGIS on the VPS (ADR-0004). 9 migrations, 6 transformations, 7 schemas (`raw`, `working`, `geo`, `ref`, `ops`, `src`, `analysis`). |
| Ingest | `ops/ingest_export.py`: one export ingested, 3 chunks, 41,963 records. `ops/INGEST_VALIDATION_REPORT.md` (2026-07-24, live DB): hashes match, nothing unaccounted. |
| Analysis | Wave 1 (overnight, home base, anomaly suite) exists as views. Wave 2 (schedule verification) waits on the owner's reference data (ADR-0012). |
| Geocoding | Provider subsystem built (ADR-0011/0014). `PROVIDER_TEST_REPORT.md` (2026-07-26, live HTTP): 3 providers pass, 2 have no templates. No enrichment run yet; ADR-0016 plans a fresh reverse geocode for about 17,467 location keys. |
| UI | Next.js 16 + deck.gl/MapLibre explore view over live data, with a known-place editor. Analytics, Tables, Export and Config tabs are placeholders. Chat is a mock. No auth. It only ever ran as a dev server on the desktop. |
| Reports | Five Evidence.dev pages in `reports/`, never rendered. |
| Ops | No migration runner, job runner (ADR-0010), backup script (ADR-0004) or standalone API (ADR-0001). No Dockerfile, compose file or Coolify app. |

### Problems found while reading

- **Credentials in tracked source.** `ops/ingest_export.py`, `ops/validate.py`,
  `ops/validate_ingest.py` and `ops/test_load_raw.py` hard-code the database host and password, and
  `ui/BUILD_BRIEF_PHASE2.md` embeds a full connection string (`reports/README.md` names the host
  only). They were removed from the current files on 2026-10-02 (Phase 0) but remain in the
  Propria monorepo history, so the password stays exposed until Vestigia moves off it (V-3). Docstore flag `faa916eb` (2026-09-24) also records six gitleaks
  `generic-api-key` findings in this history. No record says they were reviewed.
- **Which database server (settled 2026-10-02, live read).** The scripts pointed at ovh-data,
  which has been offline for about six weeks. The database is `traceiq` on `probata-db`
  (ovh-files `100.91.190.107:5432`), PostgreSQL 18.1 with PostGIS 3.6.4 and pg_duckdb 1.1.0.
  It holds one export and one chunk: 20,161 raw records = 20,160 events + 1 exception, 39,974
  waypoints, 17,467 location keys, 3 known places, 0 home-base rows, t001–t006 recorded. The July
  validation's 41,963 records were three chunks, of which chunk 3 mirrored chunk 2 byte-for-byte
  in content; chunk 1 (1,641 records, Mar–Jun 2024) is not in the live database, and whether its
  records all fall inside the full export is not yet checked.
- **An off-the-shelf neighbour.** Coolify already runs `dawarich`, a self-hosted location-history
  app with its own PostGIS. Nothing in the record compares it with Vestigia.
- **Data correctness before exhibits.** ADR-0016 says dual-device path conflation must be fixed
  before exploded paths are used as exhibits. The owner (07-25): "There absolutely is multi device
  conflation". The ingest hashes re-serialized JSON, not the original bytes that ADR-0006 asks for.
- **Doc drift.** `AGENTS.md`, `AGENT_MEMORY.md`, the parent folder's `AGENTS.md`/`AGENT_MEMORY.md`,
  `docs/adr/README.md` (lists 0001–0014 only), `ui/README.md` and `ops/FALLBACK_RESUME.md` still
  describe an independent repository at `Projects/traceIQ`, a static-export mock UI, and paths that
  no longer exist. `ops/validate.py` writes to an old absolute path.
- **Dangling gitlinks.** Five (Tether, TetherPro, notebooklm-mcp-source, notebooklm-mcp-target,
  pandoc-lua-filters) break recursive clones (monorepo plan, open item).

## Owner direction, 2026-10-02 08:08

> "i want to mirror the data from pg to surreal once we get it right. all analysis will be in
> surreal pg is only SOT data holder"

- PostgreSQL holds the source-of-truth data only: raw, working, geo and ref.
- All analysis runs in SurrealDB, on a mirror of that data.
- The mirror starts once the data is right (Phase 1 done), not before.
- This changes accepted ADRs: 0002 (pg_duckdb as the analysis engine), 0012 (analysis waves as
  Postgres views) and 0013 (Evidence.dev reports read Postgres). The existing `analysis` schema
  (t002–t006, the `timeline_exploded` materialized view) becomes the reference to port, and comes
  out of Postgres once the Surreal version gives the same answers. An ADR records this through the
  Docstore once V-7 and V-8 are answered.

## Decisions for the owner

Each has a default. Saying "defaults" approves all of them.

**V-1. What "continuation" means.**
- **A (default):** finish Vestigia as its own app first, then fold its views into the Probata
  Workbench. This is the owner's 2026-07-28 order: "stand up the trace IQ correctly finish that and
  then begin merging that into the larger MCP UI".
- B: skip the standalone app and build the geo views straight into the Workbench now.
- C: park it until the Case Bible corpus goal is met (the 09-30 plan puts the Case Bible first).
- D: compare with Dawarich first (one sitting: what it shows from the same export), and keep
  only the custom parts it cannot do: provenance, provider agreement, schedule verification.

**V-2. Where the UI and API run.** The database is already on a VPS. The UI has only ever run on
the desktop, which the hosting rule forbids.
- **A (default):** one Coolify app on ovh-app (Next.js serving the UI and its API routes), behind
  Authentik and a `svc:vestigia` tailnet service, built from the monorepo with this folder as base
  directory. No separate API service until a second client needs one (ADR-0001 stays the direction,
  not a blocker).
- B: a separate API service plus the UI, as ADR-0001 describes, now.

**V-3. The credentials in history.** The `ai` password is two letters, and `ai` is probably the
shared user for the other databases on `probata-db` (platform, temporal, contextforge, infisical,
archive), so changing it could break those apps.
- **A (default):** give Vestigia its own role (`vestigia`, a long generated password, rights on
  the `traceiq` database only), switch `~/.secrets/traceiq-db.env` and later the Coolify env to it,
  and leave history as is. Whether to strengthen the shared `ai` password is a separate question
  for the lane that owns `probata-db`.
- B: also rewrite monorepo history to remove the values. This is destructive, affects every
  session and all 35 Coolify apps, and only helps if the repository is ever made public.

**V-4. Backups (ADR-0004 said a local `pg_dump` to `E:\TraceIQ_Backups`).**
- **A (default):** a scheduled `pg_dump` on the VPS, written to B2 under the case vault, with an
  `ops.backup` row and a verified `pg_restore --list`. This matches "R2 is being retired" (09-28)
  and keeps the job off the desktop.
- B: keep the local desktop copy as ADR-0004 says.

**V-5. Third geocoding provider and the spend cap.** ADR-0016 says the `geodata` slot is not a
real vendor.
- **A (default):** Mapbox as the third source, plus a Google template, with a hard cap per run and a
  dry-run that prints the call count and cost before anything is sent.
- B: run with the three that pass today (geoapify, HERE, Radar) and add a fourth later.

**V-7. Which SurrealDB holds the mirror.** Live instances (all 3.2.4 on ovh-files): `surreal-docs`
(Docstore), `surreal-case` (case store and agent memory), `surreal-intake` (Intake graph).
- **A (default):** a new `surreal-vestigia` instance, declared in Coolify with bind mounts and its
  own credentials, the way `surreal-intake` was isolated (owner 2026-09-12: shared technology is
  not a shared runtime).
- B: a `vestigia` namespace inside `surreal-case`, next to the case store it will feed.

**V-8. How the mirror runs.**
- **A (default):** a full-rebuild job on the VPS (ADR-0010's model): after each Postgres milestone
  it reads the source tables and rewrites the Surreal side, then checks counts and hashes on both
  sides. The data changes rarely (one export so far), so this is the simplest thing that works.
- B: continuous change capture (an outbox table plus a Temporal workflow, the ADR-0052 pattern),
  for when new exports arrive often.

**V-6. Reference data only the owner can give.** No default: these are facts.
1. Confirm or correct the home-base clusters (unblocks overnight v2).
2. Acquisition method and basis for the ingested export (`OWNER TO FILL`).
3. Which account belongs to which person.
4. Custody order times and the claims ledger (unblocks wave 2, schedule verification).

## Phases

Each phase ends with a live check on the real database; any test rows it writes are purged.

### Phase 0 — make it safe and true (no owner decision needed except V-3)

0. **Done 2026-10-02:** the module's `.gitignore` boundary that silently dropped every new file in
   `traceiq-rebuild/` is gone (the same fix for five other folders across the monorepo).
1. **Done 2026-10-02, except the rotation:** every ops script reads `TRACEIQ_DSN_KV` through
   `ops/db_env.py` and stops with a clear message when it is missing; no password is left in
   tracked files; `.env.example` names both variables; the value lives in
   `~/.secrets/traceiq-db.env`; `ops/validate.py` writes its report inside the repo. Verified
   live (read-only counts through the new path). The dedicated role per V-3 is still to do.
2. Re-conform the drifted docs (`AGENTS.md`, `AGENT_MEMORY.md`, parent router, `docs/adr/README.md`
   adding 0015/0016, `ui/README.md`) and delete `ops/FALLBACK_RESUME.md`'s stale content.
3. **Server done 2026-10-02** (`probata-db`, recorded in `docs/SCHEMA.md` and `reports/README.md`).
   Still open: check the live schema object by object against migrations 0001–0009, and check
   chunk 1's records against the full export.
4. Resolve the dangling gitlinks and review the six gitleaks findings.

### Phase 1 — data you can stand behind

1. Backups per V-4, with a restore check.
2. A small migration runner: applies `db/migrations` and `db/transformations` in order and records
   a checksum per file in `ops.transformation`. Off-the-shelf first (e.g. `dbmate`) if it fits.
3. Byte-exact content hashing on ingest (ADR-0006), checked against the existing rows.
4. Dual-device conflation fixed and verified on real paths (ADR-0016's exhibit gate).
5. Geocoding enrichment per V-5: dry-run, the owner sees the cost, then the run. Output: the
   agreement matrix the owner asked for (where providers agree and disagree, per location key).

### Phase 2 — mirror to SurrealDB, analysis there

1. The mirror per V-7 and V-8, with a count-and-hash check on both sides after every run.
2. Port wave 1 (overnight, home base, anomaly suite) from the Postgres `analysis` views to
   SurrealQL, and check that both give the same answers on the real data before the Postgres
   views come out.
3. Wave 1 confirmed with the owner's home-base answers (V-6.1).
4. Wave 2 in Surreal: expected schedule versus device location, from the custody order times
   (V-6.4).
5. Anomaly and route views reviewed with the owner.

### Phase 3 — hosted and usable

1. Dockerfile and Coolify app per V-2, Authentik, tailnet service, portal tile.
2. The placeholder tabs the owner actually needs (Tables and Export first; Analytics after wave 2).
3. The chat stays a mock until ADR-0015 (proposed) is approved or rewritten.

### Phase 4 — court output and the Workbench

1. Render the five Evidence.dev reports against the live database. The portal lane already plans an
   Evidence.dev host (09-30 TODO, P-4): share it rather than run a second one. The pages query
   Postgres today; with analysis in Surreal they need a route to it, which is checked against
   Evidence.dev's connector list at that point, not assumed.
2. Exhibit export: each map or table carries its source chunk, record ids and hashes.
3. Fold the geo views into the Probata Workbench (V-1 A, last step).

## Order of work

Phase 0 is under way (V-3 still open). Phase 1 needs V-4, V-5 and V-6. Phase 2 needs V-7 and
V-8. Phase 3 needs V-2. The UI and reports in Phases 3–4 read analysis from Surreal and source
records from Postgres. Nothing in this plan touches the Case Bible, Workbench or portal lanes, which other sessions
own; Phase 4's Evidence.dev step coordinates with the portal lane.
