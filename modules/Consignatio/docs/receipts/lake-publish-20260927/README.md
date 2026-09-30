<!-- tags: consignatio, casebible, catalog, raw_duck, lakehouse, b2, parquet, receipt, lake-publish -->
# Lake publish 2026-09-27: the catalog on B2 as Parquet

> _Byline: Claude Code · Opus 5.5 · 2026-09-27 (agent for the Fable 5.1 supervising session)_
>
> Owner, 2026-09-27 00:09 EDT: "B2 is the canonical home, and that's where the index is supposed to be. That's what's supposed to be cataloged. That's what's supposed to be the lakehouse." 00:14 EDT: "Finish creating the lakehouse."
>
> This is the lake publish planned in the 2026-09-15 log section ("publish the catalog tables + corrupt_missing.csv … under `_system/lake/`"). Log entry: `docs/URGENT-TODO.md`, 2026-09-27.

## Result

- **Where:** `b2://salem-data/consignatio/_system/lake/2026-09-27/` (105 objects). The pointer `consignatio/_system/lake/LATEST` contains `2026-09-27`.
- **What:**
  - 102 tables from PG `casebible.raw_duck` (ovh-files), as Parquet with zstd compression.
  - `corrupt_missing.csv`.
  - `schema.json`: column names, PG types, Parquet types, keys, casts and the 9 view definitions.
  - `manifest.csv`.
- **Size:** 106 objects, 1,523,156,098 bytes. The Parquet part is 19,417,723 rows in 1,522,550,398 bytes.
- **Catalog record:** `raw_duck.lake_publish_20260927`, 106 rows: every object with rows, bytes, B2 key, sha256 and published_at. The same rows are in `manifest.csv` on B2.
- **Cost:** 1.52 GB at B2's list price ($6.95/TB-month, checked 2026-09-27) is about $0.011 a month.
  - Uploads and Class A/B/C API calls are free on pay-as-you-go.
  - The two verification downloads (about 3 GB) fall inside the free egress allowance (3× the stored data).
- **Add-only:** every object was written once with `rclone --immutable`. Nothing that existed on B2 was moved, renamed or deleted; `consignatio/_system/` did not exist before 00:50 EDT.
- **Script:** `casebible/tools/lake_publish_20260927.sh`, plus `lake_publish_20260927.sql` (the catalog table) and `lake_publish_20260927.tables.txt` (table list, decision, status, reason).
- **Run directory:** `ovh-files:/data/consignatio/lake-publish-20260927/` (logs, `export_results.tsv`, `readback_results.tsv`, `s3probe_results.tsv`, `manifest.csv`).

## Verification

Times are UTC; EDT is UTC−4.

1. **Export, 04:37–04:49.** For each of the 108 candidate tables, in order:
   - PG `count(*)`;
   - `COPY … TO parquet` through pg_duckdb 1.1.0 (DuckDB v1.4.3, one thread);
   - `read_parquet` count on the file;
   - PG `count(*)` again.

   All three counts are equal for 108 of 108 tables. 102 are published; 6 were excluded afterwards (list below).
2. **Upload, 04:50:45–04:51:25.** 104 objects, 1,523,134,095 bytes, 0 errors. Every object is under 200 MiB, so each went up as a single part and B2 verified its SHA-1 on receipt.
   - `rclone check` with SHA-1: 104 matching, 0 differences.
   - `rclone check --size-only`: 104 matching, 0 differences.
   - Independent listing: all 104 objects carry a SHA-1 on B2 equal to the local SHA-1 and size.
3. **Readback, 04:52:07–04:53:32.** All 104 objects were downloaded from B2. For each:
   - sha256 of the downloaded bytes equals the export sha256: 104 of 104;
   - Parquet rows counted by pg_duckdb on the downloaded files equal the PG rows: 102 of 102 tables, 19,417,723 rows in total.
4. **Finalize, 04:53:46–04:54:14.**
   - `manifest.csv` and `LATEST` were written last (`--immutable`) and read back: `LATEST` = `2026-09-27`, and the manifest sha256 matches.
   - `raw_duck.lake_publish_20260927` was loaded with 106 rows. All 106 keys exist on B2 with the same size, and no object under `_system/lake/` is missing from it.
