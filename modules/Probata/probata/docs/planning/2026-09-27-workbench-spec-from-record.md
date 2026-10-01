> _Byline: Claude Code · Opus 5.5 (subagent "Workbench spec from the record") · 2026-09-27. Moved from a session scratchpad into the repo 09:40 EDT; content unchanged._
> _Byline: Claude Code · Opus 5.5 · 2026-10-01 (P-09, PR-07, DF-01 cells: Weaviate first built, scheduled, deployed)._
> _DF-30 / O-07 rows: Claude Code · Opus 5.5 (agent `machine-auth`) · 2026-09-28._
> _Build pass: Claude Code · Opus 5.5 · 2026-09-27 11:15 EDT (agent `workbench-build`). Items built, deployed and audited are marked DONE with their commit on `main`; deploys `zh8trbgku6sq0wohrkos0j0t` (`5c1e830d`) and `bj7pdt7shdwkg75vwk2x5ele` (`f35cbd87`), both finished and healthy._

# Probata Workbench — what it must be and do now (reconstructed from the record)

> _Byline: Claude Code · Opus 5.5 · 2026-09-27 00:40 EDT. Read-only reconstruction for the live audit and the builders. It finishes a draft that a stopped agent began in this scratchpad. Every row taken from that draft was re-checked against its source; six rows were corrected (P-04, SRC-01, SRC-10, R-05, R-13, OD-10) and the late-evening record was added._

## Read first

1. **Two sessions now hold the Workbench.**
   - At 23:31 EDT the owner had the Probata Workbench task handed to session `d738e882` ("Fix stale Probata checkout path in routers").
   - At 23:59 he told that session: "The entire probata … surface right now. And intake … All of that is … yours right now" (claude:d738e882 2026-09-27T03:59:56Z).
   - This parent session (`5fb0ec97`) is also running `workbench-live` and `workbench-spec`. Coordinate before anyone builds.
2. **The live state after 23:16 EDT is not in the record.**
   - The archive merge `0ffb5b96` added a `tsnet-front` sidecar. Coolify stopped the whole Workbench about 10 minutes after each deploy, at 01:49Z and 03:01Z (commit message of `194a3603`).
   - Owner, 23:10 EDT: "Probata is not accessible at all". At 23:16 he pasted "HTTP ERROR 502".
   - The revert `194a3603` was pushed at 23:15 EDT. At 23:27 the owner gave feedback on a screenshot of `/review?mode=TEST`, so a page was viewable then. No deploy after the revert is recorded.
3. **The owner's newest direction on the surface (23:27–23:40 EDT, session `d738e882`):**
   - The whole app may be restructured. "Get rid of the tabs."
   - Files in the stores must be seen, moved, viewed and changed.
   - One common stack across every module.
   - He declined to confirm that a permanent right rail holding Repair is forbidden, and asked for a reconciled proposal (§1I).

## How to read this

**Sources read**
- Probata docs: `docs/PURPOSE.md`; all 8 files in `docs/decisions/`; `docs/DECISION_LOG.md` D-108…D-161 and D-154R…D-160R; the ADR index and ADR-0061.
- The whole ledger `docs/planning/2026-09-20-TODO.md`: 889 lines on local `main`, 879 on `origin/main`. Lines 1–846 are identical on both.
- `docs/planning/2026-09-27-TODO.md`.
- `docs/pending-review/`: the rethink, the layout diagnosis, metadata/entities, the repair builder, D02 and `d05/`.
- `docs/reviews/` for 2026-09-13, 09-23 and 09-24; `docs/design/2026-09-20-review-message-surface-handoff.md`.
- Workbench docs: `AGENT_MEMORY.md`, `web/AGENTS.md`, `web/README.md`, `web/COMPACT-SUMMARY-2026-09-24.md`, `api/README.md`, the design-mockups memory, and the ADR-0061 spec.
- Workbench code where a status depended on it: the router, nav, `main.py`, the Review component, `flags.py` and `deploy/workbench.yaml`.
- Propria root: `AGENTS.md`, `SURFACE-DESIGN-CONTRACT.md`, `docs/decisions/*`, the D12 packet and `docs/COMPACT-SUMMARY-2026-09-27.md`.
- Consignatio: `docs/receipts/PIPELINE-HISTORY-2026-09-18.md` and `docs/URGENT-TODO.md` (`origin/main`).
- Git history: the archive merge `0ffb5b96`, `94fb0a3a`, `194a3603`, and which deploy commits contain which merges.
- Docstore (read-only): health, two searches and the ADR list.

**Owner voice, 2026-09-19 → 09-27 00:03 EDT**
- Codex: 72 genuine owner turns (192 sub-agent turns excluded).
- Claude: owner turns, queued turns and tool-rejection replies from every Propria session active in the last 7 days. 432 Workbench-relevant turns were read.
- Extraction SQL and digests are in this folder: `claude_owner.sql`, `claude_owner2.sql`, `claude_owner_late.sql`, `wb_owner_filtered.txt`, `codex_user_turns.txt`.

**Citation format**
- A bare path is relative to `modules/Probata/probata/`. `Propria/` and `Consignatio/` prefixes mean `E:/AI_Workspace/Projects/Propria/` and `…/modules/Consignatio/`.
- `TODO:n` = `docs/planning/2026-09-20-TODO.md` line n. `TODO27:n` = `docs/planning/2026-09-27-TODO.md` line n. `LOG:n` = `docs/DECISION_LOG.md` line n.
- Sessions are cited as `claude:<id8> <UTC>Z` or `codex:<day>/<rollout-id8> <UTC>Z`. Local times are EDT.
- Session names:
  - `158b0721`/`77aa963a` = "Control surfaces, temp portal, xplorer intake"
  - `f3081993`/`a235e3bd` = "Missing payload ingestion workflow"
  - `b9db6641`/`5ca9789c`/`01f95b9a` = "Repository merge handoff"
  - `1955af71` = "Intake app"
  - `fa70bdb2` = "Family Law Toolkit and Advocatio integration"
  - `d738e882` = "Fix stale Probata checkout path in routers" (now holds the Workbench)
  - `5fb0ec97` = this parent
- `‡` = in the Workbench builds of Propria `98b3f9c8` (09-27 01:08–01:38Z) and `667233af` (02:46–02:50Z). Coolify stopped the app about 10 minutes after each build, and no live state after the 23:15 EDT revert is recorded.

**Status words.** done+deployed · merged-not-deployed · built-not-merged · proposed · open. "Deployed" means a Coolify deploy that included the change is recorded. Live proof is stated separately.

**Caveats**
- Docstore health returned `ok:false`: sync `5356f93f` failed at 01:02Z, which is the `data:image/` embed bug. Documents written after about 21:00 EDT may not be searchable.
- `.remember/` summaries are machine-written and are used only as leads.
- The Coolify JSON in this folder was fetched by the prior agent between 22:15 and 22:51 EDT and is used here as data. I made no Coolify call.
- I could not find the owner's verbatim ~20:00 EDT "Weaviate first" and "extract → confirm → commit" words in any `~/.claude/projects` log. They are cited from the ledger (`TODO:757-759`, `784-786`) and D-161 (`LOG:348`), which record them as owner rulings (C-12).

## 0. The frame everything is measured against

**Goal (D-159, owner 2026-09-22 19:54).**
- Six steps:
  1. Open a file or folder through an index.
  2. Verify relevance.
  3. Make sure it has a hash.
  4. Pick a parser or extractor and get it into context and the analysis platforms.
  5. Preview the result.
  6. Fill gaps and missing context.
- Sources owns steps 1–4. Review is steps 5–6 plus accept/reject, full screen.
- Machinery is a drawer or its own screen.
- Sources: `docs/decisions/2026-09-22-the-six-steps.md:29-33`; `docs/PURPOSE.md:28-35`; owner claude:f3081993 2026-09-22T23:53:59Z.

**Pipeline order (newest).**
- Extract → searchable in Weaviate → owner confirm → commit to PostgreSQL.
- The Go engine orchestrates, and every step is a Temporal activity.
- Sources: D-161 `LOG:348`; `TODO:757-759`, `TODO:784-786`; `Consignatio/docs/receipts/PIPELINE-HISTORY-2026-09-18.md:21-24`.

**What exists in code**
- The nav is Desk `/`, Sources `/sources` and Review `/review` (`modules/workbench/web/src/surfaces/primary/navigation.ts:7-29`).
- Twelve older routes stay reachable off the nav, plus the `/evidence/preview` alias (`modules/workbench/web/src/router.tsx:33-50`): classification-test, copilot, evidence-queue, intake, knowledge, matter, records, repairs, runs, schemas, surreal, tools.

### Leads from the brief, checked against the record

| Lead | What the record shows |
|---|---|
| D-154R…D-161 | D-154R…D-160R were recorded 09-09/10 and recovered on 09-26 with an R suffix (`LOG:340-347`). Asked which stand, the owner said "What the fuck are they?" (claude:77aa963a 2026-09-26T23:29:37Z). At 19:51 he answered D-154R and D-155R in substance; both concern memory and docs, not the Workbench (claude:77aa963a 2026-09-26T23:51:27Z). D-160R ("workbench for probata, intake for consignatio") is unanswered. D-161 (`LOG:348`) makes first-party projection a set of Go Temporal activities. |
| ADRs 96–98 | These are Docstore ADRs, created tonight, with no file in any repository (local `TODO:883`): 0096 forked agent tools live in the private marketplace; 0097 case PII never enters git; 0098 one shared memsearch watcher. None concerns the Workbench. Repository ADRs stop at 0062. |
| "4 decisions pending" | Listed at `TODO:77`. The owner answered at 19:29 (claude:77aa963a 2026-09-26T23:29:37Z): one shared reader "including legal"; ignore the ignorant-agent question; n8n or change detection "tracked through temporal"; the old `/data` copy, "Yes … clean it". At 19:33 he confirmed the outbox: "Implement it." `TODO:825-826` records two of the answers; `TODO:77` still says pending (DD-07). |
| Spine choice, option 1 vs 2 | `347bf0a2` (21:12) kept the spine write in Python as "the only one available". `c1fe683a` (21:35) cites the owner ruling "the engine writes; Python reports" (D-161) and derives the grants that needs. Resolved: the Go engine writes. |
| platform_runtime INSERT gap | Confirmed by `347bf0a2` and `c1fe683a`. **Closed 2026-10-01:** the minimal set plus UPDATE on the four thread tables (OD-07) is in the snapshot and applied live, read back. |
| First-party ingest broken on PG 18.1 | Confirmed: D-161 and `docs/decisions/2026-09-26-first-party-projection-is-a-temporal-activity.md:16-22`. 18.1 is only the PostgreSQL version of the proof runs. |
| ELT registration of message formats as a critical blocker | The machine summary recorded it (`Propria/.remember/today-2026-09-26.done.md:130,177`), then "ELT verified—extraction works" (`…:151,198`). Receipt `9e7904c7` (21:11) records the ELT as registered, run, with five templates (PIPELINE-HISTORY `:8-16`). **Resolved** per the newest record. |

## 1. Requirements

