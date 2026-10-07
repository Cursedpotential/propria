#!/usr/bin/env python3
"""Validate frozen native drive observations and load receipt-governed generations into the existing catalog.

Inputs: listing bundle, externally pinned native receipt, explicit operation and new
server artifact directory. Outputs: validation, TSV, SQL and exact catalog readback.
Effects: validate/readback retain server artifacts; load appends drive_objects and
drive_listing_receipts, repairs two current views and registers exactly those four
producer-owned relations after the production release. Registry descriptions come
from canonical implementation docstrings; dates and hashes come from admitted receipts.
Choose after local_drive_listing / drive_inventory_refresh_20261006, following the
reviewed bucket_objects_load.py receipt pattern. No payload processing or second inventory.
Production load requires a pinned parent release identifying a completed lake run.
Disposable test-schema SQL is generated separately; no production DDL is run by tests.
Byline: Codex / GPT-6 / 2026-10-06.
"""
import argparse
import asyncio
import datetime as dt
import hashlib
import inspect
import json
import os
from pathlib import Path
import re
import socket
import subprocess

import local_drive_listing as walker

PG_CONTAINER = 'casebible-pg-oli1nf8wmj5atb9o7oylj4um-131418412167'
BYLINE = 'Codex GPT-6 2026-10-06'


def literal(value):
    """Quote fixed-query SQL text; input string, output literal, no effects; choose catalog metadata. Byline: Codex 2026-10-06."""
    return "'" + value.replace("'", "''") + "'"


def pg_text(value):
    """Encode one PostgreSQL COPY text field; input scalar/null, output escaped text, no effects; choose explicit TSV. Byline: Codex 2026-10-06."""
    return r'\N' if value is None else str(value).replace('\\', '\\\\').replace('\t', r'\t').replace('\n', r'\n').replace('\r', r'\r')


def digest(path):
    """Hash only a retained metadata artifact; input path, output SHA256/bytes, read-only; choose frozen transport validation. Byline: Codex 2026-10-06."""
    result, size = hashlib.sha256(), 0
    with Path(path).open('rb') as source:
        while chunk := source.read(4 * 1024 * 1024):
            result.update(chunk)
            size += len(chunk)
    return result.hexdigest(), size


def schema_name(value):
    """Restrict SQL schema identifiers to catalog or disposable tests; input name, output name, no effects; choose SQL generation. Byline: Codex 2026-10-06."""
    if value != 'raw_duck' and not re.fullmatch('drive_inventory_test_[a-f0-9]{8,32}', value):
        raise ValueError('unapproved catalog/test schema')
    return value


def metadata_path(value, root_error=False):
    """Validate an observed drive-relative identity without resolving source paths; inputs text/root flag, output text, no effects; choose listing parsing. Byline: Codex 2026-10-06."""
    if value == '.' and root_error:
        return value
    if not isinstance(value, str) or not value or any(p in ('', '.', '..') for p in value.split('/')) or '\\' in value or ':' in value or '\x00' in value:
        raise ValueError('unsafe drive-relative identity')
    return value


