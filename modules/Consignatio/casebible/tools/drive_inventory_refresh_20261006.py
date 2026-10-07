#!/usr/bin/env python3
"""Track native D metadata observation and frozen VPS catalog admission as separate operations.

Inputs: scan/ship/load operation, unique E: job, workflow ID and frozen receipt pin
for ship/load. Outputs: retained metadata/transfer/validation/catalog references.
Effects: scan reads D metadata only; ship copies derived artifacts over existing SSH;
load mutates the existing VPS catalog only after a pinned parent lake-finalization release.
Choose scan for parent-approved observation while lake export is running; ship validates
without loading; load requires a separate released server bundle. Never hashes D payloads.
Siblings: local_drive_listing.py, drive_inventory_load_20261006.py; follows reviewed
inventory_refresh_20261006.py separately tracked operation and receipt patterns.
Byline: Codex / GPT-6 / 2026-10-06.
"""
import argparse
import asyncio
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import socket
import subprocess
import threading
import time
from datetime import timedelta
import uuid

import local_drive_listing as walker

HOST = 'root@100.91.190.107'
KEY = 'C:/Users/matts/.ssh/ovh'
SERVER_ROOT = '/data/consignatio/drive-inventory-refresh-20261006'


def require_native(work):
    """Restrict production orchestration to native Windows and E artifacts; input work, no output/effects; choose all local operations. Byline: Codex 2026-10-06."""
    if os.name != 'nt' or Path(work).absolute().drive.upper() != 'E:':
        raise ValueError('native Windows and E: artifacts required')


def run_command(argv, work, stem, cancelled, timeout=2400, stdin=None):
    """Run one owned SSH/scp child with retained logs and cancellation; inputs argv/job/stem/cancel, output stdout, own child effects; choose metadata transport. Byline: Codex 2026-10-06."""
    output_path, error_path = Path(work) / (stem + '.out'), Path(work) / (stem + '.err')
    start = time.monotonic()
    with output_path.open('xb') as output, error_path.open('xb') as errors:
        child = subprocess.Popen(argv, stdout=output, stderr=errors, stdin=subprocess.PIPE if stdin else subprocess.DEVNULL)
        if stdin:
            child.stdin.write(stdin.encode('utf-8'))
            child.stdin.close()
        try:
            while child.poll() is None:
                if cancelled.wait(0.5):
                    raise InterruptedError('owned metadata command cancelled')
                if time.monotonic() - start > timeout:
                    raise TimeoutError('metadata command timeout')
            if child.returncode:
                raise RuntimeError('metadata command failed; retained log=' + str(error_path))
        finally:
            if child.poll() is None:
                child.terminate()
                try:
                    child.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait()
    return output_path.read_text(encoding='utf-8-sig')


def ssh(argv):
    """Encode a fixed remote argv for existing SSH; input argv, output SSH argv, no effects; choose server metadata processing. Byline: Codex 2026-10-06."""
    return ['ssh', '-i', KEY, '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=15', HOST, shlex.join(argv)]


def read_receipt(path, sha256):
    """Pin a retained metadata receipt; inputs path/SHA256, output JSON, no effects; choose frozen ship/load references. Byline: Codex 2026-10-06."""
    raw = Path(path).read_bytes()
    if not re.fullmatch('[a-f0-9]{64}', sha256) or hashlib.sha256(raw).hexdigest() != sha256:
        raise ValueError('receipt SHA256 mismatch')
    return json.loads(raw)