### 1A. The whole surface

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| S-01 | Every element names the step it serves. An element that serves none is not on the surface. | all | `docs/decisions/2026-09-22-the-six-steps.md:29` "If it serves none, it does not go on the surface." | **open.** The Desk `/` serves no step (C-01). Review keeps Overview, Source records, Lineage and Attempts as tabs (`web/src/components/sbv/proffer-operator-preview.tsx:49-67`). |
| S-02 | Run status, receipts, package identity, storage destination, lineage, attempts and tool catalogs are never primary content. They go in a per-run drawer or on their own screen. | 5 | `the-six-steps.md:33`; `docs/pending-review/2026-09-21-intake-review-module-rethink.md:55` (ratified "Technical details" drawer) | **open.** The machinery sits in "Overview", one of four always-visible tabs (`proffer-operator-preview.tsx:67`). No drawer is built (`TODO:634`). |
| S-03 | An unavailable or unvalidated state is one small flag on the item. No banners, no caveat paragraphs. | all | rethink `:57`; memory `one-flag-no-disclaimers` | **done+deployed** on the Sources search modes (`TODO:596`) and the Review event notice (`TODO:635`). No whole-app audit is recorded. |
| S-04 | The Test/Live switch appears once, in the top bar. "REAL" never appears on screen. Test data never enters a real matter. | decision | `TODO:364` "the non-test side is 'Live'"; `Propria/docs/SURFACE-DESIGN-ADOPTION-REGISTER-2026-09-12.md:55` | **done+deployed** (`TODO:613`, `TODO:172` "Test shown once"; mode from the durable matter id `ca442dd`, `TODO:231`). Live mode answers 503 "REAL matter identity is not configured" (`docs/reviews/2026-09-23-workbench-p0-deployment-receipt.md:23`); see OD-05. |
| S-05 | A finished Test run can be promoted to Live by re-binding it to the live matter with a receipt, never by copying. | decision | `TODO:364` | **open** (OD-05). |
| S-06 | One user, one case. Plain labels, compact density, no unexplained abbreviations. | all | `modules/workbench/AGENT_MEMORY.md:35-36` | Standing. |
| S-07 | Finish and smoke-test one path before adding a nav destination. Never advertise a disconnected surface. | all | `modules/workbench/web/AGENTS.md:13-14` | Standing. The nav has three entries. |
| S-08 | The stack is React + Vite, TanStack Router/Query, Storybook and Glide Data Grid (alpha24, owner-approved), with Tauri later. The stack is shared across every module. | none | D-108 `LOG:20`; D-156/157 `LOG:336-337`; owner "already did" (claude:158b0721 2026-09-21T02:45:10Z); "migrating to a common stack … Every module" (claude:d738e882 2026-09-27T03:40:11Z) | **done** in the Workbench (`web/package.json:18,28`). Cross-module convergence is **open**. |
| S-09 | Charts: the mood strip uses vis-timeline, the conversation view Glide, summary charts Recharts and frozen reports Evidence.dev. Build them as reusable blocks. | 5 | `web/COMPACT-SUMMARY-2026-09-24.md:43-47` ("The sounds awesome"); "I wanna reusable UI tool" (claude:5ca9789c 2026-09-25T02:07:14Z); D-129 `LOG:309` | **open.** The mood strip (`d662f854`) and the category strip (`fbfe7701`, `7ac1201b`) are merged but used only in Storybook. Recharts is not installed (`web/package.json`). There is no platform Evidence.dev project (D-129). |
| S-10 | Timesketch is available as a timeline view. Progress, including LLM operations, streams to the surface. | 5 | `AGENT_MEMORY.md:37-39` | **open.** The event stream is fixed (`TODO:635`). The Timesketch step does not run (`TODO:146`). Under S-02, progress belongs in a drawer or on an Activity screen. |
| S-11 | Browser checks never run on the owner's desktop. Journeys run in the ovh-files devbox, or the check goes through the API. | none | `web/AGENTS.md:30-39` | **done** (`10c5d82`, `55f21276`, `01419f0f`; devbox run 4/4, `TODO:99`). |
| S-12 | Every submitted source stays visible across navigation and refresh. A wait never shows as active work. Open, Resume decision, Hold/Cancel and Back are durable commands. | 5 | `docs/decisions/2026-09-12-intake-workflow-visibility-and-repair-gate.md:5,55` | **open.** The run list has Reviewable/Failed/All chips (`TODO:234`). Hold, cancel and exact retry are not built (D05-C06 `not_started`, `docs/pending-review/d05/D05_LINKED_IDS_AND_CLAIMS.json:33`). **Cancel DONE `99f8e3c6`** (2026-09-28): Temporal cancel from Review, receipt in the run's history, lifecycle `cancelled`; live proof on TEST run `sODdhBY5…`. Hold and exact retry remain open. |
| S-13 | No gate the owner did not ask for. Clean files reach context without a click; only exceptions stop. | 4/5 | "too many uneeded and not asked for gates" (claude:158b0721 2026-09-21T01:52:45Z); `docs/decisions/2026-09-20-bulk-intake-owner-requirements.md:40` | **open.** Every batch item parks at the preview decision (`TODO:385`). See C-02 and C-06. |
| S-14 | Nothing is hand-made. Containers, networks, attachments and host edits are declared in repository compose files and deployed through Coolify. | none | "Nothing is supposed to be created that way. Ever." (claude:5fb0ec97 2026-09-27T02:52:32Z); `TODO27:27` | **open** for the Workbench. Its compose depends on two external networks (`deploy/workbench.yaml` on `origin/main`): `probata`, created in host prep (`:21`), and the hand-made `propria-edge` with the fixed address 10.201.0.4 (`:166-173,196-201`). The edge fix is carried at `TODO27:42`. |

### 1B. Sources (steps 1–4)

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| SRC-01 | Sources replaces Intake as the front door: a B2 browser in one viewport, folder tree left, files centre, metadata right. | 1 | `PURPOSE.md:51`; rethink `:44`; `TODO:563-567` | **done+deployed** from `9158ffc` (`TODO:630,639`). `/sources` answered 200 on 09-23 (`p0-deployment-receipt:22`). Owner: "at least mostly functional" (claude:f3081993 2026-09-22T23:39:40Z). The Desk still links to `/intake` (DF-16). |
| SRC-02 | The app reads, writes and deletes on B2, because its job is to sort, move and repair. | 1/4 | "WHY … READ ONLY WHEN THE PURPOSE … IS TO … SORT" (claude:158b0721 2026-09-20T16:26:07Z); `TODO:22` | **done**: a whole-bucket key ending `…0007` (`TODO:252-253`). |
| SRC-03 | Stores and roots are configuration (`OBJECT_STORES_JSON`, `SOURCE_ROOTS_JSON`), with the B2 vault as the default root. | 1 | `TODO:21,184-195` | **done+deployed**, live 09-20 13:52 (`TODO:195`). |
| SRC-04 | Those two values live in one bind-mounted JSON file, not as literals in compose files. | none | "cant just make a json config in the bind mount" (claude:f0898ff7 2026-09-20T22:36:27Z); `TODO:18` | **open** (queued). |
| SRC-05 | Each file carries exactly one state mark: not processed, decoded, in context or failed. Nothing is inferred from file names. | 1 | `PURPOSE.md:51`; `TODO:569-573` | **done+deployed.** Marks key on `source_ref`, so they break when a file moves (DF-19). |
| SRC-06 | Metadata and unit membership are in view: name, size, modified time, sha256, detected format, decode counts, runs and catalog provenance. Units show a badge, and a folder can be marked as a unit. | 1/2 | "i need the meta data … in view and someway to signify its a unit" (claude:158b0721 2026-09-22T13:00:10Z); rethink `:46-47` | **done+deployed** (`TODO:575-590`). Hand marks live in a Workbench JSON file, not the catalog (`TODO:577-581`). The unit-join fix `d3490220` (09-22) first shipped in the ‡ builds. |
| SRC-07 | Recorded and recognised unit patterns are auto-selected. A Takeout is a supervised proposal: the owner confirms account, parts and gaps. | 1/4 | bulk-intake `:38,42,62`; "recorded and idenitfied patterns are auto selected" (claude:a235e3bd 2026-09-21T01:16:41Z) | **open.** The Takeout proposal and its `confirm:true` gate are built (`TODO:582-584`). Auto-selection is not. |
| SRC-08 | Search ships with Sources: one box, four modes. Names and paths come from the catalog, contents from the CocoIndex super-index, meaning from Weaviate, relationships from Surreal. Results are files or units carrying their state marks. | 1 | "the catalog and corpus coco super index and surreral weiviate search gets built in" (claude:158b0721 2026-09-22T13:06:27Z); codex:20/01a0c120 2026-09-21T00:01:28Z; `TODO:367` | **open.** Names and paths work. Contents and meaning go through Intake `/filesystem/search`; relationships is gated; an unreachable mode shows a flag (`TODO:592-596`). Catalog hits carry `source_links_verified: false` (`docs/reviews/2026-09-23-probata-p0-function-checkpoint.md:30`). |
| SRC-09 | Search reaches the Intake (Consignatio) CocoIndex tooling through a proxy. Probata owns no index, and its features must lift into Xplorer later (direction C). | 1 | rethink `:65-71`; "needs to use te intake coco search tooling" (claude:158b0721 2026-09-22T13:09:19Z) | **done** as designed (`/api/intake/discovery/*`). |
| SRC-10 | Search is usable: a search button with progress, a term that still works after a filter is picked, search inside zips, and scope options (current folder, up a level, subfolders, everything). | 1 | codex:20/01a0c120 2026-09-20T23:57:53Z "maybe a search buttoon and progress"; 23:58:17Z "search in zips?"; scope list (claude:1955af71 2026-09-22T22:55:18Z, said in the Intake-app session) | **open** for Sources. Xplorer Intake has catalog name search that includes ZIP members (Docstore `note:d03_search_source_open_gate_20260924`). |
| SRC-11 | Look at a file before anything runs: the Messages view for a backup (decoded SBV-style before ingest), otherwise the picture, PDF or text. | 2 | `PURPOSE.md:31,51`; `TODO:387-395` | **done+deployed** (decoded viewer 09-21, Sources preview 09-22). Other formats are not proven in a live journey (`p0-function-checkpoint:24`). |
| SRC-12 | Every file shows its hash (sha256 or "not hashed") as a context fingerprint, with no custody wording. | 3 | `PURPOSE.md:32`; `the-six-steps.md:31`; D-152 `LOG:332` | **done+deployed**: a preview-only hash from `/api/proffer/source-inspection` (`p0-function-checkpoint:25`; `TODO:586-587`). |
| SRC-13 | Tools stream from object storage, with a range-read to sniff and a streaming hash. Custody is the streamed digest plus the object version id. A local copy exists only while a repair needs it, then it is removed. | 3/4 | "if it needs to be … repaired, it needs to be there" (claude:f3081993 2026-09-22T23:04:37Z); `TODO:649` | **open.** Decided (option A), not built: the gateway still seals whole copies (`TODO:643-649`). |
| SRC-14 | The format is detected, never declared. The recommended handler is shown with a working override before Process; the override is not a gate. | 4 | `PURPOSE.md:33`; `TODO:366` "i get to select to override auto selection" | **done+deployed** (`TODO:588-590`). |
| SRC-15 | One Process button: a file starts one run, a folder starts a batch. Nothing starts on page load. | 4 | `PURPOSE.md:51`; `TODO:598-601` | **done+deployed.** No live batch run is recorded (`TODO:385`; `p0-function-checkpoint:26`); see OD-13. |
| SRC-16 | Process works wherever a file sits. Unit marking, the move to a canonical home and Process are independent and can happen in any order. | 1/4 | "I don't have to time to nearly sort. Before Ingest" (claude:158b0721 2026-09-22T14:48:34Z); `TODO:603-606` | **done** in the design. Stable file identity across moves is open (`TODO:607-611`). |
| SRC-17 | Files in the stores can be seen, moved, viewed and changed. The move to a canonical home needs a manifest first, a verify after the write, and Takeouts moved whole. | 1 | bulk-intake `:48`; "see shit, move shit, view shit, change shit" (claude:d738e882 2026-09-27T03:38:26Z) | **open.** Sources has no move, rename or delete action. The panel only says a move is possible (`web/src/components/sources/source-metadata-panel.tsx:241`). |
| SRC-18 | Zips are read in place and extracted on demand; a Takeout's parts show as one unit. | 1 | bulk-intake `:35`; "we should be able to read from the zip" (claude:a235e3bd 2026-09-21T01:14:06Z) | **open.** The rclone archive read is proven (bulk-intake `:53`). The engine ZIP inventory is merged (`235001b5`, 09-26 19:14). There is no Sources UI. |
| SRC-19 | Zip members and already-extracted trees appear side by side with the basis of each match. | 1 | bulk-intake `:46` | **proposed.** |
| SRC-20 | Before extracting, intake looks the bytes up in the catalog, reuses a prior extraction and moves it to its permanent home. | 4 | "reuse the extracted data we have" (claude:b9db6641 2026-09-24T17:49:51Z); `TODO:691` | **open.** The design questions go to the owner first (`TODO:694`; OD-08). |
| SRC-21 | When no parser exists, a button calls an agent to guide parser creation. | 4 | bulk-intake `:39,64`; claude:a235e3bd 2026-09-21T01:16:09Z | **open.** The repair UI renders nothing for `agent_available` (`TODO:60`). |
| SRC-22 | Probata's search is its own path. It never reads Semantica's `forensic_*` collections. | 1 | "This is supposed to be entirely separate from semantica" (claude:fa70bdb2 2026-09-22T23:08:00Z) | **open.** The meaning mode proxies Intake (`TODO:594`). Nothing records Probata's own collection. |

