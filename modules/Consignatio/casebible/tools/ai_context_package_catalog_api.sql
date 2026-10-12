-- Register and independently read back an intact native AI package in the existing Case Bible catalog.
-- Inputs: exact bounded metadata JSON text and SHA256; every member already has Go version/hash readback.
-- Outputs: admitted member count or independently verified original metadata JSON.
-- Effects: insert-only existing source_occurrences, atomic_units/members and casevault_placement rows.
-- Choose for native B2 AI packages; the F:/Downloads Markdown API remains unchanged.
-- Byline: Codex GPT-6, 2026-10-08. Existing schema: B2 lake generation refresh-01a1164b-f8f0-7688-9429-3d4895f03b2f.
BEGIN;

-- validate_ai_context_package checks exact payload bytes, complete membership and retained object locators.
-- Inputs: canonical JSON text/hash. Output: validated metadata JSONB. Effects: none.
-- Choose as the private shared validator for the native register/read siblings.
CREATE OR REPLACE FUNCTION source_occurrence_api.validate_ai_context_package(payload text, expected_sha text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE b jsonb; m jsonb; root text; names text[] := '{}'; n integer; total bigint:=0;
BEGIN
  IF payload IS NULL OR octet_length(payload)>262144 OR expected_sha IS NULL
    OR expected_sha !~ '^[0-9a-f]{64}$' OR encode(sha256(convert_to(payload,'UTF8')),'hex')<>expected_sha THEN
    RAISE EXCEPTION 'exact bounded native package payload/hash required' USING ERRCODE='22023';
  END IF;
  b:=payload::jsonb;
  IF jsonb_typeof(b) IS DISTINCT FROM 'object' OR b->>'contract' IS DISTINCT FROM 'ai-context-catalog/v1'
    OR (SELECT count(*) FROM jsonb_object_keys(b))<>13
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(b) k WHERE k NOT IN
      ('contract','operation','source_ref','provider_version_id','source_version_id','source_format','package_ref','original_sha256','manifest_ref','manifest_sha256','receipt_ref','receipt_sha256','members'))
    OR b->>'operation' IS DISTINCT FROM 'context-'||(b->>'manifest_sha256')
    OR b->>'operation' !~ '^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$'
    OR b->>'source_ref' IS NULL OR b->>'source_ref' !~ '^b2://salem-data/[^[:cntrl:]]+$'
    OR octet_length(b->>'source_ref')>16384
    OR b->>'provider_version_id' IS NULL OR octet_length(b->>'provider_version_id') NOT BETWEEN 1 AND 2048
    OR b->>'provider_version_id' ~ '[[:cntrl:]]' OR b->>'provider_version_id'='null'
    OR b->>'source_version_id' IS NULL OR octet_length(b->>'source_version_id') NOT BETWEEN 1 AND 256
    OR b->>'source_version_id' ~ '[[:cntrl:]]'
    OR b->>'source_format' IS NULL OR b->>'source_format' !~ '^[a-z][a-z0-9_]{0,63}$'
    OR b->>'package_ref' IS NULL OR octet_length(b->>'package_ref')>16384 OR b->>'package_ref' ~ '[[:cntrl:]]'
    OR b->>'original_sha256' IS NULL OR b->>'original_sha256' !~ '^[0-9a-f]{64}$'
    OR b->>'manifest_sha256' IS NULL OR b->>'manifest_sha256' !~ '^[0-9a-f]{64}$'
    OR b->>'receipt_sha256' IS NULL OR b->>'receipt_sha256' !~ '^[0-9a-f]{64}$'
    OR b->>'receipt_ref' IS NULL OR b->>'receipt_ref' !~ '^file:///[^?#[:cntrl:]]+$'
    OR octet_length(b->>'receipt_ref')>4096
    OR jsonb_typeof(b->'members') IS DISTINCT FROM 'array' THEN
    RAISE EXCEPTION 'native package root metadata invalid' USING ERRCODE='22023';
  END IF;
  root:='consignatio/casevault/KnowledgeBase/ai-chats/_Incoming/context-'||(b->>'manifest_sha256')||'/';
  n:=jsonb_array_length(b->'members');
  IF n<4 OR n>65 THEN RAISE EXCEPTION 'native package member ceiling exceeded' USING ERRCODE='22023'; END IF;
  FOR m IN SELECT value FROM jsonb_array_elements(b->'members') LOOP
    IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>6
      OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN ('path','source_ref','object_ref','version_id','sha256','bytes'))
      OR m->>'path' IS NULL OR octet_length(m->>'path') NOT BETWEEN 1 AND 2048
      OR m->>'path' ~ '(^/|/$|//|(^|/)\.{1,2}(/|$)|[[:cntrl:]])' OR strpos(m->>'path',chr(92))>0
      OR m->>'path'=ANY(names) OR m->>'source_ref' IS NULL
      OR octet_length(m->>'source_ref')>16384 OR m->>'source_ref' ~ '[[:cntrl:]]'
      OR (m->>'source_ref'='' AND m->>'path'<>'package-manifest.json')
      OR (m->>'source_ref'<>'' AND m->>'source_ref' !~ '^(file:///|b2://salem-data/)')
      OR m->>'object_ref' IS NULL OR octet_length(m->>'object_ref')>16384
      OR m->>'object_ref' !~ '^b2://salem-data/[^?#]+\?versionId=[^&#?=]+$'
      OR m->>'version_id' IS NULL OR octet_length(m->>'version_id') NOT BETWEEN 1 AND 2048
      OR m->>'version_id' ~ '[[:cntrl:]]' OR m->>'version_id'='null'
      OR m->>'sha256' IS NULL OR m->>'sha256' !~ '^[0-9a-f]{64}$'
      OR jsonb_typeof(m->'bytes') IS DISTINCT FROM 'number' OR m->>'bytes' !~ '^[0-9]+$'
      OR (m->>'bytes')::bigint NOT BETWEEN 1 AND 33554432 THEN
      RAISE EXCEPTION 'native package member identity invalid' USING ERRCODE='22023';
    END IF;
    IF source_occurrence_api.decode_component(split_part(substr(m->>'object_ref',length('b2://salem-data/')+1),'?',1),false)
         IS DISTINCT FROM root||(m->>'path')
      OR source_occurrence_api.decode_component(split_part(m->>'object_ref','?versionId=',2),true) IS DISTINCT FROM m->>'version_id' THEN
      RAISE EXCEPTION 'native package key/version URI mismatch' USING ERRCODE='22023';
    END IF;
    IF m->>'path'='package-manifest.json' AND (m->>'sha256' IS DISTINCT FROM b->>'manifest_sha256'
        OR m->>'object_ref' IS DISTINCT FROM b->>'manifest_ref') THEN
      RAISE EXCEPTION 'native manifest pin mismatch' USING ERRCODE='22023';
    END IF;
    IF m->>'path' IN ('original.md','original.json') AND m->>'sha256' IS DISTINCT FROM b->>'original_sha256' THEN
      RAISE EXCEPTION 'native original pin mismatch' USING ERRCODE='22023';
    END IF;
    names:=array_append(names,m->>'path'); total:=total+(m->>'bytes')::bigint;
  END LOOP;
  IF NOT (ARRAY['prepared.json','work_products.json','package-manifest.json'] <@ names)
    OR (('original.md'=ANY(names))::integer+('original.json'=ANY(names))::integer)<>1
    OR total>134479872 THEN RAISE EXCEPTION 'native package incomplete or oversized' USING ERRCODE='22023'; END IF;
  RETURN b;
