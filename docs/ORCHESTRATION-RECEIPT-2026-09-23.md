# Propria orchestration receipt — 2026-09-23

> Byline: Codex · GPT-6 · lead/orchestrator · 2026-09-23. This is a scoped execution receipt, not a full-platform release certificate.

## Owner directives in force

- Start from `START_HERE.md`, then use the orchestrator and completion report as routing inputs rather than proof that work is complete.
- Assign each domain implementation and independent review to a separate agent when scheduler capacity permits.
- Push, merge, and deploy work that passes its actual gates; preserve explicit holds rather than bypassing them.
- On Tailnet, Tailscale is the security barrier and user surfaces must not add a second application login. Off Tailnet, user surfaces enter through one Authentik-backed portal. Claude owns portal listings, route publication, and the shared user-facing bundle.
- Never delete protected or unrelated work, reset/stash/clean shared worktrees, rewrite history, expose credentials, or restart the host.

## Repository and publication state

| Scope | Published state | Current disposition |
|---|---|---|
| Propria root | `main` / `origin/main` at `ea7fcf0d74d808b1378ac13ec81343448b4bd8d5` | Xplorer build-kit history and root reconciliation receipts are pushed. Root is clean except four protected pre-existing untracked paths: Claude's portal audit, two preview files, and `session/`. |
| Probata main / Workbench | `main` / `origin/main` at `382acf6a8f1f919b6f3abc02de40f946629d8461` | P0 pagination and Tailnet access changes are deployed. The main checkout is clean except the protected nested `modules/forks/sbv/.cnf` state. |
| Consignatio Intake hosted engine | `feat/hosted-intake-engine` / private remote at `6d6d8bd5ba8bd3bce0b36b30991bf5cebe1c5442` | Bounded hosted folder analysis fix is deployed and live health/name-search/bounded-analysis checks passed. The quarantined rollback directory remains intentionally untracked. Claude still owns the browser bundle and portal pointer. |
| Legal / Advocatio | `master` / `origin/master` at `06b69433163537bb3d66925149694ca1a12ac689` | Identity fail-closed remediation is deployed. Tailnet user pages and API health pass; forged proxy headers are denied. Verified human Tailnet identity ingress, the D08 producer contract, and public Authentik issuer configuration remain held. |
| Consignatio D03 | PR #3 merged as `cdc36470f4aa26ffafe953ec591072ec1a65519e` | OCR literal verification and decomposed Hangul/Jamo handling passed review and tests. No deployment source was identified, so this is a merge-only result. |

## Open Probata pull requests

