# ADR-0014: Provider configuration subsystem; no hardcoded credentials

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

Plaintext Radar/Google API keys were found hardcoded in corpus script blocks (OQ#15).
Owner ruling: rotation is not the priority (keys may already be dead) — the structural
fix is. ADR-0011 already generalizes to N providers; owner wants a full provider
configuration section: which functions each provider serves, options, call definitions,
and editable credentials.

## Decision

1. **No credential is ever hardcoded** — not in code, transformations, or docs. Enforced
   at review + a pre-commit secret scan on the fresh repo.
2. **Provider registry (table `provider`)**: name, enabled flag, functions served
   (geocode-reverse / place-details / road-snap / gps-source / ...), base endpoints,
   call templates (which API calls we make, with what options/fields), rate + cost
   config, notes. Adding a provider = rows + an adapter, no schema surgery (ADR-0011).
3. **Credentials table (`provider_credential`)**: provider ref, key name, value,
   status (active/rotated/dead), added/rotated timestamps — editable at runtime via the
   API/UI parameter tooling (ADR-0008 pattern), changes audited. Lives in the private
   tailnet-only PG (consistent with owner's PII-in-private-live-DB policy); values are
   excluded from every export, snapshot, dump-to-git path, and backup manifest listing
   (pg_dump backups themselves are local-only per ADR-0004).
4. **API call ledger** (already ADR-0003/0011 audit): every call records provider,
   template used, cost, cache level — "what API calls got made" is queryable per
   provider/period.
5. **Corpus scrub (owner-ordered)**: hardcoded keys are redacted out of the working
   corpus files (placeholder markers), originals remain recoverable via local-only git
   d61afbd8; harvested values stashed out-of-band in the owner's local secrets store
   for reference/possible reuse. Never in the transcript, never in the new repo.

## Consequences

### Positive
- Key changes = table edit, zero code changes; provider onboarding is data + adapter;
  OQ#15's structural cause eliminated
### Negative
- Credentials-in-DB requires the DB itself stay private (already true: tailnet-only,
  no remote git, local backups)
### Risks
- d61afbd8 still contains pre-scrub files → acceptable: local-only repo, no remote;
  revisit only if that repo ever gains a remote
