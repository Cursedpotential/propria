# Propria repository-boundary inventory

Verified: 2026-09-12

Purpose: identify the actual Git boundaries that must survive the transition to
the Propria monorepo. Counts are working-tree snapshots, not permanent facts.
Recommendations are migration guidance, not owner rulings about product scope.

## Primary roots

| Root | HEAD | State | Intended disposition |
|---|---:|---|---|
| Propria | `fcf564e` | clean, private `Cursedpotential/propria` | canonical monorepo root |
| Consignatio | `fa4f249` | 16 tracked changes, 68 untracked; 7 commits | import as `projects/consignatio` only after dirty-state reconciliation |
| Probata | `99941cc` | 579 tracked changes, 1,153 untracked; 12 behind/3 ahead; 1,046 commits | reconcile divergence and concurrent work before importing as `projects/probata` |

## Nested repositories

| Boundary | Representation in parent | Snapshot | Recommended migration treatment |
|---|---|---|---|
| Intake desktop Xplorer fork | ignored by Consignatio | clean `feat/acp-copilot` at `6990b7ed`; public remotes fetch-only, private remote writable | retain private fork boundary during initial Consignatio import; reconsider subtree only after update policy is explicit |
| Xplorer copilot build kit | ignored by Consignatio | clean `main` at `d0954e3` | small owned history; subtree candidate after parent import |
| Consignatio repair-tool-kit Codex checkout | linked worktree, ignored | branch `codex/casekit-ab` at `cfe951a` | never import as a second repository; preserve worktree until branch is integrated |
| Advocatio Legal Workbench | independent nested repo | clean `master` at `a44132f` | preserve history and decide whether it remains a separately releasable project or becomes a Probata subtree |
| Probata custom modules | ignored local repo | 12 tracked changes, 60 untracked; one commit; no remote | inspect local-only content before any import |
| SBV | raw mode-160000 gitlink | dirty `platform-sync` at `6927a23` | preserve boundary while fork-versus-donor extraction remains active |
| Timesketch | ignored nested repo | clean one-commit local snapshot at `00eff7d`; no remote | treat as controlled vendor/dependency pending provenance review |
| Vestigia/TraceIQ | untracked nested owned repo inside Probata | 2,085 tracked changes, 953 untracked at `e1a4acd` | reconcile independently, then import as top-level `projects/vestigia` rather than hiding it inside Probata |
| TraceIQ rebuild | ignored inside Vestigia | 22 tracked changes, 13 untracked at `0144af8`; additional linked worktrees exist | preserve worktrees and reconcile before choosing subtree/project disposition |

## Non-repository top-level material

- `resources/design/` is imported and verified; `design-contract/` is a
  compatibility junction.
- `projects/family-court-workbench/` is imported and tested; `FL-MCP/` is a
  compatibility junction.
- `Legal-desktop/` is a parts/reference tree with a large archive and requires a
  provenance/secret/duplication inventory before any move into `resources/`.
- Root database snapshots, scan databases, patches, worktrees, agent stores,
  credentials, caches, and generated logs are explicitly ignored. They remain
  local and are not monorepo source.

## Next safe sequence

1. Reconcile Consignatio’s 84-path dirty state and identify current file owners.
2. Produce a content-free manifest separating versioned source from the very
   large corpus/runtime areas under Consignatio.
3. Import Consignatio history into an isolated Propria worktree and verify its
   tree before overlaying local work.
4. Leave Probata and Vestigia in place until their much larger dirty/divergent
   states have dedicated reconciliation receipts.
