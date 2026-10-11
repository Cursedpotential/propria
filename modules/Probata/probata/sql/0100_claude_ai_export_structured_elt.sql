-- Byline: Codex · GPT-6 · 2026-10-03.
-- Native ChatGPT revision and declaration/status fences: Codex · GPT-6.1-Sol · 2026-10-06.
-- Admit native Claude export detection and its versioned structured-ELT template.
-- Inputs: existing handler_detected_format CHECK and raw-generation transition
-- function. Outputs: additive Claude admission. Side effects: DDL only; existing
-- formats and safeguards are preserved. Use after the schema baseline, never as
-- bootstrap. Apply through the owner's migration lane; parser code writes no DDL.
BEGIN;

DO $migration$
DECLARE
    definition text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT definition
    FROM pg_constraint
    WHERE conrelid = 'context.handler_detected_format'::regclass
      AND conname = 'handler_detected_format_format_id_check';
    IF strpos(definition, '''claude_ai_export_json''') = 0 THEN
        IF definition NOT LIKE 'CHECK (%' THEN
            RAISE EXCEPTION 'Unexpected detected-format constraint definition';
        END IF;
        definition := regexp_replace(definition, '^CHECK \(',
            'CHECK (format_id = ''claude_ai_export_json'' OR ');
        ALTER TABLE context.handler_detected_format
            DROP CONSTRAINT handler_detected_format_format_id_check;
        EXECUTE 'ALTER TABLE context.handler_detected_format ADD CONSTRAINT '
            || 'handler_detected_format_format_id_check ' || definition;
    END IF;
END;
$migration$;

-- Preserve the currently deployed function, including newer sibling template
-- admissions and all identity/hash/receipt guards. A changed seam fails closed
-- instead of replacing the function with an older bootstrap copy.
DO $migration$
DECLARE
    definition text;
    revised text;
