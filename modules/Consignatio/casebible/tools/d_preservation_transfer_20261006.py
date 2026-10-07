"""Transport a reviewed D:/Backup manifest through bounded native Windows Temporal stages.

Inputs: UTF-8 JSON {"files": [{"relative_path": "Backup/...", "size": int,
"mtime_ns": int}]} and its externally reviewed SHA256; no implicit exclusions.
Also accepts explorer JSONL rel_path/size_bytes/source_mtime with explicit UTC
offset; timestamp deltas up to the declared string precision are accepted and
recorded to accommodate rounded observations. Native mtime_ns remains exact.
Conflicting copy targets fail.
Outputs: retained stat, rclone JSON log and remote API listing receipts on E:.
Effects: prepare/dry-run write receipts; upload makes immutable B2 copies only.
Choose for local-only preservation transport, never catalog scans or local hashing.
B2 upload checksum validation is backend transport evidence, not independent full
readback. API-observed digests are recorded separately, never inferred from exit
status. Size-only existing-object skips remain unproven. Equal before/after stats
cannot exclude concurrent writes or establish a consistent snapshot. Parent must
perform VPS hashing/readback, content validation and any Katrina exclusion proof.
CLI: start-worker --manifest PATH --sha256 HEX --work E:/... --workflow-id ID
[--upload --reviewed-dry-run-sha256 HEX]. Without --upload only prepare/dry-run run.
Use --allow-quarantine-paths only for an explicitly approved frozen manifest;
directory labels do not establish junk or corruption. Error rows stay blocked;
zero-byte files require the separate empty-files-only mode described below.
For reviewed non-Backup paths use --whole-d --source-path-allowlist PATH
--source-path-allowlist-sha256 HEX. The UTF-8 raw-line allowlist must contain
exactly the manifest's D-relative file paths; this never authorizes a drive walk.
Use --empty-files-only for a separate manifest containing only size-zero files;
this preserves empty objects/metadata, never nonempty all-zero/corrupt payloads.
Use --literal-path-identity when the reviewed manifest contains literal Unicode
filenames that Windows backend defaults would decode. This pins both local and
B2 encoders to rclone Standard (Slash,Del,Ctl,Dot), whose input/output conversion
is identity, and binds that choice to the reviewed dry-run receipt. Source names,
files-from entries, log objects and B2 keys must remain exactly equal. This flag
does not alter existing receipts or the default behavior of earlier batches.
Prepare/dry-run may run during another upload; operators must queue subsequent
uploads until the primary completes. --transfers defaults to 16 and accepts only
1..16, with eight checkers, 1 MiB read buffers, 16 MiB B2 chunks and one chunk
per file in flight: about 272 MiB of these buffers at 16 transfers, excluding
other process overhead. This is a buffer calculation, not a total RSS guarantee.
For an approved upload use --after-primary to queue a separate Workflow behind
the exact primary run's successful completion; then revalidate sources normally.
Dependencies: temporalio (lazy import); stdlib tests need no SDK. No SSH executor.
Byline: Codex · GPT-6 · 2026-10-06.
"""
import argparse
import asyncio
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import threading
import time
from datetime import datetime, timedelta, timezone

ROOT = Path('D:/')
RCLONE = 'C:/Users/matts/scoop/shims/rclone.exe'
DEST = 'b2:salem-data/consignatio/casevault/Recovered/_Incoming/d-preservation-20261006/'
QUEUE = 'd-preservation-native-20261006'
PRIMARY_ID = 'd-preservation-native-20261006-upload-primary-02'
PRIMARY_RUN = '01a1143e-c7f2-7d44-b0f2-eb253a24be7f'
BYLINE = 'Codex GPT-6 2026-10-06'
LIMITS = 'metadata observations only; concurrent equal-metadata writes remain possible; server verification pending'
LITERAL_ENCODING = 'Slash,Del,Ctl,Dot'


def save(path, value):
    """Create a retained JSON artifact exclusively; inputs path/value, output SHA256, effects new file; choose for receipts. Byline: Codex 2026-10-06."""
    raw = json.dumps(value, ensure_ascii=False, indent=2).encode('utf-8')
    with path.open('xb') as output:
        output.write(raw)
    return hashlib.sha256(raw).hexdigest()


