<!-- Updated by: Codex (migration-passes/d09) | Date: 2026-09-23 | Rev: 2 | Platform: Codex / win32 | Changes: strengthen audit binding, rejection taxonomy, and replay semantics | Context: independent review remediation -->

# D09 — LegalSourcePackage v1 consumer boundary

## Scope

This receipt covers a deliberately inert Advocatio consumer boundary for the
Indicia Probata D08 contract. It does not activate a production import, resolve
an issuer key, contact Probata, persist an accepted package, merge, or deploy.

## Provenance

- Advocatio base: `origin/master@06b69433163537bb3d66925149694ca1a12ac689`
- D08 producer contract reviewed at:
  `a8ec7adad1a5186b3331c06b8bde58e1b35f3c40`
- D08 schema: `legal-source-package/v1`
- D08 signature algorithm: `ed25519`

## Implemented boundary

`LegalSourcePackageV1Consumer.accept_if_current` names one indivisible consumer
operation. A future production adapter must verify the canonical signed manifest,
read one producer snapshot for the package and all items, and conditionally
persist the accepted package and availability receipt only while that exact
producer revision remains current.

The shipped `HeldLegalSourcePackageV1Consumer` always rejects. It is not mounted
in an API route and cannot alter current workspace persistence. The existing
unsigned `1.0` envelope remains blocked by the pre-existing D08 hold.

Independent review remediation binds an availability receipt to the exact D08
schema, matter/package UUIDs, issuer, issuer key, Ed25519 algorithm, signed
manifest digest, signature, producer revision, and aware acceptance time. The
consumer input is fully keyword-only and all fail-closed outcomes have stable
typed codes. A replay must match the complete signed-package identity, reverify
signature and currentness, and return the original receipt without a second
write; a reused UUID with different identity is an explicit conflict.

## Activation gates still held

1. A configured trust store mapping the expected issuer and `issuer_key_id` to a
   trusted public key.
2. Strict wire parsing compatible with the exact D08 canonical encoding.
3. One authenticated producer snapshot adapter covering package and every item.
4. A producer current-revision compare combined with consumer package and receipt
   persistence as one durable operation, with retry/idempotency behavior proved.
5. Signed revocation/supersession/unavailable event consumption and chain state.
6. Cross-repository fixtures and live failure/recovery/concurrency proof.
7. Independent security review before any route is mounted or old state is
   migrated.

## Verification

- Remediated boundary tests: `15 passed`.
- Boundary plus adjacent source-package/citation/first-slice/review-release
  tests: `37 passed`.
- Full Legal suite: `196 passed, 1 xpassed`.
- PR-owned Ruff check and format check: pass.
- Python sdist and wheel build: pass.
- Repository-wide Ruff remains a pre-existing baseline failure outside this
  allowlist: 111 lint findings and 43 files requiring format. No baseline file
  was changed to conceal that condition.
- Final diff check: pass; changed paths remain limited to the same three PR
  files.
- Gitleaks 8.30.0 exact-remediation-commit scan: zero findings.

The pytest run emits one pre-existing `pytest-asyncio` deprecation warning from
the historical virtual-environment package path. No live integration is claimed.
