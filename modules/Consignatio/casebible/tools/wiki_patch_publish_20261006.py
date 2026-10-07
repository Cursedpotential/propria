#!/usr/bin/env python3
"""Publish one reviewed fixed-target patch to the existing Case Bible Code wiki.

Inputs: a server-local reviewed ZIP or directory containing the fixed payloads and hash manifest.
Outputs: retained preparation/upload/readback receipts with a SHA-256 for every remote file.
Effects: prepare stores local backups and a frozen manifest; upload writes only the fixed allowlisted B2 objects.
Choose after the parent has reviewed the final guide, receipt, INDEX and expected prior hashes; never for bulk sync.
Byline: Codex / GPT-6 Luna / 2026-10-06.
"""
from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import os
import re
import subprocess
import uuid
import zipfile
from datetime import timedelta, datetime, timezone
from pathlib import Path
from typing import Any

try:
    from temporalio import activity as temporal_activity
    from temporalio import workflow as temporal_workflow
    from temporalio.client import Client
    from temporalio.worker import Worker
except ImportError:  # The pure helpers remain testable on desktops without the server SDK.
    temporal_activity = None
    temporal_workflow = None
    Client = None
    Worker = None


SERVER_PROJECT_ROOT = Path('/data/consignatio')
PROJECT_ROOT = SERVER_PROJECT_ROOT / 'wiki-patch-publish-20261006'
REMOTE_PREFIX = 'b2native-full:salem-data/consignatio/casevault/Code/wiki'
QUEUE = 'casebible-wiki-patch-publish-v1'
MANIFEST_NAME = 'wiki-patch-manifest.json'
MANIFEST_SCHEMA = 'wiki-patch-publish-v1'
TARGETS = (
    'catalog-guide.md',
    'receipts/corpus-inventory-preservation-20261006.md',
    'INDEX.md',
    'plugin-inventory.md',
    'assets/tool-inventory.json',
    'assets/human-commands.csv',
)
NEW_INDEX_LINK = 'Code/wiki/receipts/corpus-inventory-preservation-20261006'
HASH_RE = re.compile(r'^[0-9a-f]{64}$')
MAX_FILE_BYTES = 5 * 1024 * 1024
MAX_MANIFEST_BYTES = 64 * 1024


def _activity_defn(name: str):
    """Build an Activity decorator when Temporal is installed.

    Inputs: the fixed Activity name. Output: a decorator bound to the SDK or an identity decorator for local tests.
    Effects: none. Choose for preserving importable validation helpers on desktops without server dependencies.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if temporal_activity is not None:
        return temporal_activity.defn(name=name)

    def decorate(function):
        """Return the undecorated function for local tests.

        Inputs: a function. Output: the same function. Effects: none; only the missing-SDK test path uses this.
        Choose through `_activity_defn` when Temporal is unavailable.
        Byline: Codex / GPT-6 Luna / 2026-10-06.
        """
        return function

    return decorate


def _workflow_defn(name: str):
    """Build a Workflow decorator when Temporal is installed.

    Inputs: the fixed Workflow name. Output: an SDK or local-test class decorator.
    Effects: none. Choose to keep pure helper tests independent of the server SDK.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if temporal_workflow is not None:
        return temporal_workflow.defn(name=name)

    def decorate(value):
        """Return the undecorated class for local tests.

        Inputs: a class. Output: the same class. Effects: none; only the missing-SDK test path uses this.
        Choose through `_workflow_defn` when Temporal is unavailable.
        Byline: Codex / GPT-6 Luna / 2026-10-06.
        """
        return value

    return decorate


def _workflow_run(function):
    """Apply Temporal's Workflow run marker when available.

    Inputs: the Workflow entry function. Output: the SDK-decorated or unchanged function.
    Effects: none. Choose for the single durable publish sequence.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    return temporal_workflow.run(function) if temporal_workflow is not None else function


def sha256(data: bytes) -> str:
    """Return the SHA-256 identity of exact bytes.

    Inputs: raw bytes. Output: lowercase hex digest. Effects: none.
    Choose for reviewed payloads, remote observations, backups and readback receipts.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    return hashlib.sha256(data).hexdigest()


