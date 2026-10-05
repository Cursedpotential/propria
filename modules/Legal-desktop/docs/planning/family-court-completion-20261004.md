# Family Court completion execution — 2026-10-04

_Byline: Codex · GPT-6 · 2026-10-04_

_Updated by: Codex · GPT-6 · 2026-10-05 — deployment readback, complete-source manifest and private sync transport proofs._

STATUS: IN_PROGRESS. The owner approved catalog registration, shared revision and citation enforcement, remaining compiled catalogs, native workdesk integration and permanent placement of all final working material. Full private names, aliases, narratives and custom case fields remain intact. Case content stays outside Git. Every local source iteration remains unchanged.

## Permanent working-data home

Owner clarification at 21:13 EDT: archive ZIP preservation is separate from placement of actual working data. All final working iterations belong directly in their permanent Case Bible homes, with no recovery staging followed by another move.

A bounded live B2 listing confirmed the existing legal home b2://salem-data/consignatio/casevault/KnowledgeBase/legal/ contains INDEX.md. Final complete source units route beneath case-law/, benchbooks/ and reference-data/; toolkit units belong beneath reference-data/family-court-toolkit/. Associated files and internal links must remain together. All 443 selected working files have now been placed and independently verified by a separate tracked replay. The archive ledger remains a separate preservation record.

The owner requested direct linkage between Case Bible files and shared SurrealDB records, plus automatic discovery of directory additions and changes. Physical object identity requires provider, bucket, key, provider VersionId, exact source-byte SHA-256 and placement receipt. File hashes differ from complete shared-record versions.

Existing superindex scheduling checks a catalog watermark every 15 minutes but does not list B2. Batch-import folder discovery lacks version/hash identities and skips a previously completed URI. Existing exact-version acquisition and shared-library proposal/validation/publication are reusable. End-to-end legal-directory observation, additive catalog visibility and version-pinned library import are missing. A partial prefix listing must never replace a whole-bucket inventory generation. Owner approved two-way revisioned sync at 21:48 EDT: Case Bible additions/changes update the shared library; Family Court and Workdesk edits write the updated file to its permanent B2 home. Conflicts retain both versions for review. No recurring sync runtime has been deployed yet; implementation uses the existing worker and version-pinned acquisition contracts.

## Verified implementation and tests

Parent integration now includes full private context retention, exact-version personal edits with retained revisions, library proposals, trusted per-claim publication checks, visible retryable Temporal validation dispatch, the starter route, and read-only generic case_query. Startup factor seeds create only missing rows. Phone and Workdesk edits preserve unknown fields and nested content.

Isolated VPS native Surreal tests passed 39 publication/revision assertions with zero production writes. Store tests passed 31 with one Windows-only filesystem test skipped. The integrated console TypeScript check and build passed; 24 focused shared-catalog, phone and dispatch tests passed. Workdesk helpers previously passed 15 focused checks and TypeScript. The native FL-MCP build passed, as did six hosted-sidecar tests and seventeen UI/helper tests. Follow-ups narrow federated tool matching, use explicit null for removed fields and present editing errors inline. These are code proofs, not deployment proofs.

Named deadline presets now resolve from the exact current shared reference record and expose its record version/hash and citation trace. Missing configuration returns a pending result without falling back to compiled values. Legacy compiled configuration still needs retained-draft migration and legal validation.

Actual authenticated VPS browser baseline: phone responds and loads shared APIs; Workdesk lists 321 references at phone and desktop viewport sizes without overflow or page errors. The integrated console, starter and Workdesk have now been deployed. Console and starter containers are healthy; Workdesk API and office containers are healthy. The worker is running with no configured healthcheck. New editor clicks and Windows Tauri shell behavior remain unverified.

ContextForge's existing family-court virtual server now associates all 31 tools. Actual federated MCP tools/list includes library-propose, library-validate and library-publish; no duplicate server was created. This proves discovery, not successful editing or publication.

Shared store counts: 321 reference records and 193 source records. These are not 514 physical files. The reconciled source ledger has 119 file-backed paths, all resolved by the complete current unit and four retained original PDFs, plus 74 citation-only entries. The full current source ledger remains in the complete source unit; individual record exports are being linked separately without overwriting originals.

