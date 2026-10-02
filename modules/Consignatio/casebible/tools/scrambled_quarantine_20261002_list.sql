-- Byline: Claude Code · Sonnet · 2026-10-02
-- Scrambled-object quarantine: the apply list for b2_version_ops_20261001.py (mode quarantine).
-- Read only. Columns are the tool's list format (tab CSV with header): file_id, src_key, size, sha1, dest_key.
--
-- Rows: scrambled objects that B2 confirmed 2026-10-02 (a visible object at the exact key with the catalog's
-- size AND SHA-1, so the key still holds the scrambled bytes), status
--   dry_run             scrambled, with an intact same-name same-size twin of another hash
--   unreadable_no_twin  a format-marker extension (png, html, pdf, mp4 ...) whose head is random bytes, no twin
-- 'suspect_no_twin' (unknown extensions: signatures, keystores) is never listed. An intact twin is never
-- listed: only keys in raw_duck.scrambled_objects_20261002 whose sha1 probed no_marker_high are.
--
-- The tool copies each object server-side (b2_copy_file, no download) to dest_key, verifies the copy's size and
-- SHA-1 against the source version, and only then hides the source key; B2 keeps the original bytes as a
-- noncurrent version (the bucket has no lifecycle rule, so versions are kept; reversible by deleting the hide
-- marker). Run: psql -At -F '	' -f scrambled_quarantine_20261002_list.sql > scrambled.tsv  (on ovh-files)

SELECT 'file_id', 'src_key', 'size', 'sha1', 'dest_key'
UNION ALL
SELECT s.b2_file_id, s.full_key, s.size::text, s.sha1, s.quarantine_key
FROM raw_duck.scrambled_objects_20261002 s
WHERE s.exists_in_b2 AND s.status IN ('dry_run', 'unreadable_no_twin') AND s.b2_file_id IS NOT NULL
	-- safety: a hash that is some other scrambled object's intact twin is never moved
	AND s.sha1 NOT IN (SELECT twin_sha1 FROM raw_duck.scrambled_objects_20261002 WHERE twin_sha1 IS NOT NULL);
