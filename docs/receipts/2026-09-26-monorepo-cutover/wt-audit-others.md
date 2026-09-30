# Git Audit - Other Propria Modules

> Mechanical, READ-ONLY audit. git fetch origin was run once per repo (twice for
> vestigia-geodata_processor, only to confirm a failure was reproducible). No merge, pull,
> checkout, commit, push, or worktree removal was performed anywhere. This covers the five
> repos assigned to this pass; Probata is a separate sibling audit.
>
> Scope: modules/Consignatio, modules/Legal-desktop, modules/vestigia-geodata_processor,
> modules/vestigia-geodata_processor/traceiq-rebuild, modules/memsearch.
>
> Byline: Claude Code - Sonnet 5 - 2026-09-26

## Executive summary

| Repo | Branch | Dirty (tracked-mod / untracked) | vs origin/branch | Local branches | Branches w/ commits on no remote | Linked worktrees | Verdict |
|---|---|---|---|---|---|---|---|
| Consignatio | main | 11 (3 / 8) | 39 ahead / 8 behind - DIVERGED | 18 | 4 | 4 | NEEDS-DECISION |
| Legal-desktop | master | 5 (2 / 3) | 0 / 0 (in sync) | 4 | 0 | 1 | SAFE-TO-IMPORT |
| vestigia-geodata_processor | main | 3039 (2085 / 954) | 0 / 0 vs a STALE cached ref - origin is unreachable | 1 | 0 | 0 | NEEDS-DECISION |
| traceiq-rebuild | master | 0 | 0 / 0 (in sync) | 5 | 1 | 0 | NEEDS-PUSH-FIRST |
| memsearch | main | 0 | n/a - origin has zero refs | 1 | 0 | 0 | SAFE-TO-IMPORT |

Note on the baseline numbers given in the task: Consignatio numbers (39 ahead / 8 behind / 11
dirty) and Legal-desktop (5 dirty) matched the live audit exactly. vestigia did NOT match, the
task said approximately 2088 dirty files, the live count today is 3039. The live number is
reported throughout as the verified figure.

---

## Consignatio

- Path: modules/Consignatio - independent git root, .git present directly.
- Remote: origin -> https://github.com/Cursedpotential/Consignatio.git. git fetch origin
  succeeded (exit 0).
- Current branch: main.
- Dirty: 11 total = 3 modified-tracked + 8 untracked.
  ```
   M docs/URGENT-TODO.md
   M docs/receipts/catalog-reconciliation-2026-09-20/RECEIPT.md
   M docs/receipts/source-recovery-2026-09-20/RECEIPT.md
  ?? Intake/docs/PROPOSAL-2026-09-21-SMART-SUGGESTIONS-AGENT.md
  ?? Intake/docs/receipts/NF-INTAKE-P0-FUNCTION-2026-09-23.md
  ?? LEVEL-2-ARCHITECTURE.md
  ?? casebible/tools/ai_chats_folder_census_20260922.sql
  ?? casebible/tools/vault_level2_scaffold_20260921.py
  ?? casebible/vault-sorted.7z
  ?? docs/receipts/2026-09-25-dawarich-compose.yml
  ?? docs/receipts/2026-09-25-dawarich-mkadmin.rb
  ```
- Total local branches: 18.
- Branches with commits that exist on NO remote (the real loss-risk list, computed as
  git rev-list BRANCH --not --remotes, checked against every remote, not just origin):

  | Branch | Unique commits | Most recent unique commit date |
  |---|---|---|
  | main | 39 | 2026-09-24 14:21:06 -0400 |
  | codex/d03-ocr-literal-20260923 | 3 | 2026-09-23 16:07:30 -0400 |
  | codex/casekit-ab | 2 | 2026-09-11 22:20:37 -0400 |
  | local-archive/pre-private-publication-20260911 | 2 | 2026-09-11 22:18:23 -0400 |

  The other 14 local branches (ai-chats-narratives-20260918, chat-dirs-index-20260918,
  chat-timeline-mvp-20260918, claude/nifty-nash-6d261e, codex/consignatio-preserve-20260913,
  codex/consolidate-consignatio-source-20260919, codex/d03-independent-review-20260923,
  codex/d03-ocr-literal-pr-20260923, codex/r2-b2-best-copy-20260913,
  codex/r2-b2-migration-reconcile-20260913, codex/route-casebible-surface-20260913,
  feat/superindex-service, intake/supervisor-2026-09-14-search-surface-spacedrive-gate,
  repair-toolkit-snapshot) have ZERO unique commits - every commit on them already exists on
  some remote branch.

