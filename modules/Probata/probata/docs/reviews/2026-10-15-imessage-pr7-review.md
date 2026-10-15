# PR 7 iMessage review fixes

Byline: Codex · GPT-6 · 2026-10-15.

Scope: all three validated comments on https://github.com/Cursedpotential/propria/pull/7,
already merged at inspection. Follow-up branch: `codex/imessage-review-20261015`,
based on `fdf23125991376de62a94af255e32030e3e8754d`. This receipt records source
validation before publication. The owner authorized committing, pushing, and opening
a follow-up PR on 2026-10-15. Deployment, live database application, and case-source
reprocessing were not performed.

- Comment 4242243432 is valid. The vendored SBV `internal/messaging_html_importers.go`
  functions `htmlRecordStart` and `projectIMessageHTML` require a div.message containing
  sent/received, with metadata inside that direction container. `sourceformat/html.go`
  now verifies that structure on the bounded retained HTML prefix. Four negative
  regression cases failed before the fix; valid sent/received and owner exports pass.
- Comment 4242243438 is valid. `deploy/data-pg.yaml:41` and `deploy/compose.yaml:26`
  mount `sql/bootstrap/schema_snapshot_20260907.sql`, not the live delta. Added only
  imessage_html to the snapshot CHECK at line 8286. Preserved the 16 existing values.
  Merely mounting the delta would fail its 17-format precondition because the snapshot
  lacked claude_ai_export_json at the initial review. Extended the existing exact-format Go contract test.

Validation: sourceformat package passed (22.544s); postgres package passed after
updating its expected format list (10.329s); git diff --check passed. Independent
read-only review found no introduced defects. Fresh PostgreSQL bootstrap was not run.
The separately attempted Python schema suite exposed an existing failure in
test_no_numbered_migrations_remain_in_sql_root due to sql/0100_claude_ai_export_structured_elt.sql.
No Python test edits remain.

All PR inline comments, review submissions and conversation comments were fetched.
The owner subsequently authorized the remaining comment, 4242243430. Confirmed it
with three independently failing routing cases: bubble-preview, from-me-extra,
and meta-info. The minimal classToken fix replaces word boundaries with quoted
value boundaries or HTML ASCII whitespace; both quoting styles remain supported.
Added class-token tests for first/middle/last tokens, ASCII whitespace, prefixes,
suffixes, and non-breaking spaces. All three substantive PR findings are now fixed
locally. No GitHub review threads were marked resolved.

Final verification after the class-token fix: `go test ./sourceformat ./postgres
-count=1` passed (sourceformat 2.638s; postgres 10.362s). `git diff --check` passed.

## PR 10 review follow-up

Codex comment 4286624052 correctly identified another existing bootstrap omission:
sourceformat/detect.go returns claude_ai_export_json, and RecommendHandler inserts
that value directly. Added it to both the snapshot CHECK and the exact test list.
The updated contract test failed against the previous snapshot before the SQL fix.
Independent read-only review confirmed the bounded change. All previous formats
remain admitted; the separate live delta and its trigger changes remain unchanged.

Verification after this fix: `go test ./sourceformat ./postgres -count=1` passed
(sourceformat 7.890s; postgres 15.295s), and `git diff --check` passed.
Fresh PostgreSQL initialization and live database application were not performed.

Prior memory supplied the PR and parser pointers; current GitHub metadata and local
code independently verified them. Docstore related-update preflight completed with
partial literal-search coverage. This local receipt has not been indexed in Docstore.
