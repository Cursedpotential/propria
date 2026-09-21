-- Byline: Claude Code · Fable 5.1 · 2026-09-20 — applied live; same text as sql/bootstrap/schema_snapshot_20260907.sql
CREATE OR REPLACE FUNCTION context.register_raw_format_subtype(p_format_id text)
 RETURNS regclass
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'context'
AS $function$
DECLARE
    v_table_name TEXT;
    v_relation REGCLASS;
    v_relation_kind "char";
    v_raw_record_attnum SMALLINT;
    v_raw_identity_id_attnum SMALLINT;
    v_native_fields_attnum SMALLINT;
    v_native_metadata_attnum SMALLINT;
BEGIN
    IF p_format_id !~ '^[a-z][a-z0-9_]{0,58}$' THEN
        RAISE EXCEPTION 'invalid raw format id %', p_format_id;
    END IF;

    v_table_name := 'raw_' || p_format_id;
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS context.%I (
            raw_record_id UUID PRIMARY KEY
                REFERENCES context.raw_record_identity(id) ON DELETE RESTRICT,
            native_fields JSONB NOT NULL DEFAULT ''{}''::jsonb
                CHECK (jsonb_typeof(native_fields) = ''object''),
            native_metadata JSONB NOT NULL DEFAULT ''{}''::jsonb
                CHECK (jsonb_typeof(native_metadata) = ''object''),
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )',
        v_table_name
    );
    v_relation := to_regclass(format('context.%I', v_table_name));

    SELECT relkind INTO v_relation_kind
    FROM pg_class
    WHERE oid = v_relation;
    IF v_relation_kind IS DISTINCT FROM 'r'::"char" THEN
        RAISE EXCEPTION 'raw subtype % must be an ordinary table, found relation kind %',
            v_relation::TEXT, v_relation_kind;
    END IF;

    SELECT attnum INTO v_raw_identity_id_attnum
    FROM pg_attribute
    WHERE attrelid = 'context.raw_record_identity'::regclass
      AND attname = 'id'
      AND atttypid = 'uuid'::regtype
      AND attnotnull
      AND NOT attisdropped;

    SELECT attnum INTO v_raw_record_attnum
    FROM pg_attribute
    WHERE attrelid = v_relation
      AND attname = 'raw_record_id'
      AND atttypid = 'uuid'::regtype
      AND attnotnull
      AND NOT attisdropped;

    IF v_raw_record_attnum IS NULL OR v_raw_identity_id_attnum IS NULL THEN
        RAISE EXCEPTION 'raw subtype % must have a NOT NULL UUID raw_record_id key column',
            v_relation::TEXT;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = v_relation
          AND contype = 'p'
          AND conkey = ARRAY[v_raw_record_attnum]::SMALLINT[]
    ) THEN
        RAISE EXCEPTION 'raw subtype % must have raw_record_id as its exact primary key',
            v_relation::TEXT;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = v_relation
          AND contype = 'f'
          AND confrelid = 'context.raw_record_identity'::regclass
          AND conkey = ARRAY[v_raw_record_attnum]::SMALLINT[]
          AND confkey = ARRAY[v_raw_identity_id_attnum]::SMALLINT[]
          AND confdeltype = 'r'
    ) THEN
        RAISE EXCEPTION 'raw subtype % must have an exact raw_record_id FK to context.raw_record_identity(id) ON DELETE RESTRICT',
            v_relation::TEXT;
    END IF;

    SELECT attnum INTO v_native_fields_attnum
    FROM pg_attribute
    WHERE attrelid = v_relation
      AND attname = 'native_fields'
      AND atttypid = 'jsonb'::regtype
      AND attnotnull
      AND NOT attisdropped;
    SELECT attnum INTO v_native_metadata_attnum
    FROM pg_attribute
    WHERE attrelid = v_relation
      AND attname = 'native_metadata'
      AND atttypid = 'jsonb'::regtype
      AND attnotnull
      AND NOT attisdropped;
    IF v_native_fields_attnum IS NULL OR v_native_metadata_attnum IS NULL THEN
        RAISE EXCEPTION 'raw subtype % must have NOT NULL JSONB native_fields and native_metadata columns',
            v_relation::TEXT;
    END IF;

    INSERT INTO context.raw_format_registry (format_id, subtype_relation)
    VALUES (p_format_id, v_relation)
    ON CONFLICT (format_id) DO NOTHING;
    IF (SELECT subtype_relation FROM context.raw_format_registry WHERE format_id = p_format_id)
       IS DISTINCT FROM v_relation THEN
        RAISE EXCEPTION 'raw format % is already registered to a different subtype relation', p_format_id;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgrelid = v_relation
          AND tgname = 'raw_subtype_append_only'
          AND NOT tgisinternal
    ) THEN
        EXECUTE format(
            'CREATE TRIGGER raw_subtype_append_only
             BEFORE UPDATE OR DELETE ON context.%I
             FOR EACH ROW EXECUTE FUNCTION context.forbid_mutation()',
            v_table_name
        );
    END IF;
    -- raw_subtype_open_generation_gate removed 2026-09-20: its guard function is not part of the D-152 database.
    -- Grants added 2026-09-20: SECURITY DEFINER creates the table as its owner; the engine role must be able to insert.
    EXECUTE format('GRANT SELECT, INSERT ON TABLE context.%I TO context_import_writer', v_table_name);
    EXECUTE format('GRANT SELECT ON TABLE context.%I TO context_reader', v_table_name);
    EXECUTE format('GRANT ALL ON TABLE context.%I TO platform_app', v_table_name);
    RETURN v_relation;
END;
$function$;
