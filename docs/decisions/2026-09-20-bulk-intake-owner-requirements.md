---
title: Bulk intake — owner requirements and verified tooling
date: 2026-09-20
status: owner requirements recorded; design proposed; nothing built
domains: [probata, intake, consignatio, workbench]
tags: [bulk-intake, archives, takeout, streaming, retry, units, parser-creation, rclone-archive, decision]
---

# Bulk intake — owner requirements and verified tooling

> _Byline: Claude Code · Fable 5.1 · 2026-09-20. Owner statements are quoted with their time (EDT). Everything under "Proposed" is a proposal awaiting the owner's go._

## Why

Owner 21:03: "we need more bulk options this one at a time bull shit it stupid as fuck, i have over 2 tb of data."

As built, the engine starts one source per call (`POST /reference-import/start`, `modules/engine/runtimeapi/proffer_preview.go`), with a hand-declared format and per-operation human gates. No batch, no fan-out, no retry flow.

## What the 2 TB is (catalog, `catalog_reconcile.object_versions WHERE visible`, 21:10)

| Kind (by extension, first cut) | Objects | Bytes | % bytes |
|---|---:|---:|---:|
| Archives | 49,126 | 1,433 GB | 69.8 |
| Photos | 212,790 | 204 GB | 9.9 |
| Other / no extension | 152,548 | 189 GB | 9.2 |
| Video | 14,467 | 139 GB | 6.8 |
| Structured/text (parser candidates) | 109,079 | 79 GB | 3.9 |
| Documents | 12,445 | 7.9 GB | 0.4 |
| Audio | 4,098 | 0.9 GB | 0.0 |

Archive patterns: **642 Google Takeout numbered parts = 1,330 GB** (each part is a complete zip); 2,364 single zips = 63 GB; 45,664 `.gz` single files = 34 GB; 446 tar/tgz = 5.8 GB; 10 7z/rar = 352 MB; **zero true split archives** (`.z01`, `.zip.001`, `.partN.rar`).

## Owner requirements (2026-09-20)

1. **Read from the zip in place; extract on demand; stitch parts.** 21:14: "allow on demand but also allow me stitch varius zip parts together ... we should be able to read from the zip." With zero true split archives in the catalog, stitching means presenting the parts of one Takeout as one unit with one member list. No byte-joining.
2. **Stream everything.** 21:14: "this is wrong STREAM EVERYTING" — said of the `read_xml` path that holds a whole document in memory inside the shared PostgreSQL. A whole-document-in-memory reader is a defect to remove, not a constraint to throttle around. Consistent with ADR-0052 ruling Q3 (Go parses every format it has a decoder for, any size): SMS XML goes SBV → NDJSON, which DuckDB reads as a stream.
3. **Failure handling offers options.** 21:15: "allow options when more than one option exists" — a retry cap with a "cannot succeed, stop" classification, and where more than one way forward exists (another handler, a repair, another route), present them instead of only stopping.
4. **Units: mark by hand, and auto-select known patterns.** 21:15: "select dir from tree and mark as unit"; 21:16: "recorded and identified patterns are auto selected." A marked pattern is recorded and reused.
5. **No parser → agent.** 21:16: "if no parser can be found add call button to agent to guide parsing creation."
6. Carried from earlier the same day: clean files reach context without a click (the owner's clicks are send-to-Surreal and promote-to-evidence, D-145); older incremental backups proven contained in a newer one are skipped; missing payloads are an exception that stops a file (`docs/planning/2026-09-20-TODO.md`).

7. **Takeouts are a highly supervised human-in-the-loop flow.** 21:17: "takeouts are gonna be touchy, many accounts many parts thats gonna be a highly supervised hitl flow." Requirement 6's click-free path does **not** apply to Takeouts. The app may recognise and propose a set; the owner confirms account, parts and gaps before anything runs. Takeouts stay atomic and are never filtered inside (owner rules 2026-09, auto-memory `takeouts-are-atomic`, `critical-evidence-categories`).

   Catalog at 21:18, parts named `takeout-<stamp>-<job>-<NNN>.zip|tgz` only: **54 sets, 350 parts, 609 GB, across 18 folders; 16 sets have missing part numbers**, none have duplicates, one part is under 1 KB (174 bytes). 563 GB of it sits under `Takeout [gdrive-salemnet]`. About 290 further parts (≈720 GB) match the looser `takeout-*-NNN` shape with other naming and are not grouped yet. Several parts sit directly under `vault/v1/` as their own top-level entries. Which account each set belongs to is not derivable from the name.

8. **Align zips and extracts, side by side.** 21:18: "should be easy to ground the zips harder to ground the extracts tho we need to align those 2 wherever we can and have them side by side." A zipped Takeout part grounds itself (export stamp in the name, intact container, member list). Already-extracted Takeout trees in the vault have lost that. Proposed mechanism, not built: a zip's central directory gives every member's path, uncompressed size and CRC-32 without downloading the archive (proven below); the catalog has each extracted file's path, size and hash. Same relative path + same size is an alignment **lead**; CRC-32 of the extracted file (or hashing the member on demand) confirms it. The screen shows zip member and extracted file side by side with the match basis; an extract with no zip counterpart, and a member with no extract, are both shown as such.

## Verified tooling (live, read-only, 21:16)

- **rclone `archive` backend** (official; rclone v1.74.3 on ovh-files, v1.74.4 on the desktop): read-only access to members of `.zip` on cloud storage without downloading the archive; `:archive:remote:path`, plus `rclone archive list|extract`. Proof on `salem-data/consignatio/vault/v1/Takeout [gdrive-salemnet]/takeout-20260415T063959Z-9-001.zip` (1,475 MB): **6,928 members listed in 5 s**; one 826-byte JSON member read out in 4 s, byte count equal to the listing, valid JSON. Nothing downloaded or written to B2. Note: the B2 remote exists only in the desktop rclone config; `/opt/casebible/rclone.conf` on ovh-files has `r2:` and `od:` only, so hosted use needs a B2 remote there.
- **Go `archive/zip` over an `io.ReaderAt` backed by range requests**: standard library; each member opens as a streaming reader. The engine-side fit for requirement 2.
- **DuckDB `zipfs` community extension: not suitable for large members.** Its README: "The selected file will be read entirely into memory, not streamed."
- tar/tgz/gz have no index: they can only be streamed front to back (5.8 GB + 34 GB of the corpus).

## Proposed order (awaiting owner go)

1. Retry cap + non-retryable classification + options when several routes exist; concurrency cap.
2. Remove the whole-document XML read: route SMS XML through the streaming SBV path regardless of size.
3. Unit registry: recorded patterns auto-select; a folder marked from the tree becomes a unit; members enumerated in place. For Takeouts the auto-selection is a **proposal screen** (set, parts present, parts missing, tiny/suspect parts, account to be confirmed by the owner) and nothing runs until the owner confirms that set.
4. Batch workflow over a unit: signature-detected formats, exceptions-only gating, batch progress with retry-failed.
5. "No parser found" exception carries a call-the-agent action for guided parser creation.
6. Photos and video (≈343 GB, 227k files) are catalog + metadata work, outside the message engine.