def transfer(config, reference, cancelled):
    """Copy only frozen metadata and reviewed producer code into a unique VPS bundle.

    Inputs: pinned local listing receipt, local job configuration and cancellation event.
    Outputs: server bundle and unchanged receipt pin.
    Effects: SSH/scp of derived JSONL, receipt and small source files; no D payload access.
    Choose after scan, independently of lake publication; never loads the catalog.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    receipt_path = Path(reference['receipt'])
    require_native(receipt_path)
    native = read_receipt(receipt_path, reference['receipt_sha256'])
    if not re.fullmatch('[a-f0-9]{32}', native.get('generation_id', '')):
        raise ValueError('invalid generation')
    bundle = SERVER_ROOT + '/' + native['generation_id']
    create = "from pathlib import Path; p=Path(" + repr(SERVER_ROOT) + "); p.mkdir(parents=True,exist_ok=True); (p/" + repr(native['generation_id']) + ').mkdir()'
    run_command(ssh(['python3', '-c', create]), config['work'], 'transfer-mkdir', cancelled)
    source = receipt_path.parent
    files = [source / 'listing.jsonl', receipt_path, source / 'walker-source.py',
             Path(__file__).with_name('local_drive_listing.py'), Path(__file__).with_name('drive_inventory_load_20261006.py')]
    for file in files:
        if not file.is_file() or file.is_symlink():
            raise ValueError('missing or unsafe frozen metadata artifact')
    run_command(['scp', '-i', KEY, '-o', 'BatchMode=yes', *map(str, files), HOST + ':' + bundle + '/'],
                config['work'], 'transfer-scp', cancelled)
    return dict(bundle=bundle, receipt_sha256=reference['receipt_sha256'], generation_id=native['generation_id'])


def cancel_catalog(tag):
    """Cancel only this Activity's tagged PostgreSQL statement; input controlled tag, no output, own DB-session effects; choose SSH interruption cleanup. Byline: Codex 2026-10-06."""
    if not re.fullmatch('driveinv:[a-f0-9]{32}:(validate|load|readback)', tag):
        raise ValueError('invalid owned SQL tag')
    sql = "SELECT pg_cancel_backend(pid) FROM pg_stat_activity WHERE pid<>pg_backend_pid() AND application_name='" + tag + "';"
    subprocess.run(ssh(['docker', 'exec', '-i', 'casebible-pg-oli1nf8wmj5atb9o7oylj4um-131418412167',
                        'psql', '-U', 'postgres', '-d', 'casebible', '-X', '-At']), input=sql, text=True,
                   capture_output=True, timeout=30, check=True)


def remote_stage(stage, config, reference, cancelled):
    """Run one VPS-only validation/load/readback subprocess; inputs stage/config/ref/cancel, output bounded refs, declared server effects; choose separate Activity. Byline: Codex 2026-10-06."""
    bundle = reference['bundle']
    if not re.fullmatch(re.escape(SERVER_ROOT) + '/[a-f0-9]{32}', bundle):
        raise ValueError('invalid frozen server bundle')
    operation_id = config['operation_id']
    if not re.fullmatch('[a-f0-9]{32}', operation_id):
        raise ValueError('invalid operation identity')
    tag = 'driveinv:' + operation_id + ':' + stage
    artifacts = bundle + '/' + stage + '-' + operation_id
    argv = ['env', 'PGAPPNAME=' + tag, 'PYTHONPATH=/data/consignatio/quarantine-catalog-20261006/python-deps',
            'python3', '-B', bundle + '/drive_inventory_load_20261006.py', stage, '--bundle', bundle,
            '--receipt-sha256', reference['receipt_sha256'], '--artifacts', artifacts]
    if stage == 'load':
        if not config.get('release') or not config.get('release_sha256'):
            raise ValueError('production load held: pinned parent lake-finalization release required')
        argv += ['--release', config['release'], '--release-sha256', config['release_sha256']]
    try:
        raw = run_command(ssh(argv), config['work'], stage, cancelled)
    except (InterruptedError, TimeoutError):
        if stage in ('load', 'readback'):
            cancel_catalog(tag)
        raise
    results = [json.loads(line) for line in raw.splitlines() if line.startswith('{')]
    if len(results) != 1:
        raise ValueError('server stage did not return one bounded receipt')
    return dict(reference, result=results[0])