## Archive catalog milestone

The separate authoritative Case Bible PostgreSQL 18 catalog is on host loopback 5475. Main generation remains 2c2ae40f-bc6a-43c6-83a7-f3d60319e4d3.

Root commit 4fcdc402d3c1d82eb38683ab8894042d2458330c is pushed. Coolify worker deployment nvwalnmxitmk8sjpfgbvwlkr finished. Workflow toolkit-catalog-registration-20261004-v1, run 01a10970-3dfa-79eb-99f4-c4317f6a8356, and independent replay run 01a10972-564b-7a0c-9e74-db83905b0f85 completed. Independent readback: 15 distinct archive occurrences, 116293214 bytes, metadata SHA-256 52c9a8ba110ec2921dd82b03661a3313fe069a3f1564539e902c4a83245ef9b4. This is an additive recovery ledger; it is not proof of main-catalog working-file visibility or final-content placement.

Preservation receipt SHA-256: 053fd22d92cf0818b60a38aecbe7cfb8f027da81945ddbe204f2469a02c61df7. All 15 archive originals remain retained.

## Specific source-validation gaps and remaining work

The primary-source review found precise corrections needed in older referee guides: court approval and absence of timely objection control finality; signature alone is not written consent to immediate entry; retain live-evidence opportunities and discretionary limits; mail service completes on mailing; both supplemental-record exceptions apply. Exact-version proposals retain these findings. Official MCR current-release clearance remains provisional; MCL 552.507 acquisition was blocked. Sources: https://www.courts.michigan.gov/siteassets/rules-instructions-administrative-orders/michigan-court-rules/michigan-court-rules.pdf and https://www.legislature.mi.gov/Laws/MCL?objectName=mcl-552-507 .

The validator's pinned acquisition/extraction, scoped RECORD ACCESS authentication and protected currency-review CLI are integrated. Live validator and currency-review principals each read all 193 source records and cannot read unrelated meta rows; no case records changed during credential/schema provisioning. Separate writer permissions and root-only approval-key mounts remain mandatory. This identity readback does not establish deployed validation-workflow or publication success.

The canonical current toolkit source snapshot is now on the VPS without changing any local original: 439 ZIP_STORED members, 56,986,768 bytes including content and skills. Tracked inventory toolkit-canonical-current-inventory-20261004-v1, run 01a109cb-8d0f-7a49-8914-4af1c5ae1952, completed and wrote the private canonical-current-inventory-v1.json receipt. Final selection compares exact member content against the fifteen preserved archive iterations; neither dates nor hashes alone establish legal accuracy.

Permanent content placement is integrated in the existing worker as ToolkitContentPlacementWorkflow and toolkit_content_placement_activity. Focused placement and worker tests pass. Commit 741a654d adds pinned mounted-ZIP support; replacement worker deployment e7r5erfjzsmv5huo6hiqfkjt finished and its container is running with zero restarts. Workflow toolkit-permanent-placement-v5-20261005 completed in run 01a10c42-7c4e-7636-b7b7-efef1db0e824: 443 verified objects, 62,056,885 bytes, 443 exact provider versions and matching latest observations. Independent manifest-to-receipt comparison found exact set equality. Receipt SHA-256 is 244061ffa2d62568938de8b8ebd0792fa924a94c7aee3b5c4d868e3c47cf9b84, size 513,268 bytes. Initial run 01a10c3f-823e-703a-9b3c-50711a369f35 failed before reading the private manifest because worker access was denied; no B2 write occurred. Exact manifest group-read permission was corrected before retry. A separate tracked readback/replay completed as toolkit-permanent-placement-v5-readback-20261005, run 01a10c49-1fce-791e-8143-a3c4cd33c044. Working-file catalog visibility and shared-record bindings remain unproven.