5. **S3 read path, 04:56:17–04:57:06.** All 106 objects were read over B2's S3 API, which is the path DuckDB httpfs and Evidence.dev use:
   - endpoint `https://s3.us-west-004.backblazeb2.com`, path-style;
   - boto3 on ovh-files, application key `B2_KEY_ID` from `~/.secrets/backblaze.env`.

   106 of 106 match on sha256 and size.

## How to read it

With DuckDB (httpfs), using an S3-capable B2 application key:

```sql
INSTALL httpfs; LOAD httpfs;
CREATE SECRET lake (TYPE s3, KEY_ID '<key id>', SECRET '<application key>',
                    REGION 'us-west-004', ENDPOINT 's3.us-west-004.backblazeb2.com', URL_STYLE 'path');
SELECT * FROM read_csv('s3://salem-data/consignatio/_system/lake/2026-09-27/manifest.csv');
SELECT count(*) FROM read_parquet('s3://salem-data/consignatio/_system/lake/2026-09-27/source_occurrences.parquet');
```

The endpoint, key and paths were proven by step 5. This DuckDB snippet itself was not run: no DuckDB with httpfs exists on ovh-files, and httpfs was deliberately not installed into the production PG.

Reading notes:
- **What is in the vault now:** `vault_objects_20260916_r4`, minus the 49 keys in `vault_onecopy_pilot_delete_20260916`, with the 19 moves in `vault_moves_20260924` applied. The 19 are superseded SMS backups moved on 09-24 to `intake/_quarantine/superseded-sms-backups/v1/`.
- **Deleted path → kept path:** `vault_delete_v7` joined to `vault_keep_v7` on `canonical_key`.
- **Occurrence → vault object, two routes, both published.**
  - Route B: `intake_catalog_fs_20260917.vault_key`, one key per occurrence. `vault_index_source_20260918` is built on it.
  - Route A: `vault_occ_v1` (md5+size → `canonical_key`), then `vault_keep_v7.dest_key`. Route B does not supersede it.
- **Bridge tables:** `b2_content`, `b2_objects` and `graded_carriers` carry keys in the intake area that was emptied on 09-16. Use them as the md5 → key → sha1 bridge, not as a listing.
- **Old table names:** receipts written before 09-24 use `chat_*` names. `catalog_renames_20260924` maps them to the current `msg_*` / `comm_*` names; for example, `chat_conversation_registry_20260924` is now `msg_conversation_registry_20260924`.
- **NUMERIC casts:** pg_duckdb cannot read NUMERIC without precision, so it reads those columns as DOUBLE. There are 35 such columns in 19 tables, all byte counts, ratios or GB figures:
  - a column whose values are all integral and below 2^53 was written as BIGINT, which is exact;
  - any other such column was written as DOUBLE.

  No hash, id or key column was cast. Each cast is listed below and in `schema.json`.
- `intake_fs_ops_20260917.at` is a DuckDB keyword. A one-row cross join kept the column qualified during export; its name and values are unchanged.

## What was not published

**Excluded after the dry run (6).** Their exports are kept at `ovh-files:/data/consignatio/lake-publish-20260927/excluded-2026-09-27/`.
- `vault_twins_dir_files` (5,693,882 rows) and `vault_twins_group_files` (1,806,413 rows): bulk intermediates of the stale twins lane.
- `r2_files`: the 08-30 R2 listing, called scratch by the 09-18 audit. The R2 catalog is published as `reconcile_r2_occurrences_20260920`.
- `recovery_manifest_20260917`: an analysis-only plan that was never materialized.
- `source_mount_candidates` and `intake_versions_differ_20260916`: working analysis tables.