def prepare(bundle, receipt_sha256, artifacts):
    """Validate frozen metadata and retain explicit load rows and skip evidence.

    Inputs: server bundle, externally reviewed receipt SHA256 and new artifacts path.
    Outputs: validated generation metadata plus objects.tsv and skips.jsonl.
    Effects: server artifact writes only; source payloads and catalog remain unchanged.
    Choose before loading or readback; mismatches fail closed and cannot become current.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    bundle, artifacts = Path(bundle), Path(artifacts)
    artifacts.mkdir(parents=True, exist_ok=False)
    receipt_path = bundle / 'listing-receipt.json'
    raw = receipt_path.read_bytes()
    if not re.fullmatch('[a-f0-9]{64}', receipt_sha256) or hashlib.sha256(raw).hexdigest() != receipt_sha256:
        raise ValueError('native receipt SHA256 mismatch')
    native = json.loads(raw)
    if native.get('schema_version') != 1 or native.get('drive') != 'D:' or native.get('root') != 'D:/' or native.get('platform') != 'nt':
        raise ValueError('native whole-D identity required')
    if native.get('scope') != walker.SCOPE or native.get('policy') != walker.POLICY or native.get('policy_sha256') != walker.POLICY_SHA256:
        raise ValueError('unapproved exclusion/reparse policy')
    if not re.fullmatch('[a-f0-9]{32}', native.get('generation_id', '')):
        raise ValueError('invalid generation ID')
    start = dt.datetime.fromisoformat(native['started_at'])
    end = dt.datetime.fromisoformat(native['ended_at'])
    if start.tzinfo is None or end.tzinfo is None or end < start or native.get('listed_at') != native['started_at']:
        raise ValueError('invalid observation interval')
    if digest(bundle / 'walker-source.py')[0] != native.get('walker_sha256'):
        raise ValueError('walker source identity mismatch')
    temporal = native.get('temporal') or {}
    if not all(temporal.get(k) for k in ('workflow_id', 'run_id', 'activity_id', 'attempt')):
        raise ValueError('missing native Temporal attribution')
    listing = bundle / 'listing.jsonl'
    raw_hash, raw_bytes = digest(listing)
    if (raw_hash, raw_bytes) != (native.get('raw_sha256'), native.get('raw_bytes')):
        raise ValueError('listing SHA256/length mismatch')
    count = dict(file_count=0, file_bytes=0, error_count=0, skip_count=0)
    seen = set()
    with listing.open(encoding='utf-8') as source, (artifacts / 'objects.tsv').open('x', encoding='utf-8', newline='\n') as tsv, (artifacts / 'skips.jsonl').open('x', encoding='utf-8', newline='\n') as skipped:
        for line in source:
            if not line.endswith('\n'):
                raise ValueError('truncated JSONL event')
            row = json.loads(line)
            kind = row.get('kind')
            path = metadata_path(row.get('rel_path'), root_error=kind == 'error')
            if path.casefold() in seen:
                raise ValueError('duplicate drive identity')
            seen.add(path.casefold())
            size, mtime, error = None, None, None
            if kind == 'file':
                size, mtime = row.get('size'), row.get('mtime')
                if type(size) is not int or not 0 <= size <= 9223372036854775807 or type(row.get('mtime_ns')) is not int:
                    raise ValueError('invalid file metadata')
                if mtime != walker.timestamp(row['mtime_ns']):
                    raise ValueError('mtime/nanosecond representation mismatch')
                count['file_count'] += 1
                count['file_bytes'] += size
            elif kind == 'error':
                error = row.get('error')
                if not isinstance(error, str) or not error.startswith(('LISTDIR ', 'STAT ')) or '\x00' in error:
                    raise ValueError('invalid traversal error')
                count['error_count'] += 1
            elif kind == 'skip':
                reason = row.get('reason')
                if reason == 'declared_root_exclusion':
                    if '/' in path or path.casefold() not in {p.casefold() for p in walker.POLICY['root_exclusions']}:
                        raise ValueError('undeclared skipped path')
                elif reason != 'reparse_point' or not (row.get('attributes', 0) & 0x400):
                    raise ValueError('unproven reparse skip')
                count['skip_count'] += 1
                skipped.write(line)
                continue
            else:
                raise ValueError('unknown event kind')
            tsv.write('\t'.join(pg_text(v) for v in (path, size, mtime, error)) + '\n')
    if any(native.get('counts', {}).get(k) != value for k, value in count.items()):
        raise ValueError('native event counts mismatch')
    if native.get('object_count') != count['file_count'] + count['error_count'] or native.get('object_bytes') != count['file_bytes']:
        raise ValueError('native catalog row/byte counts mismatch')
    finished = native.get('enumeration_finished') is True
    status = native.get('status')
    if status in ('complete', 'empty'):
        if not finished or count['error_count'] or native.get('interruption') or (status == 'empty') != (count['file_count'] == 0):
            raise ValueError('invalid completion claim')
        if native['counts'].get('directories_opened', 0) != native['counts'].get('directories_enumerated', -1) or native['counts']['directories_opened'] < 1:
            raise ValueError('directory enumeration incomplete')
    elif status not in ('partial', 'failed'):
        raise ValueError('invalid native receipt status')
    result = dict(drive='D:', listed_at=native['listed_at'], generation_id=native['generation_id'], status=status,
                  scope=walker.SCOPE, object_count=native['object_count'], object_bytes=native['object_bytes'],
                  raw_sha256=raw_hash, raw_bytes=raw_bytes, receipt_sha256=receipt_sha256, native_receipt=native,
                  bundle=str(bundle), artifacts=str(artifacts), tsv_sha256=digest(artifacts / 'objects.tsv')[0],
                  execution_host=socket.gethostname(), loader_sha256=digest(Path(__file__))[0], byline=BYLINE)
    walker.save(artifacts / 'validation.json', result)
    return result


def ddl(schema='raw_duck'):
    """Generate approved receipt-ledger/view repair without executing it; input schema, SQL output, no effects; choose atomic loader bootstrap. Byline: Codex 2026-10-06."""
    s = schema_name(schema)
    return receipt_ddl(s) + current_receipts_ddl(s) + current_objects_ddl(s)


def receipt_ddl(schema):
    """Retain drive listing completion, exclusions and hash references alongside unverified legacy observations.

    Inputs: approved catalog or disposable schema.
    Outputs: SQL for drive_listing_receipts and honest legacy bootstrap.
    Effects: none during generation; execution appends metadata receipts in the existing catalog.
    Choose for coverage evidence; drive_objects holds the individual observations.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    s = schema_name(schema)
    return f"""
CREATE TABLE IF NOT EXISTS {s}.drive_listing_receipts (
 drive text NOT NULL,listed_at timestamptz NOT NULL,generation_id text NOT NULL UNIQUE,
 status text NOT NULL CHECK(status IN ('complete','empty','partial','failed','legacy_unverified')),
 scope text NOT NULL,object_count bigint NOT NULL CHECK(object_count>=0),object_bytes numeric NOT NULL CHECK(object_bytes>=0),
 raw_sha256 text,raw_bytes bigint,receipt jsonb NOT NULL,recorded_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(drive,listed_at));
 COMMENT ON TABLE {s}.drive_listing_receipts IS {literal(summary(receipt_ddl))};
INSERT INTO {s}.drive_listing_receipts(drive,listed_at,generation_id,status,scope,object_count,object_bytes,receipt)
 SELECT o.drive,o.listed_at,'legacy:'||o.drive||':'||o.listed_at::text,'legacy_unverified','whole_drive_metadata',count(*),coalesce(sum(o.size),0),
 jsonb_build_object('status','legacy_unverified','reason','Historical observation has no completion receipt; zero recorded errors does not establish completeness','byline','{BYLINE}')
 FROM {s}.drive_objects o WHERE NOT EXISTS(SELECT 1 FROM {s}.drive_listing_receipts r WHERE r.drive=o.drive AND r.listed_at=o.listed_at)
 GROUP BY o.drive,o.listed_at ON CONFLICT DO NOTHING;
"""


