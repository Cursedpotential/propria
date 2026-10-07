#!/usr/bin/env python3
"""Verify a completed frozen D preservation upload on ovh-files through independent Temporal Activities.

Inputs: manifest SHA256/count/bytes, final upload receipt SHA256 and retained upload
artifact bundle, successful upload workflow/run, and a new /data/consignatio job.
Outputs: retained listings, cache, full SHA1/SHA256/MD5 receipts and exact catalog readback.
Effects: downloads only on ovh-files; appends to the existing preservation ledger.
Choose after d_preservation_transfer_20261006 completes, never to run or resume transport.
Uses the reviewed r2_preservation_reconcile_20261006 verify_copies/version/record pattern.
Destination hashes prove version-bound destination bytes. Historical catalog matches
are separately attributed; fresh D source hashes, format integrity and package
completeness remain unassessed. Size-only transport skips are not source-byte proof.
No source writes, new catalog tables, cache deletion, lake publication or index refresh.
CLI: temporal --manifest PATH --manifest-sha256 HEX --upload-receipt PATH
--upload-sha256 HEX --upload-workflow ID --upload-run ID --work PATH --workflow-id ID
[--expected-count N --expected-bytes N]. Copy the receipt's before/after/API/log
siblings beside the supplied receipt on the VPS. Every batch needs its own receipt.
Byline: Codex / GPT-6 / 2026-10-06.
"""
import argparse
import asyncio
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path, PureWindowsPath
import re
import signal
import socket
import stat
import subprocess
import sys
from datetime import datetime, timedelta, timezone

BYLINE = 'Codex GPT-6 2026-10-06'
PREFIX = 'consignatio/casevault/Recovered/_Incoming/d-preservation-20261006/'
REMOTE = 'b2native-full:salem-data/' + PREFIX
MANIFEST_SHA256 = '3e2560c91e8d51cc205cf19f89bec552aa26f0a8506153785adb181942803e42'
PG_CONTAINER = 'casebible-pg-oli1nf8wmj5atb9o7oylj4um-131418412167'
QUEUE = 'd-preservation-verify-20261006'
STAGES = ('validate', 'historical_sources', 'listing_before', 'download', 'hash_cache', 'listing_after', 'record')
LIMITS = ('Version-bound destination bytes; no fresh D source hash or consistent source snapshot; '
          'equal source metadata and size-only skips cannot prove source bytes; '
          'format integrity, readable content and complete packages not assessed.')


def save(path, value):
    """Retain JSON without replacing different bytes; inputs path/value, output digest, effects file creation; choose receipts. Byline: Codex 2026-10-06."""
    raw = json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2).encode('utf-8')
    try:
        with Path(path).open('xb') as output:
            output.write(raw)
    except FileExistsError:
        if Path(path).read_bytes() != raw:
            raise ValueError('retained artifact conflict: ' + str(path))
    return hashlib.sha256(raw).hexdigest()


def read_json(path):
    """Read retained UTF8 JSON; input path, output value, no effects; choose artifact loading. Byline: Codex 2026-10-06."""
    return json.loads(Path(path).read_text(encoding='utf-8-sig'))


def pinned(path, digest):
    """Validate artifact bytes against an external SHA256; inputs path/digest, output bytes, no effects; choose preflight. Byline: Codex 2026-10-06."""
    if not re.fullmatch('[0-9a-f]{64}', digest):
        raise ValueError('invalid SHA256 pin')
    raw = Path(path).read_bytes()
    if hashlib.sha256(raw).hexdigest() != digest:
        raise ValueError('artifact SHA256 mismatch: ' + str(path))
    return raw


def safe_relative(value):
    """Reject unsafe or line-breaking Backup paths; input string, output path, no effects; choose manifest/cache confinement. Byline: Codex 2026-10-06."""
    if not isinstance(value, str):
        raise ValueError('relative path must be text')
    parts = value.split('/')
    if len(parts) < 2 or parts[0] != 'Backup' or any(
            not p or p in ('.', '..') or p.rstrip(' .') != p or
            any(c in p for c in '\\:*?"<>|') or any(ord(c) < 32 for c in p) or
            re.fullmatch(r'(?i)(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(?:\..*)?', p) for p in parts):
        raise ValueError('unsafe Backup relative path')
    return value