def _rclone(*arguments: str, input_bytes: bytes | None = None) -> bytes:
    """Run one bounded rclone command without exposing server credentials.

    Inputs: exact argument vector and optional upload bytes. Output: captured stdout.
    Effects: only the rclone operation requested; callers restrict writes to fixed object keys.
    Choose over shell strings for remote reads and one-object uploads.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    result = subprocess.run(
        ['rclone', *arguments], input=input_bytes, capture_output=True, timeout=300,
    )
    if result.returncode:
        operation = arguments[0] if arguments else 'unknown'
        raise RuntimeError(f'rclone {operation} failed with exit {result.returncode}')
    return result.stdout


def _remote_path(relative: str) -> str:
    """Resolve an allowlisted wiki-relative path to its one B2 key.

    Inputs: candidate relative path. Output: full key under the existing Code/wiki prefix.
    Effects: none. Choose only for paths in the `len(TARGETS)`-object reviewed allowlist.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if relative not in TARGETS:
        raise ValueError(f'Path is outside the fixed wiki patch: {relative}')
    return f'{REMOTE_PREFIX}/{relative}'


def _remote_bytes(relative: str) -> bytes | None:
    """Read one exact B2 object, distinguishing an absent key from a zero-byte file.

    Inputs: one allowlisted relative path. Output: exact bytes, or None only when rclone reports the empty directory marker.
    Effects: B2 metadata/content reads. Choose before backups, compare-before-write and final hash readback.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    remote = _remote_path(relative)
    stat_raw = _rclone('lsjson', '--stat', remote)
    try:
        stat = json.loads(stat_raw)
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise RuntimeError(f'Invalid rclone stat for {relative}') from exc
    if stat.get('IsDir') is True:
        if stat.get('Path') == '' and stat.get('Name') == '':
            return None
        raise RuntimeError(f'Remote target is a directory, not a file: {relative}')
    if stat.get('IsDir') is not False or stat.get('Name') != Path(relative).name:
        raise RuntimeError(f'Unexpected rclone stat result for {relative}')
    data = _rclone('cat', remote)
    if type(stat.get('Size')) is not int or stat['Size'] != len(data):
        raise RuntimeError(f'Remote size changed during read: {relative}')
    return data


def _write_once(path: Path, data: bytes) -> None:
    """Create a local retained artifact without replacing prior bytes.

    Inputs: destination and exact bytes. Output: none; accepts a prior file only when bytes match.
    Effects: creates a file exclusively or verifies an identical retry; never truncates or deletes.
    Choose for frozen manifests and server-side backups.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    try:
        with path.open('xb') as stream:
            stream.write(data)
    except FileExistsError:
        if path.read_bytes() != data:
            raise RuntimeError(f'Refusing to replace retained artifact: {path}')


