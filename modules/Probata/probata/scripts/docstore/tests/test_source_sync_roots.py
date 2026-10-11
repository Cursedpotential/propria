"""Independent seven-root manifests and exact-plan/retraction regression proof.

Byline: Codex / GPT-6.1 / 2026-10-07. Isolated fixtures only, no live store.
"""
import hashlib
import errno
import asyncio
import json
import os
import sys
from contextlib import nullcontext
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import source_sync
import source_preservation
import adr


@pytest.fixture
def mounted_renames(mirror, monkeypatch):
    """Simulate independent source bind mounts and reject cross-root renames.

    Inputs: isolated mirror and monkeypatch. Output: the guarded rename function.
    Effects: only temporary fixture renames; use to reproduce the deployed EXDEV.
    """
    original = Path.replace
    roots = sorted((mirror / relative for relative, _ in source_sync.ROOTS.values()),
                   key=lambda path: len(path.parts), reverse=True)

    def guarded(source, destination):
        """Reject a rename across simulated mounts without changing either file.

        Inputs: source and destination paths. Output: original rename result.
        Effects: fixture-only rename on one mount, or EXDEV before mutation.
        """
        destination = Path(destination)
        source_mount = next((root for root in roots if source.is_relative_to(root)), mirror)
        destination_mount = next((root for root in roots if destination.is_relative_to(root)), mirror)
        if source_mount != destination_mount:
            raise OSError(errno.EXDEV, 'Invalid cross-device link')
        return original(source, destination)

    monkeypatch.setattr(Path, 'replace', guarded)
    return guarded


def entry(project='propria', path='new.md', content='# new\n'):
    return {'project':project, 'path':path, 'content':content,
            'sha256':hashlib.sha256(content.encode()).hexdigest()}


def test_explicit_roots_allow_empty_and_held_projects():
    roots=list(source_sync.ROOTS)
    assert source_sync.validated([], roots)==[]
    assert source_sync.validated([entry()], roots)==[entry()]


@pytest.mark.parametrize('roots', [[], ['propria'], 'propria',
    ['propria']*7, [*list(source_sync.ROOTS)[:-1], 'unapproved'],
    [*list(source_sync.ROOTS), 'propria'], [None]*7])
def test_invalid_root_manifest_rejected(roots):
    with pytest.raises(ValueError, match='roots'):
        source_sync.validated([], roots)


def test_legacy_files_only_still_requires_every_root():
    files=[entry(project) for project in source_sync.ROOTS]
    assert source_sync.validated(files)==files
    with pytest.raises(ValueError):
        source_sync.validated(files[:-1])
    files[-1]=entry('propria', 'different.md')
    with pytest.raises(ValueError, match='Every docs root'):
        source_sync.validated(files)


@pytest.mark.parametrize('bad', [None, 'bad', {'path':'new.md'},
    {'project':[], 'path':'new.md'}, {'project':'propria','path':[]},
    entry(path='../outside.md'), entry(path='private/no.md'),
    {**entry(), 'sha256':'wrong'}])
def test_source_validation_remains_strict(bad):
    with pytest.raises(ValueError):
        source_sync.validated([bad], list(source_sync.ROOTS))


@pytest.fixture
def mirror(tmp_path, monkeypatch):
    root=tmp_path/'mirror'
    root.mkdir()
    projects=[]
    for project,(relative,prefix) in source_sync.ROOTS.items():
        (root/relative).mkdir(parents=True, exist_ok=True)
        projects.append({'project_id':project, 'source_root':relative,
            'canonical_prefix':prefix, 'registration_status':'active',
            'required':True, 'excluded_patterns':['private/**','**/to_be_deleted/**']})
    registry=tmp_path/'registry.json'
    registry.write_text(json.dumps({'monorepo_root':str(root),'projects':projects}))
    monkeypatch.setenv('DOCSTORE_PROJECT_REGISTRY', str(registry))
    monkeypatch.setenv('DOCSTORE_SYNC_LOCK', str(tmp_path/'sync.lock'))
    monkeypatch.setattr(source_sync,'worker_lock',lambda _:nullcontext())
    return root


