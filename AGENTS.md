# Propria — Monorepo Root Contract

## Current checkout routing — verified 2026-09-20

This dated routing supersedes historical placement text below. Propria owns the
root governance, `modules/FL-MCP/`, and the preserved shared `scripts/` and
`plugins/` source. The other product repositories retain independent histories:

| Product | Current path | Commit boundary |
|---|---|---|
| Indicia Probata | `modules/Probata/probata/` | Independent repository |
| Consignatio / Intake | `modules/Consignatio/` | Independent repository |
| Advocatio legal workdesk | `modules/Legal-desktop/` | Independent repository; `master` |
| Vestigia | `modules/vestigia-geodata_processor/` | Independent outer repository |
| TraceIQ Rebuild | `modules/vestigia-geodata_processor/traceiq-rebuild/` | Independent nested repository; `master` |
| Family Court Workbench | `modules/FL-MCP/` | Propria root repository |

Read the selected module's local instructions and verify its Git root before
staging. The existing `docs/` junctions and source registry remain the Docstore
routing surface. `docs/.docstore/` contains private credentials and database
state, never publishable source.

Root `scripts/` and `plugins/` preserve moved source, including edits. This is a
source preservation cutover only: Probata's deployment-owned copies remain in
its repository, and may contain newer integrated work. Do not overwrite either
copy or repoint deployment/installed plugins until their differences and build
contexts are reconciled. See `docs/ROOT-SOURCE-RECONCILIATION-2026-09-20.md`.


<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 3 | Platform: Codex / win32 | Changes: establish universal Docstore use | Context: explicit owner clarification -->

**Owner decision — 2026-09-12 (repository topology):**
`E:\AI_Workspace\Projects\Propria` is the top of the Propria monorepo. Owned
applications and product modules converge under `projects/`; shared, reusable,
reference, and vendor-governance material converges under `resources/`. Large
corpora, databases, caches, generated indexes, secrets, and runtime state are not
monorepo source and must remain ignored or externally mounted with manifests.

Existing child Git repositories must retain their histories and local-only work
until a staged import is independently verified. ~~Consignatio remains an explicit
independent canonical repository; its `projects/consignatio` import is a duplicate
overlay pending reconciliation, not an authority transfer.~~ **Corrected 2026-09-18 (Claude Code · Opus 5; owner: Propria is the project, no separate `projects/` home):** the 09-12 `projects/consignatio` overlay is quarantined in `Consignatio/to_be_deleted/2026-09-17-projects-consignatio-overlay-from-2026-09-12/`; Consignatio stays at `Propria/Consignatio/` (a same-day move into `projects/` was reversed). An upstream fork may
remain an explicit external dependency when its update/provenance lifecycle
requires that boundary.

See [the staged migration plan](docs/MONOREPO-MIGRATION-PLAN-2026-09-12.md).

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

**Owner decision — 2026-09-12 (universal Docstore):** Probata hosts
Docstore, but Docstore serves the entire Propria monorepo. All project
documentation will be progressively registered and indexed there. Every agent and
project must be able to use its first-class semantic search, bounded recall,
resource retrieval, note/decision, revision, flag and freshness-verification
tools. Before updating documentation or recording a note, query Docstore for
related current decisions; write through its governed tools and read back the
result. Source documents keep their owning repository and authority. Universal
Docstore access does not merge Docstore with CCC or Intake.

