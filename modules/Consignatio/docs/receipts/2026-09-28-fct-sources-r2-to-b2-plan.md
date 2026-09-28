# Family Law Toolkit primary sources: R2 → B2, and a B2 → Surreal sync (Phase 1 plan)

> _Byline: Claude Code · Opus 5.5 · 2026-09-28 (agent `sources-b2-sync` for the "portal cut over" session)_

**Status:** Phase 1 done: an inventory and a plan. **No data has moved.** The transfer and the sync job wait
for the owner's sign-off on the decisions in §7.

**Owner, 2026-09-28 04:01 EDT:** "Everything needs to be migrated to B2. R2 is being retired. My preference
would be that it's synced with the canonical source, so it gets saved in the doc store database or something,
in Surreal, and then gets pulled into the plugin where it runs, and updated on change, and that's how we keep
it in sync with the legal work desk."

**Catalog load:** `casebible/tools/fct_sources_inventory_20260928.{sh,sql}` →
`raw_duck.fct_sources_inventory_20260928` (PG `casebible` on ovh-files). It was run at 2026-09-28 04:16 EDT and
loaded 543 rows: R2 143 · desktop 143 · ovh-files 143 · store 114 · B2 probe 0. Every number below comes from
that table or from a read-only store query.

---

## 1. Where the files are today

### Answer first

- **143 files, 52,446,084 bytes (50.0 MiB).** There are three byte-identical copies. They match on
  md5 + size for 143 of 143, and on sha256 between the two local copies. **None is missing and none differs.**

  | Copy | Location | Objects | Bytes |
  |---|---|---:|---:|
  | R2 | `r2:casebible-sorted/fct-sources/primary/` | 143 | 52,446,084 |
  | Desktop plugin | `~/.claude/local-plugins/plugins/family-court-toolkit/content/custody-guide/sources/primary/` | 143 | 52,446,084 |
  | ovh-files | `/data/probata/volumes/opencode/home/.claude/local-plugins/plugins/family-court-toolkit/content/custody-guide/sources/primary/` | 143 | 52,446,084 |
  | B2 (proposed prefix) | `salem-data/consignatio/casevault/KnowledgeBase/legal/fct-sources/primary/` | 0 | 0 |

- **By type:** 28 PDF (44,841,745 bytes) · 72 Markdown (6,261,416) · 42 HTML (1,335,268) · 1 `SHA256SUMS` (7,655).
- **R2 history:** uploaded 2026-09-08 12:38 EDT from the ovh-files copy by `rclone copy`. The same session's
  `rclone check` reported "0 differences found, 143 matching files" (session log). The catalog's R2 listing
  (`raw_duck.r2_files`, loaded 2026-08-30) predates that upload, which is why the prefix was absent from
  the catalog until today's load.
- **R2 is readable today.** The listing call succeeded. The error 10042 entitlement block from 2026-09-24 did
  not affect these reads.
- **A fourth copy is in Git.** The same 143 files, including the 28 PDFs, are committed in the monorepo at
  `modules/Probata/probata/deploy/docker/family-court-console/src/content/custody-guide/sources/primary/` and
  baked into the hosted console's image.

### `SHA256SUMS`

- 72 entries match the file on disk and **0 differ**.
- 4 entries name PDFs that exist in **no** copy: `michigan-court-rules_2026-07-31.pdf`,
  `michigan-rules-of-evidence_2026-01-28.pdf`, `2025-michigan-child-support-formula_eff-2025-01-01.pdf`,
  `local-court-rules-circuit_2025-06-01.pdf`. Only their Markdown extractions were kept. `fetch-sources.sh`
  can re-download them from the official URLs. That is a gap to record, not a transfer blocker.

### The store (`surreal-case`, ns `fct`, db `case`)

