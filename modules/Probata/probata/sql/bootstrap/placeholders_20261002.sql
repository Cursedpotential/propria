-- Byline: Claude Code · Sonnet · 2026-10-02
-- Placeholder people for unidentified numbers: grants and the approval guard.
--
-- Owner 2026-10-02 14:32: every unknown number gets a placeholder person (registry.entity + registry.person,
-- role_in_case 'unknown', verification_state 'proposed', requires_human_review true), and every imported
-- row for that number is linked to it. A placeholder is not a confirmed person.
--
-- This one file is additive and idempotent:
--   1. column-level UPDATE grants for platform_runtime (the Proffer starter's login), because the case
--      identity store renames, merges and re-links and platform_runtime had no UPDATE on those columns;
--   2. SELECT on the registry tables for workbench_reader (the Workbench's read-only login), so the mobile
--      view can tell a placeholder from a named person;
--   3. an approval guard: a third-party projection that names a placeholder (verification_state not
--      'confirmed') cannot be approved. This is a separate trigger beside working.validate_message_projection,
--      which only checks that every participant has an entity.
--
-- Dry run (changes nothing):   psql -U ai -d platform -f placeholders_20261002.sql
-- Apply:                       psql -U ai -d platform -v finish=COMMIT -f placeholders_20261002.sql
-- Every check row at the end must read true.

\if :{?finish}
\else
\set finish ROLLBACK
\endif

BEGIN;

-- 1. platform_runtime ---------------------------------------------------------------------------
GRANT UPDATE (display_name, canonical_name, merged_into_id, requires_human_review, review_status)
    ON registry.entity TO platform_runtime;
GRANT UPDATE (short_name, role_in_case, connection_to, relationship_type, notes, is_minor, verification_state)
    ON registry.person TO platform_runtime;
GRANT UPDATE (from_entity_id, to_entity_id) ON working.call_log TO platform_runtime;
GRANT UPDATE (entity_id) ON working.message_participant TO platform_runtime;
GRANT UPDATE (entity_id) ON working.third_party_message_participant TO platform_runtime;
GRANT UPDATE (sender_entity_id) ON working.third_party_message TO platform_runtime;

-- 2. workbench_reader ---------------------------------------------------------------------------
GRANT SELECT ON registry.entity, registry.person, registry.entity_alias TO workbench_reader;

-- 3. approval guard -----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION working.refuse_placeholder_approval()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  record_id uuid;
BEGIN
  IF TG_TABLE_NAME = 'message_projection_route' THEN
    IF NEW.decision_state <> 'approved' OR NEW.projection_kind <> 'acquired_third_party' THEN
      RETURN NULL;
    END IF;
    record_id := NEW.normalized_record_id;
  ELSE
    SELECT tm.normalized_record_id INTO record_id FROM working.third_party_message tm WHERE tm.id = NEW.message_id;
    IF record_id IS NULL OR NOT EXISTS (
      SELECT 1 FROM working.message_projection_route r
      WHERE r.normalized_record_id = record_id AND r.decision_state = 'approved' AND r.projection_kind = 'acquired_third_party'
    ) THEN
      RETURN NULL;
    END IF;
  END IF;
  IF EXISTS (
    SELECT 1
    FROM working.third_party_message tm
    JOIN working.third_party_message_participant p ON p.message_id = tm.id
    JOIN registry.person pp ON pp.id = p.entity_id
    WHERE tm.normalized_record_id = record_id AND pp.verification_state <> 'confirmed'
  ) THEN
    RAISE EXCEPTION 'PLACEHOLDER_PARTICIPANT_NOT_CONFIRMED: a third-party message names a person who is still a placeholder; name or confirm them first';
  END IF;
  RETURN NULL;
END
$$;

DROP TRIGGER IF EXISTS refuse_placeholder_approval_route ON working.message_projection_route;
CREATE CONSTRAINT TRIGGER refuse_placeholder_approval_route
    AFTER INSERT OR UPDATE ON working.message_projection_route
    DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION working.refuse_placeholder_approval();

DROP TRIGGER IF EXISTS refuse_placeholder_approval_participant ON working.third_party_message_participant;
CREATE CONSTRAINT TRIGGER refuse_placeholder_approval_participant
    AFTER INSERT OR UPDATE ON working.third_party_message_participant
    DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION working.refuse_placeholder_approval();

-- Checks: every row must be true -----------------------------------------------------------------
SELECT 'platform_runtime can update entity.display_name' AS check, has_column_privilege('platform_runtime', 'registry.entity', 'display_name', 'UPDATE') AS ok
UNION ALL SELECT 'platform_runtime can update entity.merged_into_id', has_column_privilege('platform_runtime', 'registry.entity', 'merged_into_id', 'UPDATE')
UNION ALL SELECT 'platform_runtime can update person.verification_state', has_column_privilege('platform_runtime', 'registry.person', 'verification_state', 'UPDATE')
UNION ALL SELECT 'platform_runtime can update call_log.from_entity_id', has_column_privilege('platform_runtime', 'working.call_log', 'from_entity_id', 'UPDATE')
UNION ALL SELECT 'platform_runtime can update call_log.to_entity_id', has_column_privilege('platform_runtime', 'working.call_log', 'to_entity_id', 'UPDATE')
UNION ALL SELECT 'platform_runtime can update message_participant.entity_id', has_column_privilege('platform_runtime', 'working.message_participant', 'entity_id', 'UPDATE')
UNION ALL SELECT 'platform_runtime can update third_party_message_participant.entity_id', has_column_privilege('platform_runtime', 'working.third_party_message_participant', 'entity_id', 'UPDATE')
UNION ALL SELECT 'platform_runtime can update third_party_message.sender_entity_id', has_column_privilege('platform_runtime', 'working.third_party_message', 'sender_entity_id', 'UPDATE')
UNION ALL SELECT 'platform_runtime still cannot update call_log.duration_s', NOT has_column_privilege('platform_runtime', 'working.call_log', 'duration_s', 'UPDATE')
UNION ALL SELECT 'workbench_reader can read registry.person', has_table_privilege('workbench_reader', 'registry.person', 'SELECT')
UNION ALL SELECT 'workbench_reader can read registry.entity_alias', has_table_privilege('workbench_reader', 'registry.entity_alias', 'SELECT')
UNION ALL SELECT 'approval guard on routes', EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'refuse_placeholder_approval_route' AND NOT tgisinternal)
UNION ALL SELECT 'approval guard on participants', EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'refuse_placeholder_approval_participant' AND NOT tgisinternal)
UNION ALL SELECT 'no confirmed third-party projection names a placeholder today', NOT EXISTS (
    SELECT 1 FROM working.message_projection_route r
    JOIN working.third_party_message tm ON tm.normalized_record_id = r.normalized_record_id
    JOIN working.third_party_message_participant p ON p.message_id = tm.id
    JOIN registry.person pp ON pp.id = p.entity_id
    WHERE r.decision_state = 'approved' AND r.projection_kind = 'acquired_third_party' AND pp.verification_state <> 'confirmed');

:finish;