def test_plan_nonmutating_and_apply_quarantines_only_named_retraction(mirror):
    old=mirror/'docs'/'held.md'
    old.write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS), 'files':[entry()]}
    plan=source_sync.operation('plan', payload)
    assert plan['roots']==sorted(source_sync.ROOTS)
    assert plan['retracted_sources']==['propria/held.md']
    assert old.read_bytes()==b'# held\n'
    assert not (mirror/'docs'/'new.md').exists()
    with pytest.raises(ValueError, match='without being named'):
        source_sync.operation('apply', {**payload,'plan_id':plan['plan_id']})
    assert old.read_bytes()==b'# held\n'
    receipt=source_sync.operation('apply', {**payload,'plan_id':plan['plan_id'],
        'retract':['propria/held.md']})
    assert receipt['verified'] and not receipt['index_started']
    assert (mirror/'docs'/'new.md').read_bytes()==b'# new\n'
    saved=mirror/'docs'/'to_be_deleted'/('sync-'+receipt['generation'])/'propria'/'held.md'
    assert saved.read_bytes()==b'# held\n'


def test_hold_preserved_by_including_mirror_body(mirror):
    (mirror/'docs'/'held.md').write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS),
             'files':[entry(),entry(path='held.md',content='# held\n')]}
    plan=source_sync.operation('plan', payload)
    assert not plan['retracted_sources']
    receipt=source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert receipt['verified']
    assert (mirror/'docs'/'held.md').read_bytes()==b'# held\n'


def test_roots_bound_into_exact_plan(mirror):
    files=[entry(project) for project in source_sync.ROOTS]
    explicit={'roots':list(source_sync.ROOTS),'files':files}
    plan=source_sync.operation('plan',explicit)
    with pytest.raises(ValueError,match='plan changed'):
        source_sync.operation('apply',{'files':files,'plan_id':plan['plan_id']})


def retained_entry(project='propria', path='held.md', content=b'# held\n'):
    """Build a fixture-only exact retained reference without an uploaded body.

    Inputs: source identity and bytes. Output: reference dict. Effects: none;
    choose for retention tests instead of the full-body entry helper.
    """
    return {'project':project, 'path':path, 'retain_existing':True,
            'sha256':hashlib.sha256(content).hexdigest()}


@pytest.mark.parametrize('change', [
    {'content':None}, {'content':''}, {'content':'# held\n'},
    {'retain_existing':False}, {'retain_existing':1}, {'retain_existing':'true'},
    {'retain_existing':None}, {'sha256':None}, {'sha256':[]},
    {'sha256':'A'*64}, {'sha256':'f'*63}, {'sha256':'g'*64},
])
def test_retained_reference_rejects_ambiguous_mode_and_invalid_hash(change):
    """Reject ambiguous retention or invalid hashes without any filesystem work."""
    with pytest.raises(ValueError):
        source_sync.validated([{**retained_entry(), **change}], list(source_sync.ROOTS))
    missing=retained_entry()
    missing.pop('sha256')
    with pytest.raises(ValueError, match='canonical SHA-256'):
        source_sync.validated([missing], list(source_sync.ROOTS))


@pytest.mark.parametrize('path', ['private/held.md', 'to_be_deleted/held.md',
    '_to_be_deleted/held.md', '.hidden/held.md', '../held.md', 'advocatio/held.md'])
def test_retention_cannot_bypass_scope_exclusions(path):
    """Keep the existing docs-only exclusions on body-less retained references."""
    with pytest.raises(ValueError, match='scope'):
        source_sync.validated([retained_entry(path=path)], list(source_sync.ROOTS))


def test_retention_and_body_duplicate_identity_is_rejected():
    """Reject duplicate identities regardless of the selected upload mode."""
    with pytest.raises(ValueError, match='Duplicate'):
        source_sync.validated([retained_entry(), entry(path='held.md',content='# held\n')],
                              list(source_sync.ROOTS))


