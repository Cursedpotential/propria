# ADR-0004: Placement on data-pg with milestone-triggered local backups

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

Owner's Windows machine is aging and heavily loaded; the platform already operates
PostgreSQL (`data-pg` on ovh-data via Coolify, tailnet-reachable, hosting case data —
established precedent). Owner's history includes a major local-drive data-loss incident
(D:\Backup) and a hard preference for host-visible backups. ADR-0002 chose PG+PostGIS+
pg_duckdb but deferred instance placement.

## Decision

The TraceIQ database lives in the existing `data-pg` instance on ovh-data (or a sibling
Coolify PG app on ovh-data if the ADR-0002 extension test gate demands it). The owner's
machine is a thin client over the tailnet. No local SQLite middleman — one truth only.

**Backup discipline (owner mandate):** every major operation (ingestion, compile/derive
run, schema migration) ends with an immediate local backup — `pg_dump -Fc` pulled to a
local backup directory, filename = timestamp + milestone label, SHA-256 recorded in an
in-DB backup registry, dump sanity-verified (`pg_restore --list`). **A job is not
"done" until its backup is verified.** Optional R2 mirror as secondary. Backup is also
exposed as an on-demand API/CLI verb.

## Alternatives Considered

### Run PG locally on the owner's machine
- **Pros**: no network dependency, data physically local
- **Cons**: adds constant load to an overstressed machine; another service to babysit;
  platform tooling (NocoDB etc.) points at data-pg
- **Why not**: backup mandate covers the local-copy need without the operating cost

### Local SQLite mirror/middleman
- **Pros**: offline reads
- **Cons**: reintroduces split truth — the exact failure mode (home_base loss) being escaped
- **Why not**: divergent truth is worse than waiting out a tailnet outage

## Consequences

### Positive
- Zero standing load on the local machine; DB rides maintained infra
- Local, hash-verified restore points after every milestone; loss window ≈ one job
### Negative
- Working requires tailnet connectivity (accepted)
### Risks
- Backup directory location on a healthy local drive TBD at Phase C (D: is off-limits;
  candidate: E:\TraceIQ_Backups) — verify free space before first full dump
- data-pg image may lack PostGIS/pg_duckdb → sibling-app fallback already decided
