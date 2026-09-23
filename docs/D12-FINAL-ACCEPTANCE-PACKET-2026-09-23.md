<!-- Byline: Codex, D12 independent operations/acceptance reviewer, 2026-09-23 EDT -->
# D12 current operations and release acceptance packet

**Decision: HOLD the Propria whole-system release.** This is an independent source, receipt, repository, and GitHub check; it is not a deployment, restore, native journey, or release authorization. Narrow live endpoint receipts exist for Workbench, hosted Intake, and Legal, but no common candidate/profile binds their deployed images, schemas, configuration, source versions, native behavior, and recovery. No destructive, production, credential, or case-data action was taken for this packet.

**Proof level:** `SPEC` for the reconciled acceptance contract, `SOURCE_STATIC` for inspected current configuration and Git state, and **inherited, narrow `DEPLOYED`** only where the dated lead receipt names an actual live endpoint/deployment. D12 did not independently reproduce those live probes. `NATIVE`, whole-profile `RESTORE`, and `RELEASE` are **not proven**.

## Source and readback basis

The primary canon `INDICIA_PROBATA_PLATFORM_CANON_v1.1.0.md` §§23–27 (physical lines 586–700) and its A01–A22 acceptance cases were read. The transferred A work register `work_items.json` and `acceptance.json` were parsed for W13/W18/W28/W29/W30/W31/W33/W34 and `AC-W-*`; comparison `INTEGRATED_PACKETS.json` CP14 was read. The existing D12 interim checkpoint/template, WS00 history receipt, root orchestration receipt, D08/D13 checkpoints, and current Git/GitHub observations were compared. Full A/B source-to-finding coverage remains the lead's traceability task; this packet retains the exact linked aliases for D12's eight work items and does not certify every source paragraph as inspected.

| Read source | SHA-256 |
|---|---|
| `agents/D12/D12_INTERIM_CHECKPOINT_2026-09-23.md` | `67831090C9C3FC6DA6D34798DFB52CE6FBFB781909AFCC82C705EF9BD1526B73` |
| `agents/D12/D12_EVIDENCE_RECEIPT_TEMPLATE.json` | `B17FD971B3699EC5C4C1C51333DAD34C588C2FB7CC5EB92912799BB4F9FF5B7D` |
| `transfer/.../inputs/primary/INDICIA_PROBATA_PLATFORM_CANON_v1.1.0.md` | `DCDA22307E5E4A6E8E9C8D133ED3C12C05D1A5D938538FB714A07E5AFBA95A22` |
| `comparison/.../A_handoff/registers/work_items.json` | `B58E3705294B949910D050AE967FFC4B364949B4AF4D57C278FEF1552A007E7C` |
| `comparison/.../A_handoff/registers/acceptance.json` | `8E08F3CACDAE7BED988C9F6AEC464229953784F552A7A61D351E5FE2A66B3FD4` |
| root `docs/ORCHESTRATION-RECEIPT-2026-09-23.md` | `874DBA20378237160AE3B496872E110EC1856BF6C68D782DFEA6A3E3D6A6EA98` |

## Current candidate and delivery identity

Read-only observations on 2026-09-23: Propria root `main=origin/main=6faa9f088bc65226cf45bb3820ffff15d9c90d89`; four protected untracked paths include Claude's portal audit, two preview files, and `session/`. Probata `main=21952e429b9fa949f7f51b2b02cf8353e7d89ec9` is **two local commits ahead** of `origin/main=382acf6a8f1f919b6f3abc02de40f946629d8461`, with a dirty SBV nested repository and untracked R2 blocker receipt. These two local commits concern a paused JEV evaluation plan, so they are not silently included in any candidate. Consignatio root `main=0d002d46c3440a37655ecef17acab14688fbab0b` has one tracked modification and six untracked paths; the deployed hosted-engine branch is separately identified below. Legal-desktop `master=06b69433163537bb3d66925149694ca1a12ac689` has two untracked paths. Vestigia outer `main=e1a4acd09b9fe8c11f97cf5a284412c93b4b0df9` remains deletion-heavy; nested TraceIQ `master=692d034f598e06652c84f49cadf6b1ba9e7fe144` is clean. Root ownership does not absorb these independent repositories merely by containing their directories.

