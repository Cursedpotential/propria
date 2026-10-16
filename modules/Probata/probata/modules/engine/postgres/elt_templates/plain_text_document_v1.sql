-- Byline: Claude Code · Opus 5.5 · 2026-10-15
-- plain_text_document_v1: any text file that no more specific signature claims (owner decision
-- 2026-10-15: unclaimed text imports as a plain document; every text parser stays selectable).
-- One row per paragraph, in file order. A paragraph is the text between blank lines (a line that is
-- empty or holds only spaces/tabs). Line endings are normalized to \n and a leading byte-order mark is
-- dropped; otherwise each paragraph's text is kept exactly as written, inner line breaks included.
--
-- native_fields has record_kind 'object' and carries the text under doc_text (never body/text), so the
-- generic normalizer keeps it as record_type other with the native fields verbatim; nothing here is
-- presented as a message. A file that is not valid UTF-8 is read byte-for-byte (each byte becomes the
-- character with that code) and source_encoding says so, so one stray byte never fails the file.
WITH raw_bytes AS (
	SELECT content AS bytes FROM read_blob('{{SOURCE}}')
), decoded AS (
	SELECT bytes, try(decode(bytes)) AS utf8_text FROM raw_bytes
), source_text AS (
	SELECT CASE WHEN utf8_text IS NOT NULL THEN utf8_text
		ELSE array_to_string(list_transform(regexp_extract_all(hex(bytes), '..'), h -> chr(('0x' || h)::INTEGER)), '') END AS content,
		CASE WHEN utf8_text IS NOT NULL THEN 'utf-8' ELSE 'byte-per-character fallback (not valid UTF-8)' END AS source_encoding
	FROM decoded
), normalized AS (
	SELECT regexp_replace(replace(replace(content, chr(13) || chr(10), chr(10)), chr(13), chr(10)), '^\x{FEFF}', '') AS content,
		source_encoding
	FROM source_text
), split AS (
	SELECT unnest(string_split_regex(content, '\n[ \t]*\n')) AS paragraph,
		generate_subscripts(string_split_regex(content, '\n[ \t]*\n'), 1) AS split_order,
		source_encoding
	FROM normalized
), kept AS (
	SELECT trim(paragraph, chr(10)) AS paragraph, split_order, source_encoding
	FROM split
	WHERE trim(paragraph) <> ''
), ordered AS (
	SELECT row_number() OVER (ORDER BY split_order) AS block_order, paragraph, source_encoding FROM kept
)
SELECT json_object('block_order', block_order, 'element', 'paragraph', 'text', paragraph)::VARCHAR AS stored_bytes,
	json_object(
		'record_kind', 'object', 'doc_element', 'paragraph', 'doc_text', paragraph,
		'doc_links', []::VARCHAR[], 'doc_media', []::VARCHAR[]
	)::VARCHAR AS native_fields,
	json_object('duckdb_template', 'plain_text_document_v1', 'block_order', block_order,
		'source_encoding', source_encoding)::VARCHAR AS native_metadata
FROM ordered
ORDER BY block_order
