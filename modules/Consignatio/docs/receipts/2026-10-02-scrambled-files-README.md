# Scrambled files in the vault, 2026-10-02

> _Byline: Claude Code · Sonnet · 2026-10-02_
> Source of truth: the catalog tables `raw_duck.scrambled_objects_20261002` and
> `raw_duck.scramble_head_probe_20261002` (casebible PG, ovh-files). The two CSVs beside this file are
> exports of them. Key lists only; no file content.

## What "scrambled" means

A file whose first 4 KiB has no format marker of any common type (jpeg, png, gif, webp, heic, mp4, mp3, pdf,
zip, gz, class, ...), is not text, and has the entropy of random data (7.9 to 7.95 bits per byte; ordinary
compressed media keeps its marker and never trips this). The catalog sha1 of such a file equals the scrambled
bytes, so the damage happened before cataloguing. The first five were found while building the HTML parser:
Facebook `account_activity.html`, `your_friends.html`, `your_post_audiences.html`, `30.html`, and a Google
Takeout `MyActivity.html`. No decompression, XOR or markup repair recovers them; the way back is another copy.

## The files

| File | Rows | What it is |
|---|---|---|
| `2026-10-02-scrambled-files-all.csv` | 16,812 | **List 1.** Every scrambled-looking catalog key, with category and whether B2 still has it. |
| `2026-10-02-scrambled-files-no-twin.csv` | 949 | **List 2.** Only the files with no twin and no fix (category B and C that exist in B2), with what was tried. |

Categories (column `category`):

| Category | Meaning | In B2 now | Stale catalog rows |
|---|---|---|---|
| A `scrambled_with_intact_twin` | Random head, and a same-name, same-size copy of another hash has an intact head. Recover from the twin (column `intact_twin_b2_key`). | 3,393 objects, 4,167,141,392 bytes | 9,749 |
| B `unreadable_no_twin` | The extension names a marker format (png, html, pdf, mp4, ...) but the head is random bytes, and no intact twin was found. Confirmed unreadable. | 918 objects, 1,521,482,951 bytes | 2,689 |
| C `suspect_not_in_apply_set` | Random head, extension is not a marker format (`.p7s`, `.jks`, `.draftsactiongroup`, unknown). Encrypted or signature files look the same, so these are listed and never quarantined. | 31 objects, 134,666,533 bytes | 32 |

`b2_state` is `in_b2` when B2 holds a visible object at that exact key with the catalog's size **and** sha1, and
`not_in_b2_stale_catalog_row` when it does not (the catalog lists copies that were moved or removed; there is
nothing to move for those rows).

## Quarantine (A + B in B2 = 4,311 objects, 5,688,624,343 bytes)

Owner order 2026-10-02: "Quarantine the messed-up files." Destination for each object:
`b2:salem-data/consignatio/_quarantine/scrambled-20261002/<original B2 key>` (column
`quarantine_key_if_applied`). Mechanism: the existing `b2_version_ops_20261001.py quarantine` (native
`b2_copy_file`, server-side, no download; verifies the copy's size and SHA-1 against the source version; only
then `b2_hide_file` on the original). It hides, it does not delete: the original bytes stay as a noncurrent
version and the hide marker can be removed to undo it. The bucket has no lifecycle rule, so versions are kept
and `HARD_DELETE` is false. An intact twin is never in the list. Category C is never in the list.

Status of this receipt: the dry run was built and reviewed; the apply is run by the session that owns the
deploy window, on ovh-files, with the command recorded in `docs/URGENT-TODO.md`.

## How it was found (and what it cannot see)

1. `scrambled_survey_20261002_candidates.sql`: every (file name, size) group with more than one sha1 (10,014
   groups) plus every object of the Facebook export folder `...NXPlelIY` (976 distinct objects in 6,991 rows):
   22,163 distinct sha1s.
2. `scrambled_survey_20261002_probe.py`: one ranged 4 KiB read per distinct sha1 (native B2 download with a
   `Range` header), up to 8 catalog keys tried because many catalog keys no longer exist.
3. `scrambled_survey_20261002_heads_to_sql.py`, `..._classify.sql`: load, classify, pair each scrambled sha1 with
   an intact twin of the same name and size.
4. `scrambled_survey_20261002_exists.py`, `..._exists_to_sql.py`: ask B2 whether each key still exists with the
   catalog's size and sha1 (and record the B2 file id the apply will use).
5. `scrambled_quarantine_20261002_list.sql`: the apply list.

Limit: only the two candidate sets were read. A scrambled file that has no same-name same-size sibling and is
outside the NXPlelIY folder is not found by this survey (reading every one of the 1.68 million catalog objects
would be about 1.7 million Class B calls). In the NXPlelIY folder, which was read in full: 342 distinct objects
intact, 216 scrambled, 418 no longer in B2, so the folder is not wholly scrambled.

B2 transactions of the survey (read only): about 68,000 Class B (4 KiB range downloads) and about 20,000
Class C (list calls); nothing written. Applying costs per object 1 Class B + 3 Class C and one hide (Class A,
free): 4,311 + 12,933 calls, about five cents.

## What was tried on the no-twin files (List 2)

- Twin search by same name and size: none intact (column `what_was_tried` gives the counts).
- `find_other_version` by name alone would propose other-size copies for review; none of these is a same-size twin.
- HTML: the Python repair engine (lxml recover, `repair.write-derived`) was run on scrambled HTML and returns
  noise with dozens of lossy repairs; it is not a fix. Other types have no repair engine.
- Next places to look for those files: the original export zips (`...NXPlelIY.zip` is listed in the NXPlelIY
  folder), the D: and F: drive copies, Google Drive and OneDrive, using the sha1 and name in List 2.
