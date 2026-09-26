<!-- Byline: Codex · GPT-6 · 2026-09-23 | P0 function-recovery lane -->

# Probata P0 function checkpoint — Sources continuation

## Result and proof boundary

The local Workbench candidate now loads successive pages of the B2 Sources listing. It retains first-page behavior, deduplicates file and folder rows, stops on a repeated or missing continuation token, and presents loading, retry, refresh, and terminal states. A first page containing folders but no files can still load the next page. This is a local code-and-fixture result. No live B2 listing, Temporal run, deployed browser journey, or end-to-end six-step operator flow was executed in this lane; Probata as a whole is **not certified functionally usable** by this checkpoint.

## Exact repository and workspace inventory

- Independent Git root: `E:/AI_Workspace/Projects/Propria/modules/Probata/probata`.
- Base: `main@e4966d06ed3b23ad3e7a67fb0b2563e6e33464f7`. At the pre-commit check, local `origin/main` divergence was `0 0`, and `git ls-remote origin refs/heads/main` returned that exact commit. Origin is `https://github.com/Cursedpotential/probata.git`.
- Seven linked worktrees were observed: canonical `main@e4966d06`; `probata-derive-sms-activity` `feat/derive-sms-activity@739d4b2f`; `probata-preview-mode-recovery` detached `eac18a51`; `probata-preview-search-calls` detached `d3490220`; `probata-review-message-browser` detached `eb8d8b58`; `probata-sources-screen` `feat/sources-screen@d3490220`; `probata-workbench-resume-20260920` `fix/workbench-operations-route@af74d365`. The six linked worktrees outside canonical reported no dirty paths on this inspection.
- Canonical pre-edit dirty entry: `? modules/forks/sbv`. Parent gitlink at HEAD/index and nested SBV checkout HEAD all equal `4031c3393856c3e1054a48eb020c44d92c9c04f3`; SBV is on `platform-sync` and its only nested dirt is untracked `.cnf/`. The parent status reflects nested untracked files, **not a changed gitlink pointer**. Neither SBV nor the parent pointer was edited or staged.
- Modified path and exact SHA-256 at review: `modules/workbench/web/src/components/sources/sources-screen.tsx` = `0C8EC2DB2FC504C41CA7F88A13007B65AF44365A09F4C914BB782CF66B69700C`.
- New paths and exact SHA-256 at review: `modules/workbench/web/src/components/sources/source-pages.ts` = `8FE80767C161861F480B7D86C398E251E0665CCD1E1E4DB7019617F83F08E91D`; `modules/workbench/web/smoke/source-pages.test.mjs` = `49A09DA8A8BFCCA5BE467EB29A36CB36CEC94C85C3B45D13CE917D134AC6D801`.
- Pre-edit source SHA-256 = `724FA7BA2623A7B173C850D461174ECBBCAC2DF9163865F37E65A7A8F3E1F4CA`. The timestamped backup hash matched before and after its move to `to_be_deleted/2026-09-23-probata-p0-source-backup/sources-screen.tsx.backup_20260923_0835`. Only the owner deletes from quarantine.

## D-159 six-step path at this pin

| Step | Current call path | This lane's evidence and remaining gap |
|---|---|---|
| 1. Open file/folder through an index | `/sources` -> `SourcesScreen` -> `/api/proffer/sources`; optional discovery search via `/api/intake/discovery/*` | The source browser and B2 continuation are implemented. First/next/terminal/duplicate states and 83 browser smoke checks passed locally. Real vault paging and corpus coverage are unverified. |
| 2. Verify relevance | Selected file -> `/api/proffer/source-inspection` -> inline image/PDF/text preview, or decoded-message viewer | Focused source-inspection API tests passed; source preview is read-only. Other format previews and actual owner relevance decision are not proved in a live journey. |
| 3. Ensure hash | Source inspection computes and displays SHA-256, labeled preview-only | Focused API test confirms immediate hash and no custody-digest claim. A real remote read and later authoritative custody recomputation were not exercised here. |
| 4. Pick handler and process | Sources `Process` -> `/api/proffer/start` or `/api/proffer/start-batch`; handler selection and Proffer run are backend concerns | UI and batch API contract tests passed. No Temporal/Go parser run or cross-store readback was executed. |
| 5. Preview parsed result | `/evidence/preview` Review -> Proffer preview, message/content projections, source-linked media | Browser smoke checks cover route and contract; API decoded-source tests passed. No real extracted result was inspected in a live browser. |
| 6. Fill gaps and context | Review warnings/missing-payload flags, repair decision and reversible annotations | Contract-level UI checks exist. Complete missing-context attachment and cross-source reconciliation remain unverified and require a real source/run fixture. |