### 1C. Processing behind Process (step 4)

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| P-01 | The Go engine orchestrates, and every stage is a Temporal activity. The flow is extract → confirm → commit, and one activity owns each write. | 4 | D-161 `LOG:348`; `docs/decisions/2026-09-26-first-party-projection-is-a-temporal-activity.md:10-12,32-34`; `TODO:784-786` | **done** for the normalized and entity lanes. **Open** for first-party projection (P-14). |
| P-02 | A signature registry picks one handler per signature. DuckDB ELT handles every signature it can; SBV/Go decoders cover the rest and act as backup after a logged failure. | 4 | D-149 item 9 `LOG:329`; PIPELINE-HISTORY `:8-16` | **done**: five ELT templates; `handler_selection_store.go:304-331` (receipt `9e7904c7`). |
| P-03 | Stream everything. An SMS backup goes through SBV derive (media out, per-thread NDJSON chunks, a manifest) and then DuckDB `ndjson_v1`. Derive runs by itself when an SMS backup is picked. | 4 | "have sbv extract it and split out media" (claude:158b0721 2026-09-20T22:48:33Z); "that is the whole … point" (…2026-09-21T03:52:38Z); bulk-intake `:36` | **done+deployed**, live-proven (`TODO:378-384`; 584 MB in 4 min 27 s, `TODO:222`). |
| P-04 | **Corrected.** Where derived output lives is configuration only (`DERIVED_ROOTS_JSON`, one mapping function; unset means beside the original). Its final vault home is open. | 4 | `TODO:437,447,507`; owner on the "top-level mirror" wording: "That got changed. Fucking remember." (claude:158b0721 2026-09-21T06:07:02Z; `TODO:405`) | **done** as configuration. The final home is open (OD-10). |
| P-05 | Every route produces the same normalized records, attachments and counts. The derive route is the reference, proven by a golden comparison. | 4 | "the older go and py routes needd to produce same results" (claude:158b0721 2026-09-22T13:05:25Z); `TODO:365` | **open.** |
| P-06 | Calls backups parse and publish. | 4/5 | "ITS BUILT IN TO THE SBV UPSTREAM" (claude:158b0721 2026-09-21T01:40:31Z); `TODO:229,519` | **done+deployed** (`33da837`; calls-only publish, `TODO:397-403`). |
| P-07 | A retry cap with a "cannot succeed, stop" classification. Offer options when there is more than one way forward. Cap concurrency. | 4 | bulk-intake `:37,60` | **open.** Only `execute_structured_elt_activity` is marked, plus a worker cap of 4 (`TODO:354`). The operation ledger does not see an external terminate (`TODO:282`). |
| P-08 | Batch by folder, one item at a time, with per-item status and retry-failed. Only exceptions stop a batch. | 4 | "batching is absolutely part of this" (claude:158b0721 2026-09-21T03:52:38Z); bulk-intake `:40,63` | **open.** The batch workflow and its passthrough are deployed (`TODO:406,508,599`), but items park at the preview decision (`TODO:385`). |
| P-09 | Weaviate first: extraction output is searchable before the owner confirms. Events and records go to PostgreSQL after the confirm. | 4/5 | `TODO:757-759`; PIPELINE-HISTORY `:21-24` | **built, scheduled, deployed 2026-10-01** (PR-07) into `MsgEvents20260918` (OD-06). |
| P-10 | Precommit proposal: one DuckDB bundle per attempt with two digests, and approval bound to the exact attempt. A view backed by committed rows is labelled "Committed read-back". | 5 | `docs/reviews/2026-09-13-proffer-precommit-review-contract.md:20-30,268`; owner "Preview means precommit proposal" (PIPELINE-HISTORY `:91`) | **open.** Gap-ledger G1/G2 (`docs/reviews/2026-09-13-proffer-current-state-plan-and-gap-ledger.md:120-126`); D05-C01/C02/C04/C05 are `not_started` (`D05_LINKED_IDS_AND_CLAIMS.json:28-32`). Its Weaviate and Surreal ordering is superseded (§4). |
| P-11 | Entities are extracted early and can be previewed, refined and re-run. | 5/6 | "It needs to happen at the beginning" (PIPELINE-HISTORY `:99`) | **open**; see G-06. |
| P-12 | Attachments are extracted at parse into named files beside the object, each with its own SHA. | 4 | memory `attachments-extracted-at-parse`; D-149 (5) `LOG:329` | **done** for SMS derive (`TODO:218`). |
| P-13 | A missing payload is recorded as missing at ingest, never as a 0-byte file. | 4/6 | "its even supposed to record the missing … payload" (claude:a235e3bd 2026-09-21T00:57:58Z); `TODO:302-312` | **done+deployed** in the engine and projection (`TODO:633`). No true positive has been observed live. |
| P-14 | First-party context imports into the five-table `working.first_party_context_thread` family. `working.message.id` equals `normalized_record.id`. Identity is required (D-126 sentinels). Proof is live on a disposable schema. | 4 | D-161; decision `:38-49`; "I'm trying to import context. That's our problem." (`TODO:734`) | **built and deployed 2026-10-01** (`955d1bbc`): four Go Temporal activities write the spine and the thread family; proven live as `platform_runtime` on a disposable database. The Live run on `sms-2024-11-24.xml` waits on the go-live identity rows. Detail: TODO 2026-09-30 §2 "First-party context import (D04)". |
| P-15 | A locally uploaded file can take the DuckDB path; `upload://` fails closed today. | 4 | `TODO:176-182` | **open** (OD-01). |
| P-16 | Set `DUCKDB_XML_MAX_BYTES` to 1 GiB in the worker once the routes are reconciled. It is the fallback cap, not the design. | 4 | `TODO:235(1),368` | **open.** The code reads it (default 256 MiB); the worker env does not set it. |
| P-17 | A validator re-reads derived NDJSON and chunks oversized threads. | 4 | `TODO:235(2)` | **done** (`6b46902`; 64 MiB chunks, `TODO:222`). |
| P-18 | Attachment text gets OCR and classification with the existing tools. Speech-to-text is missing. Check the super-index first. | 4 | `TODO:225,235(3)` | **open.** |
| P-19 | The repair gate judges damage from the real preview report. A clean file continues by itself. A real issue shows the issue and a proposal; the owner may override to the original with a receipt. The original is never altered, and a detector failure is not source damage. | 4 | decision 2026-09-12 `:17-24` | **done** in the engine (`TODO:46-47,226`). |
| P-20 | Copies of a message from another format, person or device are corroboration. They are never deduplicated away. | 4 | "Those are not duplicates. That's corroborating evidence." (claude:b9db6641 2026-09-24T13:28:16Z; said of catalog extraction) | **open.** Not reflected in the Proffer skip rules (`TODO:332`). |
| P-21 | All ingest goes through the engine. The Python parsers are a legacy lane. | 4 | "NOTIHNG HITS THE PYTHON FIRST" (claude:a235e3bd 2026-09-21T00:52:10Z) | Standing. The Python first-party writer is the remaining exception (P-14). |

### 1D. Review (steps 5–6 and the decision)

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| R-01 | Review is the file's preview, full screen. It lands on the messages. | 5 | `the-six-steps.md:32`; `PURPOSE.md:34` | **done+deployed**: one 728 px viewport landing on Messages (`TODO:552-554`; layout diagnosis `:96`). It lands on the most-processed reviewable run (`576e542e`, `21a48729`, deployed 09-25 02:22Z/02:29Z). The owner still reports "significant layout issues" (DF-26). |
| R-02 | One viewport: a sticky top strip (name, checkpoints, exception flags, next actions including retry), a left rail of runs, messages in the centre, and a collapsed right rail for package, repair, storage and tools. | 5 | layout diagnosis `:42-48`; "THAT WOOULD BE MUCH BETTER" (claude:f3081993 2026-09-21T03:48:34Z); "--- yes" (…2026-09-21T05:47:34Z) | **open.** The single viewport is done. The machinery is in the Overview tab, not a rail (`TODO:634`), and an always-present Actions panel sits beside the views (R-10). See C-04 and OD-14. |
| R-03 | The run list shows one line per run: the full file name, the parent backup for a derived chunk, Reviewable/Failed/All chips, name search, and failed runs hidden by default. | 5 | "massive amd ahrd to nav" (claude:158b0721 2026-09-21T03:10:53Z); `TODO:234(2)` | **done+deployed** (`eb8d8b5`, 09-20 23:18). |
| R-04 | Show file size, and whether a run is a whole backup or a piece of one. | 5 | "Can't see file names. Can't even see file size." (claude:f3081993 2026-09-22T23:40:41Z) | **open.** Names are fixed (`576e542e`). Size and whole-versus-piece are not recorded. |
| R-05 | The message view is the ported SBV front end, with dense Glide rows as the default and the SBV conversation optional. A Conversation · Table · Media switch; oldest first; server-side search ("N of M"); filters; a calls view. | 5 | "pull in the front end of the original sbv app" (claude:158b0721 2026-09-21T03:10:01Z); `docs/design/2026-09-20-review-message-surface-handoff.md:144-151`; `TODO:521,632` | **done+deployed** (09-21 02:32; Media 09-22 18:40). The code defaults to `"table"` (`web/src/components/sbv/message-browser.tsx:60`). |
| R-06 | No "Details" click per message: the detail panel follows the selection. No reading raw JSON; JSON only behind a per-row disclosure. | 5 | "clicking details on every message is productive" (claude:158b0721 2026-09-21T00:42:17Z); "i have to read in the json" (…03:13:30Z) | **done+deployed** (three-panel browser, `TODO:233`). Whether Source records still shows JSON cards is not recorded. |
| R-07 | Media works in two modes: before processing SBV decodes the base64; after ingest it streams from the derived folder. Photos show inline. | 5 | "Take the … code from there" (claude:158b0721 2026-09-21T06:05:18Z); `PURPOSE.md:34` | **done+deployed** (sha256-verified media, `TODO:403`; decoded viewer, `TODO:392-393`). |
| R-08 | A missing-payload flag on each attachment, message and thread. | 5/6 | "--- yes" (claude:f3081993 2026-09-21T05:47:34Z); `TODO:633` | **done+deployed.** No true positive has been observed live. |
| R-09 | Extra views sit in a More menu that works. An empty view shows one muted line. | 5 | "more for now" (claude:158b0721 2026-09-21T03:27:54Z); "The more button is still completely useless" (…2026-09-25T04:15:33Z) | **done+deployed** (radix dropdown, `TODO:172`). Now superseded in direction by N-02, "Get rid of the tabs". |
| R-10 | Actions are always present: edit context or metadata (append-only revision), choose a parser, choose a repair, re-run. Approve and reject sit at the gate. | 5/6/decision | "Can't modify any … metadata … can't choose a parser" (claude:158b0721 2026-09-25T04:16:28Z) | **done+deployed** (09-25; `TODO:170-172`). Answers applied through a re-run live only in page memory (`TODO:173`). |
| R-11 | Approve/reject is one decision bar. When approval is locked, the bar is disabled with the missing requirement as its tooltip. | decision | handoff (DecisionBar); rethink `:53` | **open.** The gate controls are unchanged (`TODO:172`). |
| R-12 | Show a file's conversations, found by an LLM rather than 30-minute bouts, with model labels. The chunk method can be changed, re-chunking the preview without re-extracting. | 5 | "option to show these bouts … change the chunk method" (claude:b9db6641 2026-09-24T17:49:51Z); `TODO:692-693` | **open.** Design questions go first (OD-08). The bout-review API is built but not mounted (DF-15). |
| R-13 | Flag exact records or chunks for potential promotion ("needs corroboration"). | 6/decision | "Flag it for potential promotion." (PIPELINE-HISTORY `:123`) | **done+deployed‡** as code (D05, merged `bd0acb7b` 09-26 19:08). Flag creation **fails closed with 503** "Proffer flag delegation is not configured" (`modules/workbench/api/app/service/flags.py:22-42`), because no manifest mounts the key. The review verdict is HOLD (`docs/pending-review/d05/D05_INDEPENDENT_REVIEW_2026-09-23.md:126,166`); see OD-09. |
| R-14 | Retry, cancel and resume a failed run from the page, with append-only control receipts. | 5/6 | "no  retry button" (claude:a235e3bd 2026-09-21T00:50:22Z); `TODO:322` | **open.** "Re-run" exists (`TODO:172`). Exact-stage retry, cancel and resume (D05-C06) are not started. **Cancel DONE `99f8e3c6`** (2026-09-28): Temporal cancel from Review, receipt in the run's history, lifecycle `cancelled`; live proof on TEST run `sODdhBY5…`. Exact-stage retry and resume remain open. |
| R-15 | Search across every conversation, not one run, plus a "similar" action (a Weaviate near-object search on the selected record). | 5 | rethink `:53,75` (ratified option A); "vector preview and navigation for probata and intake" (claude:158b0721 2026-09-22T13:15:30Z) | **open** (ratified, not built). |
| R-16 | The event stream connects, and the "unavailable" notice is gone. | 5 | `TODO:635` | **done+deployed** (`9158ffc`). |
| R-17 | "Flag / send on" from the detail panel. | decision | rethink `:53` (ratified) | **open.** No send-to-Surreal action is recorded; see C-03. |

