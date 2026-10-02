## Coverage & gaps (her corpus, deduped)

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


- Span: **2017-09 → 2024-08** (71 months with data)
- **Missing months (13):** 2017-11, 2017-12, 2018-01, 2020-04, 2020-05, 2020-06, 2020-07, 2020-08, 2020-09, 2020-10, 2020-11, 2020-12, 2021-01
- Thin months (<10 days with data): 2017-09 (3d/25ev), 2018-02 (4d/39ev), 2018-04 (8d/117ev), 2018-05 (6d/47ev), 2018-07 (5d/62ev), 2018-08 (4d/55ev), 2018-09 (4d/25ev), 2018-10 (4d/31ev), 2018-11 (6d/62ev), 2018-12 (5d/36ev), 2019-01 (1d/5ev), 2019-02 (5d/33ev), 2021-11 (6d/62ev), 2023-04 (7d/61ev), 2024-08 (5d/44ev)

## Rounding / precision (cache-key design test)

- Exact distinct coordinates: **6355** → r4 keys: **3014** → r3 keys: **1002** → geohash8: **1970**
- r4 keys covering >1 exact coordinate: **777/3014** (worst key covers 231 distinct coords)
- Interpretation: r4 (~11m) compresses 6355→3014 lookups (52.6% geocode-call savings); raw precision preserved separately (ADR-0009).

## Noise indicators (waypoint stream)

- Duplicate-timestamp waypoint groups: **8504** of 71264 (multi-device signal feed)
- Speed distribution (m/s): median 5.14, p95 23.75, max **427.03** (955 mph max)
- Impossible-speed waypoints (>120mph param): 45 → analysis.speed_anomalies
- Unrecorded-stop candidates (gap ≥15min, moved <100m): **862** → analysis.dwell_stops

## Google's own confidence (probability fields)

- visit: avg 0.867, min 0.450, **16 events below 0.5 confidence**
- activity: avg 0.003, min 0.000, **6356 events below 0.5 confidence**
- Low-confidence events are challengeable — flag before exhibit use.

## Schema drift census (unexpected top-level keys)

- NONE — her corpus matches the expected modern shape exactly.

## Top-10 places by visit count (preview — geocoding will name these)

| place_id | visits | overnights | first | last |
|---|---|---|---|---|
| ChIJezMgkaaHI4gRGrmY… | 823 | 325 | 2018-11-17 | 2022-05-28 |
| ChIJPXrV4aOHI4gR7TEy… | 481 | 2 | 2020-03-06 | 2024-07-25 |
| ChIJe8dWjIKGI4gRziJN… | 462 | 293 | 2021-05-21 | 2024-03-17 |
| ChIJ631fjHqHI4gRW7tV… | 458 | 5 | 2021-05-13 | 2024-07-25 |
| ChIJPXrV4aOHI4gR7TEy… | 386 | 4 | 2018-02-03 | 2020-01-25 |
| ChIJDYibgo-PI4gR52Ln… | 204 | 116 | 2023-09-28 | 2024-04-08 |
| ChIJw6C52r-HI4gRLu_a… | 173 | 45 | 2017-09-23 | 2021-09-01 |
| ChIJg-CPDKOHI4gRO7I4… | 152 | 5 | 2018-04-06 | 2024-07-26 |
| ChIJIzeaNu-PI4gR1RYH… | 142 | 36 | 2023-06-17 | 2024-08-01 |
| ChIJKYO6I4KGI4gRy-3x… | 129 | 71 | 2022-06-25 | 2024-06-29 |

## Raw quarterly-pair comparison (record-level, non-identical pairs)

- `2021-03_2021-06.json.txt`: copyA 905 records, copyB 905 records, common 905, only-A 0, only-B 0
- `2021-06_2021-09.json.txt`: copyA 1225 records, copyB 1225 records, common 1225, only-A 0, only-B 0
- `2024-03_2024-06.json.txt`: copyA 1641 records, copyB 1641 records, common 1641, only-A 0, only-B 0

## Geocode cache vintages (all kept as evidence; import both at Tier-2 setup)

- `radar_geocoding_master_good.csv` [bb0160a176]: 4761 rows/entries — …\traceiq\caches\radar_geocoding_master_good.csv
- `radar_geocoding_master_good.csv` [2aaebe38b2]: 4761 rows/entries — …\traceiq\TraceIQ_Snippets\caches\radar_geocoding_master_good.csv
- `geocoding_cache (1).json` [081e4fb273]: 5902 rows/entries — …\TL (copy 1)\untitled folder\scr\geocoding_cache (1).json
- `geocoding_cache (1).json` [1fc676861a]: 5902 rows/entries — …\traceiq\TraceIQ_Main\geocoding_cache (1).json
