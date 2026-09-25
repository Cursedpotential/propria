# Handoff

> _Byline: Claude Code · Fable 5.1 · 2026-09-21 00:18 EDT. Pointers only — the facts live in the Probata log `docs/planning/2026-09-20-TODO.md` (on `origin/main` of `Cursedpotential/probata`). Verify against git and the live system before acting._

## Resume From Here
Session 77aa963a, Probata Workbench lane ("stay on probata"). Four Opus agents are running in the background; **nothing they built is merged or deployed**. The owner was promised each agent's findings BEFORE any merge. One live DB change was applied tonight with the owner's approval (receipt below). All docs/ledger work is pushed to Probata `main` (last: `f9d4091`).

## Next Actions
1. When each agent reports, review its branch, then give the owner the findings (incl. what it could NOT verify) before merging:
   - `preview-search-and-calls` -> branch `feat/preview-search-and-calls` (worktree `_worktrees/probata-preview-search-calls`): server-side message search, calls-only preview, media endpoint `GET /api/proffer/previews/{handle}/media/{sha256}` in two modes, ascending order default.
   - `sbv-port` -> one-viewport Review shell; dense Glide rows default borrowing from SBV; straight SBV bubble view as an OPTIONAL toggle.
   - `derive-reconcile` -> branch `feat/derive-sms-activity` (worktree `_worktrees/probata-derive-sms-activity`): keep main's `derive_sms_threads_activity`, drop the duplicate, stage + routing + workflow branch, `DERIVED_ROOTS_JSON` mapping, `POST /reference-import/start-batch` (folder batch, one at a time).
   - (finished earlier) `derive-activity-routing` — its report is summarised in the log at 23:40.
2. After the derive branch merges: main's `sql/bootstrap/schema_snapshot_20260907.sql` must carry the three-value `execution_path` CHECK (live already does). Deploy order: one Coolify app at a time, explicit `GET /deploy?uuid=`; auto-deploy is OFF.
3. Fix at the derive step: owner's own number `(810) 259-4380` is keyed as a participant, so 1:1 MMS threads become "groups" split from their SMS thread.
4. Check with the owner: sms-20260110 derivation holds exactly 5,000 records (possible cap / cut file).
5. Deploy the pg healthcheck `-d` fix (committed, not deployed). Codex client move to the five-tool Docstore surface ("fix codex also") still open.
6. Queued, not started: extraction stage (entities, claim candidates) BEFORE the edit API; render step (processed records -> paginated Markdown conversations for `DerivedKnowledge/messaging/`); cross-platform merged views; repair-gate options + find-another-version; attachment text/OCR/STT plan; TraceIQ plan.

## Watch Outs
- **NO B2 work** beyond "can the app see it". Do not move the two existing `.derived/` folders. Catalog is off limits for this session.
- Vault is NOT built; top-level names (Triage, Recovered, SourceCorpus, KnowledgeBase, DerivedKnowledge, CaseManagement, EvidenceVault, Code, Archive — no numbers) must not appear in code; location is configuration only. `DerivedKnowledge/messaging/` = READABLE Markdown layer, not parser output.
- Other sessions share this repo: bulk-intake lane (owns `docs/decisions/2026-09-20-bulk-intake-owner-requirements.md`; this session claimed build-order steps 2 and 4 in the log at 23:58) and a Review-layout session. Main checkout `modules/Probata/probata` has another session's uncommitted edits — never edit it; work in `_worktrees/`.
- Edit tool for code (heredoc/python patches corrupt `\n` and `\U` escapes). Never `grep -r`. Guard hook blocks command text containing `rm -f`/`rm -r` and blocks untracked scripts against infra (commit first, pipe via ssh stdin). Bounded waits only.
- A hook blocks some compound git commands containing "merge" (e.g. `git merge-base`); run them separately.
- Owner style: answer first, bullets, short; verify before claiming; announce any recorded rule; never ask about things already decided.
