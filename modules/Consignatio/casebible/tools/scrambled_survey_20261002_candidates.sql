-- Byline: Claude Code · Sonnet · 2026-10-02
-- Scrambled-object survey, step 1: the objects whose head bytes must be read.
--
-- Background (2026-10-02): five casevault HTML files (Facebook account_activity, your_friends,
-- your_post_audiences, 30.html and a Google Takeout MyActivity.html) turned out to be scrambled bytes
-- (entropy 7.99 bits/byte, no format marker, no compression) whose catalog sha1 equals the scrambled
-- bytes, each with an intact same-size twin of a different hash.
--
-- A scrambled copy is only provable by reading its head, so the catalog cannot answer it alone; it can
-- only say WHICH objects to read. Two sets are read, one ranged GET of 4 KiB per distinct sha1:
--   (a) every (file name, size) group that holds more than one sha1 (a scrambled copy and its twin
--       share name and size and differ in hash);
--   (b) every object of the Facebook export folder facebook-potentiallycursed85-2025-08-18-NXPlelIY,
--       twin or not (the folder where the first four were found).
-- Objects under 1 KiB are skipped: entropy of a head that short proves nothing.
--
-- Output: one JSON object per distinct sha1 with up to 8 catalog keys to try (the catalog lists copies
-- that no longer exist in B2; the probe uses the first key that answers).
-- Run: psql -At -f scrambled_survey_20261002_candidates.sql > candidates.ndjson (on ovh-files, read only).

WITH multi AS (
	SELECT regexp_replace(key, '^.*/', '') AS name, size
	FROM raw_duck.vault_objects
	WHERE size >= 1024
	GROUP BY 1, 2
	HAVING count(DISTINCT sha1) > 1
), picked AS (
	SELECT v.sha1, v.size, regexp_replace(v.key, '^.*/', '') AS name, v.key,
		(v.key LIKE '%NXPlelIY%') AS in_nxplel, true AS in_multi
	FROM raw_duck.vault_objects v
	JOIN multi m ON m.name = regexp_replace(v.key, '^.*/', '') AND m.size = v.size
	WHERE v.sha1 IS NOT NULL
	UNION ALL
	SELECT v.sha1, v.size, regexp_replace(v.key, '^.*/', ''), v.key, true, false
	FROM raw_duck.vault_objects v
	WHERE v.key LIKE '%NXPlelIY%' AND v.size >= 1024 AND v.sha1 IS NOT NULL
)
SELECT json_build_object(
	'sha1', sha1, 'size', max(size), 'name', min(name),
	'in_nxplel', bool_or(in_nxplel), 'in_multi', bool_or(in_multi),
	'keys', (array_agg(DISTINCT key))[1:8],
	'copies', count(DISTINCT key)
)::text
FROM picked
GROUP BY sha1;
