"""Hand an exact completed native preservation batch to its seven-stage VPS verifier.

Inputs: pinned native run and metadata artifacts; CLI selects a local receipt workdir
and an existing server Python runtime. Outputs: Temporal/history, SHA handoff and
catalog-readback references. Effects: metadata copies to the fixed ovh-files job,
then the existing version-bound verifier's readback and ledger writes. Choose this
foreground worker instead of recurring model checks. No D payload reads or hashes,
source writes, host lifecycle actions or UI messages occur here. Explicitly
separate authorized reconciliation retains the original error-bearing history.
Equal source metadata remains an observation, not a consistent source snapshot.
Operational scope defaults remain primary-02; explicit reviewed whole-D configuration
binds positive count/bytes, allowlist and literal filename mode to the upload receipt.
detached_verifier starts one isolated server worker and awaits its Temporal result,
without holding Windows SSH open throughout the download. The optional exact
adopt_completed_verification_run performs read-only finalization of an already
completed seven-stage run, checking its source hash, scope and upload receipt pin;
it never launches another verifier or changes the original stopped orchestration.
Byline: Codex GPT-6 2026-10-06.
"""
import argparse
import asyncio
import hashlib
import json
import os
from pathlib import Path, PureWindowsPath
import shlex
from datetime import timedelta

from temporalio import activity, workflow
from temporalio.client import Client
from temporalio.common import RetryPolicy, WorkflowIDReusePolicy
from temporalio.service import RPCError, RPCStatusCode
from temporalio.worker import Worker, UnsandboxedWorkflowRunner

ROOT = Path('E:/AI_Workspace/Projects/Propria')
RECEIPTS = ROOT / 'modules/Consignatio/docs/receipts/d-preservation-native-20261006'
NATIVE = RECEIPTS / 'upload-primary-02'
MANIFEST = RECEIPTS / 'partition-dryrun02-20261006/eligible.jsonl'
MANIFEST_SHA = '3e2560c91e8d51cc205cf19f89bec552aa26f0a8506153785adb181942803e42'
PRIMARY = 'd-preservation-native-20261006-upload-primary-02'
PRIMARY_RUN = '01a1143e-c7f2-7d44-b0f2-eb253a24be7f'
FOLLOW = 'd-preservation-native-20261006-followthrough-primary-02-v2'
RECONCILE = 'd-preservation-native-20261006-reconcile-primary-02'
VERIFY = 'd-preservation-vps-verify-20261006-primary-02'
QUEUE = 'd-preservation-followthrough-native-20261006'
ADDRESS = '100.91.190.107:7233'
HOST = 'root@100.91.190.107'
KEY = 'C:/Users/matts/.ssh/ovh'
BASE = '/data/consignatio/d-preservation-verify-20261006-primary'
INBOX = BASE + '/native-primary-02'
SERVER_WORK = BASE + '/verify-primary-02'
VERIFIER = '/data/consignatio/d-preservation-verify-20261006-png/d_preservation_verify_20261006.py'
VERIFIER_SHA = '05ba1f83eecf46cd44377cfdb9b3183fae552631d15d2dd5bf61394a1babf32f'
BYLINE = 'Codex GPT-6 2026-10-06'
SSH = ['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=20', '-o', 'ServerAliveInterval=15', '-o', 'ServerAliveCountMax=4', '-i', KEY, HOST]


def server_command(argv):
    """Bind server Python to its already installed SDK; input fixed argv, output quoted command, no effects; choose existing cache without installation. Byline: Codex 2026-10-06."""
    return shlex.join(['env', 'PYTHONPATH=/data/consignatio/quarantine-catalog-20261006/python-deps', *argv])


def digest(path):
    """Hash a metadata artifact only; input path, output SHA256, read effects; choose receipt transport integrity. Byline: Codex 2026-10-06."""
    result = hashlib.sha256()
    with Path(path).open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            result.update(chunk)
    return result.hexdigest()


