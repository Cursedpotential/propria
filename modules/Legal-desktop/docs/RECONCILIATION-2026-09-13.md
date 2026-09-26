# Advocatio canonical directory reconciliation — 2026-09-13

Owner decision in this task: make `E:/AI_Workspace/Projects/Propria/Legal-desktop`
canonical and preserve its original build kit inside the application under resources.

## Completed layout

- `Legal-desktop/`: independent application Git repository, formerly
  `Probata/probata/modules/advocatio-legal_workbench/`.
- `resources/build-kit/`: original Legal-desktop directory contents, including
  the original build guide, artifacts/handoffs, source ZIPs, and extracted donors.
- `docs/`: application documentation remains in place.
- The former application path no longer exists. No compatibility junction was created.
- Empty relocation directory residue is preserved in
  `to_be_deleted/relocation-residue-20260913/`; only the owner may delete it.
- Original build material is reference material, not active application instructions.
  Its preservation does not install or enable any donor plugin or skill.

## Preservation verification

Before routing edits, SHA-256 checks passed for **4,918 inventoried files**:
1,238 original-kit files and 3,680 application files. All **606 Git metadata files**
matched their pre-move hashes, including the index. HEAD remained
`a44132f7fe854675a414a2b2b93b4dfbfa3af46c`, on the existing master branch.
No commit, push, reset, Git stash, or permanent deletion was performed.

The application had 42 modified tracked files (675 additions, 342 deletions)
and 23 untracked files before this work. Those changes were preserved.
Application edits from this task are routing/orientation in AGENTS.md,
AGENT_MEMORY.md, CLAUDE.md, README.md; preservation ignore rules in .gitignore;
and this receipt, resources README, and comparison artifacts.

The first PowerShell directory move was interrupted by hidden-directory handling
after some entries moved. Both locations were inspected; remaining directories
were moved without overwriting existing destinations. Final hash verification
returned no missing files and no mismatches.

Private environment files, runtime data, caches, virtual environments, Git internals,
reparse points and checksum sidecars are excluded from the 4,918 content comparison.
Git internals were verified separately. Other excluded paths were carried with their
directories but were not included in byte-comparison claims. See excluded-paths.csv.
ZIPs were compared as file bytes, not recursively unpacked for member comparisons.

## What differs

There were no shared relative file paths between the two original directory layouts,
after exclusions. Cross-path SHA-256 matching found:

- 1,148 of the 1,238 original-kit files have identical bytes somewhere in the application.
- 90 original-kit files have no identical-byte counterpart.
- Four of those 90 have corresponding documents: the build guide, Agno Platform
  Interface & Integration Analysis, Category 1 Persistence & Settings handoff,
  and Legal Terminal & Legal MCP Deep Analysis Part 1. Their differences are
  the application copies' September naming annotations.
- The other 86 have no confirmed counterpart by donor-aware path matching.
  This is not proof they are absent from ZIP member contents.
- 1,967 of the 3,680 application files have no identical-byte original-kit counterpart.

All variants remain preserved. No donor/document deduplication was performed.

[Open the readable comparison](reviews/2026-09-13-reconciliation/comparison.html).
It contains the four document diffs, the full 42-file uncommitted application patch,
the 23 new-file additions, and links to exhaustive comparison tables.
Authentication changes and Propria theme adoption are pre-existing work;
this task does not claim to have implemented or tested those capabilities.

## Routing and runtime

Updated the Projects router/boundary map, Propria router/memory, migration manifest,
Docstore source-root registry, Probata router/memory, and application orientation.
The source registry retains project ID advocatio and excludes resources/build-kit
from active application documentation ingestion. The kit is kept local and ignored
to avoid automatically staging large archives. Future governed knowledge-base
registration can select appropriate source material.

Both JSON registries parse; Propria diff whitespace validation passes.
The relocated Python editable-install .pth was updated to the new api directory;
importing legal_workspace with the moved virtual environment resolves to the new path.
Other generated console launchers/indexes may embed former absolute paths and should
be refreshed through their owning tools when used. Dependencies were not rebuilt;
remote indexes and production services were not changed or verified.

Smart Explore's verified canonical launcher was used for structural discovery and
selected-store recall. Docstore was unavailable through the configured Search adapter
(PROPRIA_DOCSTORE_ADAPTER absent), and no native Docstore tools were exposed here.
Results are persisted locally; governed Docstore capture, flags, and remote index
freshness remain pending and unverified.

## Historical follow-up

Application Git history begins at d8ea734 on 2026-08-18 at 14:17:40 -04:00;
the original build guide is dated 2026-08-17. This is commit/document evidence,
not proof of the earliest physical file creation. Full conversation archaeology
and product/knowledge-base analysis remain follow-up work after this relocation.
