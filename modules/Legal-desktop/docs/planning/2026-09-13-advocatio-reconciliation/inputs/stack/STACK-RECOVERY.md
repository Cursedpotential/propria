# Propria stack decision recovery

Byline: Codex delegated read-only recovery, 2026-09-13. New planning input only; no product, configuration, package, service, or existing document edits. No installation or model invocation.

## Finding

The owner selected actual frontend unification, not merely shared colors. TanStack, Storybook, and Glide are settled; desktop targets add Tauri to the common application foundation. The recovered migration assessment recommends **React + TypeScript + Vite + TanStack Router/Query + Storybook + Glide + Propria design tokens**. Advocatio's existing Next application is a migration source, not the intended permanent exception.

Chronology matters: the exact owner rulings were recorded **September 12 at 16:50–16:53 America/New_York** (20:50–20:53 UTC). The Glide result was recorded **September 12 at 23:27 local**, although its UTC date is September 13. The current September 13 conversation recalls this as “today.” Preserve the actual timestamps rather than silently redating the decision.

The owner explicitly settled unification, Tauri's desktop role, and TanStack/Storybook/Glide. The detailed choice of Vite rather than TanStack Start is the subsequent assistant migration recommendation. No separate verbatim owner approval of every detailed package choice was recovered. The owner then requested a loss analysis and expressly authorized testing the Glide alpha; that is not proof of a completed product migration.

## Exact conversation provenance

Primary source A: `C:/Users/matts/.codex/archived_sessions/rollout-2026-09-12T12-12-05-01a09663-e9bb-72c2-bfce-c0714cba8d66.jsonl`. Session ID `01a09663-e9bb-72c2-bfce-c0714cba8d66`. Line numbers below refer to the source JSONL, not an exported narrative. Quotes retain owner spelling. Only architecture-relevant excerpts are reproduced.

| Source | UTC timestamp | Role and exact evidence | Message/turn identity |
|---|---|---|---|
| A:3335 | 2026-09-12T20:50:34.662Z | Owner: “not rigjt this second” | message `msg_01a09762-e026-75b1-924c-d277b628c7ff` |
| A:3338 | 2026-09-12T20:50:35.688Z | Owner: “they were suppsed to share a stack with taurie added for desktop apps” | message `msg_01a09762-e428-7612-b9a7-9d23ff5a73c5`; turn `01a09762-b2d9-7510-91fa-3557cf8ec9bc` |
| A:3347 | 2026-09-12T20:50:59.640Z | Owner: “but they will be unified” | message `msg_01a09763-41b8-7191-9ca4-5a84b2abd6cb`; same turn |
| A:3356 | 2026-09-12T20:51:18.864Z | Owner: “this aslso but no the fractured frameworks” | message `msg_01a09763-8cd0-7951-8f48-7a26e9022d5f`; same turn |
| A:3369 | 2026-09-12T20:52:15.856Z | Owner: “tanstack storybook and glide are settled” | message `msg_01a09764-6b70-7e73-be6a-db35be708ae2`; turn `01a09764-5628-7933-a515-dab9599080f5` |
| A:3403 | 2026-09-12T20:53:08.196Z | Owner: “ideniktfy best way forward and what nees to stay and go” | message `msg_01a09765-37e3-7112-8dc3-e55c8703a4e1`; same turn |
| A:3626 | 2026-09-12T21:03:26.572Z | Assistant migration assessment: “single Vite + React + TanStack client platform”; retain product backends; port Advocatio routes and retire Next only after parity | message `msg_00b0899c05d32293016aa5bda41a2087d295b9d21bba88acee`; same turn |
| A:3689 | 2026-09-12T21:13:56.271Z | Owner: “just test the alpha” | message `msg_01a09778-432f-7bd0-9977-48bd9e053881` |
| A:4243 | 2026-09-13T03:27:24.169Z | Assistant reports isolated Glide alpha pass, exact-pin recommendation, native Tauri not proven | message `msg_00b0899c05d32293016aa61800baac87d2aa01383c54d97454` |
| A:4368 | 2026-09-13T03:32:06.642Z | Assistant reports handoff writer unavailable: no record ID, no verified readback, no loose fallback, no product adoption | message `msg_00b0899c05d32293016aa6192d13f887d2805bdd3a0d3f0d90` |