### Special attention - is main a fast-forward or a real merge?

- git rev-list --left-right --count origin/main...main gives 8 behind, 39 ahead. Confirmed
  live, matches the number given in the task exactly.
- git merge-base --is-ancestor origin/main main -> FALSE (origin/main is NOT an ancestor of
  local main).
- git merge-base --is-ancestor main origin/main -> FALSE (local main is NOT an ancestor of
  origin/main either).
- Common ancestor (git merge-base origin/main main): cfdeec87c32a3553ac004eb723fe362541924284.
- Plain answer: this is a genuine bidirectional divergence. A fast-forward is NOT possible in
  either direction, reconciling requires a real merge or rebase, not a plain pull/push.
- Are the 39 local-only commits sitting on origin under some other branch name? NO. The
  branch-risk check above (main unique_commits_not_on_any_remote = 39) checked against every
  remote-tracking ref that exists locally, not just origin/main, and still found all 39 unique
  to local. They exist nowhere on the remote today. Sanity check: local main tip commit
  d6c0f1b is itself one of those 39 unpublished commits.

### Linked worktrees (5 entries total, including the main checkout)

| Path | Branch | Dirty | Ahead of origin/main | Merged into origin/main | Pushed |
|---|---|---|---|---|---|
| modules/Consignatio (main checkout) | main | 11 | 39 | NO | NO, tip not on any remote |
| _worktrees/consignatio-d03-independent-review | codex/d03-independent-review-20260923 | 0 | 0 | YES | YES, reachable via origin/main |
| _worktrees/consignatio-d03-ocr-literal | codex/d03-ocr-literal-20260923 | 0 | 3 | NO | NO, tip not on any remote (this is the risk branch above) |
| _worktrees/consignatio-d03-ocr-literal-pr | codex/d03-ocr-literal-pr-20260923 | 0 | 0 | YES | YES, reachable via origin/main (the same-named remote branch has drifted to an older commit, but the local content is already safe inside origin/main) |
| _worktrees/consignatio-superindex-service | feat/superindex-service | 0 | 17 | NO (not yet merged to main) | YES, origin/feat/superindex-service exactly matches local tip |

None of the 5 worktree directories are locked or prunable; all paths exist on disk.

Bottom line for Consignatio: 3 of 4 linked worktrees and 14 of 18 branches are already fully
safe on origin. The risk is concentrated in exactly two places: (1) the main checkout itself,
39 unpublished commits plus 11 uncommitted files, and (2) the small
codex/d03-ocr-literal-20260923 worktree (3 unpublished commits). main cannot simply be pushed
or pulled, it needs a deliberate merge/rebase decision before .git can be safely discarded.

---

## Legal-desktop

- Path: modules/Legal-desktop - independent git root.
- Remote: origin -> https://github.com/Cursedpotential/Legal-Workspace.git. Fetch succeeded.
- Current branch: master.
- Dirty: 5 total = 2 modified-tracked + 3 untracked.
  ```
   M web/src/app/documents/page.tsx
   M web/src/app/evidence-catalog/page.tsx
  ?? .cnf/memories/project_memory.json.bak-20260921T022602-pre-consolidate
  ?? deploy/office-editor.fragment.yaml
  ?? docs/receipts/2026-09-23-advocatio-adoption-auth.md.backup_20260923_075106
  ```
- Ahead/behind origin/master: 0 / 0. origin/master is an ancestor of local master AND local
  master is an ancestor of origin/master, they are identical. Fully in sync.
- Total local branches: 4 - codex/d09-legal-package-consumer-20260923 (2026-09-23),
  fix/advocatio-tailnet-manifest-20260923 (2026-09-23), master (2026-09-26),
  wip/advocatio-adoption-auth-20260923 (2026-09-23).
- Branches with commits on no remote: NONE. All 4 branches are fully present on origin.