@pytest.mark.parametrize('missing', [False, True])
@pytest.mark.parametrize('action', ['plan', 'apply'])
def test_invalid_retention_validates_whole_plan_before_mutation(mirror, missing, action):
    """Refuse missing or changed retained sources before journal or source writes."""
    held=mirror/'docs'/'held.md'
    held.write_bytes(b'# actual\n')
    ref=retained_entry(path='missing.md' if missing else 'held.md')
    payload={'roots':list(source_sync.ROOTS), 'files':[entry(), ref], 'plan_id':'unused'}
    with pytest.raises(ValueError, match='does not exist' if missing else 'hash mismatch'):
        source_sync.operation(action,payload)
    assert held.read_bytes()==b'# actual\n'
    assert not (mirror/'docs'/'new.md').exists()
    assert not list(mirror.rglob('to_be_deleted'))
    assert not (Path(os.environ['DOCSTORE_SYNC_LOCK']).parent/'source-sync.json').exists()


def test_retention_identity_is_checked_under_worker_lock(mirror, monkeypatch):
    """Require current identity checks under the same lock during plan and apply."""
    from contextlib import contextmanager
    held=mirror/'docs'/'held.md'
    held.write_bytes(b'# held\n')
    locked=False
    original_state=source_sync.state

    @contextmanager
    def lock(_):
        """Mark fixture lock ownership until the source operation exits."""
        nonlocal locked
        locked=True
        try:
            yield
        finally:
            locked=False

    def state():
        """Assert fixture lock ownership whenever source identities are hashed."""
        assert locked
        return original_state()

    monkeypatch.setattr(source_sync,'worker_lock',lock)
    monkeypatch.setattr(source_sync,'state',state)
    payload={'roots':list(source_sync.ROOTS), 'files':[retained_entry()]}
    plan=source_sync.operation('plan',payload)
    assert not locked
    receipt=source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert receipt['verified'] and not locked


@pytest.mark.parametrize('retain_first', [False, True])
def test_same_hash_retention_mode_switch_requires_new_plan(mirror, retain_first):
    """Bind explicit retention mode into approval even when incoming bytes match."""
    (mirror/'docs'/'held.md').write_bytes(b'# held\n')
    full={'roots':list(source_sync.ROOTS), 'files':[entry(path='held.md',content='# held\n')]}
    retained={**full, 'files':[retained_entry()]}
    first,second=(retained,full) if retain_first else (full,retained)
    plan=source_sync.operation('plan',first)
    new_plan=source_sync.operation('plan',second)
    assert plan['incoming_digest']==new_plan['incoming_digest']
    assert plan['plan_id']!=new_plan['plan_id']
    with pytest.raises(ValueError,match='plan changed'):
        source_sync.operation('apply',{**second,'plan_id':plan['plan_id']})
    assert not (Path(os.environ['DOCSTORE_SYNC_LOCK']).parent/'source-sync.json').exists()


def test_retained_source_changed_after_plan_is_rejected(mirror):
    """Reject a stale retained hash without overwriting the changed original."""
    held=mirror/'docs'/'held.md'
    held.write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS), 'files':[entry(),retained_entry()]}
    plan=source_sync.operation('plan',payload)
    held.write_bytes(b'# newer\n')
    with pytest.raises(ValueError,match='hash mismatch'):
        source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert held.read_bytes()==b'# newer\n'
    assert not (mirror/'docs'/'new.md').exists()


def test_retained_source_disappeared_after_plan_is_rejected(mirror):
    """Refuse to recreate a retained source removed after the approved plan."""
    held=mirror/'docs'/'held.md'
    held.write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS), 'files':[entry(),retained_entry()]}
    plan=source_sync.operation('plan',payload)
    quarantine=held.parent/'to_be_deleted'
    quarantine.mkdir()
    held.replace(quarantine/'held.md')
    with pytest.raises(ValueError,match='does not exist'):
        source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert (quarantine/'held.md').read_bytes()==b'# held\n'
    assert not held.exists() and not (mirror/'docs'/'new.md').exists()
    assert not (Path(os.environ['DOCSTORE_SYNC_LOCK']).parent/'source-sync.json').exists()


def test_retention_still_binds_complete_current_snapshot(mirror):
    """Invalidate a retention plan when an unrelated current source arrives."""
    (mirror/'docs'/'held.md').write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS), 'files':[entry(),retained_entry()]}
    plan=source_sync.operation('plan',payload)
    (mirror/'docs'/'arrived.md').write_bytes(b'# arrived\n')
    with pytest.raises(ValueError,match='plan changed'):
        source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert not (mirror/'docs'/'new.md').exists()