**Never candidates (73 tables).**
- **Scratch (42),** per the 09-18 catalog audit: `arch`, `base`, `base2`, `cand`, `copy_manifest`, `cube`, `dated`, `dec4`, `decisions`, `dirs`, `final_survivors`, `frem`, `in_sorted`, `junk`, `junk4`, `keyed`, `local_files`, `losers`, `mcut`, `media`, `mfolder`, `ns_canonical`, `ns_unique`, `paired`, `r2_inv`, `rec_all`, `rec_surv`, `remaining`, `resolved`, `scored`, `scored2`, `scored4`, `side`, `sorted_best`, `sorted_best4`, `staged`, `superseded_in_sorted`, `surv4`, `survivors`, `survivors2`, `t`, `to_copy`.
- **Plan versions (15):** `vault_place_v1`–`v6`, `vault_occ_v0`, `vault_content_v0`, `vault_keep_v6`, `vault_mounts_v4`–`v6`, `vault_copy_manifest_v6`, `vault_units_v1`, `vault_unit_pairs_v1`.
- **Superseded listing revisions (13):** `vault_objects` (08:10, before the dedupe), `vault_objects_stage`, `vault_objects_20260916_{r2, r3, post_move, post_move_r2, post_move_r3, post_prune}`, `intake_objects_20260916_{before, post_clean, r2, r3, r4}`.
- **Staging (3):** `msg_bout_labels_stage_20260924`, `sms_backup_headcheck_stage_20260924`, `vault_moves_stage_20260924`.

**Views (9)** are not exported. Their definitions are in `schema.json`: `msg_events_20260918`, `ai_chat_events_20260918`, `msg_bout_observations_cited_20260924` and `timeline_*_20260918` (6).

**Other schemas** in `casebible` (`catalog`, `inventory`, `exports`, `recovery`, `evidence`, `analysis`, …) are older lanes outside `raw_duck` and were not assessed.

## Record notes

- At export time `raw_duck` held 190 relations (181 tables, 9 views), 43 GB. It is 191 relations with the new manifest table.
- `docs/receipts/` has no README map. The map of where file-truth lives is the 2026-09-15 section of `docs/URGENT-TODO.md`, and this publish is added there as item 8.
- The published `schema.json` says it was generated by "Claude Code · Fable 5.1", the byline the brief specified. The agent that ran the publish is Opus 5.5; the object is write-once and was not changed.
- ovh-files has no rclone remote called `b2:`. The upload used `b2native-full:` (native B2 API). The `b2s3:` remote answers 403 "not entitled".
- The Probata planning file `docs/planning/2026-09-27-TODO.md` #4 pointed Evidence.dev at `b2:salem-data/_system/lake/`. It is corrected to `b2:salem-data/consignatio/_system/lake/`.
- **Local copies on ovh-files, kept for the owner to decide on** (`/data/consignatio/lake-publish-20260927/`):
  - `2026-09-27/`: the published bytes, 1.5 GB.
  - `readback-2026-09-27/`: the copies downloaded from B2, 1.5 GB.
  - `excluded-2026-09-27/`: 200 MB.
  - `test/`: the first mechanics test, 136 KB.

## Not verified

- A DuckDB httpfs read of the lake. The S3 transport, endpoint and key were proven with boto3 (step 5).
- Whether the `B2_KEY_ID` application key is read-only. Evidence.dev is meant to get a read-only key.
- Cell-by-cell equality between PG and Parquet. Row counts match and the files on B2 are byte-identical to the exports; values were not compared one by one. The 35 NUMERIC columns carry the documented casts.
- Anything outside `raw_duck`.

## Manifest (the same rows as `manifest.csv` and `raw_duck.lake_publish_20260927`)

Every table below was checked as follows:
- PG rows before = Parquet rows at export = PG rows after = Parquet rows read back from B2;
- the sha256 was read back over the native API and over the S3 API.

`manifest.csv` also carries `b2_key` (`consignatio/_system/lake/2026-09-27/<table>.parquet`), `published_at` (04:51:25 UTC for these objects), `status` and `object_kind`. The catalog table also records the rows for `manifest.csv` and `LATEST`.

`*.csv` is git-ignored in this module, so the CSV copy of this manifest sits with the other receipt payloads at `ovh-files:/data/consignatio/receipts/lake-publish-20260927/manifest.csv`. Its sha256 matches the object on B2.

