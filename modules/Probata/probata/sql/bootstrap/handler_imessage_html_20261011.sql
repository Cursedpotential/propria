-- Admit the registered iMessage HTML parser's format in the live handler signature ledger.
-- Inputs: the existing context.handler_detected_format CHECK with its 17 verified formats.
-- Outputs: the same constraint with imessage_html added; existing rows are unchanged.
-- Side effects: one transactional DDL change. Use before deploying iMessage HTML detection.
DO $migration$
DECLARE
    actual text[];
    expected text[] := ARRAY[
        'archive', 'binary', 'callsbackuprestore_xml', 'chatgpt_official_json',
        'claude_ai_export_json', 'csv', 'docx', 'facebook_messenger_html',
        'facebook_messenger_json', 'generic_html_document', 'json',
        'messages_transcript', 'ndjson', 'pdf', 'smsbackuprestore_xml', 'text', 'xml'
    ];
BEGIN
    SELECT array_agg(DISTINCT match[1] ORDER BY match[1])
      INTO actual
      FROM pg_constraint c,
           regexp_matches(pg_get_constraintdef(c.oid), '''([^'']+)''::text', 'g') AS match
     WHERE c.conrelid = 'context.handler_detected_format'::regclass
       AND c.conname = 'handler_detected_format_format_id_check';

    IF 'imessage_html' = ANY (actual) THEN
        RETURN;
    END IF;
    IF actual IS DISTINCT FROM expected THEN
        RAISE EXCEPTION 'handler format constraint drifted; observed: %', actual;
    END IF;

    ALTER TABLE context.handler_detected_format
        DROP CONSTRAINT handler_detected_format_format_id_check;
    ALTER TABLE context.handler_detected_format
        ADD CONSTRAINT handler_detected_format_format_id_check
        CHECK (format_id = ANY (ARRAY[
            'archive', 'binary', 'callsbackuprestore_xml', 'chatgpt_official_json',
            'claude_ai_export_json', 'csv', 'docx', 'facebook_messenger_html',
            'facebook_messenger_json', 'generic_html_document', 'imessage_html',
            'json', 'messages_transcript', 'ndjson', 'pdf',
            'smsbackuprestore_xml', 'text', 'xml'
        ]));
END
$migration$;
