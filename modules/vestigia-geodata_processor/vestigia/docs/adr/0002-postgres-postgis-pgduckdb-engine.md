# ADR-0002: PostgreSQL + PostGIS + pg_duckdb as the evidence-core engine

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

ADR-0001 makes one durable database the application core for court-evidence Google
Timeline data. The engine must excel at analytical SQL (~14 views: overnight, home-base,
schedule-verification, anomaly suites over millions of waypoint rows), first-class
geospatial (geofencing, distance, spatial indexes), transactional integrity (evidence),
and court-defensibility (auditable, mainstream, expert-witness-friendly). The owner's
platform already operates Postgres (data-pg with pgcrypto/fuzzystrmatch/btree_gist,
NocoDB front-end, prior pg_duckdb ruling). Corpus scale: static, batch-loaded,
~low-millions of rows total — not streaming.

## Decision

We use PostgreSQL with the PostGIS extension for geospatial and pg_duckdb for heavy
analytical queries. TimescaleDB is deliberately deferred: it is added only if streaming/
continuous location ingest ever becomes a real workload (trigger condition recorded here);
at current static-archive scale, plain B-tree/BRIN time indexes and ordinary views suffice.

## Alternatives Considered

### PG + PostGIS + pg_duckdb + TimescaleDB (owner's initial proposal)
- **Pros**: purpose-built time-series features (hypertables, compression, continuous aggregates)
- **Cons**: those features pay off at streaming/billions-of-rows scale, not a static
  low-millions archive; Timescale and pg_duckdb both hook the PG planner/executor —
  extension-stacking fragility for zero present benefit
- **Why not**: no workload that needs it; easy to add later, hard to remove later

### SurrealDB (owner floated)
- **Pros**: multi-model flexibility, already deployed on owner's platform (data-surreal)
- **Cons**: young analytical query language on the app's most important layer; basic geo
  vs PostGIS; niche for court-defensibility; owner already hit silent no-op failures with
  it (agno LearningMachine lanes) — silent failure is intolerable for evidence
- **Why not**: JSONB covers the flexibility need; analytics + geo + defensibility all favor PG

### CrateDB (owner floated)
- **Pros**: strong distributed machine-data analytics; PG wire protocol
- **Cons**: not actually PG — no PostGIS/pgcrypto/pg_duckdb ecosystem; historically weak
  multi-row ACID semantics; solves horizontal-scale problems this project will never have
- **Why not**: loses the entire PG ecosystem for capabilities that go unused

### SQLite (prior mature fork)
- **Pros**: zero-ops, the older mature app used it
- **Cons**: no PostGIS-class geo, weak concurrent multi-client access (web UI + CLI +
  agents as peers per ADR-0001), no pg_duckdb
- **Why not**: ADR-0001's API-first multi-client model outgrows it

## Consequences

### Positive
- Maximum analytical + geospatial power on boring, defensible, already-operated technology
- Platform synergy: data-pg infra, NocoDB review front-end, existing forensic extension contract

### Negative
- Analytical views written in PG dialect — engine choice is effectively a one-way door (accepted knowingly)
- Requires a running PG instance (vs SQLite's zero-ops)

### Risks
- pg_duckdb + PostGIS coexistence in one instance is assumed, not yet proven on our
  versions → **test gate**: smoke-test both extensions together during Phase C environment
  setup before schema work begins
- Instance placement (new DB inside existing data-pg vs dedicated instance) unresolved —
  ops sub-question deferred to Phase C setup

**Open remainder of T9**: table design (raw/derived/reference layering, migration tooling).