### `current` (59 tables, 9,596,924 rows, 668,373,955 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `intake_catalog_fs_20260917` | 1,567,456 | 194,834,437 | `0082ecca50237f61a301338b90253449400fbe2e1d3e2bafd30f468f36fdf520` |  |
| `source_occurrences` | 1,567,456 | 147,484,259 | `93aeeb1b4b9b2507c094dadd279967cb715f1976013c638bf7b2b5ed8ee98acc` |  |
| `vault_index_source_20260918` | 508,152 | 94,312,545 | `693a4991bf32189825cbb6e4760e2bf8578bda65e65025146e74126de41cdf73` |  |
| `comm_events_20260918` | 551,877 | 48,145,567 | `7b4a65c5826581393d0e9b5de784db6b7d045286e7bf7e6bacd50ee4492a824f` |  |
| `comm_event_provenance_20260918` | 1,080,505 | 39,431,095 | `2b6b55eaa8994ce97f7337b32919ca656225869fc9f451b24601faef14948eb7` |  |
| `msg_extract_rows_20260924` | 254,033 | 19,925,944 | `58cc52219293929391520b960b037dc721f56c283a35c647ff8fef0efbe5579b` |  |
| `comm_dir_files_20260918` | 893,619 | 19,185,813 | `926b9e3adeec09d587979a9a99798bcc16d58662b7410d515a0764462c99a45a` |  |
| `graded_selection` | 327,600 | 18,863,588 | `18ed31daf455c7482c9a7a82c0597370c42e54a6ae32ef3d3f15b80855ac3319` |  |
| `vault_objects_20260916_r4` | 508,201 | 17,992,690 | `f22ec8ea21fc05fa28fb41eb499be5cff2127a2513d224a5cc6e0f0a628b5f74` |  |
| `tg_files` | 508,201 | 17,903,420 | `d4d3aaab209cbc3e6cd1a14f4b4a0f5612583790c4a3ad416f25bd0be77f8204` |  |
| `export_unit_members_v1` | 499,712 | 11,837,284 | `19721e5936477507b09a3914e9eea3f4c986717e31bd55d7fcb52df7272a3beb` |  |
| `msg_norm_20260924` | 135,685 | 5,557,441 | `282f0b643d924924a8bcc2683573aa4b0604bb25b55247bc03422858c6f44dde` |  |
| `tg_node_content` | 249,162 | 5,421,364 | `27700734d6bed65b4e3dd9d9564d2b824f039b174b23f37d7f8d51823034b90c` |  |
| `ai_chat_probe_20260918` | 104,740 | 5,356,439 | `3cac8a98b60622848a4f2841234957efbb29bd70efebc1860eb06e8aacde0a61` |  |
| `msg_bout_messages_20260924` | 135,629 | 3,144,733 | `f4b706cc0fa130a48b4562d95e5c03eacc20475653708d528a9c62b90246d6dd` |  |
| `msg_files_20260924` | 135,629 | 2,837,787 | `67af6f0172c0a91f1c52e5644a153d21858bd4ff8fbb56830e2cb22d7e6ea772` |  |
| `intake_catalog_dirs_20260917` | 79,511 | 2,535,490 | `60d7a7312d5e107be86b592c7ee44ed2f49a7f3630b1ce7b0fcc127641c72ff0` | nested_bytes:numeric->bigint |
| `atomic_unit_members` | 128,834 | 1,988,056 | `2d667fcf2df517439ea533d0e7dc6f399585ea92e95b044ef7b397df3da0aa12` |  |
| `backup_message_records_20260920` | 11,678 | 1,829,766 | `ea8798982493b2ece2d44c7a6a98d5d5e9ebebc5dbf362034d7cddd08e9865c8` |  |
| `recovery_integrity_20260916` | 40,304 | 1,419,628 | `04ece649bec5d58659a9b15575922995f339e706432c3f9360d268c2226986c9` |  |
| `msg_corroboration_20260924` | 32,359 | 1,310,285 | `6a70e394d59deab617aba9b8095a26f0ae6242f0382ccf5de534d8da287ffe5a` |  |
| `recovery_dump_files_20260916` | 40,304 | 1,264,972 | `c819f218f0496dca21d05a06ca17a6182c44c7f6268bc05f9b023a49db396170` |  |
| `msg_cross_device_gaps_20260924` | 18,913 | 945,422 | `b53a7c920fb28c175eac1ab56178f24dd4846155e5b1d523279d7ab93b35a587` |  |
| `corrupt_recovery` | 47,058 | 661,112 | `006a420302639688e3e52e69fd621c9017457a71cf466859836c8578ec0d7308` |  |
| `integrity_hold` | 33,685 | 614,438 | `4196db619330b0dafdaffc589e09f2f212b0f42fe9038205f8fee634c5219127` |  |
| `backup_message_occurrences_20260920` | 46,207 | 503,128 | `3a8ddcc0dc248915da25820a9f592215ca1fa273e517e00584edb330222b4ff1` |  |
| `comm_directories_20260918` | 25,596 | 428,171 | `11b6a9d3ff155ed020f1659144c123ded6a04c2a3e276855f611f003e69f7254` | bytes:numeric->bigint, readable_bytes:numeric->bigint |
| `msg_export_files_20260924` | 4,794 | 424,383 | `60b9468d2b7c85aa5d3917348517eedcd38969f9f642660b559ce0a0203978bc` |  |
| `tg_dirs` | 32,213 | 392,431 | `7c03dcf237c3f7b3f5fd3f061e1d7430d4bf81c37be602d565623ea82aef4096` | bytes:numeric->bigint |
| `sms_vanished_messages_20260924` | 7,051 | 347,646 | `3d57b4444c40e0119ac69a9699ddf75ff7a9397b30442b9c3e55bb2eac5562c5` |  |
| `comm_candidates_20260918` | 6,431 | 312,809 | `dce26502b85d45bf5d4bd5256a5f10af4ebc537d674dc9583169ca468c66d845` |  |
| `msg_bout_labels_20260924` | 1,290 | 284,749 | `102ecdeac18e4e1c01833e4adaa96ae879afb0d751636a9210c773c7453c6b4d` |  |
| `msg_bouts_20260924` | 8,283 | 201,085 | `e14f39985b008b33f27268e5d6b751122cc6dadc717af6b13273bc1a9125f03f` |  |
| `r2_candidate_preservation_20260920` | 648 | 164,167 | `5974a0da1c97887db1c63a05c04669a23f5e9722de18d8bdb23e79aa8ab3848a` |  |
| `recovered_artifacts_20260920` | 323 | 136,486 | `5104cdf5eced40867645fccd19454ff1098a67c74e439f92059edb161681a750` |  |
| `msg_extract_attempts_20260924` | 6 | 76,318 | `e29277daec938ba6da755489050b19d39f68cb50b2d385abd89de2a275a56856` |  |
| `msg_extract_lineage_20260924` | 659 | 65,301 | `8bd241325fcc50a098477b94b1395d180022ebf3209bf28fedea3f6860f65b1b` |  |
| `recovered_occurrences_20260920` | 1,449 | 62,900 | `546189272b76d64cd99c7919d2e852fc2d3fe37e34965d3e1ff7a8be54b02028` |  |
| `atomic_units` | 608 | 59,060 | `473bc4003bc599a97b3375578806e357ae3f05b70472c862e1b17cf1a55fae29` | total_bytes:numeric->bigint |
| `evidence_quality_20260920` | 4 | 20,961 | `559c3bed899e2a73e41b745729bbbb71431ec3a84b00b2148369aa3a7bbf6403` |  |
| `export_units_v1` | 542 | 20,143 | `c4ee2ee1e79a2d9d62c780abc168c23a9336ee0fa3f5f4192dea8bcefba98e77` | members:numeric->bigint, bytes:numeric->bigint |
| `msg_extract_warnings_20260924` | 113 | 11,643 | `309700f3d9a70a5f5e81acc374747292a5ea5f7e7ec313d96fbaa8a8f2eb9285` |  |
| `missing_message_payloads_20260920` | 16 | 6,842 | `1cd03441d2cbbe95180e0735df35d22e159a3691765be7150945cd507e685aef` |  |
| `intake_fs_ops_20260917` | 5 | 6,051 | `a6462d33cf3255426d8325f61b989928323cb1de0cbf88790d9216de3793ab26` |  |
| `export_unit_folds_v1` | 53 | 5,716 | `4d7f86bd840d0a0af8e16085916c3ad9ddebbfa5134e6ba2dc043e2ff9c9a516` | bytes:numeric->bigint |
| `backup_manifests_20260920` | 4 | 4,870 | `4cae01b9d7e7601e1def32709362e49393bcb0bfa91a27f1f168aba6a422cd42` |  |
| `sms_backup_coverage_20260924` | 45 | 4,831 | `af0a69c92ca870ae9b8cde1b5b1e8f1630cfa29483b2c8db3050aeac9d4725e7` |  |
| `sms_backup_headcheck_20260924` | 45 | 3,868 | `1df085d3fa8d029bac19817804d3f758c64388dd57e95e781ef09e7a890b1ab9` |  |
| `catalog_renames_20260924` | 98 | 3,803 | `1005c1a969242ee3c9a7096561b4376539439595a21fe6eb620d96311a601a8a` |  |
| `msg_identity_20260924` | 39 | 3,714 | `730f36e4686237a56deaabdfb53527ca1d0052661d50ee24db2136057aa44ad1` |  |
| `msg_conversation_registry_20260924` | 4 | 3,378 | `2b89280b4a4c0f7be39d2546a151178b68c480cd8826d0555791115988337142` |  |
| `backup_containment_20260920` | 3 | 3,072 | `05998801dcf874181436fb42c8161d560bd8ce856a9738d7c3608eb9f9e8a478` |  |
| `analysis_preservation_20260920` | 1 | 3,049 | `21decf5e8cdbf9044db71f309823970bea7ba07b79b97b0862f59aa804fbc6a0` |  |
| `recovery_holds_20260920` | 1 | 2,738 | `7535701ae9966be2a47de51a5ab594c00e76993205b599b1e3c98f62ce263b37` |  |
| `tg_nodes` | 41 | 2,346 | `963d0e8f8bc4207afd8ed7dcad5ee7c9aade19f9a16b39a2182f8df1724cf130` | bytes:numeric->bigint |
| `msg_matt_lines_on_her_phone_20260924` | 24 | 2,044 | `ccdd9027fb3abce3ac1d8a4308b2abd6d7bdb140f7bbd3fea1c2164bc36e9340` |  |
| `tg_edge_contains` | 14 | 1,195 | `399f16736cb65a6f6c1252d2972f5341e48347302c9d26af4d44d2cbafbc10dd` |  |
| `tg_edge_overlap` | 4 | 874 | `54997fca3d217154148beb3bc1c679c4b8047635aaeda9127200175468c7e9de` | pct_of_a:numeric->double, pct_of_b:numeric->double |
| `tg_edge_home` | 7 | 608 | `838e5550a02142257851f01701305d43bcc724ef85947253342012764eb3b736` |  |