| PR | Head | Result and hold |
|---|---|---|
| [#29](https://github.com/Cursedpotential/probata/pull/29) | `fdb4585644312acd5dabf714ecbc3cb5b145464d` | D02 bounded XML probe; reviewed and secret-scanned. Draft/HOLD because repository validation formats two unchanged Claude-owned security tests and full downstream gates remain open. |
| [#30](https://github.com/Cursedpotential/probata/pull/30) | `fdd390d64122fdd411ad37aff370f0b2686fbde3` | D05 exact review-target correlation; independent reviewer fixed source-version/normalized-generation binding. HOLD on the same two out-of-diff formatter files and unproved live/atomic gates. |
| [#31](https://github.com/Cursedpotential/probata/pull/31) | `d78ca38f835d3c162b7a8be1af472c42096a8dfb` | D04 bounded ZIP/OOXML inventory; independent hardening covers path collisions, special modes, offsets, retained-source digest, and package markers. Draft/HOLD on the same formatter baseline plus D12/live/full NF-PR-01 gates. |
| [#32](https://github.com/Cursedpotential/probata/pull/32) | `3c1967512407a4b22e87cfc5efc0d831167b6782` | D06 source-bound mention atoms. Review remediation makes class disagreement a retained conflict, canonicalizes spans/output, and rejects blank coordinates. Forty-three focused/adjacent tests and scoped quality/security checks pass. Draft/HOLD pending new-head independent review and the same external formatter gate. |

The two external files are `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`. They are not part of PRs #29–#32 and remain in Claude's lane; this orchestration run did not modify them to manufacture green CI.

## Live deployment receipts

### Probata Workbench

- Coolify application: `xjbuo6drbwjfby75lalk8bk7`.
- Deployed code receipt: `200d03c` with the later repository receipt state pushed through `382acf6`.
- Tailnet `/` and `/sources` returned 200; two-page B2 pagination returned 400 rows; unauthenticated public/spoofed requests were redirected to Authentik; raw-host and direct-loopback bypass attempts were denied.
- R2 remains degraded. Every active rclone `r2` profile found locally and on the scoped VPS hosts is the same key and returns `NotEntitled` for `casebible-sorted` and `nexus`. The only distinct saved Cloudflare-looking profile returns `SignatureDoesNotMatch` when corrected solely for a read-only probe. No Coolify credential was changed.
- Recovery requires a correct-account R2 key with Object Read on `casebible-sorted` and Object Read & Write on `nexus`, followed by exact list/read and staging-scope proof before deployment.
- Redacted recovery receipt: `modules/Probata/probata/docs/reviews/2026-09-23-rclone-r2-credential-recovery-blocker.md`; Docstore readback revision `docstore_revision:924fe499fd57a010f8c3d0d79c76083e64fcdbf9ad928fe774a140e5886538c8_1`.

### Intake hosted engine

- Coolify application: `dbae59tufgs5zqvb7ym9fozk`; deployment `l14471gjkum4l3n6hnrszwxc`.
- Running code: `309a27f985426747cb9c163106b06d38da5bf62b`; docs-only receipt head: `6d6d8bd5ba8bd3bce0b36b30991bf5cebe1c5442`.
- Hosted mode no longer performs unbounded recursive duplicate scans. Root analysis fails fast, direct entries are capped, preflight listing and overall analysis are bounded, and abandoned work is cancelled.
- Live `/healthz`, guarded-root, name-search, and bounded two-PDF Smart Suggestions checks passed. This is backend proof only; it is not a claim that Claude's browser bundle has been refreshed.

### Legal / Advocatio

- Coolify application: `gvghzivfmctev8dloetfssnj`; deployment `haj8fv8uvbg696a0h184fr22`.
- Deployed code: `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34`; master receipt head: `06b69433163537bb3d66925149694ca1a12ac689`.
- Full tests passed (181 passed, 1 expected failure); lint/type/diff/secret checks were clean. Invalid bearer input fails closed rather than downgrading to Tailnet identity.
- Approved imports intentionally accept zero items until a verifiable D08 producer issuer/digest exists.

## History and merge holds

- No second-history import was forced into root. Consignatio, Legal, and TraceIQ histories remain held because redacted scans found candidate findings; Vestigia remains held because of 2,085 deletions/content mismatch; Probata/SBV retains its active nested-repository boundary.
- No `master` to `main` conversion, child-pointer update, force push, reset, clean, stash, or broad stage was performed.
- The root outbound history scan used Gitleaks 8.30.0 with full redaction and reported zero findings for the pushed root candidate.

## Active and remaining domain queue

- D07 is active under a fresh agent: accepted analysis/horizon roles, competing processor lanes, knowledge clocks, as-of/as-lived/hindsight/delta/rewalk semantics, contradiction retention, and durable authored workspace state. It cannot promote evidence or legal conclusions.
- D08 still owns the one promotion/legal/investigation exchange and must provide the verified producer identity/digest that Legal currently requires.
- D09 consumes that governed exchange; deployed Legal hardening is not full D09 completion.
- D10 reusable UI work remains subordinate to each domain's authority and to Claude's portal-surface ownership.
- D11 private labeled narrative/vault-routed export work, D12 end-to-end integration/restore proof, and D13 independent legal-reference verification remain to be executed and independently reviewed.

## Proof ceiling

This receipt proves the enumerated commits, pull-request heads, local checks, live endpoint checks, deployment receipts, explicit holds, and workspace-boundary observations. It does not certify the entire 30-page program complete, does not claim browser acceptance for Claude-owned surfaces, does not convert a Docstore save into indexing proof, and does not convert a failing R2 credential into storage readiness.