def save(path, value):
    """Persist an exclusive receipt; inputs path/value, output reference, disk write; choose immutable audit evidence. Byline: Codex 2026-10-06."""
    path = Path(path)
    with path.open('x', encoding='utf-8') as output:
        json.dump(value, output, indent=2, sort_keys=True)
    return dict(receipt=str(path), sha256=digest(path))


def safe_native(reference, native=None):
    """Confine a referenced artifact to the exact upload directory; input ref, output existing path, no writes; choose all native receipt refs. Byline: Codex 2026-10-06."""
    path = Path(reference)
    if path.is_symlink() or path.resolve(strict=True).parent != (native or NATIVE).resolve(strict=True):
        raise ValueError('artifact outside exact native job: ' + str(path))
    if not path.is_file():
        raise ValueError('artifact is not a file')
    return path


def bind_upload(results, native=None, workflow_id=None, run_id=None, config=None):
    """Bind successful native output to frozen receipt and artifacts; input exact-run result, output handoff map, metadata reads only; choose completion gate. Byline: Codex 2026-10-06."""
    uploads = [r for r in results if r.get('stage') == 'upload']
    if len(uploads) != 1:
        raise ValueError('exact run lacks one upload result')
    item = uploads[0]
    receipt = safe_native(item['receipt'], native)
    if digest(receipt) != item['sha256'] or digest(MANIFEST) != MANIFEST_SHA:
        raise ValueError('frozen artifact SHA mismatch')
    data = json.loads(receipt.read_text(encoding='utf-8'))
    config = config or {}
    count, size = config.get('expected_count', 115005), config.get('expected_bytes', 14915205983)
    if type(count) is not int or not 1 <= count <= 150000 or type(size) is not int or size <= 0:
        raise ValueError('positive bounded scope count/bytes required')
    required = dict(stage='upload', status='completed', manifest_sha256=MANIFEST_SHA,
                    count=count, bytes=size, metadata_unchanged=True,
                    rclone_exit_code=0, api_exit_code=0, api_missing_count=0, api_size_mismatch_count=0)
    if config.get('whole_d'):
        required.update(whole_d=True, empty_files_only=False,
                        source_path_allowlist_sha256=config['source_path_allowlist_sha256'])
    if config.get('literal_path_identity'):
        required.update(literal_path_identity=True, backend_encoding='Slash,Del,Ctl,Dot')
    elif data.get('literal_path_identity', False):
        raise ValueError('literal upload requires an explicit matching reader mode')
    if any(data.get(k) != v for k, v in required.items()) or data.get('error') or data.get('source_changes'):
        raise ValueError('upload receipt is incomplete, changed or failed')
    temporal = data.get('temporal') or {}
    if temporal.get('workflow_id') != (workflow_id or PRIMARY) or temporal.get('run_id') != (run_id or PRIMARY_RUN):
        raise ValueError('native receipt run mismatch')
    files = [MANIFEST, receipt] + [safe_native(data[k], native) for k in ('before', 'after', 'api_observation', 'rclone_json_log')]
    if config.get('whole_d'):
        allowlist = Path(config['source_path_allowlist'])
        if allowlist.is_symlink() or digest(allowlist) != config['source_path_allowlist_sha256']:
            raise ValueError('frozen whole-D allowlist differs')
        files.append(allowlist)
    if len({p.name for p in files}) != len(files):
        raise ValueError('artifact basename collision')
    artifacts = [dict(native_ref=str(p), server_ref=INBOX + '/' + p.name,
                      basename=p.name, bytes=p.stat().st_size, sha256=digest(p)) for p in files]
    return dict(upload_sha256=item['sha256'], upload_receipt=INBOX + '/' + receipt.name,
                manifest=INBOX + '/' + MANIFEST.name, artifacts=artifacts,
                workflow_id=workflow_id or PRIMARY, run_id=run_id or PRIMARY_RUN,
                expected_count=count, expected_bytes=size,
                source_path_allowlist=INBOX + '/' + Path(config['source_path_allowlist']).name if config.get('whole_d') else None,
                requires_reconciliation=bool(data.get('rclone_observed_counts', {}).get('error_log_count', 0)), byline=BYLINE)


