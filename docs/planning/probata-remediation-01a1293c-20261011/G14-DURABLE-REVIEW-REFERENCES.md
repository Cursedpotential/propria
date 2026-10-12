# G14 — Durable references for native AI review

October 11, 2026. **Core resolution accepted by the owner.** G13 is accepted: use the existing Case registry, keep search separate from case analysis, and make scope problems fixable from Sources. This document is a remediation plan, not an implementation or release authorization.

## Verified gap and severity

G14 retains its saved-register meaning: the restored candidate-review BFF rejects the native prepared locator. Severity P1. Owner: Probata Sources/Read BFF, native AI artifact/Activity owner and Go review opener. CS06 owns the wire repair with CS05 durable-native-lifecycle and CS01 shared-contract dependencies.

The native Python producer writes bounded stage bundles as file URIs (E25). The native staging Activity carries the actual prepared_ref into its verified review_source (E23). Restored BFF ReviewSource instead requires prepared_ref to start b2://; source_pin converts a validation mismatch to HTTP 502 (E24). Combining those inspected contracts rejects the native result before candidate review.

This is a confirmed cross-branch contract mismatch, not a reproduced deployed failure. Baseline B has native staging/Go opening but no inspected restored BFF file; restored R has that BFF but no inspected native staging implementation. Native N retains the producer but lacks baseline staging. New remote main at 833ace8f99563ce7a628c40b0584c2ba765b4ef3 lacks all five inspected native entrypoint/opener files. Absence is not proof of a delivered replacement. Existing placement R5 explicitly leaves native AI compatibility open.

The Go reader is also file-only, checks configured-root containment, opens prepared.original_ref and verifies exact original bytes (E92). Relabelling a file URI as a B2 URI or relaxing only the BFF regex cannot establish readback. Python's default output root and Go's default reader root differ; verify effective deployed overrides rather than assume shared storage.

Restored API tests supply a synthetic B2 prepared_ref and mock Go responses (E94). They cover useful routing/pin checks but do not prove that actual native stage output opens through the integrated stack. Existing review UI already presents quotes/native provenance and separates decisions from graph-publication status (E95); extend it rather than build a competing review application.

## Proposed user-facing result

After extraction, the owner can open the candidate in Sources/Read, view its exact original passage and context, inspect provenance, and record the existing explicit review decision. Review availability is independent of search availability, classification and any later projection.

Sources must distinguish processing, review-artifact retention, review-artifact verification and review-ready. A missing or unresolved artifact is not an empty successful candidate list. Show the specific issue plus an available recheck, verified reference-resolution or exact-generation recovery action while retaining the selected source/package/workflow context.

Mobile and legal workdesk consumers use the same versioned source/artifact references through established services. A worker's absolute disk path is not a portable public review identity. No browser is asked to open local server files or enter a storage path.

## Proposed contract and placement

Use one versioned review-source/artifact contract across the Python producer receipts, Go staging/status, BFF, TypeScript client and trusted reader. Reuse the existing Case Bible catalog plus G12's additive source/placement/output contracts. Exact field names remain a contract-reconciliation task; do not invent another inventory or global reference service.

Required information includes:

- Stable source identity, exact native member/content/provider version and unchanged permanent original home.
- Exact prepared/candidate artifact generation, kind and schema/producer version; durable catalog relationship to the source.
- Registered immutable location/version and content digest for the review artifact, plus the original digest and native JSON pointer/codepoint or verified text-span mapping.
- Actual workflow/operation, selected verified case scope and existing actor/access authority where review requires them.

Distinguish an original's digest from a prepared-artifact digest. Keep absent retained-object IDs absent; native AI context must not acquire invented evidence/custody objects. A valid URI alone does not prove identity, complete retention, integrity or authority.

Retain native sources at their verified permanent homes. Retain review companions and their exact generation under the coordinated source-local derived layout where appropriate; protect native members and producer-supplied _derived content. Eligible durable analytical records follow the agreed DerivedKnowledge envelope/type mapping. Local scratch may accelerate processing but cannot be the only readback copy after review-ready is reported.

## Trusted opening and exact provenance

The trusted server resolves the recorded artifact reference through the catalog and approved object/provider adapter. Apply existing access checks, bounds and exact source/version/generation/digest validation; do not fetch arbitrary caller-provided file or network URLs.

Extend the existing aicontextsource.ReadPrepared/ReadFile and verifyAIContextSource seams with the agreed versioned durable reader. Preserve their original-byte, native-format, quote/span and registration checks (E92/E93). The durable reader resolves embedded references too: prepared.original_ref and candidate/prepared relationships must agree. Do not simply replace a URI prefix or infer siblings from a filename.