def load_manifest(config):
    """Validate exact frozen identities/count/bytes; input config, output normalized rows, no source reads; choose every stage. Byline: Codex 2026-10-06."""
    path = Path(config['manifest'])
    if path.stat().st_size > 128 * 1024 * 1024:
        raise ValueError('manifest exceeds 128 MiB')
    text = pinned(path, config['manifest_sha256']).decode('utf-8-sig')
    if path.suffix.lower() == '.jsonl':
        rows = [json.loads(line) for line in text.splitlines() if line.strip()]
    else:
        rows = json.loads(text)
        rows = rows['files'] if isinstance(rows, dict) else rows
    if not isinstance(rows, list) or not 0 < len(rows) <= 150000:
        raise ValueError('manifest must contain 1..150000 rows')
    normalized, seen = [], set()
    for row in rows:
        relative = safe_relative(row.get('rel_path', row.get('relative_path')))
        size = row.get('size_bytes', row.get('size'))
        if type(size) is not int or size <= 0 or relative.casefold() in seen:
            raise ValueError('duplicate identity or invalid size')
        seen.add(relative.casefold())
        if 'rel_path' in row and row.get('source_root') != 'D:/':
            raise ValueError('unexpected source root')
        if row.get('copy_target_key', PREFIX + relative) != PREFIX + relative or row.get(
                'copy_target_root', 'b2:salem-data/' + PREFIX).rstrip('/') != ('b2:salem-data/' + PREFIX).rstrip('/'):
            raise ValueError('destination contradicts frozen contract')
        if any(row.get(k) for k in ('error', 'errors', 'excluded', 'quarantine')):
            raise ValueError('blocked manifest row')
        if row.get('status') not in (None, 'source_hash_missing', 'source_hash_without_sha1_bridge', 'source_hash_not_on_current_b2'):
            raise ValueError('unapproved manifest status')
        if 'source_mtime' in row:
            stamp = row['source_mtime']
            instant = datetime.fromisoformat(stamp.replace('Z', '+00:00'))
            if instant.tzinfo is None:
                raise ValueError('manifest timestamp requires timezone')
            fraction = stamp.split('.')[1].split('+')[0].split('-')[0].rstrip('Z') if '.' in stamp else ''
            if len(fraction) > 6:
                raise ValueError('unsupported timestamp precision')
            delta = instant.astimezone(timezone.utc) - datetime(1970, 1, 1, tzinfo=timezone.utc)
            mtime_ns = (delta.days * 86400 + delta.seconds) * 1000000000 + delta.microseconds * 1000
            precision = 10 ** (9 - len(fraction))
        else:
            mtime_ns, precision = row.get('mtime_ns'), 1
        if type(mtime_ns) is not int:
            raise ValueError('missing manifest timestamp')
        normalized.append(dict(relative_path=relative, size=size, mtime_ns=mtime_ns, mtime_precision_ns=precision))
    if len(rows) != config['expected_count'] or sum(r['size'] for r in normalized) != config['expected_bytes']:
        raise ValueError('frozen manifest count/bytes mismatch')
    return normalized


def bundled(receipt_path, original):
    """Resolve a Windows receipt sibling in its server bundle; inputs receipt/ref, output safe path, no effects; choose transported artifacts. Byline: Codex 2026-10-06."""
    name = PureWindowsPath(original).name
    base = Path(receipt_path).resolve().parent
    path = base / name
    if not name or path.is_symlink() or not path.resolve(strict=True).is_relative_to(base):
        raise ValueError('unsafe bundled reference')
    return path


def validate_listing(objects, rows):
    """Require exact unique B2 keys, sizes and version IDs; inputs listing/rows, output indexed objects, no effects; choose both snapshots. Byline: Codex 2026-10-06."""
    expected = {r['relative_path']: r['size'] for r in rows}
    actual = {}
    for item in objects:
        path = safe_relative(item['Path'])
        if path in actual or item.get('IsDir') is not False or type(item.get('Size')) is not int or not isinstance(item.get('ID'), str) or not item['ID']:
            raise ValueError('duplicate, non-file or versionless destination')
        actual[path] = item
    if {k: v['Size'] for k, v in actual.items()} != expected:
        raise ValueError('destination missing/extra key or size mismatch')
    return actual