async def dispatch_activity(args):
    """Execute a tracked native or SSH stage with bounded heartbeat and retained failure evidence; input refs, output refs, stage effects; choose internal delegation. Byline: Codex 2026-10-06."""
    from temporalio import activity
    config = json.loads(Path(args['config']).read_text())
    require_native(config['work'])
    stage = args['stage']
    cancelled, progress = threading.Event(), {}
    info = activity.info()
    temporal = dict(workflow_id=info.workflow_id, run_id=info.workflow_run_id, activity_id=info.activity_id, attempt=info.attempt)
    def operation():
        """Perform the chosen operation in a cancellable thread; no inputs, bounded refs output, declared stage effects; choose heartbeat isolation. Byline: Codex 2026-10-06."""
        if stage == 'scan':
            return walker.scan('D:/', Path(config['work']) / 'scan', config['operation_id'], temporal,
                               cancelled.is_set, lambda value: progress.update(value))
        if stage == 'transfer':
            return transfer(config, args['reference'], cancelled)
        return remote_stage(stage, config, args['reference'], cancelled)
    task = asyncio.create_task(asyncio.to_thread(operation))
    try:
        while not task.done():
            activity.heartbeat(dict(stage=stage, progress=dict(progress)))
            await asyncio.wait([task], timeout=10)
        result = await task
        walker.save(Path(config['work']) / (stage + '-result.json'), dict(result=result, temporal=temporal, host=socket.gethostname()))
        return result
    except BaseException as error:
        cancelled.set()
        try:
            await asyncio.shield(task)
        except BaseException:
            pass
        walker.save(Path(config['work']) / (stage + '-failed.json'), dict(error=str(error), stage=stage, temporal=temporal, host=socket.gethostname()))
        raise


async def scan_activity(args):
    """Observe whole-D metadata with explicit coverage and no payload reads.

    Inputs: native E: configuration and scan stage.
    Outputs: retained JSONL and completion receipt references.
    Effects: Windows metadata reads and E: artifact writes only.
    Choose after parent reviews the tested scan command; may run while lake export runs.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await dispatch_activity(args)


async def transfer_activity(args):
    """Transfer a frozen drive metadata observation to an exclusive VPS bundle.

    Inputs: native receipt reference/pin and transfer stage.
    Outputs: server bundle and receipt pin.
    Effects: SSH/scp of derived metadata and source code only.
    Choose after scan; no production catalog write or lake release required.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await dispatch_activity(args)


async def validate_activity(args):
    """Validate drive listing identity, coverage and counts entirely on ovh-files.

    Inputs: frozen server bundle and receipt SHA256.
    Outputs: validated generation and derived TSV references.
    Effects: VPS metadata reads and derived artifact writes only.
    Choose after transfer; never promotes an incomplete observation.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await dispatch_activity(args)


async def load_activity(args):
    """Admit a frozen generation through the existing drive inventory and approved receipt ledger.

    Inputs: server bundle, receipt pin and pinned parent lake-finalization release.
    Outputs: atomic additive load receipt reference.
    Effects: VPS catalog DDL/DML only after release; partial/failed generations remain non-current.
    Choose after lake publication finalizes and the parent excludes overlapping publication.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await dispatch_activity(args)


async def readback_activity(args):
    """Independently compare every loaded metadata row and its receipt with frozen input.

    Inputs: server bundle and receipt pin after catalog load.
    Outputs: exact-row, exact-receipt and selected-generation evidence.
    Effects: VPS temporary staging, catalog reads and retained result artifacts.
    Choose after commit to verify complete/empty selection and partial/failed non-selection.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    return await dispatch_activity(args)


class DriveInventoryRefresh:
    """Orchestrate separately tracked scan, ship or released catalog admission.

    Inputs: native configuration path, operation and initial frozen reference.
    Outputs: bounded stage results and retained receipts.
    Effects: delegates only the stages named by the explicit operation.
    Choose scan during publication, ship for validation, and load after parent release.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    async def run(self, args: dict) -> list[dict]:
        """Run only the selected operation's dependency sequence.

        Inputs: config reference, operation and optional frozen receipt/bundle reference.
        Outputs: ordered bounded Activity results.
        Effects: orchestration only; no automatic progression from scan to catalog load.
        Choose to keep production load independently held while observation can proceed.
        Byline: Codex / GPT-6 / 2026-10-06.
        """
        from temporalio import workflow
        from temporalio.common import RetryPolicy
        stages = {'scan': ('scan',), 'ship': ('transfer', 'validate'), 'load': ('load', 'readback')}[args['operation']]
        result, reference = [], args.get('reference')
        for stage in stages:
            reference = await workflow.execute_activity('drive_inventory_' + stage + '_20261006',
                dict(config=args['config'], stage=stage, reference=reference), start_to_close_timeout=timedelta(hours=12),
                heartbeat_timeout=timedelta(seconds=60), retry_policy=RetryPolicy(maximum_attempts=1))
            result.append(reference)
        return result


