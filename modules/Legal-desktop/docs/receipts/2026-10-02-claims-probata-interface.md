# Claims, evidence gaps and the Probata interface — 2026-10-02

Owner direction: build the unfinished workdesk features, prioritize claims and
evidence before further office work, and make Probata events/entities available
without entering the same material twice. This receipt records the bounded slice.

## Shared records and separate legal work

Probata owns native entities, committed context events, source material and its
versions. Advocatio reads current descriptors from the authenticated Proffer
starter endpoint `GET /legal-context/records`. It stores only the Probata system,
kind, native ID and pinned version alongside legal work. Current source titles
and payloads are transient projections; neither current nor revision JSON stores
them. There is no second event/entity editor or replicated evidence database.

`POST /v1/claims/from-probata` opens one legal overlay per matter/system/kind/ID.
Repeated and concurrent openings resolve to the same claim ID. A newer source
version opens that existing overlay and flags the version difference. It never
overwrites a legal response or silently advances its pinned version. Legal
mutations use expected revisions and append actor-attributed history atomically.

The Claims and evidence page provides Claims, Gaps and follow-up, and Probata
records. Current Probata lists refresh every 30 seconds while visible, on focus,
and manually. An open linked claim also refreshes only its source descriptor,
preserving unsaved legal fields. Returning to a record reads the current source.
Source outages leave saved legal work accessible with a Source unavailable flag.
Request-scoped reader snapshots avoid one remote request per saved claim. Remote
source reads occur outside the SQLite writer transaction.

Entities retain the native global identity-registry scope, including current
identifier descriptors. Their version is explicitly a `view-sha256` fingerprint.
Events retain their committed candidate ID and native source-record version.
The reader selects included `candidate_context` events only where the native
event locator joins a durable preview snapshot and source version belonging to
the selected REAL/TEST matter and court case. Case and event selection share a
repeatable-read, read-only transaction. These are context records, independently
of accepted evidentiary assertion/span packages.

## Delivered claim workflow

- Save an assertion, allegation, question or theory, its originator, and response.
- Link an exact accepted package/item/assertion/version/hash/span/custody locator
  as supporting, partial, contradictory or contextual evidence.
- Save context pointers separately; they never confer factual support.
- Keep links, retirement/restoration, gaps, resolve/reopen decisions and planned
  document searches, investigation, legal research or discovery follow-ups.
- Report evidence gaps and contradictions; preserve complete revision history.
- Reject stale writes with HTTP 409 while the UI retains the unsaved input.
- Expose the same service through five workdesk MCP tools: case_claims,
  case_claim_gaps, create_case_claim, open_probata_legal_response and
  plan_claim_followup. Mutation actors come from the authenticated HTTP request.

Editing a statement retires its previous links. Exact locators must still resolve
in the currently imported accepted package; replaced/revoked references become
unavailable. Section citation resolution no longer marks every factual paragraph
supported. Numbered factual paragraphs are no longer treated as drafting guidance.

## Deployment and validation

The change belongs to the Propria monorepo, Legal-desktop and the seven explicitly
owned Proffer engine reader/composition paths. Work took place in isolated
worktrees. Shared palette and existing Next/FastAPI/Go stacks are retained.
The existing root-only Proffer service-token file is mounted read-only into
legal-api; credentials never reach the browser. Connections use the named
ovh-files tailnet host and existing starter port, not a new public listener.
Deploy the Proffer starter through Coolify before the Legal workspace.

Initial focused verification: 73 Python tests passed and the Next production
Webpack build completed. Seven upstream Go reader/composition paths passed
vendor-only focused tests for authentication, query validation, bound scope
filters, source identities/versions, fingerprints, failure and mount behavior.
The broader Python run found four existing tests that still expected superseded
Motion writer/Evidence requests labels; those expectations were updated to the
already-shipped Documents and writing/Discovery requests labels. Final regression,
current-main integration and live results are recorded below after deployment.