- **`source` has 193 rows, one per ledger entry.**
  - **0 of 193 carry a sha256.**
  - **119 carry an `r2_path`, and every one of them points at an object that does not exist.**
  - Cause: `load-content-to-store.mjs` builds `r2_path` from the ledger's `file_path`. That path is relative to
    `content/toolkit/` (117 rows: `references/…`, `templates/…`, `checklists/…`) or doubly prefixed (2 rows:
    `sources/primary/from-drive-20260907/…` becomes `…/primary/sources/primary/…`). The loader sets `r2_path`
    whenever `file_path` is present. `README-store.md` says it is set "only … when a local file actually
    exists", so the code and the document disagree.
  - The other 74 rows have no `file_path`.
- **The ledger does not link to the primary files.** Only 2 of the 143 files have a ledger entry that names them:
  the two `from-drive-20260907` benchbooks. The other 141 are reached through `master_source_directory.md` and
  the reference rows.
- **`reference` has 114 rows (kind `source_note`) holding the text of the 114 `.md`/`.html` files**, loaded
  2026-09-27 by `load-reference-materials.mjs`. **All 114 sha256 values match the desktop copy.**
- **These 29 files have no store row at all:** the 28 PDFs and `SHA256SUMS`. Two of those PDFs, the MJI Contempt
  and Sexual Assault benchbooks, have no text extraction anywhere. The other 26 PDFs have a `.md` or `.html`
  sibling that is in the store.

### Already on B2 elsewhere (bytes-once check, `raw_duck.b2_content`)

3 contents (19,305,763 bytes) are already stored in the raw corpus under `consignatio/intake/raw-dedupe/v1/source-buckets/casebible-raw/…`:

- `foc/michigan-parenting-time-guideline.pdf`
- `from-drive-20260907/mji-contempt-of-court-benchbook.pdf`
- `from-drive-20260907/mji-sexual-assault-benchbook.pdf`

---

## 2. B2 destination (decision D1)

The B2 layout convention is `salem-data/consignatio/<area>/…`. The existing areas are `intake/`, `vault/v1/`,
`casevault/`, `_system/lake/`, `reconciliation/` and `recovery/`. `casevault/` is the Level-2 vault scaffold,
deployed 2026-09-21 (`casebible/tools/vault_level2_scaffold_20260921.py`). Its `KnowledgeBase/legal/INDEX.md`
defines that folder as "Legal reference imports, research outputs and reviewed authority collections", and it
is empty apart from the index.

- **A (default): `salem-data/consignatio/casevault/KnowledgeBase/legal/fct-sources/primary/<path>`.** This uses
  the vault's own defined home for legal authority collections. The key under `primary/` is unchanged, so every
  R2 path maps to its B2 path by swapping the prefix.
- **B: `salem-data/consignatio/reference/fct-sources/primary/<path>`.** This creates a new area for application
  reference content, kept outside the case vault. Choose it if public legal material should not live inside the
  case's knowledge base.

**D2 (bytes once):** the three contents already in `raw-dedupe` would be stored a second time. The settled design
("bytes once, metadata N times") governs the evidence corpus. Here the application needs one self-contained
prefix. The duplicate costs about $0.0001 a month.

- **Default: copy all 143.**
- Alternative: skip the 3 and point their `b2_path` into `raw-dedupe`.

---

## 3. Transfer plan (dry run: nothing has been run)

| | |
|---|---|
| Objects | 143 (all of §1, including `SHA256SUMS`) |
| Bytes | 52,446,084 (50.0 MiB) |
| Source | ovh-files local copy (default, D3), hash-verified identical to R2 and to the desktop |
| Destination | `b2native-full:salem-data/consignatio/casevault/KnowledgeBase/legal/fct-sources/primary/` (per D1) |
| Runs on | ovh-files, detached (`setsid nohup … &`), never from the desktop |
| Mode | add-only: `rclone copy --immutable` refuses to overwrite anything already there |

**D3, which copy is the source:**