Earlier corroboration: `C:/Users/matts/.codex/archived_sessions/rollout-2026-09-12T11-54-02-01a09653-64ee-7c33-b6e7-bd3236b618af.jsonl:116`, 2026-09-12T16:00:50.209Z, message `msg_01a09659-9c21-7841-8697-ac0031892450`: owner requested Family Court “update the stack to be inline wit6h the tanstack/ storeybook/ react/ glide stack”. This is narrower than the later unification ruling.

A forked archive `rollout-2026-09-12T17-14-41-01a09778-f275-77c2-a8fc-a99712a1909f.jsonl` repeats the history with fork timestamps. It is corroborating duplication, not an independent later owner decision.

## Decision versus observed implementation

Package values below are **current local manifest declarations**, including ranges; they are not latest upstream versions, installed-resolution proof, or deployment proof.

| Layer | Recovered decision / assessment | Observed local implementation | Remaining consequence |
|---|---|---|---|
| Shared frontend | Owner demands actual common stack and no fractured frameworks. Assistant specifies React/TypeScript/Vite with TanStack Router/Query. | Advocatio `web/package.json`: Next `^16.3.1`, React/React DOM `19.2.3`; no TanStack, Storybook, or Glide dependency. | Port framework-specific routes, session handling, and chat transport with parity gates. Existing Next code remains useful migration source. |
| Desktop | Owner adds Tauri around the same application foundation. | Family Court has Vite build and Tauri scripts/API `^2.9.1`; Intake has Vite and Tauri API `2.11.1`. | Do not fork another desktop frontend. Native package proof is a separate exit gate. |
| TanStack | Settled by owner. Router/Query foundation and selective Table/Store usage are assessment details. | Family Court Query `^5.90.5`, Router `^1.170.32`, Table `8.21.3`, Virtual `^3.14.11`; Intake Store `0.11.1`, Table `9.2.4`. | Common conventions/version compatibility need deliberate migration. A package named TanStack Table is not itself the canvas grid. |
| Storybook | Settled; assessment makes it executable shared-component/state contract. | Family Court Storybook/React-Vite declarations `^10.5.10`; Intake exact `10.5.10`; Advocatio has no Storybook declaration. | Shared fixtures and component acceptance still needed. |
| Glide / React | Owner settled Glide, then asked to test alpha. Assistant recommends exact `6.0.4-alpha24` after isolated test. | Family Court still declares Glide `^6.0.3` with React `19.2.3`; canonical Intake declares Glide `6.0.3` with React `18.3.1`. Isolated harness declares exact alpha24 with React `19.2.3`. | Neither force React 18 as automatic mandate nor claim alpha has landed in products. Adopt through bounded browser regression and rollback gates. |
| Shared visual contract | Existing portable Carbon-Linen-Seal package remains reusable; owner later says shared colors alone are insufficient. | `resources/design/README.md`: version 1.0.0; `tokens.json` authoritative; generated CSS; pinned per-repository consumption until packaging exists. Family Court has vendored adoption. | Shared components/application conventions are additional work. Token adoption alone does not complete unification. |
| Editor / document authoring | No cross-product rich-text editor selection recovered in bounded September 12–13 owner messages. | Advocatio manifest has no Plate, Tiptap, or Lexical declaration. Glide `DataEditor` in compatibility test is a grid editor, not a legal document editor. | Carry editor selection/integration as unresolved requirement; do not invent an approved library or use grid test as document-authoring proof. |
| Agent SDK / chat | Migration assessment expressly says not to copy Family Court's Claude sidecar assumptions into every app. | Family Court declares Claude Agent SDK `^0.3.263` and Fastify `^5.2.0`. Advocatio declares `ai ^7.0.68`, `@ai-sdk/react ^4.0.71`, `@ai-sdk/openai ^4.0.43`; its historical AGENTS describes Agno adapter behind neutral interfaces. | No universal agent runtime choice recovered. Preserve domain policies and move server secrets/chat mediation outside framework-specific browser code. |
| Backend / API / data | Assessment retains product authority, API services, database and workflow technologies. Root AGENTS independently isolates CCC, Intake, and Docstore. | Advocatio root `pyproject.toml`: Python `>=3.12,<4`, FastAPI `>=0.141,<0.142`, Pydantic `>=2.13,<3`; AGENTS preserves accepted LegalSourcePackage consumption and human review/release rules. | Frontend unification does not merge storage, credentials, runtime processes, or evidence authority. |

