CREATE OR REPLACE FUNCTION context.guard_raw_generation_transition()
 RETURNS trigger
 LANGUAGE plpgsql
 SET search_path TO 'pg_catalog', 'context'
AS $function$
BEGIN
    IF OLD.status <> 'open' OR NEW.status <> 'sealed' THEN
        RAISE EXCEPTION 'raw generation lifecycle only permits open -> sealed';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.source_version_id IS DISTINCT FROM OLD.source_version_id
       OR NEW.generation_ordinal IS DISTINCT FROM OLD.generation_ordinal
       OR NEW.format_id IS DISTINCT FROM OLD.format_id
       OR NEW.parser_id IS DISTINCT FROM OLD.parser_id
       OR NEW.parser_version IS DISTINCT FROM OLD.parser_version
       OR NEW.extraction_bundle_object_id IS DISTINCT FROM OLD.extraction_bundle_object_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'sealed raw generation identity and parser provenance are immutable';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM context.raw_record_identity raw
        WHERE raw.raw_generation_id = NEW.id
    ) THEN
        RAISE EXCEPTION 'raw generation % cannot seal with zero records or envelope spans', NEW.id;
    END IF;
    IF EXISTS (
        SELECT 1
        FROM (
            SELECT record_ordinal,
                   row_number() OVER (ORDER BY record_ordinal) - 1 AS expected_ordinal
            FROM context.raw_record_identity
            WHERE raw_generation_id = NEW.id
        ) ordinals
        WHERE record_ordinal <> expected_ordinal
    ) THEN
        RAISE EXCEPTION 'raw generation % has non-contiguous record ordinals', NEW.id;
    END IF;
    PERFORM context.assert_raw_subtype_completeness(NEW.id);
    IF NOT EXISTS (SELECT 1 FROM context.hash_receipt h
                   WHERE h.hash_kind = 'context_source_fingerprint' AND h.source_version_id = NEW.source_version_id)
       OR EXISTS (SELECT 1 FROM context.raw_record_identity raw
                  WHERE raw.raw_generation_id = NEW.id
                    AND NOT EXISTS (SELECT 1 FROM context.hash_receipt h
                                    WHERE h.hash_kind = 'context_raw_record_fingerprint' AND h.raw_record_id = raw.id))
       OR NOT EXISTS (
            SELECT 1
            FROM context.hash_receipt h
            JOIN context.hash_manifest manifest ON manifest.id = h.hash_manifest_id
            WHERE h.hash_kind = 'context_raw_generation_fingerprint'
              AND h.raw_generation_id = NEW.id
              AND manifest.status = 'sealed'
              AND manifest.member_count = (
                  SELECT count(*) FROM context.raw_record_identity raw
                  WHERE raw.raw_generation_id = NEW.id
              )
       ) THEN
        RAISE EXCEPTION 'raw generation % lacks required context fingerprint receipts', NEW.id;
    END IF;
    IF EXISTS (
        SELECT 1
        FROM (VALUES ('record_accounting'), ('byte_coverage'), ('raw_source_verification')) required(kind)
        WHERE NOT EXISTS (
            SELECT 1 FROM context.reconciliation_receipt r
            WHERE r.raw_generation_id = NEW.id
              AND r.reconciliation_kind = required.kind
              AND r.status = 'success'
        )
    ) THEN
        RAISE EXCEPTION 'raw generation % lacks required successful reconciliation receipts', NEW.id;
    END IF;
    RETURN NEW;
END;
$function$

