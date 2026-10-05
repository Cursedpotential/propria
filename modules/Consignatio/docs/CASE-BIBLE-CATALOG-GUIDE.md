# Case Bible catalog and B2 lake: one system, different generations

Byline: Codex, 2026-10-04. Live verification: casebible-pg on ovh-files, PostgreSQL queries and B2 LATEST readback. This guide is the owner-requested usage guide; dataset descriptions remain in catalog_registry, not a second manually maintained registry.

## Start here

The files live on B2. PostgreSQL holds the working inventory and provenance. Parquet files under consignatio/_system/lake/ are published snapshots of selected PostgreSQL tables. They are copies for portable SQL and reporting, not independently maintained inventories. Weaviate is a separate search projection of file contents.

A Parquet file is a table saved as a file. DuckDB can query that file directly. Publishing the table does not make it fresher than its source, synchronize it automatically, or copy the underlying documents.

## Which surface to use

| Need | Use | Meaning |
|---|---|---|
| What objects are currently recorded in a bucket? | raw_duck.bucket_objects_current | Latest recorded generation per provider and bucket; check the timestamp and coverage |
| What older listings exist? | raw_duck.bucket_objects | Historical generations; do not count all generations as current objects |
| Where did a file come from? | raw_duck.source_occurrences and related lineage tables | Source observations and provenance, not proof of current physical placement |
| What does a table cover? | raw_duck.catalog_registry | Status, scope, replacement and producer; descriptions can lag and need verification |
| Portable SQL over a published generation | B2 consignatio/_system/lake/<generation>/ | Parquet snapshot plus manifest/schema; use exact generation |
| Find words, topics or events inside files | Appropriate Weaviate collection | Search coverage differs from file-inventory coverage |
| Historical cleanup/repair proof | Dated proof tables and docs/receipts | Bounded analysis, not a replacement global catalog |
| R2 Iceberg tables | R2 Data Catalog | Separate table service; enabling it does not inventory arbitrary bucket files |

Do not create another catalog, local persistent DuckDB copy, bucket inventory, or manually maintained dataset registry to answer a question these surfaces already answer. Query the existing surface and record the generation. A missing field or stale projection is a repair to this system.

## Verified reconciliation on 2026-10-04

B2 LATEST now contains 2026-10-04. The verified publication contains 90 Parquet tables plus schema.json and the explicitly historical corrupt_missing.csv receipt. All 92 uploaded artifacts were downloaded from B2 and matched SHA-256 and row counts before the pointer advanced. The ledger has 94 records, including manifest.csv and LATEST.

The published bucket_objects_current has 2,186,822 rows across eight recorded provider/bucket combinations. B2 salem-data contributes 568,130 rows from the 2026-10-04 04:38:00.50128 UTC observation; the R2 observations remain October 2. Publishing today does not refresh those underlying observations.

The current publication includes bucket_objects_current, catalog_registry, drive_objects_current, source_occurrences, the MD5/SHA-1 bridge and R2/B2 proof table. The September publication remains historical. Registry labels in any snapshot describe its own classification and never certify full inventory coverage.

Catalog registry readback: 152 current, 117 historical, 44 superseded, 4 intermediate, 33 unknown. These labels are discovery aids, not integrity or completeness certificates.

## Why there are overlapping tables

Older folder listings, raw imports, reconciliation generations, hash bridges and repair receipts preserve different observations. Their overlap is expected when lineage is retained. They must be labelled and routed, not blindly merged or deleted.

The old raw_duck.b2_objects was renamed to raw_duck_superseded.b2_intake_objects_20260914. The pre-dedupe vault listing became raw_duck_superseded.vault_objects_20260916_0810_prededupe. These are historical observations. An old Parquet file can retain its old filename after the PostgreSQL table is renamed; that does not recreate an active table.

## Read-only PostgreSQL examples

Connect to database casebible in the Coolify casebible-pg container on ovh-files using the established approved connection. Do not use probata-db on port 5432 as a substitute; it is another database service. Discover the current container rather than copying a dated name.

    SELECT provider, bucket, listed_at, count(*) AS objects
    FROM raw_duck.bucket_objects_current
    GROUP BY provider, bucket, listed_at;

    SELECT key, size, sha1, listed_at
    FROM raw_duck.bucket_objects_current
    WHERE provider = 'b2' AND bucket = 'salem-data'
      AND key LIKE 'consignatio/casevault/%'
    LIMIT 100;

    SELECT * FROM raw_duck.catalog_registry
    WHERE object_name IN ('bucket_objects_current', 'source_occurrences');