Current files inspected: `Legal-desktop/AGENTS.md`; `Legal-desktop/web/package.json`; `Legal-desktop/pyproject.toml`; `projects/family-court-workbench/package.json`; canonical `Consignatio/Intake/package.json`; `resources/design/README.md`, `CALLABILITY.md`; bounded terms in Probata `docs/DECISION_LOG.md`, `NAMING.md`, `COORDINATION.md`. The `projects/consignatio` duplicate overlay from the original assessment is not current authority; canonical Consignatio is used here.

## Compatibility and persistence boundaries

The prior test report at A:4243 says React/React DOM 19.2.3, Glide 6.0.4-alpha24, Vite 8.2.2, Storybook 10.6.0, resolved Router 1.170.33 and Query 5.102.8 passed isolated build/browser checks. Reported browser behavior: selection, ArrowDown, boolean and text edits, and 2-by-2 TSV paste on a 100,000-row lazy grid. Reported tests: 6 files, 14 passed, 4 canvas/jsdom skips then covered manually. **These are recovered prior test results, not rerun in this recovery.**

The native Tauri check failed first to find `link.exe`, then to resolve `kernel32.lib` in the configured Visual Studio environment. The report marks native packaging inconclusive. This recovery makes no new compatibility claim. The isolated artifact remains present at `C:/Users/matts/.codex/visualizations/2026/09/12/01a09663-e9bb-72c2-bfce-c0714cba8d66/glide-alpha-compat/package.json`; its alpha/React declarations were read directly.

The final prior handoff explicitly had **no Docstore record ID** because `docstore_handoff_write` was absent in the loaded tool surface. That is a plausible concrete reason the decision felt unlanded: the conversation and isolated test exist; verified governed handoff and product adoption did not follow in that source session. This is an inference about discoverability, not proof that no later task persisted it elsewhere.

No native Docstore tool surfaced in this delegated task's discovery. The parent was notified to handle governed query/registration if available. This new authorized planning input is not a claimed Docstore decision write or verified handoff. No network compatibility research was needed because the task recovered decisions and observed manifests rather than reselecting the stack.

## Contradictions and next planning constraints

1. Historical Legal-desktop AGENTS still says “Stack (locked)” Next and “Do not switch this app to Vite.” The later exact owner unification ruling supersedes that destination; preserve current runtime until a reviewed parity migration. Do not silently rewrite this old instruction in this read-only lane.
2. Shared design README describes independent vendored releases, while the later assessment proposes shared workspace packages after cutover. Repository independence and source provenance still govern; the owner-selected canonical independent Advocatio path is a September 13 exception. A shared stack does not require flattening Git histories or forcing direct sibling filesystem imports.
3. TanStack/Storybook/Glide are settled; Vite/Router/Query is the recovered concrete assessment, not a new blank-slate framework choice. TanStack Start is not the recommended common runtime in that assessment.
4. React 19 plus alpha24 passed the prior isolated test; native Tauri packaging and real product regression remain unproven. Current manifests have not uniformly adopted alpha24.
5. No approved universal editor or agent SDK was found. Backend-domain preservation is supported; pretending the Family Court sidecar is the universal backend is not.
6. Design callability matrix establishes only limited route/addressability evidence. Shared styling and source code do not prove cross-product launch, authentication exchange, mutation workflows, or deployed integration.

Recovery searched bounded September 12–13 Codex active sessions, relevant archived sessions after active histories lacked the ruling, and current Propria Claude project logs for the same dates. No qualifying later Claude owner contradiction was returned by the bounded search. This is not an assertion that every historical/private archive was searched.
