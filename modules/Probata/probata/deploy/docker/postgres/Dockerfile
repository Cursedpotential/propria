# =============================================================================
# Custom PostgreSQL 18 image — the extension contract (handoff §10.2, ADR-0013)
# Original byline: Claude Code · Kimi K3 (drift-fix) · 2026-08-12.
# Comment amendment: ChatGPT · GPT-6 Astra · 2026-10-17; D-080/D-107/D-151,
# root 2026-10-05 operating-mode contract and source read at 68f369e.
# Image, packages, user and startup command are unchanged by this amendment.
# =============================================================================
# Base: official pg_duckdb image (PG18 + DuckDB engine embedded in Postgres —
# owner decision 2026-06-10, supersedes ADR-0003's no-DuckDB stance).
# pg_duckdb also gives Postgres native S3/R2 reads via DuckDB httpfs.
#
# Added on top: PostGIS (geospatial) + pgvector (embeddings) from PGDG.
#
# pg_textsearch (BM25): staged, NOT baked yet — no PGDG package; add the
# vendor build step here when retrieval quality demands BM25+RRF (§10.4).
#
# Volume note: initialization SQL runs only on an empty data directory. Image
# changes do not authorize replaying retired numbered migrations or resetting
# an existing volume. Current schema instructions are in sql/bootstrap/README.md;
# current data operations follow the Propria root's 2026-10-05 single-case contract.

FROM pgduckdb/pgduckdb:18-v1.1.1

USER root

# --- Geospatial + vector types (proven, baked) --------------------------------
RUN apt-get update && apt-get install -y --no-install-recommends \
        postgresql-18-postgis-3 \
        postgresql-18-pgvector \
    && rm -rf /var/lib/apt/lists/*

# --- Cross-source reach and governed analytical roles ------------------------
# ADR-0032 retired the custom FDW-hub proposal; its old implementation plan and
# intermediate architecture claims remain in the dated ADRs and Git history.
# This image uses the pg_duckdb base and PGDG extensions above, without that hub.
#
# PostgreSQL remains canonical (D-080), with pg_duckdb for analytical/object-store
# access. Weaviate is the project search projection, Neo4j the semantic graph,
# and SurrealDB the governed analytical/walk target (D-107/D-151). Only the old
# Agno operational Surreal adapter/instance was retired, not that analytical role.
# Temporal owns durable execution; n8n owns the visual business/integration flow.
#
# Source configuration does not prove which extensions are enabled or which
# integration paths work on a running database; verify those on the target.
# core postgres_fdw + file_fdw still ship with the base image if a plain
# Postgres-to-Postgres link is ever needed (CREATE EXTENSION only, no build).

USER postgres

# pg_stat_statements ships with core PG; it needs shared_preload_libraries.
# pg_duckdb already requires its own preload, so chain both.
# duckdb.postgres_role: without it only superusers may run DuckDB, and the
# engine connects as platform_runtime (member of platform_duckdb — created by
# scripts/duckdb_runtime_role.sql). Postmaster-context: needs a restart.
# Claude Code · Fable 5.1 · 2026-09-20
CMD ["postgres", "-c", "shared_preload_libraries=pg_duckdb,pg_stat_statements", "-c", "duckdb.postgres_role=platform_duckdb"]
