# D08 LegalSourcePackage contract slice

**Byline:** Codex D08 · GPT-6 · 2026-09-23 (America/New_York)
**Repository:** Indicia Probata, isolated branch `codex/d08-legal-source-package-contract-20260923`
**Base:** `origin/main@382acf6a8f1f919b6f3abc02de40f946629d8461`
**Proof ceiling:** local synthetic unit and static source inspection; no production issuance, deployed pickup, restore, or legal release proof.

## Result and boundary

The new import-light `server/contracts/legal_source_package.py` defines a versioned producer/consumer wire recipe. It canonically sorts exact evidence/assertion versions, binds matter, issuer/key ID, promotion ID, retained source occurrence/version/locator, source and assertion digests, verification and human-authorization receipt IDs, status, and committed time. Its canonical UTF-8 JSON manifest has a SHA-256 digest and deterministic package identity. A detached `ed25519` signer callback signs domain-separated manifest bytes; a consumer callback must verify with a trusted producer key. Signed append-only invalidation events chain from the package digest or prior event digest. This module owns no signing key and reads no evidence or case data.

The verifier rejects wrong issuer, matter, schema, package identity, digest, signature, source ref, span, version, locator, and current status. A single canonical producer snapshot must bind the package identity, all evidence statuses and exact items to one revision. The consumer acknowledgment callback must compare that same revision against the producer and durably record availability before returning success; changed or unavailable revision fails closed. The pure module defines this interface but cannot enforce a future network/database adapter's atomicity. A successful acknowledgment establishes package **availability at that revision** only. Later invalidation must still reach the consumer. It does not adopt a legal assertion, approve a draft, release a document, file, serve, or promote evidence.

## Current-source reconciliation

- Canon §§16–18, 20 require one promotion authority/history for eligible context and investigative entrypoints, a source reread, human authorization, committed evidence version, durable delivery/read-back, and separate legal adoption/release. `D-152` in `docs/DECISION_LOG.md` expressly defers custody hashing and real promotion. `D-154` requires reopening the retained original at any later admission; a context fingerprint is not custody.
- Current `server/case_management/repository.py` reports `advanced_evidence_available=False`. Its latent `promote_evidence` writer still inserts into `analysis.evidence_item`, while the snapshot defines `evidence.evidence_item`; `server/case_management/service.py` gates the route. No existing route is enabled or altered here. Both future entrypoints must call the same promotion command and PostgreSQL history; neither Workbench flag nor Surreal analysis may become an alternate authority.
- Current Advocatio `api/legal_workspace/services/source_package.py` rejects approved package import because D08 issuer, canonical manifest, and status evidence are unavailable. The 2026-09-23 Legal deployment receipt identifies deployed commit `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34` with this hold. This D08 contract is not yet consumed by that deployment.
- `W21` / `CP03`: the exact package verification slice is implemented locally; source verification, issuer trust-root configuration, importer adapter, current-status service, and end-to-end legal read-back remain open. Original A aliases `LEGAL-02`, `R10`, `R11`, `R14`, `R22`, `R25`, `R30`, `N11`, `PB05`, `PB07`, `PB11` remain attached to W21 and are not closed by this slice. B `LSP-CONSUMPTION` / duplicate `GOV-NF-07` is mitigated on Legal by its current fail-closed hold, with valid import still unavailable. `GOV-NF-02` remains D-152-deferred with a latent SQL writer defect.
- `W24` / `CP12`: original A `R12` and B `MISSING-PROOF-FEEDBACK` / duplicate `GOV-NF-08` remain open. The current Legal request does not yet reach a Probata receiver or close against its originating issue. `ADV-J1`, `ADV-J7b`, `GOV-R03`, `GOV-R04`, `GOV-R15`, and `GOV-R16` remain integration requirements, not code-complete claims.
- Canon `A12`–`A14` (entrypoints and failed promotion), `A15`–`A16` (automatic pickup/outage and duplicate), `A17` (wrong-scope or tampered package), and `A18` (revocation/stale draft) remain acceptance scenarios. This slice exercises only synthetic portions of A17/A18. No whole-profile acceptance is claimed.

## Required integration sequence, not activated by this slice

1. The authorized promotion owner lifts D-152 and supplies one transactionally committed evidence version, source reread/custody receipt, human principal/action/scope authorization, immutable history, and the matching outbox intent. Repair the latent table mismatch and prove both entrypoints converge before exposing an issuer.
2. The issuer reads only committed versions through a separately permissioned producer interface, signs with a protected Ed25519 private key, and publishes its public trust root/key rotation record. Package version and issuer key ID are pinned. A failed source reread, missing authorization, wrong matter, unresolved source, or stale version never reaches signing.
3. The outbox stores one immutable delivery intent keyed by `(package_id, package_digest, event_type, status_sequence)` in the promotion transaction. A worker retries after outage. The Legal inbox stores one logical availability row per `(issuer, package_id)` and treats same ID/different digest as a conflict. It acknowledges only after a single revision-bound producer read-back and conditional durable acceptance of that revision; a transport acknowledgment does not imply legal adoption. The integration must prove the compare-and-acknowledge semantics under concurrent revocation, including any cross-service consistency boundary.
4. Revocation/supersession/restriction is an immutable signed status event with ordered sequence and predecessor digest. A stale or missing sequence blocks current reliance. Advocatio flags all dependent drafts/reviews for re-review and preserves already released versions with a later correction record.
5. Investigation requests need a separate durable request/outbox and Probata inbox. Stable request ID, originating issue, question/scope/constraints, status, result refs and resolution authority must survive restart. Duplicate transport creates one task; substantive result returns to the original issue and never promotes itself or closes merely on delivery.

The actual outbox tables, Legal importer adapter, public-key verifier, current-status/read-back route, issue receiver, and release invalidation are owned by their respective producer/consumer domains and require current schema/API review. No schema, service permission, live deployment, credential, or evidence status was changed here.

## Verification receipt

| Check | Result | Limit |
|---|---|---|
| Synthetic producer/consumer contract tests | 17 passed | Test signer is deliberately synthetic; does not prove production Ed25519 key management or atomic network read-back/acknowledgment. |
| Adjacent case-management route/capability tests | 51 passed | Existing hard-disabled promotion behavior; no real evidence action. |
| Focused Ruff | passed | Only the two new Python files. |
| Python compilation | passed | Syntax/import only. |
| Official Gitleaks 8.30.0, redacted file scans | passed; no leaks in each of the three candidate files | File scan, not historical repository scan. |
| Wheel build | passed | Packaging only; no application deployment. |
| Build, deployed read-back, outage catch-up, legal adoption, restore | not run | Requires integration and D12 review. |

**Negative cases exercised:** changed span/manifest, forged signature, wrong matter, wrong source or evidence version, unsupported locator, missing/revoked/superseded/unavailable package or evidence status, revocation between snapshot and acknowledgment, boolean version confusion, and tampered/out-of-order/noncanonical or disconnected first status event. A test rerun generates the same package and event identities.

**Rollback:** this branch adds one unreferenced contract module, one test file, and this receipt. An ordinary revert removes the contract without changing live state. The Legal importer remains fail-closed until an independently reviewed producer/consumer integration is deployed.