def load_manifest(path, expected, allow_quarantine_paths=False, allowed_paths=None, empty_files_only=False):
    """Validate frozen manifest bytes; inputs path/hash, output bounded rows, no payload reads; choose before every stage. Byline: Codex 2026-10-06."""
    if path.stat().st_size > 128 * 1024 * 1024:
        raise ValueError('manifest exceeds 128 MiB bound')
    raw = path.read_bytes()
    if hashlib.sha256(raw).hexdigest() != expected.lower():
        raise ValueError('frozen manifest SHA256 mismatch')
    text = raw.decode('utf-8-sig')
    if path.suffix.lower() == '.jsonl':
        rows = [json.loads(line) for line in text.splitlines() if line.strip()]
    else:
        data = json.loads(text)
        rows = data['files'] if isinstance(data, dict) else data
    if not isinstance(rows, list) or not 0 < len(rows) <= 150000:
        raise ValueError('manifest must contain 1..150000 files')
    seen = set()
    for row in rows:
        if 'rel_path' in row:
            if row.get('source_root') != 'D:/':
                raise ValueError('explorer source_root must be D:/')
            row['relative_path'] = row['rel_path']
            row['size'] = row['size_bytes']
            timestamp = row['source_mtime']
            instant = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
            if instant.tzinfo is None:
                raise ValueError('source_mtime requires an explicit timezone')
            delta = instant.astimezone(timezone.utc) - datetime(1970, 1, 1, tzinfo=timezone.utc)
            row['mtime_ns'] = (delta.days * 86400 + delta.seconds) * 1000000000 + delta.microseconds * 1000
            fraction = timestamp.split('.')[1].split('+')[0].split('-')[0].rstrip('Z') if '.' in timestamp else ''
            if len(fraction) > 6:
                raise ValueError('source_mtime precision above microseconds requires mtime_ns')
            row['mtime_precision_ns'] = 10 ** (9 - len(fraction))
        if row.get('copy_target_root', DEST).rstrip('/') != DEST.rstrip('/'):
            raise ValueError('manifest copy_target_root contradicts authorized destination')
        expected_key = DEST.split('salem-data/', 1)[1] + row['relative_path']
        if row.get('copy_target_key', expected_key) != expected_key:
            raise ValueError('manifest copy_target_key contradicts authorized destination')
        relative = row['relative_path']
        parts = relative.split('/') if isinstance(relative, str) else []
        if ((allowed_paths is None and (len(parts) < 2 or parts[0] != 'Backup')) or not parts or any(
            not p or p in ('.', '..') or p.rstrip(' .') != p or
            any(c in p for c in '\\:*?"<>|') or any(ord(c) < 32 for c in p) or
            os.path.isreserved(p) for p in parts)):
            raise ValueError('unsafe D-relative path: ' + repr(relative))
        if allowed_paths is not None and relative not in allowed_paths:
            raise ValueError('source path absent from reviewed whole-D allowlist: ' + relative)
        if not allow_quarantine_paths and any(p.casefold() in ('to_be_deleted', '_to_be_deleted', 'quarantine', '_quarantine') for p in parts):
            raise ValueError('quarantine outside transport scope')
        if relative.casefold() in seen:
            raise ValueError('duplicate Windows destination')
        seen.add(relative.casefold())
        if type(row.get('size')) is not int or (row['size'] != 0 if empty_files_only else row['size'] <= 0) or type(row.get('mtime_ns')) is not int:
            raise ValueError('size must be zero in empty-files-only mode, otherwise positive; exact integer mtime_ns required')
        if 'rel_path' not in row:
            row['mtime_precision_ns'] = 1
        if row.get('error') or row.get('errors') or row.get('quarantine') or row.get('excluded'):
            raise ValueError('error/quarantine/excluded row is not transferable')
        if row.get('status') not in (None, 'source_hash_missing', 'source_hash_without_sha1_bridge', 'source_hash_not_on_current_b2', *(['zero_byte'] if empty_files_only else [])):
            raise ValueError('unapproved manifest status')
    if allowed_paths is not None and {row['relative_path'] for row in rows} != allowed_paths:
        raise ValueError('whole-D allowlist must exactly equal frozen manifest paths')
    return rows


