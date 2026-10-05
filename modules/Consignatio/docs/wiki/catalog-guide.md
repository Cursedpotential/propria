---
title: Case Bible catalog and lake guide
type: reference
status: source-verified
date: 2026-10-04
byline: Codex / GPT-6
revision: 1
source_path: modules/Consignatio/docs/CASE-BIBLE-CATALOG-GUIDE.md
source_sha256: e232bdda47981d9ba65e1f4aa3b2498c09a25c2655f0ba0330753ed8168dff6e
---

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
| Find words, topics or events inside files | `/cb-vsearch "query"` | One content-search tool; meaning/keyword retrieval, DuckDB filters and source-linked output |
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

## Search for content

Byline: Codex, GPT-6, 2026-10-04. Search contracts checked against source, live collection schemas and live queries; presentation verified with source-retention tests.

Use `/cb-vsearch` to search what files say. Use `cbcat` to search the inventory of files. Finding a file in the inventory does not mean its text has been indexed, and a search hit does not independently prove its recorded source key is still the current physical location.

### Start with a question

```text
/cb-vsearch "arranging school pickup"
```

This searches existing indexed documents, messages and AI chats, combines meaning with keywords, and returns up to eight readable results. You do not choose a database or collection.

| What you want | Example |
|---|---|
| Messages and calls | `/cb-vsearch "arranging school pickup" --corpus messages` |
| Indexed file text/chunks | `/cb-vsearch "parenting schedule" --corpus documents` |
| AI-chat context | `/cb-vsearch "budget planning" --corpus chats` |
| Keyword matching without a query embedding | `/cb-vsearch "school pickup" --mode keyword` |
| Require a literal word in retrieved text | `/cb-vsearch "school pickup" --corpus messages --contains pickup` |
| Narrow to a recorded source path | `/cb-vsearch "budget" --source Takeout` |
| Narrow to dates | `/cb-vsearch "school" --corpus messages --from 2024-01-01 --to 2024-12-31` |
| More candidates to filter | `/cb-vsearch "school" --fetch 200 --k 12` |
| Machine-efficient output | `/cb-vsearch "school" --presentation compact` |
| Expanded JSON | `/cb-vsearch "school" --presentation json` |
| Revisit a result | `/cb-vsearch --corpus messages --id <result-id>` |

In a terminal the same tool is `python <plugin-root>/tools/cb_vsearch.py` with the arguments shown after `/cb-vsearch`. Agents resolve plugin-root from their installed plugin environment.

### Read a result

The human view shows a numbered result, its content category, title, recorded source path, available date, result ID and excerpt. It also shows the available source hash and locator (such as a message index, conversation ID, document/chunk ID or archive member). AI-chat results provide context; verify substantive claims against their source records.

`--excerpt-chars 1200` expands the displayed excerpt. Default600, maximum4000; shortened text is marked `[excerpt]`. Opening a result ID retrieves the same indexed record without generating another query embedding; the excerpt limit still applies. It does not open or download the underlying source file.

The compact view is JSON with `format: compact-columns-v1`: hit field names appear once in `columns`, values appear in the corresponding `rows`. The expanded JSON view uses named fields for each hit. Both retain citation objects, available hashes, source/record/chunk locators and truncation status. Duplicate identities retain all distinct source citations. A missing source hash is reported as `not_indexed`, never fabricated.

### How the tool works

```text
Your query -> matching query embedding (hybrid mode only)
           -> existing content indexes -> bounded candidate results
           -> in-memory DuckDB filters and duplicate-identity removal
           -> readable text or compact/expanded JSON with citations
```

CocoIndex and the search tool have different jobs. CocoIndex is the owning Intake discovery pipeline's framework for processing source changes and maintaining derived index data. Weaviate stores/searches the derived content. DuckDB shapes the returned candidates. The search tool uses the established named-vector/query-model contracts; it does not create another index or persistent result database. It runs its embedding, retrieval and DuckDB work on ovh-files through the existing SSH route.

A search does not run an indexing refresh. This tool reads the existing Intake content index and the existing message/chat indexes. The retrieval repair does not establish that all historical message/chat producers use CocoIndex or that automatic synchronization is complete. The October2 message-flow audit is the dated evidence for those producer gaps; it is not a current deployment proof.

### Coverage and filter behavior

Live metadata readback on October4: documents37,857 indexed objects, messages366,912, chats446. These are chunks/events, not counts of unique source files. They do not establish coverage of all568,130 recorded B2 objects. Each query reports its current scope counts, candidates retrieved, results returned and unavailable scopes.

Default candidate window:80 per requested category; `--fetch` increases it up to500. `--source`, `--contains` and dates are literal DuckDB filters over those retrieved candidates. They are not an exhaustive scan of every indexed object. An empty filtered result means no match in that window; widen the window, adjust the question or query the file inventory. Date filters exclude results whose date was not indexed. Ranking is relevance ordering, not a confidence percentage.

For guides, decisions and project documentation use Propria Docstore search. For repository code use CCC. Those CocoIndex-backed systems remain separate from the Case Bible source corpus; shared technology does not make them interchangeable search scopes.

### Verification and ownership

Working implementation: plugin tools/cb_vsearch.py and tools/cb_search_server.py; command instructions commands/cb-vsearch.md. Retrieval siblings: Consignatio Intake/backend/src/casebible_index/filesystem_search.py and casebible/tools/comm_timeline_mvp/search.py. Producer distinction: Probata docs/reference/2026-10-02-message-flow-as-built.md. The plugin guide is a distributed copy of this owning document, not another catalog.

Checks: live keyword and hybrid queries, all three categories, exact-ID retrieval, readable/compact/expanded JSON output, literal source/text/date filters; five synthetic presentation tests covering literal SQL-looking filters, citation retention on deduplication, excerpt markers, column packing and request boundaries. Those prove the reader/presentation behavior, not complete indexing or relevance for every question.

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

There are still 33 registry rows with unknown classification. The catalog/publication readers were repaired and installed in both desktop apps; the content-search repair described below is verified separately; already running chats may hold earlier instructions in their conversation context. The Probata whole-bucket readers were already repaired October 2; this pass also fixed four still-active loader/source-resolution joins that referenced the renamed intake-only table. Those loader patches were syntax-checked, not rerun against source data.

The publication itself does not prove every D: file reached B2, promote source data into evidence, complete R2 migration, or synchronize Weaviate search collections. Use the existing drive comparison and proof tables for those questions and preserve their bounded claims.

## Evidence and authority

- Live PostgreSQL: bucket_objects_current, catalog_registry and lake_publish_20260927 queries on ovh-files, 2026-10-04.
- Live B2: depth-one lake-prefix listing and LATEST readback, 2026-10-04.
- Producers: ../casebible/tools/bucket_objects_load.py and ../casebible/tools/lake_publish_20260927.sh (paths relative to docs).
- Historical publication: receipts/lake-publish-20260927/README.md, retrieved through Docstore; its reader snippet explicitly reports it was not executed.
- Current retention/execution authority: active owner policies casebible_consolidation_retention_20261004 revision 3 and casebible_temporal_atomicity_20261004 revision 1, as referenced by the installed retention policy. Source policy requires local copies unchanged, intact units, server execution and verified scoped projections.

- October 4 execution proof: docs/receipts/lake-publish-20261004/; B2 manifest.csv and raw_duck.lake_publish_20261004; all five success markers and Temporal run recorded above.