@activity.defn(name='d_preservation_followthrough_bind_reconciliation_20261006')
async def bind_reconciliation(args):
    """Bind a completed clean child while retaining original artifacts; input child refs/original, output combined map, metadata reads; choose recovered transport errors. Byline: Codex 2026-10-06."""
    client = await connected()
    handle = client.get_workflow_handle(RECONCILE, run_id=args['run_id'])
    results = await handle.result(follow_runs=False)
    description = await handle.describe(rpc_timeout=timedelta(seconds=30))
    if description.status.name != 'COMPLETED':
        raise ValueError('reconciliation not completed')
    bound = await asyncio.to_thread(bind_upload, results, Path(args['native']), RECONCILE, args['run_id'])
    if bound['requires_reconciliation']:
        raise ValueError('separate reconciliation still contains errors; no false clean history')
    original = args['original']
    seen = {a['basename'] for a in bound['artifacts']}
    bound['artifacts'].extend(a for a in original['artifacts'] if a['basename'] not in seen)
    bound['original_upload'] = original
    save(Path(args['work']) / 'reconciliation-binding.json', bound)
    return bound


@activity.defn(name='d_preservation_followthrough_transport_20261006')
async def owned_transport(args):
    """Execute one authorized native stage on this isolated worker; input config/stage, output receipt, source-read/copy effects; choose serialized reconciliation and D-other. Byline: Codex 2026-10-06."""
    import d_preservation_transfer_20261006 as transport
    return await transport.transport_activity(args)


@workflow.defn(name='DPreservationFollowthroughCopy20261006', sandboxed=False)
class OwnedCopy:
    """Track one serialized guarded copy or fresh reviewed partition; input config, output receipts, immutable copy effects; choose followthrough children. Byline: Codex 2026-10-06."""
    @workflow.run
    async def run(self, config: dict) -> list:
        """Run fresh review when requested then upload; input refs, output receipts, guarded copy effects; choose owner-authorized partition only. Byline: Codex 2026-10-06."""
        options = dict(start_to_close_timeout=timedelta(days=7), heartbeat_timeout=timedelta(seconds=120),
                       retry_policy=RetryPolicy(maximum_attempts=1))
        results = []
        if config.get('fresh_review'):
            for stage in ('prepare', 'dry-run'):
                results.append(await workflow.execute_activity(owned_transport, dict(stage=stage, config=config), **options))
            config = dict(config, dry_receipt=results[-1]['receipt'], reviewed_dry_run_sha256=results[-1]['sha256'])
        results.append(await workflow.execute_activity(owned_transport, dict(stage='upload', config=config), **options))
        return results


async def connected():
    """Connect with a bounded initial timeout; no inputs, output client, network read; choose all fixed Temporal access. Byline: Codex 2026-10-06."""
    return await asyncio.wait_for(Client.connect(ADDRESS), 30)


async def await_event(task):
    """Heartbeat while one existing future waits; input Task, output result, no status polling; choose long Temporal result or SSH wait. Byline: Codex 2026-10-06."""
    try:
        while not task.done():
            activity.heartbeat('awaiting result event; no describe polling')
            await asyncio.wait({task}, timeout=10)
        return task.result()
    finally:
        if not task.done():
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)


