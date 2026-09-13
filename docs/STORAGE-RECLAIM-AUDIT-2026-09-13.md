# Storage Reclaim Audit — 2026-09-13

Status: paused after safe quarantine because Smart Explore/CCC repair became the
higher-priority lane. No file was permanently deleted.

## Newly quarantined rebuildable caches

The completed, clean worktree
`E:/AI_Workspace/Projects/Propria/_worktrees/probata-tool-runtime-rename`
contained five ignored, untracked cache/build directories. Each source was
verified with `git check-ignore -v` and `git ls-files --error-unmatch`; the
worktree remained clean after the moves.

| Original path relative to worktree | Bytes |
|---|---:|
| `.venv` | 457,095,193 |
| `modules/workbench/web/node_modules` | 436,065,973 |
| `.mypy_cache` | 115,937,918 |
| `modules/workbench/web/dist` | 5,144,456 |
| `tests/__pycache__` | 4,236,578 |
| **Total newly quarantined** | **1,018,480,118** |

They now live under this single owner-deletable folder:

`E:/AI_Workspace/Projects/Propria/_worktrees/probata-tool-runtime-rename/to_be_deleted/storage-reclaim-20260913`

Deleting that folder permanently is an owner action. The containing worktree's
entire `to_be_deleted` tree is now 1,084,665,846 bytes across 51,237 files,
including a pre-existing 66,185,728-byte payload that was not inspected or
altered.

Two apparent `node_modules` candidates under
`modules/workbench/design-mockups/unified-operator-surface` were retained in both
completed worktrees because Git proved that their native binaries are tracked.

## Existing quarantine payloads

Size inspection used path and filesystem metadata only. Contents were not opened.
The largest existing owner-controlled quarantine roots found before the new move
were:

| Owner-deletable quarantine folder | Bytes | Files |
|---|---:|---:|
| `E:/AI_Workspace/Projects/typi/to_be_deleted` | 3,625,125,074 | 1,142 |
| `E:/AI_Workspace/Projects/Propria/to_be_deleted` | 1,282,661,369 | 52,318 |
| `E:/AI_Workspace/Projects/Propria/_worktrees/probata-duckdb-live-test/modules/to_be_deleted` | 876,243,003 | 16,315 |
| `E:/AI_Workspace/Projects/mitech-consult-site/to_be_deleted` | 257,374,619 | 13,290 |
| `E:/AI_Workspace/Projects/Propria/.claude/worktrees/ecstatic-tu-7c6c6c/TraceIQ/Junkyard/Source_A_Root_Folder/TO_BE_DELETED` | 102,050,787 | 1,966 |
| `E:/AI_Workspace/Projects/Propria/.claude/worktrees/pdf-page-deletion-tool-3a4414/TraceIQ/Junkyard/Source_A_Root_Folder/TO_BE_DELETED` | 102,050,787 | 1,966 |
| `E:/AI_Workspace/Projects/Propria/Consignatio/to_be_deleted` | 96,919,088 | 117 |
| tool-runtime worktree `to_be_deleted` before this audit | 66,185,728 | 1 |

The two `.claude/worktrees` rows may contain duplicate generated checkout data,
but they were retained because their worktree registration and uniqueness were
not proven before the priority change. The Consignatio quarantine was also
retained without content inspection because its canonical checkout is active and
dirty.

The raw metadata inventory is preserved at
`E:/AI_Workspace/to_be_deleted/storage-reclaim-20260913/to-be-deleted-sizes.json`.
This audit scratch directory is itself owner-controlled quarantine.

## Large retained cache and active lanes

- `E:/AI_Workspace/.cache` is 2,117,342,556 bytes across 34,172 files. It was
  retained because it is shared workspace cache state rather than cache owned by
  a completed clean worktree.
- `probata-integration-20260913` was not inspected or altered because it is the
  active integration worktree.
- `probata-docstore-recall-latency`, Docstore agent paths, and running test
  directories were not inspected or altered.
- The canonical dirty Consignatio checkout and its R2/B2 reconciliation worktree
  were not altered.
- No evidence or source-corpus content was opened; only path and size metadata was
  used.

## Reclaim boundary

Moving caches into quarantine does not itself free disk space. The owner can
reclaim 1,018,480,118 bytes from the newly created storage-reclaim folder, plus
the byte counts shown for existing `to_be_deleted` roots, by permanently deleting
only the specific quarantine folders they choose. Counts should not be summed
across a parent and nested quarantine path if both appear in a future inventory.