def _safe_run_id(value: str) -> str:
    """Validate the Temporal run ID before using it in server paths.

    Inputs: Temporal run ID. Output: the same ID if limited to safe filename characters.
    Effects: none. Choose before creating run receipts or backups.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if not re.fullmatch(r'[A-Za-z0-9_-]{1,128}', value):
        raise ValueError('Invalid Temporal run ID')
    return value


def _read_bundle(bundle: Path) -> tuple[dict[str, bytes], bytes]:
    """Read the exact reviewed payload set from a directory or ZIP without extracting it.

    Inputs: a bundle beneath the fixed server project root. Outputs: len(TARGETS) payload byte strings and manifest bytes.
    Effects: local reads only; traversal, symlink, extra-file and size violations are rejected.
    Choose to accept the parent's reviewed folder or ZIP while keeping the remote path set fixed.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    base = bundle.resolve(strict=True)
    project = PROJECT_ROOT.resolve()
    if not base.is_relative_to(project):
        raise ValueError('Reviewed bundle must be inside the server wiki patch project directory')
    names = set(TARGETS) | {MANIFEST_NAME}
    if base.is_dir():
        found: dict[str, bytes] = {}
        for item in base.rglob('*'):
            if item.is_symlink():
                raise ValueError('Reviewed bundle cannot contain symlinks')
            if item.is_dir():
                continue
            name = item.relative_to(base).as_posix()
            if name not in names or name in found:
                raise ValueError(f'Unexpected or duplicate bundle file: {name}')
            limit = MAX_MANIFEST_BYTES if name == MANIFEST_NAME else MAX_FILE_BYTES
            if item.stat().st_size > limit:
                raise ValueError(f'Reviewed bundle file exceeds size limit: {name}')
            found[name] = item.read_bytes()
    elif base.is_file() and zipfile.is_zipfile(base):
        found = {}
        with zipfile.ZipFile(base) as archive:
            for member in archive.infolist():
                name = member.filename
                if member.is_dir():
                    if name.rstrip('/') not in ('receipts',):
                        raise ValueError(f'Unexpected directory in reviewed ZIP: {name}')
                    continue
                if name not in names or name in found:
                    raise ValueError(f'Unexpected or duplicate bundle file: {name}')
                limit = MAX_MANIFEST_BYTES if name == MANIFEST_NAME else MAX_FILE_BYTES
                if member.file_size > limit:
                    raise ValueError(f'Reviewed bundle file exceeds size limit: {name}')
                found[name] = archive.read(member)
    else:
        raise ValueError('Reviewed bundle must be a directory or ZIP')
    if set(found) != names:
        raise ValueError(f'Reviewed bundle must contain exactly {len(TARGETS)} payloads plus the manifest: {", ".join(sorted(names))}')
    return {name: found[name] for name in TARGETS}, found[MANIFEST_NAME]


def _parse_manifest(raw: bytes, payloads: dict[str, bytes]) -> tuple[list[dict[str, Any]], str]:
    """Validate the parent-reviewed expected-old/new-hash manifest against all local payload bytes.

    Inputs: manifest JSON and exact fixed payload mapping. Output: ordered normalized file entries and manifest hash.
    Effects: none. Choose before comparing the current remote objects or taking backups.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if len(raw) > MAX_MANIFEST_BYTES:
        raise ValueError('Wiki patch manifest is too large')
    try:
        document = json.loads(raw)
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise ValueError('Wiki patch manifest is not valid UTF-8 JSON') from exc
    if (not isinstance(document, dict) or set(document) != {'schema', 'byline', 'files'}
            or document.get('schema') != MANIFEST_SCHEMA
            or not isinstance(document.get('byline'), str) or not document['byline'].strip()):
        raise ValueError('Unsupported wiki patch manifest schema')
    rows = document.get('files')
    if not isinstance(rows, list) or len(rows) != len(TARGETS):
        raise ValueError(f'Manifest must list exactly len(TARGETS)={len(TARGETS)} fixed wiki files')
    by_path: dict[str, dict[str, Any]] = {}
    for row in rows:
        if not isinstance(row, dict) or set(row) != {'path', 'expected_old_sha256', 'new_sha256'}:
            raise ValueError('Manifest rows must contain only path, expected_old_sha256 and new_sha256')
        path = row['path']
        old_hash = row['expected_old_sha256']
        new_hash = row['new_sha256']
        if path not in TARGETS or path in by_path:
            raise ValueError(f'Unexpected or duplicate manifest path: {path}')
        if old_hash is not None and (not isinstance(old_hash, str) or not HASH_RE.fullmatch(old_hash)):
            raise ValueError(f'Invalid expected old SHA-256 for {path}')
        if not isinstance(new_hash, str) or not HASH_RE.fullmatch(new_hash):
            raise ValueError(f'Invalid new SHA-256 for {path}')
        if sha256(payloads[path]) != new_hash:
            raise ValueError(f'Reviewed payload hash differs from manifest: {path}')
        by_path[path] = {'path': path, 'expected_old_sha256': old_hash, 'new_sha256': new_hash}
    if set(by_path) != set(TARGETS):
        raise ValueError('Manifest does not cover the fixed target set')
    normalized = [by_path[path] for path in TARGETS]
    return normalized, sha256(raw)


def _link_targets(markdown: str) -> set[str]:
    """Collect Obsidian link targets from the reviewed wiki index.

    Inputs: INDEX Markdown text. Output: normalized target set without labels or heading fragments.
    Effects: none. Choose to ensure the patch preserves current links and adds only the approved receipt link.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    return {match.split('|', 1)[0].split('#', 1)[0].strip() for match in re.findall(r'\[\[([^\]]+)\]\]', markdown)}