def test_mixed_retained_new_replaced_sources_across_seven_roots(mirror, mounted_renames):
    """Apply a zero-retraction union and verify all identities and preserved bytes."""
    files=[]
    held_stats={}
    for project,(relative,_) in source_sync.ROOTS.items():
        directory=mirror/relative
        held=directory/'held.md'
        held.write_bytes(b'# held\n')
        held_stats[project]=held.stat()
        (directory/'replace.md').write_bytes(b'# old\n')
        files.extend([retained_entry(project),entry(project),entry(project,'replace.md')])
    payload={'roots':list(source_sync.ROOTS), 'files':files}
    plan=source_sync.operation('plan',payload)
    assert not plan['retracted_sources']
    assert plan['roots']==sorted(source_sync.ROOTS)
    expected={item['project']+'/'+item['path']:item['sha256'] for item in files}
    assert plan['incoming_hashes']==expected
    assert len(plan['retained_hashes'])==7
    receipt=source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert receipt['verified'] and not receipt['index_started']
    assert receipt['actual_hashes']==expected==source_sync.state()[1]
    assert len(receipt['preserved'])==7
    for saved in receipt['preserved']:
        assert not saved['source_path'].endswith('held.md')
        assert Path(saved['backup_path']).read_bytes()==b'# old\n'
        assert saved['sha256']==hashlib.sha256(b'# old\n').hexdigest()
    for project,(relative,_) in source_sync.ROOTS.items():
        held=mirror/relative/'held.md'
        assert held.read_bytes()==b'# held\n'
        assert held.stat().st_ino==held_stats[project].st_ino
        assert held.stat().st_mtime_ns==held_stats[project].st_mtime_ns
    journal=json.loads((Path(os.environ['DOCSTORE_SYNC_LOCK']).parent/'source-sync.json').read_text())
    assert journal['state']=='verified'
    assert journal['plan']==plan and journal['actual_hashes']==expected
    assert not list(mirror.rglob('*.pending'))


def test_large_existing_retained_sources_do_not_consume_body_upload_budget(mirror):
    """Retain over 32 MiB in place while enforcing unchanged body upload limits."""
    content=b'# held\r\n'*(256*1024)
    refs=[]
    for number in range(17):
        path=f'large-{number}.md'
        (mirror/'docs'/path).write_bytes(content)
        refs.append(retained_entry(path=path,content=content))
    payload={'roots':list(source_sync.ROOTS), 'files':[entry(),*refs]}
    plan=source_sync.operation('plan',payload)
    receipt=source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    assert receipt['verified'] and receipt['preserved']==[]
    assert len(receipt['retained_hashes'])==17
    assert all('content' not in ref for ref in refs)
    assert receipt['actual_hashes']==plan['incoming_hashes']
    with pytest.raises(ValueError,match='oversized'):
        source_sync.validated([entry(content=content.decode())],payload['roots'])
    body='x'*(1024*1024)
    with pytest.raises(ValueError,match='oversized'):
        source_sync.validated([entry(path=f'{i}.md',content=body) for i in range(33)],
                              payload['roots'])


def test_retained_final_verification_failure_blocks_verified_receipt(mirror, monkeypatch):
    """Refuse success when the final retained source differs from incoming hashes."""
    held=mirror/'docs'/'held.md'
    held.write_bytes(b'# held\n')
    payload={'roots':list(source_sync.ROOTS), 'files':[retained_entry()]}
    plan=source_sync.operation('plan',payload)
    original_state=source_sync.state
    calls=0

    def changed_state():
        """Simulate an external source change after locked apply validation."""
        nonlocal calls
        calls+=1
        if calls==2:
            held.write_bytes(b'# external change\n')
        return original_state()

    monkeypatch.setattr(source_sync,'state',changed_state)
    with pytest.raises(RuntimeError,match='verification failed; indexing must not start'):
        source_sync.operation('apply',{**payload,'plan_id':plan['plan_id']})
    journal=json.loads((Path(os.environ['DOCSTORE_SYNC_LOCK']).parent/'source-sync.json').read_text())
    assert journal['state']=='applying'


