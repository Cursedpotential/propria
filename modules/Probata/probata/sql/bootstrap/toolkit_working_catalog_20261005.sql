-- Byline: Codex · GPT-6 · 2026-10-05. Parent-applied Case Bible additive working-library ledger.
-- Inputs: casebible database with existing raw_duck.source_occurrences; no production application from worker startup.
-- Outputs: immutable ledger and two guarded functions; effects: new objects/roles and bounded grants only.
-- Choose for approved permanent working files, never platform rebuild, recovery storage, or inventory generation replacement.
BEGIN;
DO $bootstrap$
BEGIN
 IF current_database() <> 'casebible' THEN RAISE EXCEPTION 'working ledger requires casebible'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='casebible_toolkit_working_owner') THEN
  CREATE ROLE casebible_toolkit_working_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='casebible_toolkit_working_writer') THEN
  CREATE ROLE casebible_toolkit_working_writer NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 END IF;
END $bootstrap$;

-- Existing role names must already satisfy the bounded principal contract; bootstrap never elevates an existing role.
DO $roles$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname IN ('casebible_toolkit_working_owner','casebible_toolkit_working_writer')
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)) THEN
  RAISE EXCEPTION 'working roles violate least privilege';
 END IF;
END $roles$;

CREATE SCHEMA IF NOT EXISTS library_catalog;
REVOKE CREATE ON SCHEMA library_catalog FROM PUBLIC;
GRANT USAGE ON SCHEMA library_catalog TO casebible_toolkit_working_owner,casebible_toolkit_working_writer;

