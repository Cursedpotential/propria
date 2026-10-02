// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// The Facebook Messenger structured-ELT template (facebook_messenger_json_v1).
// It parses one thread file (message_N.json) of a Facebook "Download your
// information" export and does nothing more: one MESSAGE row per entry of
// messages[], oldest first, in the same native_fields shape the SMS routes
// emit (record_kind, body, sender, recipients, participants, occurred_at,
// attachments), so the generic normalizer, participant resolution, the
// Weaviate-first stage and the first-party split treat it exactly like an SMS.
//
// Attachments are plain files beside the thread file in the export (no base64
// to decode). Each is linked by its locator under the source's configured
// scheme, the way the SMS route links each decoded attachment by its
// media-object locator. The export writes them as paths that end in
// <thread folder>/<kind>/<file>; the part after the thread folder is resolved
// against the folder holding the thread file.
package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
)

// structuredELTQueryFor is structuredELTQuery plus the source's own locator,
// which only the Facebook Messenger template needs.
func structuredELTQueryFor(format activities.StructuredELTFormat, sourceURL, sourceLocator string) (string, error) {
	if format != activities.StructuredELTFormatFacebookMessenger {
		return structuredELTQuery(format, sourceURL)
	}
	if strings.TrimSpace(sourceURL) == "" {
		return "", errors.New("structured elt requires a non-empty DuckDB source url")
	}
	slash := strings.LastIndex(sourceLocator, "/")
	if slash < 0 || !strings.Contains(sourceLocator, "://") {
		return "", fmt.Errorf("facebook messenger source locator %q has no folder", sourceLocator)
	}
	return facebookMessengerQuery(sourceURL, sourceLocator[:slash+1]), nil
}

// fbMojibakeFix undoes the export's text encoding: every non-ASCII character
// is written as one \u00XX escape per UTF-8 byte, so a right single quote
// arrives as three characters U+00E2 U+0080 U+0099. When every character of a
// value is at most U+00FF, its code points ARE the original UTF-8 bytes and
// are re-read as UTF-8. A value with any character above U+00FF, or whose
// bytes are not valid UTF-8, is kept exactly as written. stored_bytes always
// keeps the message exactly as the export wrote it.
const fbMojibakeFix = `CASE WHEN %[1]s IS NULL OR regexp_matches(%[1]s, '[^\x00-\xff]') THEN %[1]s
	ELSE coalesce(try(decode(from_hex(array_to_string(list_transform(string_split(%[1]s, ''),
		c -> lpad(to_hex(unicode(c)), 2, '0')), '')))), %[1]s) END`

func facebookMessengerQuery(sourceURL, sourceFolder string) string {
	url := strings.ReplaceAll(sourceURL, "'", "''")
	folder := strings.ReplaceAll(sourceFolder, "'", "''")
	fix := func(column string) string { return fmt.Sprintf(fbMojibakeFix, column) }
	return fmt.Sprintf(`
			WITH source_document AS (
				SELECT content::JSON AS document FROM read_text('%[1]s')
			), thread AS (
				SELECT CAST(json_extract(document, '$.messages[*]') AS JSON[]) AS messages,
					list_transform(coalesce(json_extract_string(document, '$.participants[*].name'), []), p -> %[3]s) AS participants,
					coalesce(regexp_extract(json_extract_string(document, '$.thread_path'), '([^/]+)$', 1), '') AS thread_dir,
					json_extract_string(document, '$.thread_path') AS thread_path,
					json_extract_string(document, '$.title') AS title
				FROM source_document
			), messages AS (
				SELECT participants, thread_dir, thread_path, title,
					unnest(messages) AS message, generate_subscripts(messages, 1) AS message_index
				FROM thread
			), shaped AS (
				SELECT *,
					json_extract_string(message, '$.sender_name') AS sender_raw,
					try_cast(json_extract_string(message, '$.timestamp_ms') AS BIGINT) AS ts_ms,
					json_extract_string(message, '$.content') AS content_raw,
					list_concat(
						coalesce(json_extract_string(message, '$.photos[*].uri'), []),
						coalesce(json_extract_string(message, '$.videos[*].uri'), []),
						coalesce(json_extract_string(message, '$.audio_files[*].uri'), []),
						coalesce(json_extract_string(message, '$.files[*].uri'), []),
						coalesce(json_extract_string(message, '$.gifs[*].uri'), [])
					) AS attachment_uris
				FROM messages
			), fixed AS (
				SELECT *, %[4]s AS body, %[5]s AS sender,
					list_transform(attachment_uris, u -> CASE WHEN thread_dir <> '' AND strpos(u, thread_dir || '/') > 0
						THEN substr(u, strpos(u, thread_dir || '/') + length(thread_dir) + 1) ELSE u END) AS attachment_paths
				FROM shaped
			)
			SELECT message::VARCHAR AS stored_bytes,
				json_object(
					'record_kind', 'message', 'body', coalesce(body, ''), 'sender', sender,
					'recipients', list_filter(participants, p -> p IS DISTINCT FROM sender),
					'participants', participants,
					'occurred_at', CASE WHEN ts_ms IS NULL THEN NULL ELSE strftime(epoch_ms(ts_ms), '%%Y-%%m-%%dT%%H:%%M:%%S.%%fZ') END,
					'attachments', list_transform(range(len(attachment_paths)), i -> json_object(
						'ordinal', i,
						'name', regexp_extract(attachment_paths[i + 1], '([^/]+)$', 1),
						'mime', CASE lower(regexp_extract(attachment_paths[i + 1], '\.([A-Za-z0-9]+)$', 1))
							WHEN 'jpg' THEN 'image/jpeg' WHEN 'jpeg' THEN 'image/jpeg' WHEN 'png' THEN 'image/png'
							WHEN 'gif' THEN 'image/gif' WHEN 'webp' THEN 'image/webp' WHEN 'mp4' THEN 'video/mp4'
							WHEN 'mov' THEN 'video/quicktime' WHEN 'm4a' THEN 'audio/mp4' WHEN 'aac' THEN 'audio/aac'
							WHEN 'mp3' THEN 'audio/mpeg' WHEN 'wav' THEN 'audio/wav' WHEN 'pdf' THEN 'application/pdf'
							ELSE NULL END,
						'uri', '%[2]s' || attachment_paths[i + 1]))
				)::VARCHAR AS native_fields,
				json_object(
					'duckdb_template', 'facebook_messenger_json_v1', 'message_index', message_index,
					'thread_path', thread_path, 'thread_title', title, 'sender_name_raw', sender_raw,
					'timestamp_ms', ts_ms, 'source_row', message
				)::VARCHAR AS native_metadata
			FROM fixed
			ORDER BY ts_ms, message_index DESC`, url, folder, fix("p"), fix("content_raw"), fix("sender_raw"))
}