- **Default: the ovh-files local copy.** It needs zero R2 operations. The verification can then use SHA-1, the
  hash B2 stores natively, because R2 offers only MD5 and B2 offers only SHA-1. rclone therefore has no common
  hash between R2 and B2 and falls back to `--size-only`.
- **Alternative: `r2:casebible-sorted/fct-sources/primary`.** This reads from the copy being retired, but the
  check is size-only.

The command, from a script file so no agent shell owns it:

```sh
SRC=/data/probata/volumes/opencode/home/.claude/local-plugins/plugins/family-court-toolkit/content/custody-guide/sources/primary
DST=b2native-full:salem-data/consignatio/casevault/KnowledgeBase/legal/fct-sources/primary
W=/data/consignatio/fct-sources-20260928
rclone copy "$SRC" "$DST" --immutable --transfers 4 --checkers 8 -v --log-file "$W/rclone-copy.log"
```

**Verification after the write** (memory `rclone-verify-after-write`):

1. `rclone check "$SRC" "$DST" --one-way` compares SHA-1 values and must report "0 differences found, 143 matching files".
2. `rclone check … --size-only` gives a second, independent comparison.
3. A fresh listing of `$DST` (`lsf -R --format pst --hash SHA1`) is loaded into
   `raw_duck.fct_sources_inventory_20260928` as a `b2` location, which the table's CHECK constraint must first
   allow. This happens in the Phase 2 script, `fct_sources_b2_copy_20260928.sh`, and makes the catalog record
   what is on B2.
4. Readback: download the 143 objects to `$W/readback/`, sha256 each, and compare with `desktop_local`. It must
   be 143 of 143. This is the same readback step `lake_publish_20260927.sh` uses.
5. Only after all four pass: record the result in this receipt and in `docs/URGENT-TODO.md`.