| GitHub Probata PR, exact open head | Current PR-level state | D12 gate |
|---|---|---|
| #29 `fdb4585644312acd5dabf714ecbc3cb5b145464d` | Draft; Python Validate failed; Go/CodeQL passed; mandatory live integration skipped | HOLD |
| #30 `fdd390d64122fdd411ad37aff370f0b2686fbde3` | Open, currently not draft; Python Validate failed; Go/CodeQL passed; live integration skipped | HOLD despite nondraft flag |
| #31 `d78ca38f835d3c162b7a8be1af472c42096a8dfb` | Draft; same red/skipped baseline | HOLD |
| #32 `3c1967512407a4b22e87cfc5efc0d831167b6782` | Draft; exact-head independent review reports no remaining PR-owned HIGH/CRITICAL; Python Validate failed; live integration skipped | HOLD |
| #33 `e7e7f1466a8ace01eab9e565bb9de01ebbd4f12a` | Draft; same independent review result and red/skipped CI | HOLD; cross-store analysis proof open |
| #34 `a8ec7adad1a5186b3331c06b8bde58e1b35f3c40` | Draft; same independent review result and red/skipped CI | HOLD; trusted signer, atomic adapter/outbox, Legal activation open |
| #35 `05e6e7dfb728811a19bbfbb761edad8e85d601b2` | Draft; same independent review result and red/skipped CI | HOLD; live publisher, route, bytes, privacy proof open |

The independent review result for #32–#35 is reported by the lead; this D12 pass independently checked the live PR heads/status checks, not the review text. The repeated Python failure is attributed in the lead receipt to formatting of two unchanged Claude-owned tests (`tests/test_authentik_deploy_contract.py`, `tests/test_docker_user_firewall_contract.py`). D12 does not treat that as a PR-owned defect, but the required CI gate is still red. The Go job passing does not by itself reconcile static toolchain declarations: `.github/workflows/validate.yml` says `1.25.7`, while `modules/engine/go.mod` says `1.26.6`; pin the effective downloaded toolchain and build image before W30.

## Service source, reverse dependency, and receipt map

| Service family | Build/CI/deploy source and reverse dependency | Present proof / next acceptance |
|---|---|---|
| Probata API, worker, tool runtime, Docstore | `pyproject.toml`, `requirements.txt`, `scripts/validate.sh`, `.github/workflows/validate.yml`, `Dockerfile`, `deploy/compose.yaml` and per-service YAML; `server/**`, `sql/**`, `scripts/**`, and `plugins/docstore/control/**` feed multiple images | Static source only for the full service set; require exact image digests, schema/migration IDs, redacted config inventory, non-skipped integration, Docstore readback and rollback compatibility. Root-preserved scripts/plugins may differ from deployment-owned Probata copies. |
| Proffer starter/worker, parser runtime, gateway | `modules/engine/go.mod`, vendor tree and `deploy/{proffer-starter,proffer-worker,parser-activity-runtime,tool-gateway}.yaml`; SBV submodule influences development build, vendored snapshot influences image | Go CI jobs passed on PR heads. Require SBV ref/vendor digest, effective Go toolchain, parser attempt/partial-sink recovery, and image-level readback. |
| Probata Workbench | `modules/workbench/web/package.json`, Workbench Dockerfile and `deploy/workbench.yaml`; consumes platform API/Proffer and B2/R2 source access | Lead receipt names Coolify app `xjbuo6drbwjfby75lalk8bk7`, code `200d03c`, Tailnet pages and 400-row B2 pagination; R2 is degraded. Browser and scoped deployed checks do not establish native parity or full search/source flow. |
| Consignatio Intake | `Intake/package.json`, backend `pyproject.toml`, Tauri config, `Intake/backend/deploy/surreal-intake.compose.yml`; consumes source filesystem, Surreal #1, backend search/analysis | Lead receipt names hosted-engine app `dbae59tufgs5zqvb7ym9fozk`, deployment `l14471gjkum4l3n6hnrszwxc`, running code `309a27f985426747cb9c163106b06d38da5bf62b`; narrow health/name-search/bounded analysis passed. Browser bundle and native filesystem journey remain unproven. No tracked CI workflow was found in the prior D12 pass. |
| Advocatio Legal API/web/renderer | `pyproject.toml`, `web/package.json`, `compose.yaml`, `deploy/Dockerfile.api`, `deploy/Dockerfile.web`; consumes signed D08 LegalSourcePackage/status and matter state | Lead receipt names app `gvghzivfmctev8dloetfssnj`, deployment `haj8fv8uvbg696a0h184fr22`, code `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34`. 181 tests/one expected failure and scoped live auth checks passed; REAL human identity ingress and verifiable D08 issuer/digest are held, imports accept zero. No whole legal restore or release test. |
| FL-MCP native Workbench | Root-owned Vite/Tauri package and sidecar/UI scripts | No current native/device or release proof; inclusion requires an explicit profile decision. |
| Vestigia/TraceIQ geo | Specialist outer/nested repositories with independent histories and historical build/config sources | W28 is parked unless geo is expressly activated; if activated, prove raw-count/provenance, derived versions, rebuild, and explicit Probata promotion. Dirty outer tree cannot be a candidate. |

