# Propria monorepo migration plan

Status: active planning and root-bootstrap phase

Owner ruling: `E:\AI_Workspace\Projects\Propria` is the repository root

Destructive operations: prohibited

## Outcome

Create one understandable repository in which owned application code lives under
`projects/`, reusable source lives under `resources/`, and data/runtime state is
kept out of Git. Preserve every source repository until its imported history,
working-tree overlay, remote mapping, build, and representative file hashes have
been verified.

## Target shape

```text
Propria/
├── projects/
│   ├── consignatio/
│   │   ├── Intake/
│   │   ├── repair-tool-kit/
│   │   └── casebible/          # source/config only; corpus bytes remain external
│   ├── probata/
│   └── family-court-workbench/
├── resources/
│   ├── design/
│   ├── libraries/
│   ├── schemas/
│   ├── vendors/
│   └── references/
├── docs/
├── tools/
├── AGENTS.md
├── AGENT_MEMORY.md
└── CLAUDE.md
```

## Current verified risks

The 2026-09-12 inventory found:

| Source root | Branch/HEAD | Tracked changes | Untracked paths | Divergence |
|---|---:|---:|---:|---|
| `Consignatio/` | `main` / `fa4f249` | 16 | 69 | equal to `origin/main` |
| `Probata/probata/` | `main` / `99941cc` | 579 | 1,153 | behind 12, ahead 3 |
| `Probata/probata/modules/vestigia-geodata_processor/` | `main` / `e1a4acd` | 2,085 | 953 | equal to recorded upstream |
| `.../traceiq-rebuild/` | `ui-scaffold` / `0144af8` | 22 | 13 | equal to recorded upstream |
| `.../modules/custom/` | `master` / `761847a` | 12 | 60 | no upstream |

Clean Xplorer, build-kit, advocatio, and timesketch boundaries also exist. Their
clean status does not decide whether they become an imported owned subtree,
controlled vendor, or external upstream dependency.

## Migration rules

1. Never permanently delete or strip a source `.git` directory in place.
2. Never move a live dirty tree as the first migration action.
3. Never use `git add .`, `git add -A`, reset, clean, stash, or an unreviewed merge.
4. Never commit secrets, corpus/evidence bytes, databases, indexes, caches, model
   artifacts, worktrees, or cloud-mount contents.
5. Create a Git bundle and repository inventory for every source before import.
6. Import committed history in an isolated migration worktree using a path-prefixed
   history or unsquashed subtree merge; do not collapse unrelated histories into
   one opaque snapshot.
7. Capture dirty state separately as tracked diff, untracked manifest, and bounded
   content hashes. Overlay it only after the history import is verified.
8. Preserve source repositories as recovery copies until the owner accepts the
   final root and all required builds/tests pass. Any eventual retirement goes to
   `to_be_deleted/`; only the owner deletes it.

## Phases

### Phase 0 — Root bootstrap

- Initialize Git at Propria with `main` as the branch.
- Commit only root governance, target-directory contracts, the migration plan,
  manifest, and safe ignore rules.
- Create or attach a private remote only after verifying the intended repository
  name and visibility; never push nested source repositories as gitlinks by accident.

### Phase 1 — Classify boundaries

For every `.git` root, record ownership (`owned`, `upstream`, `donor`, `historical`,
or `worktree`), remote/fetch/push policy, license, dirty-state counts, head, upstream
divergence, and target path. Decide whether it is imported, retained as a submodule,
or represented as a dependency manifest.

### Phase 2 — Import clean owned histories

Start with the clean, smallest owned repository. Perform each import in an isolated
root worktree, verify commit ancestry and tree hashes, then merge the reviewed root
commit. Do not start with Probata or vestigia while they remain heavily dirty.

Progress, 2026-09-12: the former `FL-MCP/` directory was verified to be the
Propria-owned Family Court Workbench rather than a shared MCP layer. Its 96 source
files moved to `projects/family-court-workbench/`; 26,679 generated/dependency files
remain present but ignored. A compatibility junction preserves the old path. The
source had no `.git` history to import. From the canonical path, 14 sidecar tests
and 8 UI tests passed under a 2 GiB Node heap ceiling.

### Phase 3 — Reconcile dirty primary products

For Consignatio and Probata, identify concurrent owners, create non-destructive
state receipts, reconcile divergent branches in isolated worktrees, then import
committed history and overlay reviewed local work. Run project-specific validation
from the new path before marking a cutover.

Progress, 2026-09-12: Consignatio’s seven-commit history at `fa4f249` was
imported unsquashed under `projects/consignatio/`. The imported subtree tree hash
`45dffef2308ff9a3ace79f0a6810585307d64bb2` exactly matched the source HEAD tree.
The original source remains in place with 85 dirty/untracked paths observed after
the import. A reviewed overlay subsequently copied 78 files byte-for-byte into the
canonical subtree; the ordered copied-file manifest is
`3404a0d0d48f2b08f589a96fa7d79f47b435305422e41d9462131ee30016eb6e`.
One Intake routing file was reconciled manually. Three runtime/generated files,
including the corpus-transfer approval marker, were excluded. Canonical-path
validation passed 107 backend tests, 12 migration-tool tests, 19 frontend tests,
and the production frontend build. The source remains untouched as the recovery
copy by this migration, but its dirty-path count changed concurrently from 85 to
53 while the source HEAD remained `fa4f249`; final cutover therefore remains
pending. The detailed receipt is
`docs/CONSIGNATIO-OVERLAY-MIGRATION-2026-09-12.md`.

### Phase 4 — Shared resources and dependency policy

Move the design contract and genuinely shared libraries into `resources/` only
after finding every consumer. Replace old paths atomically with code/config/doc
updates. Retain third-party license and provenance files.

Progress, 2026-09-12: the 14-file shared design contract moved to
`resources/design/` after a bounded consumer search. A compatibility junction at
`design-contract/` preserves old local callers; root documentation now uses the
canonical path. No content was deleted.

### Phase 5 — Cutover

Update root and project `AGENTS.md`, `CLAUDE.md`, development commands, CI, Coolify
watch paths, Tailscale service source paths, Docstore source roots, and agent tool
configuration. Verify from a fresh clone before declaring the monorepo canonical.

### Phase 6 — Recovery-source retirement

After owner acceptance, move superseded source directories to the root
`to_be_deleted/` quarantine. Do not delete them. Record hashes, bundle locations,
replacement commits, and rollback instructions.

## Acceptance gates

- A fresh clone reconstructs all versioned source without undocumented local roots.
- No corpus, secret, database, cache, index, build output, or local worktree is tracked.
- Every imported project retains traceable history and a source-to-target receipt.
- Each project’s bounded tests/builds pass from its new directory.
- Agent instructions resolve from the root without contradictory commit guidance.
- Deployment/watch paths and Docstore indexing roots are explicitly updated and tested.
- The old source directories remain recoverable until the owner approves quarantine.