def _validate_index_extension(old: bytes, new: bytes) -> None:
    """Require the reviewed INDEX to preserve links and add the fixed receipt link only on initial publication.

    Inputs: current remote INDEX bytes and reviewed replacement bytes. Output: none on success.
    Effects: none. Choose before initial publish or later refresh to preserve current navigation without new links.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    try:
        old_text, new_text = old.decode('utf-8'), new.decode('utf-8')
    except UnicodeDecodeError as exc:
        raise ValueError('Wiki INDEX must be UTF-8 Markdown') from exc
    prior_links = _link_targets(old_text)
    next_links = _link_targets(new_text)
    if not prior_links.issubset(next_links):
        raise ValueError('Reviewed INDEX removes one or more existing links')
    expected_additions = set() if NEW_INDEX_LINK in prior_links else {NEW_INDEX_LINK}
    if next_links - prior_links != expected_additions:
        raise ValueError('Reviewed INDEX may add only the corpus inventory receipt link on initial publication')


def _load_prepared(run_id: str) -> tuple[dict[str, Any], Path]:
    """Read the frozen manifest for one prepared Temporal run.

    Inputs: validated run ID. Outputs: prepared JSON and its run directory.
    Effects: local read only. Choose in upload and readback so later stages use prepare-time hashes.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    safe_id = _safe_run_id(run_id)
    run_dir = PROJECT_ROOT / 'runs' / safe_id
    manifest_path = run_dir / 'expected-old-new-hash-manifest.json'
    document = json.loads(manifest_path.read_text(encoding='utf-8'))
    if document.get('schema') != MANIFEST_SCHEMA or document.get('run_id') != safe_id:
        raise RuntimeError('Prepared manifest identity/schema mismatch')
    entries = document.get('files')
    if not isinstance(entries, list) or [entry.get('path') for entry in entries] != list(TARGETS):
        raise RuntimeError('Prepared manifest target set is invalid')
    return document, run_dir


