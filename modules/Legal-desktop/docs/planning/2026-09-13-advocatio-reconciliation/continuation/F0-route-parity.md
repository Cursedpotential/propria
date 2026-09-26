# F0 route and parity audit

Date: 2026-09-20
Repository: `E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`
Scope: read-only audit of the current Next web surface, its auth/proxy/API/chat boundaries, and the shortest common React/TypeScript/Vite/TanStack parity slice. The only new artifact in this task is this note. The existing dirty file `docs/planning/2026-09-13-advocatio-reconciliation/continuation/MOCKUP-IDENTITY-2026-09-20.md` was preserved.

## Evidence and settled target

The current application is a Next App Router client in `web/`, with `next ^16.3.1`, React `19.2.3`, AI SDK packages, and no declared TanStack, Storybook, or Glide packages (`web/package.json`). The reconciliation decision is one shared React/TypeScript client using the recorded Vite/TanStack Router/Query/Storybook/Glide baseline, with Tauri as the desktop shell. This is the migration planning baseline; the manifest does not establish approval of every package version.

The shared baseline inspected at `E:/AI_Workspace/Projects/Propria/modules/Probata/probata/modules/workbench/web/package.json` uses React `19.2.3`, Vite `8.2.2`, TanStack Router `^1.170.32`, Storybook `^10.5.10`, and the retained timeline packages. It declares `@ai-sdk/react` and `ai`, but does not declare `@tanstack/react-query`; therefore this audit does not claim Query is already present in that manifest. The baseline build is `tsc --noEmit && vite build`; its smoke command builds before running browser-facing smoke tests.

## Current route/source map

Every page below is a current `web/src/app/**/page.tsx` route. Server pages fetch the Python API through `legalApiBase()`; client forms use the same origin adapter and then refresh with `next/navigation`. The API paths are the source contract to preserve during the port.

| Route | Current role | API/source boundary | Port disposition |
|---|---|---|---|
| `/` | Matter home and next-surface links | `GET /v1/matter`, `GET /v1/audit?limit=8` | First parity fixture; preserve matter/source/version links. |
| `/case-search` | Court/source search | `SourceSearch`; `GET /v1/sources/courtlistener/search` | First reference workflow entry. |
| `/laws` | Authorities list | `GET /v1/authorities` | Common reference module. |
| `/questions` | Issue tree | `GET /v1/issues` with matter fallback; issue mutation | Common reference/detail module. |
| `/custody-factors` | Factor analysis and notes | `GET /v1/factors`, `GET /v1/factors/{letter}/analysis`, factor note mutation | Preserve both-parent and missing-proof states. |
| `/evidence` | Exhibit candidates and Bates operation | `GET /v1/exhibits`, exhibit/Bates mutations | Backend-owned custody and release semantics. |
| `/documents` | PDF/document pane | client `PdfPane`; document endpoints in component | Preserve derivative/source identity; no browser authority. |
| `/timeline` | Docket event timeline | `GET/POST/DELETE /v1/docket-events` | Shared event type; renderer choice stays view-specific. |
| `/calendar` | Calendar projection of docket events | `GET /v1/docket-events`, docket mutations | Same event source as timeline. |
| `/drafts` | Draft sections | `GET /v1/drafts`, `PATCH /v1/drafts/{section_id}` | F4 proposal/revision slice. |
| `/review` | Owner review | `GET /v1/reviews`, `GET /v1/drafts`, review mutation | Keep approval authority in Python/API state. |
| `/agreements` | Contract workbench | `GET /v1/drafts` | Proposal/work-product projection. |
| `/challenge-draft` | Red-team review | `GET /v1/redteam` | Private review projection. |
| `/final-copy` | Release candidates | `GET /v1/releases`, `GET /v1/drafts`, release mutation | Deterministic release manifest remains backend-owned. |
| `/filing-checklist` | Filing readiness checklist | `GET /v1/filing-readiness` | Checklist only; no filing action. |
| `/assistant` | Chat UI, context and unsaved fields | Intended chat BFF; currently posts `/api/assistant` | Fix route contract before parity acceptance. |
| `/assistant-log` | Agent run log and routing editor | `GET /v1/agent-runs`, `GET/PUT /v1/routing` | Routing JSON remains backend-owned. |
| `/private-notes` | Strategy/private notes | `GET /v1/strategy` and mutation | Preserve private lane and simple-surface exclusion. |
| `/open-questions` | Research questions | `GET/POST /v1/research` | Research contract, not a generic chat store. |
| `/missing-evidence` | Investigation requests | `GET/POST /v1/investigations` | Emits evidence-platform request; no evidence ingestion. |
| `/evidence-requests` | Discovery requests | `GET/POST /v1/discovery` | Backend persistence and owner date boundary. |
| `/external-sources` | Provider/source terms | `GET /v1/sources`, `GET /v1/providers` | Provider policy remains API-owned. |
| `/confidentiality-check` | Privilege scan/provider grid | `GET /v1/drafts`, `GET /v1/providers`, `POST /v1/privilege:scan` | Preserve confidential-mode and non-legal-conclusion labels. |
| `/templates` | Template instantiation | `GET /v1/templates`, `POST /v1/templates:instantiate` | Backend creates private draft. |
| `/playbooks` | Playbook list/run | `GET /v1/automations/playbooks`, run mutation | Structural automation labels only. |
| `/scheduled-jobs` | Job and playbook status | `GET /v1/automations/analysis-queue`, playbooks | APScheduler state stays Python-owned. |
| `/analysis-queue` | Analysis queue | `GET /v1/automations/analysis-queue` | Read projection; no hidden client queue. |
| `/tasks` | Owner todos | `GET/POST/PATCH /v1/todos` | Todo state stays persisted in API. |
| `/notices` | Inbound notices/events | `GET /v1/triggers` | Event/audit source remains backend-owned. |
| `/activity-log` | Audit log | `GET /v1/audit` | Immutable audit projection. |
| `/citation-check` | Citation parser | `POST /v1/citations:parse` | API validates/parses; UI displays result. |