def validate_upload(config, rows):
    """Reject incomplete transport or unaccounted identities; inputs config/rows, output pinned provenance, no D reads; choose before remote listing. Byline: Codex 2026-10-06."""
    receipt_path = Path(config['upload_receipt'])
    receipt = json.loads(pinned(receipt_path, config['upload_sha256']).decode('utf-8-sig'))
    required = dict(stage='upload', status='completed', manifest_sha256=config['manifest_sha256'],
                    count=len(rows), bytes=sum(r['size'] for r in rows), metadata_unchanged=True,
                    rclone_exit_code=0, api_exit_code=0, api_missing_count=0, api_size_mismatch_count=0,
                    destination='b2:salem-data/' + PREFIX)
    if any(receipt.get(k) != v for k, v in required.items()) or receipt.get('error') or receipt.get('source_changes'):
        raise ValueError('final successful upload receipt required')
    temporal = receipt.get('temporal', {})
    if temporal.get('workflow_id') != config['upload_workflow'] or temporal.get('run_id') != config['upload_run']:
        raise ValueError('upload workflow/run mismatch')
    refs = {k: bundled(receipt_path, receipt[k]) for k in ('before', 'after', 'api_observation', 'rclone_json_log')}
    before, after = read_json(refs['before']), read_json(refs['after'])
    if before != after or len(before) != len(rows):
        raise ValueError('source metadata changed or missing')
    observed = {r['relative_path']: r for r in before}
    if len(observed) != len(before) or set(observed) != {r['relative_path'] for r in rows}:
        raise ValueError('source metadata identities differ')
    for row in rows:
        item = observed[row['relative_path']]
        delta = item['mtime_ns'] - row['mtime_ns']
        precision = row['mtime_precision_ns']
        if item.get('error') or item.get('matches_manifest') is not True or item['size'] != row['size'] or (
                delta != 0 if precision == 1 else abs(delta) > precision):
            raise ValueError('source metadata does not match frozen manifest')
    validate_listing(read_json(refs['api_observation']), rows)
    copied, skipped = set(), set()
    expected = set(observed)
    with refs['rclone_json_log'].open(encoding='utf-8-sig') as log:
        for line in log:
            event = json.loads(line)
            if event.get('level', '').lower() in ('error', 'critical', 'fatal'):
                raise ValueError('upload log error')
            path, message = event.get('object'), event.get('msg', '')
            if message.startswith('Copied (') or message == 'Unchanged skipping':
                if path not in expected:
                    raise ValueError('unreviewed upload log identity')
                (copied if message.startswith('Copied (') else skipped).add(path)
    counts = receipt.get('rclone_observed_counts', {})
    if copied & skipped or copied | skipped != expected or any(counts.get(k) != v for k, v in {
            'copied_log_count': len(copied), 'existing_unchanged_log_count': len(skipped),
            'unclassified_manifest_count': 0, 'error_log_count': 0, 'malformed_json_lines': 0}.items()):
        raise ValueError('upload per-object accounting mismatch')
    return dict(upload_receipt_sha256=config['upload_sha256'], manifest_sha256=config['manifest_sha256'],
                artifact_sha256={k: hashlib.sha256(p.read_bytes()).hexdigest() for k, p in refs.items()},
                copied_log_count=len(copied), size_only_skip_count=len(skipped), limits=LIMITS)


def compare_versions(before, after):
    """Reject version or provider-hash changes across download; inputs indexed snapshots, output none, no effects; choose after hashing. Byline: Codex 2026-10-06."""
    fields = ('ID', 'Size', 'Hashes', 'Metadata', 'ModTime')
    if set(before) != set(after) or any(any(before[k].get(f) != after[k].get(f) for f in fields) for k in before):
        raise ValueError('destination version or metadata changed during readback')