def _prepare_sync(args: dict[str, str]) -> dict[str, Any]:
    """Validate reviewed bytes, compare current remote hashes and retain exact prior-byte backups.

    Inputs: fixed bundle path and Temporal run ID. Outputs: frozen old/new hash manifest and backup locators.
    Effects: server-local writes only under the patch project and its to_be_deleted backup directory; B2 is read-only.
    Choose as the first Activity before any wiki object is written.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    run_id = _safe_run_id(args['run_id'])
    run_dir = PROJECT_ROOT / 'runs' / run_id
    payloads, input_manifest = _read_bundle(Path(args['bundle']))
    rows, input_manifest_sha256 = _parse_manifest(input_manifest, payloads)
    existing_prepared = run_dir / 'expected-old-new-hash-manifest.json'
    if existing_prepared.exists():
        prepared, _ = _load_prepared(run_id)
        if prepared.get('input_manifest_sha256') != input_manifest_sha256:
            raise RuntimeError('This Temporal run ID already has a different reviewed manifest')
        for entry in prepared['files']:
            backup = entry.get('backup_path')
            if backup and sha256(Path(backup).read_bytes()) != entry['expected_old_sha256']:
                raise RuntimeError(f'Retained prior-byte backup hash mismatch: {entry["path"]}')
        return {'run_id': run_id, 'manifest_path': str(existing_prepared), 'status': 'prepared',
                'remote_prefix': REMOTE_PREFIX, 'files': prepared['files']}

    remote_values: dict[str, bytes | None] = {}
    for entry in rows:
        current = _remote_bytes(entry['path'])
        current_hash = sha256(current) if current is not None else None
        if current_hash != entry['expected_old_sha256']:
            raise RuntimeError(f'Remote prior hash differs from reviewed manifest: {entry["path"]}')
        remote_values[entry['path']] = current

    _validate_index_extension(remote_values['INDEX.md'] or b'', payloads['INDEX.md'])
    backup_dir = SERVER_PROJECT_ROOT / 'to_be_deleted' / 'wiki-patch-publish-20261006' / run_id
    prepared_rows = []
    for entry in rows:
        current = remote_values[entry['path']]
        backup_path = None
        if current is not None:
            backup_path = backup_dir / entry['path']
            _write_once(backup_path, current)
            if sha256(backup_path.read_bytes()) != entry['expected_old_sha256']:
                raise RuntimeError(f'Prior-byte backup verification failed: {entry["path"]}')
        prepared_rows.append({**entry, 'backup_path': str(backup_path) if backup_path else None,
                              'old_bytes': len(current) if current is not None else None,
                              'new_bytes': len(payloads[entry['path']])})

    run_dir.mkdir(parents=True, exist_ok=True)
    frozen = {
        'schema': MANIFEST_SCHEMA,
        'run_id': run_id,
        'remote_prefix': REMOTE_PREFIX,
        'input_bundle': str(Path(args['bundle']).resolve()),
        'input_manifest_sha256': input_manifest_sha256,
        'files': prepared_rows,
        'prepared_at': datetime.now(timezone.utc).isoformat(),
        'byline': 'Codex / GPT-6 Luna / 2026-10-06',
    }
    raw_frozen = (json.dumps(frozen, ensure_ascii=False, indent=2) + '\n').encode('utf-8')
    _write_once(existing_prepared, raw_frozen)
    return {'run_id': run_id, 'manifest_path': str(existing_prepared), 'status': 'prepared',
            'remote_prefix': REMOTE_PREFIX, 'files': prepared_rows}


@_activity_defn('casebible_wiki_patch_prepare_v1')
async def prepare(args: dict[str, str]) -> dict[str, Any]:
    """Run the read-only remote preflight and retain manifest/backups before upload.

    Inputs: server bundle path and Temporal run ID. Output: prepared fixed-file manifest.
    Effects: B2 reads plus local manifest and to_be_deleted backup writes; no B2 mutation.
    Choose first in the three-Activity wiki patch workflow, which stages len(TARGETS) fixed objects.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    return await asyncio.to_thread(_prepare_sync, args)


