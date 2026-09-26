# TraceIQ Database — Live Schema

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Claude Code · Fable 5 · 2026-07-24 · **This is deployed and running** on data-pg (ovh-data), database `traceiq`, PostgreSQL 18.1 + PostGIS 3.6.4 + pg_duckdb 1.1.0. Test-loaded with 1,641 real records. First verified backup at `E:\TraceIQ_Backups\`._

## The big picture

```mermaid
flowchart LR
    subgraph sources["📁 Source files"]
        EXP["Takeout exports<br/>(her 8-yr · your 14-yr · fragments)"]
    end
    subgraph rawx["🔒 raw — evidence, verbatim, append-only"]
        REC["records exactly as<br/>the JSON has them"]
    end
    subgraph workingx["⚙️ working — fast tables, rebuildable"]
        EV["events + waypoints<br/>with all basic derivations"]
    end
    subgraph analysis["📊 analysis views"]
        V["overnight · home-base ·<br/>anomalies · schedule checks"]
    end
    subgraph outputs["📄 outputs"]
        SNAP["snapshots · CSV · maps ·<br/>Evidence.dev reports"]
    end
    EXP -->|"hash + register"| REC
    REC -->|"documented, versioned<br/>transformations (for the judge)"| EV
    EV --> V --> SNAP
    GEO["🌍 geo — N providers,<br/>results never merged"] --> EV
    REF["📋 ref — your case knowledge<br/>+ tunable parameters"] --> V
```

## Evidence chain (raw schema)

```mermaid
erDiagram
    SUBJECT ||--o{ ACCOUNT : "owns"
    ACCOUNT ||--o{ EXPORT : "produced"
    ACCOUNT ||--o{ DATA_LOSS_EVENT : "suffered"
    EXPORT ||--o{ CHUNK : "split into"
    CHUNK ||--o{ RECORD : "contains"

    SUBJECT {
        int subject_id PK
        text display_name "her / you"
        text role
    }
    ACCOUNT {
        int account_id PK
        text google_account
    }
    EXPORT {
        int export_id PK
        text format_generation "legacy / modern / other-geo"
        date produced_at "when Google made it"
        date acquired_at
        text acquisition_method "the admissibility record"
        text acquisition_basis
    }
    CHUNK {
        int chunk_id PK
        char file_sha256 "file fingerprint"
        text archive_path "content-addressed archive"
        text parse_status "fail-loud: never guesses"
    }
    RECORD {
        bigint record_id PK
        char record_sha256 "per-record fingerprint"
        text google_type "exactly what Google called it"
        jsonb raw_json "VERBATIM - never edited"
    }
```

## Working layer + geocoding

```mermaid
erDiagram
    RECORD ||--o{ EVENT : "derived by versioned transformation"
    EVENT ||--o{ WAYPOINT : "ordered by sequence"
    BUILD ||--o{ EVENT : "stamped by"
    LOCATION_KEY ||--o{ GEOCODE_REQUEST : "looks up"
    GEOCODE_REQUEST ||--o| RESULT_RADAR : "primary"
    GEOCODE_REQUEST ||--o| RESULT_GOOGLE : "fallback"
    GEOCODE_REQUEST ||--o| RESULT_GEODATA : "third"
    GEOCODE_REQUEST ||--o| RESOLUTION : "disagreement-flagged"
    PROVIDER ||--o{ API_CALL : "every call saved"
    PROVIDER ||--o{ PROVIDER_CREDENTIAL : "editable keys"

    EVENT {
        bigint event_id PK
        text event_type "visit / activity / path / point"
        text raw_type "breadcrumb (trip stays visible)"
        timestamptz start_utc
        timestamp start_local "where the phone was"
        timestamp start_eastern "for humans"
        text path_time_source "3-tier inference tag"
        numeric lat_lng "full precision, never rounded"
        geometry geom "PostGIS point"
        bool overnight_simple
        bool multi_device_split
        text serial_display "speakable row label"
    }
    WAYPOINT {
        bigint waypoint_id PK
        int seq "NEVER ordered by timestamp"
        numeric speed_mps
        bool dwell_candidate "unrecorded-stop signal"
    }
    BUILD {
        int build_id PK
        bigint raw_count "reconciliation:"
        bigint derived_count "raw = derived + exceptions"
        bigint exception_count "provable, every rebuild"
    }
```

## What's in each namespace (32 tables live)

| Schema | Tables | Job |
|---|---|---|
| **raw** | subject, account, export, chunk, record, data_loss_event | Evidence. Verbatim, append-only, hashed. The only layer that can't be regenerated — everything else derives from it. |
| **working** | build, event, waypoint, exception | Your fast day-to-day tables. Rebuilt from raw by documented transformations; version-stamped; reconciliation counts prove nothing was dropped. |
| **geo** | provider, provider_credential, credential_audit, location_key, geocode_request, result_radar, result_google, result_geodata, resolution, snap_result, api_call, legacy_radar_cache | N-provider geocoding. Results never merged — cross-validated with disagreement flags + manual override. Every API call saved (it's evidence). Keys editable in-table, never in code. |
| **ref** | parameter, parameter_audit, home_base, expected_schedule, problematic_location, claim | Your case knowledge + every tunable (overnight window, thresholds) with audit trail. Exhibits record the parameter values they used. |
| **ops** | job, transformation, snapshot, backup | The machinery: job queue, transformation registry (judge-readable doc per transformation), snapshot registry, backup registry. |

## Seeded and verified tonight

| Item | Status |
|---|---|
| PostGIS 3.6.4 + pg_duckdb 1.1.0 coexistence (ADR-0002 test gate) | ✅ both live in `traceiq` DB |
| 6 tunable parameters seeded (overnight 22:00–07:00, multi-device 100m, speed 120mph, dwell 15min, cluster 200m, anchor ±5min — your recovered historical defaults) | ✅ |
| 6 providers seeded (radar, google, geodata enabled; osrm, valhalla, mapbox staged for road-snapping) | ✅ |
| **Test load: real corpus chunk `2024-03_2024-06.json.txt`** | ✅ 1,641 records — 626 visits, 516 activities, 498 timelinePaths, **1 `timelineMemory`** (the rare trip type — fail-loud caught it, exactly as designed) |
| First backup, hash-registered + restore-verified | ✅ `E:\TraceIQ_Backups\traceiq_2026-07-24_milestone-schema-init-testload.dump` |

Connection (tailnet): `host=100.119.96.29 port=5432 dbname=traceiq` · repo: `E:\AI_Workspace\Projects\traceiq-rebuild` (migration `db\migrations\0001_init.sql`, ADRs in `docs\adr\`)

Test data is clearly labeled (subject `TEST`) and rebuilds freely — real ingest starts with her export under proper provenance rows.
