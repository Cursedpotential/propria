# ADR-0003: Intake discipline and source provenance

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

The corpus comprises two full Google Timeline exports — the other party's ~8-year export
(the evidentially critical corpus) and the owner's ~14-year export — plus ~5 small
fragment exports under owner accounts (Google's Maps on-device migration destroyed ~14
years of owner data; encryption issues fragmented the remainder across accounts), plus
miscellaneous Google geo exports in other formats. Large exports were chunked into
date-range slices. A prior incident (legacy parser silently under-extracting a modern-
format file) proved that assumed formats are dangerous. Everything is court evidence.

## Decision

1. **Source hierarchy: subject → account → export → chunk → record.** Subject (whose
   movements) is a first-class dimension; all analysis (home_base, overnight, schedule
   verification) computes per subject, never blended. Export metadata includes Google
   account, production date, acquisition date/method, format generation.
2. **Ingest everything; identity/dedupe is a view-layer concern.** Overlapping exports
   both load in full; same-event multi-export attestation is corroboration, not
   duplication. Nothing dedupes at intake, ever.
3. **Format detection is structural and fail-loud.** One parser per known source format
   (plugin model); parsers claim chunks by sniffing JSON shape. Unrecognized shape =
   hard stop + registered `unparsed` status. New formats join by adding a parser, never
   by loosening one.
4. **Raw files: content-addressed archive.** Originals stored in an archive directory
   named by SHA-256, mirrored to R2; DB stores hash + metadata + path.
5. **API responses are evidence.** Every external call (geocoding etc.) persists in
   full — request, response, provider, timestamp, cost. All sources welcome; none
   privileged; none discarded.
6. **Data-loss events are provenance.** Structured `data_loss_events` registry (what,
   when, cause, affected account/date-range) — e.g., the Maps-local migration deletion —
   so gap analysis distinguishes "destroyed by Google (documented)" from "phone off"
   from "not produced."
7. **Ingest priority: her 8-year export first**; owner fragments second.
8. **Acquisition-method provenance** captured per export (discovery production, shared
   account, etc.) — the admissibility-relevant record. Owner confirms an admissibility
   fight over her export is expected (opposing counsel will object; owner argues fair
   game), so this record is built to survive challenge: acquisition method + date +
   account-access basis + authorization context + acquirer, captured at intake, immutable
   thereafter. Lawyering stays with the owner; the schema's job is that the facts are
   there when the fight comes.

## Alternatives Considered

### Dedupe at intake (single canonical event per real-world moment)
- **Pros**: smaller tables, simpler queries
- **Cons**: destroys corroboration evidence; irreversible editorial decision at the
  least-informed moment
- **Why not**: violates every-shred-lands (ADR-0001); dedupe is cheap later, undelete is impossible

### Best-effort parsing of unrecognized formats
- **Pros**: more data flows without intervention
- **Cons**: proven silent-under-extraction failure mode; silent partial evidence is worse than none
- **Why not**: fail-loud is a forensic requirement

## Consequences

### Positive
- Per-subject analysis becomes natural (her-vs-his exchange-day comparisons)
- Gaps are explainable in advance; corroboration strengthens exhibits
### Negative
- More tables and registry discipline before "fun" analysis work
### Risks
- Subject attribution errors (fragment accounts) → mitigate: account→subject mapping is
  owner-confirmed data, not inferred
