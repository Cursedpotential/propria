# Advocatio to Probata investigation requests — 2026-10-02

## Delivered behavior

An existing open evidence-investigation follow-up can be explicitly sent from
Advocatio to Probata. Advocatio saves its request body, native case scope, actual
authenticated sender and idempotency key before networking. Probata persists the
request in PostgreSQL and returns its stable request ID. Advocatio records that
acknowledgement and can refresh the request's status and typed findings.

Opening a claim, listing sources or reopening a page never dispatches a request.
A timeout leaves a saved prepared request with **Retry send**. The retry uses the
original frozen question, sources, scope, actor and key. An acknowledged request
offers **Refresh request status**, rather than another send action. A read outage
retains the upstream ID and previous status/results. Concurrent local edits are
preserved when an acknowledgement is merged back into the latest claim revision.

The initial remote state is **Received by Probata / Waiting for execution**.
Receipt does not close the local follow-up plan. Local **Cancel plan** does not
cancel an upstream request. Closing/cancelling a prepared request is prevented
until its uncertain admission has been resolved. RFAs/RFPs remain discovery;
this action sends an internal investigation request.

## Ownership and contracts

- Native case/source records remain in Probata. Legal matter/claim/follow-up IDs
  are opaque correlation identifiers, never substitutes for native case IDs.
- Shared read-through now carries the actual native matter and court-case IDs.
  A source beyond the bounded first page is resolved by its exact native ID.
  Changed or unavailable pinned source versions fail before request preparation.
- Probata validates the current selected mode/case and each native source on new
  admission. Entity fingerprints use the existing shared identity catalog; an
  entity reference does not establish participation in a particular case. Event
  references use the existing case-scoped native event projection/version.
- A successful replay is resolved before checking whether a referenced source
  has subsequently changed. Both actor/key replay and the logical correlation
  tuple deduplicate. Alternate keys/actors become durable aliases; a changed
  payload or reused alias for another intent is rejected.
- Legal's saved preparation and acknowledgement live in the existing claim
  revision/history storage. The native request and execution lifecycle belongs
  to Probata's `ops.legal_investigation_request` relation.

HTTP receiver:

```text
POST /legal-context/investigations
GET  /legal-context/investigations/{request_id}?mode=...&matter_id=...&court_case_id=...
```

Create accepts mode, native matter/court-case IDs, Legal correlation IDs, a
question and native source references. It requires the existing tailnet/service
authentication plus actor headers and a UUID Idempotency-Key. Reads require
authentication and the exact current/stored case scope. Database errors are
sanitized. Sources are bounded to 30; questions to 5,000 characters; actors to
200 characters. Results are typed summaries, native source references and
method/run strings, bounded to 100 entries and 262,144 serialized bytes. No
document bytes or arbitrary evidence payloads enter this contract.

Legal actions:

```text
POST /v1/claims/{claim_id}/followups/{followup_id}/dispatch
POST /v1/claims/{claim_id}/followups/{followup_id}/refresh-status
body: {"expected_revision": <current claim revision>}
```

These are authenticated planning actions. The actual principal is persisted as
`authentik:<subject>`, `signed-bff:legal-web-bff` or `tailnet:tailnet-device`.
Transport identities are not represented as verified humans. MCP service,
office-session and test-bypass principals cannot dispatch. The stricter
substantive human-review gate is unchanged. This distinction was checked against
the real portal's signed BFF flow rather than assuming it supplies an Authentik
human identity.

## Implementation and schema admission

Code is integrated and pushed in main at `c5906393`.

- Legal: `contracts/claims.py`, `services/claims.py`,
  `services/probata_investigations.py`, `services/probata_records.py`,
  `api/claim_routes.py`, the planning gate in `api/auth.py`, and focused tests.
- UI: `web/src/components/ClaimsWorkspace.tsx`; repeatable remote-browser proof
  in `web/smoke/investigation-dispatch-browser.mjs`.
- Probata engine: `investigation/`, `postgres/investigation_request_store*`,
  `runtimeapi/investigation_api*`, and starter `legal_context_composition*`.
- Canonical final relation definition is present verbatim in
  `sql/bootstrap/schema_snapshot_20260907.sql` and in the named CREATE-only
  `sql/bootstrap/legal_investigation_request.sql`.

This admission created the previously absent relation in its final form. It
did not alter an existing relation, apply numbered migrations or rebuild/drop
the populated production database. The canonical snapshot and named definition
were compared before admission. Production readback found zero requests. The
actual deployed connection uses `platform_runtime`; the initial live read caught
its missing new-table grant. The canonical definition and live grants now give
SELECT/INSERT/UPDATE to both `platform_runtime` and `platform_app`. The native case,
source and evidence tables were retained.

The isolated proof database `investigation_test_advocatio_20261002` was created
on ovh-files with the canonical schema and TEST case seed. Synthetic records,
test binary and synthetic token fixture are retained for independent readback.
No production-case investigation was submitted by these proofs.

## Verification

- Integrated Legal focused tests: **90 passed**. They cover frozen timeout retry,
  stable acknowledgement, concurrent edits without a held network-time SQL lock,
  response identity/scope/body validation, source lookup beyond the first page,
  planning authentication and unchanged substantive-review eligibility.
- Next webpack production build and TypeScript checks passed; browser-script
  syntax check passed. No local desktop browser was launched.
- Go targeted checks passed across investigation, postgres, runtimeapi and
  starter; full investigation/runtimeapi/starter tests and four-package vet
  passed. Full postgres testing has an existing fixture path mismatch in
  `handler_selection_store_test.go`: it uses `../../../tests/fixtures` from
  `probata/modules/engine`, while tracked fixtures are under `probata/tests`.
  Hydrating the tracked fixture confirmed this is not a missing sparse file.