def current_receipts_ddl(schema):
    """Select each drive's newest complete or empty metadata receipt, falling back to explicitly unverified legacy coverage.

    Inputs: approved schema containing drive_listing_receipts.
    Outputs: SQL for drive_inventory_current_receipts, including empty generations.
    Effects: none during generation; execution replaces the producer-owned current receipt view.
    Choose to inspect generation, scope and coverage; partial and failed observations never become current.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    s = schema_name(schema)
    return f"""
CREATE OR REPLACE VIEW {s}.drive_inventory_current_receipts AS
 SELECT DISTINCT ON(drive) * FROM {s}.drive_listing_receipts
 WHERE scope='whole_drive_metadata' AND status IN ('complete','empty','legacy_unverified')
 ORDER BY drive,(status IN ('complete','empty')) DESC,listed_at DESC;
COMMENT ON VIEW {s}.drive_inventory_current_receipts IS {literal(summary(current_receipts_ddl))};
"""


def current_objects_ddl(schema):
    """Read six-column drive metadata through the selected completion receipt, preserving legacy coverage until a verified generation is admitted.

    Inputs: approved schema containing observations and selected receipts.
    Outputs: SQL for the existing drive_objects_current reader.
    Effects: none during generation; execution replaces only the producer-owned current view.
    Choose for current object rows; consult drive_inventory_current_receipts for exclusions and empty generations.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    s = schema_name(schema)
    return f"""
CREATE OR REPLACE VIEW {s}.drive_objects_current AS
 SELECT o.drive,o.rel_path,o.size,o.mtime,o.error,o.listed_at FROM {s}.drive_objects o
 JOIN {s}.drive_inventory_current_receipts r USING(drive,listed_at);
COMMENT ON VIEW {s}.drive_objects_current IS {literal(summary(current_objects_ddl))};
"""