async def command(argv, work, label, cwd=None):
    """Run one fixed metadata transport command with heartbeat; inputs argv/work/label, output exit success, own subprocess effects; choose SSH/SCP only. Byline: Codex 2026-10-06."""
    with (Path(work) / (label + '.stdout')).open('xb') as out, (Path(work) / (label + '.stderr')).open('xb') as err:
        process = await asyncio.create_subprocess_exec(*argv, cwd=cwd, stdout=out, stderr=err)
        try:
            code = await await_event(asyncio.create_task(process.wait()))
            if code:
                raise RuntimeError(label + ' failed: ' + str(code) + '; inspect retained stderr')
        finally:
            if process.returncode is None:
                if os.name == 'nt':
                    killer = await asyncio.create_subprocess_exec('taskkill', '/PID', str(process.pid), '/T', '/F',
                                                                  stdout=asyncio.subprocess.DEVNULL, stderr=asyncio.subprocess.DEVNULL)
                    await killer.wait()
                else:
                    process.terminate()
                await process.wait()


@activity.defn(name='d_preservation_followthrough_await_20261006')
async def await_primary(config):
    """Await the exact pinned native upload and bind its reviewed scope.

    Inputs: local receipt work and optional positive whole-D count/allowlist/mode.
    Outputs: successful exact-run metadata binding or a retained stopped receipt.
    Effects: read-only Temporal wait and local metadata reads/writes; no copy dispatch.
    Choose for predecessor admission before handoff; never follow replacement runs.
    Byline: Codex / GPT-6.1-sol high / 2026-10-07; sibling: bind_reconciliation.
    """
    try:
        client = await connected()
        handle = client.get_workflow_handle(PRIMARY, run_id=PRIMARY_RUN)
        result = await await_event(asyncio.create_task(handle.result(follow_runs=False)))
        description = await handle.describe(rpc_timeout=timedelta(seconds=30))
        if description.status.name != 'COMPLETED':
            raise ValueError('primary did not complete')
        bound = await asyncio.to_thread(bind_upload, result, config=config)
        save(Path(config['work']) / 'primary-completed-binding.json', bound)
        return bound
    except Exception as error:
        save(Path(config['work']) / 'primary-gate-stopped.json', dict(error=str(error), workflow_id=PRIMARY, run_id=PRIMARY_RUN))
        raise


@activity.defn(name='d_preservation_followthrough_handoff_20261006')
async def handoff(args):
    """Copy only bound metadata into an exclusive VPS inbox and verify SHAs; input config/binding, output audit ref, SSH/SCP effects; choose after successful native completion. Byline: Codex 2026-10-06."""
    config, bound = args['config'], args['bound']
    work = Path(config['work'])
    index = work / 'handoff-index.json'
    save(index, bound)
    await command(SSH + ['mkdir -p ' + shlex.quote(BASE) + ' && mkdir ' + shlex.quote(INBOX)], work, 'create-inbox')
    for position, item in enumerate(bound['artifacts']):
        source = Path(item['native_ref'])
        if await asyncio.to_thread(digest, source) != item['sha256']:
            raise ValueError('metadata changed before transport')
        await command(['scp', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=20', '-i', KEY,
                       source.name, HOST + ':' + item['server_ref']], work, 'copy-' + str(position), str(source.parent))
    await command(['scp', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=20', '-i', KEY,
                   index.name, HOST + ':' + INBOX + '/handoff-index.json'], work, 'copy-index', str(work))
    # Hash only the explicitly bound metadata artifacts, never source payloads.
    script = ("import hashlib,json,pathlib; p=pathlib.Path(" + repr(INBOX) + "); "
              "d=json.loads((p/'handoff-index.json').read_text()); "
              "assert all(x['basename']==pathlib.Path(x['basename']).name for x in d['artifacts']); "
              "assert all(hashlib.file_digest((p/x['basename']).open('rb'),'sha256').hexdigest()==x['sha256'] "
              "and (p/x['basename']).stat().st_size==x['bytes'] for x in d['artifacts']); "
              "print(json.dumps(d,sort_keys=True))")
    await command(SSH + [server_command([config['server_python'], '-c', script])], work, 'server-sha-check')
    observed = json.loads((work / 'server-sha-check.stdout').read_text(encoding='utf-8'))
    if observed != bound:
        raise ValueError('remote SHA proof did not bind exact handoff')
    return save(work / 'handoff-receipt.json', dict(**bound, server_hash_checks_passed=True,
                index_sha256=digest(index), verifier_sha256=VERIFIER_SHA))