See [CCC / Intake / Docstore boundaries](SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

**Owner rule — 2026-09-14 (decision preflight):** every decision, in any child
project or working directory, starts at the router root and runs the preflight
in `E:\AI_Workspace\AGENTS.md`: last week of Codex sessions, `/read-memories`,
memsearch, CNF, `.remember`, Docstore `fn::current_decisions`. Never stop at the
first result. <!-- added by Claude Code · Fable 5.1 · 2026-09-14 -->

> _Byline: Codex · GPT-5 · 2026-08-27; repository reconciliation refreshed 2026-08-29._

This directory currently contains several former independent product repositories
and historical/reference material. During migration, select the intended lane and
descend into its local `AGENTS.md` and `AGENT_MEMORY.md`. Those local contracts
remain authoritative until their lane is imported and the root manifest marks the
cutover complete.

These child references resolve in the configured live workspace. The parent repository does not
currently hydrate every child on a fresh clone; use `../REPOSITORY_BOUNDARIES.md` for canonical origins
and verified representation state.

## Active products

| Work | Repository | Next instructions | Commit root |
|---|---|---|---|
| Evidence custody, ingestion, parsing, knowledge horizons, analysis, and operations — **Indicia Probata** | `Probata/probata/` | `Probata/probata/AGENTS.md` | verify inside `Probata/probata/` |
| Legal research, strategy, drafting, review, and release preparation — **advocatio** (the legal workdesk / legal workbench; GitHub repo still `Legal-Workspace`) | `Legal-desktop/` | `Legal-desktop/AGENTS.md`, `Legal-desktop/AGENT_MEMORY.md` | independent Git root at `Legal-desktop/` |
| Simple record / reference / calendar surface usable from a phone and by agents — **Family Court Workbench** | `FL-MCP/` (**2026-09-18, owner: Propria is the project, no `projects/` folder** — moved back from ~~`projects/family-court-workbench/`~~) | its local `AGENTS.md` | Propria root |
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md` | independent canonical Git root at `Consignatio/` (~~`projects/consignatio/` duplicate overlay~~ quarantined 2026-09-18) |

Current paths above were reconciled against the migration manifest on 2026-09-13.
Owner decision, 2026-09-13: Legal-desktop/ is the canonical independent Advocatio application. Its original build kit is preserved under resources/build-kit/. This is an explicit exception to the proposed projects/ layout.
The retired workspace layout is historical evidence and is not a current route.
The former `milvus-coolify/` path is absent here; do not recreate it from old routing.

The Evidence Platform is canonical for evidence. Legal Workspace consumes accepted,
versioned `LegalSourcePackage` data and never becomes a second writable evidence store.

## Reference-only areas

- `Agno-MCP-Platform-agno - alpha/` is a parts bin, not an active build.
- `dev-resources/` and `Legal-desktop/resources/build-kit/` are references/parts bins. Port bounded useful material only;
  never revive an archived iteration wholesale.
- Never open or ingest `dev-resources/Archives/OTHER_RESOURCES_TO_SORT/Secrets/`.
- Ignore `*.xxh3` checksum files during discovery.
- Historical workspace handoffs and v8.1 guides are seeds, not current product truth.

## Repository rules

1. Run `git rev-parse --show-toplevel` from the target path before staging or committing.
2. Until a project is marked imported in the root migration manifest, product
   changes commit in its existing child repository. After cutover, they commit in
   this root repository under `projects/<project>/`.
3. Root governance and migration files commit from this directory with an explicit
   path allowlist.
4. A dirty child or changed parent pointer is not authorization to stage that pointer.
5. Never reset, clean, stash, overwrite, or hard-delete concurrent work. Quarantine files under the
   owning repository's `to_be_deleted/` directory; only the owner deletes from quarantine.
6. Create Propria-owned migration worktrees under `E:\AI_Workspace\Projects\Propria\_worktrees\`.
   That directory is machine-local and ignored. During the transition, a child repository may link
   a worktree there without changing its Git boundary. Move an existing linked worktree only with
   `git worktree move`, after its owner is paused and its branch, dirty state, `.git` pointer, common
   directory, and representative hashes have been captured.

Read `AGENT_MEMORY.md` for progressive routing. The current workspace boundary
map is `../REPOSITORY_BOUNDARIES.md`; the migration manifest governs each lane's
transition state.

> _Sprint mode was removed by owner order on 2026-08-25. Confirm and discuss before changing._

## Paths

> _Byline: Claude Code · Opus 5 · 2026-09-26 — moved here from this repository's `CLAUDE.md`
> when that importer was removed, so `AGENTS.md` is the single instruction file Claude Code
> loads. Original owner order 2026-09-10 06:47, "add path guidance to the Claude MD file";
> paths current as of the 2026-09-20 `modules\` layout._

**Local (desktop)**

- **Sessions start at this mono root:** `E:\AI_Workspace\Projects\Propria`. Its Claude
  auto-memory store is `C:\Users\matts\.claude\projects\E--AI-Workspace-Projects-Propria\memory`.
  The per-product store keys (`…-Propria-Probata`, `…-Propria-Probata-probata`) are junctions
  into that same folder, so all of Propria shares one memory store.
- **Probata repo, git root:** `E:\AI_Workspace\Projects\Propria\modules\Probata\probata`. It is
  its own Git repository until its import is recorded in `docs\monorepo-migration-manifest.json`.
  Commit only from there, staging by explicit path — other sessions share that index and stage
  hundreds of their own files.
- **memsearch (shared agent memory):** ONE folder for every agent and project,
  `C:\Users\matts\.memsearch\memory`, and ONE Milvus collection,
  `agent_session_memory_nemotron3`, pinned by `C:\Users\matts\.memsearch\.collection`. Claude
  gets it from `MEMSEARCH_DIR` in `~\.claude\settings.json`; Codex from
  `~\.codex\hooks\memsearch_codex_hook.py`. Codex's own memory store is imported under
  `memory\codex\<project>\`. Both plugin copies carry a local patch that honors `.collection`;
  re-apply it after a memsearch plugin update. The `memsearch` CLI itself is the private fork
  `~\.claude\local-plugins\forks\memsearch` (uv tool `0.4.19+propria1`, 2026-09-26): never
  install memsearch from PyPI, which drops the fork's NIM fixes; reinstall steps are in the
  fork's `propria/README.md`. _(Claude Code · Opus 5.5 · 2026-09-26)_
- **Worktrees:** `E:\AI_Workspace\Projects\Propria\_worktrees`. New Propria-owned linked
  worktrees belong here; relocate an existing linked worktree only with `git worktree move`,
  after its owner is paused and its state is captured.
- **Running TODO:** `modules\Probata\probata\docs\planning\<date>-TODO.md`. Current file is
  `2026-09-20-TODO.md`.
- **Handoffs:** `modules\Probata\probata\docs\handoffs\HANDOFF-<date>-<topic>.md`.
- **Secrets:** `C:\Users\matts\.secrets`. Parse with a regex and never `source` these files.
- **rclone:** the binary is the scoop shim `C:\Users\matts\scoop\shims\rclone.exe`, and the
  config is `C:\Users\matts\scoop\apps\rclone\current\rclone.conf`.

**VPS (tailnet only)**

- **ovh-files** is `100.91.190.107`. **ovh-app** is `100.72.169.40`. Connect with
  `ssh -i ~/.ssh/ovh root@<ip>`.
- **Host roots:** `/data/probata/volumes`, `/data/probata/config`, `/data/probata/secrets`,
  `/data/probata/tsnet`.
- **Coolify render dir:** `/data/coolify/applications/<uuid>/` holds only `docker-compose.yaml`
  and `.env`, with no repo checkout. A relative bind of a repo file becomes an empty directory.
  Mount config files from absolute host paths under `/data/probata/config/<app>/`.
- **Service URLs:** `https://<name>.tilapia-skilift.ts.net`, served by each host's own
  tailscaled as a Tailscale Service. Adding one takes three steps: register,
  `tailscale serve --service`, approve the host.

**Path syntax and file tools on this machine**

- The Bash tool collapses a doubled backslash to a single one before bash runs. A Windows path
  inside a Python or JSON string then turns `\t` into a tab and `\r` into a carriage return. In
  Bash, write Windows paths with forward slashes (`E:/AI_Workspace/...`); in scripts, build a
  backslash with `chr(92)`. A quoted heredoc (`<<'EOF'`) keeps backslashes literal and is safe
  for file content.
- Prefix commands that pass `/unix/paths` to `ssh` or `docker` with `MSYS_NO_PATHCONV=1`.
- The Read, Write and Edit tools refuse paths outside the session's allowed working directories.
  `permissions.additionalDirectories` in `~\.claude\settings.json` grants **Read** outside them
  (`C:\Users\matts\.claude` was added 2026-09-26) but **not Write** — write files there with a
  Bash heredoc or a Python heredoc instead.
- The case-bible guard hook rejects any Bash command whose text contains a hard-delete pattern
  (the `rm` command with a force or recursive flag, or Python's file-removal call) anywhere in
  the command, including inside a quoted string or heredoc. Move material to a quarantine
  directory instead, and use `docker stop` with a self-removing container for throwaways. Writing
  the pattern into documentation text also trips it, so describe it rather than quoting it.

## Dispatching agents

> _Added 2026-09-26 (Claude Code · Opus 5) after two briefs in one hour pointed an agent at the
> wrong lane. Both were written from a dated defect report without checking which lane owns the
> work now. This section lives here, not in one session's memory, because it has to reach every
> session and every dispatch._

**Establish the lane before writing the brief.** A defect report, receipt, ADR or memory note
describes the code as of its own date. It does not establish which orchestrator drives the work,
which stage it belongs to, or which store it writes. Before briefing a fix, grep for the nearest
thing in the codebase that already does that job correctly, read it, and **name it in the brief**
so the agent inherits the architecture instead of re-deriving it. If no sibling exists, that
absence is the finding: it usually means a new seam, which is the owner's design decision.

**Use the right agent for the question.** Reach for a read-only explorer (`feature-dev:code-explorer`,
`Explore`, `Architect`, `Smart Explore`) to map an unfamiliar area *first*, then dispatch a builder
with what it found. A general-purpose builder handed an architectural question will implement
confidently in the wrong place.

**Every brief carries these, or the agent cannot comply:**

- **An explicit `model`.** Never leave it to inheritance. Cheapest tier that will do the job
  correctly — mechanical passes go cheap, reasoning and design go to the strong model. A model too
  weak for the task is not a saving.
- **A verification clause:** "Before writing code, confirm this brief against the code. If what
  you find contradicts it, stop and report instead of proceeding." An agent that is told the
  problem statement is provisional will surface a wrong brief; one that is not will build it.
- **Its own worktree** under `_worktrees/`, created from `origin/main`, never the shared checkout —
  other sessions hold uncommitted work there. Explicit-path staging only; never `git add -A`.
- **The machine's live constraints,** because agents rediscover these the hard way: the guard hook
  rejects any Bash command whose text contains a hard-delete pattern (the `rm` command with a force
  or recursive flag, or Python's file-removal call) anywhere, including inside a string, heredoc or
  commit message; Read/Write/Edit refuse paths outside the allowed working directories; no browser
  ever launches on this desktop; long jobs and databases live on the VPSs.
- **How follow-ups arrive:** say that corrections come by `SendMessage` from the parent session by
  name. Without that, an agent may treat a mid-task message as untrusted injection and refuse it.
- **Live validation, named concretely** — which host, which disposable schema, what to read back,
  and that test data is purged afterwards. "Verify it works" produces a claim; "insert X, read it
  back, drop the schema" produces evidence.