The pre-ingest catalog dependency is read-only: Probata's Workbench BFF queries configured Case Bible PostgreSQL catalog/unit tables for names, provenance, and unit marks; configured Intake index URL handles content/hybrid/relationship lookup. Those records currently state `source_links_verified: false`, and indexed hits do not invent a B2 source reference. Core B2 source browsing, inspection/hash, and Proffer start are Probata-owned API paths. Deployment must prove the direct internal services are reachable only on the Tailnet, using Tailscale as the access barrier without a second Intake app login. Off-Tailnet access is through one Authentik-protected portal exposing user-facing surfaces; internal APIs, storage, admin, and debug paths remain private. None of those network boundaries was verified by local tests.

## Commands and observed results

- `uv run pytest -q ...` failed before collection: `uv trampoline failed to canonicalize script path`.
- Repository `.venv/Scripts/python.exe -m pytest -q modules/workbench/api/tests/test_proffer_source_browser.py`: 11 passed, one Starlette/httpx deprecation warning.
- Repository `.venv/Scripts/python.exe -m pytest -q` for `test_intake_discovery.py`, `test_proffer_batch.py`, `test_proffer_decoded.py`, and `test_source_unit_marks.py`: 32 passed, same deprecation warning. Synthetic API tests, no live Intake/B2/Temporal call.
- Workbench web `npm run smoke`: Vite/TypeScript build succeeded and 83 smoke tests passed, including the new runtime continuation tests. `npm run build-storybook`: succeeded. `npm run lint`: exit 0 with 18 pre-existing Fast Refresh warnings after the new warning was removed. After the final listing-error Retry control was added, `npx tsc --noEmit` and 12 focused Sources tests passed.
- `git diff --check`: passed. Generated `dist`, Storybook output, and pytest reports were not staged.

## Docstore publication state

The related-decision lookup and hosted Docstore health check ran before writing. The prescribed five-root source-sync dry run, `client.py sync --root E:\\AI_Workspace\\Projects\\Propria`, could not produce a manifest because `CF_MCP_CLIENT_TOKEN` is absent from this host process. The hosted registry still advertises `/sources/Probata/probata/docs`, while the canonical local checkout is under `modules/Probata/probata`; the local `docs/probata` junction resolves to that canonical checkout. A validated five-root source plan, incremental indexing, and exact readback were therefore **not completed**. The source Markdown is durable in the owning repository, but Docstore freshness for these new pages is pending a configured sync client and mount-path reconciliation. No full-source index was triggered from an unvalidated registry.

## Checkpoint and next safe action

The planned-change record is `docs/reviews/2026-09-23-probata-p0-pagination-plan.md`. The exact commit allowlist is this receipt, that plan, the Sources component, the continuation helper, and its smoke test. Do not stage the SBV gitlink or `.cnf/`.

Next, independently run one Tailnet browser journey against the exact deployed revision: browse a folder with more than 200 objects, verify a later-page file can be selected, inspected and hashed, start one safe TEST Proffer run, inspect the parsed preview and gap controls, and read back source/run receipts. Record deployed image/config/schema identity, off-Tailnet portal behavior, and Tailnet-only internal reachability. The shared Authentik-protected portal must have exactly one Probata user-facing listing, and its live launch must work; that separate P0 gate was not checked by this lane. A local build or healthy older container does not satisfy those gates.
