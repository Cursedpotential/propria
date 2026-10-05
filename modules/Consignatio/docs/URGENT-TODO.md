# Consignatio urgent TODO

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_

Open items only (owner 2026-10-02 19:18 EDT). When an item is finished, move it to [COMPLETED-TODO.md](COMPLETED-TODO.md) with its date and proof in the same turn; never tick it and leave it here. What happened and why goes in [LOG.md](LOG.md). Each item names the log section it came from.

<!-- MAP:START -->
## Map: where everything is (owner 2026-10-02 19:30, option A)

> _Claude Code · Opus 5.5 · 2026-10-02. Keep this short; the detail lives in the registry._

**Start here: `raw_duck.catalog_registry`** (PG `casebible`, container `casebible-pg-*` on ovh-files). One row per catalog table and per outside object (buckets, Workers, ledgers, VPS folders, Weaviate collections): what it covers, as of when, `current` / `superseded` / `historical` / `unknown`, what replaced it, which script made it. Every table's own comment starts with `[registry: STATUS, as of …]`. Rebuild: `casebible/tools/catalog_registry_build.py`.

| Question | Where |
|---|---|
| What exists in which bucket (B2 and R2)? | `raw_duck.bucket_objects_current` (whole-bucket listings; loader `casebible/tools/bucket_objects_load.py`, raw listings in ovh-files `/data/consignatio/listings/`) |
| Where did each file come from? | `raw_duck.source_occurrences` (frozen 2026-09-14) and `catalog_reconcile.*` (2026-09-20) |
| Vault: kept copy vs deleted copy | `raw_duck.vault_keep_v7` ⋈ `vault_delete_v7` |
| Stale tables | schema `raw_duck_superseded` (moved 2026-10-02; nothing deleted) |
| Published copy of the catalog | B2 `salem-data/consignatio/_system/lake/<date>/` (verified 2026-10-04; `raw_duck.lake_publish_20261004`; see `CASE-BIBLE-CATALOG-GUIDE.md`) |
| Searchable content | Weaviate: messages yes (`MsgEvents20260918`, `ProfferMsgEvents20261002`), documents no (`DocEvents20261001` = 0) |
| R2 hashing | Worker `casebible-r2-hasher` (`casebible/tools/r2_hash_worker/`); older SHA-256 ledger in R2 `casebible-hash-ledger` |
| What happened and why | [LOG.md](LOG.md) · finished items [COMPLETED-TODO.md](COMPLETED-TODO.md) · receipts `docs/receipts/` |

**Renamed 2026-10-02 (old name -> new):** `raw_duck.b2_objects` -> `raw_duck_superseded.b2_intake_objects_20260914`; `raw_duck.vault_objects` -> `raw_duck_superseded.vault_objects_20260916_0810_prededupe`; `raw_duck.vault_content_v0` -> `raw_duck_superseded.vault_content_v0`. The Probata engine and `contacts_manifest.py` read `bucket_objects` now.
<!-- MAP:END -->

## Open items

### From: 2026-09-13

- [ ] **One file has no good copy anywhere:** `Google Data Export Archive Contents (0015A63D).html` (179,200 bytes, all zeros). Owner to decide whether to re-export it from Google.

### From: 2026-09-14

- [ ] **Tier 1 full rclone metadata re-inventory of both Drives (owner 01:48 EDT: "it's cheap, pull whatever rclone can pull").**
  - `rclone lsjson -R --files-only -M --hash --drive-metadata-owner read --drive-metadata-permissions read --drive-metadata-labels read`, with no `--drive-skip-gdocs`. Metadata only, no downloads.
  - Supersedes the 2026-09-13 inventories as the baseline (those lack permissions, labels and native Docs).
  - ~~Gate: … then sign-off.~~ **Corrected 02:00 EDT: AUTHORIZED.** The owner directed it at 01:48; it is a metadata-only listing and moves no data. Only constraint: don't starve the running salemnet copy-by-ID unit on the shared Drive client. Run by consignatio-d6.

- [ ] **Tier 2 follow-up tooling: Google Cloud CLI / direct APIs for evidence packets (owner 01:48 EDT).**
  - Must be ready for per-item follow-up: revisions, comments, Drive Activity, native exports, People, Gmail notifications. See the Intake spec `backend/docs/SOURCE-METADATA-CAPTURE-AND-FORENSIC-PACKAGE-SPEC.md` §7c and the research doc §6–§7.
  - Blockers: the gadmin MCP needs reconnecting with more permissions (owner action), and gcloud/API credentials need Drive scopes per shard.