The lead's dated live receipts identify narrow deployed code and actions but do not provide one synchronized digest manifest for all running containers, database schemas, frontend bundle, config values-redacted digest, and native artifacts. The D00/WS00 receipt proves one Xplorer build-kit history/tree import (23/23 entries); it does not prove fresh-clone builds of all child repositories. Consignatio, Legal, TraceIQ, and other import holds remain explicit.

## D12 gate ledger and original finding aliases

Every row below is `HOLD` at system acceptance level. The transferred work register marks each `not_started`, and corresponding `AC-W-Wxx` rows are `not_executed`; narrow domain tests do not close them. These source aliases are retained verbatim for lead traceability.

| Gate | Original linked findings | Acceptance condition and exact missing proof |
|---|---|---|
| W13 / `AC-W-W13` | FLOW-01, DATA-01, OPS-01, DATA-03, INTAKE-01, SEC-01, TEST-01, R01, R03, R10, R19, R34, N13k, PB01, PB07, PB10 | Exact named workflow succeeds and denies/fails safely from real entrypoint; independent fixture and bounded approved real sample; browser and native separate; no lost member or scope escape. No cross-domain trace. |
| W18 / `AC-W-W18` | SBV-01, R05, R17, R32, N07, N08, N09, N10, PB02, PB06 | UTF/quoted-newline CSV, XML/MMS, nested malformed packages, null/truncated enrichment, unchanged rerun, crash/resume and counted coverage under measured limits. Bounded D04 ZIP work is PR-only, not full lifecycle proof. |
| W28 / `AC-W-W28` | R33, N13g | Parked for non-geo profile; activation requires known export, unknown record, geocoder disagreement, raw-count reconciliation, derived version and source drill-down. Do not silently elevate parked geo into a release prerequisite. |
| W29 / `AC-W-W29` | OPS-02, LEGAL-01, LIFE-01, R16, R24, R27, R28, PB02, PB08 | Backup manifest plus independent isolated restore of source, PostgreSQL histories, both Surreal roles, search and Legal; keys/config available by role; semantic identity/approval/version parity, user-flow reopen, compatibility rollback and alert test. Intake's dated Surreal drill is narrower. |
| W30 / `AC-W-W30` | ARCH-01, TEST-01, R18, R19, R28, N13e, PB03, PB08 | Fresh clone/bootstrap of chosen refs, complete child/submodule resolution, exact toolchain/lock/vendor/image/schema/contract/config manifest, isolated deployment, provenance readback. Current PR CI and local refs do not form this candidate. |
| W31 / `AC-W-W31` | R31 | Optional cache/coordination role only with approved measured benefit, authority boundary, loss/rebuild/isolation test. No new cache/DB is a baseline prerequisite. |
| W33 / `AC-W-W33` | TEST-01, R01, R19 | Named profile, full linked W13/W29/W30 receipts, no critical gap, owner decision, exact artifacts/signatures, post-deploy readback. None exists. |
| W34 / `AC-W-W34` | FLOW-01, R05, R17, PB02, PB03, PB06 | Numeric scratch reserve, per-lease admission, concurrency/retry/expansion/output caps; pre-allocation oversize refusal, same-key replacement, disk-full, duplicate seal, interrupted/cancelled/crashed lease recovery and multi-worker accounting. Numeric candidate limits and stress receipts absent. |
| CP14 | TEST-01, LIFE-01, TRACEIQ-ROLE, NF-08; W13/W28/W29/W30/W31/W33 | Exact-profile recovery and conformance with native/authorized live checks, key access, projection rebuild and owner attestation. A green isolated test or prior backup is insufficient. |

