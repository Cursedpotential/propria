---
title: Propria monorepo import — nine modules fold in with full history
date: 2026-09-26
status: imported
tags: [monorepo, repository-topology, git, propria, probata, consignatio, legal-desktop, vestigia, pii]
---

# Propria monorepo import (2026-09-26)

## Current routing

The import and repository cutover are complete. All imported products and
application forks commit at the Propria root; the child `.git` directories were
retired to `.review_hold/2026-09-26-retired-child-git/`. The old product remotes
are archives. Use the root `AGENTS.md` current routing table and
`docs/monorepo-migration-manifest.json` for active paths.

This status follows the existing root contract dated 2026-09-26 and the
independent confirmation in `modules/vestigia-geodata_processor/AGENTS.md`
dated 2026-10-02. It supersedes the pre-cutover checklist retained below. It
does not re-verify live service health or the historical credential observations.

_Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17; source read at
`68f369e`. Original decision and receipt: Claude Code · Opus 5 · 2026-09-26._

## Decision

The product repositories and the forks the application uses are imported into the
Propria repository with `git subtree`, preserving every commit. Owner decision
2026-09-26, after five days of asking for the conversion to be completed.

**Imported histories**, with current module paths:

| Module | Prefix | Source |
|---|---|---|
| Indicia Probata | `modules/Probata/probata` | `main` ∪ `origin/main` |
| Consignatio / Intake | `modules/Consignatio` | `main` ∪ `origin/main` (had diverged 39/8) |
| Advocatio legal workdesk | `modules/Legal-desktop` | `master` |
| Vestigia | `modules/vestigia-geodata_processor` | `main` |
| Vestigia app (formerly TraceIQ Rebuild) | `modules/vestigia-geodata_processor/vestigia` | `master`; imported as `traceiq-rebuild`, renamed 2026-10-02 |
| SBV forensic fork | `modules/Probata/probata/modules/forks/sbv` | `platform-sync` (gitlink replaced with real content) |
| Timesketch fork | `modules/Probata/probata/modules/forks/timesketch` | `master` (no remote at import) |
| Probata custom | `modules/Probata/probata/modules/custom` | `master` (no remote at import) |
| Xplorer copilot fork | `modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot` | `feat/hosted-intake-engine` |
| probata_build_crew | `modules/Probata/probata_build_crew` | no commits — source added as new files |

**Deliberately external development tooling:** the import excluded `memsearch`
and `claude-context`. The current root contract also lists `cocoindex-code`;
all three live under `E:/AI_Workspace/plugins/forks/` (the plugin marketplace,
moved out of `~/.claude` on 2026-09-28).

## Historical import receipt — 2026-09-26

The counts, working-tree observations and credential findings in this section
are the dated import receipt. They are not current desktop, runtime, publication
or credential-status verification.

### Why subtree rather than a flat import

`git subtree add` replays every source commit under the module prefix, so `git log`
and `git blame` keep working across the move. A flat one-commit-per-module import
would have kept Propria's history small but dead-ended blame at the import date.
The owner chose full history.

### Content excluded from the monorepo

A first attempt at this import committed evidence corpus and personal data,
because "merge it all" was read as "every branch and every untracked file"
without inspecting content. That branch was never pushed and was destroyed and
rebuilt. What it had wrongly included:

- **896 files of location history** under `raw_api_responses/` — reverse-geocoding
  responses carrying timestamps, coordinates and resolved addresses.
- **879 files** under `Consignatio/_intake/`, pulled in by merging
  `codex/casekit-ab`, whose own commit message reads *"initial local commit of the
  Case Bible tree (owner order 2026-09-09, stays local)"*. It held custody case
  strategy documents, a litigation plan and evidence PDFs.
- **26 files** from a protected holding directory.

The rebuilt import applies a path filter and a branch filter. Never admitted from
a working tree: `.env`, virtualenvs, `node_modules`, protected holding
directories, quarantine directories, `_intake/`, and the Vestigia corpora
(`raw_api_responses/`, `TraceIQ_Main/`, `TraceIQ_Backups/`, `TraceIQ_Evidence/`,
`traaceiq_mess/`, `Timeline.json`, `vault-sorted.7z`). Those paths are now listed
in the root `.gitignore` so they cannot drift back in.

Two branches are excluded **entirely, not even recorded as parents**, because a
parent commit would still place their objects in this repository and push them:

- `codex/casekit-ab` — the "stays local" Case Bible corpus, 1,118 files.
- `local-archive/pre-private-publication-20260911` — a pre-sanitisation snapshot.

Both were preserved in Consignatio's `.git` and in the verified bundles under
`.reconciliation/2026-09-26-pre-cutover-bundles/`. The root's completed-cutover
contract now locates that `.git` in the retired child-Git holding directory.

Six files from a protected holding directory remain, carried in Probata's own
committed history rather than added here: a stale `index.lock`, a temp block file,
naming-sweep output and one retired code chunk. They contain no personal data or
credentials, and removing them would require rewriting Probata's history.

### Secrets — observations recorded at import

A full-history sweep found live credentials at HEAD in Probata, Consignatio,
Vestigia and TraceIQ. Current trees are being redacted separately; history is not
being scrubbed, because that would break the commit SHAs this import preserves.

Probata, Consignatio and traceiq-rebuild already publish their histories on
GitHub, so this import creates no new exposure for them. **Vestigia is different**:
its remote `Cursedpotential/TraceIQ` returns 404, so its history — which contains
two PEM private key blocks and several API credentials — has never been published.
The owner was shown this and chose full history regardless, accepting that
publishing Propria publishes those keys. Rotation remains open and is the owner's
call.

### Verification recorded at import

Every local-only branch across all nine repositories is either reachable from the
import branch or on the excluded list above. Working trees, indexes and HEADs of
the live repositories were never touched: uncommitted work was captured through a
temporary index, and conflicts were resolved in throwaway worktrees.

Result: **3,097 commits, 15,745 files, clean tree, zero tracked `.env` files.**

## Historical pre-cutover checklist — 2026-09-26, superseded

At the time of this receipt, the import was on
`integration/monorepo-import-20260926`, unpushed, and the nine child `.git`
directories were still in place. The completed-cutover authority at the top of
this document replaces that state and its child-repository commit instructions.

The checklist recorded then was:

1. Resync Consignatio and Probata changes made since the fold.
2. Wait for the in-flight secret-redaction commits.
3. Drain or retire child worktrees; Probata's recorded count was 27.
4. Re-point Coolify; the recorded inventory had 32 of 35 apps building from
   `Cursedpotential/probata`, with six missing watch paths. The dated plan and
   per-app diff remain in `docs/MONOREPO-COOLIFY-CUTOVER-PLAN-2026-09-26.md`.
5. Retire each child `.git` and keep the old remotes as archives.
6. Switch the main checkout and verify file counts per module.

These are historical migration steps, not an instruction to repeat the cutover
or to commit in a retired child repository.

_Byline: Claude Code · Opus 5 · 2026-09-26_
