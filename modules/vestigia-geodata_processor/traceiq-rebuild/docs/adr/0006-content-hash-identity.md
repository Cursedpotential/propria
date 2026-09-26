# ADR-0006: Content-hash identity for evidence; surrogates for derivations; serials for display

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

Four competing ID schemes appear across the corpus as "the" scheme: SHA-256 event_uuid
(hash of raw payload), UUIDv7, sortable serial `YYMMDDHHMMSS[.N]`, and `YYMMDDX` (OQ#2).
ADR-0005 splits identity into three distinct jobs the old schemes fought to do at once.

## Decision

1. **Raw record identity = SHA-256 of the verbatim record bytes.** Deterministic and
   recomputable by anyone from the JSON (court-demonstrable), tamper-evident. Same hash
   appearing in multiple chunks/exports = multiple attestations of one record —
   corroboration detection falls out of the identity scheme for free. Uniqueness is
   scoped per chunk; cross-chunk repeats are recorded attestations (ADR-0003 §2).
2. **Derived rows = plain surrogate keys** + provenance columns (source raw record(s),
   transformation version). Derived rows are regenerable; no ceremonial identity.
3. **Human-facing sortable serial (`YYMMDDHHMMi`-style) = generated display column** in
   views/snapshots — speakable in a courtroom, referenced by nothing.

Note: verbatim-byte hashing means the same real-world event in different export format
generations hashes differently — correct, they are different attestations. Cross-format
event matching is a Tier-1 derivation, not an identity claim.

## Alternatives Considered

### UUIDv7 everywhere
- **Pros**: time-ordered, distributed-generation-safe
- **Cons**: solves scale/distribution problems this system doesn't have; not derivable
  from evidence content; adds nothing hashes don't
- **Why not**: content hash gives determinism + tamper-evidence + corroboration free

### Sortable serial as primary identity (old CSV habit)
- **Pros**: human-friendly
- **Cons**: collides, encodes mutable interpretation (local time) into identity
- **Why not**: demoted to display column where it belongs

## Consequences

### Positive
- Identity is provable from evidence alone; corroboration automatic; OQ#2 closed
### Negative
- 64-char hex keys are ugly in ad-hoc queries (mitigated by display serial + surrogate
  ints on derived rows)
### Risks
- Canonicalization: "verbatim bytes" must be defined precisely (exact byte slice from
  source file, no re-serialization) — set at Phase C implementation, documented for court
