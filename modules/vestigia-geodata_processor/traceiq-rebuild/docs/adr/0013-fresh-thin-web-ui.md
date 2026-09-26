# ADR-0013: Fresh thin web UI over the API; old UIs mined, not carried

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner — via structured question), Claude (Claude Code · Fable 5)

## Context

Four abandoned UI efforts (OQ#6): Flask+Jinja (templates missing), React location-admin
(multi-LLM query), React Evidence Processor v4/v7 (client-only), 2× Streamlit. ADR-0001
makes any UI a thin peer over the single API.

## Decision

Build a fresh, small, purpose-built web UI over the new API: browse/filter timeline,
annotate + manual override (with audit), trigger jobs, parameter tool (ADR-0008),
snapshot/export buttons. The four old UIs are mined for ideas in Phase B (location-admin's
multi-LLM query concept noted for a later wave) but none carries code in. NocoDB remains
available as an incidental admin window onto PG views; not the deliverable.

## Addendum (owner, same day): Evidence.dev as the reporting layer

Evidence.dev (open-source SQL+markdown → static report framework, native PG support)
joins as the READ-ONLY reporting companion — not a UI replacement: milestone snapshot
reports (rendered form of ADR-0005 snapshots, versioned + re-runnable = court-friendly),
analysis dashboards for the wave-1 lanes, and cheap homes for the immature recovered
view ideas (heatmap/daily-timeline/hourly-patterns). Interactive duties (annotation,
overrides, jobs, parameters) remain with the thin web UI. evidence-studio MCP is
available in-session for component docs at Phase C. Division: Evidence reads; the UI writes.

## Alternatives Considered

### Salvage React location-admin
- **Pros**: most advanced old effort
- **Cons**: wired to dead backends; carries the fragmentation era's debt into the clean rebuild
- **Why not**: thin-over-API is small enough that fresh is cheaper than rewiring

### NocoDB-only
- **Pros**: zero build
- **Cons**: no job triggers, parameter tool, or annotation audit; not court-demo friendly
- **Why not**: fine as a side window, not the front door

## Consequences

### Positive
- OQ#6 closed; UI stays disposable/swappable (ADR-0001 consequence realized)
### Negative
- One more thing to build in Phase C (kept deliberately small)
### Risks
- Scope creep → v1 feature list above is the contract; additions get their own decisions
