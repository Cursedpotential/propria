-- Byline: Codex · GPT-6.1 · 2026-10-05.
-- Install narrowly admitted immutable AI-derived source occurrences in the existing Case Bible catalog.
-- Inputs: DBA-reviewed migration and existing dedicated service role. Outputs: restricted functions.
-- Effects: schema/functions/grants only; no source rows, second catalog, bucket generation or content index.
-- Choose after authenticated physical placement; parent applies admission, never a worker Activity.
BEGIN;
CREATE SCHEMA IF NOT EXISTS source_occurrence_api AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA source_occurrence_api FROM PUBLIC;
-- Verify an existing API namespace remains DBA-owned before installing privileged functions.
-- Inputs: existing schema ACL. Outputs: explicit admission failure. Effects: none; choose fail-closed over adopting another owner's schema.
DO $$ BEGIN
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname='source_occurrence_api'
    AND (n.nspowner<>'postgres'::regrole OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
      WHERE a.grantee<>n.nspowner AND a.privilege_type='CREATE'))) THEN
    RAISE EXCEPTION 'API schema ownership or CREATE admission differs';
  END IF;
END $$;

-- decode_component decodes bounded UTF-8 URI path/query components without losing literal path plus signs.
-- Inputs: encoded component and query flag. Outputs: decoded text. Effects: none; private validation sibling.
CREATE OR REPLACE FUNCTION source_occurrence_api.decode_component(component text, query_component boolean)
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT SET search_path = pg_catalog AS $$
DECLARE bytes bytea := ''::bytea; i integer := 1; c text;
BEGIN
  IF octet_length(component)>16384 THEN RAISE EXCEPTION 'URI component ceiling exceeded' USING ERRCODE='22023'; END IF;
  WHILE i<=char_length(component) LOOP
    c:=substr(component,i,1);
    IF c='%' THEN
      IF substr(component,i+1,2) !~ '^[0-9A-Fa-f]{2}$' THEN RAISE EXCEPTION 'invalid URI escape' USING ERRCODE='22023'; END IF;
      bytes:=bytes||decode(substr(component,i+1,2),'hex'); i:=i+3;
    ELSE
      IF query_component AND c='+' THEN c:=' '; END IF;
      bytes:=bytes||convert_to(c,'UTF8'); i:=i+1;
    END IF;
  END LOOP;
  RETURN convert_from(bytes,'UTF8');
END $$;
ALTER FUNCTION source_occurrence_api.decode_component(text,boolean) OWNER TO postgres;
REVOKE ALL ON FUNCTION source_occurrence_api.decode_component(text,boolean) FROM PUBLIC;

-- register_ai_workproduct inserts or exactly compares one complete pinned original-source batch.
-- Inputs: bounded canonical JSON text and its SHA-256. Outputs: admitted row count.
-- Effects: insert-only existing raw_duck.source_occurrences rows in one transaction; collisions roll back.
-- Choose for fixed Downloads AI-derived Markdown; authenticated root receipts remain the worker's responsibility.
CREATE OR REPLACE FUNCTION source_occurrence_api.register_ai_workproduct(payload text, expected_sha text)
RETURNS integer LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $$
DECLARE b jsonb; request jsonb; rows jsonb; r jsonb; m jsonb; sealed jsonb; existing raw_duck.source_occurrences%ROWTYPE;
        op text; key text; object_ref text; encoded_path text; encoded_version text; n integer; total bigint:=0;
        names text[]:='{}'; root_provenance text; root_manifest text; root_unit text;