def load_allowlist(config):
    """Validate a frozen exact D-relative allowlist; input whole-D flags/hash/path, output set or None, no source reads; choose explicit non-Backup transport scope. Byline: Codex 2026-10-06."""
    path, expected = config.get('source_path_allowlist'), config.get('source_path_allowlist_sha256')
    if not config.get('whole_d'):
        if path or expected:
            raise ValueError('source allowlist requires explicit --whole-d')
        return None
    if not path or not expected:
        raise ValueError('whole-D requires reviewed source-path allowlist and SHA256')
    file = Path(path)
    if file.stat().st_size > 32 * 1024 * 1024:
        raise ValueError('allowlist exceeds 32 MiB bound')
    raw = file.read_bytes()
    if hashlib.sha256(raw).hexdigest() != expected.lower():
        raise ValueError('source-path allowlist SHA256 mismatch')
    paths = raw.decode('utf-8-sig').splitlines()
    if not paths or len(paths) > 150000 or len({p.casefold() for p in paths}) != len(paths) or any(not p for p in paths):
        raise ValueError('allowlist empty, duplicate, blank or over bound')
    return set(paths)


def observe(rows, root=ROOT, heartbeat=lambda: None, whole_d=False):
    """Stat only reviewed files and reject reparse ancestors; inputs rows/root, output observations/errors, no source writes; choose for change detection. Byline: Codex 2026-10-06."""
    base = root.resolve(strict=True)
    observations = []
    for index, row in enumerate(rows):
        result = {'relative_path': row['relative_path']}
        try:
            path = base
            for part in row['relative_path'].split('/'):
                path = path / part
                info = path.lstat()
                if stat.S_ISLNK(info.st_mode) or getattr(info, 'st_file_attributes', 0) & 0x400:
                    raise ValueError('reparse point rejected')
            boundary = base if whole_d else base / 'Backup'
            if not path.resolve(strict=True).is_relative_to(boundary) or not stat.S_ISREG(info.st_mode):
                raise ValueError('not contained regular source file')
            result.update(size=info.st_size, mtime_ns=info.st_mtime_ns)
            precision = row.get('mtime_precision_ns', 1)
            delta = info.st_mtime_ns - row['mtime_ns']
            result.update(mtime_precision_ns=precision, mtime_delta_ns=delta)
            timestamp_matches = delta == 0 if precision == 1 else abs(delta) <= precision
            result['matches_manifest'] = info.st_size == row['size'] and timestamp_matches
        except (OSError, ValueError) as error:
            result.update(matches_manifest=False, error=str(error))
        observations.append(result)
        if index % 100 == 0:
            heartbeat()
    return observations


def backend_encoding_flags(literal_path_identity):
    """Bind literal local names and remote keys to rclone's identity conversion.

    Inputs: explicit Boolean literal-path mode.
    Outputs: fixed local/B2 encoding arguments, or the unchanged legacy defaults.
    Effects: none; no filesystem access or payload hashing.
    Choose for identical manifest, transport and API path namespaces over backend
    defaults that decode full-width punctuation or quote control-picture names.
    Byline: Codex / GPT-6.1-sol high / 2026-10-07; sibling: command.
    """
    if type(literal_path_identity) is not bool:
        raise ValueError('literal_path_identity must be a Boolean')
    return (['--local-encoding', LITERAL_ENCODING, '--b2-encoding', LITERAL_ENCODING]
            if literal_path_identity else [])


def command(files, dry_run, transfers=16, literal_path_identity=False):
    """Build a bounded immutable native copy for a reviewed path namespace.

    Inputs: reviewed files-from path, dry-run flag, 1..16 transfers and explicit
    literal-path identity mode.
    Outputs: fixed rclone argv, including both backend encoders when requested.
    Effects: none; construction never starts a process or reads source payloads.
    Choose for native transport; server verification independently hashes bytes.
    Byline: Codex / GPT-6.1-sol high / 2026-10-07; sibling: backend_encoding_flags.
    """
    if type(transfers) is not int or not 1 <= transfers <= 16:
        raise ValueError('transfers must be an integer from 1 through 16')
    args = [RCLONE, 'copy', 'D:/', DEST, '--files-from-raw', str(files),
            '--fast-list', '--immutable', '--metadata', '--size-only',
            '--local-no-preallocate', '--b2-disable-checksum', '--transfers', str(transfers), '--checkers', '8',
            '--buffer-size', '1MiB', '--b2-chunk-size', '16MiB', '--b2-upload-concurrency', '1',
            '--multi-thread-streams', '0', '--retries', '1', '--low-level-retries', '2',
            '--use-json-log', '--stats', '30s', '--log-level', 'DEBUG']
    return args + backend_encoding_flags(literal_path_identity) + (['--dry-run'] if dry_run else [])


