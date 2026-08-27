# TraceIQ Outer Repository — Agent Entry Point

> _Byline: Codex · GPT-5 · 2026-08-27._

This file governs `E:\AI_Workspace\Projects\traceIQ`. Before any Git action, run
`git rev-parse --show-toplevel` from the target file's directory and require that it equals this
directory. Stage only an explicit file allowlist; never use broad staging.

## Reconciliation hold

This checkout is not reconciled with its remote history. Read
`REPOSITORY_RECONCILIATION.md` before changing repository state. Until that hold is explicitly
cleared, do not checkout or restore paths, stage deletions, reset, clean, stash, delete, move, or
bulk-copy repository content. The owner-authorized additive governance files named in
`REPOSITORY_RECONCILIATION.md` may be committed with an exact allowlist; no reported deletion or
pre-existing content change may enter that commit.

## Nested repository boundary

`traceiq-rebuild/` is a separate Git repository with its own remote and commit history. Work on
that project from its own root and follow its own `AGENTS.md` and `AGENT_MEMORY.md`. Never stage
the nested repository, its files, or a pointer to it from this outer repository.

## Context rules

- Closest `AGENTS.md` wins for subtree-specific instructions.
- Read `AGENT_MEMORY.md` as a context router, not as authority over current files or Git state.
- Do not place secrets, credentials, private evidence content, or copied evidence text in agent
  instructions, memory routers, commit messages, or status documents.
- Preserve all existing data. Nothing is deleted or quarantined while the reconciliation hold is
  active.