def hash_file(root, row):
    """Read every retained destination byte with three digests; inputs cache/row, output hashes, no source effects; choose server verification. Byline: Codex 2026-10-06."""
    relative = safe_relative(row['relative_path'])
    root = Path(root).resolve(strict=True)
    path = root
    for part in relative.split('/'):
        path = path / part
        if path.is_symlink():
            raise ValueError('symlink in retained cache')
    if not path.resolve(strict=True).is_relative_to(root):
        raise ValueError('cache path escaped')
    hashes = {name: hashlib.new(name) for name in ('sha1', 'sha256', 'md5')}
    size = 0
    with path.open('rb') as source:
        initial = os.fstat(source.fileno())
        if not stat.S_ISREG(initial.st_mode):
            raise ValueError('cache member is not regular')
        while chunk := source.read(4 * 1024 * 1024):
            size += len(chunk)
            for digest in hashes.values():
                digest.update(chunk)
        final = os.fstat(source.fileno())
    if (initial.st_size, initial.st_mtime_ns, initial.st_ino) != (final.st_size, final.st_mtime_ns, final.st_ino) or size != row['size']:
        raise ValueError('cache file changed or size mismatch')
    return dict(size=size, **{name: digest.hexdigest() for name, digest in hashes.items()})


def compare_hashes(observed, provider, historical):
    """Separate destination checks from historical D matches; inputs hashes/API/catalog rows, output attribution, no effects; choose before recording. Byline: Codex 2026-10-06."""
    for name, value in provider.get('Hashes', {}).items():
        kind = name.lower().replace('-', '')
        if kind in observed and value and value.lower() != observed[kind]:
            raise ValueError('destination provider hash conflict')
    matched, unavailable = [], []
    for source in historical:
        checks = {}
        kind = (source.get('native_hash_kind') or '').lower().replace('-', '')
        native = source.get('native_hash')
        if kind in ('sha1', 'sha256', 'md5') and native:
            checks[kind] = native.lower()
        if source.get('md5'):
            if 'md5' in checks and checks['md5'] != source['md5'].lower():
                raise ValueError('historical source hash conflict')
            checks['md5'] = source['md5'].lower()
        if source['size'] != observed['size'] or any(observed[k] != v for k, v in checks.items()):
            raise ValueError('historical source hash/size conflict')
        (matched if checks else unavailable).append(dict(occurrence=source, matched_algorithms=sorted(checks)))
    return dict(historical_source_matches=matched, historical_without_supported_hash=unavailable,
                fresh_d_source_hash_verified=False, limits=LIMITS)


def quote(value):
    """Quote fixed-query text literals; input text, output SQL literal, no effects; choose existing catalog operations. Byline: Codex 2026-10-06."""
    return "'" + value.replace("'", "''") + "'"


def pg(sql):
    """Use the inspected existing PostgreSQL catalog; input SQL, output JSON lines, effects only supplied transaction; choose governed ledger access. Byline: Codex 2026-10-06."""
    output = subprocess.run(['docker', 'exec', '-i', PG_CONTAINER, 'psql', '-U', 'postgres', '-d', 'casebible',
                             '-X', '-At', '-v', 'ON_ERROR_STOP=1'], input=sql, text=True,
                            capture_output=True, check=True, timeout=240).stdout
    return [json.loads(line) for line in output.splitlines() if line.startswith('{')]


def historical_sources(rows):
    """Read exact preexisting D occurrences without changing them; input rows, output historical provenance, read-only SQL; choose hash comparison. Byline: Codex 2026-10-06."""
    result = []
    for offset in range(0, len(rows), 1000):
        paths = [r['relative_path'][len('Backup/'):] for r in rows[offset:offset + 1000]]
        result.extend(pg("BEGIN READ ONLY; SET LOCAL statement_timeout='180s'; SELECT row_to_json(s) FROM "
                         "raw_duck.source_occurrences s WHERE source='local/D-Backup' AND path IN (" +
                         ','.join(quote(p) for p in paths) + '); ROLLBACK;'))
    return result


