# Surface callability and alignment matrix

> _Byline: Codex · GPT-5 · 2026-09-12._
>
> _Evidence boundary: coordinated read-only reports from the active portal, Probata naming and Family
> Court design tasks. HTTP status proves reachability or auth gating only—not an authenticated workflow,
> data authority, mutation path or end-to-end integration._

## What the shared design may call now

| Surface | Current safe label | Addressable entry | What may be claimed |
|---|---|---|---|
| Propria | Portal | `https://homepage.tilapia-skilift.ts.net/` | Live entry point |
| Propria | Project workspace | `/progress/` | Live route; current presentation is rejected and being redesigned |
| Probata General | Evidence Operations Desk | `https://workbench.tilapia-skilift.ts.net/` | Addressable deployed SPA route; authenticated workflow not proved by HTTP |
| Probata General | Intake | `/intake` | Primary navigation route; backend workflow proof remains separate |
| Probata General | Preview | `/evidence/preview` | Primary navigation route; bounded preview contract |
| Probata gated | Matter workspace | `/matter?matter_id=<uuid>` | Direct gated route; advanced evidence behavior is capability-gated and not a completed release claim |
| Probata Advanced | Modular Service Cockpit | No application route | Approved design mock and gated surface definition only |
| advocatio | Legal Workdesk | `https://legal.tilapia-skilift.ts.net/` | HTTP reachability only; no Probata launch exchange is implemented |
| Consignatio | Intake preview | `/progress/intake/` | Component preview; Xplorer and metadata backends are not connected |
| Family Court | Michigan Family Court toolbox preview | `/progress/family-court/` | Read-only preview route; live MCP runtime not verified |

Two same-day read-only probes disagreed on the unauthenticated status of the Probata routes: the
portal task received HTTP 403 while the Probata task received HTTP 200. That may reflect request or
authentication context, but it was not reconciled here. Therefore the matrix claims addressability
only and does not claim stable public access or an authenticated end-to-end workflow.

Probata also has directly addressable source routes such as `/classification-test`, `/copilot`,
`/evidence-queue`, `/knowledge`, `/records`, `/repairs`, `/runs`, `/schemas`, `/surreal` and `/tools`.
They are advanced or reconciliation routes, not permission to advertise a finished cockpit menu.
There is no current Probata `/advanced`, `/cockpit`, `/legal` or `/advocatio` route.

## Legal integration seam

The Family Court Console is a separate local React/Tauri product, not the advocatio Legal Workdesk.
Its local navigation
contract includes Case Status, Docket, Timeline, Memos, Evidence, Evals, Reference, Tools and Chat, but
those development routes are not production deep links.

The safest initial integration candidates are eight bounded read-only MCP App operations:

- `open_dashboard`
- `route_issue`
- `calculate_planning_date`
- `get_packet_plan`
- `get_checklist`
- `audit_sources`
- `search_guide`
- `build_chronology`

Their current-session host availability was not proven. Treat them as an adapter allowlist candidate,
not a callable-production claim. The richer Family Court server also has mixed read/write tools; do
not surface them wholesale. Mutations remain behind explicit legal/custody authority and review.

## Shared context, separate authority

A future server-side launch exchange may carry a versioned, audience-bound, expiring, single-use
context reference containing only the applicable:

- `matter_id` and `court_case_id`;
- immutable source/custody record IDs, hashes and receipt references;
- durable `run_id` or correlation ID;
- projection generation or `LegalSourcePackage` ID, version, manifest hash and verification state;
- requested capability, audience, issuance/expiry, nonce and revocation/staleness state.

The receiving product creates its own session and revalidates the scope. Never carry raw secrets,
browser-asserted authorization, mutable evidence copies, private storage paths or a precomputed
`safe_for_legal_use` conclusion.

Probata/PostgreSQL owns evidence, custody, review decisions, projection generations, receipts and
legal-package issuance. The advocatio Legal Workdesk owns research, theories, drafts,
review/release preparation and
typed requests for missing proof. Legal output does not automatically become evidence. Visual
alignment does not change those ownership lines.
