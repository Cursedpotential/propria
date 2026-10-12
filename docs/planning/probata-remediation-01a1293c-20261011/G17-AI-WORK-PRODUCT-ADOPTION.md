# G17 — AI-created works and explicit legal adoption

October 11, 2026. Core design accepted by the owner: separate working material, independently saved strategies and adopted drafts, with shared mobile/legal-workdesk data. Exact schemas and adapter implementation remain to reconcile. G12 permanent-home and historical-contract preservation is now accepted. No implementation, B2 mutation, schema installation, adoption, approval or release occurs in this planning chat.

## Register identity and verified gap

This topic is G17 in the saved 36-gap register. G13 remains the separate AI BFF/candidate-staging scope gap. Preserve IDs across this plan and the existing development handoff.

The native AI extractor retains assistant-created works with native role/time, source references, JSON pointers and Unicode-codepoint spans. The inspected newer native branch retains this extraction (E29/E77). Its local created_works file is an extracted artifact, not an adoption decision or proof of durable placement.

A separate historical Go placement workflow targets KnowledgeBase/ai-chats/_derived/misc (E30). Legal toolkit placement and library-sync admission independently enforce KnowledgeBase/legal (E31/E32). These paths require coordinated compatibility work; their names cannot establish legal status.

Current main already implements private legal document imports and immutable byte revisions with an audit outbox (E78/E79). It also has distinct legal work-product review/release states, substantive human review authentication, and exact-content approval checks (E80–E82). Ordinary document import authentication is weaker than substantive human approval and uses the selected workspace matter (E83).

The inspected native extractor, legal document DTOs/import path and domain review surfaces do not establish a complete native-AI-generation-to-legal-document adoption bridge. A scoped code search found no native_json_pointer/source_generation/AIWorkproduct adoption mapping in current legal_workspace. This is a bounded inspection, not proof that every repository or deployed surface lacks an adapter.

Remote main and the native worker branch were rechecked at 9399c24824e3e0d8d83643f8ae69a41a9f019d56 and 9fe9f8b31af94242dd04aff8d745e04a5b169039. Branches, isolated services and deployed application behavior remain distinct; live adoption is unverified.

## Owner amendments: working material, retained strategy and shared mobile views

The owner requires adopted drafts to open/edit in the legal workdesk, the underlying potential strategy to persist even when a draft is unused, and mobile Family Court toolkit views over the same records. The owner accepts a separate working-material concept; its exact schema remains to reconcile. See G17-WORKING-MATERIAL-STRATEGY-MOBILE.md for source-linked strategy retention, shared-ID/revision behavior and AT71–AT73.

## Proposed distinction

| Object or fact | Meaning | What does not follow automatically |
|---|---|---|
| Native conversation/package | Original retained AI interaction and native attachments | Legal adoption, factual truth or evidence eligibility |
| AI-created work | Exact extracted output, such as a motion draft, chronology, email or strategy note | Adoption, owner approval, filing or independent corroboration |
| Working material | Tentative material for editing, comparison or combination; no commitment to use | Legal adoption or substantive approval |
| Retained strategy/theory note | Underlying potential direction saved independently of draft use | Factual truth or approved legal use |
| Adopted legal working draft | Explicit owner choice to use selected material for a legal purpose, with its own document identity and revision | Substantive approval, release or evidence promotion |
| Exact-version review/approval | Existing applicable review decision bound to identified content and scope | Approval of future edits or unrelated facts |
| Release/filing/service facts | Separately recorded authorized actions and readback | Proof inferred from a draft title, folder or export |

Application labels do not determine privilege or other legal conclusions. Preserve the existing legal review/release distinctions; do not introduce a blanket review gate for every searchable AI output or private note.

## Permanent homes and storage

Retain the native source at its verified final home. Keep its native package membership unchanged. Source-local extracted AI work uses a tracked, versioned non-native companion area under the coordinated package rules; protect producer-supplied _derived members.

An adoption creates a new linked legal working object or working revision. It does not move the chat, overwrite its AI output, reclassify every source member, or turn generated text into an original acquired document. Later edits produce a legal revision history; the exact AI text remains available for comparison.

Use the existing Case Bible catalog for source identity and artifact relationships. Reuse current legal document/review services for application work; do not create another source inventory or assume every record must be ingested through PostgreSQL. Reconcile office-document revisions and section/work-product review models instead of inventing a third competing document model.

Current Case Vault v1.2.2 classifies independently authored/received private legal drafts under documents. A retained/exported legal document follows its approved document/storage adapter and native/derived status. This proposal creates no new Legal root, adopts no fixed legacy KnowledgeBase/legal location as canon, and does not authorize moving current files. Exact future artifact-home and existing legal-root compatibility remain CS01/CS09 design reconciliation.

Portable adoption/provenance receipts and eligible durable analytical records use the existing DerivedKnowledge envelope and an approved typed mapping. Preserve original AI text separately from interpretations. Raw AI semantic retrieval remains available through its authorized Weaviate route without a universal PostgreSQL content hop.

## Sources action: Adopt into legal work

The proposal is an action available beside the source-linked AI work preview, with a linked legal editor/read view. Show surrounding conversation, authorship, native time, exact source version and generation before selection.