def test_body_upload_and_read_files_inventory_remain_compatible(mirror):
    """Preserve legacy full-body apply and the exact files-list read response."""
    files=[entry(project) for project in source_sync.ROOTS]
    plan=source_sync.operation('plan',{'files':files})
    receipt=source_sync.operation('apply',{'files':files,'plan_id':plan['plan_id']})
    assert receipt['verified'] and 'retained_hashes' not in receipt
    result=source_sync.operation('read',{'paths':['propria/new.md','propria/missing.md']})
    assert result=={'files':[{'key':'propria/new.md',**entry()}],
                    'missing':['propria/missing.md'],'remaining':[]}


def test_cross_mount_regression_new_file_before_changed_body(mirror, mounted_renames):
    """Apply the original partial-failure sequence while retaining exact old bytes.

    Inputs: seven-root fixture with EXDEV guard. Output: verified mirror/backup
    assertions. Effects: fixture writes only; use for new-file then replacement.
    """
    old = mirror/'docs'/'old.md'
    original = b'# original\r\nretain exact bytes\r\n'
    old.write_bytes(original)
    legacy = mirror/'to_be_deleted'/'old-generation'/'old.md'
    legacy.parent.mkdir(parents=True)
    with pytest.raises(OSError) as failure:
        old.replace(legacy)
    assert failure.value.errno == errno.EXDEV and old.read_bytes() == original
    historical = mirror/'docs'/'to_be_deleted'/'old-generation'/'history.md'
    historical.parent.mkdir(parents=True)
    historical.write_bytes(b'# historical backup\n')
    payload = {'roots': list(source_sync.ROOTS), 'files': [entry(), entry(path='old.md')]}
    plan = source_sync.operation('plan', payload)
    assert plan['changed'] == ['propria/new.md', 'propria/old.md']
    assert plan['retracted_sources'] == []
    receipt = source_sync.operation('apply', {**payload, 'plan_id': plan['plan_id']})
    assert receipt['verified'] and old.read_bytes() == b'# new\n'
    assert (mirror/'docs'/'new.md').read_bytes() == b'# new\n'
    saved = receipt['preserved'][0]
    assert Path(saved['backup_path']).read_bytes() == original
    assert saved['sha256'] == hashlib.sha256(original).hexdigest()
    assert historical.read_bytes() == b'# historical backup\n'
    assert source_sync.state()[1] == {'propria/new.md': entry()['sha256'],
                                      'propria/old.md': entry()['sha256']}


@pytest.mark.parametrize('project', list(source_sync.ROOTS))
def test_named_retraction_on_each_registered_mount(mirror, mounted_renames, project):
    """Quarantine a named retraction within each independently registered root.

    Inputs: source project and EXDEV-guarded fixture. Output: hash-bound receipt
    and empty active snapshot. Effects: fixture writes only; tests all seven roots.
    """
    directory = mirror/source_sync.ROOTS[project][0]
    old = directory/'nested'/'retract.md'
    old.parent.mkdir()
    old.write_bytes(b'original\r\n')
    payload = {'roots': list(source_sync.ROOTS), 'files': []}
    plan = source_sync.operation('plan', payload)
    receipt = source_sync.operation('apply', {**payload, 'plan_id': plan['plan_id'],
                                            'retract': [project+'/nested/retract.md']})
    assert receipt['verified'] and not old.exists()
    assert Path(receipt['preserved'][0]['backup_path']).is_relative_to(directory/'to_be_deleted')
    assert Path(receipt['preserved'][0]['backup_path']).read_bytes() == b'original\r\n'
    assert source_sync.state()[1] == {}


def test_symlinked_quarantine_refused_before_earlier_new_file(mirror, tmp_path):
    """Fail the entire apply before source writes when quarantine follows a symlink.

    Inputs: symlink-capable fixture. Output: unchanged source and outside bytes.
    Effects: fixture setup only on rejection; choose to test whole-batch preflight.
    """
    outside = tmp_path/'outside'
    outside.mkdir()
    quarantine = mirror/'docs'/'to_be_deleted'
    try:
        quarantine.symlink_to(outside, target_is_directory=True)
    except OSError as error:
        pytest.skip(f'Symlinks unavailable: {error}')
    old = mirror/'docs'/'old.md'
    old.write_bytes(b'original')
    payload = {'roots': list(source_sync.ROOTS), 'files': [entry(), entry(path='old.md')]}
    plan = source_sync.operation('plan', payload)
    with pytest.raises(ValueError, match='symlink'):
        source_sync.operation('apply', {**payload, 'plan_id': plan['plan_id']})
    assert old.read_bytes() == b'original'
    assert not (mirror/'docs'/'new.md').exists()
    assert list(outside.iterdir()) == []
    assert not (tmp_path/'source-sync.json').exists()


