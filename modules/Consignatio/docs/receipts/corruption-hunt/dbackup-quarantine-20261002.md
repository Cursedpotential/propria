# D:\Backup zero-filled quarantine move, 2026-10-02

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-02 (run for owner go 19:31 EDT; tool and decision from 2026-09-13)_

- **What:** 10,811 all-zero payload files (10,590,332,535 bytes, 10.59 GB) moved from `D:\Backup\<rel>` to `D:\Backup\_quarantine_zero_filled\<rel>`. Same volume, `os.renames` (metadata move), mtime and attributes kept, nothing deleted, nothing copied.
- **List:** `dbackup/D_backup_all_zero.tsv` (scan of 2026-09-13, `ALL_ZERO_PAYLOAD_FILES=10811`; 10,812 lines with header). The scan found 25,894 zero-length files in addition; they are not payloads and were not moved (09-13 rule: 0-byte files carry no hash and are not in the zero-filled list).
- **Tool:** `quarantine/local_zero_quarantine.py D:/Backup dbackup/D_backup_all_zero.tsv --apply` (the same script that did F: on 2026-09-13). Per file it re-stats (size must match the scan), re-reads the whole file (every byte zero), refuses an existing target, renames, then confirms target present with the same size and source gone; it stops at the first unverified move.
- **Result:** planned 10,811, moved 10,811; skipped: missing 0, size changed 0, no longer all-zero 0, target exists 0, ambiguous 0.
- **Independent verification (after the run):** for every ledger row the destination exists with the scanned size and the source path is gone: 10,811 ok, 0 FAIL. The ledger's source set equals the list's path set exactly.
- **Ledger:** `dbackup-quarantine-20261002.tsv` (source, dest, size, verified_zero, result, post_check; gitignored payload). Per-file decision log written by the tool: `dbackup/20261002-193633-D_Backup-local-quarantine-APPLY.jsonl`.
- **Catalog:** not written. The 09-13 catalog flag and hash-null (PG `raw_duck.integrity_hold`, local DuckDB) were done before the moves; the F: move, as logged, recorded no catalog change afterwards, so none was made here.