BEGIN
  IF payload IS NULL OR octet_length(payload)>262144 OR expected_sha IS NULL OR expected_sha !~ '^[0-9a-f]{64}$'
     OR encode(sha256(convert_to(payload,'UTF8')),'hex')<>expected_sha THEN
    RAISE EXCEPTION 'bounded canonical payload pin required' USING ERRCODE='22023';
  END IF;
  b:=payload::jsonb; request:=b->'request'; rows:=b->'rows'; op:=request->>'operation';
  IF jsonb_typeof(b) IS DISTINCT FROM 'object' OR b->>'schema' IS DISTINCT FROM 'ai-workproduct-source-occurrences/v1'
     OR jsonb_typeof(request) IS DISTINCT FROM 'object' OR jsonb_typeof(rows) IS DISTINCT FROM 'array'
     OR op IS NULL OR op !~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$'
     OR request->>'readback_sha256' IS NULL OR request->>'readback_sha256' !~ '^[0-9a-f]{64}$'
     OR request->>'readback_ref' IS NULL OR request->>'metadata_ref' IS NULL
     OR request->>'readback_ref'=request->>'metadata_ref' THEN
    RAISE EXCEPTION 'invalid batch roots' USING ERRCODE='22023';
  END IF;
  n:=jsonb_array_length(rows);
  IF (SELECT count(*) FROM jsonb_object_keys(b))<>3 OR (SELECT count(*) FROM jsonb_object_keys(request))<>4
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(b) k WHERE k NOT IN ('schema','request','rows'))
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(request) k WHERE k NOT IN ('operation','readback_ref','readback_sha256','metadata_ref'))
    OR request->>'metadata_ref' !~ '^file:///[^?#[:cntrl:]]+$' OR octet_length(request->>'metadata_ref')>4096 THEN
    RAISE EXCEPTION 'unexpected batch request fields' USING ERRCODE='22023';
  END IF;
  IF n<1 OR n>16 THEN RAISE EXCEPTION 'batch count ceiling exceeded' USING ERRCODE='22023'; END IF;
  -- Serialize equal operations, while PK conflict handling serializes competing source identities.
  PERFORM pg_advisory_xact_lock(hashtextextended('ai-workproduct/'||op,0));
  IF EXISTS(SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='local/F-Downloads'
    AND s.scope='F:/Users/matts/Downloads' AND s.metadata->>'operation'=op
    AND (s.metadata->>'catalog_batch_sha256' IS DISTINCT FROM expected_sha)) THEN
    RAISE EXCEPTION 'existing operation pin differs' USING ERRCODE='23505';
  END IF;
  FOR r IN SELECT value FROM jsonb_array_elements(rows) LOOP
    m:=r->'metadata'; key:=r->>'b2_key'; object_ref:=m->>'object_ref';
    IF jsonb_typeof(r)<>'object' OR jsonb_typeof(m)<>'object'
      OR r->>'source' IS DISTINCT FROM 'local/F-Downloads' OR r->>'scope' IS DISTINCT FROM 'F:/Users/matts/Downloads'
      OR r->>'source_id' IS DISTINCT FROM '' OR r->>'disposition' IS DISTINCT FROM 'copied'
      OR r->>'path' IS NULL OR octet_length(r->>'path')>255 OR r->>'path' !~* '^[^/\\[:cntrl:]]+\.md$'
      OR r->>'path' IN ('.','..') OR r->>'path'=ANY(names)
      OR jsonb_typeof(r->'size') IS DISTINCT FROM 'number' OR (r->>'size') !~ '^[0-9]+$'
      OR (r->>'size')::bigint<1 OR (r->>'size')::bigint>1048576
      OR m->>'schema' IS DISTINCT FROM 'ai-workproduct-source-occurrences/v1' OR m->>'operation' IS DISTINCT FROM op
      OR m ? 'catalog_batch_sha256' OR m->>'source_unit' IS NULL
      OR m->>'source_unit' !~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$'
      OR key IS DISTINCT FROM 'consignatio/casevault/KnowledgeBase/ai-chats/_derived/misc/'||(m->>'source_unit')||'/'||(r->>'path')
      OR replace(m->>'original_path',chr(92),'/') IS DISTINCT FROM 'F:/Users/matts/Downloads/'||(r->>'path')
      OR m->>'event_date_status' IS DISTINCT FROM 'unknown'
      OR m->'original_unchanged_size_mtime' IS DISTINCT FROM 'true'::jsonb
      OR jsonb_typeof(m->'observed_mtime_ns') IS DISTINCT FROM 'number' OR m->>'observed_mtime_ns' !~ '^[0-9]+$'
      OR (m->>'observed_mtime_ns')::bigint<0 OR m->>'provider' IS NULL OR octet_length(m->>'provider') NOT BETWEEN 1 AND 64
      OR m->>'transport_observed_at' IS NULL OR m->>'transport_observed_at' !~ '^\d{4}-\d{2}-\d{2}T'
      OR m->>'transport_server_path' IS NULL OR octet_length(m->>'transport_server_path') NOT BETWEEN 1 AND 4096
      OR m->>'readback_ref' IS DISTINCT FROM request->>'readback_ref'
      OR m->>'readback_sha256' IS DISTINCT FROM request->>'readback_sha256'
      OR m->>'version_id' IS NULL OR octet_length(m->>'version_id') NOT BETWEEN 1 AND 2048 OR m->>'version_id'='null'
      OR m->>'version_id' ~ '[[:cntrl:]]' THEN
      RAISE EXCEPTION 'fixed occurrence identity or provenance invalid' USING ERRCODE='22023';
    END IF;
    -- Reject extra writable fields so this API cannot smuggle provider hashes, dates or inventory semantics.
    IF (SELECT count(*) FROM jsonb_object_keys(r))<>8 OR EXISTS(SELECT 1 FROM jsonb_object_keys(r) k WHERE k NOT IN ('source','scope','path','source_id','size','disposition','b2_key','metadata'))
      OR (SELECT count(*) FROM jsonb_object_keys(m))<>20
      OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN ('schema','operation','original_path','observed_mtime_ns',
        'transport_observed_at','original_unchanged_size_mtime','event_date_status','provider','source_unit','source_ref',
        'transport_server_path','provenance_ref','provenance_sha256','placement_manifest_ref','placement_manifest_sha256',
        'readback_ref','readback_sha256','object_ref','version_id','sha256')) THEN RAISE EXCEPTION 'unexpected occurrence fields' USING ERRCODE='22023'; END IF;
    PERFORM (m->>'transport_observed_at')::timestamptz;
    FOREACH root_provenance IN ARRAY ARRAY['source_ref','provenance_ref','placement_manifest_ref','readback_ref'] LOOP
      IF m->>root_provenance IS NULL OR octet_length(m->>root_provenance)>4096
        OR m->>root_provenance !~ '^file:///[^?#[:cntrl:]]+$' THEN RAISE EXCEPTION 'mounted root reference required' USING ERRCODE='22023'; END IF;
    END LOOP;
    FOREACH root_provenance IN ARRAY ARRAY['sha256','provenance_sha256','placement_manifest_sha256','readback_sha256'] LOOP
      IF m->>root_provenance IS NULL OR m->>root_provenance !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'root hash required' USING ERRCODE='22023'; END IF;
    END LOOP;
    IF object_ref IS NULL OR octet_length(object_ref)>16384 OR object_ref !~ '^b2://salem-data/[^?#]+\?versionId=[^&#?=]+$' THEN
      RAISE EXCEPTION 'exact retained object reference required' USING ERRCODE='22023';
    END IF;
    encoded_path:=split_part(substr(object_ref,length('b2://salem-data/')+1),'?',1);
    encoded_version:=split_part(object_ref,'?versionId=',2);
    IF source_occurrence_api.decode_component(encoded_path,false)<>key
      OR source_occurrence_api.decode_component(encoded_version,true)<>m->>'version_id' THEN
      RAISE EXCEPTION 'object key or retained version differs' USING ERRCODE='22023';
    END IF;
    IF root_manifest IS NULL THEN root_manifest:=m->>'placement_manifest_ref'; root_unit:=m->>'source_unit'; END IF;
    IF m->>'placement_manifest_ref'<>root_manifest OR m->>'source_unit'<>root_unit
      OR m->>'placement_manifest_sha256' IS DISTINCT FROM rows->0->'metadata'->>'placement_manifest_sha256'
      OR m->>'provenance_ref' IS DISTINCT FROM rows->0->'metadata'->>'provenance_ref'
      OR m->>'provenance_sha256' IS DISTINCT FROM rows->0->'metadata'->>'provenance_sha256' THEN
      RAISE EXCEPTION 'whole batch root provenance differs' USING ERRCODE='22023';
    END IF;
    names:=array_append(names,r->>'path'); total:=total+(r->>'size')::bigint;
    IF total>16777216 THEN RAISE EXCEPTION 'batch bytes ceiling exceeded' USING ERRCODE='22023'; END IF;
    sealed:=m||jsonb_build_object('catalog_batch_sha256',expected_sha);
    INSERT INTO raw_duck.source_occurrences(source,scope,path,source_id,size,disposition,b2_key,metadata)
      VALUES('local/F-Downloads','F:/Users/matts/Downloads',r->>'path','',(r->>'size')::bigint,'copied',key,sealed)
      ON CONFLICT(source,scope,path,source_id) DO NOTHING;
    SELECT * INTO STRICT existing FROM raw_duck.source_occurrences s WHERE s.source='local/F-Downloads'
      AND s.scope='F:/Users/matts/Downloads' AND s.path=r->>'path' AND s.source_id='';
    IF existing.size<>(r->>'size')::bigint OR existing.disposition<>'copied' OR existing.b2_key IS DISTINCT FROM key
      OR existing.metadata IS DISTINCT FROM sealed OR existing.modtime IS NOT NULL OR existing.native_hash_kind IS NOT NULL
      OR existing.native_hash IS NOT NULL OR existing.md5 IS NOT NULL OR existing.matched_origin IS NOT NULL
      OR existing.integrity_status IS NOT NULL OR existing.integrity_reason IS NOT NULL THEN
      RAISE EXCEPTION 'existing occurrence differs; nothing replaced' USING ERRCODE='23505';
    END IF;
  END LOOP;
  IF (SELECT count(*) FROM raw_duck.source_occurrences s WHERE s.source='local/F-Downloads' AND s.scope='F:/Users/matts/Downloads'
    AND s.metadata->>'operation'=op)<>n THEN RAISE EXCEPTION 'operation row count differs' USING ERRCODE='23505'; END IF;
  RETURN n;