**Cost** ([B2 pricing](https://www.backblaze.com/cloud-storage/pricing),
[B2 transactions](https://www.backblaze.com/cloud-storage/transaction-pricing), checked 2026-09-28):

- **Storage:** 50.0 MiB at $6.95/TB-month is about **$0.0004 a month**.
- **Uploads:** 143 uploads are Class A, which is free.
- **Check and readback reads:** about 150 listing and download calls, under the free 2,500 per day for Class B
  and Class C.
- **Readback download:** 50 MiB, inside the free 3× egress.
- **R2 side (option D3-B only):** 143 GETs at $0.36 per million is about $0.00005, and R2 egress is free.
- **Total: under one cent.** R2 deletion is not part of this plan: R2 retirement removes the bucket later, as its
  own owner-approved step.

---

## 4. The sync: B2 canonical → surreal-case → toolkit and Advocatio

### The existing pattern this reuses (no new framework)

- **Docstore's pipeline**, `modules/Probata/probata/scripts/docstore/flow_docs.py`, is a CocoIndex 1.0 flow: a
  walk, then `mount_each(process_file)` with `@coco.fn(memo=True)` keyed on the file's `content_fingerprint`,
  then one declared row per item through the official `cocoindex.connectors.surrealdb` table target. Its runner,
  `worker_sync.py`, adds a lock file, a status JSON and a receipt per run.
- **`SYSTEM-BOUNDARIES.md` requires isolation.** The job is therefore **its own CocoIndex application** with its
  own app name, its own `COCOINDEX_DB` tracking state, its own lock and its own Surreal login. It shares nothing
  with Docstore or Intake except the library.

### The job: `fct-sources-sync`

- **Source:** `cocoindex.connectors.amazon_s3` (`S3Walker`, installed as the `cocoindex[amazon_s3]` extra) over
  B2's S3 endpoint (`s3.us-west-004.backblazeb2.com`), limited to the D1 prefix, using a **read-only B2
  application key scoped to that prefix**. Each object's ETag is its `content_fingerprint`, so a pass re-reads
  only objects that changed.
- **Trigger (D5):** the S3 connector has no live mode (`walk_dir(live=True)` exists only for local files).
  - **Default: a poll every 10 minutes** inside the container. That is 144 list calls a day, under the free
    2,500 per day for Class C. An unchanged pass downloads nothing.
  - Alternative: a B2 Event Notification webhook that triggers a pass immediately on upload.
- **Transform, per object:**
  - `sha256`, computed in the job, and `md5`/ETag and size from the listing;
  - `text`:
    - PDFs: **`pdftotext -layout` (poppler)**, the extractor that produced every existing `.md` extraction
      (`sources/ACQUISITION-LEDGER-2026-09-07.md`). OpenDataLoader is **not** used for text: the 2026-09-15
      bake-off found it silently drops glyphs (memory `opendataloader-pdf-tool`).
    - `.md`, `.html` and `SHA256SUMS`: the bytes as UTF-8, unchanged.
  - `text_extractor` names the tool and its version.
- **Target (D4):** one row per B2 object.
  - **A (default): a new table `source_file:<key>`**, keyed by the path under `primary/` using the same slug
    rule as `load-reference-materials.mjs`. Fields: `b2_bucket`, `b2_path`, `sha256`, `md5`, `size`, `format`,
    `text`, `text_extractor`, `ledger_ids` (from a mapping file, §6), `source_modified`.
    - One writer per table: CocoIndex reconciles the rows it declares, so it never shares rows with the Node
      loaders.
    - Ledger rows link to their files through `source.files: [source_file:…]`, which the ledger loader writes.
    - The 114 `reference` `source_note` rows are **retired** once `source_file` holds the same text, so the text
      has one owner. Retiring means deleting the store rows, which is gated on the owner.
  - **B: upsert straight into `source:<id>`.** This needs a curated file → ledger-id mapping for all 143 files
    first; 141 have no ledger id today. It also puts two writers (the ledger loader and CocoIndex) on the same rows.
- **Version:** `case_record` computes `sha256:` over the whole row (`RECORD_VERSION_SURQL`), so a changed file
  gives a new version in the toolkit and in Advocatio from the same query. The row must not carry a per-run
  timestamp, or every pass would change the version of an unchanged file.
- **Deployment:**
  - **Code:** `modules/Consignatio/fct-sources-sync/`, with a Dockerfile, the flow, the runner and README.md.
  - **Coolify:** an application on ovh-files built from `Cursedpotential/propria` with that base directory.
  - **Bind mount:** `/data/probata/volumes/fct-sources-sync/state` holds `COCOINDEX_DB`, the lock and the receipts.
  - **Credentials** (created under the owner's 2026-09-27 standing authority for our apps, stored in
    `~/.secrets/` and in Coolify, never in Git):
    - a new Surreal user, `fct_sources_sync` (EDITOR on `fct/case`);
    - the scoped B2 key.
- **Proof in Phase 2:**
  - change one file on B2 (e.g. re-upload `SHA256SUMS` with an appended comment line);
  - wait one pass;
  - the toolkit's `case_record` for `source_file:<key>` and Advocatio's `/v1/toolkit/records/source_file:<key>`
    must both return the same new `sha256:` version;
  - then restore the file, check that the version returns to the old value, and purge the test change.

---

## 5. Changes to the plugin and its copies (Phase 2)

These make the plugin read from the store instead of bundled files.

**Plugin** (`~/.claude/local-plugins/plugins/family-court-toolkit`, its own worktree):

1. `mcp-app/scripts/load-content-to-store.mjs`: stop writing `r2_path` and the always-null `sha256`, and write
   `files: [source_file:…]` from the mapping (§6). Clear the 119 dead `r2_path` values: an owner-visible store
   update, listed in §7.
2. `mcp-app/scripts/load-reference-materials.mjs`: skip `custody-guide/sources/primary/`, which the sync job owns.
3. `mcp-app/src/store.ts`:
   - `CaseSourceResult.r2_path` → `b2_path` (and `caseSourceOf`);
   - the case-extract exhibit import writes `b2_path`. The store has 0 exhibits today, so no legacy rows exist.
4. `mcp-app/src/content-store.ts` + `core.ts`: add `getSourceFile(key)` and serve a primary source's text from
   `source_file` first. The bundled file becomes a fallback only while the store is unreachable, the same
   store-first pattern `getReference` uses.
5. `mcp-app/src/store-tools.ts`, `mcp-app/README-store.md`, `skills/case-store/SKILL.md`,
   `mcp-app/tests/store.test.mjs`: rename the r2 pointer to b2 and document `source_file`.
6. `content/custody-guide/sources/README.md` (its "Plugin note" says the PDFs are not bundled; 28 are) and
   `fetch-sources.sh`: point at B2 as canonical.
7. Once the store serves every file: stop bundling `sources/primary/` (50 MiB). This is quarantined, not deleted.

**Copies:**

- **Hosted console** (`modules/Probata/probata/deploy/docker/family-court-console/src/`): the same edits.
  `store.ts` is byte-identical to the plugin's today. Drop `content/custody-guide/sources/primary/` from the
  synced subset once item 7 lands.
- **Family Court Workbench** (`modules/FL-MCP/src/types/store.ts`,
  `src/components/docket/docket-detail-drawer.tsx`): `r2_path` → `b2_path`.
- **Advocatio** (`modules/Legal-desktop/api/legal_workspace/services/family_court_toolkit.py`): add `source_file`
  to `TOOLKIT_TABLES` (option D4-A only). It has no R2 reference of its own.

---

## 6. Every other R2 dependency for this toolkit

**Retire in Phase 2:**

- **Store data:** 119 `source.r2_path` values, all dead (§1).
- **Code:**
  - `load-content-to-store.mjs` builds `casebible-sorted/fct-sources/primary/…`;
  - `store.ts` (`CaseSourceResult.r2_path`, `caseSourceOf`, and the exhibit import's `r2_path`);
  - `store-tools.ts` (the `case_source` description);
  - `store.test.mjs` (the `r2://bucket/real.txt` fixture).
  - Each file exists twice: in the plugin and in the hosted console copy.
- **UI:** FL-MCP `types/store.ts` and `docket-detail-drawer.tsx`.
- **Docs:** `README-store.md` (lines 142–148, 184, 326–339), `skills/case-store/SKILL.md` (lines 104, 113), and
  `sources/README.md`'s "Plugin note". Each also exists in the hosted copy.
- **Mapping (new, needed for D4 either way):** a file → ledger-id table. Today only the 2 benchbooks map. A
  curated mapping for the other 141 files is content work for the toolkit and is not blocking: `source_file`
  rows are complete without it.

**Not toolkit-specific; leave for the R2 retirement itself:**

- the ovh-files `r2:` remote in `/opt/casebible/rclone.conf`;
- openlist's `/r2/casebible-*` storages;
- the Probata Workbench object store (`modules/Probata/probata/modules/workbench/api/app/repo/object_store_client.py`,
  `types/source_roots.py`, OD-02).

---

## 7. Decisions for the owner

| # | Question | Default |
|---|---|---|
| D1 | B2 prefix | **A** `consignatio/casevault/KnowledgeBase/legal/fct-sources/primary/` · B `consignatio/reference/fct-sources/primary/` |
| D2 | The 3 contents already in `raw-dedupe` | **Copy all 143** · or skip 3 and point into `raw-dedupe` |
| D3 | Transfer source | **ovh-files copy** (SHA-1 check) · or R2 (size-only check) |
| D4 | Store shape | **A** new `source_file` table, links from `source`, retire the 114 `source_note` rows · B write into `source:<id>` |
| D5 | Sync trigger | **Poll every 10 min** · or B2 Event Notification webhook |
| D6 | Store clean-up | **Clear the 119 dead `r2_path` values** and retire the 114 `source_note` rows once `source_file` is verified |

Phase 2 starts only on the owner's explicit sign-off, relayed by the parent session.