@pytest.mark.parametrize('blocked_path', ['to_be_deleted', 'incoming-parent'])
def test_non_directory_ancestor_refused_before_earlier_new_file(mirror, blocked_path):
    """Reject blocked quarantine or incoming parents before the first source write.

    Inputs: regular-file ancestor and new-before-old manifest. Output: unchanged
    originals and absent new source. Effects: fixture setup only on rejection;
    tests whole-batch preflight rather than a late preservation mkdir failure.
    """
    blocked = mirror/'docs'/blocked_path
    blocked.write_bytes(b'ancestor is a file')
    old = mirror/'docs'/'old.md'
    old.write_bytes(b'original')
    files = [entry(), entry(path='old.md')]
    if blocked_path == 'incoming-parent':
        files.append(entry(path='incoming-parent/nested.md'))
    payload = {'roots': list(source_sync.ROOTS), 'files': files}
    plan = source_sync.operation('plan', payload)
    with pytest.raises(ValueError, match='not a directory'):
        source_sync.operation('apply', {**payload, 'plan_id': plan['plan_id']})
    assert old.read_bytes() == b'original'
    assert blocked.read_bytes() == b'ancestor is a file'
    assert not (mirror/'docs'/'new.md').exists()
    assert not (mirror.parent/'source-sync.json').exists()


@pytest.mark.parametrize('mismatch', ['device', 'mount'])
def test_mount_mismatch_refused_before_any_source_write(mirror, monkeypatch, mismatch):
    """Reject nested quarantine devices or bind mounts before an earlier new file.

    Inputs: mocked mount/device boundary on an isolated fixture. Output: unchanged
    source assertions. Effects: fixture setup only; complements actual EXDEV guard.
    """
    quarantine = mirror/'docs'/'to_be_deleted'
    quarantine.mkdir()
    old = mirror/'docs'/'old.md'
    old.write_bytes(b'original')
    payload = {'roots': list(source_sync.ROOTS), 'files': [entry(), entry(path='old.md')]}
    plan = source_sync.operation('plan', payload)
    if mismatch == 'device':
        original_stat = Path.stat

        def device_stat(path, *args, **kwargs):
            """Report a different fixture quarantine device without mutating paths.

            Input: stat request. Output: stat result. Effects: read-only fixture mock.
            """
            value = original_stat(path, *args, **kwargs)
            fields = list(value)
            if path == quarantine:
                fields[2] += 1
            return os.stat_result(fields)

        monkeypatch.setattr(Path, 'stat', device_stat)
    else:
        monkeypatch.setattr(source_preservation, '_mount_id',
                            lambda path: 'nested' if path.is_relative_to(quarantine) else 'docs')
    with pytest.raises(ValueError, match='crosses'):
        source_sync.operation('apply', {**payload, 'plan_id': plan['plan_id']})
    assert old.read_bytes() == b'original'
    assert not (mirror/'docs'/'new.md').exists()
    assert list(quarantine.iterdir()) == []