END $$;
ALTER FUNCTION source_occurrence_api.register_ai_workproduct(text,text) OWNER TO postgres;
REVOKE ALL ON FUNCTION source_occurrence_api.register_ai_workproduct(text,text) FROM PUBLIC;

-- read_ai_workproduct returns only one fixed-scope operation identified by its immutable canonical batch pin.
-- Inputs: operation and SHA-256. Outputs: bounded original occurrence rows with complete stored metadata.
-- Effects: read only; choose as independent registration proof without exposing raw_duck or claiming inventory refresh.
CREATE OR REPLACE FUNCTION source_occurrence_api.read_ai_workproduct(operation text, batch_sha text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $$
DECLARE result jsonb; n integer;
BEGIN
  IF operation IS NULL OR operation !~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$'
    OR batch_sha IS NULL OR batch_sha !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'operation pin required' USING ERRCODE='22023'; END IF;
  SELECT count(*) INTO n FROM raw_duck.source_occurrences s WHERE s.source='local/F-Downloads' AND s.scope='F:/Users/matts/Downloads'
    AND s.metadata->>'operation'=operation AND s.metadata->>'catalog_batch_sha256'=batch_sha;
  IF n>16 THEN RAISE EXCEPTION 'readback count ceiling exceeded' USING ERRCODE='22023'; END IF;
  IF EXISTS(SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='local/F-Downloads' AND s.scope='F:/Users/matts/Downloads'
    AND s.metadata->>'operation'=operation AND s.metadata->>'catalog_batch_sha256'=batch_sha AND
    (s.modtime IS NOT NULL OR s.native_hash_kind IS NOT NULL OR s.native_hash IS NOT NULL OR s.md5 IS NOT NULL
    OR s.matched_origin IS NOT NULL OR s.integrity_status IS NOT NULL OR s.integrity_reason IS NOT NULL)) THEN
    RAISE EXCEPTION 'stored auxiliary fields differ' USING ERRCODE='23505';
  END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('source',s.source,'scope',s.scope,'path',s.path,'source_id',s.source_id,
    'size',s.size,'disposition',s.disposition,'b2_key',s.b2_key,'metadata',s.metadata) ORDER BY s.path),'[]'::jsonb) INTO result
    FROM raw_duck.source_occurrences s WHERE s.source='local/F-Downloads' AND s.scope='F:/Users/matts/Downloads'
    AND s.metadata->>'operation'=operation AND s.metadata->>'catalog_batch_sha256'=batch_sha;
  IF octet_length(result::text)>262144 THEN RAISE EXCEPTION 'readback metadata ceiling exceeded' USING ERRCODE='22023'; END IF;
  RETURN result;
END $$;
ALTER FUNCTION source_occurrence_api.read_ai_workproduct(text,text) OWNER TO postgres;
REVOKE ALL ON FUNCTION source_occurrence_api.read_ai_workproduct(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA source_occurrence_api TO casebible_toolkit_recovery_service;
GRANT EXECUTE ON FUNCTION source_occurrence_api.register_ai_workproduct(text,text),source_occurrence_api.read_ai_workproduct(text,text)
  TO casebible_toolkit_recovery_service;
COMMIT;
