
> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._

---
scope: E:/AI_Workspace/Projects/traceIQ/traceiq-rebuild
status: current
verified_at: 2026-08-29
superseded_by: null
authority:
  - AGENTS.md
  - docs/adr/README.md
  - db/migrations/
watches:
  - AGENTS.md
  - docs/adr/README.md
  - "db/migrations/**"
contains_secrets: false
---

# TraceIQ Rebuild Memory Router

> _Byline: Codex · GPT-5 · 2026-08-27; parent-boundary refresh 2026-08-29._

- Architecture decisions: `docs/adr/README.md`, then the linked ADR.
- Schema and transformation contracts: `docs/SCHEMA.md`, `db/migrations/`,
  `db/transformations/`, and `docs/transformations/`.
- Current bounded validation: `docs/VALIDATION_REPORT.md`,
  `docs/VALIDATION_REPORT_data.md`, and the relevant `ops/*REPORT.md`.
- UI intent and implementation context: `ui/AGENTS.md`, `ui/BUILD_BRIEF.md`,
  `ui/BUILD_BRIEF_PHASE2.md`, and `ui/README.md`.
- Reporting context: `reports/README.md` and its linked pages.
- Parent checkout reconciliation is resolved in `../REPOSITORY_RECONCILIATION.md`; the outer
  repository intentionally ignores this independent child and remains outside this commit scope.

Local `.claude/`, `.remember/`, `.memsearch/`, and `.serena/` state may be stale or branch-specific.
Use it only to locate current source, then verify every material claim against tracked files and
live Git state. Never store secrets or evidence content here.

<!-- freshness
watches_hash: d2c62b5
last_verified: 2026-08-29
watches:
  - AGENTS.md
  - docs/adr/README.md
  - db/migrations/**
-->
