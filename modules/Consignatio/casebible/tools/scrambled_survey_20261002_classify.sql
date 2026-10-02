-- Byline: Claude Code · Sonnet · 2026-10-02
-- Scrambled-object survey, step 3b: classify the head probe. Read-only on the catalog tables it reads
-- (vault_objects, scramble_head_probe_20261002); it only creates raw_duck.scrambled_objects_20261002.
-- Moves nothing.
--
-- Input  raw_duck.scramble_head_probe_20261002  (scrambled_survey_20261002_heads_to_sql.py output, loaded first)
-- Run:   psql -U postgres -d casebible -v ON_ERROR_STOP=1 -f scrambled_survey_20261002_classify.sql
--
-- raw_duck.scrambled_objects_20261002   one row per CATALOG KEY of a scrambled object, with the intact
--                                       twin and the quarantine destination. status 'dry_run'.
--
-- A sha1 is SCRAMBLED when its head has no format marker, is not text and has the entropy of random data
-- (class no_marker_high) AND another sha1 of the same file name and size has an intact head (marker_ok
-- or text): the pair that was found on 2026-10-02 (a scrambled copy and its same-size twin of a
-- different hash) -> status 'dry_run'. A no_marker_high head WITHOUT an intact twin is
--   * 'unreadable_no_twin' when the extension names a format that always starts with a marker or clean text
--     (png, jpg, html, pdf, mp4, ...): a .png of random bytes is unreadable whatever else exists;
--   * 'suspect_no_twin' for any other extension (signatures, keystores, unknown): encrypted files look the
--     same, so they are listed but are never part of a quarantine apply.

BEGIN;

DROP TABLE IF EXISTS raw_duck.scrambled_objects_20261002;
CREATE TABLE raw_duck.scrambled_objects_20261002 (
	key text NOT NULL,              -- catalog key (relative to consignatio/vault/v1/)
	full_key text NOT NULL,         -- B2 key in bucket salem-data
	size bigint NOT NULL,
	sha1 text NOT NULL,             -- the scrambled bytes' sha1
	name text NOT NULL,
	ext text,
	entropy numeric,
	status text NOT NULL,           -- dry_run | unreadable_no_twin | suspect_no_twin  (apply sets quarantined)
	twin_sha1 text,
	twin_key text,                  -- full B2 key of a verified-readable intact twin, null when none
	twin_probe_class text,
	quarantine_key text NOT NULL,   -- consignatio/_quarantine/scrambled-20261002/<full_key>
	in_nxplel boolean NOT NULL,
	exists_in_b2 boolean,           -- filled by scrambled_survey_20261002_exists.py
	listed_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (key)
);

WITH bad AS (
	SELECT * FROM raw_duck.scramble_head_probe_20261002 WHERE probe_class = 'no_marker_high'
), twins AS (
	-- the intact twin of a bad sha1: same name and size, different sha1, probe class marker_ok or text.
	-- When several qualify, the one with the most catalog copies, then the lexicographically first key.
	SELECT DISTINCT ON (b.sha1) b.sha1 AS bad_sha1, t.sha1 AS twin_sha1, t.key_used AS twin_key, t.probe_class AS twin_class
	FROM bad b
	JOIN raw_duck.scramble_head_probe_20261002 t
		ON t.name = b.name AND t.size = b.size AND t.sha1 <> b.sha1 AND t.probe_class IN ('marker_ok', 'text') AND t.key_used IS NOT NULL
	ORDER BY b.sha1, t.copies DESC, t.key_used
)
INSERT INTO raw_duck.scrambled_objects_20261002 (key, full_key, size, sha1, name, ext, entropy, status, twin_sha1, twin_key, twin_probe_class, quarantine_key, in_nxplel)
SELECT v.key, 'consignatio/vault/v1/' || v.key, v.size, v.sha1, b.name, b.ext, b.entropy,
	CASE WHEN tw.twin_sha1 IS NOT NULL THEN 'dry_run'
		WHEN lower(b.ext) IN ('png','jpg','jpeg','gif','webp','heic','heif','mp4','mov','m4a','mp3','wav','pdf','zip','gz','7z','rar','tar',
			'html','htm','xml','json','csv','txt','md','docx','xlsx','pptx','xls','xlsm','doc','ppt','avi','mkv','webm','flac','ogg','svg','bmp','tif','tiff','ico')
			THEN 'unreadable_no_twin'
		ELSE 'suspect_no_twin' END,
	tw.twin_sha1, CASE WHEN tw.twin_key IS NOT NULL THEN 'consignatio/vault/v1/' || tw.twin_key END, tw.twin_class,
	'consignatio/_quarantine/scrambled-20261002/consignatio/vault/v1/' || v.key,
	v.key LIKE '%NXPlelIY%'
FROM bad b
JOIN raw_duck.vault_objects v ON v.sha1 = b.sha1
LEFT JOIN twins tw ON tw.bad_sha1 = b.sha1
ON CONFLICT (key) DO NOTHING;

COMMENT ON TABLE raw_duck.scrambled_objects_20261002 IS
	'Scrambled copies found 2026-10-02 (random-looking head, no format marker, intact same-size twin of another hash). Dry-run list for quarantine to consignatio/_quarantine/scrambled-20261002/; nothing moved. Built by scrambled_survey_20261002_*.';

COMMIT;

-- Summary (read-only)
SELECT status, count(*) AS keys, count(DISTINCT sha1) AS distinct_hashes, sum(size) AS bytes,
	count(*) FILTER (WHERE twin_sha1 IS NOT NULL) AS with_twin, count(*) FILTER (WHERE in_nxplel) AS in_nxplel
FROM raw_duck.scrambled_objects_20261002 GROUP BY status ORDER BY status;
SELECT probe_class, count(*) FROM raw_duck.scramble_head_probe_20261002 GROUP BY 1 ORDER BY 2 DESC;
