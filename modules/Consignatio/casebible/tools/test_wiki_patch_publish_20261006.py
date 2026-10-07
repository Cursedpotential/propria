"""Focused tests for the fixed-target Code/wiki publisher.

Inputs: temporary reviewed bundles and an in-memory fake for bounded rclone operations.
Outputs: assertions over backup, concurrency, target-scope and readback contracts.
Effects: temporary filesystem writes only; tests never connect to Temporal, B2 or the VPS.
Choose when changing wiki_patch_publish_20261006.py; these tests do not exercise deployment configuration.
Byline: Codex / GPT-6 Luna / 2026-10-06.
"""
from __future__ import annotations

import json
from pathlib import Path

import pytest

import wiki_patch_publish_20261006 as publisher


OLD_INDEX = (
    b'---\ntitle: Code wiki index\nbyline: Prior author\n---\n'
    b'- [[Code/wiki/catalog-guide|Catalog and lake guide]]\n'
    b'- [[Code/wiki/plugin-inventory|Plugin inventory]]\n'
)
NEW_INDEX_LINK = b'- [[Code/wiki/receipts/corpus-inventory-preservation-20261006|Inventory preservation receipt]]\n'
OLD_GUIDE = b'prior guide bytes\n'
NEW_GUIDE = b'reviewed guide with current inventory and lake status\n'
NEW_RECEIPT = b'reviewed inventory preservation receipt\n'
OLD_PLUGIN_INVENTORY = b'prior generated inventory index\n'
NEW_PLUGIN_INVENTORY = b'updated generated inventory index\n'
OLD_TOOL_INVENTORY = b'{"prior": true}\n'
NEW_TOOL_INVENTORY = b'{"consignatio": {}}\n'
OLD_HUMAN_COMMANDS = b'plugin,kind\n'
NEW_HUMAN_COMMANDS = b'plugin,kind\nConsignatio,python_cli\n'
OLD_OBJECTS = {
    'catalog-guide.md': OLD_GUIDE,
    'INDEX.md': OLD_INDEX,
    'plugin-inventory.md': OLD_PLUGIN_INVENTORY,
    'assets/tool-inventory.json': OLD_TOOL_INVENTORY,
    'assets/human-commands.csv': OLD_HUMAN_COMMANDS,
}


