# PROPRIA — Claude remediation and deployment handoff
**Date:** 2026-10-16  
**Role:** Claude = implementation/deployment agent **only after evidence and owner approvals**  
**Parent:** [AI import integrity investigation](./ai-import-integrity-investigation.md)  
**Location:** docs/probata/reviews/ai-import-remediation-handoff.md  
**Status:** INVESTIGATION HANDOFF — unresolved forensic root cause, no authorization to publish AI content, modify original sources or backfill modes.

## 1. Mission / critical constraint
Complete the smallest coherent repairs and validation for historical AI import traceability, source integrity, operation modes, browser/Temporal acceptance and accessible list persistence. **Do not redesign Propria.** Never start by restoring AI conversation bodies into PostgreSQL. The controlling owner instruction of 2026-10-15 is **AI chat content to Weaviate; extracted events, entities, timelines and artifacts to PostgreSQL**. Original source bytes, version identity, source-side provenance and explicit approval remain immutable. Old PG raw/normalized AI receipt identities exist as historical audit records but do not create a current storage mandate.

**Stop before case-data writes or recovery until a specific owner approval covers exactly that source/version/action.** This handoff is sufficient for source-level triage; it is **not** a forensic certificate of full live content reconciliation.

## 2. Sources, pins and prior outcomes
Read in order:
1. [Investigation and issue register](./ai-import-integrity-investigation.md), including confidence labels.
2. [2026-10-15 operator repair handoff](https://github.com/Cursedpotential/propria/blob/main/modules/Probata/probata/docs/handoffs/HANDOFF-2026-10-15-operator-workflow-repair.md), in particular full *Exact historical AI index readback* and *Remaining acceptance boundaries*.
3. Governing Docstore readback **document:mu6dmz1rno8c828zod9e** (owner storage correction).
4. `modules/Probata/probata/docs/PROJECT_CANON.md`, `docs/PURPOSE.md`, `docs/reference/WORKFLOW-REGISTRY.md`, architecture audit and current ADR/decision index. Treat old PG AI-content descriptions as superseded.
5. Historic browser evidence `Probata_Live_Audit_2026-10-15.html` and `Probata_Repair_Agent_Handoff.md` (library/owner copies).

**Code repository:** `Cursedpotential/propria`. Latest October 15 deployed code lives on `codex/operator-repair-integration-20261015`, **not necessarily default main**. As recorded: Workbench `28b42d1ac6c9a0d588e7d2682262b497d839b383`, starter `f215e47de933c512acb9c079184e6cea827c8d9a`, Go worker `d7ada57f86092193630a3e6e930c30a277f937bc`. Draft PR #9 was unmerged. **Reverify current SHA/deployment** and preserve unrelated in-flight changes; don't overwrite the deployed branch.

**Private investigation artifacts to obtain through approved local access** (not available to ChatGPT, so content not freshly inspected):
- `repair-contracts-20261015/.reconciliation/operator-repair-contracts/exact-ai-index-readback-20261015.json`, SHA256 `b55e855f4ff8582713d838c3a889020dc8da147ef291d1dfcd1f71495e2f0f58`.
- `.reconciliation/operator-repair-20261015/release-test-logs.tar`, `combined-store-proof-final.txt`, `postlive-test-logs.tar`, `http-after-repair.jsonl`, browser proof archives.
- Missing original/object version recovery manifest; prior exact-source run registrations and temporal execution IDs.

## 3. Sequential TODO — strict gate order
- [ ] **T00 Baseline freeze.** Read current git commit, cleanliness, deployed Coolify references, DB schema capabilities, workflow registry and owner-approved storage canon. Compare to October 15 pins. Keep an append-only evidence ledger with timestamp, environment, actor, exact source/version, command/predicate, query coverage and non-sensitive output digest.
- [ ] **T01 Verify old query evidence.** Read and SHA256-check private query receipt. Confirm corrected array membership syntax, precise collection and named vector, identity-field tokenization, query results and credential scope. If unavailable, mark missing rather than re-create claims.
- [ ] **T02 Resolve original and identities (read-only).** Locate export byte/version/manifest and archive members in governed source inventory; enumerate SHA256/hash aliases, original request, source version, source occurrence, actual native workflow IDs and native index reference. Do not stream entire export to log files or fabricate a mapping.
- [ ] **T03 Read-only end-to-end correlation.** Trace historical PG operation metadata/30 Activity + 30 receipt identities into Temporal history; inspect native-context/index acknowledgements and exact Weaviate objects via configured allowlisted clients. Correlate source/version, object properties and searchable content. Check both already-known collections and **only additional collections proven relevant by actual config or receipts**. Never treat two zero-result queries as all-store absence.
- [ ] **G1 Owner decision gate.** Present one of: (A) content found and linked, (B) source retained but native index unproven/missing, (C) original/source availability unverified, (D) original independently verified unavailable after a bounded inventory, or (E) conflict/duplicate. Supply evidence, exact impacted count if verified, estimated blast radius with uncertainty. If B/C/D/E, propose one scoped remedy and stop pending approval. **No automatic import/rebuild.**
- [ ] **T04 Malformed source recovery assessment (read-only).** Create per-sample record with source version, provider ID, filename/type claim, size, SHA/ETag, first bytes, previous provider versions, independent archive backups and hash-matched good candidates. Do not replace corrupted versions. Scope source cohort only.
- [ ] **G2 Original recovery gate.** Owner selects which new version/copy, if any, to create. Approval must specify preserved old version, evidence chain, provenance, allowed destination, rollback and expected observable result. No silent “fix MIME”.
- [ ] **T05 Legacy mode receipts.** List each blocked operation, original missing/valid/conflicting mode and exact source/version/case. Use existing `PreviewAdmissionRepairWorkflow` **only if explicitly approved** for each eligible unknown-mode binding; reject conflicting, malformed, mismatched, already-mode-bound and DEV-as-case-fingerprint scenarios. Never bulk infer modes.
- [ ] **T06 Implement safe UI/list persistence.** Separate scoped grid/list preference and their independent scroll positions in existing `source-navigation.ts` session-navigation contract; keep source/version/ETag exact, validation strict, storage failure harmless. This does not need production case writes.
- [ ] **T07 Narrow parser/test improvements, if justified.** Preserve bounded Go byte signatures and manual AI classification for unknown JSON/ZIP/Markdown. Add support **only** for approved real, versioned fixtures. Leave unknowns unclassified with an explicit safe operator path.
- [ ] **T08 Isolated test journey.** In a separately configured synthetic environment (DEV alone is not isolation), exercise browser action → authenticated BFF → Go start → Temporal workflow and Activities → authorized simulated/isolated data mutation → durable receipt/new generation → read/reopen. Include cancel, retry, failure, invalid request, conflicting mode, permissions, browser reload, idempotence, timeouts and rollback. No real case mutation.
- [ ] **T09 Real screen-reader acceptance.** On actual supported assistive tech/browser, test grid and list; record reader and browser version, focus order, row/file names, keyboard-only navigation, selection state, aria announcements, error/unknown state, modal escape, search pagination and viewport zoom. Four historically skipped automated browser tests must either run or remain explicit blockers.
- [ ] **T10 Independent review + deploy gating.** Code-review each changed file; run focused, contract, browser and integration tests. Record provenance, test/environment SHA and control cohort. If deployment approved, preserve working stacks and publish only scoped code with rollback. Recheck endpoint and exact case/source result without any background reprocess.
- [ ] **T11 Verification record.** Update original audit status per issue; attach evidence/hash manifests and exact new workflow IDs. All unsuccessful coverage remains marked incomplete. No global rule creation without explicit approval.

## 4. Exact historical import controls
| Attribute | Value / interpretation |
| --- | --- |
| Historical request | `ai-content-mattsalem85-20261006-run01`; AI collection request tokenization uses words, so request lookup is not a definitive exact-ID predicate |
| Source version | `01a1141c-b03a-7a07-ad05-9dd5632829e8` |
| Historical parser declaration | `chatgpt_official_json` |
| Raw persistence receipt | `6f3bf342-f1ce-4cfc-8197-09dec824e543` |
| Normalized persistence receipt | `4b60f043-a838-4af5-a52c-e90414001d3f` |
| Raw generation | `01a1141f-0490-7289-a35d-c8e22c8bed63` |
| Normalized generation | `8bb7c26f-0209-409b-b91c-60530ee9ae6b` |
| Indexed collection scope previously checked | `AiChatEvents20260918`, vector `text_nim`; `ProfferChunks20261002`, including array membership |
| Earlier exact scoped results | 0 source-version matches in either; 0 matching native workflow mapping in checked PG metadata/receipts; **no global absence claim** |
| PG historical metadata | 30 distinct Activity identities, 30 receipts, filesystem-stat metadata, and historical operation registration; earlier exact PG generation rows absent |
| Reported currently released UI | `ai_chat / weaviate / not_queried`; no invented PG generation or human source-review route |

**Read-only query discipline:** prefer existing approved typed repository/service methods and parameterized SQL/GraphQL. Inventory collections/schema/filters first. Log exact query structure and filters, results with counts and safe object IDs (redact chat bodies and secrets), request trace and pagination/cursors. Distinguish Query Failed, Not Queried, Partial Coverage, No Matching Identity, Wrong Tokenizer, Wrong Collection, No Mapping, and Proven Negative. Do not issue user-supplied freeform write-capable SurrealQL or object-management calls. No direct reload/reindex/update/delete.

## 5. Files and database objects to work in
Use `modules/Probata/probata/` as the working root; symbol names and source ownership take precedence over changing line numbers.

**AI parser/ingest/read:** `modules/engine/sourceformat/detect.go` (`Detect`, ~41), `modules/engine/proffer/workflow.go` (versioned routing/Activities), `modules/engine/postgres/elt_chatgpt.go`, `modules/engine/postgres/elt_templates/chatgpt_json_array_v2.sql` (historic ELT only), `server/analysis/format_router.py`, `server/analysis/chat_parse.py`, `server/tools/parsers/ai_chat/*`, `server/core/retrieval_readers.py`, `modules/workbench/api/app/service/imported_search.py`, `modules/workbench/web/src/components/sources/sources-screen.tsx` (AI provider selection and native-start path). The existing Go detector has known ChatGPT and Claude JSON signatures, AI Markdown signatures, plus general fallbacks; avoid a universal detection claim.

**Receipt/mode/operations:** `modules/engine/postgres/preview_admission_repair.go` (`RepairBindingAdmission`), `modules/engine/proffer/preview_admission_repair.go` (`PreviewAdmissionRepairWorkflow`), `modules/workbench/api/app/service/preview_mode_recovery.py` (`mode_for_operation`, `require`), `modules/engine/postgres/standalone_extraction_admission.go`, `modules/workbench/api/app/runtime/proffer.py`, `modules/workbench/api/app/runtime/analysis.py` (explicit unavailable-vs-zero). Existing objects include `context.source`, `context.source_version`, `context.proffer_preview_binding`, `context.proffer_preview_snapshot`, `context.proffer_preview_event`, `context.hash_receipt`; discover actual other table names from live schema rather than inventing them. **No new mapping DB**.

**UI accessibility:** `modules/workbench/web/src/components/sources/source-rows-grid.tsx` toggles a semantic `<ul>` with component-local `semantic`, and reports only grid scroll. `modules/workbench/web/src/components/sources/source-navigation.ts` persists `scrollRow` in `sources-navigation-v1:<routeKey>`; `sources-screen.tsx` constructs/restores this object. **Smallest fix:** extend this current state shape, not a new global preference store. Prefer separate `gridScrollRow` and `listScrollTop`, persisted `viewMode` and versioned backward-compatible decoder, restore after DOM mount/data load. Guard against storing bodies, token values, unstable scroll after data filtering, and cross-root reuse.

**Recovery references (procedural precedent, not same cohort):**
- `modules/Consignatio/docs/receipts/corruption-hunt/FINDINGS-2026-09-13.md`;
- `modules/Consignatio/docs/receipts/source-recovery-2026-09-20/PRESERVATION-AND-CONTAINMENT.md`;
- source provider delivery and `modules/workbench/api/app/service/source_inspection.py`;
- `modules/workbench/web/src/components/sources/source-media-preview.tsx`.
The September historical mass-zero corruption count must **not** be treated as the extent of the October five-sample media defect.

## 6. Prioritized issue owner/action register
| Issue | Priority / confidence | Owner-bound action | Exit proof |
| --- | --- | --- | --- |
| I-01 Historical index mismatch | P1 / CONFIRMED bounded; cause UNVERIFIED | Context/Weaviate + ingest investigator, then owner gate | Native exact source/version + read/search proof, or appropriately delimited unknown |
| I-02 Native workflow mapping | P1 / UNVERIFIED global | Temporal/ledger investigator | Supported native workflow/alias join OR documented missing link and stop |
| I-03 AI PG restoration | Excluded / intentional policy | Do not implement | PG remains derived outputs only, no AI-body writes |
| I-04 Malformed objects | P1 / CONFIRMED five bytes; recoverability UNVERIFIED | Object provenance investigator; owner gate | Original preserved, independently byte-verified replacement if authorized |
| I-05 Legacy mode block | P1 operator / intentional safeguard | Receipt/mode team with per-operation approval | Repaired audited receipt or unchanged 409, never mode inferred |
| I-06 Full Temporal/browser mutation | P1 acceptance / NOT DONE | Integration/QA | Approved synthetic E2E receipts, permissions, replay/rollback |
| I-07 Screen reader | P2 acceptance / NOT DONE | Accessibility QA | Real AT-browser log and pass/fail matrix |
| I-08 List mode/scroll | P2 / CONFIRMED | Workbench front end | Both modes' nonzero positions survive navigation/remount |
| I-09 Unknown-format identification | P2 / bounded support | Parser owner only when sample contract warrants | Known-positive, known-negative, unknown manual route tests |

## 7. Executable implementation boundaries

### AI identification and trace
**First** run read-only discovery with the actual configured readers. Require the object fingerprint and source-version identity from object storage + PG ledger, then compare field-schema/tokenizer/index properties and workflow receipts. Collect a join table `{original_sha256, source_version_id, historical_operation, native_workflow_id?, native_run_id?, native_object_id?, collection?, indexing_receipt?, readable?, search_hit?}` with nulls when unknown. **No inferred equality between IDs.**

If an actual *missing native ingestion* is proven and the owner approves a specific source:
- propose **version-pinned, idempotent** native admission using the existing Go/Temporal route and existing receipt store;
- keep originals untouched and duplicate-check by content fingerprint and native source/version identity;
- preserve old historical PG references for audit only;
- publish AI content to existing Weaviate destination; PG stores only allowed derived outputs/control metadata;
- record workflow/attempt, exact returned object IDs and native searchable readback;
- do not implement until per-source approval and a documented rollback are present.

### Malformed originals
Use exact ID/version and B2/R2 provider GET+HEAD+range, source package copy and backup hashes. Compare independently known media magic and full digest, not filename, ETag alone, or decoded browser result. Keep source status `malformed_as_retrieved` distinct from `object_unavailable`, `version_unverified` and `supported_codec_missing`. Never repair a case object in place; future good copy is a **new attributed recovered occurrence/version** only when approved. No blanket reclassification or broad quarantine based on five samples.

### Mode controls
Only existing audited admission-repair contract may append `context.proffer_preview_event` under owner LIVE identity/actor/reason/idempotency and exact source/version/matter binding. Its Go code refuses source mismatch, original conflicting or malformed mode, nonempty existing authority and identity mismatch. Test before after, repeat idempotent, conflicting retry and unrelated case; negative path must show zero writes. No repair solely because an old UI route once showed a completed run.

### Browser + Temporal
Use synthetic cases with both relevant/irrelevant messages, unknown speaker, acquired-third-party/source-of-truth distinctions, an unrelated control, safe AI content fixtures, malformed/recoverable media and legacy missing-mode operation. Test every state transition: preflight → authorized launch → Activity receipt → normalized/derived candidate (only permitted test store) → persisted generation/citations → preview → correction scope → save → new generation/reopen → cancellation/timeout/retry. Assert no illegal cross-case modifications and no hidden retry resurrects terminal operations. Separate query-only workloads from mutation tests.

### Accessible list
Persist view choice and scroll independently from canvas grid, preserving all existing source/ETag/pagination identity state. Validate list-mode `scrollTop>0` restoration after Activity→Back and remount; rerender after pagination; selection and keyboard focus remain coherent. Prefer scroll position **within the current route scope** over cross-session global preference. No blanket URL/lane mutations. Test storage-disabled fallback. Real screen-reader acceptance is independent from passing React tests.

## 8. Preservation / rollback requirements
1. Before any implementation: capture deployment image/commit/worker references and branch state; no replacing main blindly.
2. Before approved source/index mutation: snapshot/export only relevant control rows (securely), immutable original version manifest + hashes, old and proposed native index object identities, and Temporal workflow state. Verify that recovery artifact can actually be read; *a backup command is not a backup proof*.
3. Avoid rollback-by-rewriting case history. For erroneous new projections, use existing versioned reconciliation/retirement mechanisms with an audit receipt, subject to owner approval; preserve old original and Activity history.
4. Mode admission repair is append-only; do not undo by deleting receipt or falsifying original. Conflicts require owner-directed compensating policy and recorded action.
5. UI code rollback: restore prior verified Workbench pin, preserve working deployed starter/worker, and show versioning/health checks. Do not restart unrelated services.
6. Independent temporary test DB must be demonstrated empty after rollback/teardown; no dev/live false isolation.
7. Credential incident in October 15 source handoff requires scoped security follow-up/rotation by authorized owner. **Do not collect/print secrets in report, process args, logs or shell output.**

## 9. Verification / regression matrix
| Test ID | Fixture/scope | Procedure/assertion | Evidence required |
| --- | --- | --- | --- |
| V01 | Exact historical AI source | Re-run only approved read-only source version predicate in configured `AiChatEvents20260918`; inspect pagination, tokenizer and named vector | Query/response summary, count, schema, permission scope, timestamp |
| V02 | Proffer collection | Exact scalar and array member identity predicates on `ProfferChunks20261002` | Corrected query receipt, coverage, zero/partial |
| V03 | Native aliases/workflow | Trace source→operation→Temporal→native index receipt with explicit ID joins | Mapping ledger and signed-off no-guess result |
| V04 | Search and content | Search for a representative approved native content span while preserving object source ID | Reproducible hit + safe citation/locator, no body in report |
| V05 | Negative/unknown | Missing source/version/collection | `not_queried`/`unavailable` not zero; no invented PG content |
| V06 | Malformed real sample | Provider full/range/version digest; other source versions read-only | Original unchanged with SHA and parse-header mismatch; candidate lineage |
| V07 | Known-decodable synthetic media | PNG/GIF/MP4/audio full+range, Retry | Decode & play, correct pinned version, no illegal rewrite |
| V08 | Mode denial | Legacy unknown and contradictory mode | Explicit 409/refusal, no Activity or record mutation |
| V09 | Approved admission repair (if approved) | Exact single source, repeated same click/other actor/version | Immutable receipt, idempotence, refused conflicts |
| V10 | Temporal E2E | Isolated synthetic authorized start/save/cancel/retry | Real IDs, stage receipts, partial failures and rollback, source anchoring |
| V11 | AI-to-PG fence | Synthetic AI native content and allowed extracted fact | No AI body rows; only approved derived PG rows; owner-policy proof |
| V12 | Parser detection | ChatGPT, Claude, AI Markdown, human JSON, generic JSON and ZIP | No false promotion, unknown manual selector, bounded memory |
| V13 | Grid/list persistence | Nonzero grid and list scroll, switch mode, Back/remount | Retained exact route/selection/version and independent offsets |
| V14 | Accessibility | NVDA/JAWS/VoiceOver and keyboard across grid/list/results/modal/error | Real AT test notes and per-step pass/fail |
| V15 | Baseline regression | Existing affected APIs and Workbench builds | Focused Go/Python/TS and browser suite; skips listed explicitly |
| V16 | Production code release (approval required) | Prior/next deployed pins and health | Deterministic rollback and postrelease **read-only** proof |

**Pass rule:** Each test gets PASS/FAIL/BLOCKED/NOT RUN and an environment/commit/fixture reference. `not queried` is **not** `zero`, a successful storage receipt is **not** a user-visible index result, and a browser unit test is **not** a real screen-reader test.

## 10. Decisions pending owner approval
- May a new *read-only* targeted live inventory be run on the source, Weaviate schema/objects, PG control and Temporal history? Which access method owns the logs?
- If native content is absent but immutable source retained, may Claude propose a single idempotent native-import execution? **No current import permission.**
- Which malformed media versions, if any, should be reconstructed as separate versions from independently verified copies?
- For which exact legacy operations, if any, may an actor-bound operating-mode receipt be appended?
- May Claude provision a disposable isolated test database and fixture workflow (not the real case)? Exact disposal/rollback required.
- What browser/OS/screen-reader combinations are required for acceptance?
- Are there newer architecturally controlling owner decisions or ADRs than the October 15 correction?

## 11. Delivery package and stop condition
Claude should produce: (1) baseline and exact source mapping, (2) evidenced root cause per issue, (3) bounded implementation diffs (code only unless expressly approved), (4) recorded negative safety tests, (5) test matrix with real execution receipts, (6) preserve/rollback manifest, (7) owner-decision record for all write gates and (8) deployment/readback receipts only where separately authorized.

**Mandatory STOP:** if a source is missing, identity join is ambiguous, provider content differs, a receipt conflicts, the target collection cannot be verified, a case write would be needed without consent, or rollback proof is unavailable. Return the gap and required evidence; do not guess, blindly re-run ingestion, bypass receipt controls, or quietly promote new system-wide rules.

**This investigation agent performed no remediation or deployment.** The docs are placed on an independent branch so deployment remains Claude’s explicit separate responsibility.
