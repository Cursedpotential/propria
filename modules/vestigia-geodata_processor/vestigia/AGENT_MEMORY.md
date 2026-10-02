---
scope: E:/AI_Workspace/Projects/Propria/modules/vestigia-geodata_processor/vestigia
status: current
verified_at: 2026-10-02
superseded_by: null
authority:
  - AGENTS.md
  - docs/plans/2026-10-02-continuation-plan.md
  - docs/adr/README.md
  - db/migrations/
watches:
  - AGENTS.md
  - docs/adr/README.md
  - "db/migrations/**"
contains_secrets: false
---

# Vestigia Application Memory Router

> _Byline: Claude Code · Opus 5.5 · 2026-10-02._

- Next work and open decisions: `docs/plans/2026-10-02-continuation-plan.md`.
- Architecture: `docs/adr/README.md`, then the linked ADR.
- Schema and transformations: `docs/SCHEMA.md`, `db/migrations/`, `db/transformations/`,
  `docs/transformations/`.
- Validation: `docs/VALIDATION_REPORT.md`, `docs/VALIDATION_REPORT_data.md`, `ops/*REPORT.md`
  (all from 2026-07-24/26; the live database was re-read on 2026-10-02, see the plan).
- UI: `ui/README.md`, `ui/BUILD_BRIEF.md`, `ui/BUILD_BRIEF_PHASE2.md`. Reports: `reports/README.md`.
- Local `.claude/`, `.remember/`, `.memsearch/` and `.serena/` state may be stale; verify every
  material claim against tracked files and live git state. Never store secrets or evidence here.