def _upload_sync(args: dict[str, str]) -> dict[str, Any]:
    """Write every reviewed fixed object after an immediate expected-prior-hash check for each.

    Inputs: run ID, bundle path and frozen prepare receipt. Outputs: per-object write/skip receipt.
    Effects: at most len(TARGETS) exact B2 object writes using rclone rcat; no list-wide sync or deletes.
    Choose only after prepare succeeds and the parent has authorized the reviewed final bundle.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    run_id = _safe_run_id(args['run_id'])
    prepared, _ = _load_prepared(run_id)
    payloads, input_manifest = _read_bundle(Path(args['bundle']))
    _, current_manifest_hash = _parse_manifest(input_manifest, payloads)
    if current_manifest_hash != prepared.get('input_manifest_sha256'):
        raise RuntimeError('Reviewed bundle changed after prepare')
    results = []
    for entry in prepared['files']:
        path = entry['path']
        body = payloads[path]
        if len(body) != entry['new_bytes'] or sha256(body) != entry['new_sha256']:
            raise RuntimeError(f'Reviewed payload changed after prepare: {path}')
        current = _remote_bytes(path)  # This compare is immediately before this one-object write.
        current_hash = sha256(current) if current is not None else None
        if current_hash == entry['new_sha256']:
            results.append({'path': path, 'status': 'already_applied', 'sha256': current_hash})
            continue
        if current_hash != entry['expected_old_sha256']:
            raise RuntimeError(f'Remote changed after prepare; refusing overwrite: {path}')
        _rclone('rcat', _remote_path(path), input_bytes=body)
        results.append({'path': path, 'status': 'written', 'sha256': entry['new_sha256']})
    return {'run_id': run_id, 'status': 'uploaded', 'remote_prefix': REMOTE_PREFIX,
            'files': results, 'byline': 'Codex / GPT-6 Luna / 2026-10-06'}


@_activity_defn('casebible_wiki_patch_upload_v1')
async def upload(args: dict[str, str]) -> dict[str, Any]:
    """Run the compare-before-write upload Activity.

    Inputs: run ID and the unchanged reviewed bundle. Output: per-object write status.
    Effects: at most len(TARGETS) allowlisted B2 object writes after immediate prior-hash checks.
    Choose second in the workflow, only after prepare has retained verified backups.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    return await asyncio.to_thread(_upload_sync, args)


def _readback_sync(args: dict[str, str]) -> dict[str, Any]:
    """Read back every fixed target and require its bytes to match the reviewed SHA-256.

    Inputs: Temporal run ID. Outputs: exact remote size/hash results for all len(TARGETS) objects.
    Effects: B2 reads only and a retained local readback receipt; no remote mutation.
    Choose last in the workflow before reporting publication verified.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    run_id = _safe_run_id(args['run_id'])
    prepared, run_dir = _load_prepared(run_id)
    verified = []
    for entry in prepared['files']:
        actual = _remote_bytes(entry['path'])
        if actual is None:
            raise RuntimeError(f'Remote file missing at readback: {entry["path"]}')
        digest = sha256(actual)
        if digest != entry['new_sha256'] or len(actual) != entry['new_bytes']:
            raise RuntimeError(f'Remote readback mismatch: {entry["path"]}')
        verified.append({'path': entry['path'], 'bytes': len(actual), 'sha256': digest,
                         'status': 'match'})
    receipt = {'run_id': run_id, 'status': 'verified', 'remote_prefix': REMOTE_PREFIX,
               'verified_at': datetime.now(timezone.utc).isoformat(), 'files': verified,
               'byline': 'Codex / GPT-6 Luna / 2026-10-06'}
    raw_receipt = (json.dumps(receipt, ensure_ascii=False, indent=2) + '\n').encode('utf-8')
    receipt_path = run_dir / 'readback-receipt.json'
    if receipt_path.exists():
        prior = json.loads(receipt_path.read_text(encoding='utf-8'))
        if (prior.get('run_id') != run_id or prior.get('status') != 'verified'
                or prior.get('remote_prefix') != REMOTE_PREFIX or prior.get('files') != verified):
            raise RuntimeError('Existing readback receipt differs from current remote bytes')
        return prior
    _write_once(receipt_path, raw_receipt)
    return receipt


@_activity_defn('casebible_wiki_patch_readback_v1')
async def readback(args: dict[str, str]) -> dict[str, Any]:
    """Run the independent remote readback Activity.

    Inputs: Temporal run ID. Output: verified receipt with a hash for each published object.
    Effects: remote reads and one exclusive local receipt write.
    Choose last in the three-Activity workflow.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    return await asyncio.to_thread(_readback_sync, args)


