-- Capture a conversation envelope and every native mapping slot without walking
-- the graph, so null slots, orphaned nodes, cycles and unknown fields survive.
-- Input: retained JSON array locator. Output: canonical JSON, native fields,
-- metadata, raw status and reason. No writes or byte offsets. The immutable
-- source version and original object remain the exact original-byte anchor.
-- Pick only for pinned chatgpt_json_array_v2 selections. Missing message dates
-- remain unknown; conversation dates never become message dates. The body is
-- a convenience projection; source_row contains every content part unchanged.
-- Byline: Codex · GPT-6.1-Sol · 2026-10-06.
WITH documents AS (
    SELECT content::JSON AS document FROM read_text('{{SOURCE}}')
), conversations AS (
    SELECT try_cast(entry.key AS BIGINT) AS conversation_index,
        entry.value AS conversation
    FROM documents, json_each(CASE WHEN json_type(document) = 'ARRAY'
        THEN document ELSE error('ChatGPT export requires an array') END) AS entry
), slots AS (
    SELECT conversation_index, conversation, entry.key AS node_key,
        entry.value AS node
    FROM conversations, json_each(CASE WHEN json_type(json_extract(conversation, '$.mapping')) = 'OBJECT'
        THEN json_extract(conversation, '$.mapping') ELSE json_object() END) AS entry
), messages AS (
    SELECT *, json_extract(node, '$.message') AS message,
        json_extract(node, '$.message.author.role') AS source_role
    FROM slots
), projected AS (
    SELECT *, coalesce((SELECT string_agg(CASE
            WHEN json_type(part.value) = 'VARCHAR' THEN json_extract_string(part.value, '$')
            WHEN json_type(json_extract(part.value, '$.text')) = 'VARCHAR' THEN json_extract_string(part.value, '$.text')
            ELSE '' END, chr(10) ORDER BY try_cast(part.key AS BIGINT))
        FROM json_each(CASE WHEN json_type(json_extract(message, '$.content.parts')) = 'ARRAY'
            THEN json_extract(message, '$.content.parts') ELSE json_array() END) AS part), '') AS body
    FROM messages
), rows AS (
    SELECT conversation_index, 0 AS row_group, '' AS node_key,
        CASE WHEN json_type(conversation) = 'OBJECT' AND json_type(json_extract(conversation, '$.mapping')) = 'OBJECT'
            THEN json_merge_patch(conversation, json_object('mapping', NULL))
            ELSE conversation END::VARCHAR AS stored_bytes,
        json_object()::VARCHAR AS native_fields,
        json_object('duckdb_template', 'chatgpt_json_array_v2',
            'stored_bytes_representation', 'duckdb_canonical_conversation_envelope_json',
            'source_byte_offsets_available', false, 'conversation_index', conversation_index,
            'source_row', CASE WHEN json_type(conversation) = 'OBJECT' AND json_type(json_extract(conversation, '$.mapping')) = 'OBJECT'
                THEN json_merge_patch(conversation, json_object('mapping', NULL)) ELSE conversation END,
            'mapping_slot_count', CASE WHEN json_type(json_extract(conversation, '$.mapping')) = 'OBJECT'
                THEN json_array_length(json_keys(json_extract(conversation, '$.mapping'))) ELSE 0 END)::VARCHAR AS native_metadata,
        CASE WHEN json_type(conversation) = 'OBJECT' AND json_type(json_extract(conversation, '$.mapping')) = 'OBJECT'
            THEN 'envelope' ELSE 'malformed' END AS record_status,
        CASE WHEN json_type(conversation) = 'OBJECT' AND json_type(json_extract(conversation, '$.mapping')) = 'OBJECT'
            THEN 'Native conversation envelope; mapping slots are captured separately'
            ELSE 'Conversation is not an object with an object mapping' END AS status_reason
    FROM conversations
    UNION ALL
    SELECT conversation_index, 1 AS row_group, node_key,
        node::VARCHAR AS stored_bytes,
        CASE WHEN json_type(message) = 'OBJECT' THEN json_object(
            'record_kind', 'message', 'body', body,
            'source_role', source_role, 'source_created_at', json_extract(message, '$.create_time'),
            'conversation_id', coalesce(json_extract(conversation, '$.conversation_id'), json_extract(conversation, '$.id')),
            'message_id', json_extract(message, '$.id')) ELSE json_object() END::VARCHAR AS native_fields,
        json_object('duckdb_template', 'chatgpt_json_array_v2',
            'stored_bytes_representation', 'duckdb_canonical_native_mapping_slot_json',
            'source_byte_offsets_available', false, 'conversation_index', conversation_index,
            'mapping_key', node_key, 'source_row', node)::VARCHAR AS native_metadata,
        CASE WHEN json_type(message) = 'OBJECT' THEN 'parsed'
            WHEN message IS NULL OR json_type(message) = 'NULL' THEN 'envelope'
            ELSE 'unknown' END AS record_status,
        CASE WHEN json_type(message) = 'OBJECT' THEN ''
            WHEN message IS NULL OR json_type(message) = 'NULL' THEN 'Mapping slot has no native message object'
            ELSE 'Mapping slot message has an unsupported native shape' END AS status_reason
    FROM projected
)
SELECT stored_bytes, native_fields, native_metadata, record_status, status_reason
FROM rows ORDER BY conversation_index, row_group, node_key;