### Linked worktrees (2 entries total, including the main checkout)

| Path | Branch | Dirty | Ahead of origin/master | Merged into origin/master | Pushed |
|---|---|---|---|---|---|
| modules/Legal-desktop (main checkout) | master | 5 | 0 | YES (identical) | YES, exact match |
| _worktrees/legal-d09-package-consumer | codex/d09-legal-package-consumer-20260923 | 0 | 0 | YES | YES, exact match |

Bottom line for Legal-desktop: every commit in this repository, on every branch and in both
worktrees, already exists on origin. The only thing not yet in git anywhere is the 5
uncommitted files in the main checkout, carry those over explicitly (commit or copy) since a
history-based import will not include uncommitted content.

---

## vestigia-geodata_processor

- Path: modules/vestigia-geodata_processor - independent git root.
- Remote: origin -> https://github.com/Cursedpotential/TraceIQ (no .git suffix).
  git fetch origin FAILS: remote reports Repository not found, fatal error, repository
  https://github.com/Cursedpotential/TraceIQ/ not found (exit 128). Reproduced twice (once
  in the batch script, once standalone), this is not a transient network error, the named
  GitHub repository is not reachable under this URL (renamed, deleted, or made private in a
  way that returns 404 for this credential).
- Because live fetch is impossible, every vs-origin comparison below uses the STALE cached
  refs/remotes/origin/main ref already sitting in this local repo from some earlier successful
  fetch, it cannot be refreshed right now. FETCH_HEAD is 0 bytes (todays failed attempts wrote
  nothing); the cached ref own commit timestamp is 2026-09-06.
- Current branch: main, HEAD = e1a4acd09b9fe8c11f97cf5a284412c93b4b0df9, this is EXACTLY EQUAL
  to the cached origin/main SHA. Relative to that last-known snapshot there is no divergence at
  all (0 ahead / 0 behind; both merge-base ancestor checks pass trivially because the two refs
  are identical).
- Total local branches: 1 (main only). Branches with commits on no remote: none (consistent
  with local matching the cached remote exactly).
- Worktrees: 1 (just the main checkout, no linked worktrees).
- traceiq-rebuild sub-directory: confirmed NOT a submodule or gitlink (no .gitmodules, not
  present in git ls-tree HEAD), and is explicitly listed in the .gitignore for this repo
  (line 98: traceiq-rebuild/). git status correctly reports nothing for that path. It is a
  fully independent nested git repository (own .git, own git-common-dir pointing at itself)
  and is audited completely separately below, no overlap or risk of double-counting.

### Special attention - classifying the 3039 dirty files

The task described this repo as having approximately 2088 dirty files; the live, verified
count today is 3039 (954 untracked + 2085 modified-tracked). Reporting the live number per the
verify-before-claiming rule, flagging the mismatch rather than silently using the older figure.

Status-code breakdown (first two porcelain characters):

| Code | Meaning | Count |
|---|---|---|
| question-question | untracked, new, never committed | 954 |
| space-D | tracked in HEAD but missing from the working tree, deleted on disk, not staged | 2085 |
| M or MM or other | actual content edits to tracked files | 0 |

Important: not one single already-tracked file has edited content. The entire
modified-tracked bucket is pure on-disk deletions of files that are still fully present in the
current HEAD commit. Spot-verified: 04_Utilities/Claude-Slash-Commands-main accounts for 825
of the 2085 deletions, and git ls-tree confirms all 825 paths are still in HEAD. These 2085
are NOT a loss risk as long as the commit history itself is preserved through the monorepo
import and not a raw working-tree file copy, they are trivially recoverable via git checkout
from history.

One specific case fully root-caused: the 57 untracked files under top-level TraceIQ_Main
directory are byte for byte the same relative paths as 57 of the deleted files under
01_Timeline_Forensics/TraceIQ_Main (diff of the two path lists = 0 differences). This was a
plain filesystem move done outside git, no git mv was used, this is not new content and not a
loss, just an unrecorded rename.

Top-level directory prefixes by file count, all 3039 files fall into 10 top-level buckets:

