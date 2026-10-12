# G13 — Carry verified case context into native AI analysis

October 11, 2026. Core resolution accepted by the owner: existing Case registry, separate search/case analysis, and scope fixes accessible from Sources. Exact wire/version details remain to reconcile. G17 core working-material, independent strategy and shared mobile/legal-workdesk design is accepted. This document plans G13; it authorizes no implementation or operational change.

## Verified gap and severity

G13 remains the AI BFF/candidate-staging scope gap in the 36-gap register. Severity P1. Owner: Probata Sources/BFF, Go native-context workflow and candidate-review adapter. Existing canonical case identity service owns registry truth.

Baseline B and restored R AI BFF models omit matter_id and court_case_id (E21/E22). The newer native branch N still omits them (E86). The baseline Go API accepts optional IDs and includes them in deterministic workflow identity, but its request validation only bounds coordinates (E87).

Native source registration accepts optional scope; missing/incomplete/invalid UUID pairs become SQL NULL coordinates rather than a verified case association (E88). Candidate staging subsequently requires the existing approved pair and LIVE operating context (E23). Therefore passing source registration does not establish readiness for case-specific staging.

The baseline workflow attempts candidate staging before embeddings. A staging error does not prevent semantic publication/readback; the final result correctly reports partial/candidate_review_failed (E89). The defect is the missing end-to-end scope contract and actionable independent outcomes. Do not claim the baseline always reports complete or that publication occurs before the staging attempt.

Branch divergence matters: newer N retains the BFF omission but has no baseline candidate-review staging Activity in the inspected workflow (E91). Remote main M and placement R5 lack all four native entrypoint files inspected here. Absence is not a delivered replacement or deployed defect proof. Reconcile these branches before a release claim.

## Proposed user experience

Sources shows the current Case registry context beside the case-analysis action. Resolve the owner's existing single matter/court-case relationship through the established service; do not require typed IDs or invent multi-case selection machinery. Reuse a verified current context when available.

Keep these actions and outcomes distinct:

- Find, inspect, group, classify and permanently place a source/package without requiring case enrollment.
- Make authorized raw AI content searchable without a universal PostgreSQL content-ingest hop.
- Analyze for the current case and stage reviewable candidates with an explicit verified case binding.

If case context is unavailable, show the specific blocked case-analysis action and a Sources-accessible recheck or existing Case-page resolution. Preserve selected source, package, exact version, preview and operation context. Search and placement status remain independently readable.

Example: Search available; case analysis waiting for verified case context. A candidate staging failure must remain visible even when semantic publication succeeds. A planned search-only request reports case staging not requested; it must not label that as failed.

## End-to-end contract and execution

1. Reconcile the exact native integration base and existing request/status contracts before implementation. Reuse CaseIdentityStore.ReadScope and the established /case-identity/scope service as the fresh authoritative sibling (E90).
2. For an explicit case-analysis request, server-check the actual approved linked matter/court pair, existing actor authority, operating mode and applicable source enrollment/selection. Do not trust browser IDs or infer a case from folders, filenames or a source's legal topic.
3. Carry one bound scope through Sources, BFF, Go admission, Temporal input, registration/enrollment relationship, candidate staging and review readback. Reject invalid case-analysis scope before case-specific write/model effects. Inventory/classification/placement and authorized general semantic processing remain independent.
4. Bind case scope to the exact source identity/content version, native package/member references, analysis generation, actor and idempotent operation. Preserve the stable source identity; case association is separate from content version and placement. Do not create a second inventory or recopy the source.
5. Revalidate applicable write authority at durable side-effect boundaries. An unavailable registry is an explicit dependency failure, never approval from cached UI labels or constants alone. Existing DEV isolation rules remain intact; no new global gate is introduced.
6. Track semantic publication, analysis extraction, candidate staging, review and any later projections separately. Read back the exact staged generation and scope before reporting case review available. Scope validation does not solve G14's separate prepared-locator DTO mismatch.

The Go engine owns durable ingest/processing/recovery through bounded Temporal Activities. Ordinary status reads stay queries. DuckDB/extraction and downstream typed fan-out retain their agreed ownership; this change adds no parallel orchestration or mandatory PG raw-content hop.

## Retry, historical compatibility and rollback

A retry resumes the recorded operation with its original verified scope and exact input generation. Changing current UI context cannot retarget a running workflow. An intentional new analysis context requires a distinct attributable request; reuse completed semantic/extraction outputs only when their exact compatibility is verified.

Repair candidate staging independently after search publication succeeds. Reconcile a lost response using the recorded request/generation and store readback; do not duplicate candidates, embeddings, source registrations or legal drafts. Source home and native membership remain unchanged.

Preserve historical requests, nullable registration metadata, old references and Temporal replay. Add compatible readers/status fields before enabling new writers; use the existing workflow versioning mechanism for changed Activity order or payloads. Historical missing case scope remains unknown until explicitly reconciled; do not backfill invented IDs, enrollment or discovery times.

Roll back new request admission while preserving readback of both contract versions and recorded operations. Retain receipts, candidates and failure history. No mass migration, source relocation, schema application or evidence promotion follows from this plan.

## Scoped change set and release boundary

CS06 owns the Sources/BFF request/status seam, Go context starter/workflow and candidate-review adapter. CS00 reconciles branch/runtime/catalog truth; CS01 agrees exact state/enrollment contract; CS05 supplies compatible original-first native lifecycle and portable source/output references. Existing shared placement and stable identity are reused.

Urgent release: verified case binding, explicit raw-search versus case-analysis intent, no late missing-scope surprise, exact review readback and Sources-accessible recovery. Later cleanup: consolidate duplicated DTOs/capability metadata and backfill existing documentation through the generated-docstring process. Do not redesign case tenancy or add global governance.

Coordinate with Case Vault v1.2.2 and placement R5: approved taxonomy and package boundaries do not assign a case. Recorded catalog candidates do not establish final placement or case enrollment. Native integration remains separately unverified. G01–G07 frozen development files are unchanged.

## Planned acceptance

AT20: omit, mismatch, corrupt or revoke case scope; simulate registry outage and DEV canonical-write denial. Case analysis has no side effects before valid admission. Scope-free preparation/placement and authorized search-only processing remain available; no invented IDs or universal PG hop.

AT74: round-trip actual registry scope through Sources, BFF, Go, Temporal, staging and review. Verify linked pair, actor, exact source/version/generation and independently read back review availability. Run the reconciled native branch rather than only a mocked DTO.

AT75: inject staging failure after extraction, successful semantic publication, lost staging response, retries and changed UI context. Show separate statuses and a Sources recovery action; resume the original operation without duplicates or source moves. Preserve historical Temporal replay and unknown scope; a different analysis request cannot silently retarget old work.

All acceptance is planned and unexecuted. G13-VERIFICATION.json pins code and coordination observations. No application suite, live provider/catalog mutation, deployment or evidence action occurred in this planning task.

## Owner acceptance — October 11, 2026

The owner accepts reuse of the existing Case registry, search separate from case analysis, and scope problems fixable from Sources. Preserve the detailed verification and retry requirements above. Exact contract fields, implementation and live release still require reconciliation and verification. This acceptance does not dispatch G13 development or expand the frozen G01–G07 batch.