The independently read placement manifest v5 contains all 439 current ZIP members unchanged, including 383 content files and 56 skill-tree members, plus four checksum-linked original PDFs. Manifest SHA-256 is 3f5a219804149b1e3175ffe048887723c2de233789a50860b7cfbf4d781f1ce4, size 95,087 bytes. All 443 destinations are unique and safe. The complete current tree remains together under reference-data/family-court-toolkit; supplemental originals retain their exact archive paths under reference-data/retained-primary-originals. Logical categories distinguish case law, actual benchbooks and reference data without breaking source-unit links. The private review records 120 changed historical observations, specific substantive corrections and remaining per-claim currency/citator gaps; preservation does not certify all historical candidates as validated finals.

Private sync HTTP transport is wired into the source server and remains undeployed. Six transport tests cover dedicated service authentication before body/database access, exact payload bytes, strict routes and version pins, metadata limits and opaque errors. Five configuration tests include rejection of client-controlled file bindings. Type checking and build pass. Go sync implementation b901e470 is integrated as 68c7274a: 91 test/subtest pass events on both Windows and isolated Linux, including intent recovery, competing retained versions, typed codecs and exact original reads. Starter private dispatch/original routes and mounted-secret configuration are committed as c346ccb9; route admission tests pass, but these routes are not yet deployed. Optional worker registration, transactional database integration, actual incoming-content hydration and automatic two-way synchronization remain unfinished. Merely retaining observation metadata does not establish shared-library content updates.

The toolkit now fails visibly when shared-store configuration is missing; it no longer implicitly creates a separate embedded database. Explicit isolated test overrides remain supported. Four pure configuration tests, type checking and build passed. Remote URLs no longer trigger filesystem-directory creation. Canonical application source/test paths are reconciled and LF-normalized without altering private content.

Owner decision treats R2 as retired for platform use. Worker/starter B2-only configuration is deployed. Commit e9d3d4a0 removes active R2 acquisition roots and secret mounts from Workbench/Tool Gateway source, preserves exact B2 versionId references and rejects ambiguous pins; 22 focused configuration checks pass. That source is pushed but the corresponding Workbench/gateway replacements are not yet verified. Separate exec-tier configuration still contains R2 settings. Residual objects and historical provenance remain untouched.

Remaining completion steps: register working-file visibility in the existing catalog; connect exact object identities to shared records; finish and wire transactional two-way sync, incoming-content hydration and original-file proxies; reconcile canonical plugin operational guidance; deploy the completed sync composition; prove shared edits, conflicts, validation and original-file opening through actual UI interactions. Docstore revision capture and search-index freshness remain separate proofs.

## Integration progress — 2026-10-05 morning

_Byline: Codex · GPT-6 · 2026-10-05._

Commits 02902483 and 40f40ddf integrate the isolated-native-tested sync backend with actual personal-edit and validated-library publication transactions. Full snapshots including private/unknown fields are captured before COMMIT; immutable snapshot encoding and bounded recovery use the existing console process. Six new integration boundary tests and TypeScript checking/build passed. Incoming raw hydration transport checks payload/source identities and an 8 MiB byte budget; backend adoption and Go Activity integration remain in progress. Actual parent transaction SQL still needs isolated native readback.

Commit 42b26980 adds the working-file catalog registration/readback Activities, guarded fixed-source occurrence writes and private version/provenance ledger. Its isolated PostgreSQL proof passed without changing whole-bucket generation or unrelated occurrence rows. Production catalog schema/role provisioning, worker registration and actual 443-file registration are not yet verified.

Commits a6195cb0 and 4e5b419c integrate bounded original-file controls into Workdesk and native Family Court views, including read-only source aliases. Commit 90bd969f adds a pinned 443-file binding importer restricted to all 321 existing references and 193 existing source records. Distinct structured record exports preserve canonical PDF/Markdown originals. Live mapping seed and actual original opening remain unverified. Commit 71278c95 permits closed retryable sync runs to reconcile their durable intent, with focused starter tests proving retry policy and in-flight joining.

Canonical private plugin source/test reconciliation is pushed as c84b8ad; the existing hook refreshed both Claude and Codex plugin caches. Private toolkit content was unchanged. Later backend, alias and edit-integration source changes still require reconciliation before the final cross-client proof.