- Linux crosscompiled harness ran against actual PostgreSQL 18.1 through the
  actual HTTP handler/store: POST 200, identical replay 200, scoped GET 200,
  wrong scope 409, malformed JSON 422 and changed payload/key 409. Eight
  concurrent retries retained one logical receipt, request ID
  `9a4295bd-45d8-466d-bc48-f943560806f5`, initially `received` with no results.
- A second isolated run seeded a clearly synthetic registry person and passed
  native entity fingerprint admission and stale-fingerprint rejection. Its
  retained logical receipt is `ef385445-2bed-4cbd-b4e8-b27ae5243ba4`; a separate
  sourced receipt is also retained. All runs used the isolated TEST database.
- The complete HTTP/PostgreSQL harness then passed using the actual deployed
  database role, `platform_runtime`, rather than the administrator. Its retained
  logical receipt is `8052d97e-4fd0-4747-9a03-318a72daf6c3`, plus its separate
  sourced receipt. Eight retries still produced one logical request. All five
  retained rows across these repeated proofs are synthetic and isolated.
- The deployed Legal page passed the remote Chrome script at desktop and
  390×844 phone dimensions. Fixture API traffic was intercepted before reaching
  production: no dispatch on opening, prepared timeout state survives reopening,
  retry reaches acknowledgement, acknowledged requests are not resent, refresh
  outage preserves ID/status, typed findings render and the local plan stays
  open. A further phone status refresh also succeeded. Zero JavaScript errors
  and no horizontal phone overflow were observed. The script waits for enabled
  controls before clicking, avoiding a race with the preceding list refresh.
- Live portal checks returned claims page 200 and native read-through 200 with
  native matter/court-case scope. An explicitly verified nonexistent claim
  returned 404 for both GET and dispatch through the actual BFF, establishing
  that the planning gate reaches the missing-claim check rather than denying
  the portal session. No claim or investigation was created by this probe.

## Deployment and completion evidence

Both applications were submitted through Coolify after main `c5906393`:

- Proffer starter `r1084s1lsm80fsv4ol9ocij0`, deployment
  `0v47mnlb22ucpmbfl4unmjja` initially failed when Coolify lost its build-helper
  exec instance. Retry `z4d9ni8kjuywbjg2mfcnhidt` finished from main `cbeaf54d`,
  which includes `c5906393` plus concurrent unrelated work. The changed engine
  files were existing source-lifecycle heartbeat work, outside this receiver.
- Legal workspace `gvghzivfmctev8dloetfssnj`, deployment
  `7qaegqbrwokefhqkjwidpz53`.

The connector was unavailable; the documented direct Coolify API fallback was
used with the existing credential file. No direct Docker lifecycle command was
used. Legal deployment finished; its API and office containers were healthy,
and its claims page and browser workflow passed. Receiver deployment finished
and container `proffer-starter-r1084s1lsm80fsv4ol9ocij0-130302132905` was healthy.
Authenticated live reads returned native reader 200, unknown request 404 and
wrong case 409; unauthenticated request read returned 401. Production relation
readback remained zero. The live grant correction requires no container rebuild.

Remote browser artifacts are retained in the existing ovh-files devbox at
`/tmp/advocatio-investigation-phone-proof-20261002/` (proof.json, phone screenshot and
profile under to_be_deleted). The committed browser script can reproduce the
fixture proof using the existing Node/Chrome; it installs nothing. Fixture UI
proof and actual PostgreSQL/HTTP proof are separate and explicitly identified.

## Saved handoff and documentation

The delivery checklist, this receipt and
`docs/receipts/proofs/2026-10-02-investigation-dispatch.json` carry the bounded
results. Publish these to Docstore and independently retrieve/compare their
bodies; do not equate publication with semantic indexing. The checklist was
published as `document:1xkzqu2h61kqzyv4gu8l`, superseding only its prior exact
checklist record, with independent normalized body equality confirmed.

## One bounded next task

Connect this request lifecycle to one existing governed Probata investigation
execution path. First identify its current sibling runner, permission checks,
tool-run provenance and source/promotion contract; parser intake is not that
runner. Use the internal compare-and-set lifecycle seam to move received to
running and then completed/failed with bounded typed findings. Keep execution
an explicit action, preserve native sources/run provenance and return proposals
for review. Never infer accepted evidence or a supported legal statement merely
because a tool returned text.

Copyable continuation prompt:

```text
Continue from modules/Legal-desktop/docs/receipts/2026-10-02-investigation-dispatch.md.
Implement one governed execution path for a saved Probata investigation request.
Before writing code, map and confirm the actual current runner/auth/source siblings;
report contradictions before building. Do not reuse parser intake as investigation.
Use bounded agents with explicit adequate cheap models and isolated worktrees from
origin/main. Assign separate ownership for runner/store lifecycle, Legal results
interaction and proof; serialize shared contracts and routers. Preserve others' work.

Execution must be explicit, scoped and authorized, with stable request/run identity,
native source versions, typed results, failure/retry handling and no duplicate runs.
Results remain proposals until the existing review/promotion workflow accepts them.
Prove received->running->completed/failed and reopen/readback against the actual
runner in an isolated data area; no unrequested production-case execution or writes.
Use Coolify for lifecycle/deployment and the existing remote devbox for browser proof.
Never launch a browser on the owner's desktop or permanently delete retained files.

Save progress in a dated project receipt: files, decisions, commands, actual results,
remaining gaps and one next step. Commit explicit paths, merge/push after checks,
and publish all new notes/receipts to Docstore with independent body readback.
```

Other open work remains accepted source-package transport/promotion, passage-to-
claim mappings, authoritative committed-event return actions, overlay previews,
full timeline families, office workflow proof and durable proposed edits. This
delivery closes request admission/status readback, not those independent areas.