1. Select the whole work, an exact excerpt, or identified passages from multiple works.
2. Choose the current legal matter/purpose and an existing draft or new working document; server-check actual authorized scope and target revision.
3. Preview selected text and any formatting/assembly transformation, retaining exact original excerpt mappings.
4. Record the explicit owner adoption decision and materialize/read back the linked legal revision.
5. Open the legal draft with provenance, original-versus-edited comparison and independent review/release status.

Agents may prepare extraction and an adoption proposal. An agent/service credential, detected legal topic, semantic index success or ordinary document upload cannot impersonate the owner's adoption decision or substantive approval. Reuse existing verified human-review identity where applicable rather than inventing new global roles.

Keep adoption and approval separate. Existing section approval requires package/citation validation on that route; do not apply those release requirements merely to retaining an unreviewed strategy note or starting a private working draft. Fact verification, chunk approval, analytical graph publication and evidence promotion remain separate.

## Proposed adoption receipt and version relationships

Reconcile exact names/schema in CS01/CS05/CS09. The required information is:

- Stable adoption operation/decision revision and explicit actor/time/purpose.
- Exact source-unit/content/member version, AI-work artifact and derived generation.
- Native provider/thread/message pointers and selected Unicode-codepoint or other verified native span map.
- Exact extracted text digest and selection; full-work versus excerpt versus composite designation.
- Any assembly/rendering/template transformation and its producer/configuration versions.
- Selected target matter and legal document/revision/content digest, preserving both AI and human authorship history.
- Independent request/materialization/catalog-link/readback/review/publication status and supersession links.

An adoption from several works preserves each contributor rather than assigning a fictional single source. The legal document ID is distinct from the originating source ID. A new AI generation or edit never retroactively changes an old adoption receipt or carries an approval onto different bytes.

Do not treat an AI statement as factual support for its underlying case assertion. The legal drafting/review route must resolve any claimed factual/citation support to its actual applicable original records; merely preserving an AI quote does not verify the quote's claims.

## Go/Temporal execution, recovery and rollback

The Go engine orchestrates the adoption materialization/processing work through bounded Temporal Activities and existing legal services. The owner decision enters as an attributable exact-version event; ordinary UI viewing remains a query. Durable waiting does not hold a worker.

Bind an idempotent request to the source selection, target/purpose, exact input generation and operation. The decision, target revision and required audit/provenance link must agree before the UI reports adoption complete. Where stores cannot share a transaction, use the existing durable outbox/receipt boundary and reconcile each step explicitly.

If a response is lost after a legal revision commit, read back the recorded target and same request before retrying. A pending catalog link, conversion failure or semantic publication failure stays independently visible; it must not create another legal document or rewrite/move the original. Persist proposed versus completed adoption truthfully.

Editing creates the next legal revision, with conflict checks. Exact-content approval cannot silently apply to changed text. Withdraw/supersede an adoption or legal use by appending a decision and updating applicable projections; preserve original text, past working versions and recorded uses. Never delete history as rollback.

## Scoped implementation proposal and dependencies

CS01 defines the source-to-adoption receipt, existing typed mapping and explicit state meanings. CS02/CS03 provide stable identity and verified permanent placement. CS05 and CS07 provide portable AI work generations and exact source mappings. CS09 owns the linked legal-document/review adapter and legal-root compatibility; CS10 validates integrated readback.

Extend the existing native extractor/output contract, Sources/Activity/Read preview, legal work_document routes/service, legal work_product domain and review integration only where necessary. Keep existing models, APIs and history compatible. Do not dispatch this into the frozen G01–G07 development batch.

Urgent scope: prevent implicit adoption, retain exact source-to-legal lineage, expose the explicit adoption action and truthful independent states. Later convenience features can include advanced templates/composite editing, without weakening provenance or exact-version review.

## Planned acceptance

AT68: exact whole/excerpt/composite AI adoption; original home/native membership/AI text unchanged; legal revisions and original-versus-edited viewer resolve exact contributors.

AT69: distinguish viewing/extraction/import/adoption/approval/release; reject wrong actor, scope, stale selection and target conflict; retain private drafting without treating it as factual approval or evidence.

AT70: inject response loss after legal commit, pending catalog links, conversion/projection failure and concurrent edit; retry without duplicate documents, preserve durable history, and withdraw/rollback without deleting or moving sources.

Existing AT15, AT36, AT40, AT62 and AT66 remain relevant. These tests are planned and have not run. Future affected tests include test_work_documents.py, test_work_product_versions.py, test_review_release.py and test_auth.py; no implementation or deployment proof is claimed.

## Cross-surface amendment

G17-WORKING-MATERIAL-STRATEGY-MOBILE.md records direct owner requirements and remaining design choices. Existing StrategyNote/StrategyKind are the implementation sibling; exact source-generation grounding and shared mobile/workdesk adapters remain to verify. Adoption must read back as an openable/editable legal workdesk document. Strategy retention and shared mobile/desktop IDs/revisions are required; working material is an accepted interim use concept; exact schema remains to reconcile. AT71–AT73 are planned and do not expand the frozen development batch.

## Core acceptance — October 11, 2026

The owner accepted keeping working material and saved strategies separate from adopted drafts, using the same logical data on mobile and the legal workdesk. Preserve the explicit adoption, independent strategy retention and source-linked history described here. This is design acceptance, not implementation or approval of every field, adapter or rollout detail. G13 remains the separate scope gap.