`CANON:A01`–`A22` remain `specification_not_executed` in the transferred seed. A01–A04 require D02/D04 source/package identity; A05–A07 D03/D10 search/coverage/source opening; A08–A11 D07 two-role graph and authored-work durability; A12–A18 D08/D09 promotion, inbox, outage/duplicate, revocation; A19–A21 D11 exports/privacy/authorization; A22 requires the integrated W29 restore. A profile matrix must explicitly mark each as included or excluded with owner rationale and user-visible limitation; the canon's first useful slice (§26, line 678) still requires screenshot/text search, source reopening, one workspace, both promotion entrypoints, legal readback and private timeline export before that slice is called working.

## Recovery and operating envelope

Durable recovery inputs are versioned B2 source bytes and occurrence/package metadata; PostgreSQL records, provenance/approval/history and authored work; derivation/locator/model/config versions; released package bytes/manifests; and separately recoverable secrets/key access. Surreal #1 and #2, Weaviate, Neo4j/temporal views and browser caches are projections only if all authored choices and source links can be rebuilt without loss. Reprocessing creates a new derivation version when parser/OCR/embedding identity changes. Source correction marks dependent index/analysis/legal views stale; revocation invalidates current legal review/release reliance while old release receipts remain immutable. Partial sinks/indexes must expose coverage and allow idempotent replay. Retention periods, backup frequency/RPO/RTO, numeric scratch reserve and service CPU/memory limits are **not approved or measured in this packet**; W29/W34 must record the actual values per selected profile before load/restore acceptance.

Presence-only secret/config inventory for the selected profile must identify B2/R2 object access, PostgreSQL and both Surreal roles, Weaviate/Neo4j when active, signer/issuer verification keys and rotation trust, Legal inbox/service credentials, Authentik public ingress, Tailnet trust, and backup encryption/recovery key custodians. Record names, owner, scope, rotation/readback method and redacted config digest; never put values in this packet or a CI artifact. Current R2 entitlement failure means Workbench source read cannot be certified. REAL identity and D08 trusted producer remain distinct legal activation blocks.

## Exact profile and release control

No owner-approved release profile ID or inclusion/exclusion matrix was found. Therefore **no exact release profile currently passes**. The next candidate record must name one immutable `profile_id` and scope (bounded context slice, legal-consumption slice, guide publication, geo expansion, or another owner-chosen scope), included services and source types, each A01–A22 applicability and exclusion rationale, feature flags, public/Tailnet ingress, data mode, schema/migration snapshot, image/native digests, redacted config digest, backup/restore target, rollback, and owner decision reference. These labels are planning categories, not approved profiles. Guide publication is a separate conditional gate: D13's 2026-09-23 source-quality packet says `PUBLICATION BLOCKED` pending current guide tree, statute/case/local currency, contradictions, release manifest and reviewer signoff; it cannot be inferred from root publication of that research packet.

**Release decision and priority:**

1. **HOLD, P0:** recover correct-account R2 entitlement and verify scoped list/read plus staging permission; complete REAL human identity ingress and trusted D08 producer/signer/digest/atomic delivery. Do not accept zero-item Legal import as end-to-end success.
2. **HOLD, P0:** reconcile D00 child/build/deployment-source map and paused local commits; select one immutable profile/candidate. Preserve dirty work. Complete exact image/config/schema/vendor/toolchain manifest and fresh clone/isolated build (W30).
3. **HOLD, P0:** resolve the two Claude-owned formatter baseline files in their owning lane and rerun required PR checks; PRs #29–#35 remain unmerged while Python Validate is red and live integration skipped. Retain exact-head review receipts for #32–#35 but do not treat them as whole-system proof.
4. **HOLD, P0:** execute W13 user-entrypoint/negative/native and W18/W34 parser/fault/resource trials on the selected candidate; record counts, limits, partial-sink and stale-index recovery.
5. **HOLD, P0:** perform W29 isolated multi-authority restore with semantic source/legal readback, key access, compatibility and rollback evidence; then W33 independent post-deploy review and owner release decision. Include D13 signoff only if guide publication is in the profile. W28/W31 remain parked unless explicitly activated.

Use `D12_EVIDENCE_RECEIPT_TEMPLATE.json` for each execution: exact candidate and profile, input/fixture authority, command/environment, observed success and fault behavior, limits, artifact hashes, readback, independent reviewer and owner decision. This packet itself closes no work item and authorizes no external release.
