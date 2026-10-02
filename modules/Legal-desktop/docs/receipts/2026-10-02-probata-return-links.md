# Advocatio and Probata return navigation

Byline: Codex, 2026-10-02. Continuation of the claims/shared-record slice.

## Delivered behavior

The Probata Workbench `/case` page places **Open legal response** on each native
person card. It opens Advocatio `/claims?probata_kind=entity&probata_id=<native UUID>`
in another tab. The URL carries identity only. It uses the canonical Legal host
by default; a bare HTTPS origin can be configured at Vite build time through
`VITE_ADVOCATIO_WEB_ORIGIN`. Invalid origins, malformed UUIDs and the nil UUID
produce no link. No credentials travel in the URL.

Advocatio first reads `GET /v1/claims/by-origin?kind=entity&record_id=<UUID>`.
An existing response opens directly, including its saved response when the source
reader is unavailable. If none exists, it reads that native source and displays
**Add legal response**. Only pressing that button invokes the existing idempotent
creation operation. Following or reopening a link creates no claim revision.
The lookup is authenticated and scoped to the current matter.

The landing contract supports native entity and event IDs. The mounted Probata
button in this slice covers **person records only**. The inspected Case Bible
event drawer has catalog IDs; SBV review cards carry candidate IDs before
commit. Neither is passed off as a committed native timeline identity.

Source records remain owned by Probata; saved legal responses, gaps and follow-up
plans remain Advocatio overlays keyed to the same native IDs. This is shared
identity/read-through navigation, with focus/periodic refresh already available.
It is not a push subscription or a second evidence store.

## Source and verification

- Legal service/API and landing: `api/legal_workspace/services/claims.py`,
  `api/legal_workspace/api/claim_routes.py`, `probata_record_routes.py`,
  `web/src/components/ClaimsWorkspace.tsx`, `web/src/lib/probata-link.ts`.
- Probata mounted action: `modules/workbench/web/src/components/case/case-identity-screen.tsx`
  and `src/lib/advocatio-link.ts` inside the Probata module.
- Legal tests: **47 focused Python tests passed**, covering authenticated lookup,
  no automatic creation, source outage, matter isolation, exact filtered native
  identity forwarding, idempotency and retained history. Link parsing: **2 passed**.
  Next Webpack production build and TypeScript passed. No local dependencies installed.
- Workbench: focused link tests **3 passed**; lint **0 errors, 18 existing warnings**;
  production build, Storybook, and browser-free smoke **115 passed, 4 expected skips**.
  Existing chunk-size warnings remain.
- Runtime/browser scripts: Legal `web/smoke/claims-service-runtime-smoke.py` and
  `web/smoke/probata-return-browser.mjs`; Probata
  `deploy/workbench-audit/case-page.mjs` with `CASE_REQUIRE_ADVOCATIO_LINKS=yes`.
  The latter checks each mounted person link against the card's native UUID.
  Browser scripts run in the existing remote devbox, never on the owner's desktop.

Code commits: `6c53f0a7` Legal lookup/landing; `4138a5e2` Probata person action;
`29db2e7d` mounted-link audit. Integrated and pushed in main `00785f05`.

### Live verification

- Legal Coolify deployment `z0khki8ztq2dftajlmsi1efz` finished from `ec6fa869`.
  API and office containers are healthy; web, gateway and renderer are running.
  `/claims` returns **200**. The temporary **502** during container replacement
  cleared when the new gateway started.
- Workbench deployment `e7ebqv8n30lhwrvcpty8hvdy` finished from `00785f05`;
  the container is **healthy**, and the served `/case` returns **200**.
- Actual Legal browser API reads returned REAL mode and **two native entities**.
  Filtered lookup returned exactly the requested native UUID and one record;
  origin lookup returned **200/null** because no response existed. Production
  claim count was **0 before and 0 after** these read-only checks.
- Installed **Python 3.12.15** passed the expanded service/runtime smoke,
  including origin lookup and preserved response on outage. Retained synthetic
  store: `/tmp/advocatio-claims-runtime-4hwk_y14` inside the new API container.
  No production records were written.
- The deployed Legal page passed actual Chrome interaction in the **remote
  ovh-files devbox**, with API interception supplying synthetic fixtures:
  no write on arrival, explicit add only, existing response reopened, response
  retained through source outage, no errors, and no horizontal overflow at
  **390 x 844**. Every fixture API request was intercepted; no synthetic data
  reached the production API. Proof and screenshot are retained in
  `/home/kasm-user/legal-return-20261002/out/` in the devbox.
- The actual deployed Workbench `/case?mode=REAL` browser audit used the devbox's
  scoped machine identity. Both **native person cards** had valid Legal return
  links preserving their exact UUIDs. API status **200**, no browser errors,
  no abort and **zero production writes**. Retained proof/screenshots:
  `/home/kasm-user/legal-return-20261002/workbench-out/` in the same devbox.