CREATE TABLE IF NOT EXISTS library_catalog.toolkit_working_operation (
 operation_id text PRIMARY KEY CHECK(operation_id ~ '^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$'),
 metadata_sha256 text NOT NULL CHECK(metadata_sha256 ~ '^[0-9a-f]{64}$'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object' AND octet_length(payload::text)<=2097152),
 registered_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE IF NOT EXISTS library_catalog.toolkit_working_object (
 operation_id text NOT NULL REFERENCES library_catalog.toolkit_working_operation(operation_id),
 object_key text NOT NULL,
 version_id text NOT NULL CHECK(length(version_id) BETWEEN 1 AND 1024),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object' AND octet_length(payload::text)<=65536),
 PRIMARY KEY(operation_id,object_key)
);
ALTER TABLE library_catalog.toolkit_working_operation OWNER TO casebible_toolkit_working_owner;
ALTER TABLE library_catalog.toolkit_working_object OWNER TO casebible_toolkit_working_owner;
REVOKE ALL ON library_catalog.toolkit_working_operation,library_catalog.toolkit_working_object FROM PUBLIC,casebible_toolkit_working_writer;
GRANT SELECT ON library_catalog.toolkit_working_operation,library_catalog.toolkit_working_object TO casebible_toolkit_working_writer;
GRANT USAGE ON SCHEMA raw_duck TO casebible_toolkit_working_owner;
GRANT SELECT,INSERT ON raw_duck.source_occurrences TO casebible_toolkit_working_owner;

-- toolkit_b2_ref_key decodes an encoded B2 URI path for exact key comparison, including spaces and Unicode.
-- Inputs: pinned B2 URI; output: decoded key; effects: none; choose instead of comparing an encoded URI with a raw key.
CREATE OR REPLACE FUNCTION library_catalog.toolkit_b2_ref_key(p_ref text)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $decode$
 SELECT convert_from(decode(string_agg(CASE WHEN left(part[1],1)='%' THEN substr(part[1],2)
  ELSE encode(convert_to(part[1],'UTF8'),'hex') END,''),'hex'),'UTF8')
 FROM regexp_matches(split_part(substr(p_ref,length('b2://salem-data/')+1),'?',1),'(%[0-9A-Fa-f]{2}|[^%])','g') part
$decode$;
ALTER FUNCTION library_catalog.toolkit_b2_ref_key(text) OWNER TO casebible_toolkit_working_owner;
REVOKE ALL ON FUNCTION library_catalog.toolkit_b2_ref_key(text) FROM PUBLIC;

-- register_toolkit_working admits one immutable complete operation and conflict-proofs versioned source occurrences.
-- Inputs: canonical UTF-8 metadata text, SHA-256 of those exact bytes; outputs: that SHA after atomic admission/replay.
-- Effects: ledger INSERT and fixed-source occurrence INSERT only, no overwrite or retirement; choose through dedicated writer.
CREATE OR REPLACE FUNCTION library_catalog.register_toolkit_working(p_payload text,p_sha256 text)
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,library_catalog AS $register$
DECLARE
 b jsonb; req jsonb; item jsonb; place jsonb; old library_catalog.toolkit_working_operation%ROWTYPE;
 op text; k text; v text; occurrence_meta jsonb;
BEGIN
 IF octet_length(p_payload)>2097152 OR p_sha256 !~ '^[0-9a-f]{64}$' OR
    encode(sha256(convert_to(p_payload,'UTF8')),'hex') IS DISTINCT FROM p_sha256 THEN
  RAISE EXCEPTION 'working metadata hash/size admission failed';
 END IF;
 b:=p_payload::jsonb; req:=b->'request'; op:=req->>'operation_id';
 IF EXISTS (SELECT 1 FROM jsonb_object_keys(b) f WHERE f NOT IN ('schema','request','objects')) OR
    EXISTS (SELECT 1 FROM jsonb_object_keys(req) f WHERE f NOT IN
     ('operation_id','receipt_ref','receipt_sha256','manifest_ref','manifest_sha256','reference_map_ref','reference_map_sha256','expected_objects')) THEN
  RAISE EXCEPTION 'working header excludes source bodies and unknown metadata';
 END IF;
 IF (b->>'schema'='toolkit-working-catalog/v1' AND
     req->>'receipt_sha256'='244061ffa2d62568938de8b8ebd0792fa924a94c7aee3b5c4d868e3c47cf9b84' AND
     req->>'manifest_sha256'='3f5a219804149b1e3175ffe048887723c2de233789a50860b7cfbf4d781f1ce4' AND
     req->>'reference_map_sha256'='6801dcf9fc4d11d812b000180066f941b167471a1dd7e5a2bbdd3010d71232e3' AND
     req->>'expected_objects'='443' AND op ~ '^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$' AND
     jsonb_typeof(b->'objects')='array') IS NOT TRUE THEN
  RAISE EXCEPTION 'working approved evidence admission failed';
 END IF;
 IF jsonb_array_length(b->'objects')<>443 OR
    (SELECT count(DISTINCT x->'placement'->>'object_key') FROM jsonb_array_elements(b->'objects') x)<>443 THEN
  RAISE EXCEPTION 'working object count/identity admission failed';
 END IF;
 IF (req->>'receipt_ref' LIKE 'file://%' AND length(req->>'receipt_ref')<=4096 AND
     req->>'manifest_ref' LIKE 'file://%' AND length(req->>'manifest_ref')<=4096 AND
     req->>'reference_map_ref' LIKE 'file://%' AND length(req->>'reference_map_ref')<=4096 AND
     req->>'receipt_ref'<>req->>'manifest_ref' AND req->>'receipt_ref'<>req->>'reference_map_ref' AND
     req->>'manifest_ref'<>req->>'reference_map_ref') IS NOT TRUE THEN
  RAISE EXCEPTION 'working metadata locator admission failed';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('family-court-library:'||op,0));
 SELECT * INTO old FROM library_catalog.toolkit_working_operation WHERE operation_id=op;
 IF FOUND THEN
  IF old.metadata_sha256 IS DISTINCT FROM p_sha256 OR old.payload IS DISTINCT FROM b THEN
   RAISE EXCEPTION 'working operation receipt collision';
  END IF;
 ELSE
  INSERT INTO library_catalog.toolkit_working_operation(operation_id,metadata_sha256,payload) VALUES(op,p_sha256,b);
 END IF;
 FOR item IN SELECT value FROM jsonb_array_elements(b->'objects') LOOP
  place:=item->'placement'; k:=place->>'object_key'; v:=place->>'version_id';
  IF (jsonb_typeof(item)='object' AND jsonb_typeof(place)='object' AND
      octet_length(item::text)<=65536 AND length(k)<=4096 AND
      k LIKE 'consignatio/casevault/KnowledgeBase/legal/reference-data/%' AND
      k !~ '(^|/)\.\.?(/|$)' AND strpos(k,chr(92))=0 AND
      length(v) BETWEEN 1 AND 1024 AND v ~ '^[A-Za-z0-9_.-]+$' AND v<>'null' AND place->>'latest_version_id'=v AND
      place->>'object_ref' LIKE 'b2://salem-data/%' AND
      right(place->>'object_ref',length('?versionId='||v))='?versionId='||v AND
      library_catalog.toolkit_b2_ref_key(place->>'object_ref')=k AND
      place->>'status'='verified' AND coalesce(place->>'error','')='' AND
      place->>'sha256' ~ '^[0-9a-f]{64}$' AND (place->>'bytes')::bigint BETWEEN 0 AND 268435456 AND
      jsonb_typeof(place->'after_versions')='array' AND (place->'after_versions') ? v AND
      jsonb_typeof(place->'before_versions')='array' AND
      length(place->>'source_ref') BETWEEN 1 AND 4096 AND length(place->>'unit_id') BETWEEN 1 AND 64 AND
      place->>'path'<>'' AND place->>'path' !~ '(^|/)\.\.?(/|$)' AND
      item->>'archive_sha256' ~ '^[0-9a-f]{64}$' AND (item->>'archive_bytes')::bigint BETWEEN 1 AND 8589934592 AND
      length(item->>'logical_source_path') BETWEEN 1 AND 4096 AND length(item->>'link_policy') BETWEEN 1 AND 8192 AND
      item->>'logical_category' IN ('reference-data','benchbooks','case-law') AND length(item->>'resource_role') BETWEEN 1 AND 256) IS NOT TRUE THEN
   RAISE EXCEPTION 'working pinned object/version admission failed';
  END IF;
  IF EXISTS (SELECT 1 FROM jsonb_object_keys(item) f WHERE f NOT IN
      ('placement','archive_sha256','archive_bytes','logical_source_path','logical_category','resource_role','link_policy','reference_records','source_records')) OR
     EXISTS (SELECT 1 FROM jsonb_object_keys(place) f WHERE f NOT IN
      ('unit_id','source_ref','path','object_key','object_ref','version_id','sha256','bytes','before_versions','after_versions','latest_version_id','status','error')) THEN
   RAISE EXCEPTION 'working catalog excludes source bodies and unknown metadata';
  END IF;
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(coalesce(item->'reference_records','[]'::jsonb)||coalesce(item->'source_records','[]'::jsonb)) link
      WHERE (length(link->>'id') BETWEEN 1 AND 4096) IS NOT TRUE OR EXISTS
        (SELECT 1 FROM jsonb_object_keys(link) f WHERE f NOT IN ('id','record_version','sha256','content_hash','citation','source_path','file_path'))) THEN
   RAISE EXCEPTION 'working record provenance admission failed';
  END IF;
  INSERT INTO library_catalog.toolkit_working_object(operation_id,object_key,version_id,payload)
   VALUES(op,k,v,item) ON CONFLICT(operation_id,object_key) DO NOTHING;
  IF NOT EXISTS (SELECT 1 FROM library_catalog.toolkit_working_object o WHERE o.operation_id=op AND o.object_key=k AND o.version_id=v AND o.payload=item) THEN
   RAISE EXCEPTION 'working ledger version collision';
  END IF;
  occurrence_meta:=jsonb_build_object('schema','toolkit-working-catalog/v1','receipt_sha256',req->>'receipt_sha256',
   'receipt_ref',req->>'receipt_ref','manifest_sha256',req->>'manifest_sha256','manifest_ref',req->>'manifest_ref',
   'reference_map_sha256',req->>'reference_map_sha256','reference_map_ref',req->>'reference_map_ref','object',item);
  -- source_id is the pinned provider version, retaining past states across operations instead of overwriting them.
  INSERT INTO raw_duck.source_occurrences(source,scope,path,source_id,size,native_hash_kind,native_hash,
    disposition,b2_key,matched_origin,metadata,recorded_at,integrity_status,integrity_reason)
   VALUES('family-court-library','b2:salem-data',k,v,(place->>'bytes')::bigint,'sha256',place->>'sha256',
    'verified_working_placement',k,place->>'source_ref',occurrence_meta,clock_timestamp(),
    'transfer_verified','Pinned version/hash read back by placement receipt; substantive integrity not assessed here')
   ON CONFLICT(source,scope,path,source_id) DO NOTHING;
  IF NOT EXISTS (SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='family-court-library' AND
     s.scope='b2:salem-data' AND s.path=k AND s.source_id=v AND s.size=(place->>'bytes')::bigint AND
     s.native_hash_kind='sha256' AND s.native_hash=place->>'sha256' AND s.b2_key=k AND
     s.disposition='verified_working_placement' AND s.matched_origin=place->>'source_ref' AND s.metadata=occurrence_meta AND
     s.integrity_status='transfer_verified' AND s.integrity_reason='Pinned version/hash read back by placement receipt; substantive integrity not assessed here') THEN
   RAISE EXCEPTION 'working source occurrence collision';
  END IF;
 END LOOP;
 RETURN p_sha256;
