# Consignatio — agent contract

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

See [CCC / Intake / Docstore boundaries](../../SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

Consignatio is the product module at
`E:/AI_Workspace/Projects/Propria/modules/Consignatio` in the single
`Cursedpotential/propria` repository. Product code, documentation and tests commit
at the Propria root, with explicit paths under this module. The completed
[2026-09-26 import](../../docs/decisions/2026-09-26-monorepo-import.md) and
[root contract](../../AGENTS.md) supersede the former independent-repository routing.

_Routing reconciliation: ChatGPT · GPT-6 Astra · 2026-10-17; existing owner import decision._

## Safety

- **Owner standing storage rule (2026-09-11): E: is the designated development
  drive.** Project files, dependency downloads, tool caches, build outputs,
  runtime data and job temporary files must use E:, not the Windows C: drive.
  Check effective paths before installations/builds, including child processes,
  TEMP/TMP, Cargo/Rustup, npm/pnpm and Python/uv caches. No silent fallback to C:.
  If a tool cannot comply or E: lacks space, stop that operation and report it.
  Do not substitute D: without owner direction. Existing shared caches require
  a verified, coordinated relocation; never move active stores blindly.
  This is a rule for agent-controlled workloads, not a claim that Windows or
  third-party applications perform zero incidental writes to C:.

- Never permanently delete files. Move material that must be removed into this
  repository's `to_be_deleted/` directory; only the owner may delete from there.
- Never reset, clean, stash, broad-stage, or overwrite concurrent work.
- Never stage evidence bytes, databases, generated indexes, exports, credentials,
  or machine-local configuration.
- Before staging or committing, run `git rev-parse --show-toplevel` from the target
  path and confirm it resolves to the Propria monorepo root
  (`E:/AI_Workspace/Projects/Propria` in the configured desktop workspace).
- Stage only explicit, reviewed paths. Never use `git add .` or `git add -A`.
- Never initiate host reboot, restart, shutdown, sleep, hibernation or logoff.
- Do not hydrate cloud placeholders, scan a corpus, start recurring workers or
  run local model inference as an incidental development step. Use small explicit
  fixtures, bounded concurrency, timeouts and per-instance state.

## Load order and execution

Read `AGENT_MEMORY.md` next; it routes to the specific application or legacy lane.
Load applicable installed skills before implementation, especially user-named
skills. If unavailable, identify that limitation rather than pretending to use them.
Use the current request as the scope: diagnosis is read-only; authorized builds
should reach a tested slice without repeatedly asking permission for ordinary edits.
Do not replace approved technologies or expand scope without owner approval.
Keep plans, implementation and verified results distinct. Record material work in
the lane's development receipt and TODO. Never label a simulated job as real work.

## Authority boundary

The live filesystem organizer is stage one. Its browsing, search, grouping and
user-directed file operations do not require evidence acceptance or candidate
review. The following evidence/review boundaries apply when working in those
downstream workflows, not as a prohibition on organizing the owner's files.
Intake's filesystem CocoIndex app and Surreal database are separate from the
downstream evidence index and evidence-analysis database.

- PostgreSQL and governed custody workflows remain canonical for Platform state.
- The Workbench is a human review and orchestration surface, not a second evidence
  store or approval authority.
- Source media in Immich or PhotoPrism remains unchanged unless a separately
  approved, receipt-producing connector action says otherwise.
- Imported source values are immutable. Human annotations and machine proposals
  live in separate, provenance-bearing overlays.

## Application

The desktop Intake application lives under `Intake/`, with its corpus backend in
`Intake/backend/`. The original deduplication, metadata and vault-structure work is
preserved under `casebible/`, alongside Intake. Read `Intake/docs/` before
changing its contracts or interface. Keep the browser UI usable independently of
the Tauri host; privileged filesystem and process operations must pass through
explicit, least-privilege native commands.

## Imported Xplorer source

Both Xplorer source folders belong to the Propria monorepo. Read their applicable
local instructions and commit reviewed changes at the Propria root.

| Component | Path relative to this module | Import authority |
|---|---|---|
| Xplorer copilot build kit | `Intake/xplorer-copilot-buildkit/xplorer-copilot-buildkit/` | Root history import, 2026-09-23 |
| Xplorer fork (`xplorer-copilot`) | `Intake/xplorer-copilot-buildkit/xplorer-copilot/` | Monorepo import, 2026-09-26 |

`REPOSITORIES.md` records the former publication boundaries. The old per-product
remotes are archives; current work goes to `Cursedpotential/propria`. The excluded
local archival branches and corpus history stay excluded, as documented in the
root import decision. Preserve linked worktrees and their branches; their presence
does not establish a separate product repository.

## Execution direction

<!-- Moved here from `CLAUDE.md` on 2026-09-26 when that importer was removed. -->
> _Byline: Claude Code · Opus 5 · 2026-09-26_

- Follow the current owner request and the applicable local contract. Read the
  relevant skill instructions before acting; listing a skill is not using it.
- Briefly state scope, owned paths and verification, then execute the authorized
  slice. Do not substitute another architecture, indexing engine or database.
- Stop for a consequential scope change or unsafe ambiguity, not every routine
  implementation step. If told to stop, stop mutations and your background work.
- Prefer bounded tests and inspect-first diagnostics. Never infer permission for
  corpus ingestion, cloud hydration, model downloads or a persistent worker.
- Preserve concurrent work and credentials. Do not dump secrets into commands,
  permission rules, logs or prompts. Report failures honestly; no fabricated
  successful jobs, provider results or deployment claims.
- Finish with changed paths, test evidence and the next concrete test the owner
  can perform. These instructions are guardrails, not a guarantee of model
  behavior.

### Receipts and where file-truth lives

- `docs/receipts/` holds receipts, catalog exports, hash ledgers, zero-file lists
  and missing lists. Payloads stay gitignored; the `.md`, `.py`, `.sql` and `.txt`
  files are tracked. The tools under `casebible/tools/` write here.
- PostgreSQL `raw_duck` on ovh-files is the only complete catalog. `docs/receipts/`
  is its human-readable side, and `docs/LOG.md` is the only log (open items: `docs/URGENT-TODO.md`; finished: `docs/COMPLETED-TODO.md`; owner 2026-10-02). Do not
  create another index or README for this — extend the log's 2026-09-15 map instead.

## Claude-Reflect Learnings

<!-- Auto-generated by claude-reflect. Do not edit this section manually. -->

### Project Conventions
- Receipts, catalog exports, hash ledgers and missing lists live in `docs/receipts/` (payloads gitignored); never a workspace-root scratch folder. (owner 2026-09-15)
- One log: `docs/LOG.md`. Record a change there once; no README/memory/handoff fan-out of the same fact. (owner 2026-09-15)
- To-dos: `docs/URGENT-TODO.md` holds open items only; a finished item moves to `docs/COMPLETED-TODO.md` with date and proof in the same turn. (owner 2026-10-02)
- Fix known bugs in our own apps instead of working around them; if blocked, package the patch with proof and name the blocker. (owner 2026-09-15)

<!-- End claude-reflect section -->

## Catalog and lake usage (owner-requested guide, 2026-10-04)
Before creating or refreshing a catalog or lake projection, read [the Case Bible catalog guide](docs/CASE-BIBLE-CATALOG-GUIDE.md). Use the existing all-bucket inventory and registry; dated Parquet is a published generation, not another independently maintained catalog. Check generation and coverage. Preserve historical observations. The owner expressly requested this guide, superseding the older prohibition on a separate guide for this task.