@_workflow_defn('CaseBibleWikiPatchPublishV1')
class WikiPatchPublish:
    """Orchestrate prepare, upload and independent readback for the fixed wiki target set.

    Inputs: reviewed bundle path. Output: verified readback receipt or a failed stage.
    Effects: delegates server filesystem and B2 operations to three separate Activities.
    Choose for one parent-reviewed update to the existing Code/wiki objects; not for scheduled or bulk publishing.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """

    @_workflow_run
    async def run(self, args: dict[str, str]) -> dict[str, Any]:
        """Execute the three sequential Activities for all len(TARGETS) objects under one Temporal run ID.

        Inputs: reviewed bundle path. Output: final readback receipt.
        Effects: prepare local backups, upload fixed B2 objects, then read them back.
        Choose as the only Workflow entry point for this bounded patch.
        Byline: Codex / GPT-6 Luna / 2026-10-06.
        """
        run_id = temporal_workflow.info().run_id
        activity_args = {'run_id': run_id, 'bundle': args['bundle']}
        options = {'start_to_close_timeout': timedelta(minutes=20)}
        await temporal_workflow.execute_activity(prepare, activity_args, **options)
        await temporal_workflow.execute_activity(upload, activity_args, **options)
        return await temporal_workflow.execute_activity(readback, {'run_id': run_id}, **options)


async def _run_temporal(args: argparse.Namespace) -> None:
    """Start the worker or one explicitly requested wiki patch Workflow.

    Inputs: parsed worker/run options. Output: JSON worker or final Workflow result.
    Effects: worker polls Temporal; run starts server Activities that can write only the len(TARGETS) fixed B2 files.
    Choose only on ovh-files with the reviewed bundle present and owner publication authorization.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if Client is None or Worker is None:
        raise RuntimeError('Temporal Python SDK is required on the server for worker/run operations')
    client = await Client.connect(args.temporal_address)
    if args.operation == 'worker':
        async with Worker(client, task_queue=QUEUE, workflows=[WikiPatchPublish],
                          activities=[prepare, upload, readback], max_concurrent_activities=1):
            print(json.dumps({'status': 'polling', 'task_queue': QUEUE,
                              'workflow': 'CaseBibleWikiPatchPublishV1',
                              'activities': ['casebible_wiki_patch_prepare_v1',
                                             'casebible_wiki_patch_upload_v1',
                                             'casebible_wiki_patch_readback_v1'],
                              'byline': 'Codex / GPT-6 Luna / 2026-10-06'}), flush=True)
            await asyncio.Event().wait()
    else:
        if args.bundle is None:
            raise ValueError('--bundle is required for run')
        bundle = args.bundle.resolve(strict=True)
        if not bundle.is_relative_to(PROJECT_ROOT.resolve()):
            raise ValueError('Reviewed bundle must be inside the server wiki patch project directory')
        handle = await client.start_workflow(
            WikiPatchPublish.run, {'bundle': str(bundle)},
            id=args.workflow_id or ('casebible-wiki-patch-20261006-' + uuid.uuid4().hex),
            task_queue=QUEUE, execution_timeout=timedelta(hours=1),
        )
        result = await handle.result()
        print(json.dumps(result, ensure_ascii=False, indent=2))


def main() -> None:
    """Parse the worker/run interface for one fixed wiki patch.

    Inputs: command-line operation, Temporal address and optional reviewed bundle/workflow ID.
    Outputs: JSON worker state or final per-file hash receipt.
    Effects: `run` starts the three server Activities; no schedule or publication is started implicitly.
    Choose after the parent stages the reviewed bundle and authorizes publication.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=('worker', 'run'))
    parser.add_argument('--bundle', type=Path)
    parser.add_argument('--workflow-id', help='Optional stable ID for an intentional retry')
    parser.add_argument('--temporal-address', default='100.91.190.107:7233')
    parsed = parser.parse_args()
    asyncio.run(_run_temporal(parsed))


if __name__ == '__main__':
    main()