END $register$;
ALTER FUNCTION library_catalog.register_toolkit_working(text,text) OWNER TO casebible_toolkit_working_owner;
REVOKE ALL ON FUNCTION library_catalog.register_toolkit_working(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION library_catalog.register_toolkit_working(text,text) TO casebible_toolkit_working_writer;

-- read_toolkit_working independently reconstructs all versioned ledger rows and checks their fixed-source occurrence projection.
-- Inputs: one bounded operation; outputs: stored hash and reconstructed payload, or explicit missing/conflict error.
-- Effects: SELECT only; choose in a separate repeatable-read read-only Activity transaction, never use registration as readback.
CREATE OR REPLACE FUNCTION library_catalog.read_toolkit_working(p_operation text)
RETURNS TABLE(metadata_sha256 text,payload jsonb)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,library_catalog AS $readback$
DECLARE head library_catalog.toolkit_working_operation%ROWTYPE; reconstructed jsonb; req jsonb;
BEGIN
 IF (p_operation ~ '^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$') IS NOT TRUE THEN RAISE EXCEPTION 'bounded working operation required'; END IF;
 SELECT * INTO head FROM library_catalog.toolkit_working_operation o WHERE o.operation_id=p_operation;
 IF NOT FOUND THEN RAISE EXCEPTION 'working operation missing'; END IF;
 req:=head.payload->'request';
 SELECT jsonb_agg(o.payload ORDER BY o.object_key) INTO reconstructed FROM library_catalog.toolkit_working_object o WHERE o.operation_id=p_operation;
 IF jsonb_array_length(reconstructed) IS DISTINCT FROM 443 OR EXISTS (
  SELECT 1 FROM library_catalog.toolkit_working_object o WHERE o.operation_id=p_operation AND NOT EXISTS (
   SELECT 1 FROM raw_duck.source_occurrences s WHERE s.source='family-court-library' AND s.scope='b2:salem-data' AND
    s.path=o.object_key AND s.source_id=o.version_id AND s.b2_key=o.object_key AND
    o.version_id=o.payload->'placement'->>'version_id' AND o.object_key=o.payload->'placement'->>'object_key' AND
    s.size=(o.payload->'placement'->>'bytes')::bigint AND s.native_hash_kind='sha256' AND
    s.native_hash=o.payload->'placement'->>'sha256' AND s.disposition='verified_working_placement' AND
    s.matched_origin=o.payload->'placement'->>'source_ref' AND s.integrity_status='transfer_verified' AND
    s.integrity_reason='Pinned version/hash read back by placement receipt; substantive integrity not assessed here' AND
    s.metadata=jsonb_build_object('schema','toolkit-working-catalog/v1','receipt_sha256',req->>'receipt_sha256',
     'receipt_ref',req->>'receipt_ref','manifest_sha256',req->>'manifest_sha256','manifest_ref',req->>'manifest_ref',
     'reference_map_sha256',req->>'reference_map_sha256','reference_map_ref',req->>'reference_map_ref','object',o.payload))) THEN
  RAISE EXCEPTION 'working independent occurrence readback mismatch';
 END IF;
 RETURN QUERY SELECT head.metadata_sha256,jsonb_set(head.payload,'{objects}',reconstructed);
END $readback$;
ALTER FUNCTION library_catalog.read_toolkit_working(text) OWNER TO casebible_toolkit_working_owner;
REVOKE ALL ON FUNCTION library_catalog.read_toolkit_working(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION library_catalog.read_toolkit_working(text) TO casebible_toolkit_working_writer;
COMMENT ON TABLE library_catalog.toolkit_working_operation IS 'Immutable bounded registration metadata; source bodies stay in B2; scoped catalog status is separate from all projections.';
COMMENT ON TABLE library_catalog.toolkit_working_object IS 'Exact permanent B2 object versions and source-unit provenance retained per operation; no bucket listing generation.';
COMMENT ON FUNCTION library_catalog.register_toolkit_working(text,text) IS 'Admit canonical approved working metadata and fixed family-court-library occurrences atomically; exact replay only, conflicts fail. Inputs canonical JSON and SHA256; output SHA256; effects INSERT only; choose after verified placement.';
COMMENT ON FUNCTION library_catalog.read_toolkit_working(text) IS 'Independently read one immutable operation and its pinned ledger/occurrence rows. Input operation ID; outputs hash and reconstructed metadata; effects SELECT only; choose after registration in read-only transaction.';
COMMENT ON FUNCTION library_catalog.toolkit_b2_ref_key(text) IS 'Decode a B2 URI key for metadata admission. Input pinned URI; output UTF8 key; effects none; choose for encoded path equality.';
COMMIT;