BEGIN
    SELECT pg_get_functiondef('context.guard_raw_generation_transition()'::regprocedure)
        INTO definition;
    IF strpos(definition, '''claude_ai_export_json_v1''') = 0 THEN
        revised := regexp_replace(definition,
            'WHEN[[:space:]]+''chatgpt_official_json''[[:space:]]+THEN[[:space:]]+''chatgpt_json_array_v1''',
            'WHEN ''chatgpt_official_json'' THEN ''chatgpt_json_array_v1'''
            || ' WHEN ''claude_ai_export_json'' THEN CASE WHEN NEW.format_id IN '
            || '(''chatgpt_official_json'',''chatgpt_json_array'',''chatgpt_conversations_json'',''claude_ai_export_json'',''claude_conversations_json'',''gemini_activity_json'',''ai_markdown_transcript'',''ai_chat_file'',''ai_generic_json'',''ai_conversations_json'') '
            || 'THEN ''claude_ai_export_json_v1'' ELSE NULL END');
        IF revised = definition THEN
            RAISE EXCEPTION 'Missing ChatGPT template admission seam; review current raw-generation guard';
        END IF;
        definition := revised;
    END IF;
    IF strpos(definition, '''chatgpt_json_array_v2''') = 0 THEN
        revised := replace(definition,
            'WHEN ''chatgpt_official_json'' THEN ''chatgpt_json_array_v1''',
            'WHEN ''chatgpt_official_json'' THEN CASE WHEN '
            || 'verification.observed->>''duckdb_template''=''chatgpt_json_array_v2'' '
            || 'AND NEW.format_id IN (''chatgpt_official_json'',''chatgpt_json_array'',''chatgpt_conversations_json'',''claude_ai_export_json'',''claude_conversations_json'',''gemini_activity_json'',''ai_markdown_transcript'',''ai_chat_file'',''ai_generic_json'',''ai_conversations_json'') '
            || 'AND EXISTS (SELECT 1 FROM context.activity_receipt selected '
            || 'JOIN context.activity_execution selected_execution ON selected_execution.id=selected.activity_execution_id '
            || 'WHERE selected.status=''success'' AND selected_execution.source_version_id=NEW.source_version_id '
            || 'AND selected_execution.workflow_id=source.workflow_id '
            || 'AND selected_execution.activity_name=''select_parser_activity'' '
            || 'AND selected.completed_at<=NEW.created_at '
            || 'AND selected.result_ref->>''parser_id''=NEW.parser_id '
            || 'AND selected.result_ref->>''parser_version''=NEW.parser_version '
            || 'AND selected.result_ref->>''declared_format''=NEW.format_id '
            || 'AND selected.result_ref->>''duckdb_template''=''chatgpt_json_array_v2'') '
            || 'THEN ''chatgpt_json_array_v2'' ELSE ''chatgpt_json_array_v1'' END');
        IF revised = definition THEN
            RAISE EXCEPTION 'Missing ChatGPT template revision seam';
        END IF;
        definition := revised;
        revised := replace(definition, $old$raw.record_status=''parsed''$old$,
            $new$(raw.record_status=''parsed'' OR ($2=''chatgpt_json_array_v2''
            AND $3 IN (''chatgpt_official_json'',''chatgpt_json_array'',''chatgpt_conversations_json'',''claude_ai_export_json'',''claude_conversations_json'',''gemini_activity_json'',''ai_markdown_transcript'',''ai_chat_file'',''ai_generic_json'',''ai_conversations_json'')
            AND raw.record_status IN (''envelope'',''unknown'',''malformed'',''rejected'',''unparsed'')
            AND length(trim(COALESCE(raw.status_reason,'''')))>0))$new$);
        IF revised = definition THEN
            RAISE EXCEPTION 'Missing strict raw row status seam';
        END IF;
        definition := revised;
        revised := replace(definition, 'USING NEW.id, context_template;',
            'USING NEW.id, context_template, NEW.format_id;');
        IF revised = definition THEN
            RAISE EXCEPTION 'Missing native raw proof parameter seam';
        END IF;
        definition := revised;
    END IF;
    -- Claude envelopes require the same successful, source/workflow-bound
    -- selected-template pin as new ChatGPT rows. Upgrade an earlier reviewed
    -- native definition as well as a baseline function; never infer a pin.
    IF strpos(definition, 'selected.result_ref->>''duckdb_template''=''claude_ai_export_json_v1''') = 0 THEN
        revised := replace(definition, 'THEN ''claude_ai_export_json_v1'' ELSE NULL END',
            'AND EXISTS (SELECT 1 FROM context.activity_receipt selected '
            || 'JOIN context.activity_execution selected_execution ON selected_execution.id=selected.activity_execution_id '
            || 'WHERE selected.status=''success'' AND selected_execution.source_version_id=NEW.source_version_id '
            || 'AND selected_execution.workflow_id=source.workflow_id '
            || 'AND selected_execution.activity_name=''select_parser_activity'' '
            || 'AND selected.completed_at<=NEW.created_at '
            || 'AND selected.result_ref->>''parser_id''=NEW.parser_id '
            || 'AND selected.result_ref->>''parser_version''=NEW.parser_version '
            || 'AND selected.result_ref->>''declared_format''=NEW.format_id '
            || 'AND selected.result_ref->>''duckdb_template''=''claude_ai_export_json_v1'') '
            || 'THEN ''claude_ai_export_json_v1'' ELSE NULL END');
        IF revised = definition THEN
            RAISE EXCEPTION 'Missing Claude selected-template pin seam';
        END IF;
        definition := revised;
    END IF;
    IF strpos(definition, $new_status$$2 IN (''chatgpt_json_array_v2'',''claude_ai_export_json_v1'')$new_status$) = 0 THEN
        revised := replace(definition, $old_status$$2=''chatgpt_json_array_v2''$old_status$,
            $new_status$$2 IN (''chatgpt_json_array_v2'',''claude_ai_export_json_v1'')$new_status$);
        IF revised = definition THEN
            RAISE EXCEPTION 'Missing native envelope status seam';
        END IF;
        definition := revised;
    END IF;
    EXECUTE definition;
END;
$migration$;

COMMIT;