def log_counts(path, rows):
    """Count observed per-object rclone JSON outcomes; inputs log/reviewed rows, output counts, no writes; choose dry-run review without inferring copies from exit status. Byline: Codex 2026-10-06."""
    expected = {row['relative_path'] for row in rows}
    planned, copied, unchanged = set(), set(), set()
    errors = malformed = 0
    with path.open(encoding='utf-8-sig') as log:
        for line in log:
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                malformed += 1
                continue
            message = event.get('msg', '')
            object_path = event.get('object')
            if event.get('level') in ('error', 'critical', 'fatal'):
                errors += 1
            if object_path not in expected:
                continue
            if event.get('skipped') == 'copy' and 'dry-run' in message:
                planned.add(object_path)
            elif message.startswith('Copied ('):
                copied.add(object_path)
            elif message == 'Unchanged skipping':
                unchanged.add(object_path)
    return {'would_copy_count': len(planned), 'dry_run_skipped_copy_count': len(planned),
            'would_copy_manifest_bytes': sum(row['size'] for row in rows if row['relative_path'] in planned),
            'copied_log_count': len(copied), 'existing_unchanged_log_count': len(unchanged),
            'unclassified_manifest_count': len(expected - planned - copied - unchanged),
            'error_log_count': errors, 'malformed_json_lines': malformed,
            'limits': 'Observed log events only; dry-run skipped copy means planned copy, not preservation proof.'}


def run_process(args, output, errors, cancel, heartbeat):
    """Run only an owned child with retained logs; inputs argv/paths/cancel/pulse, output exit code, effects process; choose cancellable native transport. Byline: Codex 2026-10-06."""
    with output.open('xb') as stdout, errors.open('xb') as stderr:
        child = subprocess.Popen(args, stdout=stdout, stderr=stderr, shell=False)
        try:
            while child.poll() is None:
                heartbeat()
                if cancel.wait(1):
                    raise InterruptedError('activity cancelled')
            return child.returncode
        finally:
            if child.poll() is None:
                if os.name == 'nt':
                    subprocess.run(['taskkill.exe', '/PID', str(child.pid), '/T', '/F'],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=20, check=False)
                if child.poll() is None:
                    child.terminate()
                try:
                    child.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait()