def summary(unit):
    """Extract a canonical first-line description; input documented callable, output summary, no effects; choose generated registry and comments. Byline: Codex 2026-10-06."""
    return inspect.getdoc(unit).splitlines()[0]


def registry_units():
    """Bind the four owned SQL identities to canonical implementation docstrings; no inputs, tuple output, no effects; choose generated producer metadata. Byline: Codex 2026-10-06."""
    return [('drive_objects', 'table', 'listing', walker.scan),
            ('drive_objects_current', 'view', 'listing', current_objects_ddl),
            ('drive_listing_receipts', 'table', 'receipt', receipt_ddl),
            ('drive_inventory_current_receipts', 'view', 'receipt', current_receipts_ddl)]


def registry_sql(receipt, schema='raw_duck'):
    """Register only the four drive producer relations with docstring descriptions and receipt-bound current coverage.

    Inputs: validated admission receipt and approved schema with the existing catalog_registry contract.
    Outputs: guarded SQL generating descriptions, selected observation dates and source/hash evidence.
    Effects: none during generation; execution upserts four owned entries and comments in the load transaction.
    Choose after receipt admission; preserve every non-owned entry and reject unknown classifications or conflicting ownership.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    s = schema_name(schema)
    units = registry_units()
    targets = ','.join(literal(name) for name, _, _, _ in units)
    producer = {'module': Path(__file__).name, 'loader_sha256': receipt['loader_sha256'],
                'walker_sha256': receipt['native_receipt']['walker_sha256'], 'byline': BYLINE}
    admitted = {k: receipt[k] for k in ('drive', 'generation_id', 'listed_at', 'status', 'scope', 'raw_sha256', 'raw_bytes', 'receipt_sha256')}
    metadata = literal(json.dumps({'producer': producer, 'admitted_observation': admitted}, sort_keys=True)) + '::jsonb'
    sql = f"""
SELECT pg_advisory_xact_lock(81020261006);
LOCK TABLE {s}.catalog_registry IN SHARE ROW EXCLUSIVE MODE;
CREATE TEMP TABLE di_registry_before ON COMMIT DROP AS SELECT * FROM {s}.catalog_registry;
DO $guard$ BEGIN
 IF EXISTS(SELECT 1 FROM {s}.catalog_registry WHERE status='unknown') THEN
  RAISE EXCEPTION 'Unknown registry classification requires owner review'; END IF;
 IF EXISTS(SELECT 1 FROM {s}.catalog_registry WHERE object_schema={literal(s)} AND object_name IN ({targets})
  AND (status<>'current' OR coalesce(superseded_by,'')<>'' OR kind IS DISTINCT FROM
   CASE WHEN object_name IN ('drive_listing_receipts','drive_inventory_current_receipts') THEN 'receipt' ELSE 'listing' END
   OR object_type IS DISTINCT FROM CASE WHEN object_name IN ('drive_objects','drive_listing_receipts') THEN 'table' ELSE 'view' END)) THEN
  RAISE EXCEPTION 'Drive registry ownership/classification conflicts with approved producer scope'; END IF;
