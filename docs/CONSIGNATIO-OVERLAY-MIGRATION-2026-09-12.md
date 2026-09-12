# Consignatio working-overlay migration receipt

Date: 2026-09-12
Canonical monorepo: `E:/AI_Workspace/Projects/Propria`
Preserved source repository: `E:/AI_Workspace/Projects/Propria/Consignatio`
Canonical target: `E:/AI_Workspace/Projects/Propria/projects/consignatio`

## Outcome

Consignatio's committed history and reviewed working overlay are now represented
under the canonical Propria monorepo path. The original repository remains in
place as the recovery copy and was not modified by this migration. This is an
overlay import, not final cutover: no source directory was retired, no corpus
content was scanned or moved, no transfer authorization was copied, and no remote
job or service was changed.

## Committed-history baseline

- Source HEAD: `fa4f249a5c9d69bb7964ea851e71e8cecbab08a9`
- Source tree: `45dffef2308ff9a3ace79f0a6810585307d64bb2`
- Root import commit: `9a5ed62743b9ea125b17740dbae671c83083733c`
- Import mode: unsquashed history followed by a separately reviewed overlay
- Tree verification: the imported subtree exactly matched the source HEAD tree
  before the overlay was applied

## Overlay inventory and proof

The source working tree contained 85 changed or untracked paths when inspected:
16 tracked modifications and 69 untracked paths. The migration candidate manifest
contained 82 paths, 1,192,654 bytes, with SHA-256
`ad157f4735a0417304b1f2c0988069a894562f6484255df06afef8378932f73f`.
At the pre-commit recheck, the preserved source HEAD was still `fa4f249...`, but
its dirty-path count had fallen to 53. This migration did not modify or clean the
source. The changed count is recorded as concurrent source activity and is an
additional reason not to declare cutover or replace the source path yet.

Four routing/instruction files were deliberately withheld from mechanical copying
because their paths and authority must be adapted to the monorepo:

- `AGENTS.md`
- `AGENT_MEMORY.md`
- `Intake/AGENTS.md`
- `Intake/AGENT_MEMORY.md`

The canonical `Intake/AGENT_MEMORY.md` was reconciled manually so it retains the
CCC / Intake / Docstore isolation rule and points to the root
`SYSTEM-BOUNDARIES.md` at the correct relative path.

Seventy-eight source and documentation files were copied byte-for-byte: 1,179,373
bytes with ordered manifest SHA-256
`3404a0d0d48f2b08f589a96fa7d79f47b435305422e41d9462131ee30016eb6e`.
Every copied source/target pair was independently compared by SHA-256.

Three runtime or generated control files were explicitly excluded:

- `version`
- `repair-tool-kit/COMPACT-SUMMARY-2026-09-12.md`
- `casebible/r2-b2-migration-codex/operator-controls/ENABLE_CORPUS_TRANSFER.20260912.approved`

Repository ignore rules now cover generated compact summaries, the Consignatio
runtime version file, and migration approval markers. The approval marker was not
copied: importing source code must never authorize a corpus transfer.

## Credential review

The reviewed `.env.example` contains blank NVIDIA and Weaviate placeholders. A
test module contains the literal `test-only-secret`, which is fixture data rather
than an operational credential. No operational credential was identified in the
overlay review. Credentials remain external to Git.

## Canonical-path validation

All validation below ran against
`E:/AI_Workspace/Projects/Propria/projects/consignatio`, not the preserved source
checkout:

| Surface | Command boundary | Result |
|---|---|---:|
| Intake backend | existing E-drive Python environment, `pytest -q` | 107 passed |
| R2-to-B2 migration tool | `unittest discover -s tests -v`, `PYTHONPATH=src` | 12 passed |
| Intake desktop | `npm test`, existing dependency tree | 19 passed |
| Intake desktop | `npm run build`, 2 GiB Node heap ceiling | passed |

The frontend dependency directory was reused through an ignored local junction to
avoid duplicating a large dependency tree. It is not a tracked source artifact.
The build output is ignored.

## Remaining cutover gates

- Keep `Consignatio/` intact until the owner accepts canonical-path operation and
  separately coordinates any active VPS migration work.
- Do not turn the preserved source into a compatibility junction while a remote
  task may still reference its paths.
- Reconcile and import the remaining Consignatio instruction files only through
  the root routing contract; do not overwrite root authority mechanically.
- A fresh-clone build and live service checks remain Phase 5 work. Local tests and
  builds do not prove deployment, cloud credentials, or live runtime behavior.
- Any later source retirement must be a recoverable move to `to_be_deleted/` and
  remains owner-controlled. Nothing was retired in this transaction.