| Prefix | Count | Untracked or Deleted |
|---|---|---|
| 04_Utilities | 1519 | all deleted |
| raw_api_responses | 896 | all untracked |
| 02_Voice_Analysis | 236 | all deleted |
| 01_Timeline_Forensics | 118 | all deleted |
| 00_Documentation | 114 | all deleted |
| TraceIQ_Main top level, see move note above | 57 | all untracked |
| 03_Evidence_Analysis | 55 | all deleted |
| Utilities legacy, distinct from 04_Utilities | 42 | all deleted |
| Narcissistic-Abuse-AI-Configs-main.zip, stray root file | 1 | deleted |
| dot-cnf | 1 | untracked |

Top second-level prefixes, the ones that matter for the generated-vs-source judgment below:
04_Utilities/Claude-Slash-Commands-main (825), 04_Utilities/External_Modules (226),
02_Voice_Analysis/Context_Analysis_Suite (153),
04_Utilities/Manipulative-Expression-Recognition-main (117), 04_Utilities/Data_Converters (87),
02_Voice_Analysis/Chronicle_Voice_App (76), 00_Documentation/Context_Files (65),
01_Timeline_Forensics/location-admin (61), 01_Timeline_Forensics/TraceIQ_Main (57),
Utilities/SCRIPTS (42), 04_Utilities/Narcissistic-Abuse-AI-Configs-main (41),
03_Evidence_Analysis/ConflictAnalysisApp (40), 04_Utilities/Chunker (34),
04_Utilities/DirectoryScanner (33), 00_Documentation/STACK_Deployment (26).

Extensions, top ones: json 1083, md 1071, together 71 percent of all 3039 entries, then py 225,
tsx 113, ts 93, txt 57, html 36, yaml 21, pdf 21, js 21, svg 19, png 19, yml 17, sql 16,
gitignore 13, wasm 9, sh 9, plus 117 extensionless files.

Judgment, generated or vendored versus real source, this is evidence only, .gitignore was NOT
touched:

- Looks generated or vendored, arguably should not be tracked as this project own source.
  raw_api_responses holds 896 files, 100 percent untracked, zero of them ever in HEAD, these
  read as raw API response dumps, meaning data output, not authored source. The
  04_Utilities folders suffixed -main or -master (Claude-Slash-Commands-main,
  Manipulative-Expression-Recognition-main, Narcissistic-Abuse-AI-Configs-main,
  context-toolkit-plugin-main, Document-Analyser-MCP-main, langextract-mcp-main,
  TimelineExtractor-master) together account for over 1000 of the 2085 deletions. The
  -main or -master suffix is the naming convention GitHub itself applies to a Download-ZIP of a
  repo default branch, a strong signal these are vendored snapshots of other peoples repos, not
  hand-written project code. The .gitignore already in this repo already ignores several
  sibling vendored zip archives by the identical pattern (gemini-cli-custom-slash-commands-
  main.zip, gemini-history-learning-main.zip, open-whispr-main.zip,
  plistsubtractor-master.zip, and others) plus one vendored folder cache directory, confirming
  this is an existing, recognized pattern in the repo, just not yet applied to these specific
  unzipped folders.
- Looks like real, hand-organized project source: the top-level numbered taxonomy itself
  (00_Documentation, 01_Timeline_Forensics, 02_Voice_Analysis, 03_Evidence_Analysis) and the
  custom-named apps under it (Chronicle_Voice_App, ConflictAnalysisApp,
  forensic-data-refinery, location-admin, TraceIQ_Main), none carry the GitHub-zip naming
  signature, and the source mix inside them (py, ts, tsx) reads as authored applications
  rather than vendor drops.
- This is presented as evidence for the owner to decide on, not a decision made here.

Bottom line for vestigia-geodata_processor: two independent reasons for NEEDS-DECISION.
First, origin is dead with a 404, so there is no live remote to confirm todays state against
or to treat as a backup, and the only claim to safety here is an exact match against a
3-week-old cached ref. Second, 954 untracked files, mostly raw_api_responses, exist ONLY on
this disk and are not in git history at all, a history-based monorepo import will silently
drop them unless someone explicitly decides to carry them over or intentionally leave them
behind.

---

## traceiq-rebuild

- Path: modules/vestigia-geodata_processor/traceiq-rebuild - confirmed independent git root
  nested inside the vestigia working tree (own .git, own git-common-dir pointing at itself,
  not a submodule of the parent, see vestigia section above).