def stage(name, config, cancel, heartbeat):
    """Execute a fixed prepare/dry-run/upload stage; inputs frozen references, output bounded receipt, effects declared by stage; choose Temporal Activity execution. Byline: Codex 2026-10-06."""
    if os.name != 'nt':
        raise ValueError('native Windows required')
    work = Path(config['work']).resolve()
    if work.drive.upper() != 'E:' or work.is_relative_to(ROOT.resolve()):
        raise ValueError('receipt directory must be on E:')
    work.mkdir(parents=True, exist_ok=True)
    def pulse():
        """Check owned cancellation during stat batches; no inputs/output, raises on cancel, no host effects; choose bounded preparation. Byline: Codex 2026-10-06."""
        if cancel.is_set():
            raise InterruptedError('activity cancelled during preparation')
        heartbeat()
    allowed_paths = load_allowlist(config)
    rows = load_manifest(Path(config['manifest']), config['sha256'], config.get('allow_quarantine_paths', False),
                         allowed_paths, config.get('empty_files_only', False))
    encoding_flags = backend_encoding_flags(config.get('literal_path_identity', False))
    stamp = str(time.time_ns())
    prefix = work / (name + '-' + stamp)
    before = observe(rows, heartbeat=pulse, whole_d=config.get('whole_d', False))
    before_path = Path(str(prefix) + '-before.json')
    save(before_path, before)
    receipt = dict(stage=name, manifest_sha256=config['sha256'], host=os.environ.get('COMPUTERNAME'),
                   count=len(rows), bytes=sum(r['size'] for r in rows), before=str(before_path),
                   destination=DEST, server_verification='pending', byte_verified=False,
                   limits=LIMITS, temporal=config.get('temporal'), byline=BYLINE,
                   allow_quarantine_paths=config.get('allow_quarantine_paths', False),
                   whole_d=config.get('whole_d', False), empty_files_only=config.get('empty_files_only', False),
                   literal_path_identity=config.get('literal_path_identity', False),
                   backend_encoding=LITERAL_ENCODING if encoding_flags else 'rclone backend defaults',
                   source_path_allowlist_sha256=config.get('source_path_allowlist_sha256'))
    failure = None
    try:
        if not all(r['matches_manifest'] for r in before):
            raise ValueError('source stat mismatch/error; see before receipt')
        if name not in ('prepare', 'dry-run', 'upload'):
            raise ValueError('unknown stage')
        if name != 'prepare':
            if name == 'upload':
                reviewed = Path(config['dry_receipt'])
                raw = reviewed.read_bytes()
                if hashlib.sha256(raw).hexdigest() != config['reviewed_dry_run_sha256']:
                    raise ValueError('reviewed dry-run receipt hash mismatch')
                prior = json.loads(raw)
                if prior['stage'] != 'dry-run' or prior['status'] != 'completed' or prior['manifest_sha256'] != config['sha256']:
                    raise ValueError('dry-run does not approve this manifest')
                if prior.get('allow_quarantine_paths', False) != config.get('allow_quarantine_paths', False):
                    raise ValueError('dry-run quarantine-path authorization differs')
                for key in ('whole_d', 'empty_files_only', 'literal_path_identity'):
                    if prior.get(key, False) != config.get(key, False):
                        raise ValueError('dry-run scope differs: ' + key)
                if prior.get('source_path_allowlist_sha256') != config.get('source_path_allowlist_sha256'):
                    raise ValueError('dry-run source-path allowlist differs')
            files = Path(str(prefix) + '.files')
            with files.open('x', encoding='utf-8', newline='\n') as output:
                output.write(''.join(r['relative_path'] + '\n' for r in rows))
            log = Path(str(prefix) + '-rclone.jsonl')
            receipt['transfers'] = config.get('transfers', 16)
            receipt['checkers'] = 8
            receipt.update(buffer_size='1MiB', b2_chunk_size='16MiB', b2_upload_concurrency=1,
                           estimated_transfer_buffer_mib=config.get('transfers', 16) * 17,
                           memory_limit_scope='read plus multipart buffers only; other RSS overhead excluded')
            rc = run_process(command(files, name == 'dry-run', config.get('transfers', 16),
                                     config.get('literal_path_identity', False)), Path(str(prefix) + '-stdout.log'), log, cancel, heartbeat)
            receipt.update(rclone_exit_code=rc, rclone_json_log=str(log))
            receipt['rclone_observed_counts'] = log_counts(log, rows)
            if rc:
                raise RuntimeError('rclone copy failed: ' + str(rc))
            if name == 'dry-run' and (receipt['rclone_observed_counts']['unclassified_manifest_count'] or
                                      receipt['rclone_observed_counts']['malformed_json_lines'] or
                                      receipt['rclone_observed_counts']['error_log_count']):
                raise RuntimeError('dry-run JSON outcome coverage incomplete; review retained log')
            if name == 'upload':
                api = Path(str(prefix) + '-b2-api.json')
                rc = run_process([RCLONE, 'lsjson', DEST, '--recursive', '--fast-list', '--files-only', '--hash',
                                  '--metadata', '--files-from-raw', str(files), *encoding_flags], api,
                                 Path(str(prefix) + '-b2-api-errors.log'), cancel, heartbeat)
                receipt.update(api_observation=str(api), api_exit_code=rc,
                               checksum_evidence='API-reported hashes only; upload checksum validation not independently attested; full VPS readback pending')
                if rc:
                    raise RuntimeError('B2 API observation failed: ' + str(rc))
                objects = json.loads(api.read_text(encoding='utf-8-sig'))
                actual = {r['Path']: r for r in objects}
                missing = [r['relative_path'] for r in rows if r['relative_path'] not in actual]
                sizes = [r['relative_path'] for r in rows if r['relative_path'] in actual and actual[r['relative_path']]['Size'] != r['size']]
                receipt.update(api_missing_count=len(missing), api_size_mismatch_count=len(sizes),
                               api_hashes_observed_count=sum(bool(r.get('Hashes')) for r in objects))
                if missing or sizes or len(actual) != len(rows):
                    raise RuntimeError('B2 API coverage/size mismatch; see raw API receipt')
        receipt['status'] = 'completed'
    except BaseException as error:
        failure = error
        receipt.update(status='interrupted' if isinstance(error, InterruptedError) else 'failed', error=str(error))
    finally:
        after = observe(rows, whole_d=config.get('whole_d', False))
        after_path = Path(str(prefix) + '-after.json')
        save(after_path, after)
        receipt.update(after=str(after_path), metadata_unchanged=before == after)
        if before != after or not all(r['matches_manifest'] for r in after):
            receipt.update(status='failed', source_changes='observed; transfer is unproven')
            failure = failure or ValueError('source metadata changed')
        path = Path(str(prefix) + '-receipt.json')
        digest = save(path, receipt)
    if failure:
        raise RuntimeError(str(failure) + '; receipt=' + str(path)) from failure
    return {'receipt': str(path), 'sha256': digest, 'stage': name, 'server_verification': 'pending'}