END $guard$;
CREATE TEMP TABLE di_registry_context ON COMMIT DROP AS
 SELECT CASE WHEN count(*)=0 THEN NULL ELSE
  to_char(min(listed_at) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')||' .. '||
  to_char(max(listed_at) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END AS as_of,
  {metadata} || jsonb_build_object('scope','whole_drive_metadata','observation_range_only',true,
  'current_reader',{literal(s + '.drive_objects_current')},'coverage_reader',{literal(s + '.drive_inventory_current_receipts')},
  'selected_receipts',coalesce(jsonb_agg(jsonb_build_object('drive',drive,'generation_id',generation_id,'listed_at',listed_at,
   'status',status,'scope',scope,'objects',object_count,'bytes',object_bytes,'raw_sha256',raw_sha256,'raw_bytes',raw_bytes,
   'receipt_sha256',receipt->>'receipt_sha256','policy',receipt->'native_receipt'->'policy') ORDER BY drive),'[]'::jsonb)) AS evidence
 FROM {s}.drive_inventory_current_receipts;
"""
    for name, object_type, kind, unit in units:
        description = summary(unit)
        callable_ref = unit.__module__ + '.' + unit.__name__
        evidence = "'[drive-inventory-v1] '||(c.evidence || jsonb_build_object('description_source'," + literal(callable_ref) + '))::text'
        sql += f"""
INSERT INTO {s}.catalog_registry(object_schema,object_name,object_type,kind,covers,as_of,status,superseded_by,made_by,evidence)
 SELECT {literal(s)},{literal(name)},{literal(object_type)},{literal(kind)},{literal(description)},c.as_of,'current','',
 {literal(Path(__file__).name + ' | ' + callable_ref)},{evidence} FROM di_registry_context c
 ON CONFLICT(object_schema,object_name) DO UPDATE SET covers=excluded.covers,as_of=excluded.as_of,
 made_by=excluded.made_by,evidence=concat_ws(E'\n',nullif(regexp_replace({s}.catalog_registry.evidence,
 '(^|\\n)\\[drive-inventory-v1\\] [^\\n]*','','g'),''),excluded.evidence);
COMMENT ON {object_type.upper()} {s}.{name} IS {literal(description)};
"""
    sql += f"""
DO $guard$ BEGIN
 IF EXISTS(SELECT to_jsonb(g) FROM {s}.catalog_registry g WHERE NOT(object_schema={literal(s)} AND object_name IN ({targets}))
  EXCEPT SELECT to_jsonb(b) FROM di_registry_before b WHERE NOT(object_schema={literal(s)} AND object_name IN ({targets})))
 OR EXISTS(SELECT to_jsonb(b) FROM di_registry_before b WHERE NOT(object_schema={literal(s)} AND object_name IN ({targets}))
  EXCEPT SELECT to_jsonb(g) FROM {s}.catalog_registry g WHERE NOT(object_schema={literal(s)} AND object_name IN ({targets})))
 OR EXISTS(SELECT 1 FROM {s}.catalog_registry WHERE status='unknown')
 OR (SELECT count(*) FROM {s}.catalog_registry WHERE object_schema={literal(s)} AND object_name IN ({targets}))<>4 THEN
  RAISE EXCEPTION 'Drive registry scope or classification invariant failed'; END IF;
END $guard$;
"""
    return sql


def registry_equality(receipt, schema):
    """Compare four generated registry descriptions and current receipt bindings; inputs validation/schema, boolean SQL output, no effects; choose independent readback. Byline: Codex 2026-10-06."""
    s = schema_name(schema)
    values = ','.join('(' + ','.join(literal(v) for v in (name, object_type, kind, summary(unit))) + ')'
                      for name, object_type, kind, unit in registry_units())
    return f"""(SELECT count(*)=4 AND bool_and(coalesce(g.status='current' AND coalesce(g.superseded_by,'')='' AND
 g.object_type=u.object_type AND g.kind=u.kind AND g.covers=u.description AND
 (e.proof->'producer'->>'loader_sha256')={literal(receipt['loader_sha256'])} AND
 (e.proof->'admitted_observation'->>'receipt_sha256')={literal(receipt['receipt_sha256'])} AND
 (e.proof->>'current_reader')={literal(s + '.drive_objects_current')} AND
 (SELECT jsonb_agg(jsonb_build_object('drive',x->>'drive','generation_id',x->>'generation_id') ORDER BY x->>'drive')
  FROM jsonb_array_elements(e.proof->'selected_receipts') x) IS NOT DISTINCT FROM
 (SELECT jsonb_agg(jsonb_build_object('drive',drive,'generation_id',generation_id) ORDER BY drive)
  FROM {s}.drive_inventory_current_receipts),false))
 FROM (VALUES {values}) u(name,object_type,kind,description)
 JOIN {s}.catalog_registry g ON g.object_schema={literal(s)} AND g.object_name=u.name
 CROSS JOIN LATERAL (SELECT substring(g.evidence FROM '(?:^|\\n)\\[drive-inventory-v1\\] ([^\\n]*)')::jsonb proof) e)"""


def staging(tsv):
    """Build explicit temporary input staging; input retained TSV path, SQL output, temporary effects when executed; choose load and independent readback. Byline: Codex 2026-10-06."""
    return ('CREATE TEMP TABLE di_in(rel_path text PRIMARY KEY,size bigint,mtime timestamptz,error text) ON COMMIT DROP;\n'
            '\\copy di_in(rel_path,size,mtime,error) FROM STDIN WITH (FORMAT text)\n' + Path(tsv).read_text(encoding='utf-8') + '\\.\n')


def where_generation(receipt):
    """Identify one exact generation; input validated receipt, SQL predicate output, no effects; choose guarded writes/readback. Byline: Codex 2026-10-06."""
    return 'drive=' + literal(receipt['drive']) + ' AND listed_at=' + literal(receipt['listed_at']) + '::timestamptz'


def equality(receipt, schema):
    """Compare every staged/stored metadata column in both directions; inputs receipt/schema, SQL output, no effects; choose exact generation proof. Byline: Codex 2026-10-06."""
    s = schema_name(schema)
    columns = 'rel_path,size,mtime,error'
    return ('NOT EXISTS((SELECT ' + columns + f' FROM {s}.drive_objects WHERE ' + where_generation(receipt) +
            ' EXCEPT SELECT ' + columns + ' FROM di_in) UNION ALL (SELECT ' + columns + ' FROM di_in EXCEPT SELECT ' + columns +
            f' FROM {s}.drive_objects WHERE ' + where_generation(receipt) + '))')


def catalog_payload(receipt):
    """Remove operation-specific derived artifact paths from stable catalog evidence; input validation, output payload, no effects; choose replay/readback identity. Byline: Codex 2026-10-06."""
    return {key: value for key, value in receipt.items() if key != 'artifacts'}


def load_sql(receipt, schema='raw_duck'):
    """Generate one atomic additive generation load and receipt admission; inputs validation/schema, SQL output, no execution; choose after production release. Byline: Codex 2026-10-06."""
    s = schema_name(schema)
    values = [literal(receipt[k]) for k in ('drive', 'listed_at', 'generation_id', 'status', 'scope')]
    values += [str(receipt['object_count']), str(receipt['object_bytes']), literal(receipt['raw_sha256']),
               str(receipt['raw_bytes']), literal(json.dumps(catalog_payload(receipt), ensure_ascii=False)) + '::jsonb']
    predicate = where_generation(receipt)
    return ("\\set ON_ERROR_STOP 1\nBEGIN; SET LOCAL lock_timeout='15s'; SET LOCAL statement_timeout='20min'; "
            "SELECT pg_advisory_xact_lock(81020261007);\n" + ddl(s) + staging(Path(receipt['artifacts']) / 'objects.tsv') +
            f'INSERT INTO {s}.drive_objects(drive,rel_path,size,mtime,error,listed_at) SELECT ' + literal(receipt['drive']) +
            ',rel_path,size,mtime,error,' + literal(receipt['listed_at']) + '::timestamptz FROM di_in ON CONFLICT DO NOTHING;\n' +
            'DO $guard$ BEGIN IF NOT (' + equality(receipt, s) + ") THEN RAISE EXCEPTION 'Drive generation collision or exact row mismatch'; END IF; END $guard$;\n" +
            f'INSERT INTO {s}.drive_listing_receipts(drive,listed_at,generation_id,status,scope,object_count,object_bytes,raw_sha256,raw_bytes,receipt) VALUES (' +
            ','.join(values) + ') ON CONFLICT DO NOTHING;\n' +
            f'DO $guard$ BEGIN IF NOT EXISTS(SELECT 1 FROM {s}.drive_listing_receipts WHERE ' + predicate +
            ' AND receipt=' + literal(json.dumps(catalog_payload(receipt), ensure_ascii=False)) + "::jsonb) THEN RAISE EXCEPTION 'Drive receipt identity or payload collision'; END IF; END $guard$;\n" +
            registry_sql(receipt, s) + 'COMMIT;\n')


def readback_sql(receipt, schema='raw_duck'):
    """Generate an independent transaction checking all rows, receipt and currentness; inputs receipt/schema, SQL output, temporary staging only; choose after commit. Byline: Codex 2026-10-06."""
    s = schema_name(schema)
    return ('\\set ON_ERROR_STOP 1\nBEGIN ISOLATION LEVEL REPEATABLE READ; SET LOCAL statement_timeout=\'20min\';\n' +
            staging(Path(receipt['artifacts']) / 'objects.tsv') +
            "SELECT json_build_object('exact_rows'," + equality(receipt, s) +
            ",'exact_receipt',EXISTS(SELECT 1 FROM " + s + '.drive_listing_receipts WHERE ' + where_generation(receipt) +
            ' AND receipt=' + literal(json.dumps(catalog_payload(receipt), ensure_ascii=False)) + "::jsonb),'current_generation',(SELECT generation_id FROM " +
            s + '.drive_inventory_current_receipts WHERE drive=' + literal(receipt['drive']) +
            "),'current_status',(SELECT status FROM " + s + '.drive_inventory_current_receipts WHERE drive=' + literal(receipt['drive']) +
            "),'exact_registry'," + registry_equality(receipt, s) + ');\nROLLBACK;\n')


def pg(sql, log):
    """Execute server-side catalog SQL with retained output; inputs SQL/log, output JSON rows, declared SQL effects; choose inspected catalog socket. Byline: Codex 2026-10-06."""
    result = subprocess.run(['docker', 'exec', '-i', '-e', 'PGAPPNAME=' + os.environ.get('PGAPPNAME', 'drive-inventory-loader'), PG_CONTAINER, 'psql', '-U', 'postgres', '-d', 'casebible',
                             '-X', '-At', '-v', 'ON_ERROR_STOP=1'], input=sql, text=True, capture_output=True, timeout=1500)
    with Path(log).open('x', encoding='utf-8') as output:
        output.write(result.stdout + result.stderr)
    if result.returncode:
        raise RuntimeError('catalog operation failed; retained log=' + str(log))
    return [json.loads(line) for line in result.stdout.splitlines() if line.startswith('{')]


async def production_release(path, sha256):
    """Require a pinned parent release and completed lake finalization before production mutation.

    Inputs: VPS release JSON path and external SHA256; ready/lake_finalized, lake workflow/run,
    finalized marker path/SHA256 and finalized marker line are required.
    Outputs: validated release metadata.
    Effects: reads release, marker and Temporal; no DDL or catalog writes.
    Choose only for production load; absent release keeps mutation held during publication.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    from temporalio.client import Client, WorkflowExecutionStatus
    if digest(path)[0] != sha256:
        raise ValueError('parent production release SHA256 mismatch')
    release = json.loads(Path(path).read_text())
    if release.get('ready') is not True or release.get('lake_finalized') is not True:
        raise ValueError('production mutation held until lake finalized')
    marker = Path(release['finalized_marker'])
    if digest(marker)[0] != release['finalized_marker_sha256'] or release['finalized_marker_line'] not in marker.read_text().splitlines():
        raise ValueError('lake finalization marker mismatch')
    client = await Client.connect('100.91.190.107:7233')
    handle = client.get_workflow_handle(release['lake_workflow_id'], run_id=release['lake_run_id'])
    if (await handle.describe()).status != WorkflowExecutionStatus.COMPLETED:
        raise ValueError('lake publication run has not completed')
    # The pinned parent release also forbids starting another publication until this load ends.
    if release.get('publication_exclusive_until_load_finishes') is not True:
        raise ValueError('parent publication exclusion required')
    return release


def require_server(path):
    """Confine bulk metadata validation/load to ovh-files artifacts; input path, no output/effects; choose every CLI operation. Byline: Codex 2026-10-06."""
    if os.name != 'posix' or socket.gethostname().split('.')[0] != 'ovh-files' or not Path(path).resolve().is_relative_to(Path('/data/consignatio').resolve()):
        raise ValueError('bulk drive metadata processing requires ovh-files /data/consignatio')


def main():
    """Run explicit VPS validation, released loading or independent readback.

    Inputs: operation, frozen bundle/receipt pin, new artifacts path and load release when applicable.
    Outputs: retained validation/load/readback references and bounded JSON stdout.
    Effects: only load mutates production, after its required lake-finalization gate.
    Choose from the native Temporal wrapper; legacy SQL loading is disabled.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=['validate', 'load', 'readback'])
    parser.add_argument('--bundle', required=True)
    parser.add_argument('--receipt-sha256', required=True)
    parser.add_argument('--artifacts', required=True)
    parser.add_argument('--release')
    parser.add_argument('--release-sha256')
    args = parser.parse_args()
    require_server(args.bundle)
    require_server(args.artifacts)
    if args.operation == 'load':
        if not args.release or not args.release_sha256:
            parser.error('production load requires --release and --release-sha256 after lake finalizes')
        asyncio.run(production_release(args.release, args.release_sha256))
    try:
        receipt = prepare(args.bundle, args.receipt_sha256, args.artifacts)
    except BaseException as error:
        artifacts = Path(args.artifacts)
        if artifacts.is_dir():
            walker.save(artifacts / 'validation-failed.json', dict(error=str(error), error_type=type(error).__name__,
                        receipt_sha256=args.receipt_sha256, bundle=args.bundle, byline=BYLINE))
        raise
    result = dict(generation_id=receipt['generation_id'], status=receipt['status'], validation=str(Path(args.artifacts) / 'validation.json'))
    if args.operation == 'load':
        sql = load_sql(receipt)
        sql_path = Path(args.artifacts) / 'load.sql'
        with sql_path.open('x', encoding='utf-8') as output:
            output.write(sql)
        pg(sql, Path(args.artifacts) / 'load.log')
        result['loaded'] = True
    elif args.operation == 'readback':
        readback = pg(readback_sql(receipt), Path(args.artifacts) / 'readback.log')[0]
        if not readback['exact_rows'] or not readback['exact_receipt'] or not readback['exact_registry']:
            raise ValueError('independent catalog readback mismatch')
        if receipt['status'] in ('partial', 'failed') and readback['current_generation'] == receipt['generation_id']:
            raise ValueError('partial/failed generation incorrectly became current')
        if receipt['status'] in ('complete', 'empty') and readback['current_generation'] != receipt['generation_id']:
            raise ValueError('verified observation is not current; inspect concurrent newer generation')
        result['readback'] = readback
    walker.save(Path(args.artifacts) / (args.operation + '-result.json'), result)
    print(json.dumps(result))


if __name__ == '__main__':
    main()
