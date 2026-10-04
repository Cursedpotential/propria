# Family Court inventory execution — 2026-10-04

Byline: Codex, 2026-10-04.

This is a progress receipt for the owner-approved convergence recorded in Docstore note:family_court_convergence_20261004. It does not replace that decision.

Verified:

- Propria main pushed through 67e46bbd; owned changes integrated with explicit paths and unrelated work preserved.
- Workdesk Orders and Memos exposed; node table-parity test passed. Workdesk deployment and browser behavior remain unverified.
- Native Go ZIP inspection registered in the existing Proffer worker; targeted tests and go vet passed.
- Coolify deployment 35oetj52qs3redvmslsckyz4 finished. Mounted source readability and receipt-directory writability verified.
- Temporal namespace default / queue proffer-v1: workflow toolkit-package-inventory-20261004-v1, run 01a10888-7939-773c-81f5-167a42d8a2ba completed 2026-10-04T20:08:47Z.
- Inventory readback: all 15 ZIP packages complete, zero package errors, streamed archive/member SHA-256, ZIP CRC checks and 429 identical-member candidate groups. This establishes archive integrity, not legal accuracy or source-unit equivalence.
- Inventory receipt on ovh-files: /data/probata/volumes/proffer/derive-scratch/toolkit-inventory-20261004/receipts/inventory-v1.json; SHA-256 6ac30fbb95c8e95e018a2f76a5f4e0b4911f5d46e00828972b2eb8b1599bb10b.
- Selected-text worker deployment lqd2tjgxjg36oopcjikmikvj finished. Workflow toolkit-selected-text-20261004-v1, run 01a1089f-3d63-7691-b050-e8b9bce70588 completed 2026-10-04T20:33:36Z. Seventeen original text members snapshotted with verified source identities; 2,328,512 raw text bytes.
- Snapshot: /data/probata/volumes/proffer/derive-scratch/toolkit-inventory-20261004/receipts/audit-text-v1.json; SHA-256 591f8c58ab4dc7535391a38e435e16c7e9ae27673eca53a8535dbfe5b1266dab. No source bodies placed in Git or Temporal history.
- The exact shared record reference:custody-guide-verification-ledger-md already contains the final August 12 Hayes/Duperon corrections. Its stored source SHA-256 cb471a29e3ad43fe38b173f3a77b5ddfb3beb67a3063fb16f9b01cce51e05efc matches recovered snapshot member 13; version sha256:88f7c153905454a26f6e9abb9691afb9adba6d689b49dfe9eebae5c3adcac45e. The hosted packaged file is older, so replacing the working library with bundled originals would regress it. See the separate ledger review receipt for bounded legal-source observations.
- Staging originals retained under /data/probata/reconciliation/family-court-20261004; local originals untouched. These are archive integrity fingerprints, not comparison to separately hashed desktop originals.
- Desktop configuration repairs 830f719c and f69adb38 integrated; eight synthetic tests passed. Live read-only configuration resolves canonical plugin and shared OVH Surreal fct/case; counts source 193, reference 321, order 0, memo 0. Probe made no schema/data writes.
- Mobile pagination commits 06ffdf7d and 690d7b97 integrated; three meaningful tests passed, including more than 1000 fixture records and more than 300 library entries. Canonical plugin local commit 7b97f2b updated the same three source/test files; plugin commit hook refreshed Claude and Codex caches.
- Hosted console deployment abko4fntkz5xlvu0ybqvcg89 finished at build 690d7b979f12e9bd29165c0fd4766b6124176eb8; container family-court-console-sokv65ibdq2y8xdaqmd6p4rq-203156496467 healthy.
- Authenticated Tailscale API readback from the desktop: source limit=1 returns total 193 and cursor 1, next page has a distinct ID and cursor 2; reference limit=100 returns 100 of 321 and cursor 100; source cursor=193 terminates with no rows/no continuation. Invalid limit 1001 and cursor -1 return HTTP 400. VPS service-tag caller gets HTTP 403. No records changed.
- Ledger comparison b9e05250 integrated and pushed, targeted toolkit/worker-registration tests and go vet passed. Separate Workflow/Activity compare pinned snapshot indices 14/15/16 by unique source ID, added/removed IDs and differing field names. Live workflow toolkit-ledger-comparison-20261004-v1, run 01a108b4-f70f-772c-99a2-b8a8da1b353c completed at 20:57:20Z. Receipt ledger-comparison-v1.json is 84,469 bytes, SHA-256 67f89bbefc97f5a560037d8bff6ab0280c0851c309cdf39703ef2c62eff1de97. Each candidate has 191 rows; the 15-to-16 pair has 50 changed shared rows only in audit_provenance plus the plan-ID rename. This mechanical receipt does not certify every legal claim.
- Phone Library now reads paged shared source/reference records and opens exact versioned details. Packaged originals are a separate explicitly labeled view. Graphite/indigo tokens are served from the shared design contract; deployment ybid72pgudreuezbvlbang44 finished and authenticated token assets returned HTTP 200.
- Fresh content reads 0a3d3af2 remove the five-minute reference/source/lexicon cache and production silent fallback. Nine focused tests passed; direct read-only SDK query verified all 193 source IDs/cursors, with an empty terminal next page.
- Coolify deployment onwwvflsmvcsxazplbhyovsn finished at 0a3d3af2. Container family-court-console-sokv65ibdq2y8xdaqmd6p4rq-212437119661 healthy; authenticated phone root HTTP 200 with dark theme. Live read-only audit_sources, court_language_review and survival_guide calls succeeded. Audit reports 193 ledger records plus 213 directory records and 7 curated records.
- Desktop native source/reference pages and full caseRecord detail integrated through 67e46bbd. Eleven synthetic sidecar tests passed, including paging beyond 200 and malformed/nonadvancing page rejection; focused TypeScript check passed in the isolated worktree. Actual desktop rendering remains unverified.
- Canonical plugin changes committed locally through 3b03abb; both installed plugin caches refreshed. LF source normalization preserved; remote marketplace publication remains pending review of earlier policy commits.
- Owner library/storage contract recorded in modules/Legal-desktop/AGENTS.md, commit 3e600c8d.

Open:

- Substantive audit/correction comparison and final validated baseline selection. Mechanical diffs do not select a legal baseline.
- Durable Case Bible original placement and independent remote readback, then governed catalog/projection receipts. Bounded preservation implementation active; no B2 placement claimed.
- Reconcile remaining source/version/hash/pinpoint/validation evidence in surreal-case fct/case without overwriting existing final corrections or current edits. The final verification ledger is already present.
- Desktop native list/detail implementation integrated and tested. Desktop chat still starts a local MCP server; hosted ContextForge transport and actual desktop UI proof remain open.
- Workdesk native reference integration, bidirectional edits, citation validation on updates, remaining shared graphite/indigo surfaces and actual phone/desktop/workdesk click behavior remain unfinished.
- Private plugin marketplace remote publication is pending: local main has two earlier unrelated committed Case Bible changes ahead of origin, so this session has not pushed those implicitly.
- Docstore revision capture is not indexing, approval or filesystem synchronization proof. This local receipt is explicitly materialized alongside the captured revision.
