-- Byline: Claude Code · Sonnet · 2026-10-02
-- generic_html_document_v1: any HTML file that is not a more specific signature, read with the DuckDB
-- webbed extension (parse_html + XPath; the same functions exist in the DuckDB 1.4.3 build inside
-- pg_duckdb and in 1.5.x). One row per text block of the body, in document order:
--   * a "leaf" block (p, h1-h6, li, td, th, pre, blockquote, dd, dt, figcaption, caption, summary, address,
--     and any div/section/article/aside/header/footer/main/nav/form with no block inside it) contributes all
--     of its text, so link text and emphasis stay inside the sentence that holds them;
--   * a container block that has block children contributes only its own direct text, so text sitting
--     beside nested blocks is kept and nothing is counted twice.
-- script, style and noscript content never reaches the rows. An XHTML namespace declaration is removed
-- and so is the DOCTYPE before parsing (Google Voice and WhatsApp exports are XHTML: with the namespace or the
-- XHTML DOCTYPE left in, XPath without a prefix sees no element at all).
--
-- native_fields has record_kind 'object' and carries the text under doc_text (never body/text), so the
-- generic normalizer keeps it as record_type other with the native fields verbatim; nothing here is
-- presented as a message. Link targets and media sources are kept per block.
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
	SELECT content, parse_html(regexp_replace(regexp_replace(content, '<!DOCTYPE[^>]*>', '', 'i'), '(<html[^>]*?)\s+xmlns="[^"]*"', '\1', 'i')) AS h FROM source_text
), picked AS (
	SELECT unnest(xml_extract_elements(h,
		'//body//*[self::p or self::h1 or self::h2 or self::h3 or self::h4 or self::h5 or self::h6 or self::li or self::td or self::th or self::pre or self::blockquote or self::div or self::section or self::article or self::aside or self::header or self::footer or self::main or self::nav or self::dd or self::dt or self::figcaption or self::caption or self::summary or self::address or self::form][not(ancestor::script or ancestor::style or ancestor::noscript)]')) AS block,
		generate_subscripts(xml_extract_elements(h,
		'//body//*[self::p or self::h1 or self::h2 or self::h3 or self::h4 or self::h5 or self::h6 or self::li or self::td or self::th or self::pre or self::blockquote or self::div or self::section or self::article or self::aside or self::header or self::footer or self::main or self::nav or self::dd or self::dt or self::figcaption or self::caption or self::summary or self::address or self::form][not(ancestor::script or ancestor::style or ancestor::noscript)]'), 1) AS block_order
	FROM source_document
), shaped AS (
	SELECT block_order, block,
		regexp_extract(block::VARCHAR, '^<([A-Za-z0-9]+)', 1) AS element,
		len(xml_extract_elements(block, '/*//*[self::p or self::h1 or self::h2 or self::h3 or self::h4 or self::h5 or self::h6 or self::li or self::td or self::th or self::pre or self::blockquote or self::div or self::section or self::article or self::aside or self::header or self::footer or self::main or self::nav or self::dd or self::dt or self::figcaption or self::caption or self::summary or self::address or self::form]')) = 0 AS is_leaf
	FROM picked
), texted AS (
	SELECT block_order, block, element,
		trim(regexp_replace(array_to_string(
			CASE WHEN is_leaf THEN xml_extract_text(block, '//text()[not(ancestor::script or ancestor::style or ancestor::noscript)]')
				ELSE xml_extract_text(block, '/*/text()') END, ' '), '\s+', ' ', 'g')) AS block_text,
		CASE WHEN is_leaf THEN xml_extract_text(block, '//a/@href') ELSE [] END AS hrefs,
		CASE WHEN is_leaf THEN xml_extract_text(block, '//*[@src and not(starts-with(@src,"data:"))]/@src') ELSE [] END AS srcs
	FROM shaped
), document AS (
	SELECT nullif(trim(html_unescape(regexp_extract(content, '(?is)<title[^>]*>(.*?)</title>', 1))), '') AS title FROM source_document
), kept AS (
	SELECT * FROM texted WHERE block_text <> '' OR len(srcs) > 0
)
SELECT json_object('block_order', block_order, 'element', element, 'text', block_text)::VARCHAR AS stored_bytes,
	json_object(
		'record_kind', 'object', 'doc_element', element, 'doc_text', block_text,
		'doc_links', hrefs, 'doc_media', srcs
	)::VARCHAR AS native_fields,
	json_object('duckdb_template', 'generic_html_document_v1', 'block_order', block_order, 'document_title', document.title)::VARCHAR AS native_metadata
FROM kept CROSS JOIN document
ORDER BY block_order
