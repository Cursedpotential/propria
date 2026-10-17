---
scope: E:/AI_Workspace/Projects/Propria/modules/Legal-desktop
status: current
verified_at: 2026-08-27
routing_reviewed_at: 2026-10-17
superseded_by: null
authority:
  - AGENTS.md
  - docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md
  - docs/DEPLOYMENT_PLAN.md
watches:
  - AGENTS.md
  - docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md
  - docs/DEPLOYMENT_PLAN.md
contains_secrets: false
---

# Advocatio Agent Memory

> _Original memory: Codex · GPT-5 · 2026-08-27. Routing reconciled by ChatGPT ·
> GPT-6 Astra · 2026-10-17 against the completed September 26 import._

This product is `modules/Legal-desktop/` in `Cursedpotential/propria`. Commit at
the Propria root with explicit module paths. Read `../../AGENTS.md` and this
module's `AGENTS.md`, including the October 4 shared Family Court library decision.
Old independent-repository routes and former product names are historical only.

- This repository starts after evidence has been reviewed and accepted. It does not ingest, hash,
  normalize, or establish evidence.
- Evidence Platform integration uses versioned `LegalSourcePackage` references and explicit
  investigation requests for missing proof.
- Complete one evidence-linked legal-work-product vertical slice before expanding the surface.
- Strategy, red-team analysis, todos, review rationale, and agent output remain private and
  non-court-safe unless the governed release path says otherwise.
- Persist mutations durably; never rely on process memory for domain state.
- When evidence-platform behavior is in scope, read `../Probata/probata/AGENT_MEMORY.md`;
  select that product lane explicitly through the Propria root router.
- Commit Advocatio files from the Propria root using an explicit module-path allowlist.

Add deeper `AGENT_MEMORY.md` or `.agent-memory/<filename>.md` only when durable local context exists.
Use `../Probata/probata/docs/agent-memory/README.md` as the shared format contract.

<!-- freshness
watches_hash: 424e82e
last_verified: 2026-08-27
watches:
  - AGENTS.md
  - docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md
  - docs/DEPLOYMENT_PLAN.md
-->