END $$;
ALTER FUNCTION source_occurrence_api.validate_ai_context_package(text,text) OWNER TO postgres;
REVOKE ALL ON FUNCTION source_occurrence_api.validate_ai_context_package(text,text) FROM PUBLIC;

-- read_ai_context_package independently verifies existing occurrences, complete atomic membership and placements.
-- Inputs: stable operation and original canonical payload hash. Output: exact admitted metadata JSONB.
-- Effects: read only; choose after register rather than interpreting a commit as verification.
CREATE OR REPLACE FUNCTION source_occurrence_api.read_ai_context_package(operation text, payload_sha256 text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE b jsonb; m jsonb; manifest raw_duck.source_occurrences%ROWTYPE;
        r raw_duck.source_occurrences%ROWTYPE; u raw_duck.atomic_units%ROWTYPE;
        p raw_duck.casevault_placement%ROWTYPE; root text; uid bigint; total bigint:=0; n integer; object_key text;
        expected_metadata jsonb; content_manifest text;
BEGIN
  IF operation IS NULL OR operation !~ '^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$'
    OR payload_sha256 IS NULL OR payload_sha256 !~ '^[0-9a-f]{64}$' THEN
    RAISE EXCEPTION 'native package operation/hash required' USING ERRCODE='22023';
  END IF;
  SELECT s.* INTO STRICT manifest FROM raw_duck.source_occurrences s
    WHERE s.source='b2/salem-data' AND s.path='package-manifest.json'
      AND s.metadata->>'schema'='ai-context-catalog/v1' AND s.metadata->>'operation'=operation
      AND s.metadata->>'catalog_payload_sha256'=payload_sha256;
  b:=source_occurrence_api.validate_ai_context_package(manifest.metadata->>'catalog_payload_text',payload_sha256);
  IF b->>'operation' IS DISTINCT FROM operation THEN RAISE EXCEPTION 'stored operation differs' USING ERRCODE='23505'; END IF;
  root:='consignatio/casevault/KnowledgeBase/ai-chats/_Incoming/context-'||(b->>'manifest_sha256')||'/';
  n:=jsonb_array_length(b->'members'); uid:=(manifest.metadata->>'unit_id')::bigint;
  SELECT encode(sha256(convert_to(jsonb_agg(jsonb_build_array(value->>'sha256',(value->>'bytes')::bigint)
    ORDER BY value->>'sha256',(value->>'bytes')::bigint)::text,'UTF8')),'hex') INTO content_manifest FROM jsonb_array_elements(b->'members');
  SELECT a.* INTO STRICT u FROM raw_duck.atomic_units a WHERE a.unit_id=uid;
  IF u.unit_type IS DISTINCT FROM 'ai_context_package' OR u.source IS DISTINCT FROM 'b2/salem-data'
    OR u.export_root IS DISTINCT FROM b->>'source_ref' OR u.service IS DISTINCT FROM 'ai-context'
    OR u.unit_root IS DISTINCT FROM root OR u.member_count IS DISTINCT FROM n::bigint
    OR u.member_manifest IS DISTINCT FROM payload_sha256 OR u.content_manifest IS DISTINCT FROM content_manifest
    OR u.members_without_sha1 IS DISTINCT FROM n::bigint OR u.parent_unit_id IS NOT NULL THEN
    RAISE EXCEPTION 'stored atomic unit differs' USING ERRCODE='23505';
  END IF;
  IF (SELECT count(*) FROM raw_duck.source_occurrences s WHERE s.source='b2/salem-data' AND s.scope=root)<>n
    OR (SELECT count(*) FROM raw_duck.atomic_unit_members a WHERE a.unit_id=uid)<>n
    OR (SELECT count(*) FROM raw_duck.casevault_placement x WHERE x.plan_id='ai-context/'||operation)<>n THEN
    RAISE EXCEPTION 'stored package member count differs' USING ERRCODE='23505';
  END IF;
  FOR m IN SELECT value FROM jsonb_array_elements(b->'members') LOOP
    object_key:=root||(m->>'path'); total:=total+(m->>'bytes')::bigint;
    expected_metadata:=jsonb_build_object('schema','ai-context-catalog/v1','operation',operation,'catalog_payload_sha256',payload_sha256,
      'unit_id',uid,'member',m,'source_ref',b->>'source_ref','provider_version_id',b->>'provider_version_id',
      'source_version_id',b->>'source_version_id','source_format',b->>'source_format','package_ref',b->>'package_ref',
      'original_sha256',b->>'original_sha256','manifest_ref',b->>'manifest_ref','manifest_sha256',b->>'manifest_sha256',
      'receipt_ref',b->>'receipt_ref','receipt_sha256',b->>'receipt_sha256','source_date_status','unknown');
    IF m->>'path'='package-manifest.json' THEN expected_metadata:=expected_metadata||jsonb_build_object('catalog_payload_text',manifest.metadata->>'catalog_payload_text'); END IF;
    SELECT s.* INTO STRICT r FROM raw_duck.source_occurrences s WHERE s.source='b2/salem-data'
      AND s.scope=root AND s.path=m->>'path' AND s.source_id=m->>'version_id';
    IF r.size IS DISTINCT FROM (m->>'bytes')::bigint OR r.native_hash_kind IS DISTINCT FROM 'sha256'
      OR r.native_hash IS DISTINCT FROM m->>'sha256' OR r.disposition IS DISTINCT FROM 'copied'
      OR r.b2_key IS DISTINCT FROM object_key OR r.modtime IS NOT NULL OR r.md5 IS NOT NULL
      OR r.matched_origin IS NOT NULL OR r.integrity_status IS NOT NULL OR r.integrity_reason IS NOT NULL
      OR r.metadata IS DISTINCT FROM expected_metadata THEN
      RAISE EXCEPTION 'stored native occurrence differs' USING ERRCODE='23505';
    END IF;
    IF (SELECT count(*) FROM raw_duck.atomic_unit_members a WHERE a.unit_id=uid AND a.key=object_key)<>1 THEN
      RAISE EXCEPTION 'stored atomic membership differs' USING ERRCODE='23505';
    END IF;
    SELECT x.* INTO STRICT p FROM raw_duck.casevault_placement x WHERE x.plan_id='ai-context/'||operation AND x.dst_key=object_key;
    IF p.src_key IS DISTINCT FROM coalesce(nullif(m->>'source_ref',''),b->>'source_ref')
      OR p.src_size IS DISTINCT FROM (m->>'bytes')::bigint OR p.dst_size IS DISTINCT FROM (m->>'bytes')::bigint
      OR p.src_sha1 IS NOT NULL OR p.dst_sha1 IS NOT NULL OR p.status IS DISTINCT FROM 'verified'
      OR p.method IS DISTINCT FROM 'go_context_package' OR p.proof_level IS DISTINCT FROM 'sha256_exact_version_readback'
      OR p.placed_at IS NULL OR p.verified_at IS NULL THEN
      RAISE EXCEPTION 'stored native placement differs' USING ERRCODE='23505';
    END IF;
  END LOOP;
  IF u.total_bytes IS DISTINCT FROM total::numeric THEN RAISE EXCEPTION 'stored native byte total differs' USING ERRCODE='23505'; END IF;
  RETURN b;
END $$;
ALTER FUNCTION source_occurrence_api.read_ai_context_package(text,text) OWNER TO postgres;
REVOKE ALL ON FUNCTION source_occurrence_api.read_ai_context_package(text,text) FROM PUBLIC;

-- register_ai_context_package inserts one complete verified native package or admits an identical replay.
-- Inputs: exact metadata JSON text/hash. Output: member count. Effects: atomic insert-only existing tables.
-- Choose for Go's pinned native package receipt; conflicting operations or members roll back together.
CREATE OR REPLACE FUNCTION source_occurrence_api.register_ai_context_package(payload text, payload_sha256 text)
RETURNS integer LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE b jsonb; m jsonb; root text; uid bigint; op text; n integer; total bigint;
        object_key text; metadata jsonb; stamp timestamptz:=clock_timestamp(); content_manifest text;
BEGIN
  b:=source_occurrence_api.validate_ai_context_package(payload,payload_sha256);
  op:=b->>'operation'; n:=jsonb_array_length(b->'members');
  root:='consignatio/casevault/KnowledgeBase/ai-chats/_Incoming/context-'||(b->>'manifest_sha256')||'/';
  PERFORM pg_advisory_xact_lock(hashtextextended('ai-context-package/'||op,0));
  IF EXISTS(SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='b2/salem-data'
    AND s.metadata->>'schema'='ai-context-catalog/v1' AND s.metadata->>'operation'=op) THEN
    IF EXISTS(SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='b2/salem-data'
      AND s.metadata->>'schema'='ai-context-catalog/v1' AND s.metadata->>'operation'=op
      AND s.metadata->>'catalog_payload_sha256' IS DISTINCT FROM payload_sha256) THEN
      RAISE EXCEPTION 'native operation already has different immutable metadata' USING ERRCODE='23505';
    END IF;
    IF source_occurrence_api.read_ai_context_package(op,payload_sha256) IS DISTINCT FROM b THEN
      RAISE EXCEPTION 'native replay differs' USING ERRCODE='23505';
    END IF;
    RETURN n;
  END IF;
  LOCK TABLE raw_duck.atomic_units IN SHARE ROW EXCLUSIVE MODE;
  IF EXISTS(SELECT 1 FROM raw_duck.atomic_units a WHERE a.source='b2/salem-data' AND a.unit_root=root)
    OR EXISTS(SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='b2/salem-data' AND s.scope=root) THEN
    RAISE EXCEPTION 'native unit root already exists outside this operation' USING ERRCODE='23505';
  END IF;
  SELECT coalesce(max(a.unit_id),0)+1 INTO uid FROM raw_duck.atomic_units a;
  SELECT sum((value->>'bytes')::bigint) INTO total FROM jsonb_array_elements(b->'members');
  SELECT encode(sha256(convert_to(jsonb_agg(jsonb_build_array(value->>'sha256',(value->>'bytes')::bigint)
    ORDER BY value->>'sha256',(value->>'bytes')::bigint)::text,'UTF8')),'hex') INTO content_manifest FROM jsonb_array_elements(b->'members');
  INSERT INTO raw_duck.atomic_units(unit_id,unit_type,source,export_root,service,unit_root,member_count,total_bytes,member_manifest,content_manifest,members_without_sha1)
    VALUES(uid,'ai_context_package','b2/salem-data',b->>'source_ref','ai-context',root,n,total,payload_sha256,content_manifest,n);
  FOR m IN SELECT value FROM jsonb_array_elements(b->'members') LOOP
    object_key:=root||(m->>'path');
    metadata:=jsonb_build_object('schema','ai-context-catalog/v1','operation',op,'catalog_payload_sha256',payload_sha256,
      'unit_id',uid,'member',m,'source_ref',b->>'source_ref','provider_version_id',b->>'provider_version_id',
      'source_version_id',b->>'source_version_id','source_format',b->>'source_format','package_ref',b->>'package_ref',
      'original_sha256',b->>'original_sha256','manifest_ref',b->>'manifest_ref','manifest_sha256',b->>'manifest_sha256',
      'receipt_ref',b->>'receipt_ref','receipt_sha256',b->>'receipt_sha256','source_date_status','unknown');
    IF m->>'path'='package-manifest.json' THEN metadata:=metadata||jsonb_build_object('catalog_payload_text',payload); END IF;
    INSERT INTO raw_duck.source_occurrences(source,scope,path,source_id,size,native_hash_kind,native_hash,disposition,b2_key,metadata)
      VALUES('b2/salem-data',root,m->>'path',m->>'version_id',(m->>'bytes')::bigint,'sha256',m->>'sha256','copied',object_key,metadata);
    INSERT INTO raw_duck.atomic_unit_members(unit_id,key) VALUES(uid,object_key);
    INSERT INTO raw_duck.casevault_placement(plan_id,src_key,dst_key,src_size,dst_size,status,method,placed_at,proof_level,verified_at)
      VALUES('ai-context/'||op,coalesce(nullif(m->>'source_ref',''),b->>'source_ref'),object_key,(m->>'bytes')::bigint,(m->>'bytes')::bigint,
        'verified','go_context_package',stamp,'sha256_exact_version_readback',stamp);
  END LOOP;
  PERFORM source_occurrence_api.read_ai_context_package(op,payload_sha256);
  RETURN n;
END $$;
ALTER FUNCTION source_occurrence_api.register_ai_context_package(text,text) OWNER TO postgres;
REVOKE ALL ON FUNCTION source_occurrence_api.register_ai_context_package(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA source_occurrence_api TO casebible_toolkit_recovery_service;
GRANT EXECUTE ON FUNCTION source_occurrence_api.register_ai_context_package(text,text),source_occurrence_api.read_ai_context_package(text,text)
  TO casebible_toolkit_recovery_service;
COMMIT;
