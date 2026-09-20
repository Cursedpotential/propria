# Propria root source reconciliation — 2026-09-20

This receipt records source preservation and Git routing after the owner's partial
structure move and instruction to merge, push, and leave the repositories clean.
It does not change deployment, running services, or installed plugin locations.

## Current ownership

Propria tracks shared governance, `modules/FL-MCP/`, and the moved `scripts/` and
`plugins/` source. Probata, Consignatio, Legal-desktop, Vestigia, and their retained
nested repositories keep their own Git histories. The current router and migration
manifest use the actual `modules/` paths. Earlier manifest status values and paths
are preserved as dated historical snapshots rather than presented as current proof.

The Docstore registry keeps its existing `docs/` junction roots and stable canonical
prefixes. Registry path configuration is not ingestion or deployment proof.

## Import provenance and required follow-up

The shared source was already moved into the root before this reconciliation.
Its originating repository is `modules/Probata/probata/`. Probata's integrated
source retains deployment-owned copies; those were not removed or overwritten.
Comparison against Probata's integrated HEAD at this check found:

| Root source comparison | Paths |
|---|---:|
| Equal after LF normalization | 212 |
| Different from the corresponding Probata HEAD file | 41 |
| No corresponding file in Probata HEAD | 9 |
| Total root scripts/plugin source paths | 262 |

This root snapshot preserves the moved edits; it is not an assertion that all root
copies supersede the integrated Probata implementations. The root Docstore source
is not yet an independently deployable replacement. Before any runtime cutover,
reconcile the 41 differing paths, classify the 9 root-only paths, resolve build
context dependencies, and verify plugin/runtime behavior. Keep module deployment
copies until that separate migration passes. Do not copy either tree wholesale
onto the other.

Detailed path/hash comparisons, the exact staging allowlist, index backup, and
validation outputs remain locally under `.reconciliation/root-clean-20260920/`.

## Preservation and exclusions

- All 106 `FL-MCP/` to `modules/FL-MCP/` renames have identical Git blob IDs.
- `docs/.docstore/` credential backups and database state were left intact and
  excluded as an entire runtime directory; their contents were not read.
- Plugin `.state/` flags remain local; the `.gitkeep` placeholder is source.
- Five one-off scratch/error-output or memory-inspection helpers were moved to
  `to_be_deleted/root-clean-20260920/scripts/` with SHA-256 verification. One
  scratch output triggered a credential-pattern check; no value is recorded here.
- The already-missing root `RESULT-PRESENTATION-CONTRACT.md` had no same-name
  replacement found in the inspected source directories. Its original HEAD bytes
  are preserved at `to_be_deleted/root-clean-20260920/RESULT-PRESENTATION-CONTRACT.md`
  before recording the existing removal.
- No file was permanently deleted. Private/runtime artifacts remain on disk.

## Verification and limits

- Family Court Workbench: 14 sidecar tests and 13 UI tests passed.
- Untracked source syntax: Python AST, JSON parsing, and 37 shell syntax checks
  passed; these checks do not execute migration or infrastructure scripts.
- Root Docstore control snapshot: 284 tests passed, 1 skipped, 8 failed with an
  explicit E-drive test directory. Six failures depend on the service-port registry
  and worker Dockerfile retained in Probata; two expose project-registry/handoff
  fixture-contract mismatches. These failures are recorded, not hidden or treated
  as successful runtime validation.
- The current routing manifest contains 12 existing source paths.
- No deployment, production database change, or host lifecycle action occurred.

Pending: reconcile shared-source differences and the eight root Docstore test
failures before promoting this preserved source snapshot to a runtime/deployment
replacement. This is separate from the completed Git relocation and publication.