async def transport_activity(args: dict) -> dict:
    """Track one transport stage with heartbeat/cancellation; inputs refs, output receipt, effects owned rclone only; choose native queue. Byline: Codex 2026-10-06."""
    from temporalio import activity
    cancelled = threading.Event()
    info = activity.info()
    args['config'] = {**args['config'], 'temporal': {'workflow_id': info.workflow_id,
                      'run_id': info.workflow_run_id, 'activity_id': info.activity_id, 'attempt': info.attempt}}
    task = asyncio.create_task(asyncio.to_thread(stage, args['stage'], args['config'], cancelled, lambda: None))
    try:
        while not task.done():
            activity.heartbeat(args['stage'])
            await asyncio.wait([task], timeout=10)
        return await task
    except asyncio.CancelledError:
        cancelled.set()
        try:
            await asyncio.shield(task)
        except Exception:
            pass
        raise

class TransportWorkflow:
    """Orchestrate fixed native transport stages; input refs, output receipts, no server verification; choose D preservation. Byline: Codex 2026-10-06."""
    async def run(self, config: dict) -> list[dict]:
        """Run prepare/dry-run and optionally approved upload; input frozen refs, output receipt refs, effects copies only. Byline: Codex 2026-10-06."""
        from temporalio import workflow
        from temporalio.common import RetryPolicy
        result = []
        for name in (('upload',) if config['upload'] else ('prepare', 'dry-run')):
            result.append(await workflow.execute_activity(transport_activity, {'stage': name, 'config': config},
                start_to_close_timeout=timedelta(days=7), heartbeat_timeout=timedelta(seconds=60),
                retry_policy=RetryPolicy(maximum_attempts=1)))
        return result


async def primary_completion_gate() -> dict:
    """Wait for the fixed primary run to complete successfully; no inputs, output predecessor receipt, read-only Temporal effects; choose serialized follow-on uploads. Byline: Codex 2026-10-06."""
    from temporalio import activity
    from temporalio.client import Client
    client = await asyncio.wait_for(Client.connect('100.91.190.107:7233'), timeout=30)
    handle = client.get_workflow_handle(PRIMARY_ID, run_id=PRIMARY_RUN)
    while True:
        activity.heartbeat('waiting for fixed primary completion')
        description = await handle.describe(rpc_timeout=timedelta(seconds=30))
        status = description.status.name
        if status == 'COMPLETED':
            return {'workflow_id': PRIMARY_ID, 'run_id': PRIMARY_RUN, 'status': status}
        if status != 'RUNNING':
            raise RuntimeError('primary ended without successful completion: ' + status)
        for _ in range(6):
            activity.heartbeat('waiting for fixed primary completion')
            await asyncio.sleep(10)