def verifier_argv(config, bound):
    """Build the fixed seven-Activity verifier CLI; input runtime/binding, output argv, no effects; choose exclusively the primary batch. Byline: Codex 2026-10-06."""
    args = [config['server_python'], VERIFIER, 'temporal', '--manifest', bound['manifest'],
            '--manifest-sha256', MANIFEST_SHA, '--upload-receipt', bound['upload_receipt'],
            '--upload-sha256', bound['upload_sha256'], '--upload-workflow', bound['workflow_id'],
            '--upload-run', bound['run_id'], '--work', SERVER_WORK, '--workflow-id', VERIFY,
            '--expected-count', str(config.get('expected_count', 115005)),
            '--expected-bytes', str(config.get('expected_bytes', 14915205983)), '--address', ADDRESS]
    if config.get('whole_d'):
        args.extend(['--whole-d', '--source-path-allowlist', bound['source_path_allowlist'],
                     '--source-path-allowlist-sha256', config['source_path_allowlist_sha256']])
    if config.get('literal_path_identity'):
        args.append('--literal-path-identity')
    return args


async def existing_verifier(client):
    """Reject any existing stable verification execution; input client, output none, one describe read; choose duplicate prevention, never retry uncertain execution. Byline: Codex 2026-10-06."""
    try:
        found = await client.get_workflow_handle(VERIFY).describe(rpc_timeout=timedelta(seconds=30))
    except RPCError as error:
        if error.status == RPCStatusCode.NOT_FOUND:
            return
        raise
    raise RuntimeError('stable verifier already exists: ' + found.status.name + '; no duplicate launch')