- Remote: origin -> https://github.com/Cursedpotential/traceiq-rebuild.git. Fetch succeeded.
- Current branch: master.
- Dirty: 0 (clean working tree).
- Ahead/behind origin/master: 0 / 0, fully in sync (identical refs, both-direction ancestor
  checks pass).
- Total local branches: 5 - claude/chat-disappeared-51dbff (2026-07-28),
  claude/debug-broken-code-cc1804 (2026-07-24), claude/new-task-3e7193 (2026-07-24), master
  (2026-09-20), ui-scaffold (2026-09-20).
- Branches with commits on no remote:

  | Branch | Unique commits | Most recent unique commit date |
  |---|---|---|
  | claude/chat-disappeared-51dbff | 1 | 2026-07-28 06:47:30 -0400 |

  The other 3 non-master branches (claude/debug-broken-code-cc1804, claude/new-task-3e7193,
  ui-scaffold) have zero unique commits, already fully present on origin.
- Worktrees: 1 (just the master checkout, no linked worktrees).

Bottom line for traceiq-rebuild: nearly everything here is already safe, master and 3 of 4
other branches are fully backed up on origin, and the working tree is clean. The only
outstanding item is one small, roughly 2-month-old orphan branch with a single unpublished
commit, currently not checked into any worktree. Push it, or explicitly decide to abandon it,
before deleting .git.

---

## memsearch

- Path: modules/memsearch - independent git root.
- Remotes: origin -> https://github.com/Cursedpotential/memsearch.git, a personal fork under
  the same account; upstream -> https://github.com/zilliztech/memsearch.git, push disabled,
  configured as DISABLED-private-fork, this is the open-source project it was forked from.
- git fetch origin: exit 0, but returned ZERO refs. git remote show origin reports HEAD
  branch unknown, and git branch -r lists only upstream refs, nothing under origin at all.
  This reads as origin being a genuinely empty but reachable GitHub repo, nothing has ever
  been pushed there, unlike the vestigia fetch failure above which was a hard 404, the fetch
  itself succeeded cleanly here, it just had nothing to hand back.
- Current branch: main, tracking upstream/main directly, 0 ahead / 24 behind upstream.
- Dirty: 0 (clean working tree).
- Total local branches: 1 (main only). Branches with commits on no remote: none,
  git rev-list main --not --remotes returns 0, every commit on local main already exists in
  upstream/main history, consistent with being 0-ahead of upstream.
- Worktrees: 1 (just the main checkout, no linked worktrees).

Bottom line for memsearch: there is no unique commit anywhere in this repository, everything
present already exists in the public upstream project, and the working tree is clean. Worth
flagging as an FYI rather than a risk: origin, the personal fork meant to hold any
Propria-specific changes, has never received a single push. If Propria-specific work was ever
intended to live here on top of upstream, none currently exists in this checkout.

---

## Method notes

- git fetch origin was run exactly once per repo, twice only for vestigia-geodata_processor,
  to confirm its failure was reproducible and not transient. No merge, pull, checkout, commit,
  push, or worktree removal was run anywhere, at any point.
- First pass of the data-collection script hit a bash arithmetic bug: grep -c prints 0 on zero
  matches but still exits with status 1, which double-triggered a fallback and produced a
  two-line value that crashed the bash arithmetic evaluator. That crash is fatal to a
  non-interactive bash script, not just the one failing statement, which silently truncated
  the first run right after the status check for traceiq-rebuild and before memsearch was
  touched at all. No git command had run yet for memsearch at that point. Fixed by switching
  to an awk-based counter, then re-run for the two affected repos; all figures in this report
  reflect the corrected run.
- vestigia directory-prefix and extension counts were computed from git status in
  porcelain v1 format with untracked files shown in full, stripping the 2-character status
  prefix. Checked for rename arrows first, zero found, so no rename-parsing edge cases apply
  to the counts above.
- Raw command transcripts, full per-repo status listings, and parsed worktree lists are kept
  in this session scratchpad for line-by-line re-verification: others_raw.txt,
  others_raw_part2.txt, others_status_REPO.txt, others_wtlist_REPO.txt, others_wt_detail.txt.