def record_rows(config, verified, pg_execute=pg):
    """Append and independently read back exact existing-ledger payloads; inputs config/verified, output counts, SQL writes; choose after stable versions. Byline: Codex 2026-10-06."""
    records = []
    for row in verified:
        identity = '\n'.join((config['manifest_sha256'], config['upload_sha256'], config['workflow_id'],
                              config['run_id'], row['dest_key'], row['b2_object_version_id']))
        records.append(dict(occurrence_id='d-preservation-verify-20261006:' + hashlib.sha256(identity.encode()).hexdigest(),
                            source_bucket='local/D-Backup', source_path=row['relative_path'][len('Backup/'):],
                            preservation_action='preserved_immutable_verified', payload=row))
    for offset in range(0, len(records), 250):
        batch = records[offset:offset + 250]
        encoded = quote(json.dumps(batch, ensure_ascii=False))
        sql = ("BEGIN; SET LOCAL lock_timeout='15s'; SET LOCAL statement_timeout='120s'; "
               "INSERT INTO raw_duck.r2_candidate_preservation_20260920 "
               "(occurrence_id,source_bucket,source_path,preservation_action,payload) "
               "SELECT occurrence_id,source_bucket,source_path,preservation_action,payload FROM jsonb_to_recordset(" +
               encoded + "::jsonb) AS r(occurrence_id text,source_bucket text,source_path text,preservation_action text,payload jsonb) "
               "ON CONFLICT DO NOTHING; COMMIT;")
        pg_execute(sql)
        actual = pg_execute("BEGIN READ ONLY; SELECT row_to_json(p) FROM raw_duck.r2_candidate_preservation_20260920 p "
                            'WHERE occurrence_id IN (' + ','.join(quote(r['occurrence_id']) for r in batch) + '); ROLLBACK;')
        if {r['occurrence_id']: r for r in actual} != {r['occurrence_id']: r for r in batch}:
            raise ValueError('catalog payload readback conflict or missing row; prior committed batches retained')
        save(Path(config['work']) / ('catalog-readback-' + str(offset).zfill(6) + '.json'), actual)
    return dict(catalog_verified=len(records), table='raw_duck.r2_candidate_preservation_20260920')


def require_server(work):
    """Confine runtime data to ovh-files; input work path, output none, no effects; choose every heavy stage. Byline: Codex 2026-10-06."""
    if os.name != 'posix' or socket.gethostname().split('.')[0] != 'ovh-files':
        raise ValueError('heavy execution requires ovh-files')
    root = Path('/data/consignatio').resolve(strict=True)
    if not Path(work).resolve().is_relative_to(root) or Path(work).resolve() == root:
        raise ValueError('job must be beneath /data/consignatio')


async def completed_upload(config):
    """Check final upload result in its exact Temporal run; input config, output bounded status, network read only; choose validation gate. Byline: Codex 2026-10-06."""
    from temporalio.client import Client, WorkflowExecutionStatus
    client = await Client.connect(config['address'])
    handle = client.get_workflow_handle(config['upload_workflow'], run_id=config['upload_run'])
    description = await handle.describe()
    if description.status != WorkflowExecutionStatus.COMPLETED:
        raise ValueError('upload Temporal run has not completed successfully')
    results = await handle.result(follow_runs=False)
    upload = [r for r in results if r.get('stage') == 'upload']
    if len(upload) != 1 or upload[0].get('sha256') != config['upload_sha256']:
        raise ValueError('completed upload does not bind the supplied final receipt')
    return dict(workflow_id=config['upload_workflow'], run_id=config['upload_run'], status='COMPLETED',
                native_receipt_ref=upload[0]['receipt'], supplied_receipt=config['upload_receipt'])