### `lineage` (12 tables, 2,212,421 rows, 79,630,151 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `vault_delete_v7` | 1,172,978 | 61,861,463 | `5cda7ff1b8bdb6d8dbae9a8ea2fb76e4687d8175782a269b5cb84f1943535ca1` |  |
| `vault_keep_v7` | 504,458 | 11,921,497 | `f6679d2faba0f09e0c77c96a74f429be4d41c53044cd0b53ec75b6cda8876fa8` |  |
| `intake_delete_20260916` | 526,393 | 5,480,188 | `0bb375d8a78b642761b71a0175abf5ecc00baa5ac7ec5028ba46d2f63e715a45` |  |
| `vault_move_rest_20260916` | 3,683 | 187,249 | `5ec51fa34ec7eb6d747a0b07ac50c6be0cfcfaa0493716e4556311bf35f43068` |  |
| `intake_moved_delete_20260916` | 3,683 | 129,605 | `0c468265835ae841e501573d5a515730906b3c3888ba3764aaf9c12aa8eb3ecf` |  |
| `vault_onecopy_extra_20260916` | 1,128 | 25,659 | `454f2bede603e4d8b25e28979f99318d29b03ace4e08f3a36d1ea870e54f09f9` |  |
| `vault_onecopy_pilot_delete_20260916` | 49 | 6,774 | `4d3a5e61d09817e3d543361f8e16f45c03f33bb0945ee98e7a2518ff5ca51d46` |  |
| `vault_move_rest_20260916_r2` | 14 | 5,240 | `7084ea1a0b46f4c8f576e2a95d7a97be32e926cc32db2f80986542094a03bee8` |  |
| `vault_moves_20260924` | 19 | 4,802 | `b996bc191c0dfe4e814962089831cdb92aeac6f1a1635053dbb1a0886de9cda3` |  |
| `intake_moved_delete_20260916_r2` | 14 | 3,306 | `0dc6ef59cc55e7b2ee0cdb70bd2901aef47a54310d2699c2b66c07b4c65ff907` |  |
| `vault_move_rest_20260916_r3` | 1 | 2,818 | `5e4e82e3984f23f71b5d4fd33d3dc02bd72b4f032d246b90cc3396e570d6c9f9` |  |
| `intake_moved_delete_20260916_r3` | 1 | 1,550 | `8869cb9501abd8eaa4be09a1b017c3ed5f105ae841329a163edc2ee716fc03d0` |  |