### 1E. Filling gaps and missing context (step 6)

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| G-01 | Clicking any file opens a screen with all of its metadata: every embedded field for every file type, plus every sidecar. Conflicts are flagged, never merged. The original capture time shows with its named source, plus the device make and model. | 2/6 | "I need to have click on it and see a screen" (claude:158b0721 2026-09-25T23:13:02Z); "all metadata for all files, including any available accompanying sidecar" (…23:14:28Z); screenshot timestamps "critical" (claude:fa70bdb2 2026-09-21T06:13:32Z) | **done+deployed, partial** (09-26; `TODO:69-71`). Recorded metadata and sidecars show. Embedded metadata is **not read**: `ExtractEmbeddedMetadata` is a no-op, flagged "embedded metadata not read" (`docs/pending-review/2026-09-25-metadata-context-review-entities.md:50`). |
| G-02 | One shared exiftool reader in tool-runtime, used by Probata, the legal workdesk and Intake. | 2/6 | "A single shared reader, including legal." (claude:77aa963a 2026-09-26T23:29:37Z); `TODO:825` | **open.** Decided (option A), not built. |
| G-03 | Metadata corrections are attributed, append-only overlays. The recorded value is never overwritten. | 6 | metadata doc `:44` | **done+deployed** (overlay applied, `TODO:70`). Not exercised. |
| G-04 | Context review per message: to/about, about the child (yes/no/unsure), relevant. Revisions are attributed and append-only. | 2/6 | "whether or not it's about the child … Checkbox for relevance" (claude:158b0721 2026-09-25T23:13:02Z); metadata doc `:21-25,46` | **done+deployed.** No live write is recorded (`TODO:71`). |
| G-05 | Foreshadowing is a hindsight-only flag in its own table. It never reaches the as-lived view, is excluded by the horizon pre-filter, and is not projected to Weaviate or Surreal. | 6 | "we don't necessarily want to taint the table" (claude:158b0721 2026-09-25T23:13:02Z); metadata doc `:26,48` | **done+deployed**, with a tripwire test. On ignorant-agent visibility the owner said "just … ignore it" (claude:77aa963a 2026-09-26T23:29:37Z). |
| G-06 | Entities with aliases. "Extract Entities" proposes; the owner accepts, rejects, renames, retypes, merges, splits and adds aliases; the workflow commits. Re-extraction keeps the owner's edits. | 6 | "once we run the workflow they can be committed" (claude:158b0721 2026-09-25T23:15:27Z); `TODO:141-148` | **done+deployed** (`c091cae2`, `0f1f230e`; `/api/entities/proposals` 200). No extraction has run; the tables hold 0 rows (`TODO:148`). |
| G-07 | Events are auto-detected, and "Mark as event worth recalling" works on any message. An event is dated by its source record and committed to the timeline. Proposals are staged in `working.candidate_event` (owner 6 = A). | 6 | "an event worth. Recalling" (claude:158b0721 2026-09-25T23:17:28Z); `TODO:76,149` | **done+deployed** (`0f1f230e`). Follow-ups open (`TODO:146,150`): ordering uses `created_at`, source ranges are not written, the Weaviate and Surreal steps are unbuilt, and the Timesketch step does not run. |
| G-08 | Committed entities and timeline are copied at once to SurrealDB and Neo4j through the ADR-0052 outbox, run by Temporal and shown in n8n. | 6 | "entities and timeline to immediately be copied" (claude:158b0721 2026-09-26T12:32:53Z); "And neo4j" (…12:33:10Z); "Implement it." (claude:77aa963a 2026-09-26T23:33:40Z); `TODO:826` | **open.** Decided, not built. |
| G-09 | A missing-payload register: each gap is recorded against its message and backup, unresolved findings show when a file is opened, candidates are checked, and resolutions are append-only. An unchecked file shows "check unavailable" or "not assessed". | 6 | owner-pasted requirement, "need to finish this" (claude:a235e3bd 2026-09-21T00:31:26Z); `TODO:314-320`; bulk-intake `:49` (write path settled) | **open.** Only engine detection exists. |
| G-10 | Where a file is short, show the gap, its leads and a way to attach what fills it. | 6 | `PURPOSE.md:35`; `the-six-steps.md:32` | **open.** Depends on G-09. The repair builder's "find another version" is the only lead path. |

### 1F. Repair (steps 4 and 6)

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| RP-01 | A damaged source is flagged. Beyond "use original", the options are: find another version by file name, wait for more detail after parse, or look in other sources. | 4/6 | "i feel more options are needed" (claude:158b0721 2026-09-20T22:01:36Z); `TODO:240` | **open.** "Find another version" and the wait option ship in the builder. Flagging and "look in other sources" are not built. |
| RP-02 | The repair workflow builder proposes from the file name and signature, then lets the owner compose steps. It validates: registered activities, a type-checked chain, no write to the original, re-entry, destinations that resolve with matching Test/Live, and bounds. It runs on Temporal, with n8n only where a step needs it. The V2 "Open in n8n" is deferred. | 4/6 | "allow me to build out the workflow" (claude:158b0721 2026-09-25T22:25:31Z); "V2 OPTION TO OPEN IN N8N.. DEFERED" (…23:09:35Z); repair doc `:39-40,55-57` | **done+deployed** (`1a69879f`, `e02feca6`). `/api/proffer/repair/tools` answered 200 with 3 tools (tool result in claude:158b0721 2026-09-26T21:32Z). No repair run has been exercised (`TODO:71`). |
| RP-03 | After re-entry, the anchoring run's gate stays open for the owner (owner 4 = A). | decision | `TODO:76,159` | **done** as shipped. |
| RP-04 | "Repaired →" survives Temporal retention: the `repair.reentry` receipt carries the new preview handle. | 5 | `TODO:61` | **open** (engine follow-up). |
| RP-05 | "Repair, then parse the repaired copy" works from the surface. `repair.write-derived` needs `artifact_root` and `dest`. | 4 | `TODO:236` | **open.** Whether `salvage_truncated_xml` covers it is not stated. |
| RP-06 | An agent proposer for signatures the table does not cover. | 4 | repair doc `:24,62` | **open.** |

### 1G. Access, deployment and verification

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| O-01 | On the tailnet, Tailscale is the only barrier and there is no app login. Off it, one Authentik portal lists the user surfaces, with one Probata listing. Each surface has a custom-domain address and a MagicDNS address. | none | codex:23/01a0cd3c 2026-09-23T12:06:55Z "when on the tailnet tailscale is the security bariar"; "Nothing should be blocked by authentic on the tailnet" (claude:5fb0ec97 2026-09-27T03:08:56Z); "custom domain address, as well as a tailscale magic DNS address" (…03:09:37Z) | **done+deployed** on the tailnet (09-23; `p0-deployment-receipt:21-24`). The public route was fixed in `cea8f82`, but it rides `propria-edge` (S-14, DF-28). The owner's click-check of the public login is pending (`TODO:683`). |
| O-02 | Production means production, and every deploy is explicit because auto-deploy is off. | none | Probata `AGENTS.md` "Owner delivery rule"; "remove the watch flag" (claude:158b0721 2026-09-20T21:56:10Z); `TODO:238` | Standing. |
| O-03 | The Workbench, starter and worker build from `Cursedpotential/propria`, base `modules/Probata/probata`, with prefixed watch paths. | none | `Propria/AGENTS.md` "Deployment"; `Propria/docs/monorepo-coolify-cutover-2026-09-26.json` | **done.** Coolify shows the propria repository on main, base `/modules/Probata/probata`, and watch paths `modules/Probata/probata/modules/workbench/**` plus the deploy file (`coolify-workbench-detail.json`). Builds of Propria `98b3f9c8` and `667233af` finished (`workbench-deployments-list.json`, `deploy-fmsm27.json`). |
| O-04 | The health check proves the stores the app actually uses. | none | `TODO:667`; `docs/reviews/2026-09-23-workbench-p0-runtime-health-proxy-receipt.md:7-9` | **open.** It passes: `{"status":"ok","lancedb":true,"object_store":true}` at 2026-09-26T01:08Z, 12:24Z and 21:32Z (tool results in claude:158b0721), after R2 was re-enabled. It still probes the R2 buckets `casebible-sorted`/`nexus`, not the B2 default (C-10). |
| O-05 | The Workbench API tests run in CI, and ruff complexity rules replace the line-count cap. | none | `TODO:738-742` | **open.** **DONE (CI part), `5c1e830d`:** root workflow `.github/workflows/probata-workbench.yml` runs the API suite and the web lint/build/smoke/Storybook; first run 473 passed, smoke 111 pass / 4 skipped. Since the monorepo import GitHub runs only root workflows, so the nested Probata `validate.yml` does not run at all. The ruff complexity rules are **open**. |
| O-06 | The model credential is present in the Workbench env. | none | `TODO:163` | **done** (`NVIDIA_API_KEY`). |
| O-07 | The tailnet identity check admits both user and device identities. | none | "stupid allow for device and user" (codex:20/01a0c120 2026-09-21T00:04:47Z) | **open.** `TAILSCALE_DEVICE_CAPABILITY=propria.mitechconsult.com/cap/workbench` is set on the live Workbench (checked 2026-09-28), but the grant never reaches it: ovh-app's `tailscale whois` shows a null CapMap for ovh-files. Machine callers move to Authentik service accounts instead (DF-30). |
| O-08 | After each surface change, a headless browser on a VPS previews the page, never on the desktop, plus an API read-back. | none | "use headless Chrome to preview what it … looks like" (claude:5fb0ec97 2026-09-27T03:22:11Z, said of the portal); `web/AGENTS.md:30-39`; `TODO:374` | **open** for the Workbench. The devbox run on 09-26 passed 4/4 (`TODO:99`). |

