# Consignatio — current source and historical repository checkpoints

## Current source

Consignatio, Intake and both imported Xplorer source folders are part of
[`Cursedpotential/propria`](https://github.com/Cursedpotential/propria), under
`modules/Consignatio/`. Use the existing Propria checkout and commit at its root.
The [root contract](../../AGENTS.md) and completed
[September 26 import](../../docs/decisions/2026-09-26-monorepo-import.md)
supersede the separate-clone and per-product push instructions formerly here.

## Historical source backup — 2026-09-11

The following repository names, branches and checkpoints describe the September 11
backup. They are provenance for the import, not current checkout or deployment targets.

| Source | Private repository | Branch / checkpoint |
|---|---|---|
| Vault plans, Intake UI/backend, original development tools | https://github.com/Cursedpotential/Consignatio | `main`; source-only root `aa3b9bc7` plus publication documentation |
| Native Intake desktop (Xplorer fork) | https://github.com/Cursedpotential/Intake-desktop | `feat/acp-copilot`, `6990b7ed` |
| Existing Xplorer build kit | https://github.com/Cursedpotential/xplorer-copilot-buildkit | `main`, `d0954e37` |
| Repair toolkit source | https://github.com/Cursedpotential/Consignatio | `main`, under `repair-tool-kit/`; source-only integration from local `cfe951a` and its preserved working changes |

## Local history is preserved, not published

The original parent root commit `161d3c4` contained 879 imported conversation/corpus
files. It was explicitly local-only. The new `main` starts from a source-only root;
the original commit and relocation staging are preserved locally under
`local-archive/pre-private-publication-20260911` at `01ac582e`.
No working files were removed or restored over user edits.

The linked worktree `repair-tool-kit-codex/` stays on local `codex/casekit-ab` at
`cfe951a`, preserving its inherited history. Its 36 new source files were committed
locally and exported as a separate source-only root on `repair-toolkit-snapshot`.
The snapshot branch remains a historical backup. On 2026-09-20 the module source
was copied into main under `repair-tool-kit/`, including its tested engine additions,
without merging the archival ancestry. Existing planning files remain, with the
original plan README in `repair-tool-kit/BUILD-KIT-README.md`. Do not push or merge
`codex/casekit-ab` or the archival branch.

**Never use `git push --all` or `git push --mirror` here.** Push named source branches
only. The old corpus history is not part of the private source backup.

## Excluded from publication

Imported corpus, evidence bytes, ledger caches, databases, generated indexes,
copy-operation reports, local agent memories, runtime/build/dependency directories,
environment credentials and quarantine directories stay on disk and are ignored.
Original development scripts, schema/design documents and plans remain included.
Private source backup is not a backup of the owner's evidence corpus or credentials.

## Verification receipt

- Gitleaks 8.30.1 official Windows release, archive SHA-256 matched release checksum.
- Source-only parent root: 1 commit / approximately 3.46 MB scanned; no findings.
- Desktop inherited history: 1,172 commits / approximately 30.09 MB; no findings.
- Desktop staged implementation: approximately 116.59 KB; no findings.
- Repair toolkit staged source: approximately 177.07 KB; no findings.
- Reports are redacted and stay locally under
  `E:\AI_Workspace\.intake-dev\repo-audit`; credentials were not rotated or printed.
- Secret scanning is a bounded check, not proof that arbitrary content cannot
  contain sensitive information. Corpus directories were excluded independently.
- No application build, service restart, corpus scan or model workload was started
  as part of Git publication. Prior scoped test results remain in the Intake handoff.

The September 11 handoff is historical. Current application routing starts at
`AGENTS.md` and `AGENT_MEMORY.md`, then the selected Intake lane.

_Routing reconciliation: ChatGPT · GPT-6 Astra · 2026-10-17._