@activity.defn(name='d_preservation_followthrough_verify_20261006')
async def remote_verify(args):
    """Track one VPS verification and retain its exact completed catalog references.

    Inputs: pinned successful upload binding, scope and reader source identity;
    optional detached launch or exact completed verification run for read-only adoption.
    Outputs: final receipt with actual seven-Activity results and catalog readback pins.
    Effects: fresh mode starts one server verifier; adoption only reads existing proof.
    Choose detached launch to avoid holding Windows SSH for the full readback;
    choose adoption only after a completed verifier outlives its native SSH orchestration.
    Byline: Codex / GPT-6.1-sol high / 2026-10-07; sibling: existing_verifier.
    """
    config, bound = args['config'], args['bound']
    work = Path(config['work'])
    client = await connected()
    adopted_run = config.get('adopt_completed_verification_run')
    if not adopted_run:
        await existing_verifier(client)
    # Exclusive on-disk claim closes cross-client races before the existing CLI starts.
    check = ("import hashlib,pathlib; p=pathlib.Path(" + repr(VERIFIER) + "); "
             "assert hashlib.sha256(p.read_bytes()).hexdigest()==" + repr(VERIFIER_SHA) + "; "
             "pathlib.Path(" + repr(BASE + '/primary-02-verifier.claim') + ").mkdir()")
    if not adopted_run:
        await command(SSH + [server_command([config['server_python'], '-c', check])], work, 'claim-verifier')
        if config.get('detached_verifier'):
            launch = ("import os,subprocess,pathlib,json; p=pathlib.Path(" + repr(BASE + '/verifier-worker.log') + "); "
                      "log=p.open('xb'); child=subprocess.Popen(" + repr(verifier_argv(config, bound)) +
                      ",stdin=subprocess.DEVNULL,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,"
                      "env={**os.environ,'PYTHONPATH':'/data/consignatio/quarantine-catalog-20261006/python-deps'}); "
                      "print(json.dumps({'owned_worker_pid':child.pid}))")
            await command(SSH + [server_command([config['server_python'], '-c', launch])], work, 'verifier-launch')
            # Startup visibility is bounded; after creation wait on one Temporal result event.
            for attempt in range(12):
                try:
                    await client.get_workflow_handle(VERIFY).describe(rpc_timeout=timedelta(seconds=30))
                    break
                except RPCError as error:
                    if error.status != RPCStatusCode.NOT_FOUND:
                        raise
                    activity.heartbeat('waiting for newly launched verifier registration')
                    await asyncio.sleep(5)
            else:
                raise RuntimeError('detached verifier did not register; retain worker log, never duplicate launch')
        else:
            await command(SSH + [server_command(verifier_argv(config, bound))], work, 'verifier')
    handle = client.get_workflow_handle(VERIFY, run_id=adopted_run) if adopted_run else client.get_workflow_handle(VERIFY)
    if config.get('detached_verifier') and not adopted_run:
        await await_event(asyncio.create_task(handle.result(follow_runs=False)))
    description = await handle.describe(rpc_timeout=timedelta(seconds=30))
    if description.status.name != 'COMPLETED':
        raise ValueError('server verifier did not complete')
    results = await handle.result(follow_runs=False)
    if [r.get('stage') for r in results] != ['validate', 'historical_sources', 'listing_before', 'download', 'hash_cache', 'listing_after', 'record']:
        raise ValueError('unexpected verification stage results')
    expected = config.get('expected_count', 115005)
    script = ("import hashlib,json,pathlib; w=pathlib.Path(" + repr(SERVER_WORK) + "); "
              "c=json.loads((w/'config.json').read_text()); "
              "assert c['manifest_sha256']==" + repr(MANIFEST_SHA) + "; "
              "assert c['upload_sha256']==" + repr(bound['upload_sha256']) + "; "
              "p=w/'record.json'; d=json.loads(p.read_text()); "
              "assert d['catalog_verified']==" + repr(expected) + "; "
              "assert d['count']==" + repr(expected) + " and d['bytes']==" + repr(config.get('expected_bytes',14915205983)) + "; "
              "assert d['tool_sha256']==" + repr(VERIFIER_SHA) + "; "
              "assert d['temporal']['run_id']==" + repr(description.run_id) + "; "
              "refs=[dict(path=str(p),sha256=hashlib.file_digest(p.open('rb'),'sha256').hexdigest()) "
              "for p in sorted(w.glob('catalog-readback-*.json'))]; "
              "assert len(refs)==" + repr((expected + 249)//250) + "; "
              "print(json.dumps(dict(record=d,catalog_readbacks=refs,temporal=json.loads((w/'temporal-receipt.json').read_text()))))")
    await command(SSH + [server_command([config['server_python'], '-c', script])], work, 'catalog-readback')
    catalog = json.loads((work / 'catalog-readback.stdout').read_text(encoding='utf-8'))
    return save(work / 'followthrough-completed.json', dict(status='completed', verification_workflow=VERIFY,
                verification_run=description.run_id, upload_sha256=bound['upload_sha256'], catalog=catalog,
                verification_launch='adopted_exact_completed_run' if adopted_run else 'new_single_run',
                limits='Destination version-bound readback; no fresh source byte hash or consistent D snapshot.', byline=BYLINE))


@workflow.defn(name='DPreservationFollowthrough20261006', sandboxed=False)
class Followthrough:
    """Track native completion, metadata handoff and VPS verification; input config, output refs, delegated effects; choose deterministic one-shot followthrough. Byline: Codex 2026-10-06."""
    @workflow.run
    async def run(self, config: dict) -> list:
        """Sequence three non-retrying Activities; input work/runtime, output receipts, effects only after successful predecessor; choose tracked orchestration. Byline: Codex 2026-10-06."""
        options = dict(start_to_close_timeout=timedelta(days=7), heartbeat_timeout=timedelta(seconds=120),
                       retry_policy=RetryPolicy(maximum_attempts=1))
        bound = await workflow.execute_activity(await_primary, config, **options)
        if bound['requires_reconciliation']:
            child = await workflow.start_child_workflow(OwnedCopy.run, config['reconciliation'], id=RECONCILE, task_queue=QUEUE,
                                                       id_reuse_policy=WorkflowIDReusePolicy.REJECT_DUPLICATE)
            await child
            bound = await workflow.execute_activity(bind_reconciliation, dict(original=bound, run_id=child.first_execution_run_id,
                native=config['reconciliation']['work'], work=config['work']), **options)
        args = dict(config=config, bound=bound)
        transferred = await workflow.execute_activity(handoff, args, **options)
        copies = []
        for batch in config.get('d_other', []):
            copies.append(await workflow.execute_child_workflow(OwnedCopy.run, batch['config'], id=batch['workflow_id'], task_queue=QUEUE,
                          id_reuse_policy=WorkflowIDReusePolicy.REJECT_DUPLICATE))
        verified = await workflow.execute_activity(remote_verify, args, **options)
        return [transferred, *copies, verified]


async def start(config):
    """Run one owned foreground worker and retain launch/final history; input config, output receipts, worker process effects; choose cached native CLI. Byline: Codex 2026-10-06."""
    if os.name != 'nt':
        raise ValueError('native Windows only')
    work = Path(config['work'])
    if not work.resolve().is_relative_to(RECEIPTS.resolve()):
        raise ValueError('receipt work must remain beneath fixed receipt root')
    work.mkdir()
    client = await connected()
    async with Worker(client, task_queue=QUEUE, workflows=[Followthrough, OwnedCopy], activities=[await_primary, handoff, remote_verify, bind_reconciliation, owned_transport],
                      workflow_runner=UnsandboxedWorkflowRunner(), max_concurrent_activities=1):
        handle = await client.start_workflow(Followthrough.run, config, id=FOLLOW, task_queue=QUEUE,
                                            id_reuse_policy=WorkflowIDReusePolicy.REJECT_DUPLICATE)
        save(work / 'launch-receipt.json', dict(workflow_id=FOLLOW, run_id=handle.first_execution_run_id,
                                              primary=PRIMARY, primary_run=PRIMARY_RUN, config=config, byline=BYLINE))
        try:
            result = await handle.result(follow_runs=False)
            save(work / 'temporal-completed.json', dict(results=result, workflow_id=FOLLOW, run_id=handle.first_execution_run_id))
        except BaseException as error:
            save(work / 'followthrough-stopped.json', dict(error=repr(error), workflow_id=FOLLOW,
                run_id=handle.first_execution_run_id, limits='No retry; inspect server execution before any recovery.'))
            raise
        finally:
            history = await handle.fetch_history()
            save(work / 'temporal-history.json', history.to_json())


def main():
    """Parse the fixed followthrough launch; input CLI work/runtime, output audit artifacts, no UI messages; choose operator startup. Byline: Codex 2026-10-06."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--work', required=True)
    parser.add_argument('--server-python', required=True)
    parser.add_argument('--copy-plan', required=True, help='Frozen metadata-only plan for authorized reconciliation and D-other partitions')
    parser.add_argument('--copy-plan-sha256', required=True)
    args = vars(parser.parse_args())
    if not args['server_python'].startswith('/') or any(c.isspace() for c in args['server_python']):
        parser.error('server Python must be an absolute executable path')
    plan = Path(args.pop('copy_plan'))
    if digest(plan) != args.pop('copy_plan_sha256'):
        parser.error('copy-plan SHA mismatch')
    args.update(json.loads(plan.read_text(encoding='utf-8')))
    asyncio.run(start(args))


if __name__ == '__main__':
    main()
