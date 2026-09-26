---
scope: workbench
status: current
verified_at: 2026-08-27
superseded_by: null
authority:
  - AGENTS.md
  - docs/design/0061-unified-operator-surface/spec.md
  - docs/reviews/2026-08-27-workbench-auth-rotation.md
  - workbench/design-mockups/unified-operator-surface/AGENTS.md
watches:
  - workbench/**
  - docs/design/0061-unified-operator-surface/spec.md
contains_secrets: false
---

# Operator Surface Memory

> _Byline: Codex · GPT-5 · 2026-08-27._

## The six steps — measure every surface against these first

> _Owner, 2026-09-22 19:54. Full statement and record: `docs/PURPOSE.md`._

1. Open a file or folder, through an index. 2. Verify whether it is relevant. 3. Make sure it has a hash. 4. Pick a parser or extractor and get it into context and the analysis platforms. 5. Preview the result to make sure the machine did it right. 6. Fill in gaps and missing context.

Before adding anything to a surface, name which step it serves; if none, it does not go on the surface. Sources owns 1–4. Review is 5–6 plus the decision, full screen, nothing else. Run status, receipts, stores, lineage and tool catalogs are not primary content on either.

## Owner-approved product direction

- The unified surface combines an everyday **Evidence Operations Desk** with a more advanced
  **Modular Service Cockpit**, using one coherent visual language and light/dark themes.
- Finish a functional vertical slice before adding another destination. Do not expand navigation
  with stubs, disconnected dashboards, or backend claims that are not wired.
- The surface is single-user and single-case. Prefer plain-language labels, visible case/court
  context, compact operational density, and no unexplained two-letter abbreviations.
- Timesketch should be available as a timeline view across the operator experience. Temporal, n8n,
  Semantica, database, graph, and storage interfaces remain tools over canonical authority.
- Traces, logs, and progress should stream to the surface in real time, including LLM operations.

## Critical correction

The legacy Operator Console and its LanceDB staging design are not the approved unified product.
Do not revive, redeploy, or present that surface as progress toward the approved UI.

## Child memory

For the approved implementation, read
`workbench/design-mockups/unified-operator-surface/AGENT_MEMORY.md`.

<!-- freshness
watches_hash: 8f36944
last_verified: 2026-08-27
watches:
  - workbench/**/*.py
  - workbench/**/*.ts
  - workbench/**/*.tsx
  - docs/design/0061-unified-operator-surface/spec.md
-->