class QueuedTransportWorkflow:
    """Serialize an approved follow-on behind the exact primary; input frozen refs, output receipts, copies only after gate and fresh stats; choose whole-D follow-on queue. Byline: Codex 2026-10-06."""
    async def run(self, config: dict) -> list[dict]:
        """Await fixed predecessor then run a fresh guarded child transfer; input approved config, output predecessor/transfer receipts, effects bounded copies; choose queued upload. Byline: Codex 2026-10-06."""
        from temporalio import workflow
        from temporalio.common import RetryPolicy
        predecessor = await workflow.execute_activity(primary_completion_gate,
            start_to_close_timeout=timedelta(days=7), heartbeat_timeout=timedelta(seconds=120),
            retry_policy=RetryPolicy(maximum_attempts=1))
        results = await workflow.execute_child_workflow(TransportWorkflow.run, config,
            id=workflow.info().workflow_id + '-transfer', task_queue=QUEUE)
        return [{'predecessor': predecessor}, *results]


async def start_worker(config, workflow_id, address):
    """Run a foreground worker and tracked bounded workflow; inputs refs/ID/server, output history receipt, effects own worker lifetime only; choose CLI launch. Byline: Codex 2026-10-06."""
    from temporalio import activity, workflow
    from temporalio.client import Client
    from temporalio.common import RetryPolicy
    from temporalio.worker import Worker, UnsandboxedWorkflowRunner

    registered_activity = activity.defn(name='d_preservation_native_stage_20261006')(transport_activity)
    workflow.run(TransportWorkflow.run)
    registered_workflow = workflow.defn(name='DPreservationNative20261006')(TransportWorkflow)
    registered_gate = activity.defn(name='d_preservation_primary_completion_gate_20261006')(primary_completion_gate)
    workflow.run(QueuedTransportWorkflow.run)
    registered_queued = workflow.defn(name='DPreservationQueuedNative20261006')(QueuedTransportWorkflow)

    client = await Client.connect(address)
    async with Worker(client, task_queue=QUEUE, workflows=[registered_workflow, registered_queued], activities=[registered_activity, registered_gate],
                      workflow_runner=UnsandboxedWorkflowRunner(), max_concurrent_activities=1):
        entry = QueuedTransportWorkflow.run if config.get('after_primary') else TransportWorkflow.run
        handle = await client.start_workflow(entry, config, id=workflow_id, task_queue=QUEUE)
        try:
            results = await handle.result()
            history = await handle.fetch_history()
            save(Path(config['work']) / ('temporal-' + str(time.time_ns()) + '.json'),
                 {'workflow_id': handle.id, 'run_id': handle.first_execution_run_id,
                  'results': results, 'event_types': [e.event_type for e in history.events], 'byline': BYLINE})
            print(json.dumps(results))
        except asyncio.CancelledError:
            await handle.cancel()
            raise


def main():
    """Parse a foreground tracked launch; input CLI, output receipts, no transfer without --upload and reviewed receipt; choose operator entry. Byline: Codex 2026-10-06."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['start-worker'])
    for name in ('manifest', 'sha256', 'work', 'workflow-id'):
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--address', default='100.91.190.107:7233')
    parser.add_argument('--upload', action='store_true')
    parser.add_argument('--allow-quarantine-paths', action='store_true')
    parser.add_argument('--whole-d', action='store_true')
    parser.add_argument('--source-path-allowlist')
    parser.add_argument('--source-path-allowlist-sha256')
    parser.add_argument('--empty-files-only', action='store_true')
    parser.add_argument('--literal-path-identity', action='store_true')
    parser.add_argument('--after-primary', action='store_true')
    parser.add_argument('--transfers', type=int, default=16, choices=range(1, 17))
    parser.add_argument('--dry-receipt')
    parser.add_argument('--reviewed-dry-run-sha256')
    args = vars(parser.parse_args())
    if args['upload'] and not (args['dry_receipt'] and args['reviewed_dry_run_sha256']):
        parser.error('upload requires --dry-receipt and --reviewed-dry-run-sha256')
    if args['whole_d'] != bool(args['source_path_allowlist'] and args['source_path_allowlist_sha256']):
        parser.error('--whole-d requires --source-path-allowlist and --source-path-allowlist-sha256 together')
    if args['after_primary'] and not args['upload']:
        parser.error('--after-primary requires an explicitly reviewed --upload')
    workflow_id, address = args.pop('workflow_id'), args.pop('address')
    args.pop('action')
    asyncio.run(start_worker(args, workflow_id, address))


if __name__ == '__main__':
    main()
