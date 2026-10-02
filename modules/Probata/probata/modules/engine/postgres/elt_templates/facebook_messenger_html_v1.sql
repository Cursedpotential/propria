-- Byline: Claude Code · Sonnet · 2026-10-02
-- facebook_messenger_html_v1: one thread file (message_N.html) of a Facebook "Download your information"
-- export, read with the DuckDB webbed extension (parse_html + XPath). One row per message block, oldest
-- first, in the SMS native_fields shape the SMS routes and facebook_messenger_json_v1 emit, so the generic
-- normalizer, participant resolution, the Weaviate-first stage and the first-party split treat it alike.
--
-- Two export layouts are read by the same XPath (class tests are token-exact):
--   2024: div._a6-g > div._2ph_._a6-h (sender) + div._2ph_._a6-p (body) + div._3-94._a6-o (timestamp)
--   2025: section._a6-g > h2._a6-h + div._a6-p + footer._a6-o
-- Reactions are li elements of ul._a6-q: the emoji, the reactor's name and, in the 2025 layout, the time in brackets.
--
-- Text is real UTF-8 in the HTML export (unlike the JSON export, whose strings are one \u00XX escape per
-- UTF-8 byte), so emoji, ZWJ sequences and accents arrive intact and no mojibake repair is applied.
--
-- The timestamp has no zone. Facebook writes the requester's local time (the export header states the
-- UTC offset; this corpus is Eastern), so it is read as America/Detroit and recorded as inferred, the same
-- ruling as the Case Bible elt_fb_messenger_html_v1 (2026-09-18). The HTML carries whole seconds only; the
-- JSON export of the same thread carries milliseconds, so the two do not collapse onto each other.
--
-- A block without a timestamp (thread header, group-name and invite-link notices) is not a message: it is
-- emitted as record_kind 'object' so no block of the file is dropped.
WITH raw_bytes AS (
	SELECT content AS bytes FROM read_blob('{{SOURCE}}')
), decoded AS (
	SELECT bytes, try(decode(bytes)) AS utf8_text FROM raw_bytes
), source_text AS (
	-- UTF-8 when the file is valid UTF-8; otherwise each byte becomes the character with that code (a stray
	-- Windows-1252 byte inside a script block must not fail the whole file).
	SELECT CASE WHEN utf8_text IS NOT NULL THEN utf8_text
		ELSE array_to_string(list_transform(regexp_extract_all(hex(bytes), '..'), h -> chr(('0x' || h)::INTEGER)), '') END AS content
	FROM decoded
), source_document AS (
	SELECT parse_html(content) AS h FROM source_text
), block_list AS (
	SELECT xml_extract_elements(h, '//*[contains(concat(" ",normalize-space(@class)," ")," _a6-g ")]') AS blocks,
		trim(regexp_replace(array_to_string(coalesce(
			nullif(xml_extract_text(h::VARCHAR, '//*[contains(concat(" ",normalize-space(@class)," ")," _a70e ")]//text()'), []),
			xml_extract_text(h::VARCHAR, '//h1[1]//text()')), ' '), '\s+', ' ', 'g')) AS title
	FROM source_document
), blocks AS (
	SELECT title, unnest(blocks) AS block, generate_subscripts(blocks, 1) AS block_index FROM block_list
), cells AS (
	SELECT title, block_index, block,
		trim(regexp_replace(array_to_string(xml_extract_text(block, '/*/*[contains(concat(" ",normalize-space(@class)," ")," _a6-h ")]//text()'), ' '), '\s+', ' ', 'g')) AS sender_raw,
		trim(regexp_replace(array_to_string(xml_extract_text(block, '/*/*[contains(concat(" ",normalize-space(@class)," ")," _a6-o ")]//text()'), ' '), '\s+', ' ', 'g')) AS ts_raw,
		trim(regexp_replace(array_to_string(xml_extract_text(block, '/*/*[contains(concat(" ",normalize-space(@class)," ")," _a6-p ")]//text()[not(ancestor::ul)]'), ' '), '\s+', ' ', 'g')) AS body_raw,
		trim(regexp_replace(array_to_string(xml_extract_text(block, '//text()'), ' '), '\s+', ' ', 'g')) AS block_text,
		xml_extract_text(block, '//ul[contains(concat(" ",normalize-space(@class)," ")," _a6-q ")]/li') AS reaction_raw,
		list_filter(list_distinct(list_concat(
			coalesce(xml_extract_text(block, '//a/@href'), []),
			coalesce(xml_extract_text(block, '//*[@src]/@src'), []))),
			u -> u <> '' AND NOT starts_with(u, 'data:') AND NOT regexp_matches(u, '^[a-zA-Z][a-zA-Z0-9+.-]*:')) AS media_paths
	FROM blocks
), stamped AS (
	SELECT *, nullif(sender_raw, '') AS sender,
		try_strptime(regexp_replace(ts_raw, '\s*([AaPp][Mm])$', ' \1'), '%b %d, %Y %I:%M:%S %p') AS ts_local
	FROM cells
), thread AS (
	SELECT list(DISTINCT sender) FILTER (WHERE ts_raw <> '' AND sender IS NOT NULL) AS participants FROM stamped
)
SELECT block::VARCHAR AS stored_bytes,
	CASE WHEN ts_raw <> '' THEN json_object(
		'record_kind', 'message', 'body', body_raw, 'sender', sender,
		'recipients', list_filter(participants, p -> p IS DISTINCT FROM sender),
		'participants', participants,
		'occurred_at', CASE WHEN ts_local IS NULL THEN NULL
			ELSE strftime((ts_local AT TIME ZONE 'America/Detroit') AT TIME ZONE 'UTC', '%Y-%m-%dT%H:%M:%SZ') END,
		'attachments', list_transform(range(len(media_paths)), i -> json_object(
			'ordinal', i,
			'name', regexp_extract(media_paths[i + 1], '([^/]+)$', 1),
			'mime', CASE lower(regexp_extract(media_paths[i + 1], '\.([A-Za-z0-9]+)$', 1))
				WHEN 'jpg' THEN 'image/jpeg' WHEN 'jpeg' THEN 'image/jpeg' WHEN 'png' THEN 'image/png'
				WHEN 'gif' THEN 'image/gif' WHEN 'webp' THEN 'image/webp' WHEN 'heic' THEN 'image/heic'
				WHEN 'mp4' THEN 'video/mp4' WHEN 'mov' THEN 'video/quicktime' WHEN 'webm' THEN 'video/webm'
				WHEN 'm4a' THEN 'audio/mp4' WHEN 'aac' THEN 'audio/aac' WHEN 'mp3' THEN 'audio/mpeg'
				WHEN 'wav' THEN 'audio/wav' WHEN 'ogg' THEN 'audio/ogg' WHEN 'opus' THEN 'audio/opus'
				WHEN 'pdf' THEN 'application/pdf' ELSE NULL END,
			'uri', '{{EXPORT_ROOT}}' || media_paths[i + 1])),
		'reactions', list_transform(coalesce(reaction_raw, []), r -> json_object(
			'reaction', nullif(regexp_extract(r, '^([\p{So}\p{Sk}\p{Mn}\p{Cf}\p{Sm}\p{Sc}]+)', 1), ''),
			'by', trim(regexp_replace(regexp_replace(r, '^[\p{So}\p{Sk}\p{Mn}\p{Cf}\p{Sm}\p{Sc}]+', ''), '\s*\([^()]*\)\s*$', '')),
			'at_raw', nullif(regexp_extract(r, '\(([^()]*)\)\s*$', 1), '')))
	) ELSE json_object('record_kind', 'object', 'block_text', block_text) END::VARCHAR AS native_fields,
	json_object(
		'duckdb_template', 'facebook_messenger_html_v1', 'block_index', block_index,
		'thread_dir', '{{THREAD_DIR}}', 'thread_title', nullif(title, ''), 'sender_name_raw', sender_raw,
		'timestamp_raw', nullif(ts_raw, ''), 'timestamp_zone_basis', 'America/Detroit (assumed: the file carries no zone)',
		'timestamp_parsed', ts_local IS NOT NULL
	)::VARCHAR AS native_metadata
FROM stamped CROSS JOIN thread
ORDER BY ts_local NULLS FIRST, block_index DESC