def run_stage(stage, config):
    """Execute one separately tracked verification operation; inputs stage/config, output receipt ref, effects stage-specific; choose Activity subprocess. Byline: Codex 2026-10-06."""
    work = Path(config['work'])
    require_server(work)
    rows = load_manifest(config)
    result = dict(stage=stage, count=len(rows), bytes=sum(r['size'] for r in rows), byline=BYLINE,
                  host=socket.gethostname(), started_at=datetime.now(timezone.utc).isoformat(),
                  temporal={key: config[key] for key in ('workflow_id', 'run_id', 'activity_id', 'attempt')},
                  tool_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest())
    if stage == 'validate':
        result.update(validate_upload(config, rows), upload_temporal=asyncio.run(completed_upload(config)))
    elif stage == 'historical_sources':
        result['historical_occurrences'] = historical_sources(rows)
    elif stage in ('listing_before', 'listing_after'):
        files = work / 'readback.files'
        if stage == 'listing_before':
            with files.open('x', encoding='utf-8', newline='\n') as output:
                output.write(''.join(r['relative_path'] + '\n' for r in rows))
        raw_path = work / (stage + '-raw.json')
        with raw_path.open('xb') as output, (work / (stage + '.log')).open('xb') as errors:
            subprocess.run(['rclone', 'lsjson', REMOTE, '--recursive', '--fast-list', '--files-only',
                            '--hash', '--metadata', '--files-from-raw', str(files)], stdout=output,
                           stderr=errors, check=True, timeout=3600)
        listing = validate_listing(read_json(raw_path), rows)
        if stage == 'listing_before':
            upload = read_json(config['upload_receipt'])
            previous = validate_listing(read_json(bundled(config['upload_receipt'], upload['api_observation'])), rows)
            if any(previous[k]['ID'] != listing[k]['ID'] for k in listing):
                raise ValueError('destination version differs from completed upload API observation')
        else:
            compare_versions(read_json(work / 'listing_before.json')['objects'], listing)
        result.update(objects=listing, raw_sha256=hashlib.sha256(raw_path.read_bytes()).hexdigest())
    elif stage == 'download':
        cache = work / 'readback'
        cache.mkdir()  # exclusive fresh cache: never reuse unbound same-size stale bytes
        with (work / 'download.log').open('xb') as log:
            subprocess.run(['rclone', 'copy', REMOTE, str(cache), '--files-from-raw', str(work / 'readback.files'),
                            '--no-traverse', '--immutable', '--transfers', '4', '--checkers', '8',
                            '--retries', '2', '--stats', '30s'], stdout=log, stderr=log, check=True, timeout=86400)
        result.update(cache=str(cache), basis='fresh retained server cache; no preexisting files accepted')
    elif stage == 'hash_cache':
        before = read_json(work / 'listing_before.json')['objects']
        sources = {}
        for source in read_json(work / 'historical_sources.json')['historical_occurrences']:
            sources.setdefault('Backup/' + source['path'], []).append(source)
        verified = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            hashes = pool.map(hash_file, [work / 'readback'] * len(rows), rows)
            for row, observed in zip(rows, hashes):
                provider = before[row['relative_path']]
                attribution = compare_hashes(observed, provider, sources.get(row['relative_path'], []))
                verified.append(dict(**row, **attribution, verified=observed, dest_key=PREFIX + row['relative_path'],
                                     b2_object_version_id=provider['ID'], provider='b2', bucket='salem-data',
                                     source_locator='D:/' + row['relative_path'],
                                     hash_observation_kind='full_stream_b2_version_bound',
                                     hash_transport='fresh retained B2 server cache; exact version IDs bracket download and full hashing',
                                     proof_scope='destination_bytes_only; historical source matches separately attributed',
                                     upload_receipt_sha256=config['upload_sha256'], manifest_sha256=config['manifest_sha256'],
                                     workflow_id=config['workflow_id'], run_id=config['run_id'],
                                     evidence_root=str(work), upload_receipt_path=config['upload_receipt'],
                                     manifest_path=config['manifest'], byline=BYLINE))
        result['verified'] = verified
    elif stage == 'record':
        before, after = read_json(work / 'listing_before.json'), read_json(work / 'listing_after.json')
        compare_versions(before['objects'], after['objects'])
        verified = read_json(work / 'hash_cache.json')['verified']
        if {r['relative_path'] for r in verified} != {r['relative_path'] for r in rows} or len(verified) != len(rows):
            raise ValueError('hash coverage incomplete')
        receipt_chain = {name: hashlib.sha256((work / (name + '.json')).read_bytes()).hexdigest() for name in STAGES[:-1]}
        for row in verified:
            row.update(receipt_chain=receipt_chain, verified_at=datetime.now(timezone.utc).isoformat())
        result.update(record_rows(config, verified), proof=LIMITS)
    else:
        raise ValueError('unknown stage')
    path = work / (stage + '.json')
    result['completed_at'] = datetime.now(timezone.utc).isoformat()
    digest = save(path, result)
    return dict(stage=stage, receipt=str(path), sha256=digest, count=len(rows))