- [ ] **Probata workbench file management / tools — diagnosed 11:00 EDT (owner: "yes I do").** Read-only findings on ovh-app (`workbench-xjbuo6drbwjfby75lalk8bk7`, `probata-workbench:latest`, Coolify app `xjbuo6drbwjfby75lalk8bk7`, build pack dockercompose from `Cursedpotential/probata:main` `/deploy/workbench.yaml`): (1) `/api/tools` → `[]` because the deployed env had **`MCP_SERVERS=[]`** — the Tool Explorer proxies MCP servers, not tool-runtime's REST registry (`:8090/tools`, 43 tools, healthy). Live MCP endpoints from the VPS: propria-docstore docs `https://surreal-docs.tilapia-skilift.ts.net/mcp` (initialize 200); memory MCP `:8471` → 403 "Host header is not allowed" (its allowlist); Graphiti `:8071/mcp` down; gateway `:4096/mcp` → 401 (bearer). **Fix applied 11:05 EDT:** production `MCP_SERVERS` set to the docstore server via Coolify (`PATCH /envs/bulk`, 201) and redeploy queued (`aykz2c6w5ft44cz6gao118et`); verification of `/api/tools` pending the build. Add memory/gateway once their auth/allowlist is settled. **11:05 EDT — deploy finished (`aykz2c6w5ft44cz6gao118et`), runtime env correct, `/api/tools` still `[]` and no per-server warning: root cause is in `modules/workbench/api/app/config/settings.py::mcp_servers_parsed` — every entry without `gateway: "portkey"` is dropped unless `MCP_DIRECT_BYPASS_ALLOWED` (a diagnostic flag) is set.** That is the owner's shared-tool-gateway policy encoded in code, so the Tool Explorer can only show MCP servers registered behind Portkey (`PORTKEY_BASE_URL`/`PORTKEY_CONFIG` env on the workbench). **OWNER DECISION:** (a) register the propria-docstore MCP (and later memory/Graphiti/gateway) behind Portkey and set `MCP_SERVERS` entries with `gateway: "portkey"`; or (b) accept the diagnostic bypass on the tailnet-only workbench (`MCP_DIRECT_BYPASS_ALLOWED=1`) as a stopgap. I did not flip the security gate. The `MCP_SERVERS` docstore entry stays in place (harmless; becomes live under either option). Portkey in front of the workbench: `portkeyai/gateway:1.15.2` at `http://100.72.169.40:8787/v1` (`PORTKEY_ENVIRONMENT=production`, `MCP_DIRECT_BYPASS_ALLOWED=false`); whether this gateway version proxies MCP servers needs checking before option (a) is promised. Host note: ovh-app is a 7 GB box with ~2 GB available at 11:05 EDT — the portal/workbench answer in 8–15 s and one SSH timed out; memory pressure, not an outage. **Drive salemnet copy-by-ID finished 10:59 EDT: 14,748 / 14,748, 0 failed** (receipt `gdrive-salemnet.idmap.copyid-20260914-045307.receipt.jsonl`); the Drive loader still waits on the salemnet Tier-1 baseline. (2) `/api/files` → `[]` because **`staged_files` is a LanceDB table at `LANCEDB_PATH=/data/lancedb`, which is not on a mount** (mounts: `/data/inbox`, `/data/copilot`, secrets) — every redeploy wipes it; the table was created at the last start and is empty. Real fix = a persistent volume `/data/probata/volumes/workbench/lancedb:/data/lancedb` in `/deploy/workbench.yaml` (repo edit; this session's classifier denies edits in the Probata repo — needs the owner or a session with that permission; a `docker_compose_raw` API edit would be overwritten by the repo on the next deploy). Staging only happens through `POST /api/upload` / `/api/proffer/upload`; nothing reads B2 into the workbench — "view B2 from Probata" is a feature gap, not a bug.

- [ ] **Owner rulings 10:36 EDT:** (1) "if something needs my review it should never kill all the other tasks waiting on me" — a paused-for-review item never blocks unrelated work in its lane; keep going on everything else. (2) "propria is supposed to be the only plugin, entire project wide; probata is legacy and was supposed to be done" — `propria-docstore` is THE plugin; the `probata` marketplace/cache path is legacy naming to retire (follow-up: rename the marketplace entry so the cache lands under `propria/`). (3) "yes I do" → work the Probata workbench file-management lane (`/api/tools`/`/api/files` empty, monitored-actions 404) — queued right after the docstore fix. (4) OpenList: "make an agent login or use mine" — `~/.secrets/openlist.env` has the admin password, which I do not type to authenticate; a stored token is fine: owner pastes OpenList's permanent token (Settings → Other → Token) as `OPENLIST_TOKEN=` in `openlist.env`, or creates an agent user and gives its token.

### From: 2026-09-14 night — change log (portal, logins, lockdown, DNS, federation)

- [ ] Path-blocking pattern at the gate (owner 21:52) — list not written yet.

- [ ] Owner click-tests pending: Coolify / ContextForge / OpenList / Temporal Authentik logins.

- [ ] Intake co-workspace: docked-iframe review panel rejected by owner (22:02); native panels only (other lane owns rebuild).
  - _Unclear (2026-10-02 triage):_ Informational: owner rejection of docked iframe; 'other lane owns rebuild'; no later evidence either way

- [ ] Traefik hygiene: rule-less `api@internal` router label on coolify-proxy (unreachable today) — remove.

- [ ] Temporal client secret was printed in agent tool output (transcript only, not git-tracked) — rotate only if owner wants.

- [ ] **Owner click-test:** log in at https://homepage.int.mitechconsult.com and confirm the public widgets and charts render (no agent can log in as the owner).

- [ ] Owner: open the editor in a real browser and confirm the side-by-side preview renders and reloads on save; if the service-worker error persists, fix webview origin config (code-server behind Traefik/tailscale serve).

- [ ] **Security note:** the agent's `GET /api/v1/security/keys` printed Coolify's stored SSH / GitHub-App **private keys** into its transcript (not git-tracked). Owner policy says transcript-only exposure needs no rotation, but private keys are higher stakes than tokens — owner decides whether to rotate. Also: 3 stale Docker networks pruned on ovh-app (`…_workbench`, `…_agentos`, `…_librechat`) to free the address pool; root-only temp token files left in ovh-app `/tmp` (rm blocked by guard hook).

- [ ] **family-court-console: served through ContextForge, two owner decisions left** (owner 2026-10-02 22:26, "Switch family-court-console to the hosted copy through ContextForge, so sessions stop each running their own"). _(Claude Code · Sonnet 5.5 · 2026-10-03)_
  - _State, read back live 2026-10-03:_ the Coolify app `family-court-console` (`sokv65ibdq2y8xdaqmd6p4rq`, ovh-files, `100.91.190.107:9077/mcp`, case data in the shared `surreal-case`) is ContextForge gateway `family-court` (`4fb64f88c1bf4b4d8863658e32d98403`, its bearer held inside ContextForge) and virtual server `family-court-console` (`0b85f64905be468ba477d75cb2a80323`; 28 tools, 10 resources, 2 prompts). The 2026-09-08 plan's name `case-work` was not used. ContextForge names each tool `family-court-<tool>` with hyphens (`case_put` is `family-court-case-put`); the plugin's README and entry skill say so. Claude Code (propria-plugins 3.3.0, commit `142c84c`, the plugin's `.mcp.json`) and Codex (`[mcp_servers.family-court-console]` in `~/.codex/config.toml`) both reach it there. Proof: a fresh `claude -p` and a fresh `codex exec` each called `family-court-search-guide` (read-only; 12 results, first `MC-416`, the same as the direct call) while the process table was sampled: no `node mcp-app/dist/server.js` under either, and none created anywhere on the machine since. Five Claude sessions that started before the change still hold a local 3.2.4 copy until they end.
  - **Owner decision, inline widgets:** ContextForge drops the MCP-app metadata (`_meta.ui`) and serves the widget resources as `text/plain`, so the 8 inline widgets (`open_dashboard` and the route, deadline, packet, checklist, sources, search and chronology views) do not render in the apps; every tool still returns its data as text. (A) accept (default); (B) find a ContextForge version or setting that forwards `_meta` and the `text/html;profile=mcp-app` type.
  - **Owner decision, file paths:** `case_import`, `case_export` with a `path`, and `case_reference` load-from-path now read and write the hosted container's filesystem, not the desktop's. (A) leave as is (default); (B) let `case_import` take file contents instead of a path.

- [ ] **tavily / courtlistener:** no API keys in `~/.secrets` → owner supplies keys, then register.

- [ ] **cloudflare-api/bindings/builds/observability:** per-user OAuth only — cannot federate with a static key; stay direct (or revisit if ContextForge supports OAuth passthrough).
  - _Unclear (2026-10-02 triage):_ Decision-type note (stay direct); no later change found, nothing to execute

- [ ] **ContextForge v1.0.4 → v1.0.10 upgrade** still pending.

### From: 2026-09-15 — where file-truth lives (one place), receipts moved into the repo, handoff_write fix

- [ ] Junk-filtered recount of the 871 "missing" files (some are `flet_env` venv DLLs in the R2 quarantine bucket).

- [ ] Owner asked 00:25 whether an agent changed mouse/window-focus settings: read-only check shows Windows focus-follows-mouse (`UserPreferencesMask` bit 0) is ON with `ActiveWndTrkTimeout` 100 ms; nothing this session ran touches settings; when it was switched is not recorded in the registry. One-liner to turn it off given in chat.

### From: 01:25 EDT — FileFlows installed (owner 01:11 "I WANT THIS INSTALLED... CUSTOM COMPOSE")

- [ ] Coolify status field still says `exited:unhealthy` (stale; container is healthy) — refreshes on its own.

- [ ] Public `fileflows.int.mitechconsult.com` route pre-wired but not opened (needs Authentik app + DNS like the other .int surfaces).

### From: 2026-09-17 03:00–05:05 UTC — memsearch-milvus RESTORED (Claude Code · Opus 5)

- [ ] **Index pollution:** the same search returned `Consignatio/Intake/node_modules/@testing-library/jest-dom/README.md` — third-party
  node_modules content is in agent memory. Related to the 9,642 Consignatio rows item above; memsearch watch paths need excludes.

- [ ] Move off the nightly to **milvusdb/milvus v3.0.1** (released 2026-09-09) after a backup; nightly→3.0.1 readability unverified.
  Re-test JSON shredding there.

- [ ] Backups: schedule `etcdctl snapshot save` on `memsearch-etcd` + a segment backup (`milvus-backup` supports ≥3.0.1).

- [ ] Coolify's stop did not appear to wait the 300 s grace (05:00:09→05:00:19); harmless now that etcd is separate, unverified.

- [ ] Probata main checkout still shows the 09-10 session's uncommitted `M deploy/data-vector.yaml`; that file is deleted upstream,
  so the next pull there will conflict. Owner/that session to discard it (content shipped in `06a6c82`).

### From: 2026-09-16 08:10 EDT — desktop process audit + docstore hook leak (Claude Code · Fable 5.1, intake-23)

- [ ] Kill list (owner approval; nothing killed by the agent): the 54 UniGetUI pwsh (their conhosts follow), the 10 orphan bash, python3 37008. Root fix: UniGetUI → Settings → Package managers → Scoop: disable automatic update checks (or fix the `scoop update` hang: run `scoop update` in a terminal to see what it waits on — likely a git prompt or a stale lock).

### From: 2026-09-16 morning — vault twins, quarantine re-home, Spacedrive visibility

- [ ] **19:11–19:13 EDT OWNER RESET — "let's try this a different way": the twin analysis failed its main goal.** Named examples the normalizer never grouped: `Case Bible` / `Case Bible Backups` / `Case Bible BACKUP 2026-03-12` / `Backup/Case Bible` ("LITERALLY THE MAIN GOAL AND THE MAIN TASK"); `Snap_Export_2025-04-11` and `_SWEPT/Snap_Export_2025-04-11` byte-identical and both still present; top-level `Court` should fold into `Court & Legal Project` (merging with the nested `Court`); `Cube` + `Cube ACR` merge, then into `Evidence` where the same folders already exist. Root cause: `vault_name_normalize.py` keys on the exact name minus copy-numbers, no same-meaning fold (backup/backups/dated suffixes, wrapping backup/archive containers). Nothing was ever executed by the twin work; no byte in vault/v1 has been copied, moved or deleted by it. **THE PLAN NOW (owner 19:13, = the 09-15 04:55 order): owner names the pairs, agent executes one at a time — show counts (identical / carry / conflicts), server-side copy carries into the home, verify by listing, write occurrence rows, retire the folded folder. No more global normalizer passes, no more proposal tables.** Open: what "retire" means in vault/v1 (delete the folded keys — source-bucket copies untouched, every deleted key logged with hash + new home — vs park under .review_hold/merged/ vs catalog-only); owner dismissed the question at 19:12, unanswered. Rules of 15:25–15:33 still stand: Takeout / Facebook / Snapchat containers and any dir attached to a doc that references it are units, never processed inside; top-level photo dirs may merge, camera/date containers inside kept; recovery-name collisions → best-copy rule (most accurate metadata wins). First pair: Case Bible family → home `Case Bible`. _Claude Code · Fable 5.1_
  - _Unclear (2026-10-02 triage):_ Owner reset 19:13: pair-by-pair execution (Case Bible family first); no later entry shows a pair executed; line 2983 tags twin tables stale_tree. Likely overtaken by later catalog work.

### From: 2026-09-16 22:00-23:00 UTC update — source resolved by parent, import applied, boot hang under investigation

- [ ] Owner: fully restart the Claude desktop app so sessions load `MEMORY_BASIC_AUTH` and `CF_MCP_CLIENT_TOKEN`.

### From: 2026-09-16 19:13 agent — memory half fixed end to end, 0.6.4 shipped

- [ ] Owner: restart Claude desktop so sessions load 0.6.4 + env vars.

- [ ] **Docstore graph build (owner 19:21 EDT: "yes build it").** Agent dispatched 19:32: additive schema (theme, feature, rule[restriction|expectation|contract|do|dont], code_ref; edges about, describes, states w/ verbatim quote, references_code, duplicate_of, relates_to, implements, constrains, conflicts_with, rule_supersedes), NIM nemotron guided-JSON extraction on the VPS keyed by chunk hash (incremental), 25-doc sample → purge → full 1,217-doc run, query functions (graph_neighbors, rules_for, duplicates, theme_docs, feature_map, conflicts), quote-fidelity spot check, plugin 0.6.5 graph skill. No people/places entities.

### From: 2026-09-16 20:25 EDT — RESOLVED: zero FAILs, verified full run, decision amended

- [ ] **Reconciliation adapter** (`docstore_reconcile_query` / `docstore_reconcile_packet`): returns
  "Propria reconciliation adapter is unavailable". OWNER ACTION: stand up / point the control server
  at that adapter, then re-run `test_plugin.py --only mcp`.
  - _Unclear (2026-10-02 triage):_ No later mention of the adapter in file; plugin now has reconcile_* tools in session but adapter fix never logged

### From: 2026-09-17 morning — Spacedrive empty-folder bug reproduced; Xplorer-on-the-portal options

- [ ] Move `homepage`, `homepage-public`, `portal-editor` into Coolify; identify and remove/adopt the two unnamed containers on ovh-files. The Xplorer server gets built as a Coolify app from day one.
  - **2026-09-27 (Claude Code · Opus 5.5):** `homepage` and `homepage-public` are declared in git and the Coolify app `propria-portal` exists; the cutover waits on the owner's permission. `portal-editor` is an owner decision. See the 2026-09-26 portal entry at the end of this file.

- [ ] Still uses the old name: rclone `.spacedrive` marker retry loop on ovh-files (restart `rclone-openlist` to clear); stray `.spacedrive` files under `/b2` to list for owner-approved removal.

### From: 2026-09-17 08:40 EDT — Intake web-mode contract found in the donor (plan only, nothing built)

- [ ] v2: Windows/Tauri native desktop port (owner 08:43), after hosted v1 is live-verified.

### From: 2026-09-17 09:57 EDT — merge and move `Propria/projects/consignatio` (owner: "merge and move", "ensure we don't lose anything")

- [ ] Chat model quality (nemotron via Portkey: reasoning leaks, wrong tool picks) — owner model choice.

- [ ] Owner-only: remove `Propria/Consignatio.junction-2026-09-18-undo` and, once satisfied, the six `memory.pre-merge-20260918` backups.

- [ ] Next projection push: include the `FL-MCP` source_root; verify with docstore_health (cdc_attribution verified) that family-court docs keep their rows.

### From: 2026-09-18 — Intake v1 follow-ups

- [ ] Open: Chat model choice (owner). Sort-order live proof for unknown dates. `rg` content mode reads B2 objects over FUSE, so decide whether to cap or disable it in hosted mode. Deploy and wire the filesystem search service when approved.

### From: 2026-09-18 — Intake chat model switch

- [ ] **Owner choice — Gemini free-tier throttling.** Gemini 3.8 Flash on the free key returns 503 "high demand" and 429 per-minute quota often enough that about 1 in 3 live turns fell to Kimi (~100–130 s, weaker: it re-proposed steps that were already done). Options: (a) keep as is (default); (b) add a second Gemini model with its own quota (e.g. `gemini-3.7-flash`) between 3.8 Flash and Kimi; (c) a paid Gemini key.

### From: 2026-09-18 — Intake index-first search

- [ ] Owner: the scoped `intake_runtime` password (Windows Credential Manager) for the engine, so the Surreal timelines light up; nothing else is missing on the engine side.

- [ ] Owner go/no-go: step 1 = package `Intake/backend` as-is (Dockerfile + Coolify app on ovh-files, B2 mount read-only, output on /data), first run on ONE Katrina-priority subtree, verify a real search hit, before any code change to caps/extractors.

- [ ] Owner decision: SMS Backup & Restore XML handler — (A, default) SBV decoder does the streaming attachment capture + sanitized spool, DuckDB `elt_smsbackuprestore_v1` reads that spool for records (keep all attributes, not a projection); (B) SBV decoder registered as the whole handler for this signature because DuckDB cannot extract it losslessly.

- [ ] Owner go: create a separate Surreal instance (own Coolify app, storage, name — 09-10 isolation rule) for rough-draft graphs/timelines; `surreal-intake` stays unchanged as the CocoIndex graph; tonight's `tl_*` tables stay untouched until owner decides. Constraint: ovh-files ~20 GB free + a second 2 GiB-class instance.

- [ ] Owner go: Build 1 (catalog-backed source) in `Consignatio/Intake/backend` — local code only, diff shown before anything runs.
  - _Unclear (2026-10-02 triage):_ Line 2264: Build 1 PAUSED 22:52 for catalog audit; never logged resumed or cancelled (line 2413 shows Intake/backend image search code later)

- [ ] Owner go: run `vault_index_source_20260918.sql` on the catalog (creates one new dated table, reads only), then report the two check lines.

- [ ] Owner go: (1) read-only size check of every B2 area + R2 so the 15 TB is explained with numbers; (2) re-run the owner's existing grading rule (`grading_selection.sql` logic) over all `source_occurrences` → one new dated catalog table naming, per current vault file, its winning copy (oldest real date, most metadata; ties kept; zero-filled never). Nothing moved or deleted by either.
  - _Unclear (2026-10-02 triage):_ Part 1 size check answered from catalog at line 2268 (storage_sizes.txt); part 2 grading-rule re-run table not found later

- [ ] Implement project-wide default scope in the control server + plugin skills (0.8 baseline).

- [ ] Build graphs and memories automatically, and make memories graph-shaped (relationships between memories/docs), not flat rows.

- [ ] Continue targeted OD/GD/local/R2 metadata and package-completeness checks from the persisted worklist; reuse SHA-256 receipts, resolve 28,069 R2 catalog identities, and assess recovered/original metadata before BAS decisions. Server-side B2 Parquet access remains unverified. New PostgreSQL metadata uses about 6.41 GB; host approximately 20 GiB free, so budget subsequent generations.

### From: 2026-09-20 — surfaces only (owner 10:03: "control surfaces mean anything I use"; no B2 / catalog work)

- [ ] The to-do sync is still manual. Make it part of the turn that edits this file, or a scheduled pull on ovh-app.

- [ ] Not tested: that a real create/rename in a watched folder still refreshes the list. The only watched roots are on the B2 mount and the owner barred B2 writes today; test on the first owner-directed file operation.

- [ ] Advocatio: point the web app at a browser-reachable API origin (same-origin proxy like the Workbench BFF).
  - _Unclear (2026-10-02 triage):_ Line 2890: Legal Work Desk now at legal.int.mitechconsult.com, but no entry confirms the API origin fix

### From: 2026-09-20 21:05 EDT — missing-payload register: ingestion side started in Probata; Codex's uncommitted catalog tooling committed

- [ ] **Gap:** the script that created `raw_duck.missing_message_payloads_20260920` + view `catalog_reconcile.missing_message_payloads` (16 rows) was run inline by Codex and is in no tracked file. It exists verbatim in the transcript source rollout (2026-09-20T21:27:47Z). Needs a tracked dated script here; `payload_id = sha256(json.dumps([backup_sha256, mms_attributes, part_attributes], sort_keys=True))`.

### From: 2026-09-22 19:35 EDT — BUILT (not deployed): `intake_search_names` + the Find by name box

- [ ] **Not deployed.** Nothing pushed; the live search box at `https://homepage.tilapia-skilift.ts.net/progress/intake/xplorer/` is unchanged until the parent authorizes the deploy that also carries Smart Suggestions step 1.

### From: 2026-09-23 — portal surfaces, Authentik login, public Workbench route (owner 08:15–09:41 EDT)

- [ ] Owner click-checks: public portal login → Probata card; tailnet Neo4j tile (Bolt over TLS; headless Chrome can't finish a live socket).

- [ ] ~~ContextForge `/admin`, OpenCode, n8n, Temporal and Infisical still have their own app logins.~~ **2026-09-24 05:35:** OpenCode, n8n and Temporal have no app login on the tailnet; off the tailnet, the Authentik login is the only login. For n8n that works through a hook trusting Tailscale's and Authentik's identity headers. Details in Probata `docs/planning/2026-09-20-TODO.md`. **Still open (owner call):** ContextForge (its tokens also guard the public `mcp.mitechconsult.com`) and Infisical.

### From: 2026-09-24 — tailnet short names, ovh-app disk, portal asks (owner 05:33–05:41 EDT)

- [ ] **Owner go + timing:** move containerd and Docker data onto the block volume and add it to fstab. This means a 15–30 min outage of everything on ovh-app.

- [ ] **OpenCode ↔ local projects:** the owner expects OpenCode to pull/sync projects from his machine, or push out, and thinks this was part of the reason for OpenList. Check what exists: OpenCode mounts `/mnt/desktop-share` over SMB today.

### From: 2026-09-24 09:28–10:15 EDT — extraction test run on Matt's 2023–24 side: breaks found and fixed (owner 09:39: "you're testing everything … this is where we find out where the breaks are, and we fix it. Make notes. Make to-dos.")

- [ ] Readers still missing (the runner lists each as `no_reader`, per the owner's "no tool → add one"):
  - 6 PDF transcripts, including the 20 MB `Court/imessage export 8102689630 2023-2024 [gdoc].pdf`. Use pdftotext / pypdf / pypdfium2 per the bake-off.
  - 3 XLSX exports.
  - 3 Cube ACR recording-metadata JSON files.
  - 21 Google Voice MMS images, which must be extracted as attachments at parse (standing rule).
  - 12 "Messages with Katrina" CSVs plus 4 iMessage CSVs. Some are known scrambled or shuffled: check them with the repair toolkit first.

- [ ] Attachment-only messages: match by time + direction + attachment presence/type (2,660 undecided after 2024-06-27).

- [ ] Clock basis still unverified for the iMessage TXT export and the WhatsApp export (read as local). Cross-check them the way the HTML export was checked, once a second copy overlaps.

- [ ] The HTML/TXT readers read the whole file into memory (`read_text`, files ≤ 20 MB here). The owner's "stream everything" rule says replace them with line-streaming readers like the SMS sanitizer.

- [ ] His 7 SMS backups under "Messages with Katrina" (2022, 2025) are only in the 09-18 discovery index (v1 readers). Re-extract them on v2 when those years are in scope.

- [ ] Rebuild devbox from the updated Dockerfile (Probata `68be0db`) when convenient; the running container was patched in place.

- [ ] devbox disk is 93% full (15 GB free). The blank 500 GB `sdb` on ovh-files is still unused (earlier to-do).

- [ ] Owner cleanup in devbox `persist/work/`: test bundles `elt-msg-smoke`, `-smoke2`, `-smoke3`, `-smoke4` (never loaded into the catalog) and an empty folder `comm_timeline_mvp.next`. The guard blocks agent deletes.

- [ ] Auto-memory could not be written: the memory folder is outside the workspace folders admin policy allows. This morning's rules are recorded only in this entry.

### From: 2026-09-24 13:20–14:30 EDT — publish, Jev scored, her other phone files, damaged backups salvaged

- [ ] **Owner:** R2 was re-enabled by the owner; a new R2 key is needed (save to `~/.secrets/cloudflare-r2.env`). Then wire it into Workbench and the ovh-files `r2:` remote and re-check the 3 R2 roots and `nexus`.

- [ ] 13 recovered-disk XML fragments still fail (binary noise / broken attachment lines; several cut at exactly 256 MB). They need a line-level salvage that skips corrupt lines. Toolkit first.

- [ ] `f850530928.xml` shows both 9302 and 9303 as the phone's own line. Investigate before relying on its `owner_line`.

- [ ] Empty stray folders `E:/AI_Workspace/Projects/Probata/probata/scripts/jev_eval/`, left by a wrong write path and since moved. The owner deletes them (the guard blocks it).

### From: 09:32–09:40 EDT — follow-ups (owner answers)

- [ ] **Discuss later (owner 09:36): OpenList vs a sync / file-drop tool.**
  - The owner expected the storage layer to give easy desktop ↔ VPS ↔ bucket sync, and asked for Syncthing "or some other file drop type application" (09:33–09:34).
  - This conflicts with the 2026-09-14 rulings: OpenList is a viewer only, and transfers go direct to the source. On 09-13 its WebDAV path silently copied 0 of 22,565 files.
  - Owner: "I was unaware of this and we'll have to discuss this later."
  - The 2026-09-07 header of `deploy/openlist.yaml` already names "the desktop via its own WebDAV/SMB share or Syncthing".

### From: Open

- [ ] **Nine stateful services** (authentik, neo4j, pg-files, weaviate, milvus, surreal ×2,
  temporal, infisical) still carry stale webhook ids. Not broken — they simply will not
  auto-deploy on push until each is deployed once. Held back rather than cycling databases
  late at night.

- [ ] **Five dangling gitlinks in Vestigia** (`Tether`, `TetherPro`, `notebooklm-mcp-source`,
  `notebooklm-mcp-target`, `pandoc-lua-filters`) — mode 160000 with no `.gitmodules`, so a
  recursive clone fails. Inherited from Vestigia's own history.

- [ ] **Archive the four old GitHub repositories** read-only once every app has deployed from
  propria. They are the rollback path until then.

### From: 2026-09-26 23:20 – 2026-09-27 01:00 EDT — Homepage portal declared in git and rebuilt (portal lane)

- [ ] **Progress board cutover — owner's go:** it still runs as the Coolify service
  `homv6zeg4ay2r2puxtzakf83` straight from the host folder; switching it to the git-built image
  takes port 3020 from it (the same permission gate as above). Re-check the host copy against the
  README hashes first. `intake-build/` (251 MB of Intake preview releases) and
  `data/URGENT-TODO.md` stay host data, written by processes outside the repo.

- [ ] **Owner decision — the portal editor** (`portal-editor`, code-server on `/data/dashboards`,
  `portal-edit.tilapia-skilift.ts.net`): after the cutover it edits a host copy the portal no
  longer reads. Keep it, repoint it at a repository checkout, or retire it. Its tile says so.

### From: 2026-09-26 23:30 – 2026-09-27 00:45 EDT — public-portal edge: Traefik reaches Authentik through `svc:authentik`, the Workbench through one tailnet door; `propria-edge` retired (owner 22:52 "Nothing is supposed to be created that way. Ever." · 23:05 "Fucking fix it." · 23:07 · 23:08 · 23:25)

- [ ] Option for the owner: a tsnet in-container listener would put `svc:authentik` inside the tracked compose and let it move with the container. It is not proven: it needs a raw-TCP mode (tsnet-front is an HTTP proxy) and an auth-key file, and tsnet-front crash-looped without one today.

### From: 2026-09-27 00:37–00:57 EDT — lakehouse published: the catalog on B2 as Parquet

- [ ] **Owner: local staging on ovh-files.** `/data/consignatio/lake-publish-20260927/` holds the published bytes (1.5 GB), the readback copies (1.5 GB) and the excluded exports (200 MB). Delete them or keep `2026-09-27/` as a local mirror; the guard blocks agent deletes.

- [ ] **Evidence.dev (Probata 09-27 #4)** reads `LATEST` and then the dated folder. It needs a read-only S3 key; whether `B2_KEY_ID` is read-only was not checked. The DuckDB httpfs read itself has not run anywhere yet.

- [ ] **Refresh:** the next publish writes a new dated folder and rewrites `LATEST`, the first change to an existing lake object. The owner picks the cadence.

### From: Still open on this surface

- [ ] `GET /api/tools` returns HTTP 200 with an empty array.

- [ ] `GET /api/monitored-actions/capabilities` returns 404; no such backend route exists.
      Both are Docstore `note:probata_function_access_broken_20260912`, still active.

- [ ] No shared package across 40 frontend apps (14 TypeScript, 12 React, 13 Vite
      versions). `design-contract` is canonical by owner decision 2026-09-24 and has zero
      consumers. Audit: `modules/Probata/probata/docs/probata-surface-buildkit/STACK.md`.

### From: 2026-09-27 02:15–02:25 EDT — portal cutover done; LibreChat owner account created

- [ ] **Tailnet must not hit Authentik on `*.int` hostnames** (owner 2026-09-26 23:08 EDT; "there
  was supposed to be a patch"). The agent dispatch for this was refused by this session's auto-mode
  classifier ("Security Weaken") — it needs the owner's explicit go-ahead or a permission rule.
  - _Unclear (2026-10-02 triage):_ line 3077 forward-auth label fix on 4 apps and 3279 domain-wide provider covers *.int hosts, but no entry says tailnet bypass of Authentik on *.int was delivered

### From: 2026-09-27 10:25 EDT — Docstore memory writes fixed, errors report their reason, write schema documented (owner 10:01 EDT)

- [ ] **Open:** `release_api.invoke` still maps a plain `ValueError` to 409 for upgrade, adr, knowledge and sources. It needs a per-operation audit to split 409 from 422.

- [ ] **Owner call:** the root `plugins/docstore/` is a stale 0.6.3 snapshot that still documents raw `fn::remember` with a `probata` scope. (A) quarantine it, (B) replace it with the canonical tree, or (C) leave it.

### From: 2026-09-27 11:15 EDT — Probata Workbench: decision-free fixes built, deployed, six-step audit 6/6

- [ ] DF-33 (Intake lane): `/api/intake/discovery/unit-lookup` answers 503 and every Intake search mode reports false.

- [ ] DF-34: Review calls `/api/monitored-actions/capabilities`, which has no route (404). It is part of DF-13 and OD-04.

### From: 2026-09-28 04:01 EDT — Family Law Toolkit sources: R2 → B2 and a B2 → Surreal sync (plan; nothing moved)

- [ ] **Phase 1 done; Phase 2 waits on the owner's sign-off** (owner 04:01 "Everything needs to be migrated to B2. R2
  is being retired … synced with the canonical source … in Surreal … updated on change"). Plan and numbers:
  `docs/receipts/2026-09-28-fct-sources-r2-to-b2-plan.md`. Catalog load: `casebible/tools/fct_sources_inventory_20260928.{sh,sql}`
  → `raw_duck.fct_sources_inventory_20260928` (543 rows).
  - 143 files / 52,446,084 bytes, identical in R2, on the desktop and on ovh-files; none missing, none differing.
  - Store: the 193 `source` rows carry 0 sha256 values and 119 `r2_path` values that point at nothing; the 28
    PDFs have no store row. Decisions D1–D6 are in the plan's §7.

### From: 2026-09-28 05:25 EDT — Family Law Toolkit hosted: tailnet + Authentik, portal tiles, preview retired

- [ ] **Owner:** tag `svc:family-court` with `tag:docker`. The OAuth client cannot assign it: tagOwners gives it to autogroup:owner only.

- [ ] **Owner:** policy grant `tag:docker → svc:family-court`, app capability `propria.mitechconsult.com/cap/family-court`, copied from the Workbench grant. The classifier refused applying it. Needed for devbox screenshots of the logged-in app; afterwards set `TAILSCALE_DEVICE_CAPABILITY` in the console env.

- [ ] **Owner:** the autoMode line for the widget buttons. The dashboard, packet, route and search buttons stay disabled while the guide release label is STOP_AND_VERIFY.

- [ ] Unverified: the public route while logged out. The classifier refused the read-only probe.

### From: 2026-09-28 05:45 EDT — plugin marketplace moved to `E:/AI_Workspace/plugins` (owner option A, 04:52 EDT)

- [ ] **Owner, after restarting every Claude session:** `python3 E:/AI_Workspace/plugins/tools/finish_move_20260928.py` — refuses while anything runs from the old folder or any file there is newer; then renames it to `~/.claude/local-plugins.pre-move-20260928` (quarantine), leaves a junction, and reinstalls the memsearch CLI from the fork's new path. Later: `--unlink` drops the junction.
  - _Unclear (2026-10-02 triage):_ quarantine folder ~/.claude/local-plugins.pre-move-20260928 does not exist but ~/.claude/local-plugins still does; AGENTS.md says marketplace moved 2026-09-28; no entry says the finish script ran

- [ ] `~/.claude/settings.json` path strings (Edit/Write permission, autoMode lines) go in the single owner command with the `propria-plugins` rename.
  - _Unclear (2026-10-02 triage):_ part of the same owner command as 3556; no entry confirms settings.json paths updated

### From: 2026-09-28 06:40 EDT — plugins: one folder per plugin for Claude Code and Codex; marketplace renamed `propria-plugins` (owner 04:53–04:54 EDT)

- [ ] **Owner call: the hyperfocus SessionStart hook fails in every Codex session.** `hyperfocus@hyperfocus-repo` (third-party) runs the bare path `${CLAUDE_PLUGIN_ROOT}/scripts/hyperfocus-hook.sh`; Codex runs hook commands through PowerShell, which hands a `.sh` file to the Windows file association, and this machine's `.sh` association (`shfile`) points at `C:\Users\matts\AppData\Local\hermes\git\bin\bash.exe`, which no longer exists. Codex also ignores the hook's `args`, so the script would get no `--root` and exit silently anyway. Proven 2026-10-02: one SessionStart hook fails per `codex exec`; with `-c plugins.hyperfocus@hyperfocus-repo.enabled=false` none fails. Options: fork it into propria-plugins with a Codex `commandWindows` (as memsearch does), or turn its Codex hook off. _(Claude Code · Opus 5.5 · 2026-10-02)_

- [ ] **Owner, after restarting sessions:** `python3 E:/AI_Workspace/plugins/tools/finish_move_20260928.py` (from the 05:45 entry).
  - _Unclear (2026-10-02 triage):_ duplicate of 3556; no entry confirms finish_move_20260928.py ran

### From: Found, not yet fixed

- [ ] Workbench `/api/tools` still `[]`: `settings.py:203` drops the only MCP entry (no
  `gateway`), and it is not pointed at ContextForge. `/api/monitored-actions` does not exist.

- [ ] Dead routes (knowledge/evidence/ingest, zero traffic in 48 h) + Agno removal (owner option B)
  + Legal-desktop `agno_client.py` repair — route map done by agent `agno-removal-map`.

- [ ] MCP Inspector never deployed. coolify-write queue: summary-only `list_applications` /
  `list_deployments_for_app`, `get_deployment` error filter, services in `check_port_collision`,
  host port-owner operation.

### From: 2026-09-28 07:30 EDT — plugin best-practice audit (Claude Code + Codex), Codex-only plugins joined `propria-plugins` (owner 04:57 EDT)

- [ ] **Owner calls** (details in the audit doc): skill `name` ≠ folder in 130 skills (two ids per skill across harnesses); 12 commands Codex cannot migrate (`$ARGUMENTS`); hooks call `C:/Users/matts/.local/bin/python3.exe`; 10 SKILL.md bodies over 500 lines; 5 family-court agent names with spaces; Codex skill-metadata budget (~29k chars of descriptions vs ~8k); Codex memsearch hooks from the upstream clone.

- [ ] **Owner:** re-trust the changed claude-never-forgets hooks in Codex (hook text changed, so its trusted hash no longer matches).

- [ ] Docstore ADR-0096 still names `~/.claude/local-plugins` and `casebible-local`; amend it through `docstore_adr`.

### From: 2026-09-30 07:30 – 09:50 EDT — everything through ContextForge (option A), context-mode, Morph, and the `surrealdb` plugin (owner orders 09-29 08:42, 09-30 08:20 and 08:23)

- [ ] **Owner:** set the `MORPH_API_KEY` user variable, or tell this session to copy the key out of `~/.claude.json.bak-2026-07-31-repowire`.

- [ ] **Owner decision:** tools for surreal-intake. A: leave it without (default; Intake is isolated on purpose). B: add the allow-list to its compose and a ContextForge gateway with its runtime user.

- [ ] **Owner:** remove the two switched-off ContextForge entries (gateway `ctl`, server `propria-docstore-retired-8172`).

### From: 2026-09-30 15:50–16:40 EDT — Case Bible: what to grab, from the catalog alone (owner 15:52)

- [ ] **Owner go: restore the 1,450 old-version files** on B2 by server-side copy into their vault paths. No egress.
  - _Unclear (2026-10-02 triage):_ VPS copy/restore.log: 324/324 ok, restore.tsv 325 lines (11:30-11:37Z 2026-10-01) but item says 1,450 files; no line in file reports the 1,450 restored

- [ ] Google file `Google Data Export Archive Contents (0015A63D).html`: owner answered "yes" to "re-export or drop"; which one is still open.

- [ ] Build the per-source packages (D-154), with the byte check of the 227 renamed large files and the 1,049 name+size matches.

### From: 2026-09-30 22:21–23:10 EDT — is every vault file what it claims to be? (owner 22:21)

- [ ] **Owner go (no cost beyond compute): hash pass** on ovh-files over the 1,351 hash-less objects (1,316 matched-by-size + 35, about 2.6 TB): stream from B2, compute SHA-256/SHA-1/MD5, compare with each source's own SHA-256. B2 egress is free at this volume; about 3k download calls (< $0.01); about 5 h.
  - _Unclear (2026-10-02 triage):_ VPS hash-pass/run.log 790/790 ok plus quickxor_results.tsv and moved_sms results (2026-10-01) but item scopes 1,351 objects; no file line reports completion or results

- [ ] `onedrive/Pictures` provenance: find the OneDrive listing those 41,801 files came from and load it as occurrences (catalog first; no new OneDrive read unless no listing exists).
  - _Unclear (2026-10-02 triage):_ VPS onedrive-pictures/pictures.lsjson (42 MB, 2026-10-01) exists, but no line shows it loaded as catalog occurrences

- [ ] Decide the disk-image work files and the ISO: keep as artifacts, outside evidence selection.

### From: 2026-10-02 01:19 EDT – ongoing — Devbox becomes a Kasm Workspaces workspace (P-1), and agent work stops dying with the container (owner 01:19, 01:27)

- [ ] Owner: VNC_PW appeared in the Kasm service argv in `ps` (transcript only, not in git). The coolify-write half is fixed and moved to COMPLETED-TODO.md (2026-10-02).

### From: 2026-10-02 02:54–03:46 EDT — Coolify: long builds, the 4.3.23 upgrade, SSH sharing back on (owner 02:54 "a", 03:41)

- [ ] coolify-realtime still 1.0.16 (compose names a newer one); bring it in line with the next Coolify restart.

### From: 2026-10-02 07:44–08:35 EDT — Devbox redeployed through Coolify; step 1 of P-1 verified live

- [ ] **Owner:** sign Claude Code in at `http://100.91.190.107:7681` (tailnet): `claude` opens, then `/login` with the Max subscription.

- [ ] Waiting on the owner's go: `deploy/kasm/install_kasm.sh` (3389 is now free), the auto-guard A/B choice, and the corpus read-only mount paths.

### From: 2026-10-02 04:15 EDT – ongoing — casevault catalogued; messaging sources placed in their casevault home (owner 03:54)

- [ ] Owner sign-off on the copy list (10 SMS XML, 10 call logs, 2 Facebook exports of the owner–Katrina thread with 536 attachment files; ~11.7 GB), device slugs, Facebook export ids, and which Google Voice Takeout tree.
  - _Unclear (2026-10-02 triage):_ Placement tooling built and SMS backups copied into casevault 12:36Z (line 3965) but no line records the owner's sign-off on this copy list

### From: 2026-10-02 08:15 EDT – ongoing — owner's messages into Probata through Proffer (continuation of the overnight import)

- [ ] **Owner:** add 8102594380 as a confirmed phone of Matthew S. Salem ("Matt's current number", owner 08:40). The agent's write through the case-identity API was refused by the session's auto-mode classifier. Do it on the Workbench case identity page, or allow the write.
- [ ] **Same-device duplicate removal:** `message_dedupe_workflow` is written, not committed (parent session commits it, applies `probata/scripts/2026-10-02-message-dedupe.sql` as platform_dba, deploys). Then a dry run without `expected_copies`, owner review of its plan count, and the live run with the same `dedupe_id` and that count. Run the dry run while no import is committing (its deletes hold the projection validator's lock per step).
- [ ] **Owner:** the 2 stray `source_version` rows from the terminated mistaken derive stay for now (owner: leave).

### From: 2026-10-02 08:45–09:25 EDT — probata-db: `casebible` gets its own login, `ai` password rotated (owner option A, 08:49); Docstore follows the Vestigia rename

- [ ] Whoever created `investigation_test_advocatio_20261002` (owned by `ai`, today): the old `ai` password no longer works; the new one is in `~/.secrets/probata-db.env`.

- [ ] Confirm the 2026-10-03 08:15 UTC nightly build of `propria-docstore` succeeds with the new COPY path.

### From: 2026-10-02 08:38–09:00 EDT — automatic pre-deploy guard in coolify-write (owner 08:38, auto-guard A)

- [ ] The installed plugin copies follow the shared checkout `E:/AI_Workspace/plugins`. It still holds another session's uncommitted `github_app_webhook` work, so it trails `origin/main` until that session rebases.

- [ ] Each guard run copies the container layer again, about 60 MB today, into a new `~/rescued/<stamp>/`. The owner decides whether repeated identical rescues should be de-duplicated, and when old ones go to `to_be_deleted`.

### From: 2026-10-02 08:39–10:30 EDT — Kasm Workspaces CE live on ovh-files (P-1 steps 2–4, 6; owner GO 08:39)

- [ ] **ContextForge:** raise the `coolify-write` gateway's tool timeout above about 60 s, so guarded deploys report their real result.

- [ ] **Owner:** rotate the Kasm `admin@kasm.local` password. My `pgrep -af` printed it from the installer's argv into the transcript (transcript only, not git).

- [ ] Corpus read-only mounts: the owner names the paths.

### From: 2026-10-02 12:46–13:12 EDT — probata-db: every caller has its own login; `ai` refuses network logins (owner 12:46)

- [ ] 49 refusals on `context.activity_execution` (12:48–13:30 UTC) came from a login the old log lines do not name; the engine's own code only reads and inserts there, which `platform_runtime` may do. A repeat now names the login.

- [ ] Something tries to log in as `postgres` (no such role), 10 times today. The next attempt names its address.

- [ ] llm-probe answered one request with 500 after the restart (`AdminShutdown` from a pooled connection the restart closed); later requests are 200. Its pool does not check connections before use, so every database restart costs one failed request per pooled connection.

### From: 2026-10-02 12:00–16:30 EDT — scrambled files in the vault: survey, twins, quarantine list (owner 14:41 "Quarantine the messed-up files")

- [ ] Look for the 949 no-twin files elsewhere (export zips, D:/F:, Drive, OneDrive) by sha1 and name.

### From: 2026-10-02 12:00–17:00 EDT — HTML parsing tool for Facebook and other files (owner 11:59 "We need an HTML parsing tool", 12:00 "find the best one")

- [ ] iMessage HTML export (messages inside a JavaScript string) needs its own template (sibling: Case Bible `elt_imessage_html_v3`).

- [ ] Routing a Proffer run to a Python HTML tool needs a fourth execution path; an owner decision, not built.

### From: 2026-10-02 19:00–19:55 EDT — catalog registry, whole-bucket listings, R2 nothing-lost proof

- [ ] **R2 nothing-lost proof:** load the R2 listings (`/data/consignatio/listings/r2-all-20261002/`, running) into `raw_duck.bucket_objects`; match each R2 object to B2 by hash (R2 MD5 = a B2 content MD5 with B2 SHA-1, or SHA-1 from `casebible-r2-hasher` for the rest); deliver three lists (safe / missing / deliberately excluded) for owner sign-off before R2 is released.
- [ ] Owner: record a "moved to quarantine" disposition in the catalog for the D:\Backup (10,811) and F: (9,207) zero-filled files? (flag and null hash already set 09-13).
- [ ] Owner: hosted Intake UI — A rebuild/release from current code (default), B also make it a Coolify app, C also turn on content search.
- [ ] 33 registry rows are `unknown` status (mostly `inventory.*`, `llm_eval.*`, `media.*`, `knowledge.*`): classify.
- [ ] Owner: F: zero-filled files (9,206 in the ledgers, 9,207 in the 09-13 count) have no rows in the catalog (only F:'s 8,839 empty files do), so no quarantine status could be recorded. Insert occurrence rows for them, or leave F: in the receipts only? (D: done: 10,811 rows `integrity_status = moved_to_quarantine`.)
- [ ] `casebible/tools/scrambled_survey_20261002_candidates.sql` / `_classify.sql` (another session, today) read `raw_duck.vault_objects`, now renamed; a rerun fails until it is pointed at `vault_objects_20260916_r4` or `bucket_objects_current` — and `scrambled_objects_20261002` was keyed against the pre-dedupe listing (keys were existence-checked on B2), so review before reuse.

### From: 2026-10-02 20:30–21:45 EDT — toolbox, contacts, extraction, Surreal, workers (Claude Code · Opus 5.5)

- [ ] **Contacts import (`contacts_import_workflow`, live on proffer-worker 16ae62b6): fix, then dry run again.** Dry run `contacts-import-20261002-dry1`: 269 contact files, 6,155 contacts, 514 people, 970 numbers. It failed at the people step: grouping by shared number chained into one person over the 20-identifier limit. Also 6 of Matt's Google contact CSVs failed to fetch (spaces/parentheses in their keys), and contacts inside Takeout ZIPs are not listed. Fix in progress (agent `workbench-imported-mobile`, worktree `contacts-fix-20261002`). Owner goal: no non-aliased numbers from his or Katrina's backups.
- [ ] **Extract + Send to Surreal (owner 20:54–21:04):** conversation checkboxes on desktop and mobile. "Send to Surreal" writes the selection to surreal-case. "Extract" asks which extractors to use (one or several) and shows results by extractor, read-only on mobile. Extraction also starts automatically after an import is accepted. Only Go + kimi-k3 exists today; LangExtract, Semantica and CocoIndex are being added as selectable Activities (agent `extract-and-surreal`).
- [ ] **Cloudflare Workers (owner 20:50):** format sniffing (30,048 `.txt` and 9,308 `.html` AI-chat candidates), ZIP member listing (61 AI-chat ZIPs, 7.46 GB) and B2 hash backfill, each called from a Temporal Activity and writing to the catalog (agent `cf-workers-catalog`). An R2→B2 copy Worker comes after the missing-from-B2 list. Owner to decide keep/remove for the stale Workers `r2-explorer-template`, `bitter-bonus-00c1` and `casebible-sha256-backfill` once the caller check reports.
- [ ] **Re-sync ContextForge `atomic-tools`**, so its single MCP entry's description reflects the 55-tool catalog (tool-runtime deploy `u7k4zdtw4n1fn0pebnmeqtmf` is live).
- [ ] **Owner choices from the tool registration:** where the neural chunkers run (tool-runtime with a model stack, or through temporal-worker); the XML sanitizer v2's home (move the SQL into Probata, or mount it).
- [ ] **Docstring back-fill:** 41 legacy tools still have a typed `description=` (`LEGACY_EXPLICIT_DESCRIPTION` in `probata/tests/test_tool_descriptions_from_docstrings.py`). The owner's rule (root `AGENTS.md`, 20:44): docs come from docstrings going forward, with the back-fill done later as its own task.
- [ ] **Install the `propria-toolbox` plugin** (pushed 679901d): `claude plugin install propria-toolbox@propria-plugins`; Codex also needs an `atomic-tools` connector in `~/.codex/config.toml`.
- [ ] **Commit and deploy, waiting while imports run** (one proffer-worker deploy for all of them): AI-chat topic chunks (worktree `ai-chat-topic-chunks-20261002`; needs a CHECK SQL, temporal-worker mounts/env and `VOYAGE_API_KEY_FILE`); the repair toolkit (worktree `repair-toolkit-20261002`; owner choice: register repaired files on the same source version, or accept the child-run chain); the same-device dedupe (above).
- [ ] **Super Index automatic cycle (worktree `superindex-auto-20261002`): owner decision before deploying.** It runs on ovh-files, never the desktop, and its worker takes up to 4 GB on a host with ~7 GB free. The owner dismissed the deploy question at 21:42. Decisions A–G are in the agent's report (IntakeCorpus AI-chat chunks, deletion retirement, first-run duration, RAM, catalog growth, summary policy, legal vector).
- [ ] **Chunk publish should create its own collection:** `ProfferChunks20261002` was missing, so the chunk stage failed imports until it was created by hand at 21:06 with `ChunkStore.ensure_collection()`. The publish Activity should call `ensure_collection()` itself.

### From: 2026-10-04 Family Court recovery preservation

- [ ] Register the 15 exact-version preserved ZIP originals through a tracked metadata-only recovery catalog adapter, with independent catalog readback and explicit projection status. Separate ledger/view design awaits owner confirmation; preserve the September 20 global generation. See LOG.md, 2026-10-04 18:21 EDT. Byline: Codex, 2026-10-04.