### 1H. Models

| ID | Requirement | Step | Source | Status and evidence |
|---|---|---|---|---|
| M-01 | kimi-k3 on NVIDIA NIM is primary. glm-5.1 is banned everywhere. | none | `TODO:154`; Probata `AGENTS.md` "Model Provider Chain" | **done+deployed** (`5dcec86`; live classify 200, `TODO:160`). |
| M-02 | The kimi-k3 guard picks the mode by prompt size. Empty, junk or invalid-JSON replies count as failures and get one retry. Long prompts retry with thinking on (owner 2 = B). Back off on 429. No silent fallback. | none | "1 yes 2 b …" (claude:158b0721 2026-09-26T12:07:34Z); `TODO:69,160-161`; `0f652746` | **done+deployed.** Option B has not been re-verified live since the deploy (DF-25). |
| M-03 | Entity extraction uses `json_object` and requires every mention to appear verbatim. Two failures flag the batch and write nothing. | 6 | `TODO:148` | **done+deployed.** Not exercised. |
| M-04 | Agents that return through Temporal activities get the same reply check. | none | `TODO:162` | **open.** |

### 1I. The owner's newest directions (09-26 23:27–23:59 EDT) — measure the next build against these first

| ID | Direction | Source | Status |
|---|---|---|---|
| N-01 | The whole app may be restructured; he does not judge it page by page. | "The whole … app can be modified, changed, restructured." (claude:d738e882 2026-09-27T03:38:44Z) | **open** |
| N-02 | Remove the tabs. | same turn: "Get rid of the tabs." | **open.** Review has four primary tabs plus More (`proffer-operator-preview.tsx:49-67`). |
| N-03 | See, move, view and change files in the buckets, including moving them into the proper bucket structure. OpenList and Filestash are named as the access tools. | claude:d738e882 2026-09-27T03:38:26Z | **open** (SRC-17) |
| N-04 | One common stack with shared tools and libraries across every module. | claude:d738e882 2026-09-27T03:40:11Z; D-156 `LOG:336` | **open** |
| N-05 | "Significant layout issues" remain on Review (DF-26). | claude:d738e882 2026-09-27T03:27:52Z; screenshot relay claude:77aa963a 2026-09-27T03:28:58Z | **open** |
| N-06 | "The … intake? … There's no file tree … regression in the other pages." Which surface he meant is not recorded (C-09). | claude:d738e882 2026-09-27T03:27:52Z | **open** |
| N-07 | Reconcile first: read the earlier memories and his frustrations, change the back end as needed, then show him the proposal. | "Let's see what you do, what you come up with." (claude:d738e882 2026-09-27T03:37:50Z) | **open** (OD-14) |

## 2. Proposed changes and deliverables not yet executed

| ID | Item | Source | State |
|---|---|---|---|
| PR-01 | An Activity screen: one line per run or batch, a click only for exceptions, batch progress with "retry failed". | rethink `:51` (option A ratified 09-22 09:09, `TODO:356-358`); `the-six-steps.md:33` "or their own screen"; `router.tsx:46-47` "while Activity and Read land" | Ratified, not built. |
| PR-02 | A per-run "Technical details" drawer for package identity, hashes, the D-158 destination, repair detail, checkpoints, receipts and the Go tools. | rethink `:55` (ratified); `the-six-steps.md:33` | Not built; the Overview tab is used instead. |
| PR-03 | Vector preview: a "similar" action and embedding status in Sources, borrowing turbopuffer-gui's shape (components only). | rethink `:75`; claude:158b0721 2026-09-22T13:15:30Z | Not built. |
| PR-04 | The RAGFlow pieces still unbuilt: a select-all and bulk toolbar, goin18's non-persisting `POST /preview` with offset chunks, chunk-in-source highlighting, a stage-ladder strip. | `TODO:417`; "borrowing shape overall code where we can" (claude:158b0721 2026-09-21T01:37:51Z) | Not built. |
| PR-05 | A render step that turns processed messages into paginated Markdown conversations in `DerivedKnowledge/messaging/`, as its own activity. | `TODO:433` | Queued. |
| PR-06 | Cross-platform merged conversations as a derived view. It needs stable ids and one identity per participant. | `TODO:425`; claude:158b0721 2026-09-21T04:09:21Z | Not scheduled. |
| PR-07 | Weaviate-first publish before approval, in the engine. | `feat/engine-weaviate-first-20260926` (`9bb9e67b`, `3d3fca70`, `4fc3951e`) | **Built, scheduled and deployed 2026-10-01** (`a61b1cbb`, merge `06bba875`; proffer-worker deployment `k41b8iqikngprg3xyi6sqcdj`): every new run publishes its verified records to `MsgEvents20260918` before the preview and the commit; a failure stops the run. Live proof: the shared Live run on `sms-2024-11-24.xml` (pending the go-live identity). Detail: TODO 2026-09-30 §2 "Weaviate first". |
| PR-08 | UUIDv7 ids for `working.normalized_record` (`server/evidence/store.py:431`). | `fix/uuidv7-identifiers-20260926` (`8b219f02`, `3a601380`) | **Merged to `main` 2026-09-30** (`68989cad`, with `uuid6` pinned in `requirements.txt`, which the images install from; owner 09:10 "WE USE UUID7"). Not deployed. The `working.message.id` half is fixed by PR-09 (DF-04). |
| PR-09 | D04: the Go thread store, then activity wiring, the grant change, and live proof on a disposable schema. | `origin/main` `5075fc62`, `2a3b8a90`, `b2e9d946`; `c1fe683a` | **Merged and deployed 2026-10-01** (`955d1bbc`): propose / confirm / commit-messages / commit-threads activities, OD-07 grants and the `normalized_record.source_version_id` change applied live, Python writer retired; live proof on a disposable database passed. The shared Live run waits on the go-live identity. Detail: TODO 2026-09-30 §2 "First-party context import (D04)". |
| PR-10 | The precommit stack: a frozen proposal reader, a Review catalog over proposals, exact approval binding, idempotent publish with read-back, and public retry/cancel/resume. | precommit contract; D05-C01…C06 | Not started. Public cancel **DONE `99f8e3c6`**; retry and resume, and the rest of the stack, not started. |
| PR-11 | Bout review: mount `/api/conversations`, add a page for picking the conversation, and components that copy the artifact's layout. | "Do this" (claude:5ca9789c 2026-09-24T17:46:12Z); `51fdf193` | **Not to be built (parent ruling, option A, 2026-09-27):** the owner retired 30-minute bouts on 2026-09-24 22:29 EDT ("the 30 minute window … is breaking context", claude:b9db6641 2026-09-25T02:29:29Z; §4), and the catalog holds only 30-minute sets, so this page would show the retired method. `/api/conversations` stays unmounted (DF-15). |
| PR-12 | Chart schemes as YAML plus a named read-only catalog-query endpoint, and/or Evidence.dev pages. | claude:5ca9789c 2026-09-25T04:26:14Z | Undecided (OD-11). |
| PR-13 | Finish missing payloads: the record activity, per-backup check status, leads and resolutions, display in the Workbench and Intake, and a tracked register script. | `TODO:316-320` | Not built. |
| PR-14 | A streaming tool gateway with no sealed copies. | SRC-13; `TODO:649` | Not built. |
| PR-15 | One JSON config file for stores and roots. | `TODO:18` | Queued. |
| PR-16 | Local upload writes to B2 (add-only) and gets a `b2://` locator. | `TODO:19` | Queued (OD-01). |
| PR-17 | Derive follow-ups: the owner's own number as a participant, showing the derive result with "start batch on these threads", setting `DUCKDB_XML_MAX_BYTES`. | `TODO:368,385,482` | Open. |
| PR-18 | Test → Live promotion by re-binding. | `TODO:364` | Not built (OD-05). |
| PR-19 | Retry marks on the remaining activities; the ledger sees an external terminate. | bulk-intake `:60`; `TODO:282` | Open. A Workbench cancel now ends as lifecycle `cancelled` (`99f8e3c6`); an out-of-band `terminate` is still unreported (DF-05). |
| PR-20 | A trigram index for preview message search. | `TODO:527` | Not made. |
| PR-21 | Entity follow-ups: order by the anchor, byte locators and source ranges, the Weaviate and Surreal steps, the Timesketch worker. | `TODO:146,150`; PIPELINE-HISTORY `:21-24` | Open. |
| PR-22 | The rest of bulk intake: auto-select recorded patterns, exceptions-only gating with retry-failed, the no-parser agent button. | bulk-intake `:62-64` | Open. |
| PR-23 | The Workbench API suite in CI, and ruff complexity rules. | `TODO:738-742` | CI part **DONE** `5c1e830d` (see O-05); ruff complexity rules open. |
| PR-24 | One tailnet journey from a VPS against the exact deployed revision: browse more than 200 files, pick a later-page file, inspect and hash it, run one TEST run, check the preview and gap controls, read back the receipts. The D12 release gates stay HOLD. | `docs/reviews/2026-09-23-probata-p0-function-checkpoint.md:48`; `Propria/docs/D12-FINAL-ACCEPTANCE-PACKET-2026-09-23.md:81-85` | **DONE 2026-09-27 11:01 EDT against `f35cbd87`: 6/6 pass** (table below). The D12 release gates stay HOLD. |
| PR-25 | **The Workbench tsnet cutover** (block below). | D-132/D-134/D-127; directive of 2026-09-07 | Proposed, unexecuted. |
| PR-26 | Clean up the archive merge: restore the lost bylines and names, and remove the Graphiti env and pane. | `TODO27:44`; parent brief (`GRAPHITI_MCP_URL` still in the Coolify env, being removed tonight) | **DONE** `c387ef61` (DF-23, DF-24). |
| PR-27 | The six-step live audit of the Workbench, relaunched. | `TODO27:44` ("agents were stopped mid-run … must be relaunched") | **DONE**, see PR-24. |
| PR-28 | A public edge with no hand-made networks: Traefik reaches Authentik at a stable declared address, the Authentik admin gets a tailnet service address, and nothing on the tailnet goes through Authentik. | `TODO27:42`; claude:5fb0ec97 2026-09-27T03:23:28Z | Open, in the edge-authentik lane. The Workbench's public route depends on it (S-14). |
| PR-29 | Correct the doc drift DD-01…DD-13. | §5b | Open. |

### PR-24 / PR-27 — the six-step live audit, 2026-09-27

Run by `deploy/workbench-audit/audit.sh` (headless Chrome in the ovh-files devbox; `e6661766`, `f35cbd87`) against deployed `f35cbd87`, 11:01 EDT. Target: `calls-20250703043408.xml`, row 834 of the vault root (page 5 at 200 rows a page).

| Step | Result | Evidence |
|---|---|---|
| 1 Browse more than 200 files | PASS | 836 rows loaded through "Load more"; no duplicate `source_ref` across the API pages |
| 2 Pick a later-page file | PASS | row 834 selected in the grid; "Ready: calls-20250703043408.xml" |
| 3 Inspect and hash | PASS | panel sha256 `cc1cc9fb…cf3a0d` equals `POST /api/proffer/source-inspection` (9,442 bytes) |
| 4 One TEST run | PASS | Process gave "Started 1 run(s)"; handle `4s1WLWcKkWAHuhpRnQfKXx7CJlV37PcA`, 24 stages, awaiting the preview decision |
| 5 Preview and gap controls | PASS | Review opens the run: file metadata, the Calls table, Actions (decision, context, parser, repair, re-run), context review |
| 6 Receipts read back | PASS | preview 200 with six receipts (raw_source_verification, parser_selection, parser_execution, normalization, completeness, storage); operation 200 |