Live entries: https://workbench.tilapia-skilift.ts.net/case and
https://legal.tilapia-skilift.ts.net/claims . These checks prove mounted person
navigation and synthetic Legal interaction; they do not claim an implemented
investigation receiver or complete committed-event coverage.

## Investigation receiver finding and bounded next task

The inspected Probata code has no supported legal investigation receiver.
`modules/engine/runtimeapi/proffer_preview.go` exposes durable
`/reference-import/*` source/parser/review operations. `runtimeapi/router.go`
exposes parser activity and health routes. The older
`docs/schemas/platform-intake-job-contract-v1.openapi.yaml` is source intake.
None defines investigation request identity, legal correlation or result status.
The current Legal investigation contract exists on the Legal side only.

Keep existing follow-ups as saved plans until that receiver is implemented.
Do not dispatch a legal question as a parser intake, repair operation or source
promotion. The authenticated shared-record read credential establishes no write
receiver. D-008 is signed; an obsolete draft label is not a reason to hold this
work. Evidence promotion remains governed by its own contract when promotion is
actually part of a requested operation.

Recommended next slice: a native **investigation request** operation with a durable
request ID and status readback. Proposed contract, pending implementation:

1. An authenticated, scoped create operation accepts current matter/case/mode,
   actor, legal claim and follow-up IDs, native source pointers, question and
   explicit requested work. It carries an idempotency key. Scope and source IDs
   are checked against native data; a retried key returns the same request and
   changed content under that key is a conflict.
2. Persist request and initial state in Probata's canonical PostgreSQL store,
   with a reliable execution handoff. Preserve the existing worker/runtime
   architecture discovered at implementation time; parser intake is not the lane.
3. Status reads return request ID, timestamps, queued/running/completed/failed/
   cancelled state, progress and structured result pointers. Record which tool
   and source versions produced each result; credentials and raw evidence bytes
   are not duplicated into the legal overlay.
4. Advocatio explicitly sends an existing planned follow-up, persists the upstream
   ID, and reads status/results through that contract. Do not mark a plan sent
   before the upstream acknowledgement. Retries never produce a second request.
5. Results are proposed findings and linked sources. The existing proposed-change
   workflow determines which conclusions enter accepted legal work. Missing
   evidence, structure/citations and substantive review retain distinct statuses.

Acceptance: synthetic scoped request through the real receiver, duplicate retry,
outage/recovery, persisted restart/reopen, status/result readback in Advocatio,
native source-ID preservation, rejected cross-case requests and zero unrequested
production case writes. Start with request lifecycle/readback before enabling
tool execution. Parallel lanes can own receiver/store, Legal adapter and UI;
serialize shared contract/schema edits.

### Copyable bounded continuation prompt

```text
Continue the Advocatio/Probata integration from
modules/Legal-desktop/docs/receipts/2026-10-02-probata-return-links.md.
Implement one slice: a native persisted investigation-request lifecycle in
Probata, with authenticated creation, idempotency, status readback and source-ID
validation. Begin by finding the current runtime route/auth and PostgreSQL-store
siblings; verify the brief against them before code. Parser intake is not the
receiver. Start with lifecycle/readback; tool execution is a later lane.

Use bounded agents with explicit adequate cheap models and isolated worktrees
from origin/main. Assign non-overlapping ownership for the request contract/store,
runtime routes/tests, and Legal adapter/UI after the contract stabilizes. Serialize
shared schema/contract edits. Preserve concurrent work and native case/source IDs.

Prove synthetic create/read/retry/reopen/outage behavior through the actual runtime
and disposable data area; reject cross-case requests and altered payload under a
reused idempotency key. Record the upstream ID only after acknowledged creation.
No unrequested production-case writes. Use Coolify for deployment and the existing
remote devbox for browser proof; never launch a browser on the owner's desktop.

Save progress in a dated project receipt containing files, decisions, commands,
results, exact outstanding work and one next step. Commit explicit paths, integrate
and push after checks, and publish notes/receipt to Docstore with independent
readback. Never permanently delete files; preserve cleanup candidates in the owning
project's to_be_deleted directory.
```

## Remaining shared-surface work

Native committed-event return actions need the corresponding authoritative
timeline surface. Legal overlay previews in Probata are separate from this
navigation action. Accepted evidence transport, passage-to-claim mapping,
investigation dispatch and complete event families remain on the roadmap.
The existing office/template work continues independently of this bounded slice.

For any cheaper-model continuation: read this receipt and the current code first,
take an isolated `_worktrees` checkout from `origin/main`, state file ownership,
save findings and proof in a dated receipt, commit explicit paths and publish
the receipt to Docstore with independent readback. Preserve concurrent edits.
