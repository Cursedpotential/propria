# Discussion and handoffs versus current Advocatio

Evidence: [current-source report](inputs/current-app/REPORT.md) and [14 detailed capability records](inputs/current-app/coverage.json). These describe the dirty working tree as inspected on September 13, not a clean release or browser-tested deployment. No current-app screenshot was found or captured. The original terminal is a donor, not a picture of the present app.

| Area | What current source contains | Reconciliation/deliverable |
|---|---|---|
| Shell and density | Plain page labels, mouse links, clickable suggestions; Enter/Escape/Tab, command palette, assistant and split controls | Preserve these useful foundations during stack migration. Original mnemonic-primary terminal direction is superseded by current plain-language/mouse-and-keyboard requirement. Add shared source/detail behavior and phone projection; runtime usability remains to test. |
| Legal references | Curated authority records, propositions, pinpoints and parser/search routes | Full toolkit collection and remote-store reconciliation remain. Seed packet labels are not verified captured hashes; parser-backed 'Snapshot check' overstates implemented verification. |
| Methods | Templates and routed prompts | No complete shared versioned skill resource/execution binding found. Add human help, agent/workflow bindings and criterion-based review without duplicating methods. |
| Drafting | Entered-text drafts and template outlines | Add method/source-pinned generation and existing-document review. A template outline is not a court-approved form or substantively reviewed filing. |
| Review and revision | Whole-section approve/reject/request changes, content hashes, forks for released sections | Add immutable ordinary revisions and change-level proposals with accept/reject/edit and stale-base conflicts. Current support status is too broad. |
| Court-language translator | General editor/notes/assistant, no dedicated original/proposed translation model found | Preserve original voice and show court-facing wording proposals, independent factual/legal flags and explicit adoption. |
| Private context/strategy | Typed strategy/theory/idea/chat-extract notes and context passed to assistant | Expand to relationship/case context, decision history, risk/anticipated accusations, defenses and source-status distinctions. |
| Red team/firm | Red-team lenses/findings and configurable provider routing | Retain red-team work; recover and port actual firm's managing/senior/specialist/clerk operating design. Existing routing alone is not the requested multidisciplinary firm. |
| Analysis import | Approved factual packages and non-court-safe hypotheses represented | Add explicit versioned evidence-analysis import/follow-up, maintaining diagnostic/inference boundaries and private scope. |
| Timeline/calendar | Both screens query `/v1/docket-events`; manually rendered timeline articles and calendar grid | Current timeline is procedural docket presentation. Relationship, parenting, claim support and Timesketch curation need shared event/claim projection and retained visualization integrations. |
| Evidence links/reporting | Citation reference validation, source imports and missing-proof requests | Preserve reference validation; add atomic claim-to-span review, contradiction and partial-support states plus reproducible no-evidence reports. |
| Exhibits/releases | Exhibit records, owner-triggered Bates, approval/release paths | Retain boundaries; verify real derivative processing, office/form/redaction/metadata workflows and immutable revision persistence. Exhibit clerk uses accepted sources. |
| Agreement comparison | Two-text comparison; negotiation notes in component state | Persist analysis/alternatives and proposed clause edits; a comparison view is not tracked changes or a saved negotiation workflow. |
| Persistence | SQLAlchemy store selecting SQLite in current workspace path; JSON only optional mirrors | Old JSON persistence instructions are stale. Correct committed-delete/rebuild risk and constant revision fields before broad source/context import. Intended PostgreSQL deployment still requires proof. |
| Integrations | No current Timesketch/retained timeline library usage found in searched Advocatio manifests/source | Local fork and sibling commitments exist. Restore them to an explicit phased integration plan; dependency presence elsewhere does not prove this app uses it. |

## Most important functional correction

The existing source makes **citation-reference validity look too much like factual support**. `domain/support_map.py:69` credits paragraphs from section citation presence, while `services/citation_gate.py:25` checks reference integrity and locator presence. This is precisely why the owner requested distinct checks. Treat reference validity, actual support for an atomic claim, legal authority/currentness, structure and substantive legal judgment separately.

## Mockup/UI reconciliation

Preserve useful density: master/detail, pinned second panel, nearby assistant, cross-links, sortable tables, source inspector and fast navigation. Make the information hierarchy legible: reference versus case record versus draft versus proposed change versus accepted source. Technical IDs, provider diagnostics and job payloads belong in expandable details. Do not flatten everything into one dashboard or require command mnemonics.

Proposed common navigation: **Case overview; Timeline; Claims and evidence; Legal references; Documents; Strategy and risks; Hearing preparation; Discovery; Calendar and tasks; Digital firm**. These are proposed plain-language groups, not a replacement of every current route. The simpler phone surface presents the relevant records, references, calendar and readable source details with restricted actions.

Visual acceptance remains future work: source inspection supports behavior/structure findings, not layout quality, responsive fit, accessibility or live save guarantees. Phase F0 records actual browser fixtures; later phases test them. No mock or donor screenshot is used as current-runtime proof.
