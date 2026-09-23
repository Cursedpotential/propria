<!-- Byline: Codex, Propria lead/orchestrator, 2026-09-23 EDT -->
# Propria final orchestration status — 2026-09-23

**Outcome: the bounded implementation, review, deployment, and evidence work completed today is preserved and published; the whole-system release remains on HOLD.** The hold is not a lack of activity. It is the correct acceptance decision because the remaining gates require a valid Cloudflare R2 credential, one owner-approved immutable release profile, green required CI, real identity and signed producer proof, native journeys, and an isolated whole-profile restore.

This status supersedes the queue snapshot in `docs/ORCHESTRATION-RECEIPT-2026-09-23.md`. The detailed independent acceptance basis is `docs/D12-FINAL-ACCEPTANCE-PACKET-2026-09-23.md`.

## Published and deployed today

| Area | Result | Boundary |
|---|---|---|
| Root repository/history | Xplorer build-kit history and the root reconciliation receipts were pushed. The root was aligned with `origin/main` before this publication. | Independent child repositories were not silently absorbed or rewritten. |
| Probata Workbench | P0 Tailnet access and source pagination are deployed. Tailnet `/` and `/sources` returned 200; the B2 source listing proved two pages/400 rows; public, spoofed, raw-host, and direct-loopback bypass attempts were denied. | R2 remains unavailable and browser/native whole-flow acceptance is still open. |
| Consignatio Intake | Hosted bounded-analysis remediation is deployed as application `dbae59tufgs5zqvb7ym9fozk`, deployment `l14471gjkum4l3n6hnrszwxc`, running code `309a27f985426747cb9c163106b06d38da5bf62b`. Health, guarded-root, name search, and bounded two-PDF analysis passed. | This is backend proof; Claude owns the browser bundle and portal pointer. |
| Advocatio Legal | Fail-closed identity remediation is deployed as application `gvghzivfmctev8dloetfssnj`, deployment `haj8fv8uvbg696a0h184fr22`, code `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34`. The suite passed 181 tests with one expected failure. | Real human identity ingress and the signed D08 producer/adapter remain required; zero accepted imports is a safe hold, not end-to-end success. |
| Consignatio D03 | PR #3 was merged at `cdc36470f4aa26ffafe953ec591072ec1a65519e`. | No deployment source was identified, so the result is merge-only. |
| D13 legal-reference quality | Current official-rule/form checks were independently completed and published in `docs/D13-LEGAL-REFERENCE-QUALITY-PACKET-2026-09-23.md`. | Guide publication remains blocked pending current statute/case/local-source verification, contradiction repair, release manifest, and reviewer signoff. |

## Probata review queue

All seven pull requests remain deliberately unmerged. Exact-head independent reviews for PRs #32–#35 found no remaining PR-owned critical/high defect after remediation, but required Python validation is red because two unchanged Claude-owned formatter-baseline files fail, and mandatory live integration is skipped.

| PR | Exact head | Delivered scope | Disposition |
|---|---|---|---|
| #29 | `fdb4585644312acd5dabf714ecbc3cb5b145464d` | D02 bounded XML probe | Draft/HOLD |
| #30 | `fdd390d64122fdd411ad37aff370f0b2686fbde3` | D05 exact review-target correlation | HOLD despite nondraft flag |
| #31 | `d78ca38f835d3c162b7a8be1af472c42096a8dfb` | D04 bounded ZIP/OOXML inventory | Draft/HOLD |
| #32 | `3c1967512407a4b22e87cfc5efc0d831167b6782` | D06 source-bound work-product atoms | Draft/HOLD |
| #33 | `e7e7f1466a8ace01eab9e565bb9de01ebbd4f12a` | D07 temporal derivation schedule | Draft/HOLD |
| #34 | `a8ec7adad1a5186b3331c06b8bde58e1b35f3c40` | D08 immutable LegalSourcePackage contract | Draft/HOLD |
| #35 | `05e6e7dfb728811a19bbfbb761edad8e85d601b2` | D11 governed context-export contract | Draft/HOLD |

The two external files are `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`. They remain in their owning lane and were not modified to manufacture green CI.

## Rclone key recovery result

The requested alternate rclone credential was located and tested without mutating Coolify or production configuration. It is not usable: the normal Cloudflare rclone profiles all return `NotEntitled` for both required buckets, while the one distinct historical Cloudflare-looking profile returns `SignatureDoesNotMatch` under a safe read-only endpoint/config correction. Backblaze profiles are not Cloudflare R2 credentials.

Recovery therefore requires a new credential for the correct Cloudflare account with:

- Object Read on `casebible-sorted`;
- Object Read and Write on `nexus`; and
- successful exact list/read proof plus staging-only write/read/delete proof before any deployment credential is changed.

The redacted blocker receipt is retained at `modules/Probata/probata/docs/reviews/2026-09-23-rclone-r2-credential-recovery-blocker.md`, with Docstore readback revision `docstore_revision:924fe499fd57a010f8c3d0d79c76083e64fcdbf9ad928fe774a140e5886538c8_1`. It remains uncommitted in the Probata checkout so an invalid credential receipt cannot trigger a production deploy.

## Current repository boundaries

- Propria root had four protected untracked paths before this publication: Claude's portal audit, two preview files, and `session/`. They were not staged, moved, or edited.
- Probata local `main` contains two owner-paused Claude documentation commits above `origin/main=382acf6a8f1f919b6f3abc02de40f946629d8461`; they were excluded from PRs #29–#35 and were not pushed, rebased, or merged.
- Probata's nested SBV state and the untracked R2 blocker receipt remain protected.
- Consignatio, Legal, Vestigia, and nested TraceIQ keep their independent repository ownership and dirty-state boundaries.
- No force push, reset, clean, stash, broad stage, host restart, history rewrite, or permanent deletion was performed.

## Release-blocking to-do list

1. Issue the correct scoped R2 credential and prove both bucket permissions without exposing the secret.
2. Resolve the two formatter-baseline files in their owning Claude lane, then rerun required checks on PRs #29–#35.
3. Select and record one immutable release profile: exact repositories/refs, image and native digests, schemas/migrations, redacted configuration digest, feature flags, ingress mode, A01–A22 applicability, backup target, and rollback.
4. Complete real human identity ingress and the trusted D08 issuer/signer/digest plus atomic delivery/outbox and Legal consumer activation.
5. Execute W13 entrypoint/negative/native journeys and W18/W34 parser, fault, capacity, interrupted-lease, and recovery trials against the selected candidate.
6. Perform W29 isolated multi-authority restore with semantic source/legal readback, key access, compatibility, and rollback evidence.
7. Complete W33 independent post-deploy acceptance and obtain the owner's explicit release decision. Keep W28 geo and W31 cache work parked unless the owner activates them.
8. If guide publication is included, repair D13 contradictions and verify current statutes, cases, local rules, release manifest, and reviewer signoff first.

## Access and ownership contract retained

- On Tailnet, Tailscale is the security boundary; user surfaces do not add a second application login or bearer prompt.
- Off Tailnet, user surfaces enter through one Authentik-backed shared portal.
- Claude owns user-facing portal/app listings, route publication, and shared browser surfaces. This orchestration did not overwrite those files.

## Proof ceiling

The work proves the named commits, PR heads, reviews, tests, scoped live probes, deployment receipts, source checks, credential failures, and repository boundaries. It does not prove a synchronized whole-system candidate, native acceptance, complete recovery, publication readiness, or release authorization. The D12 decision is therefore **HOLD** until the listed acceptance gates are evidenced.