Console errors left: `unit-lookup` 503 (DF-33) and `monitored-actions/capabilities` 404 (DF-34); Desk and Knowledge have none. The run before the fixes (`5c1e830d`) also started TEST run `42MEbZOQ6R5Kvftflnd8_smYjbV8TZEO` on the same file. Both runs stay in the DEV test matter, parked at the preview decision with nothing approved; no route cancels or deletes an operation (R-14), and engine rows were not edited by hand.

### Run cancel (D05-C06 cancel part), 2026-09-28

Owner 2026-09-27 22:05 asked to see the two parked audit runs before anything removes them and to fix the missing cancel. Built on the recorded design (decision 2026-09-12: "Hold/cancel/retry are authenticated durable commands"; D05-C06) in `99f8e3c6`, deployed to proffer-worker `aaf6yhd0…`, proffer-starter `yo9xq2dp…` and the Workbench `kcxwuo52…`.

- **Path:** Review "Cancel this run" (reason required) -> `POST /api/proffer/previews/{handle}/cancel` -> engine `POST /reference-import/previews/{handle}/cancel` -> `cancel_request` Signal (actor, reason, time: the append-only receipt in the run's own history) -> Temporal `CancelWorkflow`. The operation query then reports the terminal lifecycle `cancelled` with "cancelled by <actor>: <reason>". No engine row is edited; the run stays listed.
- **Live proof (TEST):** run `sODdhBY596o0kFxV5NjaXnhVwX1SBset` on `calls-20250703043408.xml` was cancelled while running at stage 17: engine 202, then lifecycle `cancelled`, terminal, 22 stages recorded, the in-flight `hash_normalized_generation_activity` ended; a second cancel answered 409 "the run has already finished (cancelled)". The live identity on the tailnet is the owner's login (the proof ran from his desktop, DF-30), so the receipt names that login and says the cancel was a Claude Code proof.
- **Left for the owner:** the two audit runs, still awaiting the preview decision after the deploy:
  - https://workbench.tilapia-skilift.ts.net/review?mode=TEST&preview_handle=4s1WLWcKkWAHuhpRnQfKXx7CJlV37PcA
  - https://workbench.tilapia-skilift.ts.net/review?mode=TEST&preview_handle=42MEbZOQ6R5Kvftflnd8_smYjbV8TZEO

### PR-25 — the Workbench tsnet cutover

**What it is**
- Today the host serves `svc:workbench` by running `tailscale serve` into the loopback publish `127.0.0.1:18080` (`deploy/workbench.yaml:151-157`, `origin/main`).
- The cutover replaces that with an in-process tsnet listener. For the Python Workbench this is a Go `tsnet-front` sidecar sharing its network namespace. The Go services flip `TSNET_LISTENER_ENABLED`.
- "Step 2" then deletes the loopback publishes and runs `tailscale serve reset` on the host. That wording sits in comments in the deploy files, not in any recorded plan.

**Sources**
- D-132 (09-02): the gateway on tsnet.
- D-134 (09-02, `LOG:314`): every service gets its own Tailscale identity, and existing services migrate "over time; no big-bang conversion".
- D-127 (09-02): a flag's default is the production behaviour.
- Owner directive of 2026-09-07: "APIs without networking endpoints … bind to TS only" (quoted in `tests/test_tsnet_deploy_contract.py:3-4`).
- The code was written 2026-09-07 by "Claude Code subagent · Opus 5". It was preserved in archive commit `4bd3fa3b` (09-13 03:59) and was never on Probata main until merge `0ffb5b96` (09-26 19:50).

**State now (`origin/main`)**
- Still present: `modules/engine/cmd/tsnet-front`, `modules/engine/tsnetlisten`, `deploy/docker/tsnet-front/Dockerfile`, `tests/test_tsnet_deploy_contract.py`, and the tsnet env block in `deploy/proffer-starter.yaml:41-61` (flag default false).
- Removed: the Workbench sidecar (`194a3603`) and the parser-activity-runtime tsnet block (`94fb0a3a`).
- The contract test still requires both, so it now fails (DF-21).

**Preconditions (from the removed sidecar's comment)**
- A `tag:docker` auth key file at `/data/probata/secrets/workbench/ts-authkey`, mode 0400, owned by uid 10001.
- The state directory `/data/probata/tsnet/workbench`.
- `tailscale serve reset` on ovh-app.
- The key's absence is what crash-looped the sidecar tonight (`194a3603`).
- Tagging may itself be blocked. The OAuth client is refused `tag:docker` (HTTP 400), and `TAILSCALE_API_KEY` answers 401 (`origin/main` TODO devbox entry, 09-27 04:00Z). The global `CLAUDE.md` still calls that key working.

**Owner ruling on doing it now: none recorded**
- The revert says the cutover "remains a deliberate step with host prep, not a side effect" (`194a3603`).
- Told that moving the Workbench onto its own stable Tailscale address was "a separate step for later", the owner replied: "It is not a task for later, it already was an advertised fucking service" (claude:5fb0ec97 2026-09-27T03:15:32Z).
- Read literally, `svc:workbench` already existed as the stable address, and the revert restores it. This is not an order to run the tsnet-front cutover.
- A 09-08 ruling, "no Tailscale sidecars", is recorded for the devbox (`TODO27:27`). Whether it applies to `tsnet-front` is not recorded. See C-08 and OD-15.

## 3. Open owner decisions that block Workbench work

| ID | Decision | Options and default on record | Blocks | Source |
|---|---|---|---|---|
| OD-01 | How a local upload reaches DuckDB. | **A (default)**: mount the sealed-upload directory read-only into PostgreSQL and map `upload://<sha>`. **B**: retain every upload to object storage. **C**: use the Go decoder. The owner also asked that uploads writing to B2 (add-only) be evaluated. | P-15, PR-16 | `TODO:176-182`; `TODO:19` |
| OD-02 | Keep R2 `nexus` as the staging target, the R2 roots and the R2 health probes, or move staging to B2 and drop R2. | No lettered options. The owner said "I do need to retire R2" (claude:a235e3bd 2026-09-21T00:58:32Z), then re-enabled R2 (Consignatio `docs/URGENT-TODO.md`, 09-24 section). The 09-23 receipt asks him to confirm. | Staged upload, R2 retirement, O-04 | runtime-health receipt `:27`; `TODO:667` |
| OD-03 | Remove or persist the LanceDB `/api/files` staging. | No default. `AGENT_MEMORY.md:41-44` says the LanceDB design is not the approved product, and the compose file calls it "retired … not mounted" (`deploy/workbench.yaml:27-30`). | DF-14 | `TODO:294` |
| OD-04 | `MCP_DIRECT_BYPASS_ALLOWED`: may direct MCP doors feed `/api/tools`? | Pending since 09-14; no default recorded (compose default `false`). | DF-13; step 1 of the repair builder's build order | `TODO:295`; repair doc `:59-60` |
| OD-05 | Go-live identity: mint the real Matter and CourtCase so Live mode works. **ANSWERED 2026-10-01 07:17 EDT (owner): mint now.** One real Matter and one CourtCase, new UUIDv7 ids, recorded with the owner's approval; TEST stays for trials. Case facts (case number, court) come from the owner, never invented. | D-126: a DEV identity until go-live, then new UUIDs with a genuine owner approval. No date. | Live mode, S-05 | `LOG:304`; `p0-deployment-receipt:37,42` |
| OD-06 | The Weaviate collection name for the pre-approval search stage, and whether the stage is mandatory. It is built as optional. **ANSWERED 2026-10-01 07:17 EDT (owner): reuse `MsgEvents20260918`.** One place to search, no parallel store; Probata records are told apart by source and run id. Mandatory per the owner's Weaviate-first rulings (09-18 "IT ALL GOES TO WEAVIATE FIRST", 09-26 extract → searchable → confirm → commit). | No default name: "just make sure from onw on i approve collection names" (claude:5ca9789c 2026-09-24T13:09:55Z). | P-09, PR-07 | `3d3fca70`, `4fc3951e` |
| OD-07 | Apply the `platform_runtime` grants live. **ANSWERED 2026-10-01 07:17 EDT (owner): the minimal set plus UPDATE on the thread tables** (the `working.first_party_context_thread` family), so a thread can be extended in place. | One derived minimal set. SELECT+INSERT on `normalized_record`, `message_projection_route`, `message` and `message_participant`; INSERT on `normalized_record_event`; SELECT on six reads; no UPDATE or DELETE. Proven with a rollback, then applied live 2026-10-01 with UPDATE on the four thread tables and read back. | P-14, PR-09 | `sql/validation/2026-09-26-d04-engine-spine-grants-test.sql` (`c1fe683a`, `origin/main`) |
| OD-08 | Conversations, chunk method and intake reuse: where they sit in Review, which store is the permanent home for message rows versus the evidence spine, which chunk methods to offer. | No options recorded. | R-12, SRC-20 (PR-11 closed as option A, 2026-09-27: bouts are retired and the catalog holds only 30-minute sets) | `TODO:694` |
| OD-09 | D05 release: mount the HMAC delegation secret in the Workbench and the Platform API, set matching `PROFFER_TEST/REAL_MATTER_ID`, then a fresh review and live proof. | The record says HOLD. | R-13 (live but answering 503) | D05 review `:99-108,126,166-168` |
| OD-10 | **Corrected.** The final vault home for derived machine output (thread NDJSON, media, manifests). | `DerivedKnowledge` is organised by kind, and `messaging/` is the readable layer. The location is configuration. Options went to the owner and no answer is recorded. He also said "start a new structure next to vault. call it casevault" (claude:031c7682 2026-09-21T12:16:20Z); `casevault/` exists but nothing is bound to it (Consignatio `docs/URGENT-TODO.md:2379`). | P-04 | `TODO:431-432,441-447`; claude:158b0721 2026-09-21T06:07:02Z |
| OD-11 | Where chart schemes live. | **A**: YAML with a schema (recommended). **C**: Evidence.dev pages. The owner's edit reads "A … and … C". Not confirmed. | PR-12 | claude:5ca9789c 2026-09-25T04:26:14Z |
| OD-12 | Which of D-154R…D-160R stand. D-160R asks him to confirm "workbench for probata, intake for consignatio". | "What the fuck are they?" At 19:51 he answered D-154R and D-155R only. | The Workbench/Intake boundary (C-07) | `LOG:340,347`; claude:77aa963a 2026-09-26T23:29:37Z, 23:51:27Z |
| OD-13 | Go-ahead for a live batch over a real thread folder (106 or 132 files, each a TEST run parked at the preview decision). | No options recorded. | Live proof of SRC-15 and P-08 | `TODO:385(1)` |
| OD-14 | What may stay permanently beside the messages: the Actions/Repair panel, a collapsed rail, or a drawer. | He declined to confirm an agent's reading of D-159 and asked for a proposal (claude:d738e882 2026-09-27T03:37:50Z). | R-02, N-05, the layout work | see C-04 |
| OD-15 | Whether to run the Workbench tsnet cutover now. | No options recorded. D-134 says "over time". | PR-25 | see C-08 |
| OD-16 | The Desk `/`: keep it, repurpose it (for example as Activity), or drop it. | No options recorded. | S-01, DF-16 | see C-01 |

## 4. Superseded or retired items that must NOT be built

| Do not build | Superseded by |
|---|---|
| Intake as the front door: declared formats, a six-checkpoint rail, caveat paragraphs, opening on gdrive/local/onedrive | Sources: option A ratified 09-22 09:09 (`TODO:356-358,563`). `/intake` stays only as a deep link (`router.tsx:46-47`); the Desk must not route to it (DF-16). |
| Review as a pipeline dashboard: package cards, the tool catalog, "Next valid actions" at the bottom | D-159 (`the-six-steps.md:32-33`); layout diagnosis `:32-38` |
| Machinery as primary content, including the General tier's "persistent right custody inspector" (`Propria/SURFACE-DESIGN-CONTRACT.md:60`, 09-12) | D-159 (09-22). C-04 covers the Actions rail. |
| The SBV ThreadView as the default | Owner 09-20 23:48: dense rows are the default and the SBV conversation is optional (handoff `:146`) |
| Codex's thin `platform-message-viewer.tsx` | The SBV front-end port (`TODO:521`) |
| The SBV viewer as its own surface, or iframed (ADR-0061's bounded iframe/full-page launch); D-149 item 10 "Vite viewer … ≠ HITL preview" | SBV components ported into Review (owner 09-20 23:10; `TODO:234` "newest owner statement wins"; memory `work-surface-is-probata-workbench`) |
| SBV as a separate Tauri desktop ingest client (D-123, `LOG:298`) | D-150 and D-160R (`intake` belongs to consignatio), then 09-22: Sources in the Workbench now, Xplorer Intake as the later front door |
| The legacy Operator Console and its LanceDB staging/promote design (`modules/workbench/api/README.md`) | `AGENT_MEMORY.md:41-44`; the Proffer path |
| Opening a thread on the newest message | Oldest first (claude:158b0721 2026-09-21T03:27:54Z; handoff `:148`) |
| 30-minute silence bouts as the chunk or bout method | LLM-found conversations, prompt v3 (`TODO:692-693`; "the 30 minute window … is breaking context", claude:b9db6641 2026-09-25T02:29:29Z) |
| A read-only B2 key | Read-write-delete (`TODO:22`) |
| Hard-coded R2 sources; R2 `casebible-sorted` as the vault (D-149 item 2) | Storage as configuration with the B2 vault as default (`TODO:22,184-195`) |
| `<key>.derived/` as the final home; the 09-20 23:51 "top-level directory mirroring the source tree" | Configuration only, final home open (owner 09-21 02:07 "That got changed"; `TODO:405,447,507`) |
| DuckDB `read_xml` over whole SMS backups | SBV streaming derive → NDJSON (`TODO:216`; bulk-intake `:36`) |
| A DuckDB-first, decoder-fallback priority ladder | The signature registry (D-149 item 9 correction, `LOG:329`) |
| Sorting first: moving to the canonical home before any ingest | Concurrent, independent actions (owner 09-22 10:48; `TODO:603-606`) |
| Repairing the Python first-party writer; "Python keeps the spine" (`347bf0a2`) | Go Temporal activities; the engine writes (D-161; `c1fe683a`; decision `:60`) |
| Restoring custody guard functions into intake tables | Promotion side only (owner 09-20 21:39, `TODO:226`) |
| Custody or H1–H3 wording at intake | Context fingerprints (D-152, D-154; "refer to nothing as custody", PIPELINE-HISTORY `:62`) |
| glm-5.1 anywhere; Ollama as primary; silent classify fallbacks | kimi-k3 on NIM with explicit failure (`TODO:154,160`) |
| Per-person Weaviate columns (`catrina_class`, `katrina_ref_type`, `katrina_conf`) | The generic entity model (metadata doc `:29`; `TODO:228`) |
| "No Weaviate write before exact approval" (precommit `:30,153`) | Weaviate first, before approval (`TODO:757-759`). The PostgreSQL commit still waits for the confirm. |
| "SurrealDB receives nothing automatically; Neo4j is the first graph destination" (precommit `:27-28`) | For entities and timeline: copied at once to Surreal and Neo4j through the outbox (owner 09-26 08:32–08:33, 19:33; `TODO:826`). Other material: C-03. |
| "All canonical searchable text in PostgreSQL"; chunks in Weaviate without text everywhere (D-149 items 7–8) | D-158 (by source type), then 09-18 "IT ALL GOES TO WEIVIATE FIRST" (PIPELINE-HISTORY `:107`), then 09-26 Weaviate first with extracted events to PostgreSQL |
| Graphiti in the Workbench: `/api/graphiti/*`, the C4 graph-memory pane, `GRAPHITI_MCP_URL` | Graphiti retired (D-070 `LOG:86`; D-095 `LOG:40`) |
| Milvus as a Workbench knowledge store (`main.py:11` "Milvus-backed knowledge search") | Milvus serves memsearch only; Weaviate is the project vector store (Probata `AGENTS.md` "Stack") |
| Probata search reading Semantica's `forensic_*` collections | Its own path (claude:fa70bdb2 2026-09-22T23:08:00Z) |
| Headless browser checks on the desktop; "every deploy followed by a desktop screenshot" (`TODO:374`) | VPS devbox or API only (`web/AGENTS.md:30-39`; owner ban 09-24) |
| Next.js conventions and build in `web/` | Vite + TanStack (D-108; D-129 names `next.config.ts`, `next-env.d.ts` and `out/` as drift to clean) |
| An app login, or "Sign in" labels, on the tailnet | Tailscale only (owner 09-23 08:06; 09-26 23:08) |
| The Glide 6.0.3 / React 18.3.1 baseline for the Workbench (D-157) | The owner-approved alpha24 exception (`950542e0`; "already did") |
| "Read" as the Review screen's name (rethink option A) | "Review" (D-159, later the same day) |
| Hand-made containers, networks, attachments or host-only edits | Owner 09-26 22:52 (S-14) |

## 5. Known defects in the Workbench and its pipeline

| ID | Defect | Area | Source | Fix status |
|---|---|---|---|---|
| DF-01 | The engine writes PostgreSQL right after extraction. The Weaviate and Surreal steps are unbuilt, and with no byte locators `event_candidate_source_range` is never written. | engine | PIPELINE-HISTORY `:21-24` (`9e7904c7`) | **Weaviate step done 2026-10-01** (PR-07): records are searchable in Weaviate before the commit. Still open: the Surreal step and `event_candidate_source_range`. |
| DF-02 | First-party projection writes `working.conversation`, which the 09-07 snapshot deleted, so first-party context cannot be imported. | engine/Python | decision 2026-09-26 `:16-22`; `docs/reviews/2026-09-24-d04-first-party-projection-schema-hold.md:9` | **Fixed 2026-10-01** (PR-09, `955d1bbc`): the Go engine writes the `first_party_context_thread` family; the Python writer is retired. |
| DF-03 | `platform_runtime` cannot write `working.message` or `working.normalized_record`. It has no UPDATE on the thread tables, by design. | DB grants | `347bf0a2`; `c1fe683a` | **Fixed 2026-10-01** (OD-07): grants in the snapshot and applied live, with UPDATE on the four thread tables. |
| DF-04 | `working.message.id` is a fresh uuid4 instead of `normalized_record.id`, and `store.py:431` mints uuid4 against uuidv7 defaults. | Python/DB | decision `:38-40`; `3a601380` | **Fixed 2026-10-01** (PR-09): `working.message.id` = `working.normalized_record.id` = the context record id, minted UUIDv7 by the engine. |
| DF-05 | Only one activity has permanent-failure marks, and the ledger shows an externally terminated workflow as running. | engine | bulk-intake `:60`; `TODO:281-282` | **Open.** Cancel through the Workbench now reports `cancelled` (`99f8e3c6`); an external terminate still does not. |
| DF-06 | The tool gateway downloads whole sources to ovh-app's root disk. Duplicate staging copies filled that disk on 09-22. | gateway | `TODO:643-649` | Duplicates **fixed** (`722e88c`). Streaming is not built. |
| DF-07 | The worker crash-looped on unreadable secret files after the 09-26 deploy. | deploy | `TODO:72` | **Fixed.** |
| DF-08 | Derive keys the owner's own number as a participant, so 1:1 MMS threads split off as groups. | engine | `TODO:482`; `TODO:385(4)` | **Open.** |
| DF-09 | `create_simple_secret(name := …)` fails silently, and `platform_r2` does not exist. | API | `TODO:208` | **Open.** |
| DF-10 | 12 test failures: `KeyError: 'workflow'` at `server/api/run_routes.py:664`. | API | `TODO:324` | Not recorded as fixed. **DONE `4c53a3e7`:** six fixtures omitted `workflow`, a NOT NULL column of `ops.workflow_run`; the route is right. The `Step(on_error=)` cluster appears only under agno 3.x, not the pinned 2.8.7, so it is not the same cluster and not a code defect. Both files: 118 passed, 6 skipped. |
| DF-11 | The XML probe's substring match can send generic XML containing `<sms` to the SMS parser. | engine | `docs/pending-review/D02-PB06-BOUNDED-XML-PROBE-2026-09-23.md:36` | Known, **not fixed.** |
| DF-12 | The Timesketch projection does not run because the Python evidence-pipeline worker is not deployed, and `generation.py` orders by `created_at`. | timeline | `TODO:146,150` | **Open.** |
| DF-13 | The Atomic/Go tools screen is broken: `/api/tools` returns `[]`, capabilities returns 404, and choosing a decoder locks the page. | web/API | Docstore `note:probata_function_access_broken_20260912` (critical); "There's nothing listed … It locks up" (claude:158b0721 2026-09-21T12:10:25Z) | **Open** (OD-04). Under D-159 this is a developer page. |
| DF-14 | `/api/files` returns `[]`: LanceDB at `/data/lancedb` is not mounted, so every redeploy empties it. `/health` still reports `lancedb=true`. | API | `TODO:294`; `deploy/workbench.yaml:27-30` | **Open** (OD-03). |
| DF-15 | The bout-review router (`app/runtime/conversations.py`, `/api/conversations`) is not included in `main.py:118-144`. | API | `51fdf193`; code | **Open** (PR-11). |
| DF-16 | The Desk's main link "Open intake" goes to `/intake`, the superseded front door, and the page serves no step. | web | `web/src/surfaces/primary/evidence-operations-desk.tsx:78` (code) | **Open** (OD-16). |
| DF-17 | D05 flag creation fails closed with 503 because no deploy manifest mounts `/run/secrets/proffer-flag-delegation-key` and the matter ids are unset. The code shipped in the ‡ builds while the review verdict is HOLD. | API/deploy | `app/service/flags.py:22-42`; `server/api/inspect_routes.py:1008-1028`; D05 review `:126` | **Open** (OD-09). |
| DF-18 | The Workbench API suite is invisible to CI, and `app/service/proffer.py` (326 lines) breaks its own 300-line test. | CI | `pyproject.toml:200`; `api/tests/test_structure.py:92-99`; `TODO:738-742` | **DONE `5c1e830d`:** CI workflow (O-05); `proffer.py` split to 296 lines (`proffer_content.py`); a FastAPI 0.141 route-listing test fixed. |
| DF-19 | Sources marks break when a file moves, because they key on `source_ref` and the folder prefix. | web/API | `TODO:607-611` | **Open; needs an engine change (reported 2026-09-27).** No stable id exists end to end: runs and decode manifests are keyed on `source_ref` by the engine, the listing carries only `etag` (not a content hash), and runs carry no content hash. Keying marks on content needs the engine to record the acquisition sha256 (or vault key + hash) on each operation, plus a catalog join in the listing. |
| DF-20 | Wrong catalog unit joins. | API | `d3490220` | **Fixed**; first shipped in the ‡ builds. |
| DF-21 | `tests/test_tsnet_deploy_contract.py:36-40` still requires tsnet blocks in `workbench.yaml` and `parser-activity-runtime.yaml`, which `194a3603` and `94fb0a3a` removed, so the test fails. | tests/deploy | code (`origin/main`) | **DONE `1eb32bf4`** (2026-09-27): the parser's tsnet block restored, the test pinned to the Workbench's single published door; 56 deploy-contract tests pass. |
| DF-22 | The Workbench went down twice tonight. The archive-merge sidecar had no auth-key file, so Coolify bind-created a directory, the sidecar crash-looped, and Coolify stopped the whole app at 01:49Z and 03:01Z. | deploy | `194a3603` message; "Probata is not accessible at all" (claude:5fb0ec97 2026-09-27T03:10:50Z); "HTTP ERROR 502" (…03:16:28Z) | Reverted on `origin/main` (23:15 EDT). Redeployed and healthy since: `k10cez6p` (09-27 04:26Z), `zh8trbgk` (14:32Z) and `bj7pdt7s` (15:01Z). |
| DF-23 | The archive merge dropped the 09-26 bylines and renamed the API title and the web package. Only `main.py`, `package.json` and `package-lock.json` changed under `modules/workbench` (`git diff 411231dd 0ffb5b96`), so the merge does not explain any UI regression. | code | `TODO27:44` | **DONE `c387ef61`:** both bylines restored. The two names are kept: "Probata Workbench API" and `workbench-web` match the naming canon (`docs/NAMING.md`: `workbench`); "Knowledge Workbench" is the retired name. |
| DF-24 | Graphiti is still wired: `GRAPHITI_MCP_URL` in the Coolify env (being removed tonight, per the parent brief), plus the `/api/graphiti/*` routes and the Graphiti pane (`main.py:1,11`). | API/deploy | parent brief; `modules/workbench/api/README.md` | **DONE `c387ef61`:** routes, service, MCP client, settings, pane, client calls, types and the compose comment removed; no `GRAPHITI_*` key is left in the Coolify env (checked 2026-09-27). Live: the Knowledge page has no Graph memory tab. |
| DF-25 | kimi-k3 option B (long prompts retry with thinking on) has not been re-verified live since the 09-26 deploy. | models | `TODO:71` | **Open.** |
| DF-26 | Review layout at 1920 px, from the owner's screenshot as relayed: three competing vertical scrollbars; the Actions rail clipped ("Mark as event worth rec…") with Repair below the fold; the selected message body truncated in a narrow column; the message table cramped behind a horizontal scroll. | web | claude:d738e882 2026-09-27T03:27:52Z; relay claude:77aa963a 2026-09-27T03:28:58Z | **Open.** |
| DF-27 | `MarkEventButton` renders in two places. | web | `web/src/components/sbv/message-detail-panel.tsx:63`; `web/src/components/entities/record-peek.tsx:61`; relay claude:d738e882 2026-09-27T03:33:11Z | **DONE `4c4ccc11`:** kept on the selected message's detail panel (G-07, "any message"); removed from the mention peek. |
| DF-28 | The public route depends on the hand-made `propria-edge` network with fixed IPs. The proxy lost that attachment: "Public portal: down since 09:48 EDT" (owner-pasted, claude:5fb0ec97 2026-09-27T03:05:48Z). Whether `workbench.int` was also down is not recorded. | deploy | `deploy/workbench.yaml:166-201` | **Open** (PR-28). |
| DF-29 | Next.js residue in `web/`: `next.config.ts`, `next-env.d.ts`, `out/`, `app/layout.tsx`, `error.tsx`, `global-error.tsx`. | web | D-129 `LOG:309`; code | **DONE `008371a9`:** `next.config.ts`, `layout.tsx`, `error.tsx`, `global-error.tsx`, `favicon.ico` and the template SVGs removed (nothing imported them); `out/` did not exist. |
| DF-30 | The tailnet identity check denies device-tagged callers. | auth | runtime-health receipt `:13` | **DONE 2026-09-28** (owner option A 04:22: "Authentik service accounts: one identity system for people and machines"). The devbox is the Authentik service account `devbox` (group `propria-machines`, provider `propria-workbench-api`, app `workbench-api`). It sends its access token as `Authorization: Bearer <JWT>`; the Workbench (`api/app/runtime/machine_jwt.py`) checks the JWKS signature, exact issuer, audience, expiry and group, and records `authentik-sa:devbox`, on the Serve and Traefik doors only. Proof from inside the devbox: `/api/proffer/sources` with the token 200, without it 403 "Untrusted proxy", a tampered token 401, spoofed X-authentik headers 403. The six-step audit ran direct from the devbox, no tunnel: **6/6 PASS** (TEST run `gTR33V2U…`). Commits `7a991b6f`, `ef7cf8ba`, `82395891`; Workbench deploy `l6fduwh33vnw0w5p5df3zajg`, devbox deploy `uoxbzry4j2suzcfi0sy81z45`. ADR-0099 (proposed) records the pattern for other apps. **Later, the owner decides:** once every machine client holds an Authentik identity, the per-app Tailscale Services could narrow to people only and the `cap/workbench` capability grant could be retired. Nothing is removed or changed until the owner says so (04:22). |
| DF-31 | Past 200 loaded files, Sources sent every row to `/api/proffer/decoded/exists` (cap 200) and `/api/intake/discovery/unit-lookup` (caps 200/400), which answered 422, so decoded and unit marks stopped after page 1. | web | live audit 2026-09-27 | **DONE `804dd393`** (the client splits the lists). |
| DF-32 | "Names and paths" showed "not available yet" whenever Intake discovery was down, though it is the B2 root search. Every page logged a `/favicon.ico` 404. | web | live audit 2026-09-27 | **DONE `804dd393`.** |
| DF-33 | `/api/intake/discovery/unit-lookup` answers 503 "Pre-ingest catalog query unavailable or timed out", and the capabilities report every Intake mode false, so unit badges and catalog provenance are empty on Sources. | Intake dependency | live audit 2026-09-27 | **Open** (Intake lane, session `d738e882`). |
| DF-34 | Review calls `/api/monitored-actions/capabilities`, which has no backend route (404 on every Review load). | web/API | live audit 2026-09-27 | **Open** (part of DF-13, OD-04). |

### 5b. Doc drift (correct in the same change as any related build)

| ID | Stale claim | Current record |
|---|---|---|
| DD-01 | `modules/workbench/web/README.md:29-40` names the nav as `/`, `/intake`, `/evidence/preview` and says Glide is "not claimed". | The nav is Desk/Sources/Review, and Glide is in use. **DONE `008371a9`.** |
| DD-02 | `modules/workbench/api/README.md` describes staging + promote, LanceDB, "fixed read-only `casebible-sorted`", `nexus` staging, Graphiti routes and a Milvus note. | B2 read-write (`TODO:22`), the Proffer path, Graphiti retired. Graphiti rows and the source-bucket line **DONE `c387ef61`**; the staging and LanceDB sections remain. |
| DD-03 | `docs/design/0061-unified-operator-surface/spec.md:9` says "Proposed; implementation blocked pending owner acceptance". | ADR-0061 was accepted 08-29 (`docs/adr/README.md:79`). |
| DD-04 | `modules/workbench/AGENT_MEMORY.md` (verified 08-27) routes to the mockup memory, whose "first complete slice is intake". | Sources and Review, D-159. |
| DD-05 | `deploy/workbench.yaml:8-13,32` says "KNOWLEDGE WORKBENCH", watch paths `workbench/**`, "Branch: workbench/sprint", and "Case Bible Sorted is the fixed default ingestion source". | Prefixed watch paths, `main`, B2 default. **DONE `c387ef61`.** |
| DD-06 | `modules/workbench/api/main.py:11` says "Milvus-backed knowledge search … read-only Graphiti". | Retired roles. **DONE `c387ef61`.** |
| DD-07 | `TODO:77` lists four decisions as pending. | Answered 19:29 and 19:33. |
| DD-08 | `docs/pending-review/2026-09-21-intake-review-module-rethink.md:4` says "nothing built yet". | Sources was deployed 09-22. |
| DD-09 | `docs/decisions/2026-09-20-bulk-intake-owner-requirements.md:4` says "steps 2-6 not built". | Derive routing, batch-by-folder and the Takeout proposal are built (`TODO:378-384,406,582-584`). |
| DD-10 | The repair doc `:4` says "build in progress"; the metadata doc `:40` says "NOT applied, NOT deployed". | Both were deployed 09-26 (`TODO:69-71`). |
| DD-11 | `docs/pending-review/d05/*` says HOLD. | Merged `bd0acb7b` and built into the Workbench (‡) by the monorepo integration, not by a release decision. |
| DD-12 | `Propria/docs/decisions/2026-09-26-monorepo-import.md:4,98-103` says "imported-pending-cutover … unpushed". | `Propria/AGENTS.md`: one repository. |
| DD-13 | The docstring of `tests/test_tsnet_deploy_contract.py` describes live tsnet flags. | The manifests were reverted (DF-21). |

## 6. Contradictions I could not resolve

| ID | Conflict | Sides (newest last) | Why it stays open |
|---|---|---|---|
| C-01 | The Desk | The nav keeps Desk `/`, which serves no step, and its main link opens `/intake` (DF-16). Ratified option A lists Sources/Activity/Read; D-159 names only Sources and Review. | No ruling on the Desk (OD-16). |
| C-02 | Gates on clean files | Clean files flow without a click (bulk-intake `:40`; D-145; "not asked for gates", 09-20 21:52) versus per-attempt exact approval (precommit contract; D05) and batch items parking at the preview decision (`TODO:385`). | Which gate applies to a clean file is not ruled. |
| C-03 | Surreal and Neo4j | D-145 has the owner click "send to Surreal"; the precommit contract makes Surreal manual and Neo4j the first graph destination after approval. On 09-26 entities and timeline are to be copied "immediately" to Surreal and Neo4j. | The newest ruling wins for entities and timeline; for messages and other context the rule was not restated. |
| C-04 | What sits beside the messages | Owner "yes" to machinery in a collapsed right rail (09-21 01:47); D-159 "a drawer, or their own screen" (09-22); an always-present Actions panel, now holding the Repair builder (09-25 00:16); `SURFACE-DESIGN-CONTRACT.md:60` "persistent right custody inspector"; 09-26 23:37 he declined to confirm that a permanent Repair rail is forbidden; 23:38 "Get rid of the tabs". | He asked for a proposal (OD-14). |
| C-05 | Tabs | "More … for now" (09-20 23:27) and four primary tabs in the code, versus "Get rid of the tabs" (09-26 23:38). | The newest wins (remove tabs), but no replacement pattern is recorded. |
| C-06 | How much gating | "over designed … too many uneeded and not asked for gates" (09-20 21:52) versus the precommit contract and D05, which keep adding gates (HMAC delegation, exact approval binding). | No ruling removes the precommit contract. |
| C-07 | Workbench/Intake boundary | D-160R (unconfirmed) versus "early stage merges" (09-22 10:48) and "probata … surface … And intake, because it's kind of merged" (09-26 23:59); direction C (Xplorer Intake as front door). | OD-12. |
| C-08 | tsnet | D-134, the 09-07 "bind to TS only" directive and the tool gateway as reference implementation, versus the owner's 23:15 "it already was an advertised … service" and a 09-08 "no Tailscale sidecars" ruling recorded for the devbox (`TODO27:27`). | Whether and when the Workbench moves from host Serve to a tsnet front is not ruled (OD-15). |
| C-09 | "Intake … no file tree" | Workbench `/intake`, which the Desk links to, shows the old UnifiedIntake without the Sources tree (`app/intake/page.tsx`). The relayed interpretation assumed Xplorer Intake. | Which surface he looked at is not recorded. |
| C-10 | R2 | "I do need to retire R2" (09-20) versus R2 re-enabled and the health check probing R2 buckets (O-04). | OD-02. |
| C-11 | D05 state | The review says HOLD (`:126,166`), but the code is merged (`bd0acb7b`) and deployed (‡) through the monorepo integration. | No release decision is recorded. |
| C-12 | Provenance of the 09-26 ~20:00 rulings | Weaviate first, and extract → confirm → commit, are recorded as owner rulings (`TODO:757-759,784-786`; D-161), but no verbatim owner turn exists in `~/.claude/projects` logs. | They are treated as owner rulings per the record. |
