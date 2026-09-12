---
priority: critical
authority: owner_decision
status: active
---

# Shared result presentation and document revision contract

Owner-approved 2026-09-12. Applies to Docs/Docstore, Case Bible, Intake and future applications. Shared convention does not merge data stores, runtimes, credentials or CocoIndex tracking.

Visible implementations also follow the [Propria shared surface design contract](SURFACE-DESIGN-CONTRACT.md),
which maps these result meanings to shared context, state, accessibility and semantic token roles. The
result contract remains authoritative for data meaning; visual styling never changes lifecycle or
approval state.

## Clean results everywhere

Normalize source SDK values once, then adapt the same normalized result to agent tools, CLI/TUI, search lists, tables, previews, chat context, graphs and future federated tools. Prefer reusable DuckDB projection/tabulation for structured result sets, adapted from `Probata/probata/scripts/docstore/sq.py`; do not require a browser to run Python DuckDB or expose database SDK internals to the UI.

- Routine agent/CLI output: compact columns/rows and bounded excerpts; omit embedding arrays, repeated schema keys and irrelevant transport wrappers.
- UI: typed, pageable rows and previews, with the same field meanings and explicit full-detail retrieval. UI wire formats need not be identical to compact agent text.
- Preserve IDs, paths, provenance, score interpretation, authority, priority, lifecycle, approval revision, warnings, errors, freshness and missing-data status.
- Never silently hide row truncation, collapse corroborating occurrences, equate duplicate values with duplicate records, or describe an excerpt as a summary.
- Keep full retrieval/export possible. Compact presentation is not canonical data, source bytes, a hash input, an approval payload or a migration format.
- A shared reducer must not silently swallow statement failures, relabel incomplete output as success, or treat successful transport as successful work.
- Prefer upstream field projection and pagination as well as final rendering. Final compression alone does not reduce upstream transfer or provider cost.
- Bound input sizes, concurrency and query work; keep temp/cache on owner-designated non-system drives. No automatic corpus indexing or local inference for rendering.
- Existing public consumers need explicit versioned adapters; do not break their JSON contracts by replacing objects with arrays without migration/tests.

## Approved document lifecycle

Update the current search index. Append document revision history. Preserve human flags separately. Explicitly track which revision was approved.

- Stable logical document identity must survive edits and have an explicit rename/move association; current path-derived Docstore identity needs improvement.
- Append a revision only when content changes, with content hash, source reference and provenance. Replays are idempotent.
- Current search points at the latest successfully indexed revision. Historical search is explicit; superseded passages must not be mixed into the current body.
- Human flags remain an overlay outside ingestion-owned rows. Flag audit revision is different from document content revision.
- Approval references a specific document revision/hash and actor/time. Subsequent edits preserve history and disclose changed-since-approval; they do not silently approve new wording or erase the prior approval.
- Record correlated CocoIndex completion receipts; do not claim complete revision indexing from a document hash alone.

## Delivery tracking

- [x] Owner-approved cross-application contract recorded and linked from system boundaries.
- [x] First Docstore adapter: in-memory DuckDB, bound JSON projection, excerpts, omission metadata, no temporary files or arbitrary SQL.
- [x] Compact Docstore MCP retrieval and opt-in CLI formatting; original structured/full tools retained.
- [ ] Apply shared adapter contract to existing Case Bible/Intake UI, APIs, CLI/TUI and agent surfaces after mapping their response consumers.
- [ ] Activate and test host/federated tool connections; source package is not globally installed.
- [ ] Implement append-only document revisions, stable identity and revision-bound approvals in the ingestion pipeline.
- [ ] Validate edit/no-change/rename/failure/retry and approval cases end to end.

The checked adapter items are local source implementation, not a claim that every current application is migrated or that the document revision pipeline is implemented.
