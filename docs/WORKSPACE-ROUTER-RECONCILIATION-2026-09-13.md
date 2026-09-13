# Workspace Router Reconciliation — 2026-09-13

## Result

The live instruction chain now routes from the unversioned filesystem workspace
through the Propria monorepo without relying on the retired
`the-platform-workspace`, workspace-root `casebible`, or external
`Projects/_worktrees` locations.

The current boundary is:

- `E:/AI_Workspace` and `E:/AI_Workspace/Projects` are unversioned filesystem
  routers.
- `E:/AI_Workspace/Projects/Propria` is the canonical monorepo Git root.
- `Consignatio` is the canonical independent Consignatio/Intake Git repository.
  `projects/consignatio` is a duplicate imported overlay pending reconciliation
  and is not active authority.
- `Probata/probata` remains a separate, dirty child Git repository pending
  import.
- new and relocated Propria-owned linked worktrees live under
  `Propria/_worktrees`.

## Files changed

Unversioned workspace routers:

- `E:/AI_Workspace/AGENTS.md`
- `E:/AI_Workspace/AGENT_MEMORY.md`
- `E:/AI_Workspace/Projects/AGENTS.md`
- `E:/AI_Workspace/Projects/AGENT_MEMORY.md`
- `E:/AI_Workspace/Projects/REPOSITORY_BOUNDARIES.md`

Tracked Propria routers and state:

- `AGENTS.md`
- `AGENT_MEMORY.md`
- `CLAUDE.md`
- `docs/monorepo-migration-manifest.json`
- this receipt

The independent, clean Milvus repository also had its root assertion corrected in
`E:/AI_Workspace/Projects/dev-resources/milvus-coolify/AGENTS.md`. That change is
committed in the owning repository separately.

## Canonical Consignatio authority correction

A post-publication ownership check verified that
`E:/AI_Workspace/Projects/Propria/Consignatio` is an independent Git root on
`main` at `fa4f249a5c9d69bb7964ea851e71e8cecbab08a9`, with remote
`https://github.com/Cursedpotential/Consignatio.git`. The root and workspace
routers now preserve that repository as canonical authority. The tracked
`projects/consignatio` tree is explicitly classified as a duplicate imported
overlay pending reconciliation. No Consignatio product file was changed.

## Worktree reconciliation reflected in the manifest

`git worktree list --porcelain` from the Probata common repository verified:

- `E:/AI_Workspace/Projects/Propria/Probata/probata`
- `C:/Users/matts/.codex/worktrees/41e3/probata`
- `E:/AI_Workspace/Projects/Propria/_worktrees/probata-docstore-recall-latency`
- `E:/AI_Workspace/Projects/Propria/_worktrees/probata-duckdb-live-test`
- `E:/AI_Workspace/Projects/Propria/_worktrees/probata-integration-20260913`
- `E:/AI_Workspace/Projects/Propria/_worktrees/probata-tool-runtime-rename`

The manifest no longer advertises the two external `Projects/_worktrees` paths
or the retired `.claude/worktrees` Docstore path.

## Coverage and exclusions

The filesystem discovery found 555 instruction/router candidates named
`AGENTS.md`, `AGENT_MEMORY.md`, `CLAUDE.md`, `REPOSITORY_BOUNDARIES.md`,
`COORDINATION.md`, or `*.code-workspace`. An automated first pass retained 122
review candidates after generated/worktree/archive/vendor exclusions; remaining
donor-path false positives were classified manually.

Excluded from mutation:

- `dev-resources/Archives/**`, `dev-resources/doc-classify-donors/**`,
  `dev-resources/upstream-resources/**`, and other donor/reference copies;
- `to_be_deleted/**`, `_stale/**`, vendored dependencies, caches, build output,
  and virtual environments;
- `_worktrees/**`, `probata-worktrees/**`, `.claude/worktrees/**`, and
  `.codex/worktrees/**` generated copies;
- Legal Workspace and Family Court product/plugin content, per the task boundary;
- the dirty `Probata/probata` checkout's already modified root instruction files;
  and
- historical receipts and handoffs, where old paths remain provenance rather
  than executable routing.

The active `Probata/probata/probata.code-workspace` parses as JSON, contains only
the current repository-relative `.` folder, and requires no path correction.

## Unversioned-router custody

Before editing, exact copies were preserved under
`E:/AI_Workspace/to_be_deleted/2026-09-13-router-reconciliation`. Only the owner
may delete that quarantine material.

| File | SHA-256 before | SHA-256 after |
|---|---|---|
| `E:/AI_Workspace/AGENTS.md` | `7F1FD836F398A479CA362C6FE8F38B42EB030379FBFC1D4E89574B7C54915A5B` | `1A39499CF6C4265A8B48BB831CF2BD732CE64CAD90B97F85E0E8FD8BB540C262` |
| `E:/AI_Workspace/AGENT_MEMORY.md` | `667505FAA6597E0ECB10231CC1D25F12DF76628EC9C6F04B7AB9DFAC63C37EE5` | `17957D63692D9D2854E3158C66F7915408B9F3E3AA4A2EB95A22B1B24944CF1D` |
| `E:/AI_Workspace/Projects/AGENTS.md` | `2E7AAD46FD43EA8867AEEF522BAE85B73BB8FFCA2A0275FFD06CF8CC28D6E06B` | `F9D37EAAAEF7EAEEB93E88DC75B324207A9E2CE9EFEBF43C1F2A7CAA528913AA` |
| `E:/AI_Workspace/Projects/AGENT_MEMORY.md` | `15C33287C58C7B2EF77B0904B9BEF88416B4268AC97C9F354F9E8442B1B67C63` | `B3BDD102FA57C409A31CEAA6D8D7E156E85856900CAA1354875061E59381467A` |
| `E:/AI_Workspace/Projects/REPOSITORY_BOUNDARIES.md` | `37AD9FAF4B76B8CF17981974A00EF8C699653ED937BE5C3812F0C83F1C5F19AC` | `3E673A4B7D8E3A0ED4D97DEC1EC120EAB19ACD8F99E094517E8C27079EA7EF95` |

These files cannot be committed because neither filesystem-router directory is a
Git repository. The hashes and preserved originals make the change reviewable.

## Validation

- Every live path in the current boundary table exists.
- Propria's migration manifest parses as JSON.
- The active Probata VS Code workspace parses as JSON.
- The manifest's worktree list matches `git worktree list --porcelain` for the
  managed lanes.
- `git diff --check` passes in each changed Git repository.
- Only explicit router/state paths were staged and committed.

## Git publication

- Propria router/state implementation: `36793b72328e414bce8a47c8a3b646c2f6d2f432`,
  pushed to `origin/main`.
- Independent Milvus root-route correction:
  `0d66d35f404f799b3e3dd926a089eb067b8ea690`, pushed to that repository's
  `origin/main`.

## Remaining integration boundary

Probata's root instruction files are concurrently modified in a heavily dirty
checkout. They still contain some earlier module-route prose, so they were not
rewritten here. Their final router correction belongs in the existing Probata
integration branch after its concurrent work is reconciled. This receipt and the
workspace/Propria routers already use the verified live module names
`advocatio-legal_workbench` and `vestigia-geodata_processor`.
