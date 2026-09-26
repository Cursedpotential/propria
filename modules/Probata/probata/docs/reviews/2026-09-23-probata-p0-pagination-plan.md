<!-- Byline: Codex · GPT-6 · 2026-09-23 | Probata P0 function recovery -->

# Probata P0 planned change: Sources listing continuation

## Planned-change block

- Base: independent Probata repository `main@e4966d06ed3b23ad3e7a67fb0b2563e6e33464f7`, aligned with local `origin/main` at inspection time.
- Owner and exact intended paths: `modules/workbench/web/src/components/sources/sources-screen.tsx` and a focused browser smoke test under `modules/workbench/web/smoke/`. This plan and a final receipt are Probata documentation.
- Evidence: the Sources screen requests `pageSize: 200` once and passes `onReachEnd={() => undefined}` to the grid. The Workbench API returns `continuation_token` and accepts it on the next `/api/proffer/sources` request. Files beyond the first page cannot be opened from the folder listing.
- Intended behavior: preserve the first-page selection and search behavior; append verified continuation pages, deduplicate rows and folder prefixes, reject a repeated continuation token, and show loading, failure/retry, and end state. Reset page state when mode, root, prefix, or filter changes.
- Intended verification: focused API source-browser tests using the repository Python runtime; browser test for first, next, terminal, retry, and duplicate-token cases; frontend build, lint, smoke, and Storybook build if product source changes pass the focused gate.
- Exclusions: no Intake, SBV, root Git pointer, deployment, release, or real source processing changes. Local checks cannot certify deployed behavior.
- Rollback: timestamped backup of the existing Sources file before edit; restore only this file if the patch fails. Preserve the backup until verification, then move it to Probata `to_be_deleted/` under owner-only deletion policy.

## Checkpoint

- Pre-edit Probata status: `? modules/forks/sbv` only. The parent SBV gitlink and nested checkout both resolve to `4031c3393856c3e1054a48eb020c44d92c9c04f3`; the nested checkout has only untracked `.cnf/`.
- `uv run pytest` could not start: `uv trampoline failed to canonicalize script path`. Direct `.venv/Scripts/python.exe` will be used if it can load pytest.
- Docstore related-decision lookup completed before this documentation change. Its latest health reported API/store up with an older degraded sync and two pending enrichments; source indexing still requires separate verification.
