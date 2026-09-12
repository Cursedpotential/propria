# Projects

This directory is the destination for Propria-owned applications and product
modules. The intended first-level layout is:

| Target | Current migration source | Purpose |
|---|---|---|
| `projects/consignatio/` | `Consignatio/` | Vault, filesystem recovery, Intake, deduplication and corpus preparation |
| `projects/probata/` | `Probata/probata/` | Evidence custody, proffer, analysis and platform operations |
| `projects/family-court-workbench/` | `FL-MCP/` | Local Tauri workbench for the family-court-toolkit plugin |

Intake remains part of Consignatio at
`projects/consignatio/Intake/`; it is not a sibling evidence platform. Project
imports must preserve commit history, local changes, licenses, and provenance.
Until the migration manifest says `imported`, edit and commit in the current
source repository—not in an empty target directory.