@pytest.mark.parametrize('unsafe_quarantine', [False, True])
def test_changed_adr_preserves_original_on_registered_mount(mirror, mounted_renames, monkeypatch, unsafe_quarantine):
    """Materialize a changed ADR through the same mount-safe preservation helper.

    Inputs: bounded offline ADR row and EXDEV-guarded fixture. Output: exact
    projection and original hash receipt. Effects: fixture-only source writes;
    also proves read-only rendering never materializes or creates a backup, and
    a symlinked quarantine fails before touching the ADR source.
    """
    record = {'id': 'adr:propria_0100', 'project': 'propria', 'number': 100,
              'title': 'Preserve original', 'status': 'proposed', 'decision': 'Keep history'}

    class ProjectionDB:
        """Allow selected reads and projection metadata updates in memory only.

        Inputs: fixture queries. Outputs: one ADR row or empty matches.
        Effects: no external I/O; used only for filesystem materialization tests.
        """
        async def query(self, sql, params=None):
            """Answer the bounded projection queries without opening a database.

            Inputs: SQL and parameters. Outputs: fixture rows. Effects: none.
            """
            if sql == 'SELECT * FROM type::record($id);':
                return [record]
            assert sql.startswith(('UPDATE type::record($id) SET projection_',
                                   'SELECT id FROM document WHERE source_path='))
            return []

        async def close(self):
            """Close the fixture lease without side effects.

            Inputs: none. Output: None. Effects: none; only an offline lease.
            """

    async def connect(*args):
        """Return an offline projection lease instead of any live connection.

        Inputs: store identity. Output: fixture lease. Effects: none.
        """
        return ProjectionDB()

    monkeypatch.setattr(adr.sq, 'connect', connect)
    base = mirror/source_sync.ROOTS['probata'][0]
    target = base/'adr'/'generated'/'0100.md'
    target.parent.mkdir(parents=True)
    original = b'original ADR\r\n'
    target.write_bytes(original)
    rendered = asyncio.run(adr.refresh_projections(payload={'number': 100}))
    assert target.read_bytes() == original
    assert not (base/'to_be_deleted').exists()
    if unsafe_quarantine:
        outside = mirror.parent/'outside-adr'
        outside.mkdir()
        try:
            (base/'to_be_deleted').symlink_to(outside, target_is_directory=True)
        except OSError as error:
            pytest.skip(f'Symlinks unavailable: {error}')
        with pytest.raises(ValueError, match='symlink'):
            asyncio.run(adr.refresh_projections(materialize=True, payload={'number': 100}))
        assert target.read_bytes() == original
        assert list(outside.iterdir()) == []
        assert list(target.parent.glob('*.pending')) == []
        return
    receipt = asyncio.run(adr.refresh_projections(materialize=True, payload={'number': 100}))
    assert target.read_bytes() == rendered['projections'][0]['content'].encode()
    saved = receipt['preserved'][0]
    assert Path(saved['backup_path']).is_relative_to(base/'to_be_deleted')
    assert Path(saved['backup_path']).relative_to(base).parts[-4:] == ('probata','adr','generated','0100.md')
    assert Path(saved['backup_path']).read_bytes() == original
    assert saved['sha256'] == hashlib.sha256(original).hexdigest()
    unchanged = asyncio.run(adr.refresh_projections(materialize=True, payload={'number': 100}))
    assert unchanged['preserved'] == []
    assert source_sync.state()[1] == {'probata/adr/generated/0100.md': receipt['projections'][0]['sha256']}


def test_linux_mountinfo_distinguishes_same_device_bind_mounts(tmp_path, monkeypatch):
    """Resolve the longest Linux mount entry including escaped mountpoint spaces.

    Inputs: synthetic mountinfo with matching device numbers. Output: distinct
    mount IDs. Effects: fixture-only reads; choose to verify the Linux-only parser.
    """
    root = tmp_path/'source mount'
    nested = root/'to_be_deleted'
    root_text = root.as_posix().replace(' ', r'\040')
    nested_text = nested.as_posix().replace(' ', r'\040')
    mountinfo = (f'12 1 0:1 / {root_text} rw - ext4 /dev/source rw\n'
                 f'13 12 0:1 / {nested_text} rw - ext4 /dev/source rw\n')
    original_read = Path.read_text

    def read_mountinfo(path, *args, **kwargs):
        """Return synthetic Linux mount records only for the mountinfo reader.

        Input: read request. Output: text. Effects: read-only fixture substitution.
        """
        if path == Path('/proc/self/mountinfo'):
            return mountinfo
        return original_read(path, *args, **kwargs)

    monkeypatch.setattr(Path, 'read_text', read_mountinfo)
    with monkeypatch.context() as linux:
        linux.setattr(source_preservation.sys, 'platform', 'linux')
        assert source_preservation._mount_id(root/'old.md') == '12'
        assert source_preservation._mount_id(nested/'generation'/'old.md') == '13'
