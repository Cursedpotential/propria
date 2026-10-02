# Architecture Decision Records — Vestigia (traceiq-rebuild)

> _Byline: Claude Code · Opus 5.5 · 2026-10-02 (index re-conformed: 0015 and 0016 added, the
> pre-repository location note removed). Product name per D-140; old name TraceIQ._

**Owner direction 2026-10-02:** PostgreSQL is the source-of-truth data holder only and all
analysis moves to SurrealDB on a mirror. That changes 0002, 0012 and 0013; the superseding ADR
follows the open choices in `../plans/2026-10-02-continuation-plan.md` (V-7, V-8).

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-database-centric-core-api-first.md) | Database-centric core with API-first access (CSV pipeline retired) | accepted | 2026-07-24 |
| [0002](0002-postgres-postgis-pgduckdb-engine.md) | PostgreSQL + PostGIS + pg_duckdb as the evidence-core engine | accepted | 2026-07-24 |
| [0003](0003-intake-provenance-discipline.md) | Intake discipline and source provenance (subject→account→export→chunk→record) | accepted | 2026-07-24 |
| [0004](0004-placement-datapg-local-backups.md) | Placement on data-pg with milestone-triggered local backups | accepted | 2026-07-24 |
| [0005](0005-verbatim-raw-plus-documented-derivations.md) | Verbatim raw ingest + documented derivation layer (event model) | accepted | 2026-07-24 |
| [0006](0006-content-hash-identity.md) | Content-hash identity for evidence; surrogates for derivations; serials for display | accepted | 2026-07-24 |
| [0007](0007-materialized-working-layer.md) | Materialized working layer (physical Tier-1 tables, not views) | accepted | 2026-07-24 |
| [0008](0008-tunable-analysis-parameters.md) | Dual-clock time + tunable, audited analysis parameters | accepted | 2026-07-24 |
| [0009](0009-coordinate-representation-multidevice.md) | Coordinate representation per layer; multi-device carried + upgradeable | accepted | 2026-07-24 |
| [0010](0010-pipeline-jobrunner-full-rebuild.md) | Built-in job runner, dependency-graph stages, full rebuilds | accepted | 2026-07-24 |
| [0011](0011-geocoding-3source-snap-provider-agnostic.md) | N-provider geocoding blueprint; lean cache rows; provider-agnostic road-snapping | accepted | 2026-07-24 |
| [0012](0012-analysis-waves-homebase-durability.md) | Analysis first wave (overnight/home-base + anomalies); home_base durability | accepted | 2026-07-24 |
| [0013](0013-fresh-thin-web-ui.md) | Fresh thin web UI over the API; Evidence.dev reporting layer; old UIs mined | accepted | 2026-07-24 |
| [0014](0014-provider-configuration-subsystem.md) | Provider configuration subsystem; no hardcoded credentials | accepted | 2026-07-24 |
| [0015](0015-dual-use-agent-native-workspace.md) | Dual-use agent-native workspace (unified surface) | proposed | 2026-07-24 |
| [0016](0016-path-model-overnight-verification-scope.md) | Path segmentation, overnight definition, verification scope, "geodata" clarified | accepted | 2026-07-25 |