def _bundle(tmp_path: Path, kind: str = 'directory', keep_links: bool = True) -> tuple[Path, dict[str, bytes]]:
    """Create a fixed reviewed source package for one isolated test.

    Inputs: pytest temporary path, directory/ZIP form and whether old links are retained.
    Outputs: bundle path and mapping of exactly len(TARGETS) payload byte strings.
    Effects: writes test fixtures under pytest's temporary directory only.
    Choose to exercise the same manifest and allowlist accepted from the parent.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    payloads = {
        'catalog-guide.md': NEW_GUIDE,
        'receipts/corpus-inventory-preservation-20261006.md': NEW_RECEIPT,
        'INDEX.md': (OLD_INDEX if keep_links else b'---\ntitle: Code wiki index\n---\n') + NEW_INDEX_LINK,
        'plugin-inventory.md': NEW_PLUGIN_INVENTORY,
        'assets/tool-inventory.json': NEW_TOOL_INVENTORY,
        'assets/human-commands.csv': NEW_HUMAN_COMMANDS,
    }
    old_hashes = {name: publisher.sha256(OLD_OBJECTS[name]) if name in OLD_OBJECTS else None
                  for name in publisher.TARGETS}
    rows = [{'path': name, 'expected_old_sha256': old_hashes[name],
             'new_sha256': publisher.sha256(payloads[name])} for name in publisher.TARGETS]
    manifest = json.dumps({'schema': publisher.MANIFEST_SCHEMA,
                           'byline': 'Parent reviewed / 2026-10-06',
                           'files': rows}, indent=2).encode()
    contents = {**payloads, publisher.MANIFEST_NAME: manifest}
    if kind == 'directory':
        path = tmp_path / 'reviewed'
        for name, data in contents.items():
            target = path / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(data)
        return path, payloads
    if kind == 'zip':
        import zipfile

        path = tmp_path / 'reviewed.zip'
        with zipfile.ZipFile(path, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
            for name, data in contents.items():
                archive.writestr(name, data)
        return path, payloads
    raise AssertionError(f'Unsupported test fixture kind: {kind}')


def _fake_rclone(monkeypatch, initial: dict[str, bytes]):
    """Install an in-memory rclone boundary for the fixed B2 keys.

    Inputs: pytest monkeypatch and existing remote contents. Outputs: remote map and call ledger.
    Effects: replaces the module's rclone adapter; no network or external writes.
    Choose to prove the publisher's remote safety contract without contacting B2.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    remote = {publisher._remote_path(name): data for name, data in initial.items()}
    writes: list[str] = []

    def fake(*arguments: str, input_bytes: bytes | None = None) -> bytes:
        """Emulate the exact stat, cat and one-key rcat calls used by the publisher.

        Inputs: rclone argument vector and optional bytes. Output: JSON metadata or remote bytes.
        Effects: updates only the in-memory allowlisted object map for rcat.
        Choose as the test transport boundary, never as a production implementation.
        Byline: Codex / GPT-6 Luna / 2026-10-06.
        """
        operation, *rest = arguments
        key = rest[-1]
        if operation == 'lsjson':
            if key in remote:
                relative = key.removeprefix(publisher.REMOTE_PREFIX + '/')
                return json.dumps({'Path': relative, 'Name': Path(relative).name,
                                   'Size': len(remote[key]), 'IsDir': False}).encode()
            return b'{"Path":"","Name":"","Size":-1,"IsDir":true}'
        if operation == 'cat':
            return remote[key]
        if operation == 'rcat':
            assert input_bytes is not None
            assert key.startswith(publisher.REMOTE_PREFIX + '/')
            writes.append(key)
            remote[key] = input_bytes
            return b''
        raise AssertionError(f'Unexpected rclone operation: {operation}')

    monkeypatch.setattr(publisher, '_rclone', fake)
    return remote, writes


@pytest.fixture
def staged(monkeypatch, tmp_path):
    """Set temporary project roots and the initial remote guide/index state.

    Inputs: pytest monkeypatch and temporary directory. Outputs: bundle, payloads, remote map and write ledger.
    Effects: isolates server artifact and to_be_deleted paths under pytest temporary storage.
    Choose as common setup for prepare/upload/readback contract tests.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    project = tmp_path / 'server-project' / 'wiki-patch-publish-20261006'
    server = tmp_path / 'server-project'
    project.mkdir(parents=True)
    monkeypatch.setattr(publisher, 'PROJECT_ROOT', project)
    monkeypatch.setattr(publisher, 'SERVER_PROJECT_ROOT', server)
    bundle, payloads = _bundle(project)
    remote, writes = _fake_rclone(monkeypatch, OLD_OBJECTS)
    return {'project': project, 'server': server, 'bundle': bundle, 'payloads': payloads,
            'remote': remote, 'writes': writes, 'run_id': 'test-run-20261006'}


def test_prepare_backs_up_exact_existing_bytes_and_freezes_hashes(staged):
    """Prepare checks current hashes and retains byte-exact backups before any remote write.

    Inputs: common isolated reviewed bundle and fake current B2 objects. Outputs: prepared manifest and backups.
    Effects: local test writes only; no object upload occurs.
    Choose to verify prepare-before-upload safety.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    result = publisher._prepare_sync({'bundle': str(staged['bundle']), 'run_id': staged['run_id']})

    assert result['status'] == 'prepared'
    assert staged['writes'] == []
    frozen = json.loads(Path(result['manifest_path']).read_text())
    assert [row['path'] for row in frozen['files']] == list(publisher.TARGETS)
    for name, prior in OLD_OBJECTS.items():
        entry = next(row for row in frozen['files'] if row['path'] == name)
        backup = Path(entry['backup_path'])
        assert backup.is_relative_to(staged['server'] / 'to_be_deleted')
        assert backup.read_bytes() == prior
        assert publisher.sha256(backup.read_bytes()) == entry['expected_old_sha256']
    receipt_entry = next(row for row in frozen['files'] if row['path'].startswith('receipts/'))
    assert receipt_entry['expected_old_sha256'] is None
    assert receipt_entry['backup_path'] is None
    assert len(frozen['files']) == len(publisher.TARGETS)


