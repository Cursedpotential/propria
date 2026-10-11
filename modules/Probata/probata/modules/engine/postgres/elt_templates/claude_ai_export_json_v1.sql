-- Byline: Codex · GPT-6 · 2026-10-03.
-- Preserve every Claude conversation envelope and native message, including empty conversations.
-- Envelope correction: Codex · GPT-6.1-Sol · 2026-10-06.
-- Input: {{SOURCE}}, escaped by claudeAIExportQuery. Output: five
-- text columns, with canonical native message JSON as stored_bytes. No writes.
-- DuckDB canonicalizes nested JSON; stored_bytes is semantically source-native,
-- not a byte-identical source fragment, and no source byte offset is claimed.
-- Whole original bytes remain in the immutable retained object linked by the
-- bundle's source_version_ref and the parser execution/source receipts.
-- Pick this for claude.ai exports carrying chat_messages, never Markdown.
-- Text precedence matches transcripts.claude-ai-export: nonempty text first,
-- then joined text blocks. Unlike that legacy normalized-only parser, empty,
-- attachment-only and tool messages are retained and counted. No owner identity
-- or assistant identity is inferred from a source's sender role.
-- Existing SQL sibling: Consignatio/casebible/tools/comm_timeline_mvp/elt/
-- elt_ai_claude_v1.sql, _envelope.sql and run_ai_elt.py. This engine template
-- retains their source-time/provenance discipline while adapting to the existing
-- immutable bundle contract. It does not insert ai_turns, mint fallback IDs,
-- synthesize an owner identity, skip empty bodies or invoke Python fallback.
WITH source_document AS (
    SELECT content::JSON AS document FROM read_text('{{SOURCE}}')
), conversations AS (
    SELECT try_cast(conversation.key AS BIGINT) AS conversation_index,
        conversation.value AS conversation, json_type(document) AS source_root_kind
    FROM source_document, json_each(CASE json_type(document)
        WHEN 'ARRAY' THEN document
        WHEN 'OBJECT' THEN json_array(document)
        ELSE error('Claude export requires an array or conversation object') END) AS conversation
), messages AS (
    SELECT conversation_index, conversation, source_root_kind,
        try_cast(entry.key AS BIGINT) AS message_index,
        CASE WHEN json_type(entry.value) = 'OBJECT' THEN entry.value
            ELSE error('Claude chat_messages entries must be objects') END AS message
    FROM conversations, json_each(CASE
        WHEN json_type(conversation) = 'OBJECT'
            AND json_type(json_extract(conversation, '$.chat_messages')) = 'ARRAY'
        THEN json_extract(conversation, '$.chat_messages')
        ELSE error('Claude conversation requires a chat_messages array') END) AS entry
), shaped AS (
    SELECT *, json_extract_string(message, '$.sender') AS sender,
        CASE WHEN json_type(json_extract(message, '$.text')) = 'VARCHAR'
            AND json_extract_string(message, '$.text') <> ''
            THEN json_extract_string(message, '$.text')
            ELSE regexp_replace(coalesce((SELECT string_agg(
                CASE WHEN json_type(json_extract(block.value, '$.text')) = 'VARCHAR'
                    THEN json_extract_string(block.value, '$.text') ELSE '' END,
                chr(10) ORDER BY try_cast(block.key AS BIGINT))
                FROM json_each(CASE WHEN json_type(json_extract(message, '$.content')) = 'ARRAY'
                    THEN json_extract(message, '$.content') ELSE json_array() END) AS block
                WHERE json_extract_string(block.value, '$.type') = 'text'), ''), '^\s+|\s+$', '', 'g') END AS body,
        coalesce(json_extract(message, '$.content'), json_array()) AS content_blocks,
        coalesce(json_extract(message, '$.attachments'), json_array()) AS attachments,
        coalesce(json_extract(message, '$.files'), json_array()) AS files
    FROM messages
), captured AS (
SELECT conversation_index, -1::BIGINT AS message_index,
    json_merge_patch(conversation, json_object('chat_messages', NULL))::VARCHAR AS stored_bytes,
    json_object('record_kind', 'conversation_envelope')::VARCHAR AS native_fields,
    json_object('duckdb_template', 'claude_ai_export_json_v1',
        'stored_bytes_representation', 'duckdb_canonical_conversation_envelope_json',
        'source_byte_offsets_available', false, 'conversation_index', conversation_index,
        'source_root_kind', source_root_kind,
        'source_row', json_merge_patch(conversation, json_object('chat_messages', NULL)))::VARCHAR AS native_metadata,
    'envelope' AS record_status, 'native_conversation_envelope' AS status_reason
FROM conversations
WHERE CASE WHEN json_type(conversation) = 'OBJECT'
    AND json_type(json_extract(conversation, '$.chat_messages')) = 'ARRAY' THEN true
    ELSE error('Claude conversation requires a chat_messages array') END
UNION ALL
SELECT conversation_index, message_index, message::VARCHAR AS stored_bytes,
    json_merge_patch(message, json_object(
        'record_kind', 'message', 'body', body,
        'role', CASE sender WHEN 'human' THEN 'user' WHEN 'assistant' THEN 'assistant' ELSE sender END,
        'conversation_id', json_extract_string(conversation, '$.uuid'),
        'conversation_title', json_extract_string(conversation, '$.name'),
        'message_id', json_extract_string(message, '$.uuid'),
        'sender', sender, 'participants', json_array(), 'recipients', json_array(),
        'occurred_at', json_extract_string(message, '$.created_at')
    ))::VARCHAR AS native_fields,
    json_object(
        'duckdb_template', 'claude_ai_export_json_v1',
        'stored_bytes_representation', 'duckdb_canonical_native_message_json',
        'source_byte_offsets_available', false,
        'conversation_index', conversation_index, 'message_index', message_index,
        'source_root_kind', source_root_kind,
        'conversation_id', json_extract_string(conversation, '$.uuid'),
        'conversation_title', json_extract_string(conversation, '$.name'),
        'conversation_metadata', json_merge_patch(conversation, json_object('chat_messages', NULL)),
        'message_id', json_extract_string(message, '$.uuid'),
        'sender', sender, 'created_at', json_extract_string(message, '$.created_at'),
        'text_empty', body = '',
        'content_block_count', coalesce(json_array_length(content_blocks), 0),
        'nontext_block_count', (SELECT count(*) FROM json_each(
            CASE WHEN json_type(content_blocks) = 'ARRAY' THEN content_blocks ELSE json_array() END) AS block
            WHERE json_extract_string(block.value, '$.type') IS DISTINCT FROM 'text'),
        'attachment_count', coalesce(json_array_length(attachments), 0),
        'file_count', coalesce(json_array_length(files), 0),
        'source_row', message
    )::VARCHAR AS native_metadata, 'parsed' AS record_status, '' AS status_reason
FROM shaped
)
SELECT stored_bytes, native_fields, native_metadata, record_status, status_reason
FROM captured
ORDER BY conversation_index, message_index;