### `bridge` (3 tables, 1,265,552 rows, 49,211,787 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `b2_content` | 504,482 | 22,161,423 | `75dbb0e47d79d1d7cfa2d6753263fd43432bd4ec777ecf512970a99b4bdd93aa` |  |
| `b2_objects` | 530,070 | 18,547,974 | `8c7dbfff01ba8c8697db21b16bced86ce2d9cacb84d7c35de63bb90959a6b24e` |  |
| `graded_carriers` | 231,000 | 8,502,390 | `98fde2d25b626f09439eb608a60341b0e26856cd17c67037f46d881cde489c4c` |  |

### `reconciliation_20260920` (9 tables, 4,193,343 rows, 665,396,064 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `reconcile_occurrences_20260920` | 1,567,456 | 355,091,427 | `ad0f790054b35ab9b0c29bda605dc520dd43d6bede9658415918b088f48f276e` |  |
| `reconcile_r2_occurrences_20260920` | 1,279,556 | 178,558,683 | `531b75a1a002c2416e4f92a14e2d7c21c4e915cf35bf2e47487eb7ec153d71cc` |  |
| `reconcile_objects_20260920` | 568,069 | 72,242,802 | `3f2fe755834af9587afe3d48d7044ebe6a5798c2b9f000ea8b2edbf8422b1cc5` |  |
| `reconcile_packages_20260920` | 630,407 | 39,070,920 | `5763e2505b7fafcdd95992253083c948138c9b72498ad0778cdb25e47fa38475` |  |
| `reconcile_work_items_20260920` | 146,246 | 20,123,836 | `26998ffbe508d829677ce3b8e36260c6ad251014cfb11798e6c8876d2d3b0011` |  |
| `reconcile_native_exports_20260920` | 1,560 | 287,257 | `4a197abd94497f2d18fe379a959d459ce38c025347eecfa9e063ffff65ac56db` |  |
| `reconcile_bas_candidates_20260920` | 24 | 9,611 | `dfb67b3f0f3a7bcd0c8b4d7f07430dc860eae0e48400a35bb652ba0756997291` |  |
| `reconcile_source_observations_20260920` | 24 | 6,909 | `38c723504291c43f3324b741878d18bc8468f9f318aee93ac50e139c7b75fd5a` |  |
| `reconcile_generations_20260920` | 1 | 4,619 | `02e0b1f96cee039df1a719079316f5fb564d0d61bc67ff3d27312a9de9f1643f` |  |