These shapes were checked against the live schema; aggregate inventory and registry queries were executed. Name and size alone do not establish byte equivalence. Hash identity does not establish usable content or complete exports.

## Reading the Parquet publication

The maintained plugin launcher provides two distinct readers:

    cbcat query "SELECT provider,bucket,listed_at,count(*) FROM raw_duck.bucket_objects_current GROUP BY 1,2,3"
    cbcat lake
    cbcat lake bucket_objects_current "SELECT provider,bucket,count(*) FROM lake_table GROUP BY 1,2"
    cbcat lake catalog_registry "SELECT * FROM lake_table LIMIT 20"

The first reads the live PostgreSQL catalog. The lake commands resolve B2 LATEST and manifest.csv, download the selected Parquet artifact into a derived cache on ovh-files, verify its SHA-256 and size, then run one SELECT in a read-only database session. Every result reports the generation, table, digest and manifest row count. This does not create another persistent database. The default preview is 20 rows.

The reader was exercised against the published bucket_objects_current and returned the same eight provider/bucket counts as the export. It requires the existing SSH route and server helper lake_read.py at /data/consignatio/lake-publish-20261004/. Record the generation whenever reusing a result; consult schema.json for exported numeric casts.

## Refresh and publication

Inventory observation, catalog loading and lake publication are separate stages. New listings require explicit scope, success/failure coverage and a generation receipt. The current loader uses listing-file mtime and the view selects max(listed_at); those facts alone do not certify complete coverage. Empty buckets cannot be inferred from absence of rows.

Publication must export an identified source generation, record schema/counts/fingerprints/coverage, verify readback, and advance LATEST only after verification. Under the October 4 policy, processing runs on the VPS through separately tracked Temporal Activities; do not rerun the September standalone publisher as today's orchestrator.

The historical lake_publish_20260927.sh hard-codes the September generation and defaults to an obsolete container. It is evidence of the old publication procedure, not a safe current refresh command.

## Operational limits and remaining gaps

The October 4 publication uses a one-shot Temporal worker with five independently tracked Activities (export, schema, upload, readback, finalize). Workflow casebible-lake-20261004, successful run 01a1094f-5f6f-757c-96ca-881c61914897. It is not a recurring inventory refresh or a registration in the normal proffer-worker. The fixed date is intentional for this authorized publication. A future generation requires an explicit table scope and generation-specific receipt, not rerunning it unchanged.

The initial attempt skipped views and failed its export; the old publisher also returned a success exit code after failed phase markers. Both defects were repaired, the failed run retained, and LATEST stayed at September until the corrected run passed.

Explicit complete/partial/empty listing receipts remain a gap in the existing loader. A new timestamp alone cannot certify all bucket objects, and empty buckets cannot be inferred from zero rows. This publication preserves the existing observations; it does not make a new completeness claim.

There are still 33 registry rows with unknown classification. The plugin readers, commands and skills were repaired and installed in both desktop apps; already running chats may hold earlier instructions in their conversation context. The Probata whole-bucket readers were already repaired October 2; this pass also fixed four still-active loader/source-resolution joins that referenced the renamed intake-only table. Those loader patches were syntax-checked, not rerun against source data.

The publication itself does not prove every D: file reached B2, promote source data into evidence, complete R2 migration, or synchronize Weaviate search collections. Use the existing drive comparison and proof tables for those questions and preserve their bounded claims.

## Evidence and authority

- Live PostgreSQL: bucket_objects_current, catalog_registry and lake_publish_20260927 queries on ovh-files, 2026-10-04.
- Live B2: depth-one lake-prefix listing and LATEST readback, 2026-10-04.
- Producers: ../casebible/tools/bucket_objects_load.py and ../casebible/tools/lake_publish_20260927.sh (paths relative to docs).
- Historical publication: receipts/lake-publish-20260927/README.md, retrieved through Docstore; its reader snippet explicitly reports it was not executed.
- Current retention/execution authority: active owner policies casebible_consolidation_retention_20261004 revision 3 and casebible_temporal_atomicity_20261004 revision 1, as referenced by the installed retention policy. Source policy requires local copies unchanged, intact units, server execution and verified scoped projections.

- October 4 execution proof: docs/receipts/lake-publish-20261004/; B2 manifest.csv and raw_duck.lake_publish_20261004; all five success markers and Temporal run recorded above.