### Final verification and live receipt

- Code commits: `1b40e984` (workflow and reader), `53e97a1d` (Python 3.12
  annotation compatibility). Both were integrated into main and pushed.
- Full Python suite: **280 passed, 1 xpassed**, one existing asyncio deprecation
  warning. After the compatibility repair, **44 focused claims/reader tests**
  passed again. The three Go reader/runtime/composition packages passed after
  rebasing onto current main; no dependencies were installed.
- Final Next Webpack production build, TypeScript and smoke syntax checks passed.
  Webpack was used locally because the dependency junction lies outside
  Turbopack's worktree filesystem root; the normal Coolify build also passed.
- Proffer starter deployment `ip8zdoclninvmhevc5hei0mg`: finished and container
  healthy. The prior deployment `q1bwmjll030qipyxgkdy5ncv` was still marked building
  with no active build/compiler process and an idle helper. It was cancelled
  through Coolify to unblock the new deployment; no application stop was used.
- Initial Legal deployment `2nusggeeut7at9ilfubinqia` failed API startup: Python
  3.12 eagerly evaluated `list[GapReportRow]` inside the service after its `list`
  method shadowed the builtin. The future-annotations import corrects this.
- Legal retry `bqb17uyr6bqtupumlfagcoym`: finished; legal-api and office healthy,
  web/gateway/renderer running. `/claims` returned HTTP 200 with the Claims and
  evidence heading and Probata records tab. `/v1/claims`, `/v1/claim-gaps` and
  `/v1/claim-sources` returned 200 through the actual browser API adapter.
- Independent authenticated upstream reads and the live browser API adapter
  both returned **two entities and zero selected-case context events** in REAL
  mode. Both selected matter and court case exist. Zero does not establish that
  older timelines or other event families have already been integrated.
- An unauthenticated upstream reader request returned **401**.
- Direct authenticated MCP `tools/list` returned all five new claim tools;
  `tools/call case_claims` succeeded. ContextForge's own discovery cache was not
  independently refreshed or verified in this receipt.
- The installed **Python 3.12.15** runtime passed
  `web/smoke/claims-service-runtime-smoke.py`: durable claim/evidence/gap/follow-up,
  history/reopen, stale-write rejection, idempotent Probata opening, changed-source
  flag, preserved legal response and exclusion of source payloads from history.
  Retained fixture: `/tmp/advocatio-claims-runtime-qdzqm1bq` in the API container;
  **zero production records written** by the smoke.
- No interactive browser acceptance or phone viewport run was performed. The
  disposable CDP smoke is checked in for that remaining proof; HTTP/HTML/runtime
  checks above are not presented as browser interaction proof.

Live entry: https://legal.tilapia-skilift.ts.net/claims .

## Bounded remaining integrations

1. Investigation dispatch: send a planned request through Probata's governed
   job/request interface; persist the upstream ID and read status/results back.
2. Return links: show Advocatio legal overlays from the native Probata event/entity
   surfaces using the same IDs, without copying legal or source records.
3. Accepted evidence transport: retain the existing D08/D152 producer/verifier
   boundary. The current signed source-package issuer/transport is unfinished;
   this reader does not manufacture assertion acceptance or replace that gate.
4. Other native surfaces: project claims over accepted assertion IDs, timeline
   collections/membership and changes through their own existing contracts.
5. Document passages: carry explicit paragraph-to-claim evidence mappings into
   office/draft review. This slice corrects false support flags but does not
   implement all four independent review dimensions or substantive review.

Follow-up buttons currently save plans; they do not dispatch requests. Live source
updates are read-through refreshes, not a push subscription. The existing local
calendar/historic timeline remains separate until its native projection is wired.
No source migration, custody writes, server registration or database promotion
occurs in this slice. Browser interaction proof requires an existing remote CDP
browser plus the checked-in disposable-fixture smoke; no desktop browser is
launched and no synthetic claims are inserted into the owner's production case.