### `route_a_20260918` (1 tables, 1,785,384 rows, 45,589,768 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `vault_occ_v1` | 1,785,384 | 45,589,768 | `c95b09f265cc03bc289fba178fc3a2132a8c56d7db5c724925c673d94fa3c4da` |  |

### `historical_analysis_stale_tree` (16 tables, 348,282 rows, 8,179,255 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `vault_twins_copy_manifest_b1` | 95,101 | 3,576,613 | `2086430d0d07d1876679c8a498d3b95f17962e40cae2254a930a64b5d628a80b` |  |
| `dir_manifests` | 22,475 | 1,634,185 | `4b6792537a570b838c2974cd492a27a5aa5b5c0b1062e543e0cb9c158a314e30` | bytes:numeric->bigint |
| `vault_twins_rehome` | 43,266 | 1,246,621 | `6e7803ec7d728d84e40489e7be0f3c9f5e6f814f8bc9117167b6053cb5766d3e` |  |
| `vault_twins_dirs` | 20,816 | 763,270 | `2703ea680c3c5c678ea37f7ba58197869604e583e57784c3d20718d63cebc0c9` | total_bytes:numeric->bigint |
| `vault_twins_overlap` | 74,382 | 474,775 | `b60044677357e0b65f96b6ab26766ced0eb57c30b71d512a8c56df6482a38818` | a_bytes:numeric->bigint, b_bytes:numeric->bigint, common_bytes:numeric->bigint, only_a_bytes:numeric->bigint, only_b_bytes:numeric->bigint, overlap_pct:numeric->double |
| `vault_twins_pairs` | 74,382 | 204,292 | `5bb8da81aa479d6bf9e33bc3b0b8fe26cdbe7a7738579bfe700115fe790b721b` |  |
| `vault_twins_groups` | 7,960 | 93,096 | `b9d7502f442751d33022913c455b580ae9adc05457d96873726424187167a35a` | member_gb:numeric->double |
| `vault_twins_group_summary` | 2,131 | 75,157 | `b9c87d6ffe2ffb2e9b0ae9a440ff6a9585d03c78caf71d623f43f44215fd5a84` | total_bytes:numeric->bigint, distinct_bytes:numeric->bigint, dup_bytes:numeric->bigint, trunk_bytes:numeric->bigint |
| `dir_identical_groups` | 386 | 29,390 | `d40ea1a467536eb19f773bcc467cbd2aa9915de39faffd417e490bcdfa8ef73c` | bytes:numeric->bigint |
| `dir_samename_overlap` | 2,632 | 20,534 | `73f71f214d56b44700416f7449a5878ca605fdb31fbb59e87626f07f0b6f6d91` | overlap_of_smaller:numeric->double |
| `vault_twins_unit_safety` | 2,131 | 19,961 | `3b96d520281833c7d8712b71bb80921057586ce001a4d8c37dba49ca1e69af48` |  |
| `vault_twins_merge_plan` | 500 | 13,286 | `3db424e32624c347120b7be1b8a4b5acbcab3707d2467566aa53505bc38fa236` | member_bytes:numeric->bigint, dup_in_trunk_bytes:numeric->bigint, carry_bytes:numeric->bigint |
| `vault_twins_merge_plan_groups` | 221 | 12,982 | `860af09ba9021b8c508cf4f0ebe687bc7985a9aed9da49bf8d6ac2467d22124a` | trunk_bytes:numeric->bigint, dup_bytes:numeric->bigint, carry_bytes:numeric->bigint |
| `vault_unit_roots` | 1,844 | 11,285 | `2f0ecc138956c585dd9ae08bdde4cac8471a535837ecab603f1f9b109567d478` |  |
| `dir_twin_groups` | 40 | 3,375 | `03b03b1d3fa25bccf475c91a81d1a4348c5032e0e5d14e42e70ae6ac60ba8cb7` | files:numeric->bigint, bytes:numeric->bigint |
| `vault_twins_quarantine_roots` | 15 | 433 | `81908a610d2e1b3e1c8edf27fa1425a0543022b427c63cdc99b1d0045a7e2a2a` |  |

### `model_inferred` (1 tables, 15,106 rows, 6,159,112 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `enrichment` | 15,106 | 6,159,112 | `43c2695b723f806f6c30db058a04a4fe31562f7f67b3509abc31d71a0492a55f` |  |

### `plan_era` (1 tables, 711 rows, 10,306 bytes)

| Table | PG rows = Parquet rows = read back | Bytes | sha256 | Casts |
|---|---:|---:|---|---|
| `vault_units_v2` | 711 | 10,306 | `5c9e042cbc4e65cefefa896e0ebfefd12691d5587eda7720bb6e461bd41891b4` | bytes:numeric->bigint |

### Other objects

| Object | Rows | Bytes | sha256 |
|---|---:|---:|---|
| `corrupt_missing.csv` (receipt_export_20260914) | 3,389 | 310,046 | `047751ea0a2677d2150a5a4e74ce27d4632e9b9215198cfda30911c4879aa0e0` |
| `schema.json` (schema) | 102 | 273,651 | `2d5f9891438200500a19c5e0b10a910db4e3cc271d82e3b61a65758da23e74bf` |