async def start(config, reference):
    """Start one foreground native worker and retain its Temporal history; input config/ref, output receipts, own worker effects; choose explicit CLI operation. Byline: Codex 2026-10-06."""
    from temporalio import activity, workflow
    from temporalio.client import Client
    from temporalio.worker import Worker, UnsandboxedWorkflowRunner
    require_native(config['work'])
    work = Path(config['work'])
    work.mkdir(parents=True, exist_ok=False)
    config_path = work / 'config.json'
    walker.save(config_path, config)
    activities = [activity.defn(name='drive_inventory_scan_20261006')(scan_activity),
                  activity.defn(name='drive_inventory_transfer_20261006')(transfer_activity),
                  activity.defn(name='drive_inventory_validate_20261006')(validate_activity),
                  activity.defn(name='drive_inventory_load_20261006')(load_activity),
                  activity.defn(name='drive_inventory_readback_20261006')(readback_activity)]
    workflow.run(DriveInventoryRefresh.run)
    registered = workflow.defn(name='DriveInventoryRefresh20261006')(DriveInventoryRefresh)
    client = await Client.connect(config['address'])
    queue = 'drive-inventory-20261006-' + config['operation_id']
    async with Worker(client, task_queue=queue, workflows=[registered], activities=activities,
                      workflow_runner=UnsandboxedWorkflowRunner(), max_concurrent_activities=1):
        handle = await client.start_workflow(DriveInventoryRefresh.run,
            dict(config=str(config_path), operation=config['operation'], reference=reference), id=config['workflow_id'], task_queue=queue)
        try:
            results = await handle.result()
            history = await handle.fetch_history()
            receipt = dict(workflow_id=handle.id, run_id=handle.first_execution_run_id, results=results,
                           event_types=[e.event_type for e in history.events], byline=walker.BYLINE)
            walker.save(work / 'temporal-receipt.json', receipt)
            print(json.dumps(receipt))
        except asyncio.CancelledError:
            await handle.cancel()
            raise


def main():
    """Parse an explicitly separated native scan, metadata ship or released load.

    Inputs: operation, unique E: work path, workflow ID and operation-specific frozen pins.
    Outputs: tracked receipt references; no operation is launched merely by importing tools.
    Effects: delegates only requested operation; load requires a parent release on the VPS.
    Choose scan only after parent reviews the concrete command; production load stays separately gated.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=['scan', 'ship', 'load'])
    parser.add_argument('--work', required=True)
    parser.add_argument('--workflow-id', required=True)
    parser.add_argument('--address', default='100.91.190.107:7233')
    parser.add_argument('--receipt')
    parser.add_argument('--receipt-sha256')
    parser.add_argument('--server-bundle')
    parser.add_argument('--release')
    parser.add_argument('--release-sha256')
    config = vars(parser.parse_args())
    reference = None
    if config['operation'] == 'ship':
        if not config['receipt'] or not config['receipt_sha256']:
            parser.error('ship requires frozen --receipt and --receipt-sha256')
        reference = dict(receipt=config['receipt'], receipt_sha256=config['receipt_sha256'])
    elif config['operation'] == 'load':
        if not all(config.get(k) for k in ('server_bundle', 'receipt_sha256', 'release', 'release_sha256')):
            parser.error('load requires frozen server bundle/pin and pinned parent release after lake finalizes')
        reference = dict(bundle=config['server_bundle'], receipt_sha256=config['receipt_sha256'])
    config['operation_id'] = uuid.uuid4().hex
    asyncio.run(start(config, reference))


if __name__ == '__main__':
    main()
