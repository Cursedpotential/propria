---
scope: deploy
status: current
verified_at: 2026-10-17
verification_scope: Source routing at 68f369e; no deployment or service-health verification
superseded_by: null
authority:
  - AGENTS.md
  - docs/REPO_STRUCTURE.md
  - docs/PROJECT_CANON.md
watches:
  - deploy/**
  - deploy/compose.yaml
contains_secrets: false
---

# Deployment Memory

> _Byline: Codex · GPT-5 · 2026-08-27._
> _Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17; Propria root's
> 2026-09-26 import contract and source tree read at `68f369e`._

Paths below are relative to `modules/Probata/probata/`. This module shares the
Propria root Git boundary. The former root compose route is now
`deploy/compose.yaml`; the pre-import routing remains in Git history.

- Source is edited and validated in this checkout, committed, and pushed; Coolify builds and
  deploys on the VPS. Commit at the Propria root using explicit module-prefixed
  paths. Do not create or present a duplicate local stack as production progress.
- One Coolify application uses one `deploy/<app>.yaml`; preserve correct watch paths so unrelated
  pushes do not bounce data services. Source/base-directory routing and explicit
  exceptions are in the root `AGENTS.md`; watch paths use repository-root prefixes.
- Tailnet services bind the intended tailnet address, not assumed localhost.
- Do not manually start, stop, remove, or replace Coolify-owned containers behind Coolify.
- Never print or persist secrets in this hierarchy. Authentication configuration must remain
  rotatable without requiring application-code redeployment where the chosen identity layer allows.
- A successful build is not a live receipt. Verify health and the actual owner-facing interaction.
