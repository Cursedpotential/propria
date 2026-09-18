# Projects

This directory is the destination for Propria-owned applications and product
modules. The intended first-level layout is:

| Target | Current migration source | Purpose |
|---|---|---|
| ~~`projects/consignatio/`~~ | `Consignatio/` stays in place | ~~Vault, filesystem recovery, Intake, deduplication and corpus preparation~~ — not moved (owner 2026-09-18) |
| `projects/probata/` | `Probata/probata/` | Evidence custody, proffer, analysis and platform operations |
| `projects/family-court-workbench/` | `FL-MCP/` | Local Tauri workbench for the family-court-toolkit plugin |

Intake remains part of Consignatio at
`projects/consignatio/Intake/`; it is not a sibling evidence platform. Project
imports must preserve commit history, local changes, licenses, and provenance.
Until the migration manifest says `imported`, edit and commit in the current
source repository—not in an empty target directory.

> _2026-09-18 (Claude Code · Opus 5): the 2026-09-12 overlay copy is quarantined in `Consignatio/to_be_deleted/2026-09-17-projects-consignatio-overlay-from-2026-09-12/`. A same-day move of Consignatio into this folder was reversed by owner order: Propria is the project._
