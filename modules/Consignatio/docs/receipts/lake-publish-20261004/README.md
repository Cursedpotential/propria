# October 4 Case Bible lake publication receipt

Byline: Codex, 2026-10-04. Verified against B2 bytes, PostgreSQL ledger and Temporal execution output.

STATUS: VERIFIED PUBLICATION. This receipt does not establish complete source coverage or full drive migration.

- B2 home: salem-data/consignatio/_system/lake/2026-10-04/; LATEST readback: 2026-10-04.
- Temporal workflow: casebible-lake-20261004. Successful run: 01a1094f-5f6f-757c-96ca-881c61914897.
- Five independently tracked stages: export, schema, upload, readback, finalize.
- 90 Parquet tables, schema.json and historical corrupt_missing.csv: 92 verified artifacts. Every artifact matched SHA-256; all table row counts matched the source export. Ledger: raw_duck.lake_publish_20261004, 94 records including manifest and pointer, 1,682,018,414 total bytes.
- Published bucket_objects_current: 2,186,822 rows; B2 salem-data568,130. R2 observations are October2, not refreshed by publishing today.
- Live snapshot reader independently resolved LATEST, verified bucket_objects_current's digest faadabb7e7e88a942874b333a0d3bbc713ec4b6644e8b300a42479512d9cc795 and returned all eight provider/bucket totals.
- Initial run 01a10946-b2b0-7d65-ad45-f29e863cf577 failed because views were skipped. LATEST did not advance. The phase exit-code bug and view selection were repaired; failed evidence retained in temporal-run.log.
- Source publisher subsequently tightened pointer ordering so future executions verify manifest admission before pointer writes. This follow-up was source-reviewed and syntax-checked; it was not a second publication.

The table list is the explicit publication scope. The manifest records status, object keys and content hashes. export_results.tsv records before/after/export counts, casts and fingerprints; readback_results.tsv records independent downloaded hashes and counts. SHA256.txt fingerprints this receipt package. Neither the downloaded Parquet data nor source documents are committed here.

Usage and limits: ../../CASE-BIBLE-CATALOG-GUIDE.md. Tools: ../../../casebible/tools/lake_publish_20261004.sh and .sql; tracked runner: ../../../../Probata/probata/modules/engine/cmd/casebible-lake-publish/main.go. Plugin source: E:/AI_Workspace/plugins/plugins/case-bible.