G13's existing registry binding continues through review. List, source preview and decision all resolve the same exact source/generation. The BFF should obtain authoritative review pins from the existing actor-owned workflow/status seam rather than accept a browser-supplied replacement pin.

A new extraction generation is a different review basis. Never display new text underneath an old candidate ID/digest or transfer an old approval to changed bytes. Review decisions, graph publication, legal adoption and formal evidence remain separate facts.

## Processing and recovery

The Go engine orchestrates durable retention, verification and recovery through bounded Temporal Activities and established services. Return references rather than conversation bodies in workflow history. Query-only viewing stays an ordinary authorized read.

Verify durable artifact retention/readback before reporting review-ready. Stage or retain pending candidates according to the reconciled lifecycle, but do not call them reviewable while their required source artifacts are unavailable. Partial search publication and pending review must remain independently visible.

For interrupted retention, resume the same recorded operation with exact source/generation/member pins. Read back an uncertain commit before retrying; do not duplicate candidates, embeddings, approvals or documents. Reuse completed artifacts when their hashes and contracts match.

If an old worker-local artifact is missing, first check recorded durable copies and exact historical mappings. Recover an equivalent artifact only with verified bytes and lineage. A recomputation that changes content/producer output creates a new generation and preserves the old unresolved review state; it cannot silently impersonate old content.

## Historical compatibility, migration and rollback

Preserve existing v1 payloads, local refs, candidate bodies/digests, decisions, receipts and Temporal history. Add compatible readers and explicit capability/version handling before new writers. An old local reference may resolve through a verified historical alias to durable identical content without rewriting its stored identity.

Do not broaden file access for untrusted callers. A legacy local reader remains bounded to the existing configured root and recorded operation where applicable. New durable references must open on an authorized different worker/service without the original worker's filesystem.

Any future historical repair needs a dry-run inventory of exact old refs, source/generation maps, matching retained bytes, unresolved items and conflicts. Migration is separately authorized; this plan moves no B2 sources and repairs no historical rows.

Rollback disables new-contract admission while retaining both-version readback and all recorded content/history. Do not delete durable artifacts or revert verified sources to temporary homes.

## Scoped implementation and release order

1. CS00 reconciles the exact native/BFF integration base, actual mounts/roots, running reader versions and existing catalog capabilities. Keep source, test, deployment and live proof separate.
2. CS01/CS05 agree and implement the additive review artifact reference with G12 source/output contracts and approved G13 scope; reuse the existing opener/verifier and catalog.
3. CS06 updates producer/Go/status/BFF/client together, preserving validation and explicit unsupported-contract errors. Add Sources-accessible resolution and exact original-context viewing.
4. CS10 validates the entire real contract path on minimal synthetic data, then a separately authorized bounded live slice. A mocked B2 fixture alone is insufficient.

Urgent release: actual native result is reviewable through the integrated API, exact source context remains openable after worker replacement, and missing artifacts have truthful status/recovery. Later cleanup can consolidate duplicated DTO declarations and backfill old documentation; new units require the standing docstring contract.

This does not expand the frozen G01–G07 development batch. Case Vault v1.2.2 package boundaries/taxonomy and permanent-home rules remain intact. No new global governance, provider mutation, implementation or deployment is authorized here.

## Planned acceptance

AT21: use actual reconciled native Stage output through BFF, trusted reader, candidate list/source preview and explicit decision; verify exact original/version/generation/hash/native span without a file-versus-B2 validation failure.

AT76: reopen review through an authorized different worker/service with no dependency on the old worker-local scratch. Resolve durable prepared/original/candidate references and exact surrounding passage; source home/native membership and object-ID absence remain intact.

AT77: inject missing/partial/tampered or wrong-source/wrong-generation artifacts, root/config divergence, unsupported versions and unauthorized caller paths. Report actionable Sources blockers without empty success, arbitrary fetch or new approval; retry the exact operation without duplicates.

AT78: preserve a historical v1 local reference/candidate digest and existing decision; verify compatible alias readback, versioned new writer and rollback. Changed recomputation creates a new generation; old approvals/replay remain bound to their original content. Preserve all historical refs and source bytes.

All tests are planned, not executed. G14-VERIFICATION.json records source pins, branch absence checks and verification limits.

## Owner acceptance — 2026-10-11

Direct owner response: “OK Approved next.” The core durable review-reference resolution is accepted. Preserve exact source/version/generation/hash/native context, historical references and decisions, with artifact resolution accessible from Sources.

Exact schema and adapter details remain to reconcile before implementation. This acceptance does not expand the frozen G01–G07 development batch or authorize deployment, data mutation or evidence promotion. G15 is the next discussion.