def test_target_allowlist_is_exact_and_excludes_unreviewed_artifacts():
    """Pin the publisher to the parent-approved six paths, with no broad prefix sync.

    Inputs: publisher constant. Output: exact ordered allowlist assertion.
    Effects: none. Choose to catch accidental target expansion or omission before any publish run.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    assert publisher.TARGETS == (
        'catalog-guide.md',
        'receipts/corpus-inventory-preservation-20261006.md',
        'INDEX.md',
        'plugin-inventory.md',
        'assets/tool-inventory.json',
        'assets/human-commands.csv',
    )


@pytest.mark.skipif(publisher.temporal_activity is None, reason='Temporal Python SDK is not installed in this test environment')
def test_worker_activities_have_real_sdk_definitions_and_fixed_names():
    """Validate all worker Activity definitions through the installed Temporal SDK metadata.

    Inputs: imported publisher module with Temporal installed. Outputs: exact registered Activity names.
    Effects: module-local SDK introspection only; no Temporal connection, Worker start or workflow execution.
    Choose to catch missing @activity.defn registration that pure helper tests cannot detect.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    registered = {
        function.__temporal_activity_definition.name
        for function in (publisher.prepare, publisher.upload, publisher.readback)
    }
    assert registered == {
        'casebible_wiki_patch_prepare_v1',
        'casebible_wiki_patch_upload_v1',
        'casebible_wiki_patch_readback_v1',
    }


def test_index_extension_accepts_refresh_when_receipt_link_already_exists():
    """Allow a later reviewed INDEX refresh to retain the existing receipt link without adding another.

    Inputs: prior INDEX with the receipt link and a refreshed INDEX with identical link targets. Output: accepted validation.
    Effects: none. Choose for repeat publication of updated wiki content under the same fixed receipt.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    prior = OLD_INDEX + NEW_INDEX_LINK
    refreshed = prior + b'\nUpdated guide and inventory links remain unchanged.\n'

    publisher._validate_index_extension(prior, refreshed)


@pytest.mark.parametrize(
    ('replacement', 'message'),
    [
        (OLD_INDEX + NEW_INDEX_LINK + b'- [[Code/wiki/unreviewed|Unreviewed]]\n', 'may add only'),
        (OLD_INDEX, 'removes one or more existing links'),
    ],
    ids=('reject-extra-link', 'reject-receipt-link-removal'),
)
def test_index_extension_rejects_extra_or_removed_links_on_refresh(replacement: bytes, message: str):
    """Reject navigation expansion or loss when the fixed receipt link already exists.

    Inputs: prior INDEX with the approved receipt and one invalid replacement. Output: ValueError for either change.
    Effects: none. Choose to protect repeat refreshes from broadening or shrinking reviewed navigation.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    prior = OLD_INDEX + NEW_INDEX_LINK

    with pytest.raises(ValueError, match=message):
        publisher._validate_index_extension(prior, replacement)


