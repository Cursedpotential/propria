---
title: Propria monorepo import — nine modules fold in with full history
date: 2026-09-26
status: imported-pending-cutover
tags: [monorepo, repository-topology, git, propria, probata, consignatio, legal-desktop, vestigia, pii]
---

# Propria monorepo import (2026-09-26)

## Decision

The product repositories and the forks the application uses are imported into the
Propria repository with `git subtree`, preserving every commit. Owner decision
2026-09-26, after five days of asking for the conversion to be completed.

**Imported** at their existing paths:

| Module | Prefix | Source |
|---|---|---|
| Indicia Probata | `modules/Probata/probata` | `main` ∪ `origin/main` |
| Consignatio / Intake | `modules/Consignatio` | `main` ∪ `origin/main` (had diverged 39/8) |
| Advocatio legal workdesk | `modules/Legal-desktop` | `master` |
| Vestigia | `modules/vestigia-geodata_processor` | `main` |
| TraceIQ Rebuild | `…/traceiq-rebuild` | `master` |
| SBV forensic fork | `modules/Probata/probata/modules/forks/sbv` | `platform-sync` (gitlink replaced with real content) |
| Timesketch fork | `…/modules/forks/timesketch` | `master` (no remote — single copy) |
| Probata custom | `…/modules/custom` | `master` (no remote — single copy) |
| Xplorer copilot fork | `…/xplorer-copilot-buildkit/xplorer-copilot` | `feat/hosted-intake-engine` |
| probata_build_crew | `modules/Probata/probata_build_crew` | no commits — source added as new files |

**Deliberately excluded**, by owner rulings the same day: `memsearch` and
`claude-context` are development tooling rather than application forks, and move
to `~/.claude/local-plugins/forks/`.

## Why subtree rather than a flat import

`git subtree add` replays every source commit under the module prefix, so `git log`
and `git blame` keep working across the move. A flat one-commit-per-module import
would have kept Propria's history small but dead-ended blame at the import date.
The owner chose full history.

## Content excluded from the monorepo

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

Neither is lost. Both remain in Consignatio's own `.git` and in the verified
bundles under `.reconciliation/2026-09-26-pre-cutover-bundles/`.

Six files from a protected holding directory remain, carried in Probata's own
committed history rather than added here: a stale `index.lock`, a temp block file,
naming-sweep output and one retired code chunk. They contain no personal data or
credentials, and removing them would require rewriting Probata's history.

## Secrets

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

## Verification

Every local-only branch across all nine repositories is either reachable from the
import branch or on the excluded list above. Working trees, indexes and HEADs of
the live repositories were never touched: uncommitted work was captured through a
temporary index, and conflicts were resolved in throwaway worktrees.

Result: **3,097 commits, 15,745 files, clean tree, zero tracked `.env` files.**

## Not yet done: the cutover

The import lives on `integration/monorepo-import-20260926` and is unpushed. The
nine child `.git` directories remain in place and authoritative.

Remaining work, in order:

1. Resync — Consignatio and Probata have moved since the fold.
2. Wait for the in-flight secret-redaction commits to land.
3. Drain or retire the child worktrees; Probata alone has 27.
4. Re-point Coolify: 32 of 35 apps build from `Cursedpotential/probata`, and six
   have no watch paths, so every root commit would redeploy them. Plan and
   per-app diff are in `docs/MONOREPO-COOLIFY-CUTOVER-PLAN-2026-09-26.md`.
5. Move each child `.git` into quarantine; keep the old remotes as archives.
6. Switch the main checkout over and verify file counts per module.

Until step 6, product changes still commit in the child repositories.

_Byline: Claude Code · Opus 5 · 2026-09-26_
