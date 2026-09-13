# Propria root reconciliation receipt — 2026-09-13

## Scope

This receipt covers only the Git repository rooted at
`E:/AI_Workspace/Projects/Propria`. No nested repository source, linked worktree,
runtime service, deployment, database, or index was changed.

## Canonical topology

- `E:/AI_Workspace/Projects/Propria` is the monorepo root.
- Root-owned application source converges under `projects/`; shared source and
  governed references converge under `resources/`.
- `Probata/probata` remains a transitional child Git repository. Its future
  target is `projects/probata`, which does not yet exist.
- `Consignatio` remains a dirty recovery source after its committed history and
  reviewed overlay were imported into `projects/consignatio`.
- New Propria-owned worktrees belong under the ignored
  `E:/AI_Workspace/Projects/Propria/_worktrees/` directory. A child repository
  may link a worktree there during migration without flattening its Git boundary.

## Verified state before this reconciliation commit

The root fetched `origin` successfully. `origin/main` was
`3e155fc997e848f1260df6b9ca8642cb3dba2f3d`; local `main` was one commit ahead at
`fba6f9b74344f6854c973b51f1ce860240c6bc37` and zero commits behind. That local
commit is the already-validated Family Court Workbench design adoption.

The only uncommitted root-owned governance paths were the migration-plan
cutover gate, the Docstore source registry, and the verified Probata
function-access failure flag. `.cnf/memories/project_memory.json` is machine-local
runtime state and is now explicitly ignored.

The required Docstore registry roots all resolved in this local checkout:

| Project ID | Source root | Ingestion claim |
|---|---|---|
| `propria` | `.` | pending |
| `probata` | `Probata/probata/docs` | current full source |
| `consignatio` | `projects/consignatio` | pending multi-root CDC |
| `family-court-workbench` | `projects/family-court-workbench` | pending multi-root CDC |
| `advocatio` | `Probata/probata/modules/advocatio-legal_workbench` | pending multi-root CDC |
| `vestigia` | `Probata/probata/modules/vestigia-geodata_processor` | pending multi-root CDC |

This proves path existence, not deployment, indexing, CDC, or retrieval. File
counts are intentionally omitted because raw recursive enumeration crosses nested
repositories and generated dependency trees before registry filters are applied.

Probata was observed at
`2a26edfcd31c319a3bff3e3c1247f070dab92bdf`, ahead 4 and behind 22, with 654
status paths. Its linked-worktree registry included the main checkout, one Codex
detached worktree, two branch worktrees under the historical global
`Projects/_worktrees` directory, and one nested Claude worktree. All were
preserved in place.

## Changes reconciled

- Corrected the root memory router to use current Propria, Probata, Consignatio,
  and advocatio paths rather than retired `the-platform-workspace` authorities.
- Corrected the active advocatio route in the root contract.
- Documented `_worktrees/` as the ignored Propria-owned convention and recorded
  the safe gate for eventual relocation of existing linked worktrees.
- Updated the migration manifest with the current Probata HEAD, divergence,
  status-path count, target absence, and linked-worktree inventory.
- Versioned the Docstore source registry with an explicit path-only verification
  boundary. The registry preserves all `pending-multi-root-cdc` states; no false
  indexing-success claim was added.
- Preserved and versioned the independently verified function-access failure flag.

## Gated Probata import sequence

1. Pause every Probata writer and capture each checkout's branch, HEAD, dirty
   status, untracked manifest, `.git` pointer/common directory, and bounded hashes.
2. Reconcile Probata's ahead/behind history and the owned branch worktrees without
   reset, clean, stash, force push, or raw directory moves.
3. Create a recovery bundle and independently preserve the reviewed dirty overlay.
4. Import committed Probata history into an isolated Propria root worktree under
   `projects/probata`, preserving ancestry and verifying tree hashes.
5. Overlay only reviewed owned files, then run Probata's bounded tests, builds,
   deployment-path checks, and Docstore cutover gates.
6. Keep the source repository recoverable until owner acceptance. Any retirement
   moves to `to_be_deleted/`; only the owner deletes it.

## Publication boundary

The root reconciliation is published on `main` together with the preceding
Family Court Workbench commit. The exact pushed commit range and remote result are
reported by the completing agent after Git validation. This receipt itself makes
no claim that Probata, Docstore multi-root CDC, or any runtime deployment is cut
over.