@pytest.mark.parametrize('bundle_kind', ['directory', 'zip'])
def test_three_activity_contract_updates_fixed_keys_and_reads_all_back(staged, bundle_kind, tmp_path, monkeypatch):
    """The three-stage workflow updates only len(TARGETS) allowlisted objects and verifies every hash.

    Inputs: reviewed directory or ZIP and fake old remote bytes. Output: successful fixed-target readback receipt.
    Effects: fake remote map only; production B2, Temporal and VPS are never contacted.
    Choose to verify supported parent handoff formats and end-to-end hash contract.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    if bundle_kind == 'zip':
        bundle, payloads = _bundle(staged['project'], kind='zip')
        staged['bundle'] = bundle
        staged['payloads'] = payloads
        remote, writes = _fake_rclone(monkeypatch, OLD_OBJECTS)
        staged['remote'], staged['writes'] = remote, writes

    args = {'bundle': str(staged['bundle']), 'run_id': staged['run_id']}
    publisher._prepare_sync(args)
    uploaded = publisher._upload_sync(args)
    readback = publisher._readback_sync({'run_id': staged['run_id']})

    expected_keys = [publisher._remote_path(name) for name in publisher.TARGETS]
    assert publisher._link_targets(staged['payloads']['INDEX.md'].decode()) - publisher._link_targets(OLD_INDEX.decode()) == {publisher.NEW_INDEX_LINK}
    assert staged['writes'] == expected_keys
    assert uploaded['status'] == 'uploaded'
    assert readback['status'] == 'verified'
    assert {row['path']: row['sha256'] for row in readback['files']} == {
        name: publisher.sha256(data) for name, data in staged['payloads'].items()
    }
    assert set(staged['remote']) == set(expected_keys)


@pytest.mark.parametrize('changed_path', publisher.TARGETS)
def test_upload_stops_before_overwriting_any_target_whose_hash_changes_after_prepare(staged, changed_path):
    """Upload refuses each individual overwrite if that target's remote bytes drift after prepare.

    Inputs: prepared manifest and one simulated concurrent object edit. Output: RuntimeError at that object.
    Effects: only earlier targets and the simulated edit can change; the conflicting key is never overwritten.
    Choose to verify every target's compare-immediately-before-write guard.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    args = {'bundle': str(staged['bundle']), 'run_id': staged['run_id']}
    publisher._prepare_sync(args)
    changed_bytes = b'concurrent owner edit'
    staged['remote'][publisher._remote_path(changed_path)] = changed_bytes

    with pytest.raises(RuntimeError, match='Remote changed after prepare'):
        publisher._upload_sync(args)
    assert publisher._remote_path(changed_path) not in staged['writes']
    assert staged['remote'][publisher._remote_path(changed_path)] == changed_bytes


def test_prepare_rejects_index_that_drops_existing_navigation(staged, tmp_path, monkeypatch):
    """Prepare refuses an INDEX patch that omits an existing wiki link.

    Inputs: remote INDEX and reviewed payload without its existing link. Output: ValueError before backups.
    Effects: no remote or retained backup writes.
    Choose to protect the existing Code/wiki navigation contract.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    bundle, _ = _bundle(staged['project'], keep_links=False)
    with pytest.raises(ValueError, match='removes one or more existing links'):
        publisher._prepare_sync({'bundle': str(bundle), 'run_id': staged['run_id']})
    assert not (staged['server'] / 'to_be_deleted').exists()
    assert staged['writes'] == []


def test_readback_rejects_mismatched_remote_bytes(staged):
    """Readback does not report verified when any published object differs from reviewed bytes.

    Inputs: prepared and uploaded test run with one subsequent remote alteration. Output: RuntimeError.
    Effects: simulated remote map only.
    Choose to verify final independent hash comparison for every target.
    Byline: Codex / GPT-6 Luna / 2026-10-06.
    """
    args = {'bundle': str(staged['bundle']), 'run_id': staged['run_id']}
    publisher._prepare_sync(args)
    publisher._upload_sync(args)
    staged['remote'][publisher._remote_path('INDEX.md')] = b'changed after upload'

    with pytest.raises(RuntimeError, match='Remote readback mismatch'):
        publisher._readback_sync({'run_id': staged['run_id']})