async def verification_activity(args):
    """Run one cancellable server stage with heartbeat; input stage/config ref, output receipt ref, owned child effects; choose Temporal tracking. Byline: Codex 2026-10-06."""
    from temporalio import activity
    info = activity.info()
    config = read_json(args['config'])
    config.update(run_id=info.workflow_run_id, workflow_id=info.workflow_id, activity_id=info.activity_id, attempt=info.attempt)
    runtime = Path(config['work']) / (args['stage'] + '-config.json')
    save(runtime, config)
    child = await asyncio.create_subprocess_exec(sys.executable, __file__, '_stage', args['stage'], str(runtime),
                                                stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE,
                                                start_new_session=True)
    task = asyncio.create_task(child.communicate())
    try:
        while not task.done():
            activity.heartbeat(args['stage'])
            await asyncio.wait([task], timeout=10)
        out, error = await task
        if child.returncode:
            raise RuntimeError(error.decode(errors='replace')[-4000:])
        return json.loads(out)
    except BaseException as error:
        save(Path(config['work']) / (args['stage'] + '-failed.json'),
             dict(stage=args['stage'], error=str(error), error_type=type(error).__name__,
                  config_ref=str(runtime), host=socket.gethostname(), at=datetime.now(timezone.utc).isoformat()))
        raise
    finally:
        if child.returncode is None:
            try:
                os.killpg(child.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            try:
                await asyncio.wait_for(child.wait(), timeout=15)
            except asyncio.TimeoutError:
                os.killpg(child.pid, signal.SIGKILL)
                await child.wait()


async def validate_activity(args):
    """Validate the frozen D manifest and its exact completed native upload receipt.

    Inputs: server configuration reference and validate stage.
    Outputs: pinned completion and transport-validation receipt reference.
    Effects: reads artifacts and upload Temporal history; writes retained server receipts.
    Choose before any destination listing or download; delegates verification_activity.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


async def historical_sources_activity(args):
    """Retain existing historical D source observations for the frozen manifest.

    Inputs: server configuration reference and historical_sources stage.
    Outputs: historical occurrence receipt reference with catalog source identities.
    Effects: reads source_occurrences and writes a retained artifact; no catalog mutation.
    Choose after upload validation to distinguish historical source hashes from new destination hashes.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


async def listing_before_activity(args):
    """Bind the exact uploaded destination keys to B2 version IDs before readback.

    Inputs: server configuration reference and listing_before stage.
    Outputs: exact key, size, version and provider metadata receipt reference.
    Effects: read-only B2 listing and retained files-from/listing artifact creation.
    Choose before download to reject missing keys, size changes and upload-version changes.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


async def download_activity(args):
    """Download only the frozen destination keys into a fresh retained VPS cache.

    Inputs: server configuration reference and download stage with reviewed files-from artifact.
    Outputs: cache completion receipt reference.
    Effects: B2 reads and server cache writes; retains all files and leaves source bytes unchanged.
    Choose after the before-version listing instead of desktop hashing or stale-cache reuse.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


async def hash_cache_activity(args):
    """Hash every retained destination byte and compare available historical source hashes.

    Inputs: server configuration reference and hash_cache stage with cache and historical receipts.
    Outputs: size, SHA1, SHA256, MD5 and separately attributed source-match receipt reference.
    Effects: server cache reads and retained receipt writes; no fresh D source payload access.
    Choose after download to reject byte conflicts without claiming format or package validity.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


async def listing_after_activity(args):
    """Verify that B2 destination versions stayed unchanged across full readback hashing.

    Inputs: server configuration reference and listing_after stage with the before-version receipt.
    Outputs: unchanged-version listing receipt reference or explicit failure.
    Effects: read-only B2 listing and retained server artifact writes.
    Choose after hashing and before catalog mutation to close the version bracket.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


async def record_activity(args):
    """Append version-bound destination proof to the existing preservation ledger and read it back.

    Inputs: server configuration reference and record stage with completed hash and version receipts.
    Outputs: exact catalog payload readback and completion receipt references.
    Effects: additive existing-ledger inserts and server receipt writes; no new catalog table.
    Choose only after unchanged destination versions are proven; fresh D source hash remains unverified.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await verification_activity(args)


class VerificationWorkflow:
    """Orchestrate seven independent verification Activities for a completed D preservation batch.

    Inputs: frozen server configuration reference.
    Outputs: ordered retained receipt references, never source payloads.
    Effects: delegates validated B2 reads, retained cache creation and additive ledger recording.
    Choose after native transport completes; never starts or resumes an upload.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    async def run(self, config_path: str) -> list[dict]:
        """Run frozen upload gates, version snapshots, hashing and catalog proof in dependency order.

        Inputs: absolute server configuration path.
        Outputs: seven bounded Activity receipt references.
        Effects: orchestration only; Activities declare their own reads and writes.
        Choose for a completed authorized batch requiring destination-version proof and catalog readback.
        Byline: Codex / GPT-6 / 2026-10-06.
        """
        from temporalio import workflow
        from temporalio.common import RetryPolicy
        result = []
        for stage in STAGES:
            result.append(await workflow.execute_activity('d_preservation_verify_' + stage + '_20261006',
                dict(stage=stage, config=config_path), start_to_close_timeout=timedelta(hours=30),
                heartbeat_timeout=timedelta(seconds=60), retry_policy=RetryPolicy(maximum_attempts=1)))
        return result


async def start(config):
    """Start an isolated one-shot server worker and retain history; input config, output final receipt, own worker effects; choose CLI. Byline: Codex 2026-10-06."""
    from temporalio import activity, workflow
    from temporalio.client import Client
    from temporalio.worker import Worker, UnsandboxedWorkflowRunner
    work = Path(config['work'])
    require_server(work)
    work.mkdir()  # separate batch/run; never reuse or overwrite another worker's artifacts
    config_path = work / 'config.json'
    save(config_path, config)
    activities = [
        activity.defn(name='d_preservation_verify_validate_20261006')(validate_activity),
        activity.defn(name='d_preservation_verify_historical_sources_20261006')(historical_sources_activity),
        activity.defn(name='d_preservation_verify_listing_before_20261006')(listing_before_activity),
        activity.defn(name='d_preservation_verify_download_20261006')(download_activity),
        activity.defn(name='d_preservation_verify_hash_cache_20261006')(hash_cache_activity),
        activity.defn(name='d_preservation_verify_listing_after_20261006')(listing_after_activity),
        activity.defn(name='d_preservation_verify_record_20261006')(record_activity),
    ]
    workflow.run(VerificationWorkflow.run)
    registered = workflow.defn(name='DPreservationVerify20261006')(VerificationWorkflow)
    client = await Client.connect(config['address'])
    queue = QUEUE + '-' + hashlib.sha256(config['workflow_id'].encode()).hexdigest()[:12]
    async with Worker(client, task_queue=queue, workflows=[registered], activities=activities,
                      workflow_runner=UnsandboxedWorkflowRunner(), max_concurrent_activities=1):
        handle = await client.start_workflow(VerificationWorkflow.run, str(config_path),
                                             id=config['workflow_id'], task_queue=queue)
        try:
            results = await handle.result()
            history = await handle.fetch_history()
            receipt = dict(workflow_id=handle.id, run_id=handle.first_execution_run_id, results=results,
                           event_types=[e.event_type for e in history.events], byline=BYLINE, limits=LIMITS)
            save(work / 'temporal-receipt.json', receipt)
            print(json.dumps(receipt))
        except asyncio.CancelledError:
            await handle.cancel()
            raise


def main():
    """Parse explicit frozen server verification references; input CLI, output tracked receipts, no unfinished transfers; choose operator entry. Byline: Codex 2026-10-06."""
    if len(sys.argv) == 4 and sys.argv[1] == '_stage':
        print(json.dumps(run_stage(sys.argv[2], read_json(sys.argv[3]))))
        return
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['temporal'])
    for name in ('manifest', 'upload-receipt', 'upload-sha256', 'upload-workflow', 'upload-run', 'work', 'workflow-id'):
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--manifest-sha256', default=MANIFEST_SHA256)
    parser.add_argument('--expected-count', type=int, default=115005)
    parser.add_argument('--expected-bytes', type=int, default=14915205983)
    parser.add_argument('--address', default='100.91.190.107:7233')
    config = vars(parser.parse_args())
    config.pop('action')
    asyncio.run(start(config))


if __name__ == '__main__':
    main()
