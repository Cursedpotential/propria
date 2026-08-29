# TraceIQ Outer Repository — Agent Entry Point

> _Byline: Codex · GPT-5 · 2026-08-27; reconciliation resolved 2026-08-29._

This file governs `E:\AI_Workspace\Projects\traceIQ`. Before any Git action, run
`git rev-parse --show-toplevel` from the target file's directory and require that it equals this
directory. Stage only an explicit file allowlist; never use broad staging.

## Reconciled boundary

The former tracked-missing/untracked collision hold was resolved on 2026-08-29. Read
`REPOSITORY_RECONCILIATION.md` for proof. The child checkout is clean and pushed; the workspace
router represents it as one raw Gitlink. Local dumps, evidence files, timelines, tool memory, and
`traceiq-rebuild/` remain physically present but are intentionally ignored by this repository.

## Nested repository boundary

`traceiq-rebuild/` is a separate Git repository with its own remote and commit history. Work on
that project from its own root and follow its own `AGENTS.md` and `AGENT_MEMORY.md`. Never stage
the nested repository, its files, or a pointer to it from this outer repository.

## Context rules

- Closest `AGENTS.md` wins for subtree-specific instructions.
- Read `AGENT_MEMORY.md` as a context router, not as authority over current files or Git state.
- Do not place secrets, credentials, private evidence content, or copied evidence text in agent
  instructions, memory routers, commit messages, or status documents.
- Preserve all existing data. Never hard-delete; approved removals go to this repository's
  `to_be_deleted/` boundary and only the owner permanently deletes there.