The two Next route handlers are `/api/legal/[...path]` and `/api/chat`. `/api/legal` is a same-origin all-method adapter for the private legal API. `/api/chat` assembles surface context and live unsaved fields, loads `/v1/routing`, and calls either the configured agent run path or confidential gateway path. There is no separate `/api/assistant` handler in the current tree.

## Next-specific dependencies and auth findings

1. `web/src/proxy.ts` is a public-ingress guard. When `LEGAL_PUBLIC_HOST` matches the request host, it requires `x-authentik-jwt`, calls `/v1/auth/whoami` with that bearer, and blocks with 401/503 before the page or API route runs. Tailnet hosts bypass this proxy guard.
2. `web/src/app/api/legal/[...path]/route.ts` repeats the public-host check and requires the same injected header. With a header it forwards `Authorization: Bearer ...`; without it, it signs the request using `LEGAL_BFF_SIGNING_SECRET`, including timestamp, nonce, method, target and body hash. The Python API validates the OIDC bearer or the short-lived BFF signature. A Vite static client cannot safely reproduce the BFF signing secret in browser code.
3. The current browser adapter is same-origin (`/api/legal`), so a Vite port needs one of: a separately deployed same-origin gateway/edge function that owns this forwarding and signing behavior, or a server-side adapter retained beside the Vite client. Direct browser access to `LEGAL_API_INTERNAL_URL` would expose the private topology and would remove the current auth boundary.
4. The current page shell and forms use `usePathname`, `useRouter`, and `useSearchParams` from `next/navigation`; the assistant additionally uses `sessionStorage` and same-origin `postMessage` for live context. These become explicit router and browser-state contracts in the common client.
5. `next.config.ts` sets standalone output and a Turbopack root. The port must replace this deployment behavior with the shared Vite build and an independently documented adapter deployment; no equivalent Vite config was invented here.
6. `@ai-sdk/openai`, `@ai-sdk/react`, and `ai` are declared, but no current source import uses them and `/api/chat` returns ordinary JSON after waiting for the Python response. No SSE, `streamText`, `ReadableStream`, or AI SDK stream response was found. “Streaming parity” therefore means first preserving the current request/response and confidential-routing behavior, then adding a deliberately specified stream protocol if the backend exposes one.
7. Concrete current defect: `web/src/app/assistant/page.tsx:64` posts to `/api/assistant`, which is absent. The implemented handler is `web/src/app/api/chat/route.ts`. The assistant page therefore cannot be used as a passing parity reference until this endpoint mismatch is resolved and checked.

## Shortest common parity slice

Use one source-linked reference/detail path as F0/F1 proof:

`/case-search` → source search result → source/version detail projection → return to `/` or `/questions` with the same identity.

The common client should implement only the shell, route table, typed API client, loading/denied/stale/missing-source/partial-support fixtures, and one source inspector. It should call the existing legal API through the replacement adapter boundary and carry the source ID plus exact version/provenance fields through navigation. It should not copy legal rules, evidence bytes, review authority, agent routing, or database state into TypeScript. The first implementation slice can use TanStack Router and the shared React/Vite build contract; Query adoption requires an explicit package decision because it is absent from the inspected baseline manifest.

Acceptance for this slice:

- `/case-search` loads and displays a source hit through the common adapter.
- Selecting a hit opens the same source identity and exact version in the inspector fixture; missing or stale support is visibly distinct.
- An unauthorized/denied adapter response is rendered as a distinct state and does not fall back to a browser-held token or direct private API URL.
- The existing Next route defect is recorded as fixed or explicitly blocked before claiming assistant parity.
- The Python/FastAPI legal API remains the owner of mutations, source/version semantics, confidentiality, review, release, and evidence-platform requests.
- The common client builds with the shared baseline command and has Storybook fixtures for the required F0/F1 states. Browser interaction proof is required; a component snapshot alone is insufficient.

This is a route and dependency disposition, not evidence of a completed migration, deployment, live auth validation, or production parity.

Coordinator follow-through: commit `a233171` corrects the assistant form to call
`/api/chat` and render network/non-JSON failures. Production build and isolated
browser success/error fixtures passed after the change. The SSR identity and
shared-client adapter work above remains part of the migration boundary.
